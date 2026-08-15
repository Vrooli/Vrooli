package main

import (
	"net/http"
)

// applyPlanInput is the single input to planning. The review endpoint and the
// executor both consume the result of buildApplyPlan; neither surface derives
// its own list of host mutations.
type applyPlanInput struct {
	Closure      closureResult
	Requirements hostRequirementsResponse
	State        OperatorState
}

func buildApplyPlan(input applyPlanInput) []applyItem {
	items := make([]applyItem, 0, len(input.Requirements.Tools)+len(input.Requirements.Safeguards)+len(input.Closure.Resources)+len(input.Closure.Scenarios))
	for _, item := range input.Requirements.Tools {
		if item.Status != "required" && item.Status != "opted_in" {
			continue
		}
		items = append(items, applyItem{ID: "tool:" + item.Name, Kind: "tool", Name: item.Name, Required: item.Required, Privileged: item.Privilege == "elevated"})
	}
	for _, item := range input.Requirements.Safeguards {
		if item.Status != "required" && item.Status != "opted_in" {
			continue
		}
		items = append(items, applyItem{ID: "safeguard:" + item.Name, Kind: "safeguard", Name: item.Name, Required: item.Required, Privileged: item.Privilege == "elevated"})
	}
	for _, member := range input.Closure.Resources {
		if choice, ok := input.State.Resources[member.Name]; ok && choice.Enabled != nil && !*choice.Enabled && !member.Required {
			continue
		}
		items = append(items, applyItem{ID: "resource:" + member.Name, Kind: "resource", Name: member.Name, Required: member.Required})
	}
	for _, member := range input.Closure.Scenarios {
		dependencies := make([]string, 0)
		for _, resource := range input.Closure.Resources {
			for _, provenance := range resource.Provenance {
				if provenance.From == member.Name {
					dependencies = append(dependencies, "resource:"+resource.Name)
					break
				}
			}
		}
		items = append(items, applyItem{ID: "scenario:" + member.Name, Kind: "scenario", Name: member.Name, Dependencies: dependencies, Required: member.Required || member.Direct})
	}
	return items
}

func (s *Server) handleV2ApplyPlan(w http.ResponseWriter, r *http.Request) {
	root, err := manifestRoot()
	if err != nil {
		if writeCatalogDegraded(w, err) {
			return
		}
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
	state, err := loadOperatorStateFor(r.Context())
	if err != nil {
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
	writeJSON(w, http.StatusOK, map[string]any{"items": buildApplyPlan(applyPlanInput{Closure: closure, Requirements: requirements, State: state})})
}
