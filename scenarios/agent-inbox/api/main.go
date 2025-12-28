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

	"agent-inbox/config"
	"agent-inbox/handlers"
	"agent-inbox/integrations"
	"agent-inbox/middleware"
	"agent-inbox/persistence"
	"agent-inbox/services"

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

	// Startup reconciliation: recover orphaned tool calls from prior crash/restart
	// TEMPORAL FLOW: This ensures progress continuity across server restarts
	reconcileSvc := services.NewReconciliationService(repo)
	if reconciled, err := reconcileSvc.ReconcileOrphanedToolCalls(context.Background()); err != nil {
		log.Printf("warning: reconciliation failed: %v", err)
	} else if reconciled > 0 {
		log.Printf("reconciliation: reconciled %d orphaned tool calls", reconciled)
	}

	// Create storage service for file uploads
	storageCfg := config.GetStorageConfig()
	storage := services.NewLocalStorageService(storageCfg)

	// Create handlers with all dependencies
	h := handlers.New(repo, integrations.NewOllamaClient(), storage)

	// Create upload handlers
	uploadHandlers := handlers.NewUploadHandlers(storage, storageCfg)
	uploadHandlers.SetRepo(repo)

	router := mux.NewRouter()
	router.Use(middleware.RequestID) // Request ID first for tracing
	router.Use(middleware.Logging)
	router.Use(middleware.CORS)
	h.RegisterRoutes(router)
	uploadHandlers.RegisterRoutes(router)

	// Start server with graceful shutdown
	if err := server.Run(server.Config{
		Handler: gorillahandlers.RecoveryHandler()(router),
		Cleanup: func(ctx context.Context) error { return db.Close() },
	}); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
