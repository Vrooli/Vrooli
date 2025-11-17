package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type ScenarioExistenceResponse struct {
	Exists       bool   `json:"exists"`
	Path         string `json:"path,omitempty"`
	LastModified string `json:"last_modified,omitempty"`
}

func handleScenarioExistence(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := strings.TrimSpace(vars["name"])
	if name == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "scenario name is required"})
		return
	}
	if strings.Contains(name, "..") {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid scenario name"})
		return
	}

	vrooliRoot, err := getVrooliRoot()
	if err != nil {
		respondInternalError(w, "failed to resolve Vrooli root", err)
		return
	}

	scenarioPath := filepath.Join(vrooliRoot, "scenarios", name)
	info, err := os.Stat(scenarioPath)
	if err != nil {
		if os.IsNotExist(err) {
			respondJSON(w, http.StatusOK, ScenarioExistenceResponse{Exists: false})
			return
		}
		respondInternalError(w, "failed to inspect scenario path", err)
		return
	}

	if !info.IsDir() {
		respondJSON(w, http.StatusOK, ScenarioExistenceResponse{Exists: false})
		return
	}

	respondJSON(w, http.StatusOK, ScenarioExistenceResponse{
		Exists:       true,
		Path:         scenarioPath,
		LastModified: info.ModTime().Format(time.RFC3339),
	})
}
