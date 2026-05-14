package heartbeat

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"prompt-manager/store"
	"prompt-manager/teamconfig"
	"strings"
	"time"

	"github.com/google/uuid"
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

// TeamExecStoreRegistrar is the subset of TeamExecutionStore that Executor
// needs to register a RunID for an in-flight running entry. Kept as an
// interface so tests can inject a fake without depending on the full store.
type TeamExecStoreRegistrar interface {
	SetRunningRunID(teamID, agentID, runID string)
}

// Executor handles the actual execution of heartbeats
type Executor struct {
	teamStore        *store.FileTeamStore
	agentStore       *store.FileAgentStore
	agentClient      AgentClient
	vrooliRoot       string
	promptBuilder    *PromptBuilder
	runRegistry      *RunRegistry
	handoffExtractor HandoffExtractor
	teamExecStore    TeamExecStoreRegistrar
	OnComplete       func(teamID, agentID string)
}

// SetTeamExecStore wires the team execution store so Execute can register
// the agent-manager RunID against the team's running entry. Called from
// main.go after both the executor and the store have been constructed
// (they have a circular dependency at construction time).
func (e *Executor) SetTeamExecStore(s TeamExecStoreRegistrar) {
	e.teamExecStore = s
}

// NewExecutor creates a new heartbeat executor
func NewExecutor(
	teamStore *store.FileTeamStore,
	agentStore *store.FileAgentStore,
	agentClient AgentClient,
	vrooliRoot string,
	runRegistry *RunRegistry,
	handoffExtractor HandoffExtractor,
) *Executor {
	if handoffExtractor == nil {
		handoffExtractor = NewSentinelExtractor()
	}
	promptBuilder := NewPromptBuilder(teamStore, agentStore)
	return &Executor{
		teamStore:        teamStore,
		agentStore:       agentStore,
		agentClient:      agentClient,
		vrooliRoot:       vrooliRoot,
		promptBuilder:    promptBuilder,
		runRegistry:      runRegistry,
		handoffExtractor: handoffExtractor,
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

	// Auto-prune stale shared state before building the prompt.
	if _, pruneErr := e.teamStore.PruneSharedState(ctx, teamID); pruneErr != nil {
		log.Printf("Warning: prune shared state for %s: %v", teamID, pruneErr)
	}

	// Resolve default profile key based on runtime mode when not explicitly set.
	if profileKey == "" {
		profileKey = DefaultProfileKeyForRuntimeMode(team.Runtime.Mode)
	}

	// Build the resolved profile and validate compatibility with runtime mode.
	resolvedProfile := BuildDefaultProfileForRuntimeMode(profileKey, team.Runtime.Mode)
	if err := validateProfileCompatibility(team, resolvedProfile); err != nil {
		result.Error = fmt.Errorf("profile mismatch: %w", err)
		result.Status = store.HeartbeatStatusFailed
		return result, result.Error
	}

	contract := team.Contract()

	// Build the prompt (single-process leader-led teams use Claude Code interop)
	var prompt string
	if teamconfig.UsesSingleProcessInterop(contract) {
		prompt, err = e.promptBuilder.BuildTeamLeadPrompt(ctx, teamID, agentID, e.vrooliRoot)
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
	attemptID := uuid.NewString()

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
		e.updateConfigFailed(ctx, teamID, agentID, config, startedAt, result.Error.Error(), attemptID, profileKey, "", "", "", "agent_client_missing")
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
		e.updateConfigFailed(ctx, teamID, agentID, config, startedAt, result.Error.Error(), attemptID, profileKey, "", "", "", "creating_task")
		return result, result.Error
	}

	// Create run — include Defaults so agent-manager can auto-create the
	// profile if EnsureProfile failed at startup (e.g. agent-manager wasn't
	// ready yet).
	//
	// Environment carries VROOLI_PROMPT_MANAGER_ATTRIBUTION so the spawned
	// agent's CLI inherits structured attribution and forwards it as the
	// X-Vrooli-Attribution header on every API write (canon:
	// docs/agent-system/RUNTIME_ATTRIBUTION.md § Env-var bridge).
	runTag := fmt.Sprintf("heartbeat-%s-%s-%s", teamID, agentID, timestamp)
	attribKey, attribValue := buildHeartbeatAttributionEnv(teamID, agentID)
	runReq := &CreateRunRequest{
		TaskID: createdTask.ID,
		ProfileRef: &ProfileRef{
			ProfileKey:     profileKey,
			Defaults:       resolvedProfile,
			UpdateExisting: true,
		},
		Tag: &runTag,
		Environment: map[string]string{
			attribKey: attribValue,
		},
	}

	run, err := e.agentClient.CreateRun(ctx, runReq)
	if err != nil {
		result.Error = fmt.Errorf("creating run: %w", err)
		result.Status = store.HeartbeatStatusFailed
		e.updateConfigFailed(ctx, teamID, agentID, config, startedAt, result.Error.Error(), attemptID, profileKey, createdTask.ID, "", runTag, "creating_run")
		return result, result.Error
	}

	result.RunID = run.ID
	config.LastExecution.RunID = run.ID
	_ = e.teamStore.SetHeartbeatConfig(ctx, teamID, agentID, config)
	if e.teamExecStore != nil {
		e.teamExecStore.SetRunningRunID(teamID, agentID, run.ID)
	}
	e.appendAttempt(context.Background(), &store.HeartbeatAttempt{
		ID:         attemptID,
		TeamID:     teamID,
		AgentID:    agentID,
		ProfileKey: profileKey,
		TaskID:     createdTask.ID,
		RunID:      run.ID,
		Tag:        runTag,
		Status:     store.HeartbeatStatusRunning,
		Phase:      "run_created",
		StartedAt:  startedAt.Format(time.RFC3339),
	})

	// Wait for completion asynchronously with cancellable context
	waitCtx, waitCancel := context.WithCancel(context.Background())
	if e.runRegistry != nil {
		e.runRegistry.Register(teamID, agentID, run.ID, startedAt, waitCancel)
	}
	go func() {
		defer waitCancel()
		e.waitForCompletion(waitCtx, teamID, agentID, run.ID, attemptID, profileKey, createdTask.ID, runTag, startedAt, logPath, config.EffectiveTimeout())
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
func (e *Executor) waitForCompletion(ctx context.Context, teamID, agentID, runID, attemptID, profileKey, taskID, tag string, startedAt time.Time, logPath string, timeout time.Duration) {
	if e.runRegistry != nil {
		defer e.runRegistry.Unregister(teamID, agentID)
	}

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
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
		category := classifyHeartbeatError(err.Error())
		e.appendAttempt(cfgCtx, &store.HeartbeatAttempt{
			ID:            attemptID,
			TeamID:        teamID,
			AgentID:       agentID,
			ProfileKey:    profileKey,
			TaskID:        taskID,
			RunID:         runID,
			Tag:           tag,
			Status:        status,
			Phase:         "waiting_for_completion",
			StartedAt:     startedAt.Format(time.RFC3339),
			EndedAt:       endedAt.Format(time.RFC3339),
			ErrorCategory: category,
			Error:         err.Error(),
			Recovery:      recoveryForHeartbeatError(category),
		})
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
		category := ""
		recovery := ""
		if errMsg != "" {
			category = classifyHeartbeatError(errMsg)
			recovery = recoveryForHeartbeatError(category)
		}
		e.appendAttempt(cfgCtx, &store.HeartbeatAttempt{
			ID:            attemptID,
			TeamID:        teamID,
			AgentID:       agentID,
			ProfileKey:    profileKey,
			TaskID:        taskID,
			RunID:         runID,
			Tag:           tag,
			Status:        status,
			Phase:         "run_terminal",
			StartedAt:     startedAt.Format(time.RFC3339),
			EndedAt:       endedAt.Format(time.RFC3339),
			ErrorCategory: category,
			Error:         errMsg,
			Recovery:      recovery,
		})
	}

	// Extract and store handoff (best-effort, non-blocking)
	e.extractAndStoreHandoff(cfgCtx, teamID, agentID, runID, endedAt)

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

// extractAndStoreHandoff fetches run events and extracts/stores the handoff (best-effort).
func (e *Executor) extractAndStoreHandoff(ctx context.Context, teamID, agentID, runID string, endedAt time.Time) {
	if e.handoffExtractor == nil || e.agentClient == nil {
		return
	}

	eventsJSON, err := e.agentClient.GetRunEvents(ctx, runID, -1, 0)
	if err != nil {
		log.Printf("Warning: failed to fetch run events for handoff extraction (run %s): %v", runID, err)
		return
	}

	content, err := e.handoffExtractor.Extract(ctx, eventsJSON)
	if err != nil {
		log.Printf("Warning: handoff extraction failed (run %s): %v", runID, err)
		return
	}

	if content == "" {
		return
	}

	// Store last handoff for prompt injection
	if err := e.teamStore.SetLastHandoff(ctx, teamID, agentID, content); err != nil {
		log.Printf("Warning: failed to store last handoff (run %s): %v", runID, err)
	}

	// Append to team history
	entry := &store.HandoffEntry{
		AgentID:   agentID,
		RunID:     runID,
		Timestamp: endedAt.Format(time.RFC3339),
		Content:   content,
	}
	if err := e.teamStore.AppendHandoffHistory(ctx, teamID, entry); err != nil {
		log.Printf("Warning: failed to append handoff history (run %s): %v", runID, err)
	}
}

// updateConfigFailed updates config with failed status
func (e *Executor) updateConfigFailed(ctx context.Context, teamID, agentID string, config *store.HeartbeatConfig, startedAt time.Time, errMsg, attemptID, profileKey, taskID, runID, tag, phase string) {
	endedAt := time.Now().UTC()
	config.LastExecution = &store.HeartbeatExecResult{
		StartedAt: startedAt.Format(time.RFC3339),
		EndedAt:   endedAt.Format(time.RFC3339),
		Status:    store.HeartbeatStatusFailed,
		Error:     errMsg,
	}
	_ = e.teamStore.SetHeartbeatConfig(ctx, teamID, agentID, config)
	category := classifyHeartbeatError(errMsg)
	e.appendAttempt(ctx, &store.HeartbeatAttempt{
		ID:            attemptID,
		TeamID:        teamID,
		AgentID:       agentID,
		ProfileKey:    profileKey,
		TaskID:        taskID,
		RunID:         runID,
		Tag:           tag,
		Status:        store.HeartbeatStatusFailed,
		Phase:         phase,
		StartedAt:     startedAt.Format(time.RFC3339),
		EndedAt:       endedAt.Format(time.RFC3339),
		ErrorCategory: category,
		Error:         errMsg,
		Recovery:      recoveryForHeartbeatError(category),
	})
}

func (e *Executor) appendAttempt(ctx context.Context, attempt *store.HeartbeatAttempt) {
	if e.teamStore == nil || attempt == nil || attempt.TeamID == "" {
		return
	}
	if attempt.ID == "" {
		attempt.ID = uuid.NewString()
	}
	if attempt.Phase == "" {
		attempt.Phase = "unknown"
	}
	if err := e.teamStore.AppendHeartbeatAttempt(ctx, attempt.TeamID, attempt); err != nil {
		log.Printf("Warning: failed to append heartbeat attempt for %s/%s: %v", attempt.TeamID, attempt.AgentID, err)
	}
}

func classifyHeartbeatError(message string) string {
	lower := strings.ToLower(message)
	switch {
	case strings.Contains(lower, "invalid json request body"), strings.Contains(lower, "validation error"):
		return "contract_validation"
	case strings.Contains(lower, "agent client is not configured"):
		return "configuration"
	case strings.Contains(lower, "connection refused"), strings.Contains(lower, "no such host"), strings.Contains(lower, "deadline exceeded"):
		return "dependency_unavailable"
	case strings.Contains(lower, "not found"):
		return "missing_dependency"
	case strings.Contains(lower, "cancel"):
		return "cancelled"
	default:
		return "agent_run_failed"
	}
}

func recoveryForHeartbeatError(category string) string {
	switch category {
	case "contract_validation":
		return "fix_integration_contract"
	case "configuration":
		return "configure_agent_manager_client"
	case "dependency_unavailable":
		return "retry_when_dependency_healthy"
	case "missing_dependency":
		return "reconcile_missing_resource"
	case "cancelled":
		return "operator_cancelled"
	default:
		return "inspect_run_or_retry"
	}
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

	// Use config's profile key if set; otherwise pass empty string so
	// Execute() resolves the default based on the team's runtime mode.
	return e.Execute(ctx, teamID, agentID, config.ProfileKey)
}
