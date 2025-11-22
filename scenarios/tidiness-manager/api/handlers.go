package main

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"time"
)

// LightScanRequest defines the request body for light scanning
type LightScanRequest struct {
	ScenarioPath string `json:"scenario_path"`
	TimeoutSec   int    `json:"timeout_sec,omitempty"`
}

// ParseRequest defines request for parsing lint/type output
type ParseRequest struct {
	Scenario string `json:"scenario"`
	Tool     string `json:"tool"`
	Output   string `json:"output"`
}

// handleLightScan performs a complete light scan on a scenario
func (s *Server) handleLightScan(w http.ResponseWriter, r *http.Request) {
	var req LightScanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ScenarioPath == "" {
		respondError(w, http.StatusBadRequest, "scenario_path is required")
		return
	}

	// Validate path exists
	absPath, err := filepath.Abs(req.ScenarioPath)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid scenario_path")
		return
	}

	timeout := 120 * time.Second
	if req.TimeoutSec > 0 {
		timeout = time.Duration(req.TimeoutSec) * time.Second
	}

	scanner := NewLightScanner(absPath, timeout)
	result, err := scanner.Scan(r.Context())
	if err != nil {
		s.log("light scan failed", map[string]interface{}{
			"error":    err.Error(),
			"scenario": absPath,
		})
		respondError(w, http.StatusInternalServerError, "scan failed: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// handleParseLint parses lint output into structured issues
func (s *Server) handleParseLint(w http.ResponseWriter, r *http.Request) {
	var req ParseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Scenario == "" || req.Tool == "" {
		respondError(w, http.StatusBadRequest, "scenario and tool are required")
		return
	}

	issues := ParseLintOutput(req.Scenario, req.Tool, req.Output)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"issues": issues,
		"count":  len(issues),
	})
}

// handleParseType parses type checker output into structured issues
func (s *Server) handleParseType(w http.ResponseWriter, r *http.Request) {
	var req ParseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Scenario == "" || req.Tool == "" {
		respondError(w, http.StatusBadRequest, "scenario and tool are required")
		return
	}

	issues := ParseTypeOutput(req.Scenario, req.Tool, req.Output)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"issues": issues,
		"count":  len(issues),
	})
}

// respondJSON writes a JSON response
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondError writes an error response
func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{
		"error": message,
	})
}
