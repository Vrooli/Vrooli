package build

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"scenario-to-desktop-api/shared/packaging"
	"scenario-to-desktop-api/shared/validation"

	"github.com/gorilla/mux"

	httputil "scenario-to-desktop-api/shared/http"
)

// Handler provides HTTP handlers for build operations.
type Handler struct {
	service      Service
	store        Store
	scenarioRoot string
	logger       *slog.Logger
}

// HandlerOption configures a Handler.
type HandlerOption func(*Handler)

// WithScenarioRoot sets the scenario root directory.
func WithScenarioRoot(root string) HandlerOption {
	return func(h *Handler) {
		h.scenarioRoot = root
	}
}

// WithHandlerLogger sets the logger for the handler.
func WithHandlerLogger(logger *slog.Logger) HandlerOption {
	return func(h *Handler) {
		h.logger = logger
	}
}

// NewHandler creates a new build handler.
func NewHandler(service Service, store Store, opts ...HandlerOption) *Handler {
	h := &Handler{
		service: service,
		store:   store,
		logger:  slog.Default(),
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// RegisterRoutes registers build routes on the given router.
// Build operations use /api/v1/pipeline/* endpoints.
// These utility endpoints remain active for artifact retrieval and notifications.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	// Download built package - useful for retrieving artifacts
	r.HandleFunc("/api/v1/desktop/download/{scenario_name}/{platform}", h.HandleDownload).Methods("GET")

	// Build complete webhook - useful for notifications
	r.HandleFunc("/api/v1/desktop/webhook/build-complete", h.HandleBuildCompleteWebhook).Methods("POST")
}

// HandleDownload handles GET requests to download a built package.
func (h *Handler) HandleDownload(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["scenario_name"]
	platform := vars["platform"]

	if scenarioName == "" || platform == "" {
		httputil.WriteBadRequest(w, "scenario_name and platform are required")
		return
	}

	// Validate platform
	validPlatforms := []string{"win", "mac", "linux"}
	if !slices.Contains(validPlatforms, platform) {
		httputil.WriteBadRequest(w, fmt.Sprintf("Invalid platform '%s'. Must be one of: win, mac, linux", platform))
		return
	}

	// Validate scenario name for security
	if !validation.IsSafeScenarioName(scenarioName) {
		httputil.WriteBadRequest(w, "Invalid scenario name")
		return
	}

	// Find built package
	distPath := filepath.Join(h.scenarioRoot, scenarioName, "platforms", "electron", "dist-electron")
	packageFile, err := packaging.FindBuiltPackage(distPath, platform)
	if err != nil {
		httputil.WriteNotFound(w, fmt.Sprintf("Built package not found: %s. Build the desktop app first.", err))
		return
	}

	// Get file info
	fileInfo, err := os.Stat(packageFile)
	if err != nil {
		httputil.WriteInternalError(w, "Failed to read package file")
		return
	}

	// Set appropriate content-type and headers
	filename := filepath.Base(packageFile)
	contentType := detectPackageContentType(packageFile)

	h.logger.Info("serving download",
		"scenario", scenarioName,
		"platform", platform,
		"file", filename,
		"size", fileInfo.Size())

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

	// Stream file to client
	http.ServeFile(w, r, packageFile)
}

// HandleBuildCompleteWebhook handles POST requests from build completion webhooks.
func (h *Handler) HandleBuildCompleteWebhook(w http.ResponseWriter, r *http.Request) {
	buildID := r.Header.Get("X-Build-ID")
	if buildID == "" {
		httputil.WriteBadRequest(w, "Missing X-Build-ID header")
		return
	}

	var result map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
		httputil.WriteBadRequest(w, "Invalid JSON format")
		return
	}

	// Update build status if it exists
	h.store.Update(buildID, func(status *Status) {
		if resultStatus, ok := result["status"].(string); ok {
			status.Status = resultStatus
		}
		if resultStatus, ok := result["status"].(string); ok && (resultStatus == "completed" || resultStatus == "failed") {
			now := time.Now()
			status.CompletedAt = &now
		}
	})

	h.logger.Info("build webhook received",
		"build_id", buildID,
		"status", result["status"],
		"has_error", result["error"] != nil)

	httputil.WriteJSONOK(w, map[string]string{"status": "received"})
}

// detectPackageContentType determines the content type based on file extension.
func detectPackageContentType(packageFile string) string {
	switch {
	case strings.HasSuffix(packageFile, ".msi"):
		return "application/x-msi"
	case strings.HasSuffix(packageFile, ".pkg"):
		return "application/vnd.apple.installer+xml"
	case strings.HasSuffix(packageFile, ".exe"):
		return "application/x-msdownload"
	case strings.HasSuffix(packageFile, ".dmg"):
		return "application/x-apple-diskimage"
	case strings.HasSuffix(packageFile, ".AppImage"):
		return "application/x-executable"
	case strings.HasSuffix(packageFile, ".deb"):
		return "application/vnd.debian.binary-package"
	case strings.HasSuffix(packageFile, ".zip"):
		return "application/zip"
	default:
		return "application/octet-stream"
	}
}
