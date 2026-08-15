package main

import (
	"net/http"
	"sort"
)

// recommendationResponse is the shared default for the browser and terminal
// surfaces. It is derived from manifests, so a new deployment does not need a
// second hard-coded starter list in onboarding.
type recommendationResponse struct {
	Profile     string   `json:"profile"`
	Scenarios   []string `json:"scenarios"`
	Resources   []string `json:"resources"`
	Explanation string   `json:"explanation"`
}

func buildRecommendation() (recommendationResponse, error) {
	root, err := manifestRoot()
	if err != nil {
		return recommendationResponse{}, err
	}
	models, err := loadScenarioReadModels()
	if err != nil {
		return recommendationResponse{}, err
	}
	state := OperatorState{Scenarios: map[string]ScenarioChoice{}}
	for _, model := range models {
		if model.SystemRequired {
			enabled := true
			state.Scenarios[model.Name] = ScenarioChoice{Enabled: &enabled}
		}
	}
	if len(state.Scenarios) == 0 && len(models) > 0 {
		enabled := true
		state.Scenarios[models[0].Name] = ScenarioChoice{Enabled: &enabled}
	}
	closure, err := resolveClosureForState(root, models, state)
	if err != nil {
		return recommendationResponse{}, err
	}
	scenarios := make([]string, 0, len(state.Scenarios))
	for name := range state.Scenarios {
		scenarios = append(scenarios, name)
	}
	resources := make([]string, 0, len(closure.Resources))
	for _, member := range closure.Resources {
		if member.Required {
			resources = append(resources, member.Name)
		}
	}
	sort.Strings(scenarios)
	sort.Strings(resources)
	return recommendationResponse{
		Profile:     "starter",
		Scenarios:   scenarios,
		Resources:   resources,
		Explanation: "The starter profile contains system-required capabilities and their required dependency closure. Accept it to finish with no scenario names to type.",
	}, nil
}

func (s *Server) handleV2Recommendation(w http.ResponseWriter, _ *http.Request) {
	recommendation, err := buildRecommendation()
	if err != nil {
		if writeCatalogDegraded(w, err) {
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, recommendation)
}
