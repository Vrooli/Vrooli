package main

import (
	"encoding/json"
	"net/http"

	"github.com/vrooli/vrooli/internal/operatorstate"
)

func (s *Server) handleV2RecommendationAccept(w http.ResponseWriter, r *http.Request) {
	recommendation, err := buildRecommendation()
	if err != nil {
		if writeCatalogDegraded(w, err) {
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	scenarios := make(map[string]operatorstate.ScenarioChoice, len(recommendation.Scenarios))
	for _, name := range recommendation.Scenarios {
		enabled := true
		scenarios[name] = operatorstate.ScenarioChoice{Enabled: &enabled}
	}
	patch, _ := json.Marshal(map[string]any{"active_profile": recommendation.Profile, "scenarios": scenarios})
	state, err := operatorStateService().Apply(r.Context(), patch)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	step, stepID := sessionPosition(state)
	writeJSON(w, http.StatusOK, sessionResponse{Step: step, StepID: stepID, FirstUnsatisfiedStep: firstUnsatisfiedStep(state), Completion: state.Completion != nil})
}
