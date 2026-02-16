package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
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
	agentCtx, agentCancel := context.WithCancel(context.Background())
	defer agentCancel()
	go agentLoop.Run(agentCtx)

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

	// Create Fiber app
	app := fiber.New(fiber.Config{
		BodyLimit:    1 << 20, // 1 MB
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 300 * time.Second,
	})

	// Register routes
	RegisterRoutes(app, agentLoop, apiKey)

	// Graceful shutdown on SIGINT/SIGTERM
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Shutting down...")
		cancel()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		app.ShutdownWithContext(shutdownCtx)
	}()

	log.Printf("HTTP server listening on %s", addr)
	if err := app.Listen(addr, fiber.ListenConfig{GracefulContext: ctx}); err != nil {
		log.Fatalf("HTTP server error: %v", err)
	}

	agentCancel()
	agentLoop.Stop()
	log.Println("Stopped")
}
