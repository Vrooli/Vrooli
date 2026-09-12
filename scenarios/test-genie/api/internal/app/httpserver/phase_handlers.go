package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"test-genie/internal/execution"
	"test-genie/internal/orchestrator"
	"test-genie/internal/orchestrator/phases"
	"test-genie/internal/shared"
)

func (s *Server) handleListPhases(w http.ResponseWriter, r *http.Request) {
	if s.phaseCatalog == nil {
		s.writeError(w, http.StatusInternalServerError, "phase catalog unavailable")
		return
	}
	toggles, err := s.phaseCatalog.GlobalPhaseToggles()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	descriptors := s.phaseCatalog.DescribePhases()
	s.writeJSON(w, http.StatusOK, s.phaseSettingsPayload(descriptors, toggles))
}

func (s *Server) handleInspectPhase(w http.ResponseWriter, r *http.Request) {
	if s.phaseCatalog == nil {
		s.writeError(w, http.StatusInternalServerError, "phase catalog unavailable")
		return
	}
	key := phases.NormalizeKey(mux.Vars(r)["phase"])
	if key == "" {
		s.writeError(w, http.StatusBadRequest, "phase is required")
		return
	}
	for _, descriptor := range s.phaseCatalog.DescribePhases() {
		if phases.NormalizeKey(descriptor.Name) == key {
			s.writeJSON(w, http.StatusOK, map[string]interface{}{"phase": descriptor})
			return
		}
	}
	s.writeError(w, http.StatusNotFound, "phase not found")
}

func (s *Server) handlePreviewPhaseApplicability(w http.ResponseWriter, r *http.Request) {
	target := strings.TrimSpace(r.URL.Query().Get("target"))
	if target == "" {
		s.writeError(w, http.StatusBadRequest, "target is required")
		return
	}
	if s.executionPlanner == nil {
		s.writeError(w, http.StatusInternalServerError, "execution planner unavailable")
		return
	}
	preview, err := s.executionPlanner.Preview(r.Context(), orchestrator.SuiteExecutionRequest{
		// Keep ScenarioName populated for legacy planner adapters while Target
		// carries the contract-backed expression for generalized targets.
		ScenarioName: target,
		Target:       target,
		Preset:       strings.TrimSpace(r.URL.Query().Get("preset")),
	})
	if err != nil {
		var vErr shared.ValidationError
		if errors.As(err, &vErr) {
			s.writeError(w, http.StatusBadRequest, vErr.Error())
			return
		}
		s.log("phase applicability preview failed", map[string]interface{}{"error": err.Error()})
		s.writeError(w, http.StatusInternalServerError, "failed to preview phase applicability")
		return
	}
	phase := phases.NormalizeKey(r.URL.Query().Get("phase"))
	if phase != "" {
		if selected, ok := findPlannedPhase(preview.Phases, phase); ok {
			s.writeJSON(w, http.StatusOK, map[string]interface{}{
				"scenarioName": target,
				"phase":        selected,
				"status":       selected.ApplicabilityStatus,
			})
			return
		}
		if notApplicable, ok := findPlannedPhase(preview.NotApplicablePhases, phase); ok {
			s.writeJSON(w, http.StatusOK, map[string]interface{}{
				"scenarioName": target,
				"phase":        notApplicable,
				"status":       notApplicable.ApplicabilityStatus,
			})
			return
		}
		s.writeError(w, http.StatusNotFound, "phase not found in applicability preview")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"scenarioName":        target,
		"phases":              preview.Phases,
		"notApplicablePhases": preview.NotApplicablePhases,
		"warnings":            preview.Warnings,
	})
}

func (s *Server) handleGetPhaseSettings(w http.ResponseWriter, r *http.Request) {
	if s.phaseCatalog == nil {
		s.writeError(w, http.StatusInternalServerError, "phase catalog unavailable")
		return
	}
	toggles, err := s.phaseCatalog.GlobalPhaseToggles()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	descriptors := s.phaseCatalog.DescribePhases()
	s.writeJSON(w, http.StatusOK, s.phaseSettingsPayload(descriptors, toggles))
}

func (s *Server) handleUpdatePhaseSettings(w http.ResponseWriter, r *http.Request) {
	if s.phaseCatalog == nil {
		s.writeError(w, http.StatusInternalServerError, "phase catalog unavailable")
		return
	}
	var payload struct {
		Phases map[string]orchestrator.PhaseToggle `json:"phases"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}

	known := make(map[string]struct{})
	for _, descriptor := range s.phaseCatalog.DescribePhases() {
		known[phases.NormalizeKey(descriptor.Name)] = struct{}{}
	}

	filtered := orchestrator.PhaseToggleConfig{Phases: map[string]orchestrator.PhaseToggle{}}
	for name, toggle := range payload.Phases {
		key := phases.NormalizeKey(name)
		if key == "" {
			continue
		}
		if _, ok := known[key]; !ok {
			continue
		}
		filtered.Phases[key] = toggle
	}

	saved, err := s.phaseCatalog.SaveGlobalPhaseToggles(filtered)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	descriptors := s.phaseCatalog.DescribePhases()
	s.writeJSON(w, http.StatusOK, s.phaseSettingsPayload(descriptors, saved))
}

func (s *Server) phaseSettingsPayload(descriptors []phases.Descriptor, toggles orchestrator.PhaseToggleConfig) map[string]interface{} {
	phaseToggles := toggles.Phases
	if phaseToggles == nil {
		phaseToggles = map[string]orchestrator.PhaseToggle{}
	}
	return map[string]interface{}{
		"items":   descriptors,
		"count":   len(descriptors),
		"toggles": phaseToggles,
	}
}

func findPlannedPhase(items []execution.PlannedPhase, phase string) (execution.PlannedPhase, bool) {
	for _, item := range items {
		if phases.NormalizeKey(item.Name) == phase {
			return item, true
		}
	}
	return execution.PlannedPhase{}, false
}
