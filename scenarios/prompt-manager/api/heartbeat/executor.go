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
	agentClient   AgentClient
	vrooliRoot    string
	promptBuilder *PromptBuilder
	runRegistry   *RunRegistry
	OnComplete    func(teamID, agentID string)
}

// NewExecutor creates a new heartbeat executor
func NewExecutor(
	teamStore *store.FileTeamStore,
	agentStore *store.FileAgentStore,
	agentClient AgentClient,
	vrooliRoot string,
	runRegistry *RunRegistry,
) *Executor {
	promptBuilder := NewPromptBuilder(teamStore, agentStore)
	return &Executor{
		teamStore:     teamStore,
		agentStore:    agentStore,
		agentClient:   agentClient,
		vrooliRoot:    vrooliRoot,
		promptBuilder: promptBuilder,
		runRegistry:   runRegistry,
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

	// Enforce team-level heartbeat gating before any prompt or config work.
	team, err := e.teamStore.Get(ctx, teamID)
	if err != nil {
		result.Error = fmt.Errorf("getting team: %w", err)
		result.Status = store.HeartbeatStatusFailed
		return result, result.Error
	}
	if err := validateTeamEnabled(team); err != nil {
		result.Error = fmt.Errorf("heartbeat blocked: %w", err)
		result.Status = store.HeartbeatStatusFailed
		return result, result.Error
	}

	// Build the prompt (branch on spawnMode for single-process teams)
	var prompt string
	if team.SpawnMode == "single-process" {
		prompt, err = e.promptBuilder.BuildTeamLeadPrompt(ctx, teamID, e.vrooliRoot)
	} else {
		prompt, err = e.BuildPrompt(ctx, teamID, agentID)
	}
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
	if config == nil {
		result.Error = fmt.Errorf("heartbeat config not found")
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

	// Guard: agent client must be configured
	if e.agentClient == nil {
		result.Error = fmt.Errorf("agent client is not configured")
		result.Status = store.HeartbeatStatusFailed
		return result, result.Error
	}

	// Create task in agent-manager
	task := &Task{
		Title:       fmt.Sprintf("Heartbeat: %s/%s", teamID, agentID),
		Description: prompt,
		ScopePath:   e.vrooliRoot,
		ProjectRoot: e.vrooliRoot,
	}

	createdTask, err := e.agentClient.CreateTask(ctx, task)
	if err != nil {
		result.Error = fmt.Errorf("creating task: %w", err)
		result.Status = store.HeartbeatStatusFailed
		e.updateConfigFailed(ctx, teamID, agentID, config, startedAt, result.Error.Error())
		return result, result.Error
	}

	// Create run — include Defaults so agent-manager can auto-create the
	// profile if EnsureProfile failed at startup (e.g. agent-manager wasn't
	// ready yet).
	runTag := fmt.Sprintf("heartbeat-%s-%s-%s", teamID, agentID, timestamp)
	runReq := &CreateRunRequest{
		TaskID: createdTask.ID,
		ProfileRef: &ProfileRef{
			ProfileKey: profileKey,
			Defaults:   BuildDefaultProfile(profileKey),
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

	// Wait for completion asynchronously with cancellable context
	waitCtx, waitCancel := context.WithCancel(context.Background())
	if e.runRegistry != nil {
		e.runRegistry.Register(teamID, agentID, run.ID, startedAt, waitCancel)
	}
	go func() {
		defer waitCancel()
		e.waitForCompletion(waitCtx, teamID, agentID, run.ID, startedAt, logPath)
	}()

	result.Status = store.HeartbeatStatusRunning
	return result, nil
}

// BuildContext constructs team/agent context for external consumption.
func (e *Executor) BuildContext(ctx context.Context, teamID, agentID string) (string, error) {
	if e.promptBuilder == nil {
		return "", fmt.Errorf("prompt builder is not configured")
	}
	return e.promptBuilder.BuildContext(ctx, PromptBuildRequest{TeamID: teamID, AgentID: agentID})
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

// BuildPromptStructured returns the prompt as structured sections.
func (e *Executor) BuildPromptStructured(ctx context.Context, teamID, agentID string) ([]PromptSection, error) {
	if e.promptBuilder == nil {
		return nil, fmt.Errorf("prompt builder is not configured")
	}
	return e.promptBuilder.BuildStructured(ctx, PromptBuildRequest{
		TeamID:  teamID,
		AgentID: agentID,
	})
}

// waitForCompletion polls for run completion and updates config
func (e *Executor) waitForCompletion(ctx context.Context, teamID, agentID, runID string, startedAt time.Time, logPath string) {
	if e.runRegistry != nil {
		defer e.runRegistry.Unregister(teamID, agentID)
	}

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()

	run, err := e.agentClient.WaitForRun(timeoutCtx, runID, 5*time.Second)

	endedAt := time.Now().UTC()

	// Use a background context for config updates since the parent ctx may be cancelled
	cfgCtx := context.Background()

	// Get current config
	config, configErr := e.teamStore.GetHeartbeatConfig(cfgCtx, teamID, agentID)
	if configErr != nil {
		return
	}
	if config == nil {
		return
	}

	if err != nil {
		// Check if the error is due to context cancellation (stop was requested)
		status := store.HeartbeatStatusFailed
		if ctx.Err() != nil {
			status = store.HeartbeatStatusCancelled
		}
		config.LastExecution = &store.HeartbeatExecResult{
			StartedAt: startedAt.Format(time.RFC3339),
			EndedAt:   endedAt.Format(time.RFC3339),
			Status:    status,
			RunID:     runID,
			LogPath:   logPath,
			Error:     err.Error(),
		}
	} else {
		status := store.HeartbeatStatusCompleted
		errMsg := ""
		if IsFailedStatus(run.Status) {
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

	_ = e.teamStore.SetHeartbeatConfig(cfgCtx, teamID, agentID, config)

	if e.OnComplete != nil {
		e.OnComplete(teamID, agentID)
	}
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
	if config == nil {
		return nil, fmt.Errorf("heartbeat config not found")
	}

	profileKey := "prompt-manager-heartbeat"
	if config.ProfileKey != "" {
		profileKey = config.ProfileKey
	}

	return e.Execute(ctx, teamID, agentID, profileKey)
}
