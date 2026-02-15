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
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"agent-inbox/config"
	"agent-inbox/handlers"
	"agent-inbox/integrations"
	"agent-inbox/middleware"
	"agent-inbox/persistence"
	"agent-inbox/services"

	gorillahandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"
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

	// Default path
	dataRoot := strings.TrimSpace(os.Getenv("SQLITE_DATABASE_PATH"))
	if dataRoot == "" {
		dataRoot = strings.TrimSpace(os.Getenv("VROOLI_DATA"))
	}
	if dataRoot == "" {
		home, _ := os.UserHomeDir()
		if home == "" {
			home = "."
		}
		dataRoot = filepath.Join(home, ".vrooli", "data", "sqlite", "databases")
	}

	return sqliteFileDSN(filepath.Join(dataRoot, "agent-inbox.db"))
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

// =============================================================================
// Async Repository Adapter
// =============================================================================

// asyncRepoAdapter bridges persistence.Repository to services.AsyncOperationRepository.
// It converts between persistence and services record types.
type asyncRepoAdapter struct {
	repo *persistence.Repository
}

func newAsyncRepoAdapter(repo *persistence.Repository) *asyncRepoAdapter {
	return &asyncRepoAdapter{repo: repo}
}

// Convert persistence record to services record
func toServiceRecord(rec *persistence.AsyncOperationRecord) *services.AsyncOperationRecord {
	if rec == nil {
		return nil
	}
	return &services.AsyncOperationRecord{
		ToolCallID:    rec.ToolCallID,
		ChatID:        rec.ChatID,
		ToolName:      rec.ToolName,
		ScenarioName:  rec.ScenarioName,
		OperationID:   rec.OperationID,
		Status:        rec.Status,
		Progress:      rec.Progress,
		Message:       rec.Message,
		Phase:         rec.Phase,
		Result:        json.RawMessage(rec.Result),
		Error:         rec.Error,
		AsyncBehavior: json.RawMessage(rec.AsyncBehavior),
		StartedAt:     rec.StartedAt,
		UpdatedAt:     rec.UpdatedAt,
		CompletedAt:   rec.CompletedAt,
	}
}

// Convert services record to persistence record
func toPersistenceRecord(rec *services.AsyncOperationRecord) *persistence.AsyncOperationRecord {
	if rec == nil {
		return nil
	}
	return &persistence.AsyncOperationRecord{
		ToolCallID:    rec.ToolCallID,
		ChatID:        rec.ChatID,
		ToolName:      rec.ToolName,
		ScenarioName:  rec.ScenarioName,
		OperationID:   rec.OperationID,
		Status:        rec.Status,
		Progress:      rec.Progress,
		Message:       rec.Message,
		Phase:         rec.Phase,
		Result:        json.RawMessage(rec.Result),
		Error:         rec.Error,
		AsyncBehavior: json.RawMessage(rec.AsyncBehavior),
		StartedAt:     rec.StartedAt,
		UpdatedAt:     rec.UpdatedAt,
		CompletedAt:   rec.CompletedAt,
	}
}

// Convert persistence event to services event
func toServiceEvent(ev *persistence.AsyncCompletionEventRecord) *services.AsyncCompletionEventRecord {
	if ev == nil {
		return nil
	}
	return &services.AsyncCompletionEventRecord{
		ID:         ev.ID,
		ChatID:     ev.ChatID,
		ToolCallID: ev.ToolCallID,
		ToolName:   ev.ToolName,
		Status:     ev.Status,
		Result:     json.RawMessage(ev.Result),
		Error:      ev.Error,
		CreatedAt:  ev.CreatedAt,
	}
}

// Convert services event to persistence event
func toPersistenceEvent(ev *services.AsyncCompletionEventRecord) *persistence.AsyncCompletionEventRecord {
	if ev == nil {
		return nil
	}
	return &persistence.AsyncCompletionEventRecord{
		ID:         ev.ID,
		ChatID:     ev.ChatID,
		ToolCallID: ev.ToolCallID,
		ToolName:   ev.ToolName,
		Status:     ev.Status,
		Result:     json.RawMessage(ev.Result),
		Error:      ev.Error,
		CreatedAt:  ev.CreatedAt,
	}
}

// Implement services.AsyncOperationRepository interface

func (a *asyncRepoAdapter) CreateAsyncOperation(ctx context.Context, op *services.AsyncOperationRecord) error {
	return a.repo.CreateAsyncOperation(ctx, toPersistenceRecord(op))
}

func (a *asyncRepoAdapter) UpdateAsyncOperation(ctx context.Context, op *services.AsyncOperationRecord) error {
	return a.repo.UpdateAsyncOperation(ctx, toPersistenceRecord(op))
}

func (a *asyncRepoAdapter) GetAsyncOperationByToolCallID(ctx context.Context, toolCallID string) (*services.AsyncOperationRecord, error) {
	rec, err := a.repo.GetAsyncOperationByToolCallID(ctx, toolCallID)
	if err != nil {
		return nil, err
	}
	return toServiceRecord(rec), nil
}

func (a *asyncRepoAdapter) GetActiveAsyncOperationsByChatID(ctx context.Context, chatID string) ([]*services.AsyncOperationRecord, error) {
	recs, err := a.repo.GetActiveAsyncOperationsByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}
	result := make([]*services.AsyncOperationRecord, len(recs))
	for i, rec := range recs {
		result[i] = toServiceRecord(rec)
	}
	return result, nil
}

func (a *asyncRepoAdapter) GetAllActiveAsyncOperations(ctx context.Context) ([]*services.AsyncOperationRecord, error) {
	recs, err := a.repo.GetAllActiveAsyncOperations(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*services.AsyncOperationRecord, len(recs))
	for i, rec := range recs {
		result[i] = toServiceRecord(rec)
	}
	return result, nil
}

func (a *asyncRepoAdapter) DeleteAsyncOperation(ctx context.Context, toolCallID string) error {
	return a.repo.DeleteAsyncOperation(ctx, toolCallID)
}

func (a *asyncRepoAdapter) CleanupCompletedAsyncOperations(ctx context.Context, olderThan time.Duration) (int64, error) {
	return a.repo.CleanupCompletedAsyncOperations(ctx, olderThan)
}

func (a *asyncRepoAdapter) UpdateAsyncOperationStatus(ctx context.Context, toolCallID, status string) error {
	return a.repo.UpdateAsyncOperationStatus(ctx, toolCallID, status)
}

func (a *asyncRepoAdapter) GetCompletedAsyncOperationsByChatID(ctx context.Context, chatID string, limit, offset int) ([]*services.AsyncOperationRecord, int, error) {
	recs, total, err := a.repo.GetCompletedAsyncOperationsByChatID(ctx, chatID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	result := make([]*services.AsyncOperationRecord, len(recs))
	for i, rec := range recs {
		result[i] = toServiceRecord(rec)
	}
	return result, total, nil
}

func (a *asyncRepoAdapter) CreateCompletionEvent(ctx context.Context, event *services.AsyncCompletionEventRecord) error {
	return a.repo.CreateCompletionEvent(ctx, toPersistenceEvent(event))
}

func (a *asyncRepoAdapter) GetCompletionEventsSince(ctx context.Context, chatID string, since time.Time) ([]*services.AsyncCompletionEventRecord, error) {
	events, err := a.repo.GetCompletionEventsSince(ctx, chatID, since)
	if err != nil {
		return nil, err
	}
	result := make([]*services.AsyncCompletionEventRecord, len(events))
	for i, ev := range events {
		result[i] = toServiceEvent(ev)
	}
	return result, nil
}

func (a *asyncRepoAdapter) CleanupOldCompletionEvents(ctx context.Context, olderThan time.Duration) (int64, error) {
	return a.repo.CleanupOldCompletionEvents(ctx, olderThan)
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
	// IMPORTANT: Pass the same toolExecutor AND toolRegistry to handlers so that:
	// 1. Tools registered by handlers.ToolRegistry are available to asyncTracker for status polling
	// 2. A single ToolRegistry cache is shared across all completion services
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

	router := mux.NewRouter()
	router.Use(middleware.RequestID) // Request ID first for tracing
	router.Use(middleware.Logging)
	router.Use(middleware.CORS)
	h.RegisterRoutes(router)
	uploadHandlers.RegisterRoutes(router)

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
