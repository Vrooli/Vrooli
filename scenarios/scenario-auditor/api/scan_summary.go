package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
)

type scanSummaryLookup func(jobID string) (*ViolationSummary, bool)

type scanArtifactLookup func(jobID string) (path string, jobFound, artifactFound bool)

func serveScanSummary(w http.ResponseWriter, r *http.Request, scanName string, notFoundErr error, lookup scanSummaryLookup) {
	jobID := strings.TrimSpace(mux.Vars(r)["jobId"])
	if jobID == "" {
		HTTPError(w, "Job ID is required", http.StatusBadRequest, nil)
		return
	}

	summary, found := lookup(jobID)
	if !found {
		HTTPError(w, scanName+" not found", http.StatusNotFound, notFoundErr)
		return
	}
	if summary == nil {
		HTTPError(w, "Summary not available; scan still running", http.StatusAccepted, nil)
		return
	}

	limit, minSeverity, groupBy := parseSummaryFilters(r)
	filtered := cloneSummary(summary, limit, minSeverity)
	if filtered == nil {
		HTTPError(w, "Summary unavailable", http.StatusInternalServerError, nil)
		return
	}

	response := map[string]any{
		"summary": filtered,
		"filters": map[string]any{
			"limit":        limit,
			"min_severity": minSeverity,
			"group_by":     groupBy,
		},
	}
	if groupBy == "rule" {
		response["groups"] = filtered.ByRule
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func serveScanArtifact(w http.ResponseWriter, r *http.Request, filenamePrefix, notFoundMessage string, notFoundErr error, lookup scanArtifactLookup) {
	jobID := strings.TrimSpace(mux.Vars(r)["jobId"])
	if jobID == "" {
		HTTPError(w, "Job ID is required", http.StatusBadRequest, nil)
		return
	}
	artifactPath, jobFound, artifactFound := lookup(jobID)
	if !jobFound {
		HTTPError(w, notFoundMessage, http.StatusNotFound, notFoundErr)
		return
	}
	if !artifactFound {
		HTTPError(w, "No artifact recorded for this scan", http.StatusNotFound, nil)
		return
	}

	artifactPath, err := resolveArtifactAbsolutePath(artifactPath)
	if err != nil {
		HTTPError(w, "Artifact path invalid", http.StatusBadRequest, err)
		return
	}
	file, err := os.Open(artifactPath)
	if err != nil {
		HTTPError(w, "Artifact not found", http.StatusNotFound, err)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		HTTPError(w, "Unable to read artifact", http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s-%s.json", filenamePrefix, jobID))
	http.ServeContent(w, r, filepath.Base(artifactPath), info.ModTime(), file)
}
