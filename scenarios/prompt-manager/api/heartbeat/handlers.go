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
	"strconv"
	"strings"
	"time"

	"prompt-manager/store"

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
}

// NewHandlers creates new heartbeat handlers
func NewHandlers(
	teamStore *store.FileTeamStore,
	agentStore *store.FileAgentStore,
	relationStore store.RelationStore,
	scheduler *Scheduler,
	executor *Executor,
	runRegistry *RunRegistry,
	agentClient AgentClient,
	teamExecStore *TeamExecutionStore,
) *Handlers {
	return &Handlers{
		teamStore:     teamStore,
		agentStore:    agentStore,
		relationStore: relationStore,
		scheduler:     scheduler,
		executor:      executor,
		runRegistry:   runRegistry,
		agentClient:   agentClient,
		teamExecStore: teamExecStore,
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

	if err := h.teamStore.SetHeartbeatConfig(ctx, teamID, agentID, config); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Team not found", http.StatusNotFound)
			return
		}
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
	wasEnabled := config.Enabled
	if req.Enabled != nil {
		config.Enabled = *req.Enabled
	}

	if err := h.teamStore.SetHeartbeatConfig(ctx, teamID, agentID, config); err != nil {
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

	// Route through team execution store for serialized execution
	if h.teamExecStore != nil {
		// Resolve profile key from heartbeat config
		profileKey := "prompt-manager-heartbeat"
		config, err := h.teamStore.GetHeartbeatConfig(ctx, teamID, agentID)
		if err == nil && config != nil && config.ProfileKey != "" {
			profileKey = config.ProfileKey
		}

		enqueueResult, err := h.teamExecStore.Enqueue(ctx, teamID, agentID, profileKey)
		if err != nil {
			if IsMemberAlreadyQueued(err) {
				http.Error(w, err.Error(), http.StatusConflict)
				return
			}
			if IsTeamDisabled(err) {
				http.Error(w, "Team is disabled; enable the team to run heartbeats", http.StatusConflict)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(enqueueResult)
		return
	}

	// Fallback: direct execution
	result, err := h.executor.TriggerManual(ctx, teamID, agentID)
	if err != nil {
		if IsTeamDisabled(err) {
			http.Error(w, "Team is disabled; enable the team to run heartbeats", http.StatusConflict)
			return
		}
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := TriggerHeartbeatResponse{
		TeamID:  result.TeamID,
		AgentID: result.AgentID,
		RunID:   result.RunID,
		Status:  result.Status,
		LogPath: result.LogPath,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
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

	logPath := h.teamStore.GetMemberLogPath(teamID, agentID, strings.TrimSuffix(logID, ".log"))
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

// TriggerTeam handles POST /teams/{id}/trigger - triggers all or lead heartbeat based on spawnMode.
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

	if team.SpawnMode == "single-process" {
		// Find team lead from org chart (agent with no manager)
		leadAgentID := h.findTeamLead(ctx, teamID)
		if leadAgentID == "" {
			http.Error(w, "no team lead found in org chart", http.StatusBadRequest)
			return
		}

		// Route through team execution store if available
		if h.teamExecStore != nil {
			profileKey := "prompt-manager-heartbeat"
			config, cfgErr := h.teamStore.GetHeartbeatConfig(ctx, teamID, leadAgentID)
			if cfgErr == nil && config != nil && config.ProfileKey != "" {
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

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(TriggerTeamResponse{
				TeamID:    teamID,
				SpawnMode: "single-process",
				Triggers:  []TriggerHeartbeatResponse{{TeamID: teamID, AgentID: leadAgentID, Status: enqueueResult.Status}},
			})
			return
		}

		result, err := h.executor.TriggerManual(ctx, teamID, leadAgentID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(TriggerTeamResponse{
			TeamID:    teamID,
			SpawnMode: "single-process",
			Triggers:  []TriggerHeartbeatResponse{{TeamID: result.TeamID, AgentID: result.AgentID, RunID: result.RunID, Status: result.Status, LogPath: result.LogPath}},
		})
		return
	}

	// multi-process (default): trigger all member heartbeats
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
			profileKey := "prompt-manager-heartbeat"
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(TriggerTeamResponse{
		TeamID:    teamID,
		SpawnMode: "multi-process",
		Triggers:  triggers,
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

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"run": run})
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

// GetTeamExecutionStatus handles GET /teams/{id}/execution-status - returns team execution queue state.
func (h *Handlers) GetTeamExecutionStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	teamID := vars["id"]

	// Verify team exists
	if _, err := h.teamStore.Get(ctx, teamID); err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}

	if h.teamExecStore == nil {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(TeamExecutionStatus{
			TeamID: teamID,
			State:  "idle",
			Queue:  []string{},
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

// findTeamLead returns the agent ID of the team lead (agent with no manager in org chart).
// Falls back to the first member if no org chart is defined.
func (h *Handlers) findTeamLead(ctx context.Context, teamID string) string {
	org, err := h.teamStore.GetOrgChart(ctx, teamID)
	if err == nil && len(org.Edges) > 0 {
		// Collect all agents that are reports (have a manager)
		reports := make(map[string]bool, len(org.Edges))
		managers := make(map[string]bool, len(org.Edges))
		for _, edge := range org.Edges {
			reports[edge.ReportAgentID] = true
			managers[edge.ManagerAgentID] = true
		}
		// Find a manager who is not a report to anyone
		for managerID := range managers {
			if !reports[managerID] {
				return managerID
			}
		}
	}

	// Fallback: first member
	if h.relationStore != nil {
		members, err := h.relationStore.ListTeamMembers(ctx, teamID)
		if err == nil && len(members) > 0 {
			return members[0].AgentID
		}
	}
	return ""
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
		TeamID:     config.TeamID,
		AgentID:    config.AgentID,
		Enabled:    config.Enabled,
		Schedule:   config.Schedule,
		ProfileKey: config.ProfileKey,
		CreatedAt:  config.CreatedAt,
		UpdatedAt:  config.UpdatedAt,
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
