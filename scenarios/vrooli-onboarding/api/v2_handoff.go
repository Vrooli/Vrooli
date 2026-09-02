package main

import (
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strings"

	setupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/setup/v1"
)

const onboardingHandoffBodyLimit = 64 << 10

// onboardingHandoffRequest is deliberately narrower than operator state. The
// bridge sends identity only; onboarding remains the authority for the
// selection that is returned.
type onboardingHandoffRequest struct {
	MachineID        string                      `json:"machine_id"`
	NodeID           string                      `json:"node_id"`
	NodeKind         string                      `json:"node_kind"`
	DesiredSelection *onboardingHandoffSelection `json:"desired_selection,omitempty"`
}

type onboardingHandoffSelection = setupv1.Selection

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
	if request.DesiredSelection != nil {
		selection := *request.DesiredSelection
		selection.Apply = true
		sort.Strings(selection.Scenarios)
		sort.Strings(selection.OptionalResources)
		sort.Strings(selection.HostTools)
		sort.Strings(selection.HostSafeguards)
		if len(selection.OperatingMode) == 0 {
			selection.OperatingMode = nil
		}
		writeJSON(w, http.StatusOK, selection)
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

	selection := onboardingHandoffSelection{SchemaVersion: "v1", Apply: true, OperatingMode: map[string]string{}}
	for _, model := range models {
		if model.Enabled {
			selection.Scenarios = append(selection.Scenarios, model.Name)
		}
		if choice, ok := state.Scenarios[model.Name]; ok && choice.AutoRestart != nil {
			selection.OperatingMode[model.Name] = map[bool]string{true: "auto-restart", false: "manual"}[*choice.AutoRestart]
		}
	}
	for _, member := range closure.Resources {
		if !member.Required {
			selection.OptionalResources = append(selection.OptionalResources, member.Name)
		}
	}
	for _, item := range requirements.Tools {
		if item.Status == "opted_in" {
			selection.HostTools = append(selection.HostTools, item.Name)
		}
	}
	for _, item := range requirements.Safeguards {
		if item.Status == "opted_in" {
			selection.HostSafeguards = append(selection.HostSafeguards, item.Name)
		}
	}
	sort.Strings(selection.Scenarios)
	sort.Strings(selection.OptionalResources)
	sort.Strings(selection.HostTools)
	sort.Strings(selection.HostSafeguards)
	if len(selection.OperatingMode) == 0 {
		selection.OperatingMode = nil
	}
	writeJSON(w, http.StatusOK, selection)
}
