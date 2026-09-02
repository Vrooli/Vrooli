package main

import (
	"net/http"
	"time"

	"github.com/vrooli/api-core/nodereach"
)

type onboardingTarget struct {
	ID     string `json:"id"`
	Name   string `json:"name,omitempty"`
	Status string `json:"status,omitempty"`
}

func (s *Server) handleV2Targets(w http.ResponseWriter, r *http.Request) {
	targets := []onboardingTarget{{ID: "local", Name: "This machine", Status: "local"}}
	if s.bridge == nil {
		s.bridge = nodereach.New(nodereach.Config{})
	}
	nodes, err := s.bridge.List(r.Context(), 5*time.Second)
	if err != nil {
		// Local onboarding remains usable when Bridge is not running. The
		// response says why remote choices are absent instead of inventing a
		// stale node list.
		w.Header().Set("X-Vrooli-Target-Discovery", "bridge-unavailable")
		writeJSON(w, http.StatusOK, map[string]any{"targets": targets, "error": err.Error()})
		return
	}
	for _, node := range nodes {
		if node == nil || node.GetId() == "" {
			continue
		}
		targets = append(targets, onboardingTarget{ID: node.GetId(), Name: node.GetName(), Status: node.GetStatus().String()})
	}
	writeJSON(w, http.StatusOK, map[string]any{"targets": targets})
}
