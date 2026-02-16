package main

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"sync"

	"github.com/gofiber/fiber/v3"
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

// chatHandler holds per-endpoint state for the chat route.
type chatHandler struct {
	agent     AgentProcessor
	apiKey    string
	sem       chan struct{}
	sessionMu sync.Map // per-session mutexes: map[string]*sync.Mutex
}

// RegisterRoutes wires up health and chat endpoints on the Fiber app.
func RegisterRoutes(app *fiber.App, agent AgentProcessor, apiKey string) {
	h := &chatHandler{
		agent:  agent,
		apiKey: apiKey,
		sem:    make(chan struct{}, maxConcurrentRequests),
	}

	app.Get("/api/health", healthHandler)
	app.Post("/api/chat", h.handleChat)
}

func healthHandler(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *chatHandler) handleChat(c fiber.Ctx) error {
	// Bearer token authentication (skip if no key configured)
	if h.apiKey != "" {
		auth := c.Get("Authorization")
		if auth != "Bearer "+h.apiKey {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
	}

	var req chatRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.Message == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "message is required"})
	}

	// Validate conversation_id
	conversationID := req.ConversationID
	if conversationID == "" {
		conversationID = "default"
	}
	if !validConversationID.MatchString(conversationID) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid conversation_id"})
	}
	sessionKey := fmt.Sprintf("http:%s", conversationID)

	// Concurrency limiter — reject if all slots are in use
	select {
	case h.sem <- struct{}{}:
		defer func() { <-h.sem }()
	default:
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "server busy"})
	}

	// Per-session mutex prevents race conditions in session history
	lockI, _ := h.sessionMu.LoadOrStore(sessionKey, &sync.Mutex{})
	lock := lockI.(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()

	response, err := h.agent.ProcessDirect(c.Context(), req.Message, sessionKey)
	if err != nil {
		log.Printf("ProcessDirect error (session=%s): %v", sessionKey, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
	}

	return c.JSON(chatResponse{
		Response:       response,
		ConversationID: conversationID,
	})
}
