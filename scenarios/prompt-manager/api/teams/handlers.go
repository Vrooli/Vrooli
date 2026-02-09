package teams

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"prompt-manager/store"
	"prompt-manager/validation"

	"github.com/gorilla/mux"
)

// defaultReadCCConfig reads a Claude Code team config from the standard location.
func defaultReadCCConfig(teamName string) ([]byte, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to determine home directory: %w", err)
	}
	configPath := filepath.Join(homeDir, ".claude", "teams", teamName, "config.json")
	return os.ReadFile(configPath)
}

// Handlers provides HTTP handlers for team operations.
type Handlers struct {
	teamStore          store.TeamStore
	agentStore         store.AgentStore
	relationStore      store.RelationStore
	indexStore         store.IndexStore
	heartbeatScheduler HeartbeatScheduler
	readCCConfig       func(teamName string) ([]byte, error) // Testing seam for CC config reader
	listCCTeamDirs     func() ([]AvailableCCTeam, error)     // Testing seam for CC team listing
}

// NewHandlers creates a new teams handler.
func NewHandlers(
	teamStore store.TeamStore,
	agentStore store.AgentStore,
	relationStore store.RelationStore,
	indexStore store.IndexStore,
	heartbeatScheduler HeartbeatScheduler,
) *Handlers {
	return &Handlers{
		teamStore:          teamStore,
		agentStore:         agentStore,
		relationStore:      relationStore,
		indexStore:         indexStore,
		heartbeatScheduler: heartbeatScheduler,
		readCCConfig:       defaultReadCCConfig,
		listCCTeamDirs:     defaultListCCTeamDirs,
	}
}

// HeartbeatScheduler defines the scheduler behavior needed by team handlers.
type HeartbeatScheduler interface {
	Schedule(teamID, agentID, schedule string) error
	Unschedule(teamID, agentID string)
}

// SetHeartbeatScheduler attaches a scheduler after handlers are constructed.
func (h *Handlers) SetHeartbeatScheduler(scheduler HeartbeatScheduler) {
	h.heartbeatScheduler = scheduler
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

	if req.SpawnMode != "" {
		if req.SpawnMode != "multi-process" && req.SpawnMode != "single-process" {
			http.Error(w, "spawnMode must be 'multi-process' or 'single-process'", http.StatusBadRequest)
			return
		}
		team.SpawnMode = req.SpawnMode
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
	if req.Enabled != nil {
		updates.Enabled = *req.Enabled
		updates.EnabledSet = true
	}
	if req.SpawnMode != nil {
		if *req.SpawnMode != "multi-process" && *req.SpawnMode != "single-process" {
			http.Error(w, "spawnMode must be 'multi-process' or 'single-process'", http.StatusBadRequest)
			return
		}
		updates.SpawnMode = *req.SpawnMode
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

	if req.Enabled != nil {
		h.updateHeartbeatSchedules(ctx, id, *req.Enabled)
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

	h.updateHeartbeatSchedules(ctx, id, false)

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

	h.cleanupMemberData(ctx, teamID, agentID)

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

	if _, err := h.teamStore.Get(ctx, id); err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
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

	if err := h.teamStore.SetRoles(ctx, id, roles); err != nil {
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
		resp.Edges = append(resp.Edges, orgEdgeToDTO(edge))
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

	if _, err := h.teamStore.Get(ctx, id); err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}

	if err := h.validateOrgChartEdges(ctx, id, req.Edges); err != nil {
		if isValidationError(err) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	org := &store.OrgChart{
		TeamID: id,
		Edges:  make([]store.OrgEdge, 0, len(req.Edges)),
	}
	for _, e := range req.Edges {
		org.Edges = append(org.Edges, orgEdgeFromDTO(e))
	}

	if err := h.teamStore.SetOrgChart(ctx, id, org); err != nil {
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

// ListSharedFiles handles GET /teams/{id}/shared/files - lists files in team shared folder.
func (h *Handlers) ListSharedFiles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	id := vars["id"]

	fileStore, ok := h.teamStore.(*store.FileTeamStore)
	if !ok {
		http.Error(w, "ListSharedFiles not supported", http.StatusInternalServerError)
		return
	}

	files, err := fileStore.ListSharedFiles(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Team not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	mapped := make([]TeamSharedFileEntry, 0, len(files))
	for _, entry := range files {
		mapped = append(mapped, TeamSharedFileEntry{
			Path:  entry.Path,
			IsDir: entry.IsDir,
			Size:  entry.Size,
		})
	}

	resp := TeamSharedFileListResponse{
		TeamID: id,
		Files:  mapped,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// GetSharedFile handles GET /teams/{id}/shared/files/content - returns file content.
func (h *Handlers) GetSharedFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	id := vars["id"]
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "path is required", http.StatusBadRequest)
		return
	}

	fileStore, ok := h.teamStore.(*store.FileTeamStore)
	if !ok {
		http.Error(w, "GetSharedFile not supported", http.StatusInternalServerError)
		return
	}

	content, err := fileStore.ReadSharedFile(ctx, id, path)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Team not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		if strings.Contains(err.Error(), "invalid path") || strings.Contains(err.Error(), "path is required") {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := TeamSharedFileContentResponse{
		TeamID:  id,
		Path:    path,
		Content: content,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// SetSharedFile handles PUT /teams/{id}/shared/files/content - writes file content.
func (h *Handlers) SetSharedFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	id := vars["id"]
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "path is required", http.StatusBadRequest)
		return
	}

	var req TeamSharedFileWriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fileStore, ok := h.teamStore.(*store.FileTeamStore)
	if !ok {
		http.Error(w, "SetSharedFile not supported", http.StatusInternalServerError)
		return
	}

	if err := fileStore.WriteSharedFile(ctx, id, path, req.Content); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Team not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// CreateSharedFile handles POST /teams/{id}/shared/files - creates a new file or directory.
func (h *Handlers) CreateSharedFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	id := vars["id"]

	var req TeamSharedFileCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Path == "" {
		http.Error(w, "path is required", http.StatusBadRequest)
		return
	}

	fileStore, ok := h.teamStore.(*store.FileTeamStore)
	if !ok {
		http.Error(w, "CreateSharedFile not supported", http.StatusInternalServerError)
		return
	}

	if err := fileStore.CreateSharedFile(ctx, id, req.Path, req.Content, req.IsDir); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Team not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// RenameSharedFile handles POST /teams/{id}/shared/files/rename - renames a file.
func (h *Handlers) RenameSharedFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	id := vars["id"]

	var req TeamSharedFileRenameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.From == "" || req.To == "" {
		http.Error(w, "from and to are required", http.StatusBadRequest)
		return
	}

	fileStore, ok := h.teamStore.(*store.FileTeamStore)
	if !ok {
		http.Error(w, "RenameSharedFile not supported", http.StatusInternalServerError)
		return
	}

	if err := fileStore.RenameSharedFile(ctx, id, req.From, req.To); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Team not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteSharedFile handles DELETE /teams/{id}/shared/files - deletes a file or directory.
func (h *Handlers) DeleteSharedFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	id := vars["id"]
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "path is required", http.StatusBadRequest)
		return
	}

	fileStore, ok := h.teamStore.(*store.FileTeamStore)
	if !ok {
		http.Error(w, "DeleteSharedFile not supported", http.StatusInternalServerError)
		return
	}

	if err := fileStore.DeleteSharedFile(ctx, id, path); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Team not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helper functions

func (h *Handlers) updateHeartbeatSchedules(ctx context.Context, teamID string, enable bool) {
	if h.heartbeatScheduler == nil {
		return
	}

	fileStore, ok := h.teamStore.(*store.FileTeamStore)
	if !ok {
		return
	}

	configs, err := fileStore.ListHeartbeatConfigs(ctx, teamID)
	if err != nil {
		log.Printf("Warning: Failed to list heartbeat configs for team %s: %v", teamID, err)
		return
	}

	for _, config := range configs {
		if enable {
			if !config.Enabled {
				continue
			}
			if err := h.heartbeatScheduler.Schedule(config.TeamID, config.AgentID, config.Schedule); err != nil {
				log.Printf("Warning: Failed to schedule heartbeat for %s/%s: %v", config.TeamID, config.AgentID, err)
			}
			continue
		}
		h.heartbeatScheduler.Unschedule(config.TeamID, config.AgentID)
	}
}

func (h *Handlers) cleanupMemberData(ctx context.Context, teamID, agentID string) {
	if h.heartbeatScheduler != nil {
		h.heartbeatScheduler.Unschedule(teamID, agentID)
	}

	fileStore, ok := h.teamStore.(*store.FileTeamStore)
	if !ok {
		return
	}

	if err := fileStore.DeleteMemberData(ctx, teamID, agentID); err != nil {
		log.Printf("Warning: Failed to delete member data for %s/%s: %v", teamID, agentID, err)
	}
}

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
		Enabled:     t.Enabled,
		SpawnMode:   t.SpawnMode,
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
