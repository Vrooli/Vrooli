package smoketest

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	httputil "scenario-to-desktop-api/shared/http"
)

// Handler provides HTTP handlers for smoke test operations.
type Handler struct {
	service        Service
	store          Store
	cancelManager  CancelManager
	packageFinder  PackageFinder
	outputPathFunc func(scenarioName string) string
}

// HandlerOption configures a Handler.
type HandlerOption func(*Handler)

// WithOutputPathFunc sets the function to determine output paths.
func WithOutputPathFunc(fn func(scenarioName string) string) HandlerOption {
	return func(h *Handler) {
		h.outputPathFunc = fn
	}
}

// WithPackageFinder sets the package finder.
func WithPackageFinder(pf PackageFinder) HandlerOption {
	return func(h *Handler) {
		h.packageFinder = pf
	}
}

// NewHandler creates a new smoke test handler.
func NewHandler(service Service, store Store, cancelManager CancelManager, opts ...HandlerOption) *Handler {
	h := &Handler{
		service:       service,
		store:         store,
		cancelManager: cancelManager,
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// RegisterRoutes registers smoke test routes on the given router.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/desktop/smoke-test/start", h.HandleStart).Methods("POST")
	r.HandleFunc("/api/v1/desktop/smoke-test/status/{smoke_test_id}", h.HandleStatus).Methods("GET")
	r.HandleFunc("/api/v1/desktop/smoke-test/cancel/{smoke_test_id}", h.HandleCancel).Methods("POST")
}

// HandleStart handles POST requests to start a smoke test.
func (h *Handler) HandleStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.WriteBadRequest(w, "method not allowed")
		return
	}

	var req StartRequest
	if !httputil.DecodeJSON(w, r, &req) {
		return
	}

	if req.ScenarioName == "" {
		httputil.WriteBadRequest(w, "scenario_name is required")
		return
	}
	if !isSafeScenarioName(req.ScenarioName) {
		httputil.WriteBadRequest(w, "invalid scenario_name")
		return
	}

	currentPlatform := h.service.CurrentPlatform()
	if req.Platform != "" && req.Platform != currentPlatform {
		httputil.WriteBadRequest(w, fmt.Sprintf("platform mismatch: server is %s", currentPlatform))
		return
	}

	if h.outputPathFunc == nil || h.packageFinder == nil {
		httputil.WriteInternalError(w, "handler not properly configured")
		return
	}

	desktopPath := h.outputPathFunc(req.ScenarioName)
	distPath := filepath.Join(desktopPath, "dist-electron")
	artifactPath, err := h.packageFinder.FindBuiltPackage(distPath, currentPlatform)
	if err != nil {
		httputil.WriteNotFound(w, fmt.Sprintf("no matching installer for %s", currentPlatform))
		return
	}

	smokeTestID := uuid.New().String()
	status := &Status{
		SmokeTestID:  smokeTestID,
		ScenarioName: req.ScenarioName,
		Platform:     currentPlatform,
		Status:       "running",
		ArtifactPath: artifactPath,
		StartedAt:    time.Now(),
		Logs: []string{
			fmt.Sprintf("Detected platform: %s", currentPlatform),
			fmt.Sprintf("Matched artifact: %s", filepath.Base(artifactPath)),
		},
	}
	h.store.Save(status)

	ctx, cancel := context.WithCancel(context.Background())
	h.cancelManager.SetCancel(smokeTestID, cancel)
	go h.service.PerformSmokeTest(ctx, smokeTestID, req.ScenarioName, artifactPath, currentPlatform)

	httputil.WriteJSONOK(w, StartResponse{
		SmokeTestID:  status.SmokeTestID,
		ScenarioName: status.ScenarioName,
		Platform:     status.Platform,
		Status:       status.Status,
		ArtifactPath: status.ArtifactPath,
		StartedAt:    status.StartedAt,
		Logs:         status.Logs,
	})
}

// HandleStatus handles GET requests for smoke test status.
func (h *Handler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["smoke_test_id"]
	if id == "" {
		httputil.WriteBadRequest(w, "smoke_test_id is required")
		return
	}

	status, ok := h.store.Get(id)
	if !ok {
		httputil.WriteNotFound(w, "smoke test")
		return
	}

	httputil.WriteJSONOK(w, status)
}

// HandleCancel handles POST requests to cancel a smoke test.
func (h *Handler) HandleCancel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["smoke_test_id"]
	if id == "" {
		httputil.WriteBadRequest(w, "smoke_test_id is required")
		return
	}

	cancel := h.cancelManager.TakeCancel(id)
	if cancel == nil {
		if _, ok := h.store.Get(id); ok {
			h.store.Update(id, func(status *Status) {
				status.Status = "failed"
				status.Error = "smoke test cancel requested but no running process was found"
				now := time.Now()
				status.CompletedAt = &now
			})
			httputil.WriteJSONOK(w, CancelResponse{Status: "cancelled"})
			return
		}
		httputil.WriteNotFound(w, "smoke test")
		return
	}

	cancel()
	httputil.WriteJSONOK(w, CancelResponse{Status: "cancelling"})
}

// isSafeScenarioName checks if a scenario name is safe to use.
func isSafeScenarioName(name string) bool {
	if name == "" {
		return false
	}
	// Prevent path traversal
	if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return false
	}
	return true
}
