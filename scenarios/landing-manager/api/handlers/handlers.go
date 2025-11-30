// Package handlers contains HTTP handlers for the landing-manager API.
package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"landing-manager/errors"
	"landing-manager/services"
	"landing-manager/util"
)

// Handler wraps shared dependencies for all handlers.
// Dependencies are injected via constructors to enable testing with substitutes.
type Handler struct {
	DB               *sql.DB
	Registry         *services.TemplateRegistry
	Generator        *services.ScenarioGenerator
	PersonaService   *services.PersonaService
	PreviewService   *services.PreviewService
	AnalyticsService *services.AnalyticsService
	HTTPClient       *http.Client
	// CmdExecutor is the seam for executing CLI commands.
	// If nil, uses util.DefaultCommandExecutor (production behavior).
	// Tests can inject a MockCommandExecutor to avoid shelling out.
	CmdExecutor util.CommandExecutor
}

// NewHandler creates a new handler with all dependencies
func NewHandler(db *sql.DB, registry *services.TemplateRegistry, generator *services.ScenarioGenerator,
	personaService *services.PersonaService, previewService *services.PreviewService,
	analyticsService *services.AnalyticsService) *Handler {
	return &Handler{
		DB:               db,
		Registry:         registry,
		Generator:        generator,
		PersonaService:   personaService,
		PreviewService:   previewService,
		AnalyticsService: analyticsService,
		HTTPClient:       &http.Client{Timeout: 15 * time.Second},
		CmdExecutor:      nil, // uses util.DefaultCommandExecutor
	}
}

// NewHandlerWithHTTPClient creates a new handler with a custom HTTP client (for testing)
func NewHandlerWithHTTPClient(db *sql.DB, registry *services.TemplateRegistry, generator *services.ScenarioGenerator,
	personaService *services.PersonaService, previewService *services.PreviewService,
	analyticsService *services.AnalyticsService, httpClient *http.Client) *Handler {
	return &Handler{
		DB:               db,
		Registry:         registry,
		Generator:        generator,
		PersonaService:   personaService,
		PreviewService:   previewService,
		AnalyticsService: analyticsService,
		HTTPClient:       httpClient,
		CmdExecutor:      nil, // uses util.DefaultCommandExecutor
	}
}

// NewHandlerWithExecutor creates a handler with a custom command executor.
// This constructor is the seam for testing - pass a MockCommandExecutor to avoid
// shelling out to the actual vrooli CLI during lifecycle operations.
func NewHandlerWithExecutor(db *sql.DB, registry *services.TemplateRegistry, generator *services.ScenarioGenerator,
	personaService *services.PersonaService, previewService *services.PreviewService,
	analyticsService *services.AnalyticsService, cmdExecutor util.CommandExecutor) *Handler {
	return &Handler{
		DB:               db,
		Registry:         registry,
		Generator:        generator,
		PersonaService:   personaService,
		PreviewService:   previewService,
		AnalyticsService: analyticsService,
		HTTPClient:       &http.Client{Timeout: 15 * time.Second},
		CmdExecutor:      cmdExecutor,
	}
}

// executor returns the command executor, defaulting to the global if none set
func (h *Handler) executor() util.CommandExecutor {
	if h.CmdExecutor != nil {
		return h.CmdExecutor
	}
	return util.DefaultCommandExecutor
}

// RespondJSON writes a JSON response with the given status code
func (h *Handler) RespondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		util.LogStructured("json_encode_failed", map[string]interface{}{"error": err.Error()})
	}
}

// RespondError writes a JSON error response with consistent structure
func (h *Handler) RespondError(w http.ResponseWriter, statusCode int, message string) {
	h.RespondJSON(w, statusCode, map[string]interface{}{
		"success": false,
		"message": message,
	})
}

// Log writes a structured log entry
func (h *Handler) Log(msg string, fields map[string]interface{}) {
	util.LogStructured(msg, fields)
}

// RespondAppError writes an AppError as a JSON response with appropriate status code.
// Decision: Uses the centralized HTTPStatus() method on AppError, which defines
// the canonical mapping from error codes to HTTP status codes.
func (h *Handler) RespondAppError(w http.ResponseWriter, err *errors.AppError) {
	// Decision: HTTP status is determined by the error code via the centralized mapping
	statusCode := err.HTTPStatus()

	// Log the error with full context
	logFields := map[string]interface{}{
		"error_code":    err.Code,
		"http_status":   statusCode,
		"message":       err.Message,
		"recoverable":   err.Recoverable,
		"is_client_err": err.IsClientError(),
	}
	if err.Details != "" {
		logFields["details"] = err.Details
	}
	if err.Cause != nil {
		logFields["cause"] = err.Cause.Error()
	}
	util.LogStructuredError(string(err.Code), logFields)

	h.RespondJSON(w, statusCode, err.ToResponse())
}
