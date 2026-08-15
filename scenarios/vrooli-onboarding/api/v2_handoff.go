package main

import (
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strings"
)

const onboardingHandoffBodyLimit = 64 << 10

// onboardingHandoffRequest is deliberately narrower than operator state. The
// bridge sends identity only; onboarding remains the authority for the
// selection that is returned.
type onboardingHandoffRequest struct {
	MachineID string `json:"machine_id"`
	NodeID    string `json:"node_id"`
	NodeKind  string `json:"node_kind"`
}

type onboardingHandoffSelection struct {
	Scenarios         []string                         `json:"scenarios,omitempty"`
	OptionalResources []string                         `json:"optional_resources,omitempty"`
	Host              onboardingHandoffHost            `json:"host,omitempty"`
	OperatingMode     map[string]onboardingHandoffMode `json:"operating_mode,omitempty"`
	Apply             bool                             `json:"apply"`
}

type onboardingHandoffHost struct {
	Tools      []string `json:"tools,omitempty"`
	Safeguards []string `json:"safeguards,omitempty"`
}

type onboardingHandoffMode struct {
	AutoRestart bool `json:"auto_restart"`
}

// handleV2Handoff is the read-only JSON seam consumed by vrooli-bridge. It
// projects the effective, manifest-derived selection into the bridge contract
// without exposing operator-state internals or any credential values.
func (s *Server) handleV2Handoff(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var request onboardingHandoffRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, onboardingHandoffBodyLimit))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid onboarding handoff request: " + err.Error()})
		return
	}
	if strings.TrimSpace(request.NodeID) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid onboarding handoff request: node_id is required"})
		return
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid onboarding handoff request: exactly one JSON object is required"})
		return
	}

	root, err := manifestRoot()
	if err != nil {
		if writeCatalogDegraded(w, err) {
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	state, err := loadOperatorStateFor(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	models, err := loadScenarioReadModels()
	if err != nil {
		if writeCatalogDegraded(w, err) {
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	closure, err := resolveClosureForState(root, models, state)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	requirements, err := deriveV2HostRequirements(root, state, models)
	if err != nil {
		if writeCatalogDegraded(w, err) {
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	selection := onboardingHandoffSelection{Apply: true, OperatingMode: map[string]onboardingHandoffMode{}}
	for _, model := range models {
		if model.Enabled {
			selection.Scenarios = append(selection.Scenarios, model.Name)
		}
		if choice, ok := state.Scenarios[model.Name]; ok && choice.AutoRestart != nil {
			selection.OperatingMode[model.Name] = onboardingHandoffMode{AutoRestart: *choice.AutoRestart}
		}
	}
	for _, member := range closure.Resources {
		if !member.Required {
			selection.OptionalResources = append(selection.OptionalResources, member.Name)
		}
	}
	for _, item := range requirements.Tools {
		if item.Status == "opted_in" {
			selection.Host.Tools = append(selection.Host.Tools, item.Name)
		}
	}
	for _, item := range requirements.Safeguards {
		if item.Status == "opted_in" {
			selection.Host.Safeguards = append(selection.Host.Safeguards, item.Name)
		}
	}
	sort.Strings(selection.Scenarios)
	sort.Strings(selection.OptionalResources)
	sort.Strings(selection.Host.Tools)
	sort.Strings(selection.Host.Safeguards)
	if len(selection.OperatingMode) == 0 {
		selection.OperatingMode = nil
	}
	writeJSON(w, http.StatusOK, selection)
}
