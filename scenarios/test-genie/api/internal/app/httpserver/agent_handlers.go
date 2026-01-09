package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/mux"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"

	"test-genie/agentmanager"
)

// =============================================================================
// MODELS - Agent model listing from agent-manager
// =============================================================================

type agentModel struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Provider    string `json:"provider"`
	DisplayName string `json:"display_name"`
	Source      string `json:"source"`
	Description string `json:"description"`
}

func (s *Server) handleListAgentModels(w http.ResponseWriter, r *http.Request) {
	if !s.agentService.IsAvailable(r.Context()) {
		s.writeError(w, http.StatusServiceUnavailable, "agent-manager is not available")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()

	resp, err := s.agentService.GetProfileWithModels(ctx)
	if err != nil {
		s.log("list agent models failed", map[string]interface{}{"error": err.Error()})
		s.writeError(w, http.StatusBadGateway, fmt.Sprintf("failed to load agent models: %s", err.Error()))
		return
	}

	models := make([]agentModel, 0, len(resp.GetAvailableModels()))
	for _, model := range resp.GetAvailableModels() {
		id := strings.TrimSpace(model.GetId())
		if id == "" {
			continue
		}

		label := strings.TrimSpace(model.GetLabel())
		if label == "" {
			label = id
		}

		models = append(models, agentModel{
			ID:          id,
			Name:        label,
			DisplayName: label,
			Provider:    strings.TrimSpace(model.GetProvider()),
			Source:      "agent-manager",
			Description: strings.TrimSpace(model.GetDescription()),
		})
	}

	sort.Slice(models, func(i, j int) bool {
		if models[i].Provider == models[j].Provider {
			return models[i].ID < models[j].ID
		}
		return models[i].Provider < models[j].Provider
	})

	payload := map[string]interface{}{
		"items": models,
		"count": len(models),
	}
	s.writeJSON(w, http.StatusOK, payload)
}

// =============================================================================
// SPAWN - Agent spawning via agent-manager
// =============================================================================

type agentSpawnRequest struct {
	Prompts        []string `json:"prompts"`
	Model          string   `json:"model"`
	Concurrency    int      `json:"concurrency"`
	MaxTurns       int      `json:"maxTurns"`
	TimeoutSeconds int      `json:"timeoutSeconds"`
	Scenario       string   `json:"scenario"`
	Scope          []string `json:"scope"`
	Phases         []string `json:"phases"`
	NetworkEnabled bool     `json:"networkEnabled"`
}

type agentSpawnResult struct {
	PromptIndex int    `json:"promptIndex"`
	AgentID     string `json:"agentId,omitempty"`
	RunID       string `json:"runId,omitempty"`
	Status      string `json:"status"`
	Error       string `json:"error,omitempty"`
}

type agentSpawnResponse struct {
	BatchID         string             `json:"batchId,omitempty"`
	Items           []agentSpawnResult `json:"items"`
	Count           int                `json:"count"`
	Capped          bool               `json:"capped"`
	ValidationError string             `json:"validationError,omitempty"`
}

// handleSpawnAgents spawns one or more agents to execute prompts via agent-manager.
func (s *Server) handleSpawnAgents(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var payload agentSpawnRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	// Check if agent-manager is available
	if !s.agentService.IsAvailable(r.Context()) {
		s.writeError(w, http.StatusServiceUnavailable, "agent-manager is not available")
		return
	}

	// Validate required fields
	prompts := sanitizePrompts(payload.Prompts)
	if len(prompts) == 0 {
		s.writeError(w, http.StatusBadRequest, "at least one prompt is required")
		return
	}

	model := strings.TrimSpace(payload.Model)
	if model == "" {
		s.writeError(w, http.StatusBadRequest, "model is required")
		return
	}

	scenario := strings.TrimSpace(payload.Scenario)
	if scenario == "" {
		s.writeError(w, http.StatusBadRequest, "scenario is required for agent spawning")
		return
	}

	// Cap prompts at 12
	const maxPrompts = 12
	capped := false
	if len(prompts) > maxPrompts {
		prompts = prompts[:maxPrompts]
		capped = true
	}

	// Generate and prepend preamble to each prompt
	repoRoot := os.Getenv("VROOLI_ROOT")
	if repoRoot == "" {
		repoRoot = filepath.Join(os.Getenv("HOME"), "Vrooli")
	}

	preambleCfg := agentmanager.PreambleConfig{
		Scenario:       scenario,
		Scope:          payload.Scope,
		RepoRoot:       repoRoot,
		MaxFiles:       agentmanager.DefaultMaxFilesChanged,
		MaxBytes:       agentmanager.DefaultMaxBytesWritten,
		NetworkEnabled: payload.NetworkEnabled,
	}
	preamble := agentmanager.GenerateSafetyPreamble(preambleCfg)

	// Build prompt configs with preamble prepended
	promptConfigs := make([]agentmanager.PromptConfig, len(prompts))
	for i, prompt := range prompts {
		fullPrompt := prompt
		if preamble != "" && !strings.Contains(prompt, "## SECURITY CONSTRAINTS") {
			fullPrompt = preamble + prompt
		}
		promptConfigs[i] = agentmanager.PromptConfig{
			Text:   fullPrompt,
			Phases: payload.Phases,
		}
	}

	// Set timeout
	timeout := time.Duration(payload.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 15 * time.Minute // Default 15 minutes for test generation
	}

	// Spawn batch via agent-manager
	batchReq := agentmanager.BatchSpawnRequest{
		Scenario:    scenario,
		Scope:       payload.Scope,
		Prompts:     promptConfigs,
		Model:       model,
		Concurrency: payload.Concurrency,
		MaxTurns:    payload.MaxTurns,
		Timeout:     timeout,
	}

	result, err := s.agentService.SpawnBatch(r.Context(), batchReq)
	if err != nil {
		s.log("spawn batch failed", map[string]interface{}{
			"error":    err.Error(),
			"scenario": scenario,
		})
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to spawn agents: %s", err.Error()))
		return
	}

	// Convert results to API format
	items := make([]agentSpawnResult, len(result.Runs))
	for i, run := range result.Runs {
		items[i] = agentSpawnResult{
			PromptIndex: run.PromptIndex,
			AgentID:     run.Tag,  // Use tag as agent ID for UI compatibility
			RunID:       run.RunID,
			Status:      run.Status,
			Error:       run.Error,
		}
	}

	response := agentSpawnResponse{
		BatchID: result.BatchID,
		Items:   items,
		Count:   len(items),
		Capped:  capped,
	}

	s.log("agents spawned", map[string]interface{}{
		"batchId":  result.BatchID,
		"scenario": scenario,
		"count":    len(items),
	})

	s.writeJSON(w, http.StatusOK, response)
}

func sanitizePrompts(prompts []string) []string {
	result := make([]string, 0, len(prompts))
	for _, p := range prompts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// =============================================================================
// LIST/GET - Agent listing and details via agent-manager
// =============================================================================

// handleListActiveAgents returns all active test-genie agents from agent-manager.
func (s *Server) handleListActiveAgents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !s.agentService.IsAvailable(ctx) {
		s.writeError(w, http.StatusServiceUnavailable, "agent-manager is not available")
		return
	}

	includeAll := r.URL.Query().Get("all") == "true"

	var runs []*domainpb.Run
	var err error

	if includeAll {
		runs, err = s.agentService.ListAllRuns(ctx)
	} else {
		runs, err = s.agentService.ListActiveRuns(ctx)
	}

	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list agents: %s", err.Error()))
		return
	}

	// Convert to API format
	activeAgents := make([]*ActiveAgent, 0, len(runs))
	for _, run := range runs {
		activeAgents = append(activeAgents, runToActiveAgent(run))
	}

	payload := map[string]interface{}{
		"items": activeAgents,
		"count": len(activeAgents),
	}
	s.writeJSON(w, http.StatusOK, payload)
}

// handleGetAgent returns details for a specific agent by tag or run ID.
func (s *Server) handleGetAgent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	agentID := vars["id"]

	if !s.agentService.IsAvailable(ctx) {
		s.writeError(w, http.StatusServiceUnavailable, "agent-manager is not available")
		return
	}

	// Try to get by tag first (which is what we use as agent ID)
	run, err := s.agentService.GetRunByTag(ctx, agentID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get agent: %s", err.Error()))
		return
	}

	// If not found by tag, try by run ID
	if run == nil {
		run, err = s.agentService.GetRun(ctx, agentID)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get agent: %s", err.Error()))
			return
		}
	}

	if run == nil {
		s.writeError(w, http.StatusNotFound, fmt.Sprintf("agent not found: %s", agentID))
		return
	}

	s.writeJSON(w, http.StatusOK, runToActiveAgent(run))
}

// =============================================================================
// STOP - Agent stopping via agent-manager
// =============================================================================

// handleStopAgent stops a running agent.
func (s *Server) handleStopAgent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	agentID := vars["id"]

	if !s.agentService.IsAvailable(ctx) {
		s.writeError(w, http.StatusServiceUnavailable, "agent-manager is not available")
		return
	}

	// Try to get the run first to get the actual run ID
	run, _ := s.agentService.GetRunByTag(ctx, agentID)
	runID := agentID
	if run != nil {
		runID = run.Id
	}

	if err := s.agentService.StopRun(ctx, runID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.writeError(w, http.StatusNotFound, err.Error())
			return
		}
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.log("agent stopped", map[string]interface{}{"agentId": agentID, "runId": runID})

	// Get updated status
	run, _ = s.agentService.GetRun(ctx, runID)

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "agent stopped",
		"agent":   runToActiveAgent(run),
	})
}

// handleStopAllAgents stops all running test-genie agents.
func (s *Server) handleStopAllAgents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !s.agentService.IsAvailable(ctx) {
		s.writeError(w, http.StatusServiceUnavailable, "agent-manager is not available")
		return
	}

	result, err := s.agentService.StopAllRuns(ctx)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to stop agents: %s", err.Error()))
		return
	}

	s.log("all agents stopped", map[string]interface{}{
		"stoppedCount": result.StoppedCount,
		"failedCount":  result.FailedCount,
	})

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":      "all agents stopped",
		"stoppedCount": result.StoppedCount,
		"failedCount":  result.FailedCount,
		"errors":       result.Errors,
	})
}

// =============================================================================
// UTILITY - Security info and containment status
// =============================================================================

// handleGetBlockedCommands returns command validation info for reference.
func (s *Server) handleGetBlockedCommands(w http.ResponseWriter, r *http.Request) {
	allowedCommands := agentmanager.GetDefaultSafeTools()

	response := map[string]interface{}{
		"allowedTools":  allowedCommands,
		"securityModel": "agent-manager profiles",
		"note":          "Security constraints are now managed by agent-manager profiles. Bash command restrictions are specified in the agent profile configuration.",
	}
	s.writeJSON(w, http.StatusOK, response)
}

// handleGetAgentManagerStatus returns the status of the agent-manager connection.
func (s *Server) handleGetAgentManagerStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	available := s.agentService.IsAvailable(ctx)
	enabled := s.agentService.IsEnabled()

	response := map[string]interface{}{
		"enabled":   enabled,
		"available": available,
	}

	if available {
		url, err := s.agentService.ResolveURL(ctx)
		if err == nil {
			response["url"] = url
		}
		response["profileId"] = s.agentService.GetProfileID()
	}

	s.writeJSON(w, http.StatusOK, response)
}

// handleGetAgentManagerWSUrl returns the WebSocket URL for agent-manager.
func (s *Server) handleGetAgentManagerWSUrl(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if !s.agentService.IsAvailable(ctx) {
		s.writeError(w, http.StatusServiceUnavailable, "agent-manager is not available")
		return
	}

	baseURL, err := s.agentService.ResolveURL(ctx)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to resolve agent-manager URL: %s", err.Error()))
		return
	}

	// Convert HTTP URL to WebSocket URL
	wsURL := strings.Replace(baseURL, "http://", "ws://", 1)
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
	wsURL = wsURL + "/api/v1/ws"

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"url": wsURL,
	})
}

// =============================================================================
// TYPE CONVERSIONS
// =============================================================================

// runToActiveAgent converts an agent-manager Run to the ActiveAgent format for API responses.
func runToActiveAgent(run *domainpb.Run) *ActiveAgent {
	if run == nil {
		return nil
	}

	agent := &ActiveAgent{
		ID:       run.GetTag(),
		RunID:    run.Id,
		Status:   AgentStatus(agentmanager.MapRunStatus(run.Status)),
	}

	// Extract scenario from tag if possible (format: test-genie-{batchId}-{index})
	if run.GetTag() != "" {
		parts := strings.Split(run.GetTag(), "-")
		if len(parts) >= 2 && parts[0] == "test" && parts[1] == "genie" {
			// Tag format is test-genie-{batchId}-{index}
			agent.ID = run.GetTag()
		}
	}

	if run.StartedAt != nil {
		agent.StartedAt = run.StartedAt.AsTime()
	}

	if run.EndedAt != nil {
		agent.CompletedAt = run.EndedAt.AsTime()
	}

	if run.Summary != nil {
		agent.Output = run.Summary.Description
		agent.TokensUsed = run.Summary.TokensUsed
		agent.CostEstimate = run.Summary.CostEstimate
	}

	if run.ErrorMsg != "" {
		agent.Error = run.ErrorMsg
	}

	return agent
}
