package heartbeat

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"prompt-manager/store"

	"github.com/gorilla/mux"
)

// Handlers provides HTTP handlers for heartbeat operations
type Handlers struct {
	teamStore     *store.FileTeamStore
	relationStore store.RelationStore
	scheduler     *Scheduler
	executor      *Executor
}

// NewHandlers creates new heartbeat handlers
func NewHandlers(teamStore *store.FileTeamStore, relationStore store.RelationStore, scheduler *Scheduler, executor *Executor) *Handlers {
	return &Handlers{
		teamStore:     teamStore,
		relationStore: relationStore,
		scheduler:     scheduler,
		executor:      executor,
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

	// Get next execution time from scheduler
	if config.Enabled && h.scheduler != nil {
		if nextRun := h.scheduler.GetNextRun(config.TeamID, config.AgentID); nextRun != nil {
			resp.NextExecution = nextRun.Format(time.RFC3339)
		}
	}

	return resp
}
