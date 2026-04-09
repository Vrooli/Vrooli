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

	httputil "scenario-to-desktop-api/shared/http"
	"scenario-to-desktop-api/shared/validation"
)

// Handler provides HTTP handlers for record endpoints.
type Handler struct {
	records        Store
	builds         BuildStoreAdapter
	smokeTests     SmokeTestStoreAdapter
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

// WithSmokeTestStore sets the smoke test store for enriching records with video data.
func WithSmokeTestStore(store SmokeTestStoreAdapter) HandlerOption {
	return func(h *Handler) {
		h.smokeTests = store
	}
}

// SetSmokeTestStore sets the smoke test store after construction.
// This is used when the smoke test store is created after the handler.
func (h *Handler) SetSmokeTestStore(store SmokeTestStoreAdapter) {
	h.smokeTests = store
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
		if rec != nil && h.smokeTests != nil {
			if stID, sr, ok := h.smokeTests.GetByScenario(rec.ScenarioName); ok {
				item.SmokeTestID = stID
				item.ScreenRecording = sr
			}
		}
		results = append(results, item)
	}

	httputil.WriteJSON(w, http.StatusOK, ListResponse{Records: results})
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

	absSrc, absDest, err := h.resolveMovePaths(rec, &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := performMove(absSrc, absDest); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.updateRecordAfterMove(rec, &req, absDest)

	httputil.WriteJSON(w, http.StatusOK, MoveResult{
		RecordID: recordID,
		From:     absSrc,
		To:       absDest,
		Status:   "moved",
	})
}

// resolveMovePaths validates and resolves absolute source and destination paths for a move.
func (h *Handler) resolveMovePaths(rec *DesktopAppRecord, req *MoveRequest) (string, string, error) {
	if req.Target == "" {
		req.Target = "destination"
	}

	dest := rec.DestinationPath
	if req.Target == "custom" {
		if req.DestinationPath == "" {
			return "", "", fmt.Errorf("destination_path required for custom target")
		}
		dest = req.DestinationPath
	}
	if dest == "" {
		return "", "", fmt.Errorf("no destination path recorded")
	}

	absSrc, err := filepath.Abs(rec.OutputPath)
	if err != nil {
		return "", "", fmt.Errorf("resolve source: %v", err)
	}
	absDest, err := filepath.Abs(dest)
	if err != nil {
		return "", "", fmt.Errorf("resolve destination: %v", err)
	}
	if absSrc == absDest {
		return "", "", fmt.Errorf("source and destination are the same")
	}
	if _, err := os.Stat(absSrc); err != nil {
		return "", "", fmt.Errorf("source missing: %v", err)
	}
	return absSrc, absDest, nil
}

// performMove creates the destination directory and renames the source.
func performMove(absSrc, absDest string) error {
	if err := os.MkdirAll(filepath.Dir(absDest), 0o755); err != nil {
		return fmt.Errorf("prepare destination: %v", err)
	}
	if err := os.Rename(absSrc, absDest); err != nil {
		return fmt.Errorf("move failed: %v", err)
	}
	return nil
}

// updateRecordAfterMove persists path changes to the record and build stores.
func (h *Handler) updateRecordAfterMove(rec *DesktopAppRecord, req *MoveRequest, absDest string) {
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

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"status":          "success",
		"scenario_name":   scenarioName,
		"deleted_path":    desktopPath,
		"removed_records": removedRecords,
		"message":         message,
	})
}
