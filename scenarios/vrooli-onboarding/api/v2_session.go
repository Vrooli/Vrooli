package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/vrooli/vrooli/internal/operatorstate"
)

type sessionResponse struct {
	Step                 int    `json:"step"`
	StepID               string `json:"step_id,omitempty"`
	FirstUnsatisfiedStep int    `json:"first_unsatisfied_step"`
	Completion           bool   `json:"completion"`
}

func stepIDAt(ordinal int) string {
	for _, step := range onboardingSteps {
		if step.Ordinal == ordinal {
			return step.ID
		}
	}
	return ""
}

func stepOrdinalForID(id string) (int, bool) {
	for _, step := range onboardingSteps {
		if step.ID == id {
			return step.Ordinal, true
		}
	}
	return 0, false
}

func sessionPosition(state OperatorState) (int, string) {
	if state.Session == nil {
		return 0, stepIDAt(0)
	}
	if id := strings.TrimSpace(state.Session.StepID); id != "" {
		if ordinal, ok := stepOrdinalForID(id); ok {
			return ordinal, id
		}
	}
	return state.Session.Step, stepIDAt(state.Session.Step)
}

func (s *Server) handleV2Session(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		state, err := loadOperatorStateFor(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		step, stepID := sessionPosition(state)
		if state.Session != nil && state.Session.StepID == "" && stepID != "" {
			patch, _ := json.Marshal(map[string]any{"session": operatorstate.Session{Step: step, StepID: stepID}})
			if migrated, migrationErr := operatorStateService().Apply(r.Context(), patch); migrationErr == nil {
				_ = migrated
				slog.Info("onboarding session migrated", "step", step, "step_id", stepID)
			}
		}
		first := firstUnsatisfiedStep(state)
		writeJSON(w, http.StatusOK, sessionResponse{Step: step, StepID: stepID, FirstUnsatisfiedStep: first, Completion: state.Completion != nil})
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
	patch, _ := json.Marshal(map[string]any{"session": operatorstate.Session{Step: request.Step, StepID: stepIDAt(request.Step)}})
	state, err := operatorStateService().Apply(r.Context(), patch)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	step, stepID := sessionPosition(state)
	writeJSON(w, http.StatusOK, sessionResponse{Step: step, StepID: stepID, FirstUnsatisfiedStep: firstUnsatisfiedStep(state), Completion: state.Completion != nil})
}
