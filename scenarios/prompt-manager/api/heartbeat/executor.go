package heartbeat

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"prompt-manager/store"
)

// ExecutionResult represents the result of a heartbeat execution
type ExecutionResult struct {
	TeamID    string
	AgentID   string
	RunID     string
	Status    string
	StartedAt time.Time
	EndedAt   *time.Time
	LogPath   string
	Error     error
}

// Executor handles the actual execution of heartbeats
type Executor struct {
	teamStore     *store.FileTeamStore
	agentStore    *store.FileAgentStore
	agentClient   *AgentManagerClient
	vrooliRoot    string
	promptBuilder *PromptBuilder
}

// NewExecutor creates a new heartbeat executor
func NewExecutor(
	teamStore *store.FileTeamStore,
	agentStore *store.FileAgentStore,
	agentClient *AgentManagerClient,
	vrooliRoot string,
) *Executor {
	promptBuilder := NewPromptBuilder(teamStore, agentStore)
	return &Executor{
		teamStore:     teamStore,
		agentStore:    agentStore,
		agentClient:   agentClient,
		vrooliRoot:    vrooliRoot,
		promptBuilder: promptBuilder,
	}
}

// Execute runs a heartbeat for a team member
func (e *Executor) Execute(ctx context.Context, teamID, agentID, profileKey string) (*ExecutionResult, error) {
	startedAt := time.Now().UTC()
	result := &ExecutionResult{
		TeamID:    teamID,
		AgentID:   agentID,
		StartedAt: startedAt,
	}

	// Build the prompt
	prompt, err := e.BuildPrompt(ctx, teamID, agentID)
	if err != nil {
		result.Error = fmt.Errorf("building prompt: %w", err)
		result.Status = store.HeartbeatStatusFailed
		return result, result.Error
	}

	// Update config with running status
	config, err := e.teamStore.GetHeartbeatConfig(ctx, teamID, agentID)
	if err != nil {
		result.Error = fmt.Errorf("getting heartbeat config: %w", err)
		result.Status = store.HeartbeatStatusFailed
		return result, result.Error
	}

	// Generate log path
	timestamp := startedAt.Format("2006-01-02T15-04-05Z")
	logPath := e.teamStore.GetMemberLogPath(teamID, agentID, timestamp)
	result.LogPath = logPath

	// Update config to running
	config.LastExecution = &store.HeartbeatExecResult{
		StartedAt: startedAt.Format(time.RFC3339),
		Status:    store.HeartbeatStatusRunning,
		LogPath:   logPath,
	}
	if err := e.teamStore.SetHeartbeatConfig(ctx, teamID, agentID, config); err != nil {
		result.Error = fmt.Errorf("updating config to running: %w", err)
		result.Status = store.HeartbeatStatusFailed
		return result, result.Error
	}

	// Create task in agent-manager
	task := &Task{
		Prompt:      prompt,
		WorkingDir:  e.vrooliRoot,
		Description: fmt.Sprintf("Heartbeat: %s/%s", teamID, agentID),
	}

	createdTask, err := e.agentClient.CreateTask(ctx, task)
	if err != nil {
		result.Error = fmt.Errorf("creating task: %w", err)
		result.Status = store.HeartbeatStatusFailed
		e.updateConfigFailed(ctx, teamID, agentID, config, startedAt, result.Error.Error())
		return result, result.Error
	}

	// Create run
	runTag := fmt.Sprintf("heartbeat-%s-%s-%s", teamID, agentID, timestamp)
	runReq := &CreateRunRequest{
		TaskID: createdTask.ID,
		ProfileRef: &ProfileRef{
			ProfileKey: profileKey,
		},
		Tag:     &runTag,
		RunMode: "RUN_MODE_IN_PLACE",
	}

	run, err := e.agentClient.CreateRun(ctx, runReq)
	if err != nil {
		result.Error = fmt.Errorf("creating run: %w", err)
		result.Status = store.HeartbeatStatusFailed
		e.updateConfigFailed(ctx, teamID, agentID, config, startedAt, result.Error.Error())
		return result, result.Error
	}

	result.RunID = run.ID
	config.LastExecution.RunID = run.ID
	_ = e.teamStore.SetHeartbeatConfig(ctx, teamID, agentID, config)

	// Wait for completion asynchronously
	go e.waitForCompletion(context.Background(), teamID, agentID, run.ID, startedAt, logPath)

	result.Status = store.HeartbeatStatusRunning
	return result, nil
}

// BuildPrompt constructs the full prompt for heartbeat execution.
func (e *Executor) BuildPrompt(ctx context.Context, teamID, agentID string) (string, error) {
	if e.promptBuilder == nil {
		return "", fmt.Errorf("prompt builder is not configured")
	}
	return e.promptBuilder.Build(ctx, PromptBuildRequest{
		TeamID:  teamID,
		AgentID: agentID,
	})
}

// waitForCompletion polls for run completion and updates config
func (e *Executor) waitForCompletion(ctx context.Context, teamID, agentID, runID string, startedAt time.Time, logPath string) {
	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()

	run, err := e.agentClient.WaitForRun(timeoutCtx, runID, 5*time.Second)

	endedAt := time.Now().UTC()

	// Get current config
	config, configErr := e.teamStore.GetHeartbeatConfig(ctx, teamID, agentID)
	if configErr != nil {
		return
	}

	if err != nil {
		config.LastExecution = &store.HeartbeatExecResult{
			StartedAt: startedAt.Format(time.RFC3339),
			EndedAt:   endedAt.Format(time.RFC3339),
			Status:    store.HeartbeatStatusFailed,
			RunID:     runID,
			LogPath:   logPath,
			Error:     err.Error(),
		}
	} else {
		status := store.HeartbeatStatusCompleted
		errMsg := ""
		if run.Status == "RUN_STATUS_FAILED" || run.Status == "failed" {
			status = store.HeartbeatStatusFailed
			errMsg = run.Error
		}

		config.LastExecution = &store.HeartbeatExecResult{
			StartedAt: startedAt.Format(time.RFC3339),
			EndedAt:   endedAt.Format(time.RFC3339),
			Status:    status,
			RunID:     runID,
			LogPath:   logPath,
			Error:     errMsg,
		}
	}

	// Write log file
	logContent := fmt.Sprintf("Heartbeat execution for %s/%s\n", teamID, agentID)
	logContent += fmt.Sprintf("Started: %s\n", startedAt.Format(time.RFC3339))
	logContent += fmt.Sprintf("Ended: %s\n", endedAt.Format(time.RFC3339))
	logContent += fmt.Sprintf("Run ID: %s\n", runID)
	logContent += fmt.Sprintf("Status: %s\n", config.LastExecution.Status)
	if config.LastExecution.Error != "" {
		logContent += fmt.Sprintf("Error: %s\n", config.LastExecution.Error)
	}

	// Ensure logs directory exists and write log
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err == nil {
		_ = os.WriteFile(logPath, []byte(logContent), 0o644)
	}

	_ = e.teamStore.SetHeartbeatConfig(ctx, teamID, agentID, config)
}

// updateConfigFailed updates config with failed status
func (e *Executor) updateConfigFailed(ctx context.Context, teamID, agentID string, config *store.HeartbeatConfig, startedAt time.Time, errMsg string) {
	endedAt := time.Now().UTC()
	config.LastExecution = &store.HeartbeatExecResult{
		StartedAt: startedAt.Format(time.RFC3339),
		EndedAt:   endedAt.Format(time.RFC3339),
		Status:    store.HeartbeatStatusFailed,
		Error:     errMsg,
	}
	_ = e.teamStore.SetHeartbeatConfig(ctx, teamID, agentID, config)
}

// TriggerManual manually triggers a heartbeat execution
func (e *Executor) TriggerManual(ctx context.Context, teamID, agentID string) (*ExecutionResult, error) {
	// Get heartbeat config to get profile key
	config, err := e.teamStore.GetHeartbeatConfig(ctx, teamID, agentID)
	if err != nil {
		return nil, fmt.Errorf("getting heartbeat config: %w", err)
	}

	profileKey := "prompt-manager-heartbeat"
	if config != nil && config.ProfileKey != "" {
		profileKey = config.ProfileKey
	}

	return e.Execute(ctx, teamID, agentID, profileKey)
}
