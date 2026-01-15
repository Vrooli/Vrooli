package records

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"

	"scenario-to-desktop-api/shared/validation"
)

// Handler provides HTTP handlers for record endpoints.
type Handler struct {
	records        Store
	builds         BuildStoreAdapter
	logger         *slog.Logger
	scenarioRoot   string
	outputPathFunc func(scenarioName string) string
}

// BuildStoreAdapter adapts the build store interface for record operations.
type BuildStoreAdapter interface {
	Get(id string) (*BuildStatusView, bool)
	Update(id string, fn func(status *BuildStatusView)) error
}

// HandlerOption configures a Handler.
type HandlerOption func(*Handler)

// WithScenarioRoot sets the scenario root path.
func WithScenarioRoot(root string) HandlerOption {
	return func(h *Handler) {
		h.scenarioRoot = root
	}
}

// WithOutputPathFunc sets the function for computing desktop output paths.
func WithOutputPathFunc(fn func(scenarioName string) string) HandlerOption {
	return func(h *Handler) {
		h.outputPathFunc = fn
	}
}

// NewHandler creates a new records handler.
func NewHandler(records Store, builds BuildStoreAdapter, logger *slog.Logger, opts ...HandlerOption) *Handler {
	h := &Handler{
		records: records,
		builds:  builds,
		logger:  logger,
	}
	for _, opt := range opts {
		opt(h)
	}
	// Default output path function
	if h.outputPathFunc == nil && h.scenarioRoot != "" {
		h.outputPathFunc = func(scenarioName string) string {
			return filepath.Join(h.scenarioRoot, scenarioName, "platforms", "electron")
		}
	}
	return h
}

// RegisterRoutes registers record routes on the given router.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/desktop/records", h.ListHandler).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/desktop/records/{record_id}/move", h.MoveHandler).Methods("POST", "OPTIONS")
	// DELETE endpoint for cleaning up desktop apps (moved from generation handler)
	r.HandleFunc("/api/v1/desktop/delete/{scenario_name}", h.DeleteHandler).Methods("DELETE", "OPTIONS")
}

// ListHandler returns the persisted desktop generation records.
func (h *Handler) ListHandler(w http.ResponseWriter, r *http.Request) {
	if h.records == nil {
		http.Error(w, "record store unavailable", http.StatusInternalServerError)
		return
	}

	var results []RecordWithBuild
	for _, rec := range h.records.List() {
		item := RecordWithBuild{Record: rec}
		if rec != nil && rec.BuildID != "" && h.builds != nil {
			if bs, ok := h.builds.Get(rec.BuildID); ok {
				item.Build = bs
				item.HasBuild = true
				item.BuildState = bs.Status
			}
		}
		results = append(results, item)
	}

	writeJSON(w, http.StatusOK, ListResponse{Records: results})
}

// MoveHandler moves a generated desktop wrapper from its current path to a destination.
func (h *Handler) MoveHandler(w http.ResponseWriter, r *http.Request) {
	if h.records == nil {
		http.Error(w, "record store unavailable", http.StatusInternalServerError)
		return
	}

	recordID := mux.Vars(r)["record_id"]
	rec, ok := h.records.Get(recordID)
	if !ok || rec == nil {
		http.Error(w, "record not found", http.StatusNotFound)
		return
	}

	var req MoveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if req.Target == "" {
		req.Target = "destination"
	}

	src := rec.OutputPath
	dest := rec.DestinationPath
	if req.Target == "custom" {
		if req.DestinationPath == "" {
			http.Error(w, "destination_path required for custom target", http.StatusBadRequest)
			return
		}
		dest = req.DestinationPath
	}
	if dest == "" {
		http.Error(w, "no destination path recorded", http.StatusBadRequest)
		return
	}

	absSrc, err := filepath.Abs(src)
	if err != nil {
		http.Error(w, fmt.Sprintf("resolve source: %v", err), http.StatusBadRequest)
		return
	}
	absDest, err := filepath.Abs(dest)
	if err != nil {
		http.Error(w, fmt.Sprintf("resolve destination: %v", err), http.StatusBadRequest)
		return
	}
	if absSrc == absDest {
		http.Error(w, "source and destination are the same", http.StatusBadRequest)
		return
	}

	if _, err := os.Stat(absSrc); err != nil {
		http.Error(w, fmt.Sprintf("source missing: %v", err), http.StatusBadRequest)
		return
	}

	if err := os.MkdirAll(filepath.Dir(absDest), 0o755); err != nil {
		http.Error(w, fmt.Sprintf("prepare destination: %v", err), http.StatusInternalServerError)
		return
	}

	if err := os.Rename(absSrc, absDest); err != nil {
		http.Error(w, fmt.Sprintf("move failed: %v", err), http.StatusInternalServerError)
		return
	}

	rec.OutputPath = absDest
	if req.Target != "custom" {
		rec.LocationMode = "proper"
	}
	if err := h.records.Upsert(rec); err != nil {
		h.logger.Warn("failed to update record after move", "error", err)
	}

	if rec.BuildID != "" && h.builds != nil {
		_ = h.builds.Update(rec.BuildID, func(status *BuildStatusView) {
			status.OutputPath = absDest
			if status.Metadata == nil {
				status.Metadata = map[string]interface{}{}
			}
			status.Metadata["moved_to"] = absDest
		})
	}

	writeJSON(w, http.StatusOK, MoveResult{
		RecordID: recordID,
		From:     absSrc,
		To:       absDest,
		Status:   "moved",
	})
}

// DeleteHandler handles DELETE requests to delete a generated desktop application.
func (h *Handler) DeleteHandler(w http.ResponseWriter, r *http.Request) {
	scenarioName := mux.Vars(r)["scenario_name"]
	if scenarioName == "" {
		http.Error(w, "scenario_name is required", http.StatusBadRequest)
		return
	}

	// Validate scenario name to prevent path traversal
	if !validation.IsSafeScenarioName(scenarioName) {
		http.Error(w, "Invalid scenario name", http.StatusBadRequest)
		return
	}

	// Ensure we have a way to compute the desktop path
	if h.outputPathFunc == nil {
		http.Error(w, "delete not configured (no output path function)", http.StatusInternalServerError)
		return
	}

	desktopPath := h.outputPathFunc(scenarioName)

	// Security check: verify path is within expected location
	absDesktopPath, err := filepath.Abs(desktopPath)
	if err != nil {
		http.Error(w, "Failed to resolve desktop path", http.StatusInternalServerError)
		return
	}

	// Ensure path contains "platforms/electron" to prevent accidental deletion
	if !strings.Contains(absDesktopPath, filepath.Join("platforms", "electron")) {
		h.logger.Error("path traversal attempt detected",
			"scenario", scenarioName,
			"path", absDesktopPath)
		http.Error(w, "Security violation: invalid path", http.StatusBadRequest)
		return
	}

	// Check if desktop directory exists
	_, statErr := os.Stat(desktopPath)
	if statErr == nil {
		// Remove the entire platforms/electron directory
		if err := os.RemoveAll(desktopPath); err != nil {
			h.logger.Error("failed to delete desktop directory",
				"scenario", scenarioName,
				"path", desktopPath,
				"error", err)
			http.Error(w, fmt.Sprintf("Failed to delete desktop directory: %v", err), http.StatusInternalServerError)
			return
		}
		h.logger.Info("deleted desktop application",
			"scenario", scenarioName,
			"path", desktopPath)
	} else if !errors.Is(statErr, os.ErrNotExist) {
		http.Error(w, fmt.Sprintf("Failed to read desktop directory: %v", statErr), http.StatusInternalServerError)
		return
	}

	// Clean up associated records
	removedRecords := 0
	if h.records != nil {
		removedRecords = h.records.DeleteByScenario(scenarioName)
	}

	message := fmt.Sprintf("Desktop version of '%s' deleted successfully", scenarioName)
	if errors.Is(statErr, os.ErrNotExist) {
		message = fmt.Sprintf("Desktop version of '%s' was already missing; cleaned up record state.", scenarioName)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":          "success",
		"scenario_name":   scenarioName,
		"deleted_path":    desktopPath,
		"removed_records": removedRecords,
		"message":         message,
	})
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
