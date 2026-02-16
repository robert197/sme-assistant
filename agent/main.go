package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

func main() {
	// Load picoclaw config
	cfgPath := os.Getenv("SME_CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "/root/.picoclaw/config.json"
	}

	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create provider
	provider, err := providers.CreateProvider(cfg)
	if err != nil {
		log.Fatalf("Failed to create provider: %v", err)
	}

	// Create message bus and agent loop
	msgBus := bus.NewMessageBus()
	agentLoop := agent.NewAgentLoop(cfg, msgBus, provider)

	// Start agent loop in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go agentLoop.Run(ctx)

	// HTTP server config from environment
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	host := os.Getenv("HOST")
	if host == "" {
		host = "0.0.0.0"
	}
	apiKey := os.Getenv("ASSISTANT_API_KEY")

	addr := fmt.Sprintf("%s:%s", host, port)
	handler := NewHTTPHandler(agentLoop, apiKey)

	srv := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 300 * time.Second,
	}

	// Start HTTP server
	go func() {
		log.Printf("HTTP server listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Graceful shutdown on SIGINT/SIGTERM
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	srv.Shutdown(shutdownCtx)

	agentLoop.Stop()
	log.Println("Stopped")
}
