// Package main provides desktop record management handlers.
// This file contains handlers for listing, moving, and managing
// persisted records of desktop app generations.
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
)

// ============================================================================
// HTTP Handlers
// ============================================================================

// listDesktopRecordsHandler returns the persisted desktop generation records.
func (s *Server) listDesktopRecordsHandler(w http.ResponseWriter, _ *http.Request) {
	if s.records == nil {
		http.Error(w, "record store unavailable", http.StatusInternalServerError)
		return
	}

	type recordWithBuild struct {
		Record     *DesktopAppRecord `json:"record"`
		Build      *BuildStatus      `json:"build_status,omitempty"`
		HasBuild   bool              `json:"has_build"`
		BuildState string            `json:"build_state,omitempty"`
	}

	var results []recordWithBuild
	for _, rec := range s.records.List() {
		item := recordWithBuild{Record: rec}
		if rec != nil && rec.BuildID != "" {
			if bs, ok := s.builds.Get(rec.BuildID); ok {
				item.Build = bs
				item.HasBuild = true
				item.BuildState = bs.Status
			}
		}
		results = append(results, item)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"records": results,
	})
}

// moveDesktopRecordHandler moves a generated desktop wrapper from its current path to a destination.
func (s *Server) moveDesktopRecordHandler(w http.ResponseWriter, r *http.Request) {
	if s.records == nil {
		http.Error(w, "record store unavailable", http.StatusInternalServerError)
		return
	}

	recordID := mux.Vars(r)["record_id"]
	rec, ok := s.records.Get(recordID)
	if !ok || rec == nil {
		http.Error(w, "record not found", http.StatusNotFound)
		return
	}

	var req struct {
		Target          string `json:"target"`           // "destination" (default) or "custom"
		DestinationPath string `json:"destination_path"` // required when target == "custom"
	}
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
	if err := s.records.Upsert(rec); err != nil {
		s.logger.Warn("failed to update record after move", "error", err)
	}

	if rec.BuildID != "" {
		_ = s.builds.Update(rec.BuildID, func(status *BuildStatus) {
			status.OutputPath = absDest
			if status.Metadata == nil {
				status.Metadata = map[string]interface{}{}
			}
			status.Metadata["moved_to"] = absDest
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"record_id": recordID,
		"from":      absSrc,
		"to":        absDest,
		"status":    "moved",
	})
}

// Helper functions listScenarioRecords and recordOutputPath are defined in handlers_scenario.go
