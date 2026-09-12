package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

type agentRole struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Intent      string `json:"intent"`
	Description string `json:"description"`
	Source      string `json:"source"`
}

func (s *Server) handleListAgentRoles(w http.ResponseWriter, r *http.Request) {
	if !s.agentService.IsAvailable(r.Context()) {
		s.writeError(w, http.StatusServiceUnavailable, "agent-manager is not available")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()
	response, err := s.agentService.GetRoleCatalog(ctx)
	if err != nil {
		s.writeError(w, http.StatusBadGateway, fmt.Sprintf("failed to load agent roles: %s", err))
		return
	}
	roles := make([]agentRole, 0, len(response.GetCatalog().GetRoles()))
	for _, role := range response.GetCatalog().GetRoles() {
		id := strings.TrimSpace(role.GetRoleRef())
		if id == "" {
			continue
		}
		label := strings.TrimSpace(role.GetDescription())
		if label == "" {
			label = id
		}
		roles = append(roles, agentRole{ID: id, Label: label, Intent: strings.TrimSpace(role.GetIntent()), Description: strings.TrimSpace(role.GetDescription()), Source: "agent-manager-role-policy"})
	}
	sort.Slice(roles, func(i, j int) bool { return roles[i].ID < roles[j].ID })
	s.writeJSON(w, http.StatusOK, map[string]any{"items": roles, "count": len(roles)})
}
