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
	schema "agent-inbox/internal/core"
	"agent-inbox/persistence"
	"agent-inbox/services"

	gorillahandlers "github.com/gorilla/handlers"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"
	_ "modernc.org/sqlite"
)

func main() {
	// Preflight checks
	if preflight.Run(preflight.Config{
		ScenarioName: "agent-inbox",
	}) {
		return
	}

	// The database is resolved from this scenario's own identity, so no
	// inherited environment can point it at a sibling's file.
	db, err := database.Connect(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     "agent-inbox",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
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

	// Create async tracker with persistence for crash recovery
	// The adapter bridges persistence.Repository to services.AsyncOperationRepository
	asyncRepoAdapter := newAsyncRepoAdapter(repo)
	toolExecutor := integrations.NewToolExecutor()
	asyncTracker := services.NewAsyncTrackerService(toolExecutor, asyncRepoAdapter)

	// Recover any active async operations from prior crash/restart
	if err := asyncTracker.RecoverOperations(context.Background()); err != nil {
		log.Printf("warning: async operation recovery failed: %v", err)
	}

	// Create storage service for file uploads
	storageCfg := config.GetStorageConfig()
	storage := services.NewLocalStorageService(storageCfg)

	// Create templates service for file-based template storage
	cfg := config.Default()
	templatesSvc := services.NewTemplatesService(&cfg.Templates)
	if err := templatesSvc.EnsureDirectories(); err != nil {
		log.Printf("warning: failed to create template directories: %v", err)
	}

	// Create skills service with prompt-manager sync
	skillsSvc := services.NewPromptSyncService(&cfg.PromptSync, &cfg.Skills)
	if err := skillsSvc.EnsureDirectories(); err != nil {
		log.Printf("warning: failed to create skill directories: %v", err)
	}
	// Start background sync from prompt-manager
	skillsSvc.Start()

	// Create handlers with all dependencies (pass pre-configured async tracker and shared executor)
	ollamaClient := integrations.NewOllamaClient()
	h := handlers.New(repo, ollamaClient, storage, asyncTracker, toolExecutor)
	h.Templates = templatesSvc
	h.Skills = skillsSvc

	h.SuggestionsSettings = services.NewSuggestionsSettingsService("../config/suggestions.json")

	// Create skill suggestion service (reuses same OllamaClient and PromptManagerURL)
	if cfg.SkillSuggest.Enabled {
		h.SkillSuggest = services.NewSkillSuggestService(
			ollamaClient,
			cfg.PromptSync.PromptManagerURL,
			&cfg.SkillSuggest,
		)
	}

	// Wire agent-manager client (may not be available at startup)
	agentClient, err := integrations.NewAgentManagerClient()
	if err != nil {
		log.Printf("warning: agent-manager not available: %v", err)
	}
	h.AgentClient = agentClient

	// Create upload handlers
	uploadHandlers := handlers.NewUploadHandlers(storage, storageCfg)
	uploadHandlers.SetRepo(repo)

	router := registerRoutes(h, uploadHandlers)

	// Start server with graceful shutdown

	if err := database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(schema.Schema)); err != nil {
		log.Fatalf("database schema initialization failed: %v", err)
	}
	if err := server.Run(server.Config{
		Handler: gorillahandlers.RecoveryHandler()(router),
		Cleanup: func(ctx context.Context) error {
			// Stop prompt sync background goroutine
			skillsSvc.Stop()
			// Gracefully shutdown async tracker (cancels polling, marks ops as interrupted)
			if err := asyncTracker.Shutdown(ctx); err != nil {
				log.Printf("warning: async tracker shutdown error: %v", err)
			}
			// Close database connection
			return db.Close()
		},
	}); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
