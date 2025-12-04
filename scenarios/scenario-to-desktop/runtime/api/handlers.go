// Package api provides the HTTP control API for the bundle runtime.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	"scenario-to-desktop-runtime/fileutil"
	"scenario-to-desktop-runtime/gpu"
	"scenario-to-desktop-runtime/health"
	"scenario-to-desktop-runtime/infra"
	"scenario-to-desktop-runtime/manifest"
)

// BundleValidationResult contains the results of bundle validation.
type BundleValidationResult struct {
	Valid            bool              `json:"valid"`
	Errors           []BundleError     `json:"errors,omitempty"`
	Warnings         []BundleWarning   `json:"warnings,omitempty"`
	MissingBinaries  []MissingBinary   `json:"missing_binaries,omitempty"`
	MissingAssets    []MissingAsset    `json:"missing_assets,omitempty"`
	InvalidChecksums []InvalidChecksum `json:"invalid_checksums,omitempty"`
}

// BundleError represents a fatal bundle validation error.
type BundleError struct {
	Code    string `json:"code"`
	Service string `json:"service,omitempty"`
	Path    string `json:"path,omitempty"`
	Message string `json:"message"`
}

// BundleWarning represents a non-fatal bundle validation warning.
type BundleWarning struct {
	Code    string `json:"code"`
	Service string `json:"service,omitempty"`
	Path    string `json:"path,omitempty"`
	Message string `json:"message"`
}

// MissingBinary describes a missing service binary.
type MissingBinary struct {
	ServiceID string `json:"service_id"`
	Platform  string `json:"platform"`
	Path      string `json:"path"`
}

// MissingAsset describes a missing service asset.
type MissingAsset struct {
	ServiceID string `json:"service_id"`
	Path      string `json:"path"`
}

// InvalidChecksum describes an asset with an invalid checksum.
type InvalidChecksum struct {
	ServiceID string `json:"service_id"`
	Path      string `json:"path"`
	Expected  string `json:"expected"`
	Actual    string `json:"actual"`
}

// Runtime defines the interface that the API layer uses to interact with the supervisor.
type Runtime interface {
	// Shutdown gracefully stops all services.
	Shutdown(ctx context.Context) error
	// ServiceStatuses returns the current status of all services.
	ServiceStatuses() map[string]health.Status
	// PortMap returns allocated ports for all services.
	PortMap() map[string]map[string]int
	// TelemetryPath returns the telemetry file path.
	TelemetryPath() string
	// TelemetryUploadURL returns the telemetry upload URL.
	TelemetryUploadURL() string
	// Manifest returns the bundle manifest.
	Manifest() *manifest.Manifest
	// AppDataDir returns the application data directory.
	AppDataDir() string
	// FileSystem returns the file system abstraction.
	FileSystem() infra.FileSystem
	// SecretStore returns the secret store for secret management.
	SecretStore() SecretStore
	// StartServicesIfReady triggers service startup if secrets are ready.
	StartServicesIfReady()
	// RecordTelemetry records a telemetry event.
	RecordTelemetry(event string, details map[string]interface{}) error
	// GPUStatus returns GPU detection info.
	GPUStatus() gpu.Status
	// ValidateBundle performs pre-flight bundle validation.
	ValidateBundle() *BundleValidationResult
}

// SecretStore defines the interface for secret management used by the API.
type SecretStore interface {
	Get() map[string]string
	Set(secrets map[string]string)
	Merge(newSecrets map[string]string) map[string]string
	MissingRequiredFrom(secrets map[string]string) []string
	Persist(secrets map[string]string) error
	// Validate checks secrets against manifest requirements (required, format).
	// Returns nil if all validations pass, or an error with details.
	Validate(secrets map[string]string) error
	// GenerateMissing generates values for per_install_generated secrets that don't exist.
	GenerateMissing(existingSecrets map[string]string) (map[string]string, error)
}

// Server handles HTTP requests for the control API.
type Server struct {
	runtime   Runtime
	authToken string
}

// NewServer creates a new API server.
func NewServer(rt Runtime, authToken string) *Server {
	return &Server{
		runtime:   rt,
		authToken: authToken,
	}
}

// RegisterHandlers sets up the control API HTTP routes.
func (s *Server) RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/readyz", s.handleReady)
	mux.HandleFunc("/ports", s.handlePorts)
	mux.HandleFunc("/logs/tail", s.handleLogs)
	mux.HandleFunc("/shutdown", s.handleShutdown)
	mux.HandleFunc("/secrets", s.handleSecrets)
	mux.HandleFunc("/telemetry", s.handleTelemetry)
	mux.HandleFunc("/validate", s.handleValidate)
}

// AuthMiddleware returns middleware that enforces bearer token authentication.
// The /healthz endpoint is exempted to allow Electron to poll before auth.
func (s *Server) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Health endpoint is unauthenticated for Electron startup checks.
		if r.URL.Path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "missing auth", http.StatusUnauthorized)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token != s.authToken {
			http.Error(w, "invalid auth", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// handleHealth returns basic runtime health status.
// This endpoint is unauthenticated to allow Electron to gate on it.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	statuses := s.runtime.ServiceStatuses()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":   "ok",
		"services": len(statuses),
		"runtime":  runtime.Version(),
	})
}

// handleReady returns readiness status for all services.
func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	statuses := s.runtime.ServiceStatuses()
	allReady := true
	for _, st := range statuses {
		if !st.Ready {
			allReady = false
			break
		}
	}

	gpuStatus := s.runtime.GPUStatus()
	requirements := map[string]string{}
	for _, svc := range s.runtime.Manifest().Services {
		if req := svc.GPURequirement(); req != "" {
			requirements[svc.ID] = req
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ready":   allReady,
		"details": statuses,
		"gpu": map[string]interface{}{
			"available":    gpuStatus.Available,
			"method":       gpuStatus.Method,
			"reason":       gpuStatus.Reason,
			"requirements": requirements,
		},
	})
}

// handlePorts returns the current port allocation map.
func (s *Server) handlePorts(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"services": s.runtime.PortMap(),
	})
}

// handleLogs returns the last N lines from a service's log file.
func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	serviceID := r.URL.Query().Get("serviceId")
	linesParam := r.URL.Query().Get("lines")

	lines := 200
	if linesParam != "" {
		if v, err := strconv.Atoi(strings.TrimSpace(linesParam)); err == nil && v > 0 {
			lines = v
		}
	}

	// Find the service.
	m := s.runtime.Manifest()
	var service *manifest.Service
	for i := range m.Services {
		if m.Services[i].ID == serviceID {
			service = &m.Services[i]
			break
		}
	}

	if service == nil {
		http.Error(w, "unknown service", http.StatusBadRequest)
		return
	}

	if service.LogDir == "" {
		http.Error(w, "service has no log_dir", http.StatusBadRequest)
		return
	}

	logPath := manifest.ResolvePath(s.runtime.AppDataDir(), service.LogDir)
	fs := s.runtime.FileSystem()

	info, err := fs.Stat(logPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("log path unavailable: %v", err), http.StatusBadRequest)
		return
	}
	if info.IsDir() {
		http.Error(w, "log_dir points to a directory; expected file path", http.StatusBadRequest)
		return
	}

	content, err := fileutil.TailFile(fs, logPath, lines)
	if err != nil {
		http.Error(w, fmt.Sprintf("tail logs: %v", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write(content)
}

// handleShutdown initiates graceful shutdown.
func (s *Server) handleShutdown(w http.ResponseWriter, r *http.Request) {
	go func() {
		time.Sleep(100 * time.Millisecond)
		_ = s.runtime.Shutdown(context.Background())
	}()
	writeJSON(w, http.StatusOK, map[string]string{"status": "stopping"})
}

// handleSecrets handles GET and POST for secret management.
func (s *Server) handleSecrets(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleSecretsGet(w)
	case http.MethodPost:
		s.handleSecretsPost(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// secretView is the response format for GET /secrets.
type secretView struct {
	ID          string            `json:"id"`
	Class       string            `json:"class"`
	Required    bool              `json:"required"`
	HasValue    bool              `json:"has_value"`
	Description string            `json:"description,omitempty"`
	Format      string            `json:"format,omitempty"`
	Prompt      map[string]string `json:"prompt,omitempty"`
}

// handleSecretsGet returns the list of secrets and their current state.
func (s *Server) handleSecretsGet(w http.ResponseWriter) {
	var secrets []secretView
	current := s.runtime.SecretStore().Get()
	m := s.runtime.Manifest()

	for _, sec := range m.Secrets {
		required := true
		if sec.Required != nil {
			required = *sec.Required
		}
		val := strings.TrimSpace(current[sec.ID])
		secrets = append(secrets, secretView{
			ID:          sec.ID,
			Class:       sec.Class,
			Required:    required,
			HasValue:    val != "",
			Description: sec.Description,
			Format:      sec.Format,
			Prompt:      sec.Prompt,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"secrets": secrets,
	})
}

// handleSecretsPost receives and stores new secret values.
func (s *Server) handleSecretsPost(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var payload struct {
		Secrets map[string]string `json:"secrets"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if payload.Secrets == nil {
		payload.Secrets = map[string]string{}
	}

	store := s.runtime.SecretStore()

	// Merge with existing secrets.
	merged := store.Merge(payload.Secrets)

	// Auto-generate per_install_generated secrets that are missing.
	generated, err := store.GenerateMissing(merged)
	if err != nil {
		_ = s.runtime.RecordTelemetry("secrets_generation_failed", map[string]interface{}{"error": err.Error()})
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"error": fmt.Sprintf("failed to generate secrets: %v", err),
		})
		return
	}
	if len(generated) > 0 {
		for k, v := range generated {
			merged[k] = v
		}
		_ = s.runtime.RecordTelemetry("secrets_generated", map[string]interface{}{
			"count": len(generated),
			"ids":   keys(generated),
		})
	}

	// Validate all secrets (required + format).
	if validationErr := store.Validate(merged); validationErr != nil {
		_ = s.runtime.RecordTelemetry("secrets_validation_failed", map[string]interface{}{"errors": validationErr.Error()})
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error":             "secret validation failed",
			"validation_errors": validationErr.Error(),
		})
		return
	}

	// Persist secrets.
	if err := store.Persist(merged); err != nil {
		http.Error(w, "persist secrets", http.StatusInternalServerError)
		return
	}

	store.Set(merged)

	_ = s.runtime.RecordTelemetry("secrets_updated", map[string]interface{}{"count": len(merged)})
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})

	// Start services if this was the first time secrets were provided.
	s.runtime.StartServicesIfReady()
}

// keys returns the keys of a map as a slice.
func keys(m map[string]string) []string {
	result := make([]string, 0, len(m))
	for k := range m {
		result = append(result, k)
	}
	return result
}

// handleTelemetry returns telemetry file path and upload URL.
func (s *Server) handleTelemetry(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"path":       s.runtime.TelemetryPath(),
		"upload_url": s.runtime.TelemetryUploadURL(),
	})
}

// handleValidate performs pre-flight bundle validation and returns results.
// This endpoint verifies all binaries and assets exist before service startup.
func (s *Server) handleValidate(w http.ResponseWriter, r *http.Request) {
	result := s.runtime.ValidateBundle()

	// Record validation telemetry.
	details := map[string]interface{}{
		"valid":             result.Valid,
		"error_count":       len(result.Errors),
		"warning_count":     len(result.Warnings),
		"missing_binaries":  len(result.MissingBinaries),
		"missing_assets":    len(result.MissingAssets),
		"invalid_checksums": len(result.InvalidChecksums),
	}
	_ = s.runtime.RecordTelemetry("bundle_validation", details)

	status := http.StatusOK
	if !result.Valid {
		status = http.StatusUnprocessableEntity
	}

	writeJSON(w, status, result)
}

// writeJSON sends a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
