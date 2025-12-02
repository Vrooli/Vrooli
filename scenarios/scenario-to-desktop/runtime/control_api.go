package bundleruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"scenario-to-desktop-runtime/manifest"
)

// registerHandlers sets up the control API HTTP routes.
func (s *Supervisor) registerHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/readyz", s.handleReady)
	mux.HandleFunc("/ports", s.handlePorts)
	mux.HandleFunc("/logs/tail", s.handleLogs)
	mux.HandleFunc("/shutdown", s.handleShutdown)
	mux.HandleFunc("/secrets", s.handleSecrets)
	mux.HandleFunc("/telemetry", s.handleTelemetry)
}

// handleHealth returns basic runtime health status.
// This endpoint is unauthenticated to allow Electron to gate on it.
func (s *Supervisor) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":   "ok",
		"services": len(s.serviceStatus),
		"runtime":  runtime.Version(),
	})
}

// handleReady returns readiness status for all services.
func (s *Supervisor) handleReady(w http.ResponseWriter, r *http.Request) {
	allReady := true
	details := make(map[string]ServiceStatus)

	s.mu.RLock()
	for id, st := range s.serviceStatus {
		details[id] = st
		if !st.Ready {
			allReady = false
		}
	}
	s.mu.RUnlock()

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"ready":   allReady,
		"details": details,
	})
}

// handlePorts returns the current port allocation map.
func (s *Supervisor) handlePorts(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"services": s.portMap,
	})
}

// handleLogs returns the last N lines from a service's log file.
func (s *Supervisor) handleLogs(w http.ResponseWriter, r *http.Request) {
	serviceID := r.URL.Query().Get("serviceId")
	linesParam := r.URL.Query().Get("lines")

	lines := 200
	if linesParam != "" {
		if v, err := parsePositiveInt(linesParam); err == nil {
			lines = v
		}
	}

	// Find the service.
	var service manifest.Service
	found := false
	for _, svc := range s.opts.Manifest.Services {
		if svc.ID == serviceID {
			service = svc
			found = true
			break
		}
	}

	if !found {
		http.Error(w, "unknown service", http.StatusBadRequest)
		return
	}

	if service.LogDir == "" {
		http.Error(w, "service has no log_dir", http.StatusBadRequest)
		return
	}

	logPath := manifest.ResolvePath(s.appData, service.LogDir)
	info, err := os.Stat(logPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("log path unavailable: %v", err), http.StatusBadRequest)
		return
	}
	if info.IsDir() {
		http.Error(w, "log_dir points to a directory; expected file path", http.StatusBadRequest)
		return
	}

	content, err := tailFile(logPath, lines)
	if err != nil {
		http.Error(w, fmt.Sprintf("tail logs: %v", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write(content)
}

// handleShutdown initiates graceful shutdown.
func (s *Supervisor) handleShutdown(w http.ResponseWriter, r *http.Request) {
	go func() {
		time.Sleep(100 * time.Millisecond)
		_ = s.Shutdown(context.Background())
	}()
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "stopping"})
}

// handleSecrets handles GET and POST for secret management.
func (s *Supervisor) handleSecrets(w http.ResponseWriter, r *http.Request) {
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
	Prompt      map[string]string `json:"prompt,omitempty"`
}

// handleSecretsGet returns the list of secrets and their current state.
func (s *Supervisor) handleSecretsGet(w http.ResponseWriter) {
	var secrets []secretView
	current := s.secretsCopy()

	for _, sec := range s.opts.Manifest.Secrets {
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
			Prompt:      sec.Prompt,
		})
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"secrets": secrets,
	})
}

// handleSecretsPost receives and stores new secret values.
func (s *Supervisor) handleSecretsPost(w http.ResponseWriter, r *http.Request) {
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

	// Merge with existing secrets.
	merged := s.secretsCopy()
	for k, v := range payload.Secrets {
		merged[k] = v
	}

	// Validate all required secrets are present.
	missing := s.missingRequiredSecretsFrom(merged)
	if len(missing) > 0 {
		msg := fmt.Sprintf("missing required secrets: %s", strings.Join(missing, ", "))
		_ = s.recordTelemetry("secrets_missing", map[string]interface{}{"missing": missing})
		s.writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error":   msg,
			"missing": missing,
		})

		// Update status for affected services.
		for _, svc := range s.opts.Manifest.Services {
			needs := intersection(missing, svc.Secrets)
			if len(needs) == 0 {
				continue
			}
			s.setStatus(svc.ID, ServiceStatus{Ready: false, Message: msg})
		}
		return
	}

	// Persist secrets.
	if err := s.persistSecrets(merged); err != nil {
		http.Error(w, "persist secrets", http.StatusInternalServerError)
		return
	}

	s.mu.Lock()
	s.secrets = merged
	s.mu.Unlock()

	_ = s.recordTelemetry("secrets_updated", map[string]interface{}{"count": len(merged)})
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})

	// Start services if this was the first time secrets were provided.
	if !s.servicesStarted {
		s.startServicesAsync()
	}
}

// handleTelemetry returns telemetry file path and upload URL.
func (s *Supervisor) handleTelemetry(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"path":       s.telemetryPath,
		"upload_url": s.opts.Manifest.Telemetry.UploadTo,
	})
}

// authMiddleware enforces bearer token authentication.
// The /healthz endpoint is exempted to allow Electron to poll before auth.
func (s *Supervisor) authMiddleware(next http.Handler) http.Handler {
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

// writeJSON sends a JSON response with the given status code.
func (s *Supervisor) writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
