package server

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"

	"app-issue-tracker-api/internal/automation"
	issuespkg "app-issue-tracker-api/internal/issues"
	"app-issue-tracker-api/internal/logging"
	"app-issue-tracker-api/internal/server/services"
)

// NewServer constructs a configured Server instance and the associated HTTP handler.
// The caller is responsible for starting long-running background routines (e.g. processor loop).
func NewServer(config *Config, opts ...Option) (*Server, http.Handler, error) {
	if config == nil {
		return nil, nil, fmt.Errorf("config must not be nil")
	}
	if strings.TrimSpace(config.IssuesDir) == "" {
		return nil, nil, fmt.Errorf("config.IssuesDir must not be empty")
	}

	// Load agent settings once per process to ensure downstream dependencies are ready.
	if _, err := LoadAgentSettings(config.ScenarioRoot); err != nil {
		return nil, nil, fmt.Errorf("failed to load agent settings: %w", err)
	}
	logging.LogInfo("Agent settings loaded", "scenario_root", config.ScenarioRoot)

	// Ensure issues directory structure exists before wiring storage.
	for _, folder := range issuespkg.ValidStatuses() {
		folderPath := filepath.Join(config.IssuesDir, folder)
		if err := os.MkdirAll(folderPath, 0o755); err != nil {
			return nil, nil, fmt.Errorf("failed to ensure issue folder exists (%s): %w", folderPath, err)
		}
	}

	templatesDir := filepath.Join(config.IssuesDir, "templates")
	if err := os.MkdirAll(templatesDir, 0o755); err != nil {
		return nil, nil, fmt.Errorf("failed to ensure templates folder exists (%s): %w", templatesDir, err)
	}

	legacyWaitingDir := filepath.Join(config.IssuesDir, "waiting")
	if entries, err := os.ReadDir(legacyWaitingDir); err == nil {
		if len(entries) > 0 {
			logging.LogInfo("Migrating legacy waiting issues into open queue", "count", len(entries))
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				oldPath := filepath.Join(legacyWaitingDir, entry.Name())
				newPath := filepath.Join(config.IssuesDir, "open", entry.Name())
				if err := os.Rename(oldPath, newPath); err != nil {
					logging.LogWarn("Failed to migrate waiting issue", "issue", entry.Name(), "error", err)
				}
			}
		}
		if err := os.Remove(legacyWaitingDir); err != nil && !errors.Is(err, os.ErrNotExist) {
			logging.LogWarn("Failed to remove legacy waiting directory", "path", legacyWaitingDir, "error", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		logging.LogWarn("Failed to inspect legacy waiting directory", "path", legacyWaitingDir, "error", err)
	}

	server := &Server{config: config}

	for _, opt := range opts {
		opt(server)
	}

	server.processor = automation.NewProcessor(newProcessorHost(server))

	storeWasDefault := server.ensureStore(config.IssuesDir)
	server.ensureHub()
	server.ensureCommandFactory()
	server.wsUpgrader = newWebSocketUpgrader(config)

	server.issues = services.NewIssueService(server.store, services.NewArtifactManager(), server.processor, nil)
	server.investigations = NewInvestigationService(server)
	server.content = NewIssueContentService(server)

	storageType := fmt.Sprintf("%T", server.store)
	attrs := []any{"type", storageType}
	if storeWasDefault {
		attrs = append(attrs, "issues_dir", config.IssuesDir)
	}
	logging.LogInfo("Issue storage configured", attrs...)

	r := mux.NewRouter()
	r.HandleFunc("/health", server.healthHandler).Methods("GET")

	v1 := r.PathPrefix("/api/v1").Subrouter()
	v1.HandleFunc("/ws", server.handleWebSocket).Methods("GET")
	v1.HandleFunc("/components", server.getComponentsHandler).Methods("GET")
	v1.HandleFunc("/issues", server.getIssuesHandler).Methods("GET")
	v1.HandleFunc("/issues", server.createIssueHandler).Methods("POST")
	v1.HandleFunc("/issues/{id}/attachments/{attachment:.*}", server.getIssueAttachmentHandler).Methods("GET")
	v1.HandleFunc("/issues/{id}", server.getIssueHandler).Methods("GET")
	v1.HandleFunc("/issues/{id}", server.updateIssueHandler).Methods("PUT", "PATCH")
	v1.HandleFunc("/issues/{id}", server.deleteIssueHandler).Methods("DELETE")
	v1.HandleFunc("/issues/{id}/agent/conversation", server.getIssueAgentConversationHandler).Methods("GET")
	v1.HandleFunc("/issues/search", server.searchIssuesHandler).Methods("GET")
	v1.HandleFunc("/agents", server.getAgentsHandler).Methods("GET")
	v1.HandleFunc("/agent/settings", server.getAgentSettingsHandler).Methods("GET")
	v1.HandleFunc("/agent/settings", server.updateAgentSettingsHandler).Methods("PATCH")
	v1.HandleFunc("/apps", server.getAppsHandler).Methods("GET")
	v1.HandleFunc("/metadata/statuses", server.getIssueStatusesHandler).Methods("GET")
	v1.HandleFunc("/investigate", server.triggerInvestigationHandler).Methods("POST")
	v1.HandleFunc("/investigate/preview", server.previewInvestigationPromptHandler).Methods("POST")
	v1.HandleFunc("/stats", server.getStatsHandler).Methods("GET")
	v1.HandleFunc("/export", server.exportIssuesHandler).Methods("GET")
	v1.HandleFunc("/automation/processor", server.getProcessorHandler).Methods("GET")
	v1.HandleFunc("/automation/processor", server.updateProcessorHandler).Methods("PATCH")
	v1.HandleFunc("/automation/processor/reset-counter", server.resetIssueCounterHandler).Methods("POST")
	v1.HandleFunc("/rate-limit-status", server.getRateLimitStatusHandler).Methods("GET")
	v1.HandleFunc("/processes/running", server.getRunningProcessesHandler).Methods("GET")
	v1.HandleFunc("/processes/running/{id}", server.stopRunningProcessHandler).Methods("DELETE")

	handler := corsMiddleware(r)

	return server, handler, nil
}
