package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

type scenarioSummary struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Version     string   `json:"version,omitempty"`
	Status      string   `json:"status,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Path        string   `json:"path,omitempty"`
}

type scenarioListPayload struct {
	Success   bool              `json:"success"`
	Scenarios []scenarioSummary `json:"scenarios"`
}

type ScenarioHandlers struct{}

func NewScenarioHandlers() *ScenarioHandlers {
	return &ScenarioHandlers{}
}

func (s *APIServer) scenarioListHandler(w http.ResponseWriter, r *http.Request) {
	s.handlers.scenarios.ScenarioList(w, r)
}

func (h *ScenarioHandlers) ScenarioList(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	scenarios, err := fetchScenarioList(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to load scenarios: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"scenarios": scenarios,
		"count":     len(scenarios),
	})
}

func fetchScenarioList(ctx context.Context) ([]scenarioSummary, error) {
	cmd := exec.CommandContext(ctx, "vrooli", "scenario", "list", "--json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("vrooli scenario list failed: %w", err)
	}

	var payload scenarioListPayload
	if err := json.Unmarshal(output, &payload); err != nil {
		return nil, fmt.Errorf("unable to parse scenario list: %w", err)
	}
	if !payload.Success && len(payload.Scenarios) == 0 {
		return nil, fmt.Errorf("scenario list did not return results")
	}

	// Normalize names to avoid trailing slashes or whitespace from CLI output
	for i, scenario := range payload.Scenarios {
		payload.Scenarios[i].Name = strings.TrimSpace(scenario.Name)
		payload.Scenarios[i].Description = strings.TrimSpace(scenario.Description)
		payload.Scenarios[i].Version = strings.TrimSpace(scenario.Version)
		payload.Scenarios[i].Status = strings.TrimSpace(scenario.Status)
		payload.Scenarios[i].Path = strings.TrimSpace(scenario.Path)
	}

	return payload.Scenarios, nil
}
