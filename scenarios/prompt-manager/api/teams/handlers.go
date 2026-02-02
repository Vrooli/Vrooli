package teams

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"prompt-manager/store"
	"prompt-manager/validation"

	"github.com/gorilla/mux"
)

// Handlers provides HTTP handlers for team operations.
type Handlers struct {
	teamStore     store.TeamStore
	agentStore    store.AgentStore
	relationStore store.RelationStore
	indexStore    store.IndexStore
}

// NewHandlers creates a new teams handler.
func NewHandlers(teamStore store.TeamStore, agentStore store.AgentStore, relationStore store.RelationStore, indexStore store.IndexStore) *Handlers {
	return &Handlers{
		teamStore:     teamStore,
		agentStore:    agentStore,
		relationStore: relationStore,
		indexStore:    indexStore,
	}
}

// List handles GET /teams - returns all teams.
func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	teams, err := h.teamStore.List(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	responses := make([]Response, 0, len(teams))
	for _, t := range teams {
		resp := h.toResponse(ctx, &t)
		responses = append(responses, resp)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(responses)
}

// Get handles GET /teams/{id} - returns a single team with details.
func (h *Handlers) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	id := vars["id"]

	team, err := h.teamStore.Get(ctx, id)
	if err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(h.toDetailsResponse(ctx, team))
}

// Create handles POST /teams - creates a new team.
func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.DisplayName == "" {
		http.Error(w, "displayName is required", http.StatusBadRequest)
		return
	}

	// Generate ID if not provided
	id := req.ID
	if id == "" {
		id = validation.Slugify(req.DisplayName)
	}

	team := &store.Team{
		ID:          id,
		DisplayName: req.DisplayName,
		Mission:     req.Mission,
	}

	if err := h.teamStore.Create(ctx, team); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Regenerate index
	if h.indexStore != nil {
		_ = h.indexStore.RegenerateTeams(ctx)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(h.toDetailsResponse(ctx, team))
}

// Update handles PUT /teams/{id} - updates an existing team.
func (h *Handlers) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	id := vars["id"]

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Build updates
	updates := &store.Team{}
	if req.DisplayName != nil {
		updates.DisplayName = *req.DisplayName
	}
	if req.Mission != nil {
		updates.Mission = *req.Mission
	}

	if err := h.teamStore.Update(ctx, id, updates); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Regenerate index
	if h.indexStore != nil {
		_ = h.indexStore.RegenerateTeams(ctx)
	}

	// Get updated team
	team, _ := h.teamStore.Get(ctx, id)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(h.toDetailsResponse(ctx, team))
}

// Delete handles DELETE /teams/{id} - deletes a team.
func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	id := vars["id"]

	// Delete all team member relations first
	if h.relationStore != nil {
		members, _ := h.relationStore.ListTeamMembers(ctx, id)
		for _, m := range members {
			_ = h.relationStore.DeleteTeamMember(ctx, id, m.AgentID)
		}
	}

	if err := h.teamStore.Delete(ctx, id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Regenerate index
	if h.indexStore != nil {
		_ = h.indexStore.RegenerateTeams(ctx)
	}

	w.WriteHeader(http.StatusNoContent)
}

// AddMember handles POST /teams/{id}/members - adds a member to a team.
func (h *Handlers) AddMember(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	teamID := vars["id"]

	var req AddMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.AgentID == "" {
		http.Error(w, "agentId is required", http.StatusBadRequest)
		return
	}

	// Verify team exists
	if _, err := h.teamStore.Get(ctx, teamID); err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}

	// Verify agent exists
	if _, err := h.agentStore.Get(ctx, req.AgentID); err != nil {
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}

	rel := &store.TeamMemberRelation{
		TeamID:  teamID,
		AgentID: req.AgentID,
		Roles:   req.Roles,
		Status:  store.MemberStatusActive,
	}

	if err := h.relationStore.SetTeamMember(ctx, rel); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Regenerate index
	if h.indexStore != nil {
		_ = h.indexStore.RegenerateTeams(ctx)
	}

	// Get agent for response
	agent, _ := h.agentStore.Get(ctx, req.AgentID)
	member := MemberDTO{
		AgentID:     req.AgentID,
		DisplayName: agent.DisplayName,
		Roles:       req.Roles,
		Status:      store.MemberStatusActive,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(member)
}

// UpdateMember handles PUT /teams/{id}/members/{agentId} - updates a team member.
func (h *Handlers) UpdateMember(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	teamID := vars["id"]
	agentID := vars["agentId"]

	var req UpdateMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get existing membership
	membership, err := h.relationStore.GetTeamMember(ctx, teamID, agentID)
	if err != nil {
		http.Error(w, "Membership not found", http.StatusNotFound)
		return
	}

	// Apply updates
	if req.Roles != nil {
		membership.Roles = req.Roles
	}
	if req.Status != nil {
		membership.Status = *req.Status
	}

	if err := h.relationStore.SetTeamMember(ctx, membership); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get agent for response
	agent, _ := h.agentStore.Get(ctx, agentID)
	member := MemberDTO{
		AgentID:     agentID,
		DisplayName: agent.DisplayName,
		Roles:       membership.Roles,
		Status:      membership.Status,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(member)
}

// RemoveMember handles DELETE /teams/{id}/members/{agentId} - removes a member from a team.
func (h *Handlers) RemoveMember(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	teamID := vars["id"]
	agentID := vars["agentId"]

	if err := h.relationStore.DeleteTeamMember(ctx, teamID, agentID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Regenerate index
	if h.indexStore != nil {
		_ = h.indexStore.RegenerateTeams(ctx)
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetRoles handles GET /teams/{id}/roles - returns team roles.
func (h *Handlers) GetRoles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	id := vars["id"]

	roles, err := h.teamStore.GetRoles(ctx, id)
	if err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}

	roleDTOs := make([]RoleDTO, 0, len(roles.Roles))
	for _, role := range roles.Roles {
		roleDTOs = append(roleDTOs, RoleDTO{
			ID:          role.ID,
			Name:        role.Name,
			Description: role.Description,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(roleDTOs)
}

// SetRoles handles PUT /teams/{id}/roles - sets team roles.
func (h *Handlers) SetRoles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	id := vars["id"]

	var req SetRolesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get the FileTeamStore to access SetRoles (not in interface)
	fileTeamStore, ok := h.teamStore.(*store.FileTeamStore)
	if !ok {
		http.Error(w, "SetRoles not supported", http.StatusInternalServerError)
		return
	}

	roles := &store.TeamRoles{
		TeamID: id,
		Roles:  make([]store.Role, 0, len(req.Roles)),
	}
	for _, r := range req.Roles {
		roles.Roles = append(roles.Roles, store.Role{
			ID:          r.ID,
			Name:        r.Name,
			Description: r.Description,
		})
	}

	if err := fileTeamStore.SetRoles(ctx, id, roles); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(req.Roles)
}

// GetOrgChart handles GET /teams/{id}/org - returns the org chart for a team.
func (h *Handlers) GetOrgChart(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	id := vars["id"]

	// Verify team exists
	if _, err := h.teamStore.Get(ctx, id); err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}

	org, err := h.teamStore.GetOrgChart(ctx, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to DTO response
	resp := OrgChartResponse{
		TeamID: id,
		Edges:  make([]OrgEdgeDTO, 0, len(org.Edges)),
	}
	for _, edge := range org.Edges {
		resp.Edges = append(resp.Edges, OrgEdgeDTO{
			ManagerAgentID: edge.ManagerAgentID,
			ReportAgentID:  edge.ReportAgentID,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// SetOrgChart handles PUT /teams/{id}/org - sets the org chart for a team.
func (h *Handlers) SetOrgChart(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	id := vars["id"]

	var req SetOrgChartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get the FileTeamStore to access SetOrgChart (not in interface)
	fileTeamStore, ok := h.teamStore.(*store.FileTeamStore)
	if !ok {
		http.Error(w, "SetOrgChart not supported", http.StatusInternalServerError)
		return
	}

	org := &store.OrgChart{
		TeamID: id,
		Edges:  make([]store.OrgEdge, 0, len(req.Edges)),
	}
	for _, e := range req.Edges {
		org.Edges = append(org.Edges, store.OrgEdge{
			ManagerAgentID: e.ManagerAgentID,
			ReportAgentID:  e.ReportAgentID,
		})
	}

	if err := fileTeamStore.SetOrgChart(ctx, id, org); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the updated org chart
	resp := OrgChartResponse{
		TeamID: id,
		Edges:  req.Edges,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// Helper functions

func (h *Handlers) toResponse(ctx context.Context, t *store.Team) Response {
	memberCount := 0
	if h.relationStore != nil {
		members, err := h.relationStore.ListTeamMembers(ctx, t.ID)
		if err == nil {
			memberCount = len(members)
		}
	}

	return Response{
		ID:          t.ID,
		DisplayName: t.DisplayName,
		Mission:     t.Mission,
		MemberCount: memberCount,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}

func (h *Handlers) toDetailsResponse(ctx context.Context, t *store.Team) TeamDetailsResponse {
	resp := TeamDetailsResponse{
		Response: h.toResponse(ctx, t),
		Roles:    []RoleDTO{},
		Members:  []MemberDTO{},
	}

	// Get roles
	if roles, err := h.teamStore.GetRoles(ctx, t.ID); err == nil {
		for _, role := range roles.Roles {
			resp.Roles = append(resp.Roles, RoleDTO{
				ID:          role.ID,
				Name:        role.Name,
				Description: role.Description,
			})
		}
	}

	// Get members
	if h.relationStore != nil {
		members, err := h.relationStore.ListTeamMembers(ctx, t.ID)
		if err == nil {
			for _, m := range members {
				memberDTO := MemberDTO{
					AgentID: m.AgentID,
					Roles:   m.Roles,
					Status:  m.Status,
				}
				// Get agent display name
				if h.agentStore != nil {
					if agent, err := h.agentStore.Get(ctx, m.AgentID); err == nil {
						memberDTO.DisplayName = agent.DisplayName
					}
				}
				resp.Members = append(resp.Members, memberDTO)
			}
		}
	}

	// Get defaults
	return resp
}
