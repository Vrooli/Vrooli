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
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"agent-inbox/config"
	"agent-inbox/handlers"
	"agent-inbox/integrations"
	"agent-inbox/persistence"
	"agent-inbox/services"

	gorillahandlers "github.com/gorilla/handlers"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"
)

// =============================================================================
// SQLite DSN Builder
// =============================================================================

// sqliteDSN resolves the SQLite database file path and returns a DSN with pragmas.
func sqliteDSN() (string, error) {
	// Check scenario-specific env var first
	if path := strings.TrimSpace(os.Getenv("AI_SQLITE_PATH")); path != "" {
		return sqliteFileDSN(path)
	}
	// Check generic env vars
	if path := strings.TrimSpace(os.Getenv("SQLITE_PATH")); path != "" {
		return sqliteFileDSN(path)
	}
	if path := strings.TrimSpace(os.Getenv("SQLITE_DB")); path != "" {
		return sqliteFileDSN(path)
	}
	if dsn := strings.TrimSpace(os.Getenv("DATABASE_URL")); strings.HasPrefix(dsn, "file:") {
		return dsn, nil
	}

	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return "", fmt.Errorf("create storage resolver: %w", err)
	}
	path, err := resolver.Path(storage.Options{ScenarioID: "agent-inbox"}, storage.ClassData, "agent-inbox.db")
	if err != nil {
		return "", fmt.Errorf("resolve agent inbox db path: %w", err)
	}
	return sqliteFileDSN(path)
}

func sqliteFileDSN(path string) (string, error) {
	if strings.HasPrefix(path, "file:") {
		return path, nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("prepare sqlite directory: %w", err)
	}

	return fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=cache_size(-2000)&_pragma=synchronous(NORMAL)&_pragma=temp_store(MEMORY)",
		path,
	), nil
}

func main() {
	// Preflight checks
	if preflight.Run(preflight.Config{
		ScenarioName: "agent-inbox",
	}) {
		return
	}

	// Build SQLite DSN and connect
	dsn, err := sqliteDSN()
	if err != nil {
		log.Fatalf("sqlite configuration failed: %v", err)
	}

	db, err := database.Connect(context.Background(), database.Config{
		Driver:       "sqlite",
		DSN:          dsn,
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
	toolRegistry := services.NewToolRegistry(repo, toolExecutor)
	asyncTracker := services.NewAsyncTrackerService(toolRegistry, toolExecutor, asyncRepoAdapter)

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

	// Create handlers with all dependencies (pass pre-configured async tracker, shared executor, and registry)
	ollamaClient := integrations.NewOllamaClient()
	h := handlers.New(repo, ollamaClient, storage, asyncTracker, toolExecutor, toolRegistry)
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
