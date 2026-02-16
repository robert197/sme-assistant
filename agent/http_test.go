package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
)

// --- mock agent ---

type mockAgent struct {
	response string
	err      error
	delay    time.Duration
	calls    atomic.Int32
}

func (m *mockAgent) ProcessDirect(_ context.Context, content, sessionKey string) (string, error) {
	m.calls.Add(1)
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

// --- helpers ---

func newTestApp(agent AgentProcessor, apiKey string) *fiber.App {
	app := fiber.New(fiber.Config{BodyLimit: 1 << 20})
	RegisterRoutes(app, agent, apiKey)
	return app
}

func postChat(app *fiber.App, body string, headers ...string) (*chatResponse, int, string) {
	req := httptest.NewRequest("POST", "/api/chat", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	resp, err := app.Test(req)
	if err != nil {
		return nil, 0, fmt.Sprintf("app.Test error: %v", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var cr chatResponse
	json.Unmarshal(raw, &cr)
	return &cr, resp.StatusCode, string(raw)
}

// --- tests ---

func TestHealthEndpoint(t *testing.T) {
	app := newTestApp(&mockAgent{response: "ok"}, "")

	req := httptest.NewRequest("GET", "/api/health", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
	var body map[string]string
	json.NewDecoder(resp.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Fatalf("want status=ok, got %v", body)
	}
}

func TestChatSuccess(t *testing.T) {
	app := newTestApp(&mockAgent{response: "Hello!"}, "")

	cr, code, _ := postChat(app, `{"message":"hi"}`)
	if code != 200 {
		t.Fatalf("want 200, got %d", code)
	}
	if cr.Response != "Hello!" {
		t.Fatalf("want response=Hello!, got %q", cr.Response)
	}
	if cr.ConversationID != "default" {
		t.Fatalf("want conversation_id=default, got %q", cr.ConversationID)
	}
}

func TestChatWithConversationID(t *testing.T) {
	app := newTestApp(&mockAgent{response: "ok"}, "")

	cr, code, _ := postChat(app, `{"message":"hi","conversation_id":"session-42"}`)
	if code != 200 {
		t.Fatalf("want 200, got %d", code)
	}
	if cr.ConversationID != "session-42" {
		t.Fatalf("want conversation_id=session-42, got %q", cr.ConversationID)
	}
}

func TestAuthRequired(t *testing.T) {
	app := newTestApp(&mockAgent{response: "ok"}, "secret-key")

	tests := []struct {
		name   string
		header string
		want   int
	}{
		{"no header", "", 401},
		{"wrong key", "Bearer wrong", 401},
		{"correct key", "Bearer secret-key", 200},
		{"missing Bearer prefix", "secret-key", 401},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var headers []string
			if tt.header != "" {
				headers = []string{"Authorization", tt.header}
			}
			_, code, _ := postChat(app, `{"message":"hi"}`, headers...)
			if code != tt.want {
				t.Fatalf("want %d, got %d", tt.want, code)
			}
		})
	}
}

func TestAuthSkippedWhenNoKey(t *testing.T) {
	app := newTestApp(&mockAgent{response: "ok"}, "")

	_, code, _ := postChat(app, `{"message":"hi"}`)
	if code != 200 {
		t.Fatalf("want 200 (no auth required), got %d", code)
	}
}

func TestMissingMessage(t *testing.T) {
	app := newTestApp(&mockAgent{response: "ok"}, "")

	_, code, raw := postChat(app, `{"conversation_id":"test"}`)
	if code != 400 {
		t.Fatalf("want 400, got %d: %s", code, raw)
	}
}

func TestInvalidJSON(t *testing.T) {
	app := newTestApp(&mockAgent{response: "ok"}, "")

	_, code, _ := postChat(app, `not json`)
	if code != 400 {
		t.Fatalf("want 400, got %d", code)
	}
}

func TestInvalidConversationID(t *testing.T) {
	app := newTestApp(&mockAgent{response: "ok"}, "")

	bad := []string{
		`../../etc/passwd`,
		`a b c`,
		`<script>`,
		strings.Repeat("a", 65),
	}
	for _, id := range bad {
		t.Run(id, func(t *testing.T) {
			body := fmt.Sprintf(`{"message":"hi","conversation_id":"%s"}`, id)
			_, code, _ := postChat(app, body)
			if code != 400 {
				t.Fatalf("want 400 for conversation_id=%q, got %d", id, code)
			}
		})
	}
}

func TestValidConversationIDs(t *testing.T) {
	app := newTestApp(&mockAgent{response: "ok"}, "")

	good := []string{"abc", "session-1", "user_42", "A-Z_0-9", strings.Repeat("x", 64)}
	for _, id := range good {
		t.Run(id, func(t *testing.T) {
			body := fmt.Sprintf(`{"message":"hi","conversation_id":"%s"}`, id)
			_, code, _ := postChat(app, body)
			if code != 200 {
				t.Fatalf("want 200 for conversation_id=%q, got %d", id, code)
			}
		})
	}
}

func TestMethodNotAllowed(t *testing.T) {
	app := newTestApp(&mockAgent{response: "ok"}, "")

	for _, method := range []string{"GET", "PUT", "DELETE", "PATCH"} {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/chat", nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test: %v", err)
			}
			if resp.StatusCode != 405 {
				t.Fatalf("want 405, got %d", resp.StatusCode)
			}
		})
	}
}

func TestConcurrencyLimiter(t *testing.T) {
	blocker := make(chan struct{})
	ba := &blockingAgent{ch: blocker}

	app := fiber.New(fiber.Config{BodyLimit: 1 << 20})
	h := &chatHandler{
		agent:  ba,
		apiKey: "",
		sem:    make(chan struct{}, maxConcurrentRequests),
	}
	app.Get("/api/health", healthHandler)
	app.Post("/api/chat", h.handleChat)

	// Fill all 4 slots with blocking requests
	var wg sync.WaitGroup
	for i := 0; i < maxConcurrentRequests; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			body := fmt.Sprintf(`{"message":"hi","conversation_id":"slot-%d"}`, i)
			req := httptest.NewRequest("POST", "/api/chat", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			app.Test(req, fiber.TestConfig{Timeout: 5 * time.Second})
		}(i)
	}

	// Give goroutines time to acquire semaphore slots
	time.Sleep(100 * time.Millisecond)

	// 5th request should get 503
	_, code, _ := postChatOn(app, `{"message":"hi","conversation_id":"overflow"}`)
	if code != 503 {
		t.Fatalf("want 503 when all slots full, got %d", code)
	}

	// Release blocked requests
	close(blocker)
	wg.Wait()
}

// blockingAgent blocks ProcessDirect until channel is closed.
type blockingAgent struct{ ch chan struct{} }

func (b *blockingAgent) ProcessDirect(_ context.Context, _, _ string) (string, error) {
	<-b.ch
	return "ok", nil
}

func postChatOn(app *fiber.App, body string) (*chatResponse, int, string) {
	req := httptest.NewRequest("POST", "/api/chat", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 1 * time.Second})
	if err != nil {
		return nil, 0, fmt.Sprintf("app.Test error: %v", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var cr chatResponse
	json.Unmarshal(raw, &cr)
	return &cr, resp.StatusCode, string(raw)
}

func TestAgentError(t *testing.T) {
	agent := &mockAgent{err: fmt.Errorf("provider down")}
	app := newTestApp(agent, "")

	_, code, raw := postChat(app, `{"message":"hi"}`)
	if code != 500 {
		t.Fatalf("want 500, got %d: %s", code, raw)
	}
}

func TestSessionMutexSerializes(t *testing.T) {
	// Track execution order: if mutex works, calls to same session
	// should not overlap (call count increments sequentially).
	var running atomic.Int32
	var maxConcurrent atomic.Int32

	agent := &mockAgent{delay: 50 * time.Millisecond, response: "ok"}
	ta := &trackingAgent{inner: agent, running: &running, maxRunning: &maxConcurrent}

	app := fiber.New(fiber.Config{BodyLimit: 1 << 20})
	RegisterRoutes(app, ta, "")

	// Fire 3 requests to the SAME session concurrently
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("POST", "/api/chat",
				strings.NewReader(`{"message":"hi","conversation_id":"same-session"}`))
			req.Header.Set("Content-Type", "application/json")
			app.Test(req, fiber.TestConfig{Timeout: 5 * time.Second})
		}()
	}
	wg.Wait()

	// With the per-session mutex, max concurrent for same session should be 1
	if maxConcurrent.Load() > 1 {
		t.Fatalf("expected max 1 concurrent for same session, got %d", maxConcurrent.Load())
	}
}

type trackingAgent struct {
	inner      *mockAgent
	running    *atomic.Int32
	maxRunning *atomic.Int32
}

func (ta *trackingAgent) ProcessDirect(ctx context.Context, content, sessionKey string) (string, error) {
	cur := ta.running.Add(1)
	for {
		old := ta.maxRunning.Load()
		if cur <= old || ta.maxRunning.CompareAndSwap(old, cur) {
			break
		}
	}
	defer ta.running.Add(-1)
	return ta.inner.ProcessDirect(ctx, content, sessionKey)
}
