// Package handlers provides HTTP request handlers for the sandbox API.
//
// This package is organized by domain area:
//   - handlers.go: Core types, interfaces, and response helpers
//   - admin.go: Health, driver info, and stats endpoints
//   - sandbox.go: Sandbox CRUD operations
//   - diff.go: Diff generation, approval, and rejection
//   - process.go: Process execution and management
//   - gc.go: Garbage collection operations
//   - audit.go: Audit log retrieval
//   - metrics.go: Prometheus metrics export
//   - rebase.go: Conflict detection and rebase workflow
package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	"workspace-sandbox/internal/config"
	"workspace-sandbox/internal/driver"
	"workspace-sandbox/internal/metrics"
	"workspace-sandbox/internal/process"
	"workspace-sandbox/internal/sandbox"
	"workspace-sandbox/internal/types"
)

// StatsGetter is an interface for retrieving sandbox statistics.
type StatsGetter interface {
	GetStats(ctx context.Context) (*types.SandboxStats, error)
}

// Handlers contains dependencies for HTTP handlers.
// Dependencies are expressed as interfaces to enable testing with mocks.
type Handlers struct {
	Service         sandbox.ServiceAPI  // Service interface for testability
	DriverManager   *driver.Manager     // Driver manager for hot-swapping drivers
	DB              Pinger
	Config          config.Config           // Unified configuration for accessing levers
	StatsGetter     StatsGetter             // For retrieving sandbox statistics
	ProcessTracker  *process.Tracker        // For tracking sandbox processes (OT-P0-008)
	GCService       GCService               // For garbage collection operations (OT-P1-003)
	InUserNamespace bool                    // Whether API is running in a user namespace
}

// Driver returns the current driver from the manager.
// This is a convenience method that maintains backward compatibility.
func (h *Handlers) Driver() driver.Driver {
	return h.DriverManager
}

// Version is the API version string.
// This is set at build time or defaults to "dev".
var Version = "1.0.0"

// Pinger is an interface for checking database connectivity.
type Pinger interface {
	PingContext(ctx context.Context) error
}

// --- Response Types ---

// ErrorResponse represents a standard error response with optional guidance.
type ErrorResponse struct {
	Error     string `json:"error"`
	Code      int    `json:"code"`
	Success   bool   `json:"success"`
	Hint      string `json:"hint,omitempty"`      // Actionable guidance for resolving the error
	Retryable bool   `json:"retryable,omitempty"` // Whether the operation might succeed on retry
}

// Hintable is an interface for errors that provide resolution hints.
type Hintable interface {
	Hint() string
}

// SuccessResponse wraps successful responses with metadata.
type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
}

// --- Response Helpers ---
// These helpers reduce repetition and ensure consistent response formats.

// JSONError writes a JSON error response with the given message and status code.
func (h *Handlers) JSONError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   message,
		Code:    code,
		Success: false,
	})
}

// JSONSuccess writes a JSON success response with the given data.
func (h *Handlers) JSONSuccess(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// JSONCreated writes a JSON response for newly created resources.
func (h *Handlers) JSONCreated(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(data)
}

// HandleDomainError writes an appropriate error response based on error type.
// It checks for domain errors first (which know their HTTP status),
// then falls back to internal server error for unknown errors.
// Returns true if an error was handled, false if err was nil.
func (h *Handlers) HandleDomainError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}

	// Check if error implements DomainError interface
	if domainErr, ok := err.(types.DomainError); ok {
		h.JSONDomainError(w, domainErr)
		return true
	}

	// Fallback to internal server error for unknown errors
	h.JSONError(w, err.Error(), http.StatusInternalServerError)
	return true
}

// JSONDomainError writes a JSON error response for domain errors with hints.
func (h *Handlers) JSONDomainError(w http.ResponseWriter, err types.DomainError) {
	response := ErrorResponse{
		Error:     err.Error(),
		Code:      err.HTTPStatus(),
		Success:   false,
		Retryable: err.IsRetryable(),
	}

	// Include hint if the error provides one
	if hintable, ok := err.(Hintable); ok {
		response.Hint = hintable.Hint()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.HTTPStatus())
	json.NewEncoder(w).Encode(response)
}

// --- Route Registration ---
// RegisterRoutes registers all API routes on the provided router.
// This centralizes route knowledge with the handlers and makes the API surface explicit.

// RegisterRoutes registers all routes for the workspace-sandbox API.
// Organized by domain for clarity:
//   - Health: Service health checks
//   - Sandbox: CRUD operations for sandboxes
//   - Workflow: Diff, approve, reject, conflict, rebase
//   - Process: Process execution and management
//   - Admin: Driver info, stats, GC, audit
//   - Metrics: Prometheus metrics
func (h *Handlers) RegisterRoutes(router *mux.Router, metricsCollector *metrics.Collector) {
	// --- API Info Endpoints ---
	// Exposed at root and /api for discoverability
	router.HandleFunc("/", h.APIInfo).Methods("GET")
	router.HandleFunc("/api", h.APIInfo).Methods("GET")

	// --- Health Endpoints ---
	// Exposed at both root and API paths for flexibility
	router.HandleFunc("/health", h.Health).Methods("GET")
	router.HandleFunc("/api/v1/health", h.Health).Methods("GET")

	// API v1 subrouter
	api := router.PathPrefix("/api/v1").Subrouter()

	// --- Sandbox CRUD ---
	api.HandleFunc("/sandboxes", h.CreateSandbox).Methods("POST")
	api.HandleFunc("/sandboxes", h.ListSandboxes).Methods("GET")
	api.HandleFunc("/sandboxes/{id}", h.GetSandbox).Methods("GET")
	api.HandleFunc("/sandboxes/{id}", h.DeleteSandbox).Methods("DELETE")
	api.HandleFunc("/sandboxes/{id}/stop", h.StopSandbox).Methods("POST")
	api.HandleFunc("/sandboxes/{id}/start", h.StartSandbox).Methods("POST")

	// --- Workflow: Diff and Approval ---
	api.HandleFunc("/sandboxes/{id}/diff", h.GetDiff).Methods("GET")
	api.HandleFunc("/sandboxes/{id}/approve", h.Approve).Methods("POST")
	api.HandleFunc("/sandboxes/{id}/reject", h.Reject).Methods("POST")
	api.HandleFunc("/sandboxes/{id}/discard", h.Discard).Methods("POST")

	// --- Workflow: Conflict Detection and Rebase (OT-P2-003) ---
	api.HandleFunc("/sandboxes/{id}/conflicts", h.CheckConflicts).Methods("GET")
	api.HandleFunc("/sandboxes/{id}/rebase", h.Rebase).Methods("POST")

	// --- Workspace and File Operations ---
	api.HandleFunc("/sandboxes/{id}/workspace", h.GetWorkspace).Methods("GET")
	api.HandleFunc("/sandboxes/{id}/files", h.ListFiles).Methods("GET")
	api.HandleFunc("/sandboxes/{id}/files/content", h.ReadFile).Methods("GET")
	api.HandleFunc("/sandboxes/{id}/files/content", h.WriteFile).Methods("PUT")
	api.HandleFunc("/sandboxes/{id}/files/content", h.DeleteFile).Methods("DELETE")
	api.HandleFunc("/sandboxes/{id}/files/mkdir", h.Mkdir).Methods("POST")
	api.HandleFunc("/sandboxes/{id}/files/download", h.DownloadFile).Methods("GET")

	// --- Process Isolation and Execution (OT-P0-003) ---
	api.HandleFunc("/sandboxes/{id}/exec", h.Exec).Methods("POST")
	api.HandleFunc("/sandboxes/{id}/processes", h.StartProcess).Methods("POST")
	api.HandleFunc("/sandboxes/{id}/processes", h.ListProcesses).Methods("GET")
	api.HandleFunc("/sandboxes/{id}/processes/{pid}", h.KillProcess).Methods("DELETE")
	api.HandleFunc("/sandboxes/{id}/processes/kill-all", h.KillAllProcesses).Methods("POST")

	// --- Admin: Driver and System Info ---
	api.HandleFunc("/driver/info", h.DriverInfo).Methods("GET")
	api.HandleFunc("/driver/options", h.DriverOptions).Methods("GET")
	api.HandleFunc("/driver/select", h.SelectDriver).Methods("POST")
	api.HandleFunc("/driver/preference", h.GetDriverPreference).Methods("GET")
	api.HandleFunc("/driver/bwrap", h.BwrapInfo).Methods("GET")
	api.HandleFunc("/validate-path", h.ValidatePath).Methods("GET")

	// --- Admin: Stats ---
	api.HandleFunc("/stats", h.Stats).Methods("GET")
	api.HandleFunc("/stats/processes", h.ProcessStats).Methods("GET")

	// --- Admin: Garbage Collection (OT-P1-003) ---
	api.HandleFunc("/gc", h.GC).Methods("POST")
	api.HandleFunc("/gc/preview", h.GCPreview).Methods("POST")

	// --- Admin: Audit Logs (OT-P1-004) ---
	api.HandleFunc("/audit", h.GetAuditLog).Methods("GET")
	api.HandleFunc("/sandboxes/{id}/audit", h.GetSandboxAuditLog).Methods("GET")

	// --- Metrics (OT-P1-008) ---
	// Supports Prometheus format (default) and JSON (?format=json)
	metricsHandler := func(w http.ResponseWriter, r *http.Request) {
		h.Metrics(w, r, metricsCollector)
	}
	api.HandleFunc("/metrics", metricsHandler).Methods("GET")
	router.HandleFunc("/metrics", metricsHandler).Methods("GET") // Also at root for Prometheus
}
