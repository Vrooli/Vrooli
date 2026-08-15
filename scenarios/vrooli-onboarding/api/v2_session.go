package main

import (
	"encoding/json"
	"net/http"

	"github.com/vrooli/vrooli/internal/operatorstate"
)

type sessionResponse struct {
	Step                 int  `json:"step"`
	FirstUnsatisfiedStep int  `json:"first_unsatisfied_step"`
	Completion           bool `json:"completion"`
}

func (s *Server) handleV2Session(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		state, err := loadOperatorStateFor(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		step := 0
		if state.Session != nil {
			step = state.Session.Step
		}
		first := firstUnsatisfiedStep(state)
		writeJSON(w, http.StatusOK, sessionResponse{Step: step, FirstUnsatisfiedStep: first, Completion: state.Completion != nil})
		return
	}

	var request struct {
		Step int `json:"step"`
	}
	if !decodeJSONBody(w, r, &request) {
		return
	}
	if request.Step < 0 || request.Step >= len(onboardingSteps) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "step is outside the onboarding step model"})
		return
	}
	patch, _ := json.Marshal(map[string]any{"session": operatorstate.Session{Step: request.Step}})
	state, err := operatorStateService().Apply(r.Context(), patch)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, sessionResponse{Step: state.Session.Step, FirstUnsatisfiedStep: firstUnsatisfiedStep(state), Completion: state.Completion != nil})
}
