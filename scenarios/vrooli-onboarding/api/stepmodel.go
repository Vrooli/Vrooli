package main

import (
	"net/http"
)

type onboardingStep struct {
	ID        string
	Ordinal   int
	Title     string
	Route     string
	Deferred  bool
	Satisfied func(OperatorState) bool
}

var onboardingSteps = []onboardingStep{
	{ID: "welcome", Ordinal: 0, Title: "Welcome", Route: "/setup/welcome", Satisfied: func(s OperatorState) bool { return s.Session != nil }},
	{ID: "scenarios", Ordinal: 1, Title: "Scenarios", Route: "/setup/scenarios", Satisfied: func(s OperatorState) bool {
		for _, choice := range s.Scenarios {
			if choice.Enabled != nil && *choice.Enabled {
				return true
			}
		}
		return false
	}},
	{ID: "core-set", Ordinal: 2, Title: "Core Set", Route: "/setup/core-set", Satisfied: func(s OperatorState) bool { return s.Core != nil && len(s.Core.Seed) > 0 }},
	{ID: "resources", Ordinal: 3, Title: "Resources", Route: "/setup/resources", Satisfied: func(s OperatorState) bool { return s.Resources != nil }},
	{ID: "credentials", Ordinal: 4, Title: "Credentials", Route: "/setup/credentials", Satisfied: func(s OperatorState) bool { return s.Scenarios != nil }},
	{ID: "integrations", Ordinal: 5, Title: "Integrations", Route: "/setup/integrations", Deferred: true, Satisfied: func(s OperatorState) bool { return s.Version != "" }},
	{ID: "host", Ordinal: 6, Title: "Host", Route: "/setup/host", Satisfied: func(s OperatorState) bool { return s.HostTools != nil || s.HostSafeguards != nil }},
	{ID: "operating-mode", Ordinal: 7, Title: "Operating Mode", Route: "/setup/operating-mode", Satisfied: func(s OperatorState) bool { return s.Scenarios != nil }},
	{ID: "apply", Ordinal: 8, Title: "Apply", Route: "/setup/apply", Satisfied: func(s OperatorState) bool { return s.Completion != nil }},
	{ID: "validation", Ordinal: 9, Title: "Validation", Route: "/setup/validation", Satisfied: func(s OperatorState) bool { return s.Completion != nil }},
}

type stepModelResponse struct {
	ID       string `json:"id"`
	Ordinal  int    `json:"ordinal"`
	Title    string `json:"title"`
	Route    string `json:"route"`
	Deferred bool   `json:"deferred"`
}

func publicStepModel() []stepModelResponse {
	model := make([]stepModelResponse, 0, len(onboardingSteps))
	for _, step := range onboardingSteps {
		model = append(model, stepModelResponse{ID: step.ID, Ordinal: step.Ordinal, Title: step.Title, Route: step.Route, Deferred: step.Deferred})
	}
	return model
}

func firstUnsatisfiedStep(state OperatorState) int {
	for _, step := range onboardingSteps {
		if !step.Satisfied(state) {
			return step.Ordinal
		}
	}
	return len(onboardingSteps) - 1
}

func (s *Server) handleV2Steps(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"steps": publicStepModel()})
}
