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
	// Observed maps an apply item ID to what this host was measured to be in
	// before the plan is shown. The operator asked which items were already
	// applied; without this the plan is a desired-state list that reads as if
	// every entry were a pending change. buildApplyPlan stays pure: the caller
	// does the measuring and passes the result in.
	Observed map[string]string
}

func observedState(observed map[string]string, id string) string {
	if state, ok := observed[id]; ok && state != "" {
		return state
	}
	return applyStateUnknown
}

func plannedRequirementState(status string, observed map[string]string, id string) string {
	if status == "not_applicable" {
		return "not_applicable"
	}
	return observedState(observed, id)
}

// observeApplyStates measures the host for the item kinds that can be checked
// without side effects. Tools resolve through PATH; safeguards resolve through
// their declared verification files, or, when they are handler-owned, through
// the control plane's read-only inspection boundary. This is the same
// inspection the readiness endpoint performs.
//
// Resources and scenarios are reported as unknown. That is a cost verdict, not
// a capability one: sampling them means a control-plane round trip per item
// (`vrooli resource status` alone takes tens of seconds), which is too slow to
// run before every plan render. They are the only remaining unknowns by design.
func observeApplyStates(root string, requirements hostRequirementsResponse) map[string]string {
	observed := map[string]string{}
	for _, tool := range requirements.Tools {
		observed["tool:"+tool.Name] = applyStateFromReadiness(inspectToolReadiness(tool).Status)
	}
	for _, safeguard := range requirements.Safeguards {
		observed["safeguard:"+safeguard.Name] = applyStateFromReadiness(inspectSafeguardReadiness(root, safeguard).Status)
	}
	return observed
}

func applyStateFromReadiness(status string) string {
	switch status {
	case "ready":
		return applyStateSatisfied
	case "missing":
		return applyStatePending
	case "degraded":
		// Configured but not yet in effect (a safeguard awaiting a reboot).
		// "Already in place" would overstate it: the operator still has
		// something to do before the host is actually protected.
		return applyStatePending
	default:
		// "deferred" and "unsupported" both mean this process did not reach a
		// verdict about applying, so neither may be presented as a fact.
		return applyStateUnknown
	}
}

func buildApplyPlan(input applyPlanInput) []applyItem {
	items := make([]applyItem, 0, len(input.Requirements.Tools)+len(input.Requirements.Safeguards)+len(input.Closure.Resources)+len(input.Closure.Scenarios))
	for _, item := range input.Requirements.Tools {
		if item.Status != "required" && item.Status != "opted_in" && item.Status != "not_applicable" {
			continue
		}
		items = append(items, applyItem{ID: "tool:" + item.Name, Kind: "tool", Name: item.Name, Required: item.Required, Privileged: item.Privilege == "elevated", State: plannedRequirementState(item.Status, input.Observed, "tool:"+item.Name)})
	}
	for _, item := range input.Requirements.Safeguards {
		if item.Status != "required" && item.Status != "opted_in" && item.Status != "not_applicable" {
			continue
		}
		items = append(items, applyItem{ID: "safeguard:" + item.Name, Kind: "safeguard", Name: item.Name, Required: item.Required, Privileged: item.Privilege == "elevated", State: plannedRequirementState(item.Status, input.Observed, "safeguard:"+item.Name)})
	}
	for _, member := range input.Closure.Resources {
		if choice, ok := input.State.Resources[member.Name]; ok && choice.Enabled != nil && !*choice.Enabled && !member.Required {
			continue
		}
		items = append(items, applyItem{ID: "resource:" + member.Name, Kind: "resource", Name: member.Name, Required: member.Required, State: observedState(input.Observed, "resource:"+member.Name)})
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
		items = append(items, applyItem{ID: "scenario:" + member.Name, Kind: "scenario", Name: member.Name, Dependencies: dependencies, Required: member.Required || member.Direct, State: observedState(input.Observed, "scenario:"+member.Name)})
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
	hostModels, err := hostRequirementScenarioModels(root, models, state)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	requirements, err := deriveV2HostRequirements(root, state, hostModels)
	if err != nil {
		if writeCatalogDegraded(w, err) {
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": buildApplyPlan(applyPlanInput{Closure: closure, Requirements: requirements, State: state, Observed: observeApplyStates(root, requirements)})})
}
