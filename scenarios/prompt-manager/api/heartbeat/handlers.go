package heartbeat

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
	"strconv"
	"strings"
	"time"

	"prompt-manager/store"
	"prompt-manager/teamconfig"

	"github.com/gorilla/mux"
)

// Handlers provides HTTP handlers for heartbeat operations
type Handlers struct {
	teamStore     *store.FileTeamStore
	agentStore    *store.FileAgentStore
	relationStore store.RelationStore
	scheduler     *Scheduler
	executor      *Executor
	runRegistry   *RunRegistry
	agentClient   AgentClient
	teamExecStore *TeamExecutionStore
	controlStore  *HeartbeatControlStore
	// swarmClient is the HTTP client used to invoke swarm-manager when
	// auto-creating an initiative on decision-accept. Lazily initialised the
	// first time it is used; tests inject a stub via SetSwarmInitiativeClient.
	swarmClient *SwarmInitiativeClient
}

// SetControlStore attaches the heartbeat engagement guard.
func (h *Handlers) SetControlStore(controlStore *HeartbeatControlStore) {
	h.controlStore = controlStore
}

// SetSwarmInitiativeClient overrides the swarm-manager initiative client.
// Intended for tests; production code uses the lazily-initialised default.
func (h *Handlers) SetSwarmInitiativeClient(c *SwarmInitiativeClient) {
	h.swarmClient = c
}

// NewHandlers creates new heartbeat handlers
// HandlersDeps carries what the heartbeat handlers need.
//
// This replaced eight positional parameters, of which most call sites passed
// nil for most. `NewHandlers(nil, nil, nil, nil, nil, nil, mockClient, nil)`
// gave a reader no way to know which nil mattered, and adding a dependency
// meant editing every call site to add one more nil. A named field says which
// collaborator a test actually needs; every other field being absent is then
// information rather than noise.
type HandlersDeps struct {
	TeamStore     *store.FileTeamStore
	AgentStore    *store.FileAgentStore
	RelationStore store.RelationStore
	Scheduler     *Scheduler
	Executor      *Executor
	RunRegistry   *RunRegistry
	AgentClient   AgentClient
	TeamExecStore *TeamExecutionStore
}

func NewHandlers(deps HandlersDeps) *Handlers {
	return &Handlers{
		teamStore:     deps.TeamStore,
		agentStore:    deps.AgentStore,
		relationStore: deps.RelationStore,
		scheduler:     deps.Scheduler,
		executor:      deps.Executor,
		runRegistry:   deps.RunRegistry,
		agentClient:   deps.AgentClient,
		teamExecStore: deps.TeamExecStore,
	}
}

var errMemberNotFound = errors.New("team member not found")

func (h *Handlers) requireMember(ctx context.Context, teamID, agentID string) error {
	if h.relationStore == nil {
		return errors.New("relation store not configured")
	}
	if agentID == "" {
		return errors.New("agentId is required")
	}
	if _, err := h.relationStore.GetTeamMember(ctx, teamID, agentID); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return errMemberNotFound
		}
		return err
	}
	return nil
}

func (h *Handlers) requireActiveLeadHeartbeatConfig(ctx context.Context, team *store.Team) (*store.HeartbeatConfig, error) {
	if team == nil {
		return nil, fmt.Errorf("team not found")
	}
	if h.relationStore == nil {
		return nil, fmt.Errorf("relation store not configured")
	}

	leadAgentID := strings.TrimSpace(team.Coordination.LeadAgentID)
	if leadAgentID == "" {
		return nil, fmt.Errorf("coordination.leadAgentId is required for leader-led single-process teams")
	}

	membership, err := h.relationStore.GetTeamMember(ctx, team.ID, leadAgentID)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || strings.Contains(strings.ToLower(err.Error()), "not found") {
			return nil, fmt.Errorf("coordination.leadAgentId %q must reference an active team member", leadAgentID)
		}
		return nil, fmt.Errorf("validating coordination.leadAgentId %q: %w", leadAgentID, err)
	}
	if membership.Status != store.MemberStatusActive {
		return nil, fmt.Errorf("coordination.leadAgentId %q must reference an active team member", leadAgentID)
	}

	config, err := h.teamStore.GetHeartbeatConfig(ctx, team.ID, leadAgentID)
	if err != nil {
		return nil, fmt.Errorf("validating lead heartbeat config: %w", err)
	}
	if config == nil {
		return nil, fmt.Errorf("leader-led single-process teams require a heartbeat config for lead agent %q", leadAgentID)
	}

	return config, nil
}

// ListHeartbeats handles GET /teams/{id}/heartbeats - lists all heartbeat configs for a team
func (h *Handlers) ListHeartbeats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	teamID := vars["id"]

	configs, err := h.teamStore.ListHeartbeatConfigs(ctx, teamID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Team not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	responses := make([]HeartbeatConfigResponse, 0, len(configs))
	for _, config := range configs {
		responses = append(responses, h.toResponse(&config))
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(responses)
}

// PreviewPrompt handles POST /prompt-preview - builds a prompt preview for an agent (optionally within a team).
func (h *Handlers) PreviewPrompt(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req PromptPreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.AgentID) == "" {
		http.Error(w, "agentId is required", http.StatusBadRequest)
		return
	}

	if h.executor == nil {
		http.Error(w, "Prompt preview not available", http.StatusInternalServerError)
		return
	}

	prompt, err := h.executor.BuildPrompt(ctx, req.TeamID, req.AgentID)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || strings.Contains(err.Error(), "not found") {
			http.Error(w, "Agent or team not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(PromptPreviewResponse{
		AgentID: req.AgentID,
		TeamID:  req.TeamID,
		Prompt:  prompt,
	})
}

// PreviewPromptStructured handles POST /prompt-preview-structured - returns the prompt as structured sections.
func (h *Handlers) PreviewPromptStructured(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req PromptPreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.AgentID) == "" {
		http.Error(w, "agentId is required", http.StatusBadRequest)
		return
	}

	if h.executor == nil {
		http.Error(w, "Prompt preview not available", http.StatusInternalServerError)
		return
	}

	sections, err := h.executor.BuildPromptStructured(ctx, req.TeamID, req.AgentID)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || strings.Contains(err.Error(), "not found") {
			http.Error(w, "Agent or team not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(StructuredPromptPreviewResponse{
		AgentID:  req.AgentID,
		TeamID:   req.TeamID,
		Sections: sections,
	})
}

// PreviewPromptMatrix handles GET /teams/{id}/prompt-matrix - returns structured prompts for every member.
func (h *Handlers) PreviewPromptMatrix(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamID := mux.Vars(r)["id"]

	// Validate team exists
	if _, err := h.teamStore.Get(ctx, teamID); err != nil {
		http.Error(w, "team not found", http.StatusNotFound)
		return
	}

	// List members
	members, err := h.relationStore.ListTeamMembers(ctx, teamID)
	if err != nil {
		http.Error(w, "failed to list team members", http.StatusInternalServerError)
		return
	}

	entries := make([]TeamPromptMatrixEntry, 0, len(members))
	for _, m := range members {
		entry := TeamPromptMatrixEntry{AgentID: m.AgentID}

		// Resolve display name
		if ag, err := h.agentStore.Get(ctx, m.AgentID); err == nil && ag.DisplayName != "" {
			entry.DisplayName = ag.DisplayName
		} else {
			entry.DisplayName = m.AgentID
		}

		// Build structured prompt
		if h.executor == nil {
			entry.Error = "executor not configured"
		} else {
			sections, err := h.executor.BuildPromptStructured(ctx, teamID, m.AgentID)
			if err != nil {
				entry.Error = err.Error()
			} else {
				entry.Sections = sections
			}
		}

		entries = append(entries, entry)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(TeamPromptMatrixResponse{
		TeamID:  teamID,
		Entries: entries,
	})
}

// GetHeartbeat handles GET /teams/{id}/heartbeats/{agentId} - gets heartbeat config
func (h *Handlers) GetHeartbeat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	teamID := vars["id"]
	agentID := vars["agentId"]

	if _, err := h.teamStore.Get(ctx, teamID); err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}
	if err := h.requireMember(ctx, teamID, agentID); err != nil {
		if errors.Is(err, errMemberNotFound) {
			http.Error(w, "Team member not found", http.StatusNotFound)
			return
		}
		if strings.Contains(err.Error(), "required") {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	config, err := h.teamStore.GetHeartbeatConfig(ctx, teamID, agentID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Team not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if config == nil {
		http.Error(w, "Heartbeat config not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(h.toResponse(config))
}

// CreateHeartbeat handles POST /teams/{id}/heartbeats/{agentId} - creates heartbeat config
func (h *Handlers) CreateHeartbeat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	teamID := vars["id"]
	agentID := vars["agentId"]

	var req CreateHeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Schedule == "" {
		http.Error(w, "schedule is required", http.StatusBadRequest)
		return
	}

	team, err := h.teamStore.Get(ctx, teamID)
	if err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}
	if err := h.requireMember(ctx, teamID, agentID); err != nil {
		if errors.Is(err, errMemberNotFound) {
			http.Error(w, "Team member not found", http.StatusNotFound)
			return
		}
		if strings.Contains(err.Error(), "required") {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Check if config already exists
	existing, _ := h.teamStore.GetHeartbeatConfig(ctx, teamID, agentID)
	if existing != nil {
		http.Error(w, "Heartbeat config already exists", http.StatusConflict)
		return
	}

	// Create config
	config := &store.HeartbeatConfig{
		TeamID:     teamID,
		AgentID:    agentID,
		Schedule:   req.Schedule,
		ProfileKey: req.ProfileKey,
		Enabled:    false, // Off by default
	}

	if req.Enabled != nil {
		config.Enabled = *req.Enabled
	}

	if req.TimeoutSeconds > 0 {
		v := req.TimeoutSeconds
		if v < store.MinHeartbeatTimeoutSeconds {
			v = store.MinHeartbeatTimeoutSeconds
		}
		if v > store.MaxHeartbeatTimeoutSeconds {
			v = store.MaxHeartbeatTimeoutSeconds
		}
		config.TimeoutSeconds = v
	}
	engagementEvent, err := h.operatorEngagementEventFromRequest(r, teamID, "heartbeat-config-created")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.teamStore.SetHeartbeatConfig(ctx, teamID, agentID, config); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Team not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := h.recordOperatorEngagementEvent(ctx, engagementEvent); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Schedule if enabled
	if config.Enabled && team.Enabled && h.scheduler != nil {
		if err := h.scheduler.Schedule(teamID, agentID, config.Schedule); err != nil {
			// Log error but don't fail - config is saved
			config.Enabled = false
			_ = h.teamStore.SetHeartbeatConfig(ctx, teamID, agentID, config)
			http.Error(w, "Invalid cron schedule: "+err.Error(), http.StatusBadRequest)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(h.toResponse(config))
}

// UpdateHeartbeat handles PUT /teams/{id}/heartbeats/{agentId} - updates heartbeat config
func (h *Handlers) UpdateHeartbeat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	teamID := vars["id"]
	agentID := vars["agentId"]

	team, err := h.teamStore.Get(ctx, teamID)
	if err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}
	if err := h.requireMember(ctx, teamID, agentID); err != nil {
		if errors.Is(err, errMemberNotFound) {
			http.Error(w, "Team member not found", http.StatusNotFound)
			return
		}
		if strings.Contains(err.Error(), "required") {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var req UpdateHeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	config, err := h.teamStore.GetHeartbeatConfig(ctx, teamID, agentID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Team not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if config == nil {
		http.Error(w, "Heartbeat config not found", http.StatusNotFound)
		return
	}

	// Apply updates
	if req.Schedule != nil {
		config.Schedule = *req.Schedule
	}
	if req.ProfileKey != nil {
		config.ProfileKey = *req.ProfileKey
	}
	if req.TimeoutSeconds != nil {
		v := *req.TimeoutSeconds
		if v < store.MinHeartbeatTimeoutSeconds {
			v = store.MinHeartbeatTimeoutSeconds
		}
		if v > store.MaxHeartbeatTimeoutSeconds {
			v = store.MaxHeartbeatTimeoutSeconds
		}
		config.TimeoutSeconds = v
	}
	wasEnabled := config.Enabled
	if req.Enabled != nil {
		config.Enabled = *req.Enabled
	}

	engagementEvent, err := h.operatorEngagementEventFromRequest(r, teamID, "heartbeat-config-updated")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.teamStore.SetHeartbeatConfig(ctx, teamID, agentID, config); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := h.recordOperatorEngagementEvent(ctx, engagementEvent); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Handle schedule changes
	if h.scheduler != nil {
		if config.Enabled && team.Enabled && (!wasEnabled || req.Schedule != nil) {
			// Need to (re)schedule
			if err := h.scheduler.Schedule(teamID, agentID, config.Schedule); err != nil {
				config.Enabled = false
				_ = h.teamStore.SetHeartbeatConfig(ctx, teamID, agentID, config)
				http.Error(w, "Invalid cron schedule: "+err.Error(), http.StatusBadRequest)
				return
			}
		} else if (!config.Enabled && wasEnabled) || (!team.Enabled && wasEnabled) {
			// Need to unschedule
			h.scheduler.Unschedule(teamID, agentID)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(h.toResponse(config))
}

// DeleteHeartbeat handles DELETE /teams/{id}/heartbeats/{agentId} - deletes heartbeat config
func (h *Handlers) DeleteHeartbeat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	teamID := vars["id"]
	agentID := vars["agentId"]

	if _, err := h.teamStore.Get(ctx, teamID); err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}
	if err := h.requireMember(ctx, teamID, agentID); err != nil {
		if errors.Is(err, errMemberNotFound) {
			http.Error(w, "Team member not found", http.StatusNotFound)
			return
		}
		if strings.Contains(err.Error(), "required") {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Unschedule first
	if h.scheduler != nil {
		h.scheduler.Unschedule(teamID, agentID)
	}

	if err := h.teamStore.DeleteHeartbeatConfig(ctx, teamID, agentID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// TriggerHeartbeat handles POST /teams/{id}/heartbeats/{agentId}/trigger - manual trigger
func (h *Handlers) TriggerHeartbeat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	teamID := vars["id"]
	agentID := vars["agentId"]

	if h.controlStore != nil {
		if paused, err := h.controlStore.AllowStart(ctx, teamID); err != nil {
			if errors.Is(err, ErrHeartbeatPaused) {
				writeHeartbeatPaused(w, paused)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	engagementEvent, err := h.operatorEngagementEventFromRequest(r, teamID, "heartbeat-manual-trigger")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, status, err := h.triggerHeartbeatMember(ctx, teamID, agentID)
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}
	if err := h.recordOperatorEngagementEvent(ctx, engagementEvent); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handlers) triggerHeartbeatMember(ctx context.Context, teamID, agentID string) (*TriggerHeartbeatResponse, int, error) {
	if h.executor == nil {
		return nil, http.StatusServiceUnavailable, errors.New("Executor not configured")
	}

	team, err := h.teamStore.Get(ctx, teamID)
	if err != nil {
		return nil, http.StatusNotFound, errors.New("Team not found")
	}
	if err := h.requireMember(ctx, teamID, agentID); err != nil {
		if errors.Is(err, errMemberNotFound) {
			return nil, http.StatusNotFound, errors.New("Team member not found")
		}
		if strings.Contains(err.Error(), "required") {
			return nil, http.StatusBadRequest, err
		}
		return nil, http.StatusInternalServerError, err
	}

	// Route through the team execution store so the configured queue policy is enforced.
	if h.teamExecStore != nil {
		// Resolve profile key from heartbeat config; fall back to the
		// runtime-mode default when not explicitly set.
		profileKey := DefaultProfileKeyForRuntimeMode(team.Runtime.Mode)
		config, err := h.teamStore.GetHeartbeatConfig(ctx, teamID, agentID)
		if err == nil && config != nil && config.ProfileKey != "" {
			profileKey = config.ProfileKey
		}

		enqueueResult, err := h.teamExecStore.Enqueue(ctx, teamID, agentID, profileKey)
		if err != nil {
			if IsMemberAlreadyQueued(err) {
				return nil, http.StatusConflict, err
			}
			if IsTeamDisabled(err) {
				return nil, http.StatusConflict, errors.New("Team is disabled; enable the team to run heartbeats")
			}
			return nil, http.StatusInternalServerError, err
		}

		return &TriggerHeartbeatResponse{
			TeamID:   enqueueResult.TeamID,
			AgentID:  enqueueResult.AgentID,
			Status:   enqueueResult.Status,
			Position: enqueueResult.Position,
		}, http.StatusAccepted, nil
	}

	// Fallback: direct execution
	result, err := h.executor.TriggerManual(ctx, teamID, agentID)
	if err != nil {
		if IsTeamDisabled(err) {
			return nil, http.StatusConflict, errors.New("Team is disabled; enable the team to run heartbeats")
		}
		if strings.Contains(err.Error(), "not found") {
			return nil, http.StatusNotFound, err
		}
		return nil, http.StatusInternalServerError, err
	}

	return &TriggerHeartbeatResponse{
		TeamID:  result.TeamID,
		AgentID: result.AgentID,
		RunID:   result.RunID,
		Status:  result.Status,
		LogPath: result.LogPath,
	}, http.StatusAccepted, nil
}

func (h *Handlers) resolveHeartbeatTargetFromRunTag(ctx context.Context, tag string) (string, string, error) {
	const heartbeatTagPrefix = "heartbeat-"
	const heartbeatTagTimestampLayout = "2006-01-02T15-04-05Z"
	const heartbeatTagTimestampLen = len("2006-01-02T15-04-05Z")

	if !strings.HasPrefix(tag, heartbeatTagPrefix) {
		return "", "", fmt.Errorf("run is not a heartbeat run")
	}
	rest := strings.TrimPrefix(tag, heartbeatTagPrefix)
	if len(rest) <= heartbeatTagTimestampLen+1 {
		return "", "", fmt.Errorf("heartbeat run tag is malformed")
	}
	sepIdx := len(rest) - (heartbeatTagTimestampLen + 1)
	if sepIdx < 1 || rest[sepIdx] != '-' {
		return "", "", fmt.Errorf("heartbeat run tag is malformed")
	}
	if _, err := time.Parse(heartbeatTagTimestampLayout, rest[sepIdx+1:]); err != nil {
		return "", "", fmt.Errorf("heartbeat run tag has invalid timestamp")
	}

	teamAndAgent := rest[:sepIdx]
	teams, err := h.teamStore.List(ctx)
	if err != nil {
		return "", "", fmt.Errorf("list teams: %w", err)
	}
	sort.Slice(teams, func(i, j int) bool {
		return len(teams[i].ID) > len(teams[j].ID)
	})

	for _, team := range teams {
		prefix := team.ID + "-"
		if !strings.HasPrefix(teamAndAgent, prefix) {
			continue
		}
		agentID := strings.TrimPrefix(teamAndAgent, prefix)
		if agentID == "" {
			continue
		}
		if err := h.requireMember(ctx, team.ID, agentID); err == nil {
			return team.ID, agentID, nil
		}
	}

	return "", "", fmt.Errorf("could not resolve heartbeat team/member from run tag")
}

// RetryRun handles POST /runs/{runId}/retry - retries a heartbeat run by re-triggering its team/member heartbeat.
func (h *Handlers) RetryRun(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	runID := vars["runId"]
	if runID == "" {
		http.Error(w, "runId is required", http.StatusBadRequest)
		return
	}
	if h.agentClient == nil {
		http.Error(w, "agent client not configured", http.StatusServiceUnavailable)
		return
	}

	run, err := h.agentClient.GetRun(r.Context(), runID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	if run == nil {
		http.Error(w, "run not found", http.StatusNotFound)
		return
	}
	if strings.TrimSpace(run.Tag) == "" {
		http.Error(w, "run has no tag; cannot map to heartbeat", http.StatusBadRequest)
		return
	}

	teamID, agentID, err := h.resolveHeartbeatTargetFromRunTag(r.Context(), run.Tag)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, status, err := h.triggerHeartbeatMember(r.Context(), teamID, agentID)
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(resp)
}

// ListTeamLogs handles GET /teams/{id}/heartbeats/logs - lists execution logs across all team members
func (h *Handlers) ListTeamLogs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	teamID := vars["id"]

	if _, err := h.teamStore.Get(ctx, teamID); err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}

	// Parse pagination params
	limit := 25
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}
	agentIDFilter := r.URL.Query().Get("agentId")

	// Get team members
	members, err := h.teamStore.GetMembers(ctx, teamID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get team members: %v", err), http.StatusInternalServerError)
		return
	}

	// Build agent display name lookup
	agentNames := make(map[string]string, len(members))
	for _, m := range members {
		name := m.AgentID
		if agent, err := h.agentStore.Get(ctx, m.AgentID); err == nil {
			name = agent.DisplayName
		}
		agentNames[m.AgentID] = name
	}

	// Collect logs from each member (or just the filtered one)
	var allEntries []TeamLogEntry
	for _, m := range members {
		if agentIDFilter != "" && m.AgentID != agentIDFilter {
			continue
		}
		logs, err := h.teamStore.ListMemberLogs(ctx, teamID, m.AgentID)
		if err != nil {
			continue
		}
		for _, logFile := range logs {
			timestamp := strings.TrimSuffix(logFile, ".log")
			allEntries = append(allEntries, TeamLogEntry{
				AgentID:          m.AgentID,
				AgentDisplayName: agentNames[m.AgentID],
				Filename:         logFile,
				Timestamp:        timestamp,
			})
		}
	}

	// Sort by timestamp descending
	sort.Slice(allEntries, func(i, j int) bool {
		return allEntries[i].Timestamp > allEntries[j].Timestamp
	})

	total := len(allEntries)

	// Apply pagination
	if offset > len(allEntries) {
		allEntries = nil
	} else {
		allEntries = allEntries[offset:]
	}
	if len(allEntries) > limit {
		allEntries = allEntries[:limit]
	}

	hasMore := offset+len(allEntries) < total

	resp := TeamLogListResponse{
		TeamID:  teamID,
		Logs:    allEntries,
		Total:   total,
		HasMore: hasMore,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// ListLogs handles GET /teams/{id}/heartbeats/{agentId}/logs - lists execution logs
func (h *Handlers) ListLogs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	teamID := vars["id"]
	agentID := vars["agentId"]

	if _, err := h.teamStore.Get(ctx, teamID); err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}
	if err := h.requireMember(ctx, teamID, agentID); err != nil {
		if errors.Is(err, errMemberNotFound) {
			http.Error(w, "Team member not found", http.StatusNotFound)
			return
		}
		if strings.Contains(err.Error(), "required") {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logs, err := h.teamStore.ListMemberLogs(ctx, teamID, agentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	entries := make([]LogEntry, 0, len(logs))
	for _, log := range logs {
		// Extract timestamp from filename (format: 2006-01-02T15-04-05Z.log)
		timestamp := strings.TrimSuffix(log, ".log")
		entries = append(entries, LogEntry{
			Filename:  log,
			Timestamp: timestamp,
		})
	}

	resp := LogListResponse{
		TeamID:  teamID,
		AgentID: agentID,
		Logs:    entries,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// GetLog handles GET /teams/{id}/heartbeats/{agentId}/logs/{logId} - gets log content
func (h *Handlers) GetLog(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	teamID := vars["id"]
	agentID := vars["agentId"]
	logID := vars["logId"]

	if _, err := h.teamStore.Get(ctx, teamID); err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}
	if err := h.requireMember(ctx, teamID, agentID); err != nil {
		if errors.Is(err, errMemberNotFound) {
			http.Error(w, "Team member not found", http.StatusNotFound)
			return
		}
		if strings.Contains(err.Error(), "required") {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Ensure logID has .log extension
	if !strings.HasSuffix(logID, ".log") {
		logID += ".log"
	}

	logPath := h.teamStore.GetMemberLogPathForContext(ctx, teamID, agentID, strings.TrimSuffix(logID, ".log"))
	content, err := store.ReadContent(logPath)
	if err != nil {
		http.Error(w, "Log not found", http.StatusNotFound)
		return
	}

	resp := LogContentResponse{
		TeamID:   teamID,
		AgentID:  agentID,
		Filename: logID,
		Content:  content,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// GetResponsibilities handles GET /teams/{id}/members/{agentId}/responsibilities
func (h *Handlers) GetResponsibilities(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	teamID := vars["id"]
	agentID := vars["agentId"]

	if _, err := h.teamStore.Get(ctx, teamID); err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}
	if err := h.requireMember(ctx, teamID, agentID); err != nil {
		if errors.Is(err, errMemberNotFound) {
			http.Error(w, "Team member not found", http.StatusNotFound)
			return
		}
		if strings.Contains(err.Error(), "required") {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	content, err := h.teamStore.GetResponsibilities(ctx, teamID, agentID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Team not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := MemberDocResponse{
		TeamID:  teamID,
		AgentID: agentID,
		Content: content,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// SetResponsibilities handles PUT /teams/{id}/members/{agentId}/responsibilities
func (h *Handlers) SetResponsibilities(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	teamID := vars["id"]
	agentID := vars["agentId"]

	if _, err := h.teamStore.Get(ctx, teamID); err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}
	if err := h.requireMember(ctx, teamID, agentID); err != nil {
		if errors.Is(err, errMemberNotFound) {
			http.Error(w, "Team member not found", http.StatusNotFound)
			return
		}
		if strings.Contains(err.Error(), "required") {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var req MemberDocRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.teamStore.SetResponsibilities(ctx, teamID, agentID, req.Content); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Team not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := MemberDocResponse{
		TeamID:  teamID,
		AgentID: agentID,
		Content: req.Content,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// GetHeartbeatInstructions handles GET /teams/{id}/members/{agentId}/heartbeat-instructions
func (h *Handlers) GetHeartbeatInstructions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	teamID := vars["id"]
	agentID := vars["agentId"]

	if _, err := h.teamStore.Get(ctx, teamID); err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}
	if err := h.requireMember(ctx, teamID, agentID); err != nil {
		if errors.Is(err, errMemberNotFound) {
			http.Error(w, "Team member not found", http.StatusNotFound)
			return
		}
		if strings.Contains(err.Error(), "required") {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	content, err := h.teamStore.GetHeartbeatInstructions(ctx, teamID, agentID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Team not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := MemberDocResponse{
		TeamID:  teamID,
		AgentID: agentID,
		Content: content,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// SetHeartbeatInstructions handles PUT /teams/{id}/members/{agentId}/heartbeat-instructions
func (h *Handlers) SetHeartbeatInstructions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	teamID := vars["id"]
	agentID := vars["agentId"]

	if _, err := h.teamStore.Get(ctx, teamID); err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}
	if err := h.requireMember(ctx, teamID, agentID); err != nil {
		if errors.Is(err, errMemberNotFound) {
			http.Error(w, "Team member not found", http.StatusNotFound)
			return
		}
		if strings.Contains(err.Error(), "required") {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var req MemberDocRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.teamStore.SetHeartbeatInstructions(ctx, teamID, agentID, req.Content); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Team not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := MemberDocResponse{
		TeamID:  teamID,
		AgentID: agentID,
		Content: req.Content,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// ListRunning handles GET /heartbeats/running - lists all currently running agents
func (h *Handlers) ListRunning(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.runRegistry == nil {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(RunningAgentsResponse{Count: 0, Agents: []RunningAgentEntry{}})
		return
	}

	active := h.runRegistry.ListActive()
	entries := make([]RunningAgentEntry, 0, len(active))
	now := time.Now().UTC()

	for _, run := range active {
		entry := RunningAgentEntry{
			TeamID:    run.TeamID,
			AgentID:   run.AgentID,
			RunID:     run.RunID,
			StartedAt: run.StartedAt.Format(time.RFC3339),
			Duration:  formatDuration(now.Sub(run.StartedAt)),
		}

		// Look up display names (best-effort)
		if h.teamStore != nil {
			if team, err := h.teamStore.Get(ctx, run.TeamID); err == nil && team != nil {
				entry.TeamName = team.DisplayName
			}
		}
		if h.agentStore != nil {
			if agent, err := h.agentStore.Get(ctx, run.AgentID); err == nil && agent != nil {
				entry.AgentName = agent.DisplayName
			}
		}

		entries = append(entries, entry)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(RunningAgentsResponse{
		Count:  len(entries),
		Agents: entries,
	})
}

// StopRunning handles POST /heartbeats/running/{teamId}/{agentId}/stop
func (h *Handlers) StopRunning(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	teamID := vars["teamId"]
	agentID := vars["agentId"]

	if h.runRegistry == nil {
		http.Error(w, "Run registry not available", http.StatusServiceUnavailable)
		return
	}

	run, ok := h.runRegistry.GetActiveRun(teamID, agentID)
	if !ok {
		http.Error(w, "No active run found for this agent", http.StatusNotFound)
		return
	}

	// Request agent-manager to stop the run (best-effort)
	if h.agentClient != nil {
		if err := h.agentClient.StopRun(ctx, run.RunID); err != nil {
			log.Printf("heartbeat: best-effort StopRun(%s) failed: %v", run.RunID, err)
		}
	}

	// Cancel the local waitForCompletion goroutine (nil for recovered runs)
	if run.CancelFn != nil {
		run.CancelFn()
	}

	// Remove from registry
	h.runRegistry.Unregister(teamID, agentID)

	// Update heartbeat config status to cancelled
	config, err := h.teamStore.GetHeartbeatConfig(ctx, teamID, agentID)
	if err == nil && config != nil && config.LastExecution != nil && config.LastExecution.Status == store.HeartbeatStatusRunning {
		config.LastExecution.Status = store.HeartbeatStatusCancelled
		config.LastExecution.EndedAt = time.Now().UTC().Format(time.RFC3339)
		_ = h.teamStore.SetHeartbeatConfig(ctx, teamID, agentID, config)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(StopAgentResponse{
		TeamID:  teamID,
		AgentID: agentID,
		RunID:   run.RunID,
		Status:  "stopped",
	})
}

// GetHeartbeatControl returns global heartbeat pause/policy status.
func (h *Handlers) GetHeartbeatControl(w http.ResponseWriter, r *http.Request) {
	if h.controlStore == nil {
		http.Error(w, "heartbeat control store not configured", http.StatusServiceUnavailable)
		return
	}
	teams, err := h.teamStore.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp, err := h.controlStore.Status(r.Context(), teams)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// UpdateHeartbeatControlPolicy updates the global auto-pause policy.
func (h *Handlers) UpdateHeartbeatControlPolicy(w http.ResponseWriter, r *http.Request) {
	if h.controlStore == nil {
		http.Error(w, "heartbeat control store not configured", http.StatusServiceUnavailable)
		return
	}
	attr, isHuman, err := attributionFromRequest(r, "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !isHuman {
		http.Error(w, "operator-direct attribution is required", http.StatusForbidden)
		return
	}
	var req HeartbeatControlPolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	resp, err := h.controlStore.UpdateGlobalPolicy(r.Context(), req, attr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// PauseHeartbeatControl manually pauses global heartbeat starts.
func (h *Handlers) PauseHeartbeatControl(w http.ResponseWriter, r *http.Request) {
	h.handleHeartbeatControlPause(w, r, "")
}

// ResumeHeartbeatControl resumes global heartbeat starts.
func (h *Handlers) ResumeHeartbeatControl(w http.ResponseWriter, r *http.Request) {
	h.handleHeartbeatControlResume(w, r, "")
}

// GetTeamHeartbeatControl returns heartbeat pause/policy status for one team.
func (h *Handlers) GetTeamHeartbeatControl(w http.ResponseWriter, r *http.Request) {
	if h.controlStore == nil {
		http.Error(w, "heartbeat control store not configured", http.StatusServiceUnavailable)
		return
	}
	teamID := mux.Vars(r)["id"]
	if _, err := h.teamStore.Get(r.Context(), teamID); err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}
	resp, err := h.controlStore.TeamStatus(r.Context(), teamID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// UpdateTeamHeartbeatControlPolicy updates one team's auto-pause override.
func (h *Handlers) UpdateTeamHeartbeatControlPolicy(w http.ResponseWriter, r *http.Request) {
	if h.controlStore == nil {
		http.Error(w, "heartbeat control store not configured", http.StatusServiceUnavailable)
		return
	}
	teamID := mux.Vars(r)["id"]
	if _, err := h.teamStore.Get(r.Context(), teamID); err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}
	attr, isHuman, err := attributionFromRequest(r, teamID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !isHuman {
		http.Error(w, "operator-direct attribution is required", http.StatusForbidden)
		return
	}
	var req HeartbeatControlTeamPolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	resp, err := h.controlStore.UpdateTeamPolicy(r.Context(), teamID, req, attr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// PauseTeamHeartbeatControl manually pauses heartbeat starts for one team.
func (h *Handlers) PauseTeamHeartbeatControl(w http.ResponseWriter, r *http.Request) {
	h.handleHeartbeatControlPause(w, r, mux.Vars(r)["id"])
}

// ResumeTeamHeartbeatControl resumes heartbeat starts for one team.
func (h *Handlers) ResumeTeamHeartbeatControl(w http.ResponseWriter, r *http.Request) {
	h.handleHeartbeatControlResume(w, r, mux.Vars(r)["id"])
}

func (h *Handlers) handleHeartbeatControlPause(w http.ResponseWriter, r *http.Request, teamID string) {
	if h.controlStore == nil {
		http.Error(w, "heartbeat control store not configured", http.StatusServiceUnavailable)
		return
	}
	if teamID != "" {
		if _, err := h.teamStore.Get(r.Context(), teamID); err != nil {
			http.Error(w, "Team not found", http.StatusNotFound)
			return
		}
	}
	attr, isHuman, err := attributionFromRequest(r, teamID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !isHuman {
		http.Error(w, "operator-direct attribution is required", http.StatusForbidden)
		return
	}
	var req HeartbeatControlPauseRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	resp, err := h.controlStore.Pause(r.Context(), teamID, req.Reason, attr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.unscheduleHeartbeats(r.Context(), teamID)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handlers) handleHeartbeatControlResume(w http.ResponseWriter, r *http.Request, teamID string) {
	if h.controlStore == nil {
		http.Error(w, "heartbeat control store not configured", http.StatusServiceUnavailable)
		return
	}
	if teamID != "" {
		if _, err := h.teamStore.Get(r.Context(), teamID); err != nil {
			http.Error(w, "Team not found", http.StatusNotFound)
			return
		}
	}
	attr, isHuman, err := attributionFromRequest(r, teamID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !isHuman {
		http.Error(w, "operator-direct attribution is required", http.StatusForbidden)
		return
	}
	resp, err := h.controlStore.Resume(r.Context(), teamID, attr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.scheduleEnabledHeartbeats(r.Context(), teamID)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handlers) operatorEngagementEventFromRequest(r *http.Request, teamID, reason string) (*HumanEngagementEvent, error) {
	if h.controlStore == nil {
		return nil, nil
	}
	attr, isHuman, err := attributionFromRequest(r, teamID)
	if err != nil {
		return nil, err
	}
	if !isHuman {
		return nil, nil
	}
	return &HumanEngagementEvent{
		TeamID:      teamID,
		Reason:      reason,
		Attribution: attr,
	}, nil
}

func (h *Handlers) recordOperatorEngagementEvent(ctx context.Context, event *HumanEngagementEvent) error {
	if h.controlStore == nil || event == nil {
		return nil
	}
	return h.controlStore.RecordHumanEngagement(ctx, *event)
}

func (h *Handlers) scheduleEnabledHeartbeats(ctx context.Context, teamID string) {
	if h.scheduler == nil {
		return
	}
	teams, err := h.teamStore.List(ctx)
	if err != nil {
		log.Printf("heartbeat control: list teams for resume failed: %v", err)
		return
	}
	for _, team := range teams {
		if teamID != "" && team.ID != teamID {
			continue
		}
		if !team.Enabled {
			continue
		}
		configs, err := h.teamStore.ListHeartbeatConfigs(ctx, team.ID)
		if err != nil {
			log.Printf("heartbeat control: list heartbeat configs for %s failed: %v", team.ID, err)
			continue
		}
		for _, config := range configs {
			if !config.Enabled {
				continue
			}
			if err := h.scheduler.Schedule(config.TeamID, config.AgentID, config.Schedule); err != nil {
				log.Printf("heartbeat control: schedule %s/%s after resume failed: %v", config.TeamID, config.AgentID, err)
			}
		}
	}
}

func (h *Handlers) unscheduleHeartbeats(ctx context.Context, teamID string) {
	if h.scheduler == nil {
		return
	}
	teams, err := h.teamStore.List(ctx)
	if err != nil {
		log.Printf("heartbeat control: list teams for pause failed: %v", err)
		return
	}
	for _, team := range teams {
		if teamID != "" && team.ID != teamID {
			continue
		}
		configs, err := h.teamStore.ListHeartbeatConfigs(ctx, team.ID)
		if err != nil {
			continue
		}
		for _, config := range configs {
			h.scheduler.Unschedule(config.TeamID, config.AgentID)
		}
	}
}

func writeHeartbeatPaused(w http.ResponseWriter, paused *HeartbeatPausedErrorResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusLocked)
	_ = json.NewEncoder(w).Encode(paused)
}

// TriggerTeam handles POST /teams/{id}/trigger - triggers the team according to its runtime and coordination policy.
func (h *Handlers) TriggerTeam(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	teamID := vars["id"]

	if h.executor == nil {
		http.Error(w, "Executor not configured", http.StatusServiceUnavailable)
		return
	}

	team, err := h.teamStore.Get(ctx, teamID)
	if err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}

	if !team.Enabled {
		http.Error(w, "Team is disabled; enable the team to run heartbeats", http.StatusConflict)
		return
	}
	if h.controlStore != nil {
		if paused, err := h.controlStore.AllowStart(ctx, teamID); err != nil {
			if errors.Is(err, ErrHeartbeatPaused) {
				writeHeartbeatPaused(w, paused)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	engagementEvent, err := h.operatorEngagementEventFromRequest(r, teamID, "team-heartbeat-manual-trigger")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	contract := team.Contract()
	if teamconfig.TeamTriggerTargetsLead(contract) {
		leadAgentID := team.Coordination.LeadAgentID
		config, cfgErr := h.requireActiveLeadHeartbeatConfig(ctx, team)
		if cfgErr != nil {
			status := http.StatusBadRequest
			if strings.Contains(strings.ToLower(cfgErr.Error()), "not configured") {
				status = http.StatusServiceUnavailable
			}
			http.Error(w, cfgErr.Error(), status)
			return
		}

		// Route through team execution store if available
		if h.teamExecStore != nil {
			profileKey := DefaultProfileKeyForRuntimeMode(team.Runtime.Mode)
			if config.ProfileKey != "" {
				profileKey = config.ProfileKey
			}

			enqueueResult, enqErr := h.teamExecStore.Enqueue(ctx, teamID, leadAgentID, profileKey)
			if enqErr != nil {
				if IsMemberAlreadyQueued(enqErr) {
					http.Error(w, enqErr.Error(), http.StatusConflict)
					return
				}
				http.Error(w, enqErr.Error(), http.StatusInternalServerError)
				return
			}
			if err := h.recordOperatorEngagementEvent(ctx, engagementEvent); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(TriggerTeamResponse{
				TeamID:              teamID,
				RuntimeMode:         team.Runtime.Mode,
				CoordinationPattern: team.Coordination.Pattern,
				QueuePolicy:         team.Execution.QueuePolicy,
				Triggers:            []TriggerHeartbeatResponse{{TeamID: teamID, AgentID: leadAgentID, Status: enqueueResult.Status}},
			})
			return
		}

		result, err := h.executor.TriggerManual(ctx, teamID, leadAgentID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := h.recordOperatorEngagementEvent(ctx, engagementEvent); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(TriggerTeamResponse{
			TeamID:              teamID,
			RuntimeMode:         team.Runtime.Mode,
			CoordinationPattern: team.Coordination.Pattern,
			QueuePolicy:         team.Execution.QueuePolicy,
			Triggers:            []TriggerHeartbeatResponse{{TeamID: result.TeamID, AgentID: result.AgentID, RunID: result.RunID, Status: result.Status, LogPath: result.LogPath}},
		})
		return
	}

	// Multi-process teams trigger all member heartbeats with configs.
	members, err := h.relationStore.ListTeamMembers(ctx, teamID)
	if err != nil {
		http.Error(w, "failed to list team members", http.StatusInternalServerError)
		return
	}

	var triggers []TriggerHeartbeatResponse
	for _, m := range members {
		// Only trigger members that have heartbeat configs
		config, err := h.teamStore.GetHeartbeatConfig(ctx, teamID, m.AgentID)
		if err != nil || config == nil {
			continue
		}

		// Route through team execution store if available
		if h.teamExecStore != nil {
			profileKey := DefaultProfileKeyForRuntimeMode(team.Runtime.Mode)
			if config.ProfileKey != "" {
				profileKey = config.ProfileKey
			}

			enqueueResult, enqErr := h.teamExecStore.Enqueue(ctx, teamID, m.AgentID, profileKey)
			if enqErr != nil {
				if IsMemberAlreadyQueued(enqErr) {
					log.Printf("Warning: Skipped heartbeat for %s/%s: already queued", teamID, m.AgentID)
					continue
				}
				log.Printf("Warning: Failed to enqueue heartbeat for %s/%s: %v", teamID, m.AgentID, enqErr)
				continue
			}
			triggers = append(triggers, TriggerHeartbeatResponse{
				TeamID:  teamID,
				AgentID: m.AgentID,
				Status:  enqueueResult.Status,
			})
			continue
		}

		result, err := h.executor.TriggerManual(ctx, teamID, m.AgentID)
		if err != nil {
			log.Printf("Warning: Failed to trigger heartbeat for %s/%s: %v", teamID, m.AgentID, err)
			continue
		}
		triggers = append(triggers, TriggerHeartbeatResponse{
			TeamID:  result.TeamID,
			AgentID: result.AgentID,
			RunID:   result.RunID,
			Status:  result.Status,
			LogPath: result.LogPath,
		})
	}

	if err := h.recordOperatorEngagementEvent(ctx, engagementEvent); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(TriggerTeamResponse{
		TeamID:              teamID,
		RuntimeMode:         team.Runtime.Mode,
		CoordinationPattern: team.Coordination.Pattern,
		QueuePolicy:         team.Execution.QueuePolicy,
		Triggers:            triggers,
	})
}

// GetRun handles GET /runs/{runId} - proxies run details to agent-manager.
func (h *Handlers) GetRun(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	runID := vars["runId"]

	if runID == "" {
		http.Error(w, "runId is required", http.StatusBadRequest)
		return
	}

	if h.agentClient == nil {
		http.Error(w, "agent client not configured", http.StatusServiceUnavailable)
		return
	}

	run, err := h.agentClient.GetRun(r.Context(), runID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	if run == nil {
		http.Error(w, "run not found", http.StatusNotFound)
		return
	}

	resp := map[string]interface{}{"run": run}
	if tag := strings.TrimSpace(run.Tag); tag != "" {
		if teamID, agentID, err := h.resolveHeartbeatTargetFromRunTag(r.Context(), tag); err == nil {
			resp["team_id"] = teamID
			resp["agent_id"] = agentID
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// GetRunEvents handles GET /runs/{runId}/events - proxies event requests to agent-manager.
func (h *Handlers) GetRunEvents(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	runID := vars["runId"]

	if runID == "" {
		http.Error(w, "runId is required", http.StatusBadRequest)
		return
	}

	afterSequence := int64(-1)
	if v := r.URL.Query().Get("after_sequence"); v != "" {
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			http.Error(w, "invalid after_sequence", http.StatusBadRequest)
			return
		}
		afterSequence = parsed
	}

	limit := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			http.Error(w, "invalid limit", http.StatusBadRequest)
			return
		}
		limit = parsed
	}

	if h.agentClient == nil {
		http.Error(w, "agent client not configured", http.StatusServiceUnavailable)
		return
	}

	data, err := h.agentClient.GetRunEvents(r.Context(), runID, afterSequence, limit)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(data)
}

// ListRuns handles GET /runs - proxies run listing to agent-manager.
func (h *Handlers) ListRuns(w http.ResponseWriter, r *http.Request) {
	if h.agentClient == nil {
		http.Error(w, "agent client not configured", http.StatusServiceUnavailable)
		return
	}

	opts := ListRunsOptions{
		Status:                    r.URL.Query().Get("status"),
		TagPrefix:                 r.URL.Query().Get("tag_prefix"),
		ProfileKey:                r.URL.Query().Get("profile_key"),
		TaskID:                    r.URL.Query().Get("task_id"),
		InvestigatesRunID:         r.URL.Query().Get("investigates_run_id"),
		AppliesInvestigationRunID: r.URL.Query().Get("applies_investigation_run_id"),
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			http.Error(w, "invalid limit", http.StatusBadRequest)
			return
		}
		opts.Limit = parsed
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			http.Error(w, "invalid offset", http.StatusBadRequest)
			return
		}
		opts.Offset = parsed
	}

	// agent-manager list-runs filters by agent_profile_id (UUID), not profile_key.
	// Resolve profile_key to ID before forwarding so UI profile scoping works.
	if opts.ProfileKey != "" {
		resolved, err := h.agentClient.EnsureProfile(r.Context(), &EnsureProfileRequest{
			ProfileKey: opts.ProfileKey,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		if resolved == nil || resolved.Profile == nil || resolved.Profile.ID == "" {
			http.Error(w, "failed to resolve profile_key to profile id", http.StatusBadGateway)
			return
		}
		opts.AgentProfileID = resolved.Profile.ID
	}

	result, err := h.agentClient.ListRuns(r.Context(), opts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// ListHeartbeatAttempts handles GET /heartbeat-attempts - lists local heartbeat
// dispatch attempts, including failures that happened before agent-manager
// could create a run.
func (h *Handlers) ListHeartbeatAttempts(w http.ResponseWriter, r *http.Request) {
	if h.teamStore == nil {
		http.Error(w, "team store not configured", http.StatusServiceUnavailable)
		return
	}

	limit := 100
	offset := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			http.Error(w, "invalid limit", http.StatusBadRequest)
			return
		}
		limit = parsed
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			http.Error(w, "invalid offset", http.StatusBadRequest)
			return
		}
		offset = parsed
	}

	attempts, total, err := h.teamStore.ListHeartbeatAttempts(
		r.Context(),
		r.URL.Query().Get("team_id"),
		r.URL.Query().Get("agent_id"),
		r.URL.Query().Get("status"),
		r.URL.Query().Get("profile_key"),
		limit,
		offset,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(HeartbeatAttemptListResponse{
		Attempts: attempts,
		Total:    total,
		HasMore:  offset+len(attempts) < total,
	})
}

// ContinueRun handles POST /runs/{runId}/continue - sends a continue message to a paused run.
func (h *Handlers) ContinueRun(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	runID := vars["runId"]

	if runID == "" {
		http.Error(w, "runId is required", http.StatusBadRequest)
		return
	}

	if h.agentClient == nil {
		http.Error(w, "agent client not configured", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	run, err := h.agentClient.ContinueRun(r.Context(), runID, req.Message)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"run": run})
}

// CreateInvestigationRun handles POST /runs/investigate - creates an investigation run.
func (h *Handlers) CreateInvestigationRun(w http.ResponseWriter, r *http.Request) {
	if h.agentClient == nil {
		w.Header().Set("X-Vrooli-Error-Hop", "prompt-manager-api")
		http.Error(w, "agent client not configured", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		RunIDs        []string `json:"run_ids"`
		Depth         string   `json:"depth"`
		CustomContext string   `json:"custom_context"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	run, err := h.agentClient.CreateInvestigationRun(r.Context(), req.RunIDs, req.Depth, req.CustomContext)
	if err != nil {
		w.Header().Set("X-Vrooli-Error-Hop", "prompt-manager-api->agent-manager")
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"run": run})
}

// CreateInvestigationApplyRun handles POST /runs/investigation-apply - applies an investigation.
func (h *Handlers) CreateInvestigationApplyRun(w http.ResponseWriter, r *http.Request) {
	if h.agentClient == nil {
		w.Header().Set("X-Vrooli-Error-Hop", "prompt-manager-api")
		http.Error(w, "agent client not configured", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		InvestigationRunID string `json:"investigation_run_id"`
		CustomContext      string `json:"custom_context"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	run, err := h.agentClient.CreateInvestigationApplyRun(r.Context(), req.InvestigationRunID, req.CustomContext)
	if err != nil {
		w.Header().Set("X-Vrooli-Error-Hop", "prompt-manager-api->agent-manager")
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"run": run})
}

// CreateTask handles POST /tasks - creates a task via agent-manager.
func (h *Handlers) CreateTask(w http.ResponseWriter, r *http.Request) {
	if h.agentClient == nil {
		http.Error(w, "agent client not configured", http.StatusServiceUnavailable)
		return
	}

	var req CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	task, err := h.agentClient.CreateTask(r.Context(), req.Task)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"task": task})
}

// CreateRun handles POST /runs - creates a run via agent-manager.
func (h *Handlers) CreateRun(w http.ResponseWriter, r *http.Request) {
	if h.agentClient == nil {
		http.Error(w, "agent client not configured", http.StatusServiceUnavailable)
		return
	}

	var req CreateRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	run, err := h.agentClient.CreateRun(r.Context(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"run": run})
}

// GetTeamExecutionStatus handles GET /teams/{id}/execution-status - returns team execution queue state.
func (h *Handlers) GetTeamExecutionStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	teamID := vars["id"]

	team, err := h.teamStore.Get(ctx, teamID)
	if err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}

	if h.teamExecStore == nil {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(TeamExecutionStatus{
			TeamID:            teamID,
			State:             "idle",
			RunningAgentIDs:   []string{},
			Queue:             []string{},
			QueuePolicy:       team.Execution.QueuePolicy,
			MaxConcurrentRuns: team.Execution.MaxConcurrentRuns,
		})
		return
	}

	status := h.teamExecStore.Status(teamID)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(status)
}

// GetMemberContext handles GET /teams/{id}/members/{agentId}/context
func (h *Handlers) GetMemberContext(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	teamID := vars["id"]
	agentID := vars["agentId"]

	if h.executor == nil {
		http.Error(w, "Executor not configured", http.StatusServiceUnavailable)
		return
	}

	if _, err := h.teamStore.Get(ctx, teamID); err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}
	if err := h.requireMember(ctx, teamID, agentID); err != nil {
		if errors.Is(err, errMemberNotFound) {
			http.Error(w, "Team member not found", http.StatusNotFound)
			return
		}
		if strings.Contains(err.Error(), "required") {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	prompt, err := h.executor.BuildContext(ctx, teamID, agentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(MemberContextResponse{
		TeamID:  teamID,
		AgentID: agentID,
		Prompt:  prompt,
	})
}

// formatDuration returns a human-readable duration string.
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

// toResponse converts a HeartbeatConfig to API response
func (h *Handlers) toResponse(config *store.HeartbeatConfig) HeartbeatConfigResponse {
	resp := HeartbeatConfigResponse{
		TeamID:         config.TeamID,
		AgentID:        config.AgentID,
		Enabled:        config.Enabled,
		Schedule:       config.Schedule,
		ProfileKey:     config.ProfileKey,
		TimeoutSeconds: config.TimeoutSeconds,
		CreatedAt:      config.CreatedAt,
		UpdatedAt:      config.UpdatedAt,
	}

	if config.LastExecution != nil {
		resp.LastExecution = &HeartbeatExecResultDTO{
			StartedAt: config.LastExecution.StartedAt,
			EndedAt:   config.LastExecution.EndedAt,
			Status:    config.LastExecution.Status,
			RunID:     config.LastExecution.RunID,
			LogPath:   filepath.Base(config.LastExecution.LogPath),
			Error:     config.LastExecution.Error,
		}
	}

	// Get next execution times from scheduler
	if config.Enabled && h.scheduler != nil {
		if nextRun := h.scheduler.GetNextRun(config.TeamID, config.AgentID); nextRun != nil {
			resp.NextExecution = nextRun.Format(time.RFC3339)
		}
		if runs := h.scheduler.GetNextRuns(config.TeamID, config.AgentID, 5); len(runs) > 0 {
			resp.NextExecutions = make([]string, len(runs))
			for i, r := range runs {
				resp.NextExecutions[i] = r.Format(time.RFC3339)
			}
		}
	}

	return resp
}

// --- Handoff handlers ---

// GetLastHandoff handles GET /teams/{id}/members/{agentId}/handoff
func (h *Handlers) GetLastHandoff(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	teamID := vars["id"]
	agentID := vars["agentId"]

	content, err := h.teamStore.GetLastHandoff(r.Context(), teamID, agentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if content == "" {
		http.Error(w, "no handoff found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(HandoffResponse{
		TeamID:  teamID,
		AgentID: agentID,
		Content: content,
	})
}

// GetHandoffHistory handles GET /teams/{id}/handoff-history
func (h *Handlers) GetHandoffHistory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	teamID := vars["id"]
	agentFilter := r.URL.Query().Get("agent")
	last := 20
	if v := r.URL.Query().Get("last"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			last = n
		}
	}

	entries, err := h.teamStore.GetHandoffHistory(r.Context(), teamID, agentFilter, last)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if entries == nil {
		entries = []store.HandoffEntry{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(HandoffHistoryResponse{
		TeamID:  teamID,
		Entries: entries,
	})
}

// ClearHandoffHistory handles DELETE /teams/{id}/handoff-history
func (h *Handlers) ClearHandoffHistory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	teamID := vars["id"]
	agentFilter := r.URL.Query().Get("agent")

	if err := h.teamStore.ClearHandoffHistory(r.Context(), teamID, agentFilter); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ClearLastHandoff handles DELETE /teams/{id}/members/{agentId}/handoff
func (h *Handlers) ClearLastHandoff(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	teamID := vars["id"]
	agentID := vars["agentId"]

	if err := h.teamStore.ClearLastHandoff(r.Context(), teamID, agentID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Task Board handlers ---

// GetTaskBoard handles GET /teams/{id}/tasks
func (h *Handlers) GetTaskBoard(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	teamID := vars["id"]

	// Parse pagination params
	limit := 25
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	offset := 0
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	statusFilter := r.URL.Query().Get("status")
	assigneeFilter := r.URL.Query().Get("assignee")

	board, err := h.teamStore.GetTaskBoard(r.Context(), teamID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Apply filters
	tasks := board.Tasks
	if statusFilter != "" || assigneeFilter != "" {
		filtered := make([]store.TeamTask, 0, len(tasks))
		for _, t := range tasks {
			if statusFilter != "" && t.Status != statusFilter {
				continue
			}
			if assigneeFilter != "" && t.Assignee != assigneeFilter {
				continue
			}
			filtered = append(filtered, t)
		}
		tasks = filtered
	}

	total := len(tasks)

	// Apply offset/limit
	if offset >= len(tasks) {
		tasks = nil
	} else {
		end := offset + limit
		if end > len(tasks) {
			end = len(tasks)
		}
		tasks = tasks[offset:end]
	}
	if tasks == nil {
		tasks = []store.TeamTask{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(TaskBoardResponse{
		TeamID: teamID,
		Tasks:  tasks,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

// AddTask handles POST /teams/{id}/tasks
func (h *Handlers) AddTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	teamID := vars["id"]

	var req AddTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		http.Error(w, "title is required", http.StatusBadRequest)
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	task := store.TeamTask{
		ID:        fmt.Sprintf("task-%s", generateID()),
		Title:     req.Title,
		Status:    "todo",
		Assignee:  req.Assignee,
		Priority:  req.Priority,
		CreatedBy: req.From,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if task.Priority == "" {
		task.Priority = "P3"
	}

	board, err := h.teamStore.GetTaskBoard(r.Context(), teamID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	board.Tasks = append(board.Tasks, task)
	if err := h.teamStore.SaveTaskBoard(r.Context(), teamID, board); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(task)
}

// UpdateTaskHandler handles PATCH /teams/{id}/tasks/{taskId}
func (h *Handlers) UpdateTaskHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	teamID := vars["id"]
	taskID := vars["taskId"]

	var req UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	err := h.teamStore.UpdateTask(r.Context(), teamID, taskID, func(task *store.TeamTask) {
		if req.Title != nil && strings.TrimSpace(*req.Title) != "" {
			task.Title = *req.Title
		}
		if req.Status != nil {
			task.Status = *req.Status
		}
		if req.Assignee != nil {
			task.Assignee = *req.Assignee
		}
		if req.Priority != nil {
			task.Priority = *req.Priority
		}
		if req.Note != nil && strings.TrimSpace(*req.Note) != "" {
			task.Notes = append(task.Notes, store.TaskNote{
				At:   now,
				By:   "", // could extract from auth context
				Text: *req.Note,
			})
		}
		task.UpdatedAt = now
	})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	task, err := h.teamStore.GetTask(r.Context(), teamID, taskID)
	if err != nil || task == nil {
		http.Error(w, "task updated but fetch failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(task)
}

// DeleteTaskHandler handles DELETE /teams/{id}/tasks/{taskId}
func (h *Handlers) DeleteTaskHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	teamID := vars["id"]
	taskID := vars["taskId"]

	err := h.teamStore.DeleteTask(r.Context(), teamID, taskID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- Decision Log handlers ---

// AddDecision handles POST /teams/{id}/decisions
func (h *Handlers) AddDecision(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	teamID := vars["id"]

	var req AddDecisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	// Multi-option decisions use topic+options; simple decisions use decision+rationale.
	if len(req.Options) > 0 {
		if strings.TrimSpace(req.Topic) == "" {
			http.Error(w, "topic is required when options are provided", http.StatusBadRequest)
			return
		}
	} else {
		if strings.TrimSpace(req.Decision) == "" {
			http.Error(w, "decision is required", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.Rationale) == "" {
			http.Error(w, "rationale is required", http.StatusBadRequest)
			return
		}
	}

	// initiative_metadata is only valid on initiative-proposal decisions.
	if req.InitiativeMetadata != nil {
		if req.Context != store.DecisionContextInitiativeProposal {
			writeDecisionFieldError(w, "initiative_metadata", "initiative_metadata is only valid when context=initiative-proposal")
			return
		}
		if err := validateInitiativeMetadata(req.InitiativeMetadata); err != nil {
			writeDecisionFieldError(w, "initiative_metadata", err.Error())
			return
		}
	}

	entry := &store.DecisionEntry{
		ID:                 fmt.Sprintf("dec-%s", generateID()),
		At:                 time.Now().UTC().Format(time.RFC3339),
		By:                 req.By,
		Decision:           req.Decision,
		Rationale:          req.Rationale,
		Context:            req.Context,
		Supersedes:         req.Supersedes,
		Status:             store.DecisionStatusPending,
		Topic:              req.Topic,
		Description:        req.Description,
		Options:            req.Options,
		InitiativeMetadata: req.InitiativeMetadata,
	}

	if err := h.teamStore.AppendDecision(r.Context(), teamID, entry); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(entry)
}

// GetDecisions handles GET /teams/{id}/decisions
func (h *Handlers) GetDecisions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	teamID := vars["id"]
	contextFilter := r.URL.Query().Get("context")
	statusFilter := r.URL.Query().Get("status")
	last := 10
	if v := r.URL.Query().Get("last"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			last = n
		}
	}

	entries, total, err := h.teamStore.GetDecisions(r.Context(), teamID, contextFilter, statusFilter, last)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if entries == nil {
		entries = []store.DecisionEntry{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(DecisionListResponse{
		TeamID:  teamID,
		Entries: entries,
		Total:   total,
		Last:    last,
	})
}

// UpdateDecisionHandler handles PATCH /teams/{id}/decisions/{decisionId}
func (h *Handlers) UpdateDecisionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	teamID := vars["id"]
	decisionID := vars["decisionId"]

	var req UpdateDecisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Modifications != nil {
		if err := validateDecisionModifications(req.Modifications); err != nil {
			writeDecisionFieldError(w, "modifications", err.Error())
			return
		}
		// Immutability: if the stored decision already has modifications, reject
		// any attempt to change them (accept-once semantics).
		existing, _ := h.findDecision(r.Context(), teamID, decisionID)
		if existing != nil && existing.Modifications != nil {
			writeDecisionFieldError(w, "modifications", "modifications are immutable once set on an accepted decision")
			return
		}
	}

	// initiative_metadata: validate shape, scope to initiative-proposal context,
	// and enforce post-accept immutability (mirrors modifications rule).
	if req.InitiativeMetadata != nil {
		if err := validateInitiativeMetadata(req.InitiativeMetadata); err != nil {
			writeDecisionFieldError(w, "initiative_metadata", err.Error())
			return
		}
		existing, _ := h.findDecision(r.Context(), teamID, decisionID)
		if existing != nil {
			ctxTag := existing.Context
			if req.Context != nil {
				ctxTag = *req.Context
			}
			if ctxTag != store.DecisionContextInitiativeProposal {
				writeDecisionFieldError(w, "initiative_metadata", "initiative_metadata is only valid when context=initiative-proposal")
				return
			}
			if existing.Status == store.DecisionStatusAccepted {
				writeDecisionFieldError(w, "initiative_metadata", "initiative_metadata is immutable once the decision has been accepted")
				return
			}
		}
	}

	// auto_create_status manual-recovery transitions (per d8=C). Allowed only
	// from a "failed" starting state. failed→created requires a non-empty ref.
	if req.AutoCreateStatus != nil {
		newStatus := strings.TrimSpace(*req.AutoCreateStatus)
		existing, _ := h.findDecision(r.Context(), teamID, decisionID)
		if existing == nil {
			writeDecisionFieldError(w, "auto_create_status", "decision not found")
			return
		}
		if existing.AutoCreateStatus != store.AutoCreateStatusFailed {
			writeDecisionFieldError(w, "auto_create_status", fmt.Sprintf("auto_create_status can only be transitioned from %q (current: %q)", store.AutoCreateStatusFailed, existing.AutoCreateStatus))
			return
		}
		switch newStatus {
		case store.AutoCreateStatusCreated:
			if req.AutoCreateInitiativeRef == nil || strings.TrimSpace(*req.AutoCreateInitiativeRef) == "" {
				writeDecisionFieldError(w, "auto_create_initiative_ref", "auto_create_initiative_ref is required when setting auto_create_status=created")
				return
			}
		case store.AutoCreateStatusFailed:
			// re-record an updated error — allowed
		default:
			writeDecisionFieldError(w, "auto_create_status", fmt.Sprintf("auto_create_status %q is not a permitted transition (allowed: %q, %q)", newStatus, store.AutoCreateStatusCreated, store.AutoCreateStatusFailed))
			return
		}
	}

	// Determine effective status: if selecting an option on a pending decision, implicitly accept.
	effectiveStatus := req.Status
	if req.Selected != nil && strings.TrimSpace(*req.Selected) != "" && req.Status == nil {
		accepted := store.DecisionStatusAccepted
		effectiveStatus = &accepted
	}

	// Single-proposal decisions (no Options) are accepted without --selected.
	// Reject any explicit --selected on this shape — including the legacy
	// `__other__ + freeform="accept as proposed"` workaround. Reads of
	// historical entries with that shape are still tolerated by the read path.
	acceptAsProposed := false
	if effectiveStatus != nil && *effectiveStatus == store.DecisionStatusAccepted {
		if existing, _ := h.findDecision(r.Context(), teamID, decisionID); existing != nil && len(existing.Options) == 0 {
			if req.Selected != nil && strings.TrimSpace(*req.Selected) != "" {
				writeDecisionFieldError(w, "selected", "single-proposal decisions are accepted with no --selected; rerun without it")
				return
			}
			acceptAsProposed = true
		}
	}

	// Defer-specific validation and transition checks.
	var deferRevisitDate string // canonical YYYY-MM-DD form, set when transitioning to deferred
	var deferAuditNote string   // appended to existing notes (re-defer only)
	if effectiveStatus != nil && *effectiveStatus == store.DecisionStatusDeferred {
		if req.RevisitAfter == nil || strings.TrimSpace(*req.RevisitAfter) == "" {
			writeDecisionFieldError(w, "revisit_after", "revisit_after is required when status=deferred (format: YYYY-MM-DD)")
			return
		}
		parsed, err := time.Parse("2006-01-02", strings.TrimSpace(*req.RevisitAfter))
		if err != nil {
			writeDecisionFieldError(w, "revisit_after", "revisit_after must be an ISO-8601 date (YYYY-MM-DD)")
			return
		}
		today := time.Now().UTC().Truncate(24 * time.Hour)
		parsedDay := parsed.UTC().Truncate(24 * time.Hour)
		if parsedDay.Before(today) {
			writeDecisionFieldError(w, "revisit_after", "revisit_after must be today or in the future")
			return
		}
		maxDate := today.AddDate(0, 0, store.MaxRevisitAfterDays)
		if parsedDay.After(maxDate) {
			writeDecisionFieldError(w, "revisit_after", fmt.Sprintf("revisit_after must be within %d days of today", store.MaxRevisitAfterDays))
			return
		}
		deferRevisitDate = parsedDay.Format("2006-01-02")

		// Validate source transition.
		existing, _ := h.findDecision(r.Context(), teamID, decisionID)
		if existing != nil {
			switch existing.Status {
			case store.DecisionStatusPending, "":
				// fresh defer — ok
			case store.DecisionStatusDeferred:
				// re-defer in place — preserve audit trail
				prev := ""
				if existing.RevisitAfter != nil {
					prev = *existing.RevisitAfter
				}
				deferAuditNote = fmt.Sprintf("[re-deferred] %s → %s", prev, deferRevisitDate)
			default:
				writeDecisionFieldError(w, "status", fmt.Sprintf("cannot defer decision in status %q (only pending or deferred can be deferred)", existing.Status))
				return
			}
		}
	}

	// Validate transitions OUT of deferred to a non-defer status.
	if effectiveStatus != nil && *effectiveStatus != store.DecisionStatusDeferred {
		existing, _ := h.findDecision(r.Context(), teamID, decisionID)
		if existing != nil && existing.Status == store.DecisionStatusDeferred {
			switch *effectiveStatus {
			case store.DecisionStatusPending, store.DecisionStatusAccepted, store.DecisionStatusRejected:
				// allowed — un-defer / accept / reject
			default:
				writeDecisionFieldError(w, "status", fmt.Sprintf("cannot transition deferred decision to %q (allowed: pending, accepted, rejected, deferred)", *effectiveStatus))
				return
			}
		}
	}

	// Approval enforcement: check if the team is in approval mode and restrict agent callers.
	if effectiveStatus != nil {
		if blocked, msg := h.checkApprovalEnforcement(r, teamID, decisionID, *effectiveStatus); blocked {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(msg)
			return
		}
	}
	var engagementAttribution store.AttributionInfo
	shouldRecordEngagement := false
	if effectiveStatus != nil && isHumanDecisionEngagementStatus(*effectiveStatus) && h.controlStore != nil {
		attr, isHuman, err := attributionFromRequest(r, teamID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if isHuman {
			engagementAttribution = attr
			shouldRecordEngagement = true
		}
	}

	// d5: accepting an initiative-proposal decision requires initiative_metadata
	// to be present (either already on the decision or supplied in this PATCH).
	if effectiveStatus != nil && *effectiveStatus == store.DecisionStatusAccepted {
		existing, _ := h.findDecision(r.Context(), teamID, decisionID)
		if existing != nil && existing.Context == store.DecisionContextInitiativeProposal {
			if existing.InitiativeMetadata == nil && req.InitiativeMetadata == nil {
				writeDecisionFieldError(w, "initiative_metadata",
					"add --initiative-metadata to the decision (or use decision-update) before accepting")
				return
			}
		}
	}

	err := h.teamStore.UpdateDecision(r.Context(), teamID, decisionID, func(d *store.DecisionEntry) {
		if req.Decision != nil && strings.TrimSpace(*req.Decision) != "" {
			d.Decision = *req.Decision
		}
		if req.Rationale != nil {
			d.Rationale = *req.Rationale
		}
		if req.Context != nil {
			d.Context = *req.Context
		}
		if effectiveStatus != nil {
			d.Status = *effectiveStatus
		}
		if req.Supersedes != nil {
			d.Supersedes = *req.Supersedes
		}
		if req.Topic != nil {
			d.Topic = *req.Topic
		}
		if req.Description != nil {
			d.Description = *req.Description
		}
		if req.Options != nil {
			d.Options = *req.Options
		}
		if req.Selected != nil {
			d.Selected = *req.Selected
		}
		if req.Freeform != nil {
			d.Freeform = *req.Freeform
		}
		if req.Notes != nil {
			d.Notes = *req.Notes
		}
		if req.Modifications != nil {
			m := *req.Modifications
			d.Modifications = &m
		}
		if req.InitiativeMetadata != nil {
			meta := *req.InitiativeMetadata
			d.InitiativeMetadata = &meta
		}
		if req.AutoCreateStatus != nil {
			d.AutoCreateStatus = strings.TrimSpace(*req.AutoCreateStatus)
			// Clear stale error when flipping to created.
			if d.AutoCreateStatus == store.AutoCreateStatusCreated {
				d.AutoCreateError = ""
			}
		}
		if req.AutoCreateError != nil {
			d.AutoCreateError = *req.AutoCreateError
		}
		if req.AutoCreateInitiativeRef != nil {
			d.AutoCreateInitiativeRef = strings.TrimSpace(*req.AutoCreateInitiativeRef)
		}
		if acceptAsProposed {
			d.AcceptedAsProposed = true
		}
		// Defer-related state.
		if effectiveStatus != nil && *effectiveStatus == store.DecisionStatusDeferred {
			rev := deferRevisitDate
			d.RevisitAfter = &rev
			if deferAuditNote != "" {
				if strings.TrimSpace(d.Notes) == "" {
					d.Notes = deferAuditNote
				} else {
					d.Notes = d.Notes + "\n" + deferAuditNote
				}
			}
		} else if effectiveStatus != nil {
			// Transitioning out of deferred to pending/accepted/rejected — clear the date.
			d.RevisitAfter = nil
		}
	})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch updated entry to return
	entries, _, err := h.teamStore.GetDecisions(r.Context(), teamID, "", "", 0)
	if err != nil {
		http.Error(w, "decision updated but fetch failed", http.StatusInternalServerError)
		return
	}
	var updated *store.DecisionEntry
	for i := range entries {
		if entries[i].ID == decisionID {
			updated = &entries[i]
			break
		}
	}
	if updated == nil {
		http.Error(w, "decision updated but not found in response", http.StatusInternalServerError)
		return
	}
	if shouldRecordEngagement {
		if err := h.controlStore.RecordHumanEngagement(r.Context(), HumanEngagementEvent{
			TeamID:      teamID,
			Reason:      "decision-" + *effectiveStatus,
			Attribution: engagementAttribution,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Auto-create initiative when transitioning an initiative-proposal decision
	// to accepted. Failure does not roll back the accept (per d4=A); the
	// failure is persisted on the decision and surfaced in the response with
	// a pre-filled manual workaround (per d8=C + d9=A).
	var autoOutcome *AutoCreateOutcome
	justAcceptedProposal := effectiveStatus != nil &&
		*effectiveStatus == store.DecisionStatusAccepted &&
		updated.Context == store.DecisionContextInitiativeProposal &&
		updated.InitiativeMetadata != nil &&
		updated.AutoCreateStatus != store.AutoCreateStatusCreated
	if justAcceptedProposal {
		autoOutcome = h.runAutoCreateInitiative(r.Context(), teamID, updated)
		// Persist the outcome on the decision record.
		_ = h.teamStore.UpdateDecision(r.Context(), teamID, decisionID, func(d *store.DecisionEntry) {
			d.AutoCreateStatus = autoOutcome.Status
			if autoOutcome.Status == store.AutoCreateStatusCreated {
				d.AutoCreateInitiativeRef = autoOutcome.InitiativeRef
				d.AutoCreateError = ""
			} else {
				d.AutoCreateError = autoOutcome.Error
			}
		})
		// Reflect persisted state in the returned entry.
		updated.AutoCreateStatus = autoOutcome.Status
		if autoOutcome.Status == store.AutoCreateStatusCreated {
			updated.AutoCreateInitiativeRef = autoOutcome.InitiativeRef
			updated.AutoCreateError = ""
		} else {
			updated.AutoCreateError = autoOutcome.Error
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(UpdateDecisionResponse{
		DecisionEntry:     *updated,
		AutoCreateOutcome: autoOutcome,
	})
}

func isHumanDecisionEngagementStatus(status string) bool {
	switch status {
	case store.DecisionStatusAccepted, store.DecisionStatusRejected, store.DecisionStatusDeferred, store.DecisionStatusPending:
		return true
	default:
		return false
	}
}

// UpdateDecisionResponse extends the persisted decision with an optional
// AutoCreateOutcome payload populated when an `initiative-proposal` decision
// is accepted. The outcome carries (a) the success ref or (b) the failure
// reason plus the pre-filled manual-recovery commands.
type UpdateDecisionResponse struct {
	store.DecisionEntry
	AutoCreateOutcome *AutoCreateOutcome `json:"auto_create_outcome,omitempty"`
}

// approvalError is the JSON response body for approval enforcement failures.
type approvalError struct {
	Error         string `json:"error"`
	Message       string `json:"message"`
	CurrentStatus string `json:"currentStatus,omitempty"`
}

// checkApprovalEnforcement checks whether the requested status transition is allowed.
// Returns (true, error) if blocked, (false, nil) if allowed.
func (h *Handlers) checkApprovalEnforcement(r *http.Request, teamID, decisionID, newStatus string) (bool, *approvalError) {
	ctx := r.Context()

	// Fetch team to check decision mode
	team, err := h.teamStore.Get(ctx, teamID)
	if err != nil || team == nil {
		return false, nil // fail open if team not found
	}

	if team.DecisionMode != "approval" {
		return false, nil // yolo mode — no restrictions
	}

	// Determine if caller is an agent
	callerID := r.Header.Get("X-Caller-ID")
	if callerID == "" || callerID == "ui-user" {
		return false, nil // human caller — no restrictions
	}

	// Check if the caller ID matches a team member (agent)
	isAgent := false
	if h.relationStore != nil {
		members, err := h.relationStore.ListTeamMembers(ctx, teamID)
		if err == nil {
			for _, m := range members {
				if m.AgentID == callerID {
					isAgent = true
					break
				}
			}
		}
	}
	if !isAgent {
		return false, nil // not a known agent — treat as human
	}

	// Agents in approval mode cannot set accepted or rejected
	if newStatus == store.DecisionStatusAccepted || newStatus == store.DecisionStatusRejected {
		return true, &approvalError{
			Error:   "decision_approval_required",
			Message: "This team requires human approval. Do not proceed with this decision until a human sets the status to 'accepted'.",
		}
	}

	// Agents can set running only if current status is accepted
	if newStatus == store.DecisionStatusRunning {
		currentStatus := h.getDecisionStatus(ctx, teamID, decisionID)
		if currentStatus != store.DecisionStatusAccepted {
			return true, &approvalError{
				Error:         "decision_not_accepted",
				Message:       "Decision must be accepted by a human before it can be set to running. Current status: " + currentStatus,
				CurrentStatus: currentStatus,
			}
		}
	}

	// Agents can set completed only if current status is running
	if newStatus == store.DecisionStatusCompleted {
		currentStatus := h.getDecisionStatus(ctx, teamID, decisionID)
		if currentStatus != store.DecisionStatusRunning {
			return true, &approvalError{
				Error:         "decision_not_running",
				Message:       "Decision must be running before it can be set to completed. Current status: " + currentStatus,
				CurrentStatus: currentStatus,
			}
		}
	}

	return false, nil
}

// getDecisionStatus returns the current status of a decision, or empty string if not found.
func (h *Handlers) getDecisionStatus(ctx context.Context, teamID, decisionID string) string {
	entries, _, err := h.teamStore.GetDecisions(ctx, teamID, "", "", 0)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if e.ID == decisionID {
			return e.Status
		}
	}
	return ""
}

// findDecision returns the DecisionEntry with the given ID, or nil if absent.
func (h *Handlers) findDecision(ctx context.Context, teamID, decisionID string) (*store.DecisionEntry, error) {
	entries, _, err := h.teamStore.GetDecisions(ctx, teamID, "", "", 0)
	if err != nil {
		return nil, err
	}
	for i := range entries {
		if entries[i].ID == decisionID {
			return &entries[i], nil
		}
	}
	return nil, nil
}

// decisionModificationsMaxRationale is the max UTF-8 byte length for
// DecisionModifications.Rationale.
const decisionModificationsMaxRationale = 4096

// validateDecisionModifications enforces the contract policy:
// reject entirely-empty payloads, reject empty-string entries in arrays,
// and bound rationale length. See
// docs/reference/decision-modifications-contract.md.
func validateDecisionModifications(m *store.DecisionModifications) error {
	if m == nil {
		return nil
	}
	for i, s := range m.ExcludedClauses {
		if strings.TrimSpace(s) == "" {
			return fmt.Errorf("excluded_clauses[%d] must be a non-empty string", i)
		}
	}
	for i, s := range m.Additions {
		if strings.TrimSpace(s) == "" {
			return fmt.Errorf("additions[%d] must be a non-empty string", i)
		}
	}
	if len(m.Rationale) > decisionModificationsMaxRationale {
		return fmt.Errorf("rationale exceeds %d bytes", decisionModificationsMaxRationale)
	}
	// Reject entirely-empty objects — operator should omit the field instead.
	if len(m.ExcludedClauses) == 0 && len(m.Additions) == 0 && strings.TrimSpace(m.Rationale) == "" {
		return fmt.Errorf("modifications must contain at least one of excluded_clauses, additions, or rationale")
	}
	return nil
}

// runAutoCreateInitiative invokes the swarm-manager initiative create
// endpoint for an accepted initiative-proposal decision. Always returns a
// non-nil outcome — success populates InitiativeRef; failure populates
// Error and the pre-filled manual-recovery commands (per d4=A + d8=C + d9=A).
func (h *Handlers) runAutoCreateInitiative(ctx context.Context, teamID string, d *store.DecisionEntry) *AutoCreateOutcome {
	meta := d.InitiativeMetadata
	target := resolvedTargetScenario(meta)
	desc := buildInitiativeDescription(d)
	createReq := initiativeCreateRequest{
		Name:        strings.TrimSpace(meta.Name),
		Title:       resolvedInitiativeTitle(d, meta),
		Description: desc,
		Priority:    meta.Priority,
		DependsOn:   meta.DependsOn,
	}

	client := h.swarmClient
	if client == nil {
		client = NewSwarmInitiativeClient(30 * time.Second)
	}

	createdName, err := client.Create(ctx, target, createReq)
	if err == nil {
		return &AutoCreateOutcome{
			Status:         store.AutoCreateStatusCreated,
			InitiativeRef:  fmt.Sprintf("%s/%s", target, createdName),
			TargetScenario: target,
			InitiativeName: createdName,
			Priority:       createReq.Priority,
		}
	}

	// Failure path: render the workaround commands. Materialise the
	// description body to a tmp file so the operator does not re-author it.
	descFile := ""
	if desc != "" {
		f, ferr := os.CreateTemp("", fmt.Sprintf("%s-initiative-description-*.md", d.ID))
		if ferr == nil {
			if _, werr := f.WriteString(desc); werr == nil {
				descFile = f.Name()
			}
			_ = f.Close()
		}
	}
	workaround := buildWorkaroundCommand(createReq, descFile, target)
	resolveCmd := buildResolveCommand(teamID, d.ID, fmt.Sprintf("%s/%s", target, createReq.Name))

	return &AutoCreateOutcome{
		Status:             store.AutoCreateStatusFailed,
		Error:              err.Error(),
		WorkaroundCommand:  workaround,
		ResolveCommand:     resolveCmd,
		DescriptionTmpFile: descFile,
		TargetScenario:     target,
		InitiativeName:     createReq.Name,
		Priority:           createReq.Priority,
	}
}

// writeDecisionFieldError emits a structured field-violation response.
func writeDecisionFieldError(w http.ResponseWriter, field, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error":   "invalid_field",
		"field":   field,
		"message": message,
	})
}

// DeleteDecisionHandler handles DELETE /teams/{id}/decisions/{decisionId}
func (h *Handlers) DeleteDecisionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	teamID := vars["id"]
	decisionID := vars["decisionId"]

	err := h.teamStore.DeleteDecision(r.Context(), teamID, decisionID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Knowledge Log handlers ---

// AddKnowledge handles POST /teams/{id}/knowledge.
//
// Validates the X-Vrooli-Attribution header (canon:
// docs/agent-system/RUNTIME_ATTRIBUTION.md), derives the display
// `Caller`, and persists the structured `Attribution` alongside the
// entry. Mutating writes without the header are rejected with HTTP
// 400; team_id mismatches between header and URL likewise return 400.
func (h *Handlers) AddKnowledge(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	teamID := vars["id"]

	info, err := parseAttributionHeader(r.Header.Get(attributionHeaderName))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := validateAttribution(info, teamID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var req AddKnowledgeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Topic) == "" {
		http.Error(w, "topic is required", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		http.Error(w, "content is required", http.StatusBadRequest)
		return
	}
	if len(req.CallerNote) > callerNoteMaxLen {
		http.Error(w, fmt.Sprintf("caller_note exceeds %d-character cap", callerNoteMaxLen), http.StatusBadRequest)
		return
	}

	entry := &store.KnowledgeEntry{
		ID:          fmt.Sprintf("knw-%s", generateID()),
		At:          time.Now().UTC().Format(time.RFC3339),
		Topic:       req.Topic,
		Content:     req.Content,
		Source:      req.Source,
		Supersedes:  req.Supersedes,
		Caller:      deriveCaller(info, teamID),
		CallerNote:  req.CallerNote,
		Attribution: info,
	}

	if err := h.teamStore.AppendKnowledge(r.Context(), teamID, entry); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(entry)
}

// GetKnowledge handles GET /teams/{id}/knowledge
func (h *Handlers) GetKnowledge(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	teamID := vars["id"]
	topicFilter := r.URL.Query().Get("topic")
	topicPrefix := r.URL.Query().Get("topic_prefix")
	if topicFilter != "" && topicPrefix != "" {
		http.Error(w, "topic and topic_prefix are mutually exclusive", http.StatusBadRequest)
		return
	}
	last := 20
	if v := r.URL.Query().Get("last"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			last = n
		}
	}

	entries, err := h.teamStore.GetKnowledge(r.Context(), teamID, topicFilter, topicPrefix, last)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if entries == nil {
		entries = []store.KnowledgeEntry{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(KnowledgeListResponse{
		TeamID:  teamID,
		Entries: entries,
	})
}

// UpdateKnowledgeHandler handles PATCH /teams/{id}/knowledge/{knowledgeId}
func (h *Handlers) UpdateKnowledgeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	teamID := vars["id"]
	knowledgeID := vars["knowledgeId"]

	var req UpdateKnowledgeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	err := h.teamStore.UpdateKnowledge(r.Context(), teamID, knowledgeID, func(k *store.KnowledgeEntry) {
		if req.Topic != nil && strings.TrimSpace(*req.Topic) != "" {
			k.Topic = *req.Topic
		}
		if req.Content != nil {
			k.Content = *req.Content
		}
		if req.Source != nil {
			k.Source = *req.Source
		}
		if req.Supersedes != nil {
			k.Supersedes = *req.Supersedes
		}
	})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch updated entry to return
	entries, err := h.teamStore.GetKnowledge(r.Context(), teamID, "", "", 0)
	if err != nil {
		http.Error(w, "knowledge updated but fetch failed", http.StatusInternalServerError)
		return
	}
	for _, e := range entries {
		if e.ID == knowledgeID {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(e)
			return
		}
	}
	http.Error(w, "knowledge updated but not found in response", http.StatusInternalServerError)
}

// DeleteKnowledgeHandler handles DELETE /teams/{id}/knowledge/{knowledgeId}
func (h *Handlers) DeleteKnowledgeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	teamID := vars["id"]
	knowledgeID := vars["knowledgeId"]

	err := h.teamStore.DeleteKnowledge(r.Context(), teamID, knowledgeID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GetRetention returns the effective retention config for a team.
func (h *Handlers) GetRetention(w http.ResponseWriter, r *http.Request) {
	teamID := mux.Vars(r)["id"]

	team, err := h.teamStore.Get(r.Context(), teamID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	retention := team.EffectiveRetention()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(retention)
}

// PruneSharedState triggers pruning of stale shared state for a team.
func (h *Handlers) PruneSharedState(w http.ResponseWriter, r *http.Request) {
	teamID := mux.Vars(r)["id"]

	result, err := h.teamStore.PruneSharedState(r.Context(), teamID)
	if err != nil {
		if strings.Contains(err.Error(), "no such file") {
			http.Error(w, fmt.Sprintf("team not found: %s", teamID), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// generateID creates a short unique identifier.
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// ClearTeamQueueRunning handles
// DELETE /teams/{id}/queue/running/{agentId}[?force=true]
// — removes a single running entry from the team's queue. By default refuses
// if the backing agent-manager run is still active; pass ?force=true to
// override (operator escape hatch for runs that agent-manager itself has
// lost track of).
func (h *Handlers) ClearTeamQueueRunning(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	teamID := vars["id"]
	agentID := vars["agentId"]

	if h.teamExecStore == nil {
		http.Error(w, "team execution store not configured", http.StatusServiceUnavailable)
		return
	}

	force := r.URL.Query().Get("force") == "true"
	if err := h.teamExecStore.ClearRunning(ctx, teamID, agentID, force); err != nil {
		switch {
		case err == ErrRunningEntryNotFound:
			http.Error(w, err.Error(), http.StatusNotFound)
		case IsRunningStillActive(err):
			http.Error(w, err.Error(), http.StatusConflict)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	status := h.teamExecStore.Status(teamID)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(status)
}
