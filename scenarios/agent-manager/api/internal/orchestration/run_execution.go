// This file coordinates agent execution after a run has been created.
package orchestration

import (
	"agent-manager/internal/adapters/runner"
	agentconfig "agent-manager/internal/config"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/interactive"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/orchestration/phases"
	"agent-manager/internal/orchestration/spawn"
	"agent-manager/internal/promptmanager"
	"agent-manager/internal/runstate"
	"context"
	"fmt"
	"github.com/google/uuid"
	"strings"
	"sync"
	"time"
)

// executeRun handles the actual agent execution (runs in background).
// This delegates to RunExecutor for the actual work. `started` is the
// spawn dispatcher's slot-release callback, fired the moment the run
// reaches RunStatusRunning so the next queued run can begin its
// codex-bootstrap window.
func (o *Orchestrator) executeRun(ctx context.Context, run *domain.Run, task *domain.Task, profile *domain.AgentProfile, prompt string, systemPrompt string, existingSandboxWorkDir string, attachments []runner.Attachment, customEnv map[string]string, started spawn.StartedFn) {
	// Interactive runs take the parallel execution path: agent-manager launches
	// the real interactive CLI in a web-console session and tails its transcript
	// to completion, instead of owning a codec stdout pipe. Selected by the run's
	// ExecutionMode (design §1).
	if run.ExecutionMode.Normalized() == domain.ExecutionModeInteractive {
		o.executeInteractiveRun(ctx, run, task, interactiveInitialPrompt(systemPrompt, prompt), started)
		return
	}

	// Codec-pipe runners have no native objective channel, so the engine-owned
	// completion contract is delivered as exactly one prompt suffix. Interactive
	// runs decide inside executeInteractiveRun between /goal and this suffix.
	prompt = initialPromptWithUntil(run, prompt)

	runStateRoot, err := o.resolveRunStateRoot(ctx)
	if err != nil {
		started()
		phases.FailWithError(ctx, phases.FailWithErrorInput{Deps: phases.Deps{Runs: o.runs, Events: o.events, Broadcaster: o.broadcaster}, Run: run, Err: fmt.Errorf("resolve run state root: %w", err)})
		return
	}
	executor := NewRunExecutor(
		o.runs,
		o.runners,
		o.sandbox,
		o.events,
		run,
		task,
		profile,
		prompt,
		systemPrompt,
	)
	executor.WithClock(o.now)
	executor.WithIdentitySecret(o.identitySecret)
	executor.WithTerminalObserver(o.observeFiniteSerialTerminal)
	executor.WithRunStateRoot(runStateRoot)
	if source, ok := o.promptClient.(promptmanager.SourceClient); ok {
		executor.WithSkillSource(source)
	}
	executor.WithRunStateWriteObserver(func() { o.recordRunStateWrite(ctx) })
	executor.WithStructuredResultResolver(o.structuredResults)
	// Apply orchestration-settings overrides to executor levers when a store
	// is wired. Defaults come from config.DefaultLevers().
	if levers, ok := o.executorLevers(); ok {
		executor.WithLevers(levers)
	}
	// Configure executor with checkpoint repository if available
	if o.checkpoints != nil {
		executor.WithCheckpointRepository(o.checkpoints)
	}
	if o.healthStore != nil {
		executor.WithModelHealthReporter(newHealthMarkerAdapter(o.healthStore, run.ID.String()))
	}
	if run.SandboxID != nil {
		workDir := existingSandboxWorkDir
		if workDir == "" && o.sandbox != nil {
			if resolved, err := o.sandbox.GetWorkspacePath(ctx, *run.SandboxID); err == nil {
				workDir = resolved
			}
		}
		executor.WithExistingSandbox(*run.SandboxID, workDir)
	}
	// Configure executor with broadcaster for real-time WebSocket updates
	if o.broadcaster != nil {
		executor.WithBroadcaster(o.broadcaster)
	}
	if o.workspaceSandbox != nil {
		executor.WithWorkspaceSandboxEnsurer(o.workspaceSandbox)
	}
	if len(attachments) > 0 {
		executor.WithAttachments(attachments)
	}
	if len(customEnv) > 0 {
		executor.WithCustomEnvironment(customEnv)
	}
	if len(o.identitySecret) > 0 {
		executor.WithIdentitySecret(o.identitySecret)
	}
	executor.WithOnRunning(started)
	executor.Execute(ctx)
	// Completion-driven advance: a workflow run just reached terminal (a parked
	// run keeps its attempt open — the wake leg nudges when it later completes).
	if !executor.parked {
		o.nudgeWorkflowForRun(run.ID)
	}
}

// interactiveInitialPrompt reconstructs the single-channel prompt to type into
// an interactive CLI. The codec-pipe path splits a task into a system prompt
// (instructions) and a user message (context + question); an interactive CLI
// launched raw has no separate system channel, so both halves are recombined
// into one prompt. When there is no system prompt (the common no-attachment
// case) the user message already carries the full task.
func interactiveInitialPrompt(systemPrompt, userMessage string) string {
	systemPrompt = strings.TrimSpace(systemPrompt)
	userMessage = strings.TrimSpace(userMessage)
	switch {
	case systemPrompt == "":
		return userMessage
	case userMessage == "":
		return systemPrompt
	default:
		return systemPrompt + "\n\n" + userMessage
	}
}

// initialPromptWithUntil keeps the completion test in the agent's task channel
// for both substrates. It is deliberately delivered as prompt text rather
// than a runner-specific CLI flag: interactive TUIs have no stable flag seam,
// and codec-pipe runners must receive the same contract as their TUI peers.
func initialPromptWithUntil(run *domain.Run, prompt string) string {
	if run == nil || run.ResolvedConfig == nil || strings.TrimSpace(run.ResolvedConfig.Until) == "" {
		return prompt
	}
	until := strings.TrimSpace(run.ResolvedConfig.Until)
	return strings.TrimSpace(prompt) + "\n\nCompletion contract (engine-owned): stop only when this test is satisfied:\n" + until
}

func nativeObjectiveFor(run *domain.Run, caps runner.Capabilities) string {
	if run == nil || run.ResolvedConfig == nil || strings.TrimSpace(run.ResolvedConfig.Until) == "" {
		return ""
	}
	objective := strings.TrimSpace(run.ResolvedConfig.Until)
	sandbox := string(run.InteractiveSandboxMode())
	for _, capability := range caps.SpawnCapabilities {
		if capability.ExecutionMode != string(domain.ExecutionModeInteractive) || !capability.NativeObjective {
			continue
		}
		for _, mode := range capability.SandboxModes {
			if mode == sandbox {
				return boundedNativeObjective(objective)
			}
		}
	}
	return ""
}

const (
	// Claude Code rejects native /goal conditions longer than 4000 characters.
	// Keep the full engine-owned contract in the task prompt and use this short
	// delegation when that contract is too large for the harness field.
	maxNativeObjectiveCharacters = 4000
	boundedNativeObjectiveText   = "Follow the complete engine-owned completion contract included in the task prompt exactly. Do not stop until it is satisfied; return blocked, abstained, or an operator-decision approval request when required."
	// nativeGoalPointer replaces the full completion contract in the task prompt
	// when the harness installs the goal natively via /goal. The finish line then
	// reaches the agent exactly once, through /goal.
	nativeGoalPointer = "\n\nYour goal is installed with /goal; the finish line is its text."
)

func boundedNativeObjective(objective string) string {
	if len([]rune(objective)) <= maxNativeObjectiveCharacters {
		return objective
	}
	return boundedNativeObjectiveText
}

// executeInteractiveRun drives an interactive run to completion via the
// interactive.Coordinator: it launches the real interactive CLI in a web-console
// session, tails the agent-owned transcript, and finalizes the run on the
// terminal marker (design §1). It is the parallel path to the codec-pipe
// RunExecutor and leaves the Continue/Stop seam (Substrate.Stop, SendText) for
// Phase 5.
func (o *Orchestrator) executeInteractiveRun(ctx context.Context, run *domain.Run, task *domain.Task, initialPrompt string, started spawn.StartedFn) {
	// The spawn slot must be released even if we fail before reaching Running.
	releaseOnce := sync.Once{}
	release := func() {
		if started != nil {
			releaseOnce.Do(started)
		}
	}
	defer release()

	if o.interactiveSessions == nil {
		o.failInteractiveRun(ctx, run, "interactive execution mode is not configured (no web-console session controller wired)")
		return
	}
	if run.ResolvedConfig == nil {
		o.failInteractiveRun(ctx, run, "interactive run has no resolved config")
		return
	}
	if !interactive.SupportsInteractive(run.ResolvedConfig.RunnerType) {
		o.failInteractiveRun(ctx, run, fmt.Sprintf("runner %q is not supported in interactive mode", run.ResolvedConfig.RunnerType))
		return
	}
	// Recheck at launch, including retained rows admitted by older versions.
	// Reading/recovering a running session does not invoke this launch gate.
	if err := validateExecutionContainment(run.ExecutionMode, run.InteractiveSandboxMode(), run.ResolvedConfig); err != nil {
		o.failInteractiveRun(ctx, run, err.Error())
		return
	}

	runStateRoot, err := o.resolveRunStateRoot(ctx)
	if err != nil {
		o.failInteractiveRun(ctx, run, "resolve interactive run state: "+err.Error())
		return
	}

	var workDir string
	if run.RunMode == domain.RunModeSandboxed {
		setup, err := phases.SetupWorkspace(ctx, phases.SetupWorkspaceInput{
			Deps: phases.Deps{Runs: o.runs, Events: o.events, Broadcaster: o.broadcaster, Levers: o.runLevers(), WorkspaceSandbox: o.workspaceSandbox},
			Run:  run, Task: task, Sandbox: o.sandbox, RunStateRoot: runStateRoot,
		})
		if err != nil {
			o.failInteractiveRun(ctx, run, fmt.Sprintf("prepare protected interactive tracking workspace: %v", err))
			return
		}
		run.SandboxID = setup.SandboxID
		workDir = setup.WorkDir
	} else {
		var err error
		workDir, err = phases.UseInPlaceWorkspace(task)
		if err != nil {
			o.failInteractiveRun(ctx, run, fmt.Sprintf("resolve interactive working directory: %v", err))
			return
		}
	}
	runDir, err := runstate.RunDir(runStateRoot, run.ID)
	if err != nil {
		o.failInteractiveRun(ctx, run, "resolve interactive run state: "+err.Error())
		return
	}

	coord := interactive.NewCoordinator(interactive.CoordinatorDeps{
		Substrate:   interactive.NewSubstrate(o.interactiveSessions, interactive.RegistryLaunchInfo(o.runners)),
		Tailer:      interactive.NewTailer(interactive.RegistryParser(o.runners)),
		Sessions:    o.interactiveSessions,
		Runs:        o.runs,
		Broadcaster: o.broadcaster,
		NewSink:     o.interactiveEventSink,
		Result:      o.persistedResultBuilder,
	})

	// Register the live coordinator so StopRun can cancel it deterministically and
	// wait for it to exit before finalizing (no late tail Update can resurrect a
	// stopped run). The context is cancellable via the registry; unregister +
	// signal done when Execute returns.
	runCtx, driver := o.interactiveDrivers.register(ctx, run.ID)
	defer o.interactiveDrivers.finish(run.ID, driver)

	// Interactive runs carry the same context deadline the codec-pipe path
	// derives from the resolved config. Without it a goal run whose harness
	// never reaches a terminal idles until the process dies (B4). Reaching the
	// deadline surfaces as context.DeadlineExceeded; the coordinator records it
	// as the timeout interruption Phase 3 types.
	if run.ResolvedConfig.Timeout > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(runCtx, run.ResolvedConfig.Timeout)
		defer cancel()
	}

	selectedRunner, _ := o.runners.Get(run.ResolvedConfig.RunnerType)
	var nativeObjective string
	if selectedRunner != nil {
		nativeObjective = nativeObjectiveFor(run, selectedRunner.Capabilities())
	}
	// Keep the launch seam defensive: profiles, adapters, or a future resolver
	// may provide the objective through another path. The complete engine-owned
	// contract remains in the task prompt; the harness field must stay within
	// Claude Code's native limit regardless of where the value came from.
	nativeObjective = boundedNativeObjective(strings.TrimSpace(nativeObjective))
	// One owner for the finish line. When the harness installs the goal natively,
	// the prompt only points at /goal; otherwise the full contract travels as a
	// single prompt suffix.
	deliveredPrompt := initialPrompt
	if nativeObjective == "" {
		deliveredPrompt = initialPromptWithUntil(run, initialPrompt)
	} else {
		deliveredPrompt = strings.TrimSpace(initialPrompt) + nativeGoalPointer
	}
	if err := coord.Execute(runCtx, run, interactive.LaunchParams{
		RunID:           run.ID,
		RunnerType:      run.ResolvedConfig.RunnerType,
		Tag:             run.GetTag(),
		WorkingDir:      workDir,
		RunDir:          runDir,
		DisplayLabel:    run.GetTag(),
		Prompt:          deliveredPrompt,
		NativeObjective: nativeObjective,
		Model:           run.ResolvedConfig.Model,
		Effort:          run.ResolvedConfig.Effort,
		Config:          run.ResolvedConfig,
	}, release); err != nil {
		obs.Component("interactive").Warn("interactive run finalize failed",
			obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
	}
	// Tracking mode uses the same sandbox provenance/lifecycle contract as
	// codec-pipe execution. The coordinator owns terminal detection; once it
	// returns, finalize the attributed sandbox before exposing the terminal run.
	o.finalizeSandboxForTerminalRun(ctx, run)
}

// Interactive execution uses a host web-console terminal, not the protected
// launcher. Capability declarations alone cannot enforce this boundary: direct
// requests and previously persisted rows also reach this path.
func validateExecutionContainment(mode domain.ExecutionMode, sandboxMode domain.SandboxMode, cfg *domain.RunConfig) error {
	if mode != domain.ExecutionModeInteractive {
		return nil
	}
	required := sandboxMode.Effective() == domain.SandboxModeProtected
	if cfg != nil {
		required = required || cfg.RequireEffectContainment || (cfg.SandboxConfig != nil && cfg.SandboxConfig.WritePolicy != nil)
	}
	if required {
		return domain.NewValidationErrorWithHint("executionMode", "interactive execution cannot enforce protected containment", "Use codec_pipe with protected mode; tracking/off interactive sessions do not provide containment")
	}
	return nil
}

// interactiveEventSink builds the per-run event sink interactive tail events are
// emitted into, mirroring the codec-pipe executor's sink selection.
func (o *Orchestrator) interactiveEventSink(runID uuid.UUID) runner.EventSink {
	switch {
	case o.events != nil && o.broadcaster != nil:
		return &broadcastingEventSink{store: o.events, runID: runID, broadcaster: o.broadcaster}
	case o.events != nil:
		return &eventStoreAdapter{store: o.events, runID: runID}
	default:
		return &noOpEventSink{}
	}
}

// failInteractiveRun marks an interactive run failed with an explicit reason
// (used for pre-launch misconfiguration/validation failures).
func (o *Orchestrator) failInteractiveRun(ctx context.Context, run *domain.Run, reason string) {
	now := o.now()
	run.Status = domain.RunStatusFailed
	run.Phase = domain.RunPhaseCompleted
	run.ErrorMsg = reason
	run.EndedAt = &now
	run.UpdatedAt = now
	if err := o.runs.Update(ctx, run); err != nil {
		obs.Component("interactive").Warn("failed to persist interactive run failure",
			obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
	}
	if o.broadcaster != nil {
		o.broadcaster.BroadcastRunStatus(run)
	}
}

// executorLevers folds the runtime OrchestrationSettings store (when wired)
// onto the static config.DefaultLevers() to produce the per-run lever set
// the executor reads. Returns ok=false when no override store is configured —
// callers fall through to the executor's built-in defaults.
//
// Only fields that the OrchestrationSettings model actually exposes are
// overridden. Other levers (recovery polls, scanner buffers, diagnostics)
// stay at compile-time defaults until they are surfaced as runtime knobs.
func (o *Orchestrator) executorLevers() (agentconfig.Levers, bool) {
	if o.orchestrationSettings == nil {
		return agentconfig.Levers{}, false
	}
	return o.runLevers(), true
}

func (o *Orchestrator) runLevers() agentconfig.Levers {
	levers := agentconfig.DefaultLevers()
	if o.orchestrationSettings == nil {
		return levers
	}
	s := o.orchestrationSettings.Get()
	if s.RunExecution.RunTimeoutMinutes > 0 {
		levers.Execution.DefaultTimeout = time.Duration(s.RunExecution.RunTimeoutMinutes) * time.Minute
	}
	if s.HealthDetection.HeartbeatIntervalSeconds > 0 {
		levers.Heartbeat.RunHeartbeatInterval = time.Duration(s.HealthDetection.HeartbeatIntervalSeconds) * time.Second
	}
	if s.HealthDetection.StaleThresholdSeconds > 0 {
		levers.Heartbeat.StaleThreshold = time.Duration(s.HealthDetection.StaleThresholdSeconds) * time.Second
	}
	return levers
}

// -----------------------------------------------------------------------------
// Run Resumption Operations (Interruption Resilience)
// -----------------------------------------------------------------------------

// ResumeRun attempts to resume a stalled or interrupted run from its last checkpoint.
// This enables safe recovery from crashes, network issues, or intentional pauses.
//
// IDEMPOTENCY: Resuming an already-running or completed run is a no-op.
// TEMPORAL FLOW: Validates the run hasn't exceeded its stale threshold.
// PROGRESS CONTINUITY: Uses checkpoints to skip completed phases.
func (o *Orchestrator) ResumeRun(ctx context.Context, id uuid.UUID) (*domain.Run, error) {
	run, err := o.GetRun(ctx, id)
	if err != nil {
		return nil, err
	}
	if run.ExecutionMode.Normalized() == domain.ExecutionModeImported {
		return nil, importedRunLifecycleError("resume")
	}

	if err := o.checkEffortContinuation(ctx, run); err != nil {
		return nil, domain.RefuseBeforeEffects(err)
	}
	// Validate resumability using domain decision helper
	if !run.IsResumable() {
		return nil, domain.NewStateError("Run", string(run.Status), "resume",
			fmt.Sprintf("run in %s state cannot be resumed", run.Status))
	}
	// A resting review resumption is new admission. Take this fence before
	// reading resource policy so maintenance refusal remains authoritative and
	// does not require a retained run configuration or resource CLI.
	var releaseAdmission func()
	if run.Status != domain.RunStatusPending && run.Status != domain.RunStatusStarting && run.Status != domain.RunStatusRunning && run.Status != domain.RunStatusParked {
		releaseAdmission, err = o.admitMaintenance(ctx)
		if err != nil {
			return nil, err
		}
		defer releaseAdmission()
	}
	if err := validateExecutionModel(run.ResolvedConfig); err != nil {
		return nil, err
	}
	currentExclusions, err := currentModelExclusions(ctx, run.ResolvedConfig, o.rolePolicy, o.roleResolver)
	if err != nil {
		return nil, err
	}
	// Get the last checkpoint
	var checkpoint *domain.RunCheckpoint
	if o.checkpoints != nil {
		checkpoint, err = o.checkpoints.Get(ctx, id)
		if err != nil {
			return nil, err
		}
	}

	// If no checkpoint, start from the beginning
	if checkpoint == nil {
		checkpoint = domain.NewCheckpoint(id, domain.RunPhaseQueued)
	}

	// Get associated entities
	task, err := o.GetTask(ctx, run.TaskID)
	if err != nil {
		return nil, err
	}

	// Get profile if available (may be nil for inline config runs)
	var profile *domain.AgentProfile
	if run.AgentProfileID != nil {
		profile, err = o.GetProfile(ctx, *run.AgentProfileID)
		if err != nil {
			return nil, err
		}
	}

	// Update status to running
	run.Status = domain.RunStatusRunning
	run.UpdatedAt = o.now()
	if err := o.runs.Update(ctx, run); err != nil {
		return nil, err
	}

	// Resume goes through the same dispatcher as initial spawn —
	// per contract decision 2 in SEAMS.md, no goroutine spawn outside
	// spawn.Dispatcher.Enqueue.
	if err := o.dispatcher.Enqueue(&spawn.Job{
		RunID:      run.ID,
		RunMode:    run.RunMode,
		RunnerType: runnerTypeOrEmpty(run),
		Sink:       o.dispatcherSink(run.ID),
		Fn: func(started spawn.StartedFn) {
			defer obs.RecoverToFailure("run resumption dispatch", func(failure obs.PanicFailure) {
				o.recoverPanickedRun(run, failure)
			})
			o.resumeRun(context.WithoutCancel(ctx), run, task, profile, checkpoint, currentExclusions, started)
		},
		OnPanic: func(failure obs.PanicFailure) {
			o.recoverPanickedRun(run, failure)
		},
	}); err != nil {
		// Revert so the run stays resumable instead of stranded as a
		// running row with no process (until the stale sweep reaps it).
		run.Status = domain.RunStatusPending
		run.UpdatedAt = o.now()
		if revertErr := o.runs.Update(ctx, run); revertErr != nil {
			obs.Component("orchestrator").Error("failed to revert run status after enqueue failure", obs.KeyRunID, run.ID.String(), obs.KeyError, revertErr.Error())
		}
		return nil, err
	}

	return o.attachRunActions(ctx, run), nil
}

// resumeRun handles the actual agent resumption (runs in background).
// `started` is the spawn dispatcher's slot-release callback.
func (o *Orchestrator) resumeRun(ctx context.Context, run *domain.Run, task *domain.Task, profile *domain.AgentProfile, checkpoint *domain.RunCheckpoint, currentExclusions map[domain.RunnerType][]string, started spawn.StartedFn) {
	runStateRoot, err := o.resolveRunStateRoot(ctx)
	if err != nil {
		started()
		phases.FailWithError(ctx, phases.FailWithErrorInput{Deps: phases.Deps{Runs: o.runs, Events: o.events, Broadcaster: o.broadcaster}, Run: run, Err: fmt.Errorf("resolve run state root: %w", err)})
		return
	}
	executor := NewRunExecutor(
		o.runs,
		o.runners,
		o.sandbox,
		o.events,
		run,
		task,
		profile,
		"", // No new prompt for resume
		"", // No system prompt for resume (session persists instructions)
	)
	executor.WithClock(o.now)
	executor.WithIdentitySecret(o.identitySecret)
	executor.WithTerminalObserver(o.observeFiniteSerialTerminal)
	executor.WithRunStateRoot(runStateRoot)
	executor.WithRunStateWriteObserver(func() { o.recordRunStateWrite(ctx) })
	executor.WithStructuredResultResolver(o.structuredResults)
	executor.WithCurrentModelExclusions(currentExclusions)
	executor.WithCurrentModelAdmission(func(admissionCtx context.Context, candidate domain.ExecutionCandidate, model string) (bool, error) {
		return o.currentCandidateAllowed(admissionCtx, run.ResolvedConfig, candidate, model)
	})
	// Apply orchestration-settings overrides to executor levers when a store
	// is wired. Defaults come from config.DefaultLevers().
	if levers, ok := o.executorLevers(); ok {
		executor.WithLevers(levers)
	}

	// Configure for resumption
	if o.checkpoints != nil {
		executor.WithCheckpointRepository(o.checkpoints)
	}
	// Configure executor with broadcaster for real-time WebSocket updates
	if o.broadcaster != nil {
		executor.WithBroadcaster(o.broadcaster)
	}
	if o.workspaceSandbox != nil {
		executor.WithWorkspaceSandboxEnsurer(o.workspaceSandbox)
	}
	if len(o.identitySecret) > 0 {
		executor.WithIdentitySecret(o.identitySecret)
	}
	executor.WithResumeFrom(checkpoint)
	executor.WithOnRunning(started)

	executor.Execute(ctx)
	// Completion-driven advance for a workflow run recovered/resumed after a
	// restart between run-terminal and the original nudge.
	if !executor.parked {
		o.nudgeWorkflowForRun(run.ID)
	}
}
