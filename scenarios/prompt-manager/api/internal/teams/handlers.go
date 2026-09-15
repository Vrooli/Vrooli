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
	"sort"
	"strings"

	"prompt-manager/internal/store"
	"prompt-manager/internal/teamconfig"
	"prompt-manager/internal/validation"

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

// GraphInvalidator allows triggering graph index invalidation.
type GraphInvalidator interface {
	Invalidate()
}

// AITeamIndexer defines the interface for AI search index operations on teams.
type AITeamIndexer interface {
	IndexTeam(ctx context.Context, teamID string) error
	DeleteTeamFromIndex(ctx context.Context, teamID string) error
}

// Handlers provides HTTP handlers for team operations.
type Handlers struct {
	teamStore          store.TeamStore
	agentStore         store.AgentStore
	relationStore      store.RelationStore
	indexStore         store.IndexStore
	heartbeatScheduler HeartbeatScheduler
	effortSupervisor   EffortSupervisorStarter
	graphInvalidator   GraphInvalidator
	aiIndexer          AITeamIndexer
	readCCConfig       func(teamName string) ([]byte, error) // Testing seam for CC config reader
	listCCTeamDirs     func() ([]AvailableCCTeam, error)     // Testing seam for CC team listing
}

type createValidationError struct{ err error }

func (e *createValidationError) Error() string { return e.err.Error() }
func (e *createValidationError) Unwrap() error { return e.err }

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

// EffortSupervisorStarter arms the one standing supervision service when an
// eligible finite effort team becomes active.
type EffortSupervisorStarter interface {
	EnsureStandingSupervisorStarted(ctx context.Context) error
}

// SetHeartbeatScheduler attaches a scheduler after handlers are constructed.
func (h *Handlers) SetHeartbeatScheduler(scheduler HeartbeatScheduler) {
	h.heartbeatScheduler = scheduler
}

// SetEffortSupervisorStarter attaches control-plane-owned supervisor admission.
func (h *Handlers) SetEffortSupervisorStarter(starter EffortSupervisorStarter) {
	h.effortSupervisor = starter
}

// SetGraphInvalidator sets the graph invalidator.
func (h *Handlers) SetGraphInvalidator(inv GraphInvalidator) {
	h.graphInvalidator = inv
}

// SetAIIndexer sets the AI search indexer for async index updates.
func (h *Handlers) SetAIIndexer(indexer AITeamIndexer) {
	h.aiIndexer = indexer
}

func (h *Handlers) invalidateGraph() {
	if h.graphInvalidator != nil {
		h.graphInvalidator.Invalidate()
	}
}

func (h *Handlers) asyncIndexTeam(ctx context.Context, teamID string) {
	if h.aiIndexer == nil {
		return
	}
	go func() {
		if err := h.aiIndexer.IndexTeam(ctx, teamID); err != nil {
			log.Printf("[teams] AI index update failed for %s: %v", teamID, err)
		}
	}()
}

func (h *Handlers) asyncDeleteTeamFromIndex(ctx context.Context, teamID string) {
	if h.aiIndexer == nil {
		return
	}
	go func() {
		if err := h.aiIndexer.DeleteTeamFromIndex(ctx, teamID); err != nil {
			log.Printf("[teams] AI index delete failed for %s: %v", teamID, err)
		}
	}()
}

// ListTeams returns all teams with their projected summary fields.
func (h *Handlers) ListTeams(ctx context.Context) ([]Response, error) {
	teams, err := h.teamStore.List(ctx)
	if err != nil {
		return nil, err
	}

	responses := make([]Response, 0, len(teams))
	for _, t := range teams {
		resp := h.toResponse(ctx, &t)
		responses = append(responses, resp)
	}
	return responses, nil
}

// GetTeam returns one team with its projected details without crossing the
// HTTP compatibility boundary. REST and Connect adapters share this path.
func (h *Handlers) GetTeam(ctx context.Context, id string) (TeamDetailsResponse, error) {
	team, err := h.teamStore.Get(ctx, id)
	if err != nil {
		return TeamDetailsResponse{}, fmt.Errorf("Team not found")
	}
	return h.toDetailsResponse(ctx, team), nil
}

// CreateTeam validates and persists a team without crossing the HTTP
// compatibility boundary. REST and Connect adapters share this operation.
func (h *Handlers) CreateTeam(ctx context.Context, req CreateRequest) (TeamDetailsResponse, error) {
	if req.DisplayName == "" {
		return TeamDetailsResponse{}, fmt.Errorf("displayName is required")
	}

	id := req.ID
	if id == "" {
		id = validation.Slugify(req.DisplayName)
	}

	team := &store.Team{
		Purpose: req.Purpose, Lifetime: req.Lifetime, EffortRefs: req.EffortRefs,
		ID:                id,
		DisplayName:       req.DisplayName,
		Mission:           req.Mission,
		Runtime:           req.Runtime,
		Coordination:      req.Coordination,
		Execution:         req.Execution,
		OperatingContract: req.OperatingContract,
		Archived:          req.Archived,
	}
	if team.Archived {
		team.Enabled = false
	}
	if err := teamconfig.Validate(team.Contract()); err != nil {
		return TeamDetailsResponse{}, &createValidationError{err: err}
	}
	if err := h.validateEnabledTeamState(ctx, team); err != nil {
		return TeamDetailsResponse{}, &createValidationError{err: err}
	}
	if err := h.teamStore.Create(ctx, team); err != nil {
		return TeamDetailsResponse{}, err
	}

	if h.indexStore != nil {
		_ = h.indexStore.RegenerateTeams(ctx)
	}
	h.invalidateGraph()
	h.asyncIndexTeam(context.Background(), team.ID)
	return h.toDetailsResponse(ctx, team), nil
}

// DeleteTeam removes a team and its member relations without crossing the
// HTTP compatibility boundary. REST and Connect adapters share this path.
func (h *Handlers) DeleteTeam(ctx context.Context, id string) error {
	h.updateHeartbeatSchedules(ctx, id, false)

	if h.relationStore != nil {
		members, _ := h.relationStore.ListTeamMembers(ctx, id)
		for _, member := range members {
			_ = h.relationStore.DeleteTeamMember(ctx, id, member.AgentID)
		}
	}

	if err := h.teamStore.Delete(ctx, id); err != nil {
		return err
	}
	if h.indexStore != nil {
		_ = h.indexStore.RegenerateTeams(ctx)
	}
	h.invalidateGraph()
	h.asyncDeleteTeamFromIndex(context.Background(), id)
	return nil
}

// List handles GET /teams - returns all teams.
func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	responses, err := h.ListTeams(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(responses)
}

// Get handles GET /teams/{id} - returns a single team with details.
func (h *Handlers) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	id := vars["id"]

	team, err := h.GetTeam(ctx, id)
	if err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(team)
}

// Create handles POST /teams - creates a new team.
func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	team, err := h.CreateTeam(ctx, req)
	if err != nil {
		status := http.StatusInternalServerError
		var validationErr *createValidationError
		if err.Error() == "displayName is required" || errors.As(err, &validationErr) {
			status = http.StatusBadRequest
		}
		if strings.Contains(err.Error(), "already exists") {
			status = http.StatusConflict
		}
		http.Error(w, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(team)
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
	if req.Purpose != nil {
		updates.Purpose = *req.Purpose
		updates.PurposeSet = true
	}
	if req.Lifetime != nil {
		updates.Lifetime = *req.Lifetime
		updates.LifetimeSet = true
	}
	if req.EffortRefs != nil {
		updates.EffortRefs = append([]string{}, (*req.EffortRefs)...)
	}

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
	if req.Archived != nil {
		updates.Archived = *req.Archived
		updates.ArchivedSet = true
	}
	if req.Runtime != nil {
		updates.Runtime = *req.Runtime
	}
	if req.Coordination != nil {
		updates.Coordination = *req.Coordination
	}
	if req.Execution != nil {
		updates.Execution = *req.Execution
	}
	if req.OperatingContract != nil {
		updates.OperatingContract = req.OperatingContract
	}

	current, err := h.teamStore.Get(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	previousEnabled := current.Enabled
	merged := *current
	if updates.PurposeSet {
		merged.Purpose = updates.Purpose
	}
	if updates.LifetimeSet {
		merged.Lifetime = updates.Lifetime
	}
	if updates.EffortRefs != nil {
		merged.EffortRefs = updates.EffortRefs
	}

	if updates.DisplayName != "" {
		merged.DisplayName = updates.DisplayName
	}
	if updates.Mission != "" {
		merged.Mission = updates.Mission
	}
	if updates.EnabledSet {
		merged.Enabled = updates.Enabled
	}
	if updates.ArchivedSet {
		merged.Archived = updates.Archived
		if merged.Archived {
			merged.Enabled = false
		}
	}
	if merged.Archived && merged.Enabled {
		http.Error(w, "archived teams cannot be enabled; restore the team first", http.StatusBadRequest)
		return
	}
	if updates.Runtime.Mode != "" {
		merged.Runtime = updates.Runtime
	}
	if updates.Coordination.Pattern != "" {
		merged.Coordination = updates.Coordination
	}
	if updates.Execution.QueuePolicy != "" {
		merged.Execution = updates.Execution
	}
	if updates.OperatingContract != nil {
		merged.OperatingContract = updates.OperatingContract
	}
	if err := teamconfig.Validate(merged.Contract()); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.validateEnabledTeamState(ctx, &merged); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
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
	h.asyncIndexTeam(context.Background(), id)

	if req.Enabled != nil {
		h.updateHeartbeatSchedules(ctx, id, *req.Enabled)
		if *req.Enabled && eligibleFiniteEffortTeam(&merged) && h.effortSupervisor != nil {
			if err := h.effortSupervisor.EnsureStandingSupervisorStarted(ctx); err != nil {
				rollback := &store.Team{Enabled: previousEnabled, EnabledSet: true}
				if rollbackErr := h.teamStore.Update(ctx, id, rollback); rollbackErr != nil {
					log.Printf("automatic standing supervisor admission for %s failed and rollback failed: %v (admission: %v)", id, rollbackErr, err)
				} else {
					h.updateHeartbeatSchedules(ctx, id, previousEnabled)
				}
				http.Error(w, fmt.Sprintf("standing supervisor admission failed: %v", err), http.StatusServiceUnavailable)
				return
			}
		}
	}
	if req.Archived != nil && *req.Archived {
		h.updateHeartbeatSchedules(ctx, id, false)
	}

	// Get updated team
	team, _ := h.teamStore.Get(ctx, id)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(h.toDetailsResponse(ctx, team))
}

func eligibleFiniteEffortTeam(team *store.Team) bool {
	return team != nil && team.Enabled && !team.Archived &&
		team.Purpose == teamconfig.PurposeDelivery &&
		team.Lifetime == teamconfig.LifetimeFinite && len(team.EffortRefs) > 0
}

// Delete handles DELETE /teams/{id} - deletes a team.
func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	id := vars["id"]

	if err := h.DeleteTeam(ctx, id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) AddTeamMember(ctx context.Context, teamID string, req AddMemberRequest) (MemberDTO, error) {
	if req.AgentID == "" {
		return MemberDTO{}, fmt.Errorf("agentId is required")
	}
	if _, err := h.teamStore.Get(ctx, teamID); err != nil {
		return MemberDTO{}, fmt.Errorf("Team not found")
	}
	if _, err := h.agentStore.Get(ctx, req.AgentID); err != nil {
		return MemberDTO{}, fmt.Errorf("Agent not found")
	}
	rel := &store.TeamMemberRelation{TeamID: teamID, AgentID: req.AgentID, Roles: req.Roles, Status: store.MemberStatusActive}
	if err := h.relationStore.SetTeamMember(ctx, rel); err != nil {
		return MemberDTO{}, err
	}
	if h.indexStore != nil {
		_ = h.indexStore.RegenerateTeams(ctx)
	}
	h.asyncIndexTeam(context.Background(), teamID)
	agent, _ := h.agentStore.Get(ctx, req.AgentID)
	return MemberDTO{AgentID: req.AgentID, DisplayName: agent.DisplayName, Roles: req.Roles, Status: store.MemberStatusActive}, nil
}

// AddMember handles POST /teams/{id}/members - adds a member to a team.
func (h *Handlers) AddMember(w http.ResponseWriter, r *http.Request) {
	var req AddMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	member, err := h.AddTeamMember(r.Context(), mux.Vars(r)["id"], req)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "agentId is required" {
			status = http.StatusBadRequest
		}
		if err.Error() == "Team not found" || err.Error() == "Agent not found" {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(member)
}

func (h *Handlers) UpdateTeamMember(ctx context.Context, teamID, agentID string, req UpdateMemberRequest) (MemberDTO, error) {
	membership, err := h.relationStore.GetTeamMember(ctx, teamID, agentID)
	if err != nil {
		return MemberDTO{}, fmt.Errorf("Membership not found")
	}
	team, err := h.teamStore.Get(ctx, teamID)
	if err != nil {
		return MemberDTO{}, fmt.Errorf("Team not found")
	}
	if req.Roles != nil {
		membership.Roles = req.Roles
	}
	if req.Status != nil {
		if h.isProtectedLeadMember(team, agentID) && *req.Status != store.MemberStatusActive {
			return MemberDTO{}, fmt.Errorf("cannot deactivate the configured lead while the leader-led team is enabled; change coordination.leadAgentId or disable the team first")
		}
		membership.Status = *req.Status
	}
	if err := h.relationStore.SetTeamMember(ctx, membership); err != nil {
		return MemberDTO{}, err
	}
	agent, _ := h.agentStore.Get(ctx, agentID)
	return MemberDTO{AgentID: agentID, DisplayName: agent.DisplayName, Roles: membership.Roles, Status: membership.Status}, nil
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

	member, err := h.UpdateTeamMember(ctx, teamID, agentID, req)
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "Membership not found" || err.Error() == "Team not found" {
			status = http.StatusNotFound
		}
		if strings.HasPrefix(err.Error(), "cannot deactivate") {
			status = http.StatusConflict
		}
		http.Error(w, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(member)
}

func (h *Handlers) RemoveTeamMember(ctx context.Context, teamID, agentID string) error {
	team, err := h.teamStore.Get(ctx, teamID)
	if err != nil {
		return fmt.Errorf("Team not found")
	}
	if h.isProtectedLeadMember(team, agentID) {
		return fmt.Errorf("cannot remove the configured lead while the leader-led team is enabled; change coordination.leadAgentId or disable the team first")
	}
	h.cleanupMemberData(ctx, teamID, agentID)
	if err := h.relationStore.DeleteTeamMember(ctx, teamID, agentID); err != nil {
		return err
	}
	if h.indexStore != nil {
		_ = h.indexStore.RegenerateTeams(ctx)
	}
	h.asyncIndexTeam(context.Background(), teamID)
	return nil
}

// RemoveMember handles DELETE /teams/{id}/members/{agentId} - removes a member from a team.
func (h *Handlers) RemoveMember(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	teamID := vars["id"]
	agentID := vars["agentId"]
	if err := h.RemoveTeamMember(ctx, teamID, agentID); err != nil {
		if err.Error() == "Team not found" {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		if strings.HasPrefix(err.Error(), "cannot remove") {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) ListTeamRoles(ctx context.Context, id string) ([]RoleDTO, error) {
	roles, err := h.teamStore.GetRoles(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("Team not found")
	}
	roleDTOs := make([]RoleDTO, 0, len(roles.Roles))
	for _, role := range roles.Roles {
		roleDTOs = append(roleDTOs, RoleDTO{
			ID:          role.ID,
			Name:        role.Name,
			Description: role.Description,
		})
	}
	return roleDTOs, nil
}

// GetRoles handles GET /teams/{id}/roles - returns team roles.
func (h *Handlers) GetRoles(w http.ResponseWriter, r *http.Request) {
	roleDTOs, err := h.ListTeamRoles(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(roleDTOs)
}

func (h *Handlers) SetTeamRoles(ctx context.Context, id string, req SetRolesRequest) ([]RoleDTO, error) {
	if _, err := h.teamStore.Get(ctx, id); err != nil {
		return nil, fmt.Errorf("Team not found")
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
		return nil, err
	}
	return req.Roles, nil
}

// SetRoles handles PUT /teams/{id}/roles - sets team roles.
func (h *Handlers) SetRoles(w http.ResponseWriter, r *http.Request) {
	var req SetRolesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	roles, err := h.SetTeamRoles(r.Context(), mux.Vars(r)["id"], req)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(roles)
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
		TeamID:           id,
		Edges:            make([]OrgEdgeDTO, 0, len(org.Edges)),
		ManagedTeamEdges: make([]ManagedTeamEdgeDTO, 0),
	}
	for _, edge := range org.Edges {
		resp.Edges = append(resp.Edges, orgEdgeToDTO(edge))
	}
	for _, edge := range h.effectiveManagedTeamEdges(ctx, id) {
		resp.ManagedTeamEdges = append(resp.ManagedTeamEdges, managedTeamEdgeToDTO(edge))
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
	if err := h.validateManagedTeamEdges(ctx, id, req.ManagedTeamEdges); err != nil {
		if isValidationError(err) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	org, err := h.teamStore.GetOrgChart(ctx, id)
	if err != nil {
		org = &store.OrgChart{TeamID: id}
	}
	org.TeamID = id
	org.Edges = make([]store.OrgEdge, 0, len(req.Edges))
	for _, e := range req.Edges {
		org.Edges = append(org.Edges, orgEdgeFromDTO(e))
	}
	// Member-level edits use the same endpoint. Preserve team-level edges when
	// older clients omit the new field; an explicit empty array clears them.
	if req.ManagedTeamEdges != nil {
		org.ManagedTeamEdges = make([]store.ManagedTeamEdge, 0, len(req.ManagedTeamEdges))
		for _, edge := range req.ManagedTeamEdges {
			org.ManagedTeamEdges = append(org.ManagedTeamEdges, managedTeamEdgeFromDTO(edge, id))
		}
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
		TeamID:           id,
		Edges:            req.Edges,
		ManagedTeamEdges: req.ManagedTeamEdges,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handlers) ListTeamSharedFiles(ctx context.Context, id string) (TeamSharedFileListResponse, error) {
	fileStore, ok := h.teamStore.(*store.FileTeamStore)
	if !ok {
		return TeamSharedFileListResponse{}, fmt.Errorf("ListSharedFiles not supported")
	}
	files, err := fileStore.ListSharedFiles(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return TeamSharedFileListResponse{}, fmt.Errorf("Team not found")
		}
		return TeamSharedFileListResponse{}, err
	}
	mapped := make([]TeamSharedFileEntry, 0, len(files))
	for _, entry := range files {
		mapped = append(mapped, TeamSharedFileEntry{Path: entry.Path, IsDir: entry.IsDir, Size: entry.Size})
	}
	return TeamSharedFileListResponse{TeamID: id, Files: mapped}, nil
}

func (h *Handlers) ReadTeamSharedFile(ctx context.Context, id, path string) (TeamSharedFileContentResponse, error) {
	if path == "" {
		return TeamSharedFileContentResponse{}, fmt.Errorf("path is required")
	}
	fileStore, ok := h.teamStore.(*store.FileTeamStore)
	if !ok {
		return TeamSharedFileContentResponse{}, fmt.Errorf("GetSharedFile not supported")
	}
	content, err := fileStore.ReadSharedFile(ctx, id, path)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return TeamSharedFileContentResponse{}, fmt.Errorf("Team not found")
		}
		if errors.Is(err, os.ErrNotExist) {
			return TeamSharedFileContentResponse{}, fmt.Errorf("File not found")
		}
		return TeamSharedFileContentResponse{}, err
	}
	return TeamSharedFileContentResponse{TeamID: id, Path: path, Content: content}, nil
}

func (h *Handlers) WriteTeamSharedFile(ctx context.Context, id, path, content string) error {
	if path == "" {
		return fmt.Errorf("path is required")
	}
	fileStore, ok := h.teamStore.(*store.FileTeamStore)
	if !ok {
		return fmt.Errorf("SetSharedFile not supported")
	}
	if err := fileStore.WriteSharedFile(ctx, id, path, content); err != nil {
		return err
	}
	h.invalidateGraph()
	return nil
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
	h.invalidateGraph()

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) CreateTeamSharedFile(ctx context.Context, id string, req TeamSharedFileCreateRequest) error {
	if req.Path == "" {
		return fmt.Errorf("path is required")
	}
	fileStore, ok := h.teamStore.(*store.FileTeamStore)
	if !ok {
		return fmt.Errorf("CreateSharedFile not supported")
	}
	if err := fileStore.CreateSharedFile(ctx, id, req.Path, req.Content, req.IsDir); err != nil {
		return err
	}
	h.invalidateGraph()
	return nil
}

func (h *Handlers) RenameTeamSharedFile(ctx context.Context, id string, req TeamSharedFileRenameRequest) error {
	if req.From == "" || req.To == "" {
		return fmt.Errorf("from and to are required")
	}
	fileStore, ok := h.teamStore.(*store.FileTeamStore)
	if !ok {
		return fmt.Errorf("RenameSharedFile not supported")
	}
	if err := fileStore.RenameSharedFile(ctx, id, req.From, req.To); err != nil {
		return err
	}
	h.invalidateGraph()
	return nil
}

func (h *Handlers) DeleteTeamSharedFile(ctx context.Context, id, path string) error {
	if path == "" {
		return fmt.Errorf("path is required")
	}
	fileStore, ok := h.teamStore.(*store.FileTeamStore)
	if !ok {
		return fmt.Errorf("DeleteSharedFile not supported")
	}
	if err := fileStore.DeleteSharedFile(ctx, id, path); err != nil {
		return err
	}
	h.invalidateGraph()
	return nil
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
	h.invalidateGraph()

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
	h.invalidateGraph()

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
	h.invalidateGraph()

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) ListExclusiveMembers(ctx context.Context, id string) (ExclusiveMembersResponse, error) {
	if _, err := h.teamStore.Get(ctx, id); err != nil {
		return ExclusiveMembersResponse{}, fmt.Errorf("Team not found")
	}
	var members []store.TeamMemberRelation
	if h.relationStore != nil {
		var err error
		members, err = h.relationStore.ListTeamMembers(ctx, id)
		if err != nil {
			return ExclusiveMembersResponse{}, err
		}
	}

	exclusive := make([]ExclusiveMemberDTO, 0)
	for _, m := range members {
		teams, err := h.relationStore.ListAgentTeams(ctx, m.AgentID)
		if err != nil {
			continue
		}
		if len(teams) == 1 {
			displayName := m.AgentID
			if h.agentStore != nil {
				if agent, err := h.agentStore.Get(ctx, m.AgentID); err == nil {
					displayName = agent.DisplayName
				}
			}
			exclusive = append(exclusive, ExclusiveMemberDTO{
				AgentID:     m.AgentID,
				DisplayName: displayName,
			})
		}
	}

	return ExclusiveMembersResponse{TeamID: id, Members: exclusive}, nil
}

// GetExclusiveMembers handles GET /teams/{id}/exclusive-members - returns members only in this team.
func (h *Handlers) GetExclusiveMembers(w http.ResponseWriter, r *http.Request) {
	resp, err := h.ListExclusiveMembers(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "Team not found" {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
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

func (h *Handlers) validateEnabledTeamState(ctx context.Context, team *store.Team) error {
	if team == nil || !team.Enabled || team.Coordination.Pattern != teamconfig.CoordinationPatternLeaderLed {
		return nil
	}
	return h.validateActiveLeadMember(ctx, team)
}

func (h *Handlers) validateActiveLeadMember(ctx context.Context, team *store.Team) error {
	if h.relationStore == nil {
		return errors.New("relation store not configured")
	}

	leadAgentID := strings.TrimSpace(team.Coordination.LeadAgentID)
	if leadAgentID == "" {
		return fmt.Errorf("coordination.leadAgentId is required for leader-led teams")
	}

	membership, err := h.relationStore.GetTeamMember(ctx, team.ID, leadAgentID)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || strings.Contains(strings.ToLower(err.Error()), "not found") {
			return fmt.Errorf("coordination.leadAgentId %q must reference an active team member", leadAgentID)
		}
		return fmt.Errorf("validating coordination.leadAgentId %q: %w", leadAgentID, err)
	}
	if membership.Status != store.MemberStatusActive {
		return fmt.Errorf("coordination.leadAgentId %q must reference an active team member", leadAgentID)
	}
	return nil
}

func (h *Handlers) isProtectedLeadMember(team *store.Team, agentID string) bool {
	return team != nil &&
		team.Enabled &&
		team.Coordination.Pattern == teamconfig.CoordinationPatternLeaderLed &&
		team.Coordination.LeadAgentID == agentID
}

func (h *Handlers) toResponse(ctx context.Context, t *store.Team) Response {
	memberCount := 0
	if h.relationStore != nil {
		members, err := h.relationStore.ListTeamMembers(ctx, t.ID)
		if err == nil {
			memberCount = len(members)
		}
	}

	managed, managedBy := h.teamRelationshipIDs(ctx, t.ID)
	return Response{
		Purpose: t.Purpose, Lifetime: t.Lifetime, EffortRefs: t.EffortRefs, ObjectivesServed: t.ObjectivesServed,
		ID:                 t.ID,
		DisplayName:        t.DisplayName,
		Mission:            t.Mission,
		Enabled:            t.Enabled,
		Archived:           t.Archived,
		Runtime:            t.Runtime,
		Coordination:       t.Coordination,
		Execution:          t.Execution,
		OperatingContract:  t.OperatingContract,
		ValidationFindings: t.ValidationFindings,
		MemberCount:        memberCount,
		CreatedAt:          t.CreatedAt,
		UpdatedAt:          t.UpdatedAt,
		ManagedTeamIDs:     managed,
		ManagedByTeamIDs:   managedBy,
	}
}

func (h *Handlers) teamRelationshipIDs(ctx context.Context, teamID string) ([]string, []string) {
	managed := []string{}
	managedBy := []string{}
	teams, err := h.teamStore.List(ctx)
	if err != nil {
		return managed, managedBy
	}
	archived := make(map[string]bool, len(teams))
	for _, team := range teams {
		archived[team.ID] = team.Archived
	}
	for _, team := range teams {
		if archived[team.ID] {
			continue
		}
		for _, edge := range h.effectiveManagedTeamEdges(ctx, team.ID) {
			if edge.ManagerTeamID == teamID && edge.Status != "archived" && !archived[edge.ManagedTeamID] {
				managed = append(managed, edge.ManagedTeamID)
			}
			if edge.ManagedTeamID == teamID && edge.Status != "archived" && !archived[edge.ManagerTeamID] {
				managedBy = append(managedBy, edge.ManagerTeamID)
			}
		}
	}
	sort.Strings(managed)
	sort.Strings(managedBy)
	return managed, managedBy
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
