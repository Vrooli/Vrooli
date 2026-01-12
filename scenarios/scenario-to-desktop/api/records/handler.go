package records

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
)

// Handler provides HTTP handlers for record endpoints.
type Handler struct {
	records Store
	builds  BuildStoreAdapter
	logger  *slog.Logger
}

// BuildStoreAdapter adapts the build store interface for record operations.
type BuildStoreAdapter interface {
	Get(id string) (*BuildStatusView, bool)
	Update(id string, fn func(status *BuildStatusView)) error
}

// NewHandler creates a new records handler.
func NewHandler(records Store, builds BuildStoreAdapter, logger *slog.Logger) *Handler {
	return &Handler{
		records: records,
		builds:  builds,
		logger:  logger,
	}
}

// RegisterRoutes registers record routes on the given router.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/desktop/records", h.ListHandler).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/v1/desktop/records/{record_id}/move", h.MoveHandler).Methods("POST", "OPTIONS")
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

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
