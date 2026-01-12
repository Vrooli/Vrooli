package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func (s *Server) smokeTestStartHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		ScenarioName string `json:"scenario_name"`
		Platform     string `json:"platform"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid JSON payload", http.StatusBadRequest)
		return
	}

	if request.ScenarioName == "" {
		http.Error(w, "scenario_name is required", http.StatusBadRequest)
		return
	}
	if !isSafeScenarioName(request.ScenarioName) {
		http.Error(w, "invalid scenario_name", http.StatusBadRequest)
		return
	}

	currentPlatform := currentSmokeTestPlatform()
	if request.Platform != "" && request.Platform != currentPlatform {
		http.Error(w, fmt.Sprintf("platform mismatch: server is %s", currentPlatform), http.StatusBadRequest)
		return
	}

	desktopPath := s.standardOutputPath(request.ScenarioName)
	distPath := filepath.Join(desktopPath, "dist-electron")
	artifactPath, err := s.findBuiltPackage(distPath, currentPlatform)
	if err != nil {
		http.Error(w, fmt.Sprintf("no matching installer for %s: %v", currentPlatform, err), http.StatusNotFound)
		return
	}

	smokeTestID := uuid.New().String()
	status := &SmokeTestStatus{
		SmokeTestID:  smokeTestID,
		ScenarioName: request.ScenarioName,
		Platform:     currentPlatform,
		Status:       "running",
		ArtifactPath: artifactPath,
		StartedAt:    time.Now(),
		Logs: []string{
			fmt.Sprintf("Detected platform: %s", currentPlatform),
			fmt.Sprintf("Matched artifact: %s", filepath.Base(artifactPath)),
		},
	}
	s.smokeTests.Save(status)

	ctx, cancel := context.WithCancel(context.Background())
	s.setSmokeTestCancel(smokeTestID, cancel)
	go s.performSmokeTest(ctx, smokeTestID, request.ScenarioName, artifactPath, currentPlatform)

	writeJSONResponse(w, http.StatusOK, status)
}

func (s *Server) smokeTestStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["smoke_test_id"]
	if id == "" {
		http.Error(w, "smoke_test_id is required", http.StatusBadRequest)
		return
	}

	status, ok := s.smokeTests.Get(id)
	if !ok {
		http.Error(w, "smoke test not found", http.StatusNotFound)
		return
	}

	writeJSONResponse(w, http.StatusOK, status)
}

func (s *Server) smokeTestCancelHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["smoke_test_id"]
	if id == "" {
		http.Error(w, "smoke_test_id is required", http.StatusBadRequest)
		return
	}

	cancel := s.takeSmokeTestCancel(id)
	if cancel == nil {
		if _, ok := s.smokeTests.Get(id); ok {
			s.smokeTests.Update(id, func(status *SmokeTestStatus) {
				status.Status = "failed"
				status.Error = "smoke test cancel requested but no running process was found"
				now := time.Now()
				status.CompletedAt = &now
			})
			writeJSONResponse(w, http.StatusOK, map[string]string{
				"status": "cancelled",
			})
			return
		}
		http.Error(w, "smoke test not found", http.StatusNotFound)
		return
	}

	cancel()
	writeJSONResponse(w, http.StatusOK, map[string]string{
		"status": "cancelling",
	})
}
