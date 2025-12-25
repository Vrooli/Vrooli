// Package main provides the entry point for the Agent Inbox API.
//
// Architecture (Screaming Architecture):
//
//	domain/       Core domain types (Chat, Message, Label, ToolCall)
//	persistence/  Database operations (Repository)
//	integrations/ External services (OpenRouter, Ollama, AgentManager)
//	handlers/     HTTP handlers organized by domain
//	middleware/   HTTP middleware (CORS, Logging)
package main

import (
	"context"
	"log"

	"agent-inbox/handlers"
	"agent-inbox/integrations"
	"agent-inbox/middleware"
	"agent-inbox/persistence"

	gorillahandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"
)

func main() {
	// Preflight checks
	if preflight.Run(preflight.Config{
		ScenarioName: "agent-inbox",
	}) {
		return
	}

	// Connect to database
	db, err := database.Connect(context.Background(), database.Config{
		Driver: database.DriverPostgres,
	})
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}

	repo := persistence.NewRepository(db)
	if err := repo.InitSchema(context.Background()); err != nil {
		log.Fatalf("failed to initialize schema: %v", err)
	}

	// Create handlers with all dependencies
	h := handlers.New(repo, integrations.NewOllamaClient())

	router := mux.NewRouter()
	router.Use(middleware.RequestID) // Request ID first for tracing
	router.Use(middleware.Logging)
	router.Use(middleware.CORS)
	h.RegisterRoutes(router)

	// Start server with graceful shutdown
	if err := server.Run(server.Config{
		Handler: gorillahandlers.RecoveryHandler()(router),
		Cleanup: func(ctx context.Context) error { return db.Close() },
	}); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
