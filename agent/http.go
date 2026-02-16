package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"sync"
)

// AgentProcessor is the interface for synchronous message processing.
type AgentProcessor interface {
	ProcessDirect(ctx context.Context, content, sessionKey string) (string, error)
}

// maxConcurrentRequests limits simultaneous in-flight chat requests.
const maxConcurrentRequests = 4

// validConversationID restricts conversation IDs to safe characters.
var validConversationID = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

type chatRequest struct {
	Message        string `json:"message"`
	ConversationID string `json:"conversation_id,omitempty"`
}

type chatResponse struct {
	Response       string `json:"response"`
	ConversationID string `json:"conversation_id"`
}

// HTTPHandler provides the HTTP API for the assistant.
type HTTPHandler struct {
	agent     AgentProcessor
	apiKey    string
	sem       chan struct{}
	sessionMu sync.Map // per-session mutexes: map[string]*sync.Mutex
}

// NewHTTPHandler creates an http.Handler with health and chat endpoints.
func NewHTTPHandler(agent AgentProcessor, apiKey string) http.Handler {
	h := &HTTPHandler{
		agent:  agent,
		apiKey: apiKey,
		sem:    make(chan struct{}, maxConcurrentRequests),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", h.healthHandler)
	mux.HandleFunc("/api/chat", h.chatHandler)
	return mux
}

func (h *HTTPHandler) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *HTTPHandler) chatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Bearer token authentication (skip if no key configured)
	if h.apiKey != "" {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer "+h.apiKey {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
	}

	// Limit request body to 1 MB
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Message == "" {
		http.Error(w, `{"error":"message is required"}`, http.StatusBadRequest)
		return
	}

	// Validate conversation_id
	conversationID := req.ConversationID
	if conversationID == "" {
		conversationID = "default"
	}
	if !validConversationID.MatchString(conversationID) {
		http.Error(w, `{"error":"invalid conversation_id"}`, http.StatusBadRequest)
		return
	}
	sessionKey := fmt.Sprintf("http:%s", conversationID)

	// Concurrency limiter — reject if all slots are in use
	select {
	case h.sem <- struct{}{}:
		defer func() { <-h.sem }()
	default:
		http.Error(w, `{"error":"server busy"}`, http.StatusServiceUnavailable)
		return
	}

	// Per-session mutex prevents race conditions in session history
	lockI, _ := h.sessionMu.LoadOrStore(sessionKey, &sync.Mutex{})
	lock := lockI.(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()

	response, err := h.agent.ProcessDirect(r.Context(), req.Message, sessionKey)
	if err != nil {
		log.Printf("ProcessDirect error (session=%s): %v", sessionKey, err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chatResponse{
		Response:       response,
		ConversationID: conversationID,
	})
}
