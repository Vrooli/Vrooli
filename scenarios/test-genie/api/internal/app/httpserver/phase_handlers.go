package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"test-genie/internal/orchestrator"
	"test-genie/internal/orchestrator/phases"
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
		known[normalizePhaseName(descriptor.Name)] = struct{}{}
	}

	filtered := orchestrator.PhaseToggleConfig{Phases: map[string]orchestrator.PhaseToggle{}}
	for name, toggle := range payload.Phases {
		key := normalizePhaseName(name)
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

func normalizePhaseName(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}
