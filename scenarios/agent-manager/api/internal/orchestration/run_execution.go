// This file coordinates agent execution after a run has been created.
package orchestration

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/domain"
	"agent-manager/internal/health"
	"agent-manager/internal/identity"
	"agent-manager/internal/orchestration/interactive"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/orchestration/phases"
	"agent-manager/internal/orchestration/spawn"
	"agent-manager/internal/repository"
	"agent-manager/internal/runstate"

	agentconfig "agent-manager/internal/config"

	"github.com/google/uuid"
)

func (o *Orchestrator) ContinueRun(ctx context.Context, req ContinueRunRequest) (*domain.Run, error) {
	if req.IdempotencyKey != "" && o.idempotency != nil {
		existing, err := o.idempotency.Check(ctx, req.IdempotencyKey)
		if err != nil {
			return nil, err
		}
		if existing != nil && existing.Status == domain.IdempotencyStatusComplete && existing.EntityID != nil {
			return o.GetRun(ctx, *existing.EntityID)
		}
		if existing != nil && existing.Status == domain.IdempotencyStatusPending {
			return nil, domain.NewStateError("Run", "continuing", "continue", "a continuation with this idempotency key is already in progress")
		}
	}
	// Validate message
	if strings.TrimSpace(req.Message) == "" {
		return nil, domain.NewValidationError("message", "message is required")
	}

	// Get the run
	run, err := o.GetRun(ctx, req.RunID)
	if err != nil {
		return nil, err
	}

	if allowed, reason := domain.CanContinueRun(run); !allowed {
		return nil, domain.NewStateError("Run", string(run.Status), "continue", reason)
	}
	if req.IdempotencyKey != "" && o.idempotency != nil {
		if _, err := o.idempotency.Reserve(ctx, req.IdempotencyKey, time.Hour); err != nil {
			return nil, domain.NewStateError("Run", "continuing", "continue", "a continuation with this idempotency key is already in progress")
		}
	}

	// Interactive runs continue by typing the follow-up into the still-live
	// web-console session (never a process respawn) and reattaching a tailer to
	// drive the new turn to completion — see continueInteractiveRun.
	if run.ExecutionMode.Normalized() == domain.ExecutionModeInteractive {
		if req.MaxTurns != nil || req.Timeout != nil || req.ResultSpec != nil {
			o.markIdempotencyFailed(ctx, req.IdempotencyKey)
			return nil, domain.NewValidationError("continuationOverrides", "interactive continuation does not support per-turn workflow overrides")
		}
		continued, err := o.continueInteractiveRun(ctx, run, req.Message, req.AttachmentIDs)
		if err != nil {
			o.markIdempotencyFailed(ctx, req.IdempotencyKey)
			return nil, err
		}
		o.markIdempotencyComplete(ctx, req.IdempotencyKey, continued.ID, "Run")
		return continued, nil
	}
	continued, err := o.resumeConversation(ctx, run, req.Message, req.AttachmentIDs, "Continuation requested", continuationOverrides{MaxTurns: req.MaxTurns, Timeout: req.Timeout, ResultSpec: req.ResultSpec})
	if err != nil {
		o.markIdempotencyFailed(ctx, req.IdempotencyKey)
		return nil, err
	}
	o.markIdempotencyComplete(ctx, req.IdempotencyKey, continued.ID, "Run")
	return continued, nil
}

// resumeConversation drives a single session-resume turn shared by both
// operator-driven continuation (ContinueRun) and waiter-driven wake (WakeRun):
// it resolves the runner, re-acquires the checkpointed sandbox, transitions the
// run back to running (resetting the heartbeat in the same transition — the
// faefb9cb54 invariant), re-assembles the full process env + a freshly
// regenerated identity token (Phase 0 assembler), and spawns executeContinuation
// with `message` injected as the next user turn. Centralising it keeps continue
// and wake from ever diverging on env/identity/heartbeat handling. Callers are
// responsible for the precondition gate (CanContinueRun for continue, the parked
// guard for wake) before calling this.
type continuationOverrides struct {
	MaxTurns   *int
	Timeout    *time.Duration
	ResultSpec *domain.ResultSpec
}

func (o *Orchestrator) resumeConversation(ctx context.Context, run *domain.Run, message string, attachmentIDs []string, reason string, overrides continuationOverrides) (*domain.Run, error) {
	// The immutable resolved config is the sole execution authority.
	var runnerType domain.RunnerType
	if run.ResolvedConfig != nil {
		runnerType = run.ResolvedConfig.RunnerType
	}

	if runnerType == "" {
		return nil, domain.NewStateError("Run", string(run.Status), "continue",
			"cannot determine runner type for this run")
	}

	// Get the runner
	if o.runners == nil {
		return nil, domain.NewConfigMissingError("runners", "runner registry not configured", nil)
	}

	r, err := o.runners.Get(runnerType)
	if err != nil {
		return nil, err
	}

	// Check runner supports continuation
	caps := r.Capabilities()
	if !caps.SupportsContinuation {
		return nil, runner.ErrContinuationNotSupported
	}

	task, err := o.GetTask(ctx, run.TaskID)
	if err != nil {
		return nil, err
	}

	// Profile is best-effort: it only supplies ProfileKey to the regenerated
	// identity token (a continuation can still run without it).
	var profile *domain.AgentProfile
	if run.AgentProfileID != nil {
		if p, perr := o.GetProfile(ctx, *run.AgentProfileID); perr == nil {
			profile = p
		}
	}

	workDir, err := o.prepareContinuationSandbox(ctx, run, task)
	if err != nil {
		return nil, err
	}

	now := o.now()
	run, err = o.applyRunStatusTransition(ctx, RunStatusTransitionInput{
		Run:           run,
		NewStatus:     domain.RunStatusRunning,
		Phase:         domain.RunPhaseExecuting,
		Reason:        reason,
		LastHeartbeat: &now,
	})
	if err != nil {
		return nil, err
	}

	// Emit user message event (with attachment metadata if present, resolved later)
	// Note: We defer emitting until after attachment resolution so we can include URLs.
	emitUserMessage := func(attachments []runner.Attachment) {
		if o.events == nil {
			return
		}
		var attInfo []domain.MessageAttachmentInfo
		if o.storage != nil {
			for _, att := range attachments {
				meta, err := o.storage.Get(ctx, att.ID)
				if err == nil {
					attInfo = append(attInfo, domain.MessageAttachmentInfo{
						ID:          meta.ID,
						FileName:    meta.FileName,
						ContentType: meta.ContentType,
						URL:         o.storage.GetServingURL(meta.StoragePath),
					})
				}
			}
		}
		var userEvent *domain.RunEvent
		if len(attInfo) > 0 {
			userEvent = domain.NewMessageEventWithAttachments(run.ID, "user", message, attInfo)
		} else {
			userEvent = domain.NewMessageEvent(run.ID, "user", message)
		}
		if err := o.appendAndBroadcastEvents(ctx, run.ID, userEvent); err != nil {
			obs.Component("orchestrator").Warn("failed to append continuation user message", obs.KeyRunID, run.ID.String(), "eventType", "message", obs.KeyError, err.Error())
		}
	}

	eventSink := o.runEventSink(run.ID)

	// Resolve attachments
	var attachments []runner.Attachment
	if len(attachmentIDs) > 0 && o.storage != nil {
		metas, err := o.storage.GetMultiple(ctx, attachmentIDs)
		if err != nil {
			// Log but continue without attachments
			obs.Component("orchestrator").Warn("failed to resolve continuation attachments", obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
		}
		for _, meta := range metas {
			attachments = append(attachments, runner.Attachment{
				ID:          meta.ID,
				FileName:    meta.FileName,
				ContentType: meta.ContentType,
				FilePath:    o.storage.GetFilePath(meta.StoragePath),
			})
		}
	}

	// Emit user message event now that attachments are resolved
	emitUserMessage(attachments)

	transcriptCfg, cleanupTranscript, err := o.prepareRunTranscript(ctx, run, workDir)
	if err != nil {
		return nil, err
	}

	// Re-assemble the full process env for this turn synchronously (before the
	// background goroutine spawns) so the run mutation from identity
	// regeneration happens on this goroutine — the same one that returns
	// attachRunActions(run) below — and never races the executeContinuation
	// goroutine. The original Execute call injected custom env + sandbox routing
	// + an identity token; a continuation that left ContinueRequest.Environment
	// nil silently dropped all three (the latent bug this fixes). We:
	//   1. regenerate the identity token (the plaintext is never stored, only
	//      the hash, so the original is unrecoverable — and a long-parked run
	//      could otherwise outlive its 24h TTL). GenerateIdentityToken
	//      re-persists the hash so /identity/verify keeps working.
	//   2. re-derive VROOLI_SANDBOX_* from the live (resumed) sandbox.
	//   3. re-inject the persisted custom env.
	// Both Execute and this path build env through phases.AssembleRunEnv so they
	// can never diverge again.
	continueEnv, err := o.assembleContinuationEnv(ctx, run, task, profile, workDir)
	if err != nil {
		return nil, err
	}

	// Hand the background turn its OWN *Run value. executeContinuation mutates
	// the run in place (status via applyRunStatusTransition, SessionID,
	// heartbeat) on a separate goroutine, while this goroutine returns
	// attachRunActions(run) below — sharing the same pointer is a data race
	// (the deferred -race failure in
	// TestContinueRun_ProtectedSandboxCarriesLauncherInputsAndLifecycleEvents).
	// A shallow copy is sufficient: the racing fields are scalars / freshly
	// reassigned pointers, and the persisted DB row remains the single source of
	// truth, so the background copy and the returned snapshot converge on reload.
	runForExec := *run
	if run.ResolvedConfig != nil {
		resolved := *run.ResolvedConfig
		if overrides.MaxTurns != nil {
			resolved.MaxTurns = *overrides.MaxTurns
		}
		if overrides.Timeout != nil {
			resolved.Timeout = *overrides.Timeout
		}
		if overrides.ResultSpec != nil {
			resolved.ResultSpec = overrides.ResultSpec
		}
		runForExec.ResolvedConfig = &resolved
	}

	// Execute continuation asynchronously
	go o.executeContinuation(context.WithoutCancel(ctx), &runForExec, r, eventSink, message, workDir, attachments, continueEnv, transcriptCfg, cleanupTranscript)

	return o.attachRunActions(ctx, run), nil
}

// assembleContinuationEnv regenerates the run's identity token and builds the
// full process env (custom + sandbox + identity) for a continuation/wake turn.
// Called synchronously on the request goroutine so the identity-hash persist
// does not race the background executeContinuation goroutine.
func (o *Orchestrator) assembleContinuationEnv(ctx context.Context, run *domain.Run, task *domain.Task, profile *domain.AgentProfile, workDir string) (map[string]string, error) {
	scopePath := ""
	if task != nil {
		scopePath = task.ScopePath
	}
	identityToken := ""
	if len(o.identitySecret) > 0 {
		identityToken = phases.GenerateIdentityToken(ctx, phases.GenerateIdentityTokenInput{
			Deps:    phases.Deps{Runs: o.runs, Events: o.events, Broadcaster: o.broadcaster},
			Run:     run,
			Profile: profile,
			Task:    task,
			Secret:  o.identitySecret,
		})
	}
	env := phases.AssembleRunEnv(phases.AssembleRunEnvInput{
		Custom:        run.CustomEnv,
		RunMode:       run.RunMode,
		SandboxID:     run.SandboxID,
		WorkDir:       workDir,
		ScopePath:     scopePath,
		IdentityToken: identityToken,
	})
	if run.ResolvedConfig == nil {
		return env, nil
	}
	runStateRoot, err := o.resolveRunStateRoot(ctx)
	if err != nil {
		return nil, err
	}
	sessionEnv, err := PrepareCodecSessionHome(runStateRoot, run.ID, run.ResolvedConfig.RunnerType)
	if err != nil {
		return nil, err
	}
	for key, value := range sessionEnv {
		if env == nil {
			env = make(map[string]string)
		}
		env[key] = value
	}
	return env, nil
}

func (o *Orchestrator) prepareContinuationSandbox(ctx context.Context, run *domain.Run, task *domain.Task) (string, error) {
	if run == nil {
		return "", domain.NewValidationError("run", "run is required")
	}
	if run.SandboxID == nil {
		if task != nil && task.ProjectRoot != "" {
			return task.ProjectRoot, nil
		}
		return "", nil
	}
	if o.sandbox == nil {
		return "", domain.NewConfigMissingError("sandbox", "provider not configured", nil)
	}

	sb, err := o.sandbox.Get(ctx, *run.SandboxID)
	if err != nil {
		return "", err
	}
	switch sb.Status {
	case sandbox.SandboxStatusCheckpointed:
		sb, err = o.sandbox.Resume(ctx, sb.ID)
		if err != nil {
			return "", err
		}
	case sandbox.SandboxStatusStopped:
		if err := o.sandbox.Start(ctx, sb.ID); err != nil {
			return "", err
		}
		sb, err = o.sandbox.Get(ctx, sb.ID)
		if err != nil {
			return "", err
		}
	case sandbox.SandboxStatusActive:
	case sandbox.SandboxStatusDeleted:
		// Terminal retention deliberately deletes successful sandboxes. A codec
		// session still remains continuable because its home is run-scoped, so
		// provision a fresh workspace rather than weakening the delete policy.
		out, createErr := phases.SetupWorkspace(ctx, phases.SetupWorkspaceInput{
			Deps: phases.Deps{
				Runs:             o.runs,
				Events:           o.events,
				Broadcaster:      o.broadcaster,
				Levers:           o.runLevers(),
				WorkspaceSandbox: o.workspaceSandbox,
			},
			Run:                      run,
			Task:                     task,
			Sandbox:                  o.sandbox,
			SandboxIdempotencySuffix: ":continuation:" + run.SessionID,
		})
		if createErr != nil {
			return "", createErr
		}
		if out.SandboxID == nil || out.WorkDir == "" {
			return "", domain.NewStateError("Sandbox", string(sb.Status), "continue", "replacement sandbox did not provide a workspace")
		}
		return out.WorkDir, nil
	case sandbox.SandboxStatusRejected, sandbox.SandboxStatusApproved, sandbox.SandboxStatusError:
		return "", domain.NewStateError("Sandbox", string(sb.Status), "continue", "sandbox is not resumable")
	default:
		return "", domain.NewStateError("Sandbox", string(sb.Status), "continue", "sandbox is not active or checkpointed")
	}

	workDir := sb.WorkDir
	if workDir == "" {
		workDir, err = o.sandbox.GetWorkspacePath(ctx, sb.ID)
		if err != nil {
			return "", err
		}
	}
	if workDir == "" {
		return "", domain.NewStateError("Sandbox", string(sb.Status), "continue", "resumed sandbox did not provide a workspace path")
	}
	return workDir, nil
}

// DeleteRunMessage appends a message_deleted event for a message event.
// The original message remains in the append-only stream for auditability.
func (o *Orchestrator) DeleteRunMessage(ctx context.Context, runID uuid.UUID, eventID uuid.UUID) (*domain.RunEvent, error) {
	if o.events == nil {
		return nil, domain.NewConfigMissingError("eventStore", "not configured", nil)
	}

	events, err := o.events.Get(ctx, runID, event.GetOptions{
		AfterSequence: -1,
		EventTypes:    []domain.RunEventType{domain.EventTypeMessage, domain.EventTypeMessageDeleted},
	})
	if err != nil {
		return nil, err
	}

	var target *domain.RunEvent
	alreadyDeleted := false
	targetID := eventID.String()
	for _, evt := range events {
		if evt == nil {
			continue
		}
		if evt.ID == eventID {
			target = evt
			continue
		}
		if data, ok := evt.Data.(*domain.MessageDeletedEventData); ok && data.TargetEventID == targetID {
			alreadyDeleted = true
		}
	}

	if target == nil {
		return nil, domain.NewNotFoundErrorWithID("RunEvent", targetID)
	}
	if target.EventType != domain.EventTypeMessage {
		return nil, domain.NewValidationError("eventId", "only message events can be deleted")
	}
	if alreadyDeleted {
		return nil, domain.NewStateError("RunEvent", "deleted", "delete", "message already deleted")
	}

	deleteEvent := domain.NewMessageDeletedEvent(runID, targetID)
	if err := o.appendAndBroadcastEvents(ctx, runID, deleteEvent); err != nil {
		return nil, err
	}
	return deleteEvent, nil
}

func (o *Orchestrator) prepareRunTranscript(ctx context.Context, run *domain.Run, workDir string) (*runner.TranscriptConfig, func(), error) {
	if run == nil || run.ResolvedConfig == nil {
		return nil, nil, nil
	}

	startedAt := o.now().UTC()
	if run.StartedAt != nil {
		startedAt = run.StartedAt.UTC()
	}
	runStateRoot, err := o.resolveRunStateRoot(ctx)
	if err != nil {
		return nil, nil, err
	}
	state, err := runstate.Open(run.ID, runstate.OpenOptions{
		RootDir:    runStateRoot,
		RunnerType: run.ResolvedConfig.RunnerType,
		WorkingDir: workDir,
		StartedAt:  startedAt,
		OnWrite:    func() { o.recordRunStateWrite(ctx) },
	})
	if err != nil {
		return nil, nil, err
	}

	snap := state.Snapshot()
	run.TranscriptPath = snap.TranscriptPath
	run.TranscriptCursor = snap.Cursor.TranscriptCursor
	run.TranscriptLastSeq = snap.Cursor.TranscriptLastSeq
	if err := o.runs.Update(ctx, run); err != nil {
		_ = state.Close()
		return nil, nil, err
	}

	cfg := &runner.TranscriptConfig{
		TranscriptPath: snap.TranscriptPath,
		StderrPath:     snap.StderrPath,
		StdoutFile:     state.TranscriptWriter(),
		StderrFile:     state.StderrWriter(),
		OnProcessStart: func(pid, pgid int) error {
			run.RunnerPID = pid
			run.RunnerPGID = pgid
			if err := state.PersistProcess(pid, pgid); err != nil {
				return err
			}
			return o.runs.Update(context.Background(), run)
		},
		OnAdvance: func(cursor, lastSeq int64) error {
			if cursor > run.TranscriptCursor {
				run.TranscriptCursor = cursor
			}
			if lastSeq > run.TranscriptLastSeq {
				run.TranscriptLastSeq = lastSeq
			}
			if err := state.PersistCursor(run.TranscriptCursor, run.TranscriptLastSeq); err != nil {
				return err
			}
			return o.runs.Update(context.Background(), run)
		},
		OnSessionID: func(sessionID string) error {
			if sessionID == "" || run.SessionID == sessionID {
				return nil
			}
			run.SessionID = sessionID
			if err := state.PersistSessionID(sessionID); err != nil {
				return err
			}
			return o.runs.Update(context.Background(), run)
		},
	}

	return cfg, func() { _ = state.Close() }, nil
}

// executeContinuation handles the actual continuation execution (runs in background).
// Each continuation turn gets its own timeout from RunTimeoutMinutes, so a timed-out
// run can be continued indefinitely — each "continue" message resets the clock.
func (o *Orchestrator) executeContinuation(ctx context.Context, run *domain.Run, r runner.Runner, eventSink runner.EventSink, message string, workDir string, attachments []runner.Attachment, continueEnv map[string]string, transcript *runner.TranscriptConfig, cleanupTranscript func()) {
	// Registered before cleanupTranscript so it also contains a cleanup panic.
	defer obs.RecoverToFailure("run continuation", func(failure obs.PanicFailure) {
		o.recoverPanickedRun(run, failure)
	})
	if cleanupTranscript != nil {
		defer cleanupTranscript()
	}
	previousResult := run.Result
	previousSummary := run.Summary

	levers := o.runLevers()
	timeout := levers.Execution.DefaultTimeout
	if run.ResolvedConfig != nil && run.ResolvedConfig.Timeout > 0 {
		timeout = run.ResolvedConfig.Timeout
	}
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Start heartbeat loop so the reconciler doesn't kill us during execution.
	// Heartbeat uses the parent ctx so it survives after execCtx deadline fires.
	heartbeatStop := make(chan struct{})
	heartbeatDone := make(chan struct{})
	go phases.RunHeartbeatLoop(ctx, phases.HeartbeatLoopInput{
		Deps: phases.Deps{
			Runs:        o.runs,
			Events:      o.events,
			Broadcaster: o.broadcaster,
			Levers:      levers,
		},
		Run:    run,
		Levers: levers,
		Stop:   heartbeatStop,
		Done:   heartbeatDone,
	})
	// stopHeartbeat is idempotent (sync.Once): it must run BEFORE the terminal
	// status transition below, because the heartbeat loop mutates the same *run
	// (LastHeartbeat) that applyRunStatusTransition writes — concurrent writes to
	// the shared run are a data race (the deferred -race failure). The defer is a
	// safety net for the early-return paths.
	var stopOnce sync.Once
	stopHeartbeat := func() {
		stopOnce.Do(func() {
			close(heartbeatStop)
			<-heartbeatDone
		})
	}
	defer stopHeartbeat()

	// Build continue request — pass the run's ResolvedConfig and SandboxID
	// so launcherSelector.PickFor routes the continuation through the same
	// host-or-sandbox path as the original Execute call. Without these,
	// protected runs would silently downgrade to host on continuation.
	continueReq := runner.ContinueRequest{
		RunID:          run.ID,
		SessionID:      run.SessionID,
		Prompt:         message,
		WorkingDir:     workDir,
		EventSink:      eventSink,
		Environment:    continueEnv,
		Attachments:    attachments,
		Transcript:     transcript,
		ResolvedConfig: run.ResolvedConfig,
		SandboxID:      run.SandboxID,
	}

	// Execute continuation with per-turn timeout
	result, err := r.Continue(execCtx, continueReq)
	if result != nil && result.Result != nil && run.ResolvedConfig != nil && o.structuredResults != nil {
		result.Result.Structured = o.structuredResults.Resolve(execCtx, run.ResolvedConfig.ResultSpec, result.Result)
	}

	// Stop the heartbeat loop before the terminal transition so its run writes
	// cannot race the transition's run writes (see stopHeartbeat above).
	stopHeartbeat()

	// Park coordination (durable park/resume): if the agent parked mid-turn,
	// ParkRunFromAgent transitioned the run running→parked and terminated this
	// process — which is why r.Continue returned. The park owns the lifecycle
	// from here; applying a terminal transition would clobber the park
	// (parked→failed is an allowed edge for waiter errors) and the continuation
	// checkpoint would tear down the sandbox the wake needs. Skip both.
	if o.isRunParked(ctx, run.ID) {
		obs.Component("continuation").Info("continuation ended on a parked run; leaving park intact",
			obs.KeyRunID, run.ID.String())
		return
	}

	now := o.now()
	transition := RunStatusTransitionInput{
		Run:     run,
		EndedAt: &now,
		Reason:  "Continuation completed",
	}
	if result != nil {
		transition.Result = result.Result
		transition.Summary = result.Summary
	}

	if execCtx.Err() == context.DeadlineExceeded {
		// Continuation timed out — mark as failed but preserve session ID
		// so the user can continue again with a fresh timeout.
		transition.NewStatus = domain.RunStatusFailed
		transition.Phase = domain.RunPhaseCompleted
		transition.ErrorMsg = fmt.Sprintf("continuation exceeded timeout of %s", timeout)
		run.ErrorMsg = transition.ErrorMsg
		if result != nil && result.SessionID != "" {
			run.SessionID = result.SessionID
		}
		if o.events != nil {
			errorEvent := domain.NewErrorEvent(run.ID, "continuation_timeout", run.ErrorMsg, true)
			if err := o.appendAndBroadcastEvents(ctx, run.ID, errorEvent); err != nil {
				obs.Component("orchestrator").Warn("failed to append continuation_timeout event", obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
			}
		}
	} else if err != nil {
		transition.NewStatus = domain.RunStatusFailed
		transition.Phase = domain.RunPhaseCompleted
		transition.ErrorMsg = err.Error()
		run.ErrorMsg = transition.ErrorMsg
		if o.events != nil {
			errorEvent := domain.NewErrorEvent(run.ID, "continuation_error", err.Error(), false)
			if appendErr := o.appendAndBroadcastEvents(ctx, run.ID, errorEvent); appendErr != nil {
				obs.Component("orchestrator").Warn("failed to append continuation_error event", obs.KeyRunID, run.ID.String(), obs.KeyError, appendErr.Error())
			}
		}
	} else if result != nil && !result.Success {
		transition.NewStatus = domain.RunStatusFailed
		transition.Phase = domain.RunPhaseCompleted
		transition.ErrorMsg = result.ErrorMessage
		transition.ExitCode = &result.ExitCode
		transition.Result = result.Result
		transition.Summary = result.Summary
		run.ErrorMsg = transition.ErrorMsg
		if o.events != nil && result.ErrorMessage != "" {
			errorEvent := domain.NewErrorEvent(run.ID, "continuation_error", result.ErrorMessage, false)
			if appendErr := o.appendAndBroadcastEvents(ctx, run.ID, errorEvent); appendErr != nil {
				obs.Component("orchestrator").Warn("failed to append continuation_error event", obs.KeyRunID, run.ID.String(), obs.KeyError, appendErr.Error())
			}
		}
	} else if result != nil {
		transition.NewStatus = domain.RunStatusComplete
		transition.Phase = domain.RunPhaseCompleted
		transition.Summary = result.Summary
		transition.Result = result.Result
	} else {
		transition.NewStatus = domain.RunStatusComplete
		transition.Phase = domain.RunPhaseCompleted
	}
	preservedPartialResult := false
	if transition.NewStatus == domain.RunStatusFailed && hasStructuredResult(previousResult) && !hasStructuredResult(transition.Result) {
		transition.Result = previousResult
		if transition.Summary == nil {
			transition.Summary = previousSummary
		}
		preservedPartialResult = true
	}

	// Always preserve session ID from the result for further continuation,
	// regardless of success/failure. The runner populates SessionID from
	// stream events received before process termination.
	if result != nil && result.SessionID != "" {
		run.SessionID = result.SessionID
	}

	updatedRun, err := o.applyRunStatusTransition(ctx, transition)
	if err != nil {
		obs.Component("continuation").Error("continuation status transition failed",
			obs.KeyRunID, run.ID.String(),
			obs.KeyError, err.Error(),
		)
		return
	}
	run = updatedRun
	if preservedPartialResult && o.events != nil {
		preservedTurn := 1
		if previousSummary != nil && previousSummary.TurnsUsed > 0 {
			preservedTurn = previousSummary.TurnsUsed
		}
		phases.EmitSystemEvent(ctx, phases.Deps{Events: o.events, Broadcaster: o.broadcaster}, run.ID, "info",
			fmt.Sprintf("continuation failed on turn %d; preserved structured result from successful turn %d", preservedTurn+1, preservedTurn))
	}
	if runStateRoot, rootErr := o.resolveRunStateRoot(ctx); rootErr == nil {
		EmitCodexGoalUsage(ctx, runStateRoot, phases.Deps{Events: o.events, Broadcaster: o.broadcaster}, run)
	}
	o.checkpointContinuationTurn(ctx, run, result, execCtx.Err() == context.DeadlineExceeded)
	if run.ResolvedConfig != nil {
		runStateRoot, rootErr := o.resolveRunStateRoot(ctx)
		if rootErr != nil {
			obs.Component("continuation").Warn("failed to resolve run-scoped session root", obs.KeyRunID, run.ID.String(), obs.KeyError, rootErr.Error())
		} else if cleanupErr := CleanupCodecSessionHomeCredentials(runStateRoot, run.ID, run.ResolvedConfig.RunnerType); cleanupErr != nil {
			obs.Component("continuation").Warn("failed to clean run-scoped session credentials",
				obs.KeyRunID, run.ID.String(),
				obs.KeyError, cleanupErr.Error(),
			)
		}
	}
	// Completion-driven advance: a workflow continue-node run just reached
	// terminal (the parked path returned above, keeping its attempt open).
	o.nudgeWorkflowForRun(run.ID)
}

func hasStructuredResult(result *domain.RunResult) bool {
	return result != nil && result.Structured != nil && len(result.Structured.Value) > 0
}

func (o *Orchestrator) checkpointContinuationTurn(ctx context.Context, run *domain.Run, result *runner.ExecuteResult, timedOut bool) {
	if run == nil || run.RunMode != domain.RunModeSandboxed || run.SandboxID == nil || o.sandbox == nil {
		return
	}
	outcome := domain.ContractRunOutcomeSuccess
	switch {
	case timedOut:
		outcome = domain.ContractRunOutcomeTimeout
	case run.Status == domain.RunStatusCancelled:
		outcome = domain.ContractRunOutcomeCancelled
	case run.Status == domain.RunStatusFailed:
		outcome = domain.ContractRunOutcomeFailure
	}
	cost := 0.0
	if result != nil {
		cost = result.Metrics.CostEstimateUSD
	}
	phases.ApplyAtRunEnd(ctx, phases.ApplyAtRunEndInput{
		Deps:      phases.Deps{Runs: o.runs, Events: o.events, Broadcaster: o.broadcaster, Levers: o.runLevers(), WorkspaceSandbox: o.workspaceSandbox},
		Run:       run,
		SandboxID: run.SandboxID,
		Sandbox:   o.sandbox,
		Outcome:   outcome,
		Cost:      cost,
	})
	if o.runs != nil {
		if err := o.runs.Update(ctx, run); err != nil {
			obs.Component("continuation").Error("continuation checkpoint status update failed",
				obs.KeyRunID, run.ID.String(),
				obs.KeyError, err.Error(),
			)
		}
	}
}

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
	executor.WithRunStateRoot(runStateRoot)
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
	// Policy-gate backstop (locked decision 5): interactive mode is never allowed
	// for protected (sandboxed) runs. CreateRun rejects this at validation time;
	// this defends the execution path against any run that reached here mislabeled.
	if err := domain.ValidateInteractiveRunMode(run.ExecutionMode, run.RunMode); err != nil {
		o.failInteractiveRun(ctx, run, err.Error())
		return
	}

	workDir, err := phases.UseInPlaceWorkspace(task)
	if err != nil {
		o.failInteractiveRun(ctx, run, fmt.Sprintf("resolve interactive working directory: %v", err))
		return
	}
	runStateRoot, err := o.resolveRunStateRoot(ctx)
	if err != nil {
		o.failInteractiveRun(ctx, run, "resolve interactive run state: "+err.Error())
		return
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

	if err := coord.Execute(runCtx, run, interactive.LaunchParams{
		RunID:        run.ID,
		RunnerType:   run.ResolvedConfig.RunnerType,
		Tag:          run.GetTag(),
		WorkingDir:   workDir,
		RunDir:       runDir,
		DisplayLabel: run.GetTag(),
		Prompt:       initialPrompt,
		Model:        run.ResolvedConfig.Model,
		Effort:       run.ResolvedConfig.Effort,
	}, release); err != nil {
		obs.Component("interactive").Warn("interactive run finalize failed",
			obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
	}
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

	// Validate resumability using domain decision helper
	if !run.IsResumable() {
		return nil, domain.NewStateError("Run", string(run.Status), "resume",
			fmt.Sprintf("run in %s state cannot be resumed", run.Status))
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
			o.resumeRun(context.WithoutCancel(ctx), run, task, profile, checkpoint, started)
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
func (o *Orchestrator) resumeRun(ctx context.Context, run *domain.Run, task *domain.Task, profile *domain.AgentProfile, checkpoint *domain.RunCheckpoint, started spawn.StartedFn) {
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
	executor.WithRunStateRoot(runStateRoot)
	executor.WithRunStateWriteObserver(func() { o.recordRunStateWrite(ctx) })
	executor.WithStructuredResultResolver(o.structuredResults)
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

// GetRunProgress returns the current progress of a run for display.
// This provides visibility into what phase a run is in and estimated completion.
func (o *Orchestrator) GetRunProgress(ctx context.Context, id uuid.UUID) (*domain.RunProgress, error) {
	run, err := o.GetRun(ctx, id)
	if err != nil {
		return nil, err
	}

	progress := &domain.RunProgress{
		Phase:            run.Phase,
		PhaseDescription: run.Phase.Description(),
		PercentComplete:  run.ProgressPercent,
		LastUpdate:       run.UpdatedAt,
	}

	// Calculate elapsed time
	if run.StartedAt != nil {
		progress.ElapsedTime = time.Since(*run.StartedAt)
	}

	// Add current action description based on phase
	switch run.Phase {
	case domain.RunPhaseExecuting:
		progress.CurrentAction = "Agent is working on the task"
	case domain.RunPhaseAwaitingReview:
		progress.CurrentAction = "Changes ready for review"
	case domain.RunPhaseApplying:
		progress.CurrentAction = "Applying approved changes"
	}

	return progress, nil
}

// ListStaleRuns returns runs that appear to have stalled based on their last heartbeat.
// This enables monitoring and automatic recovery of stuck runs.
func (o *Orchestrator) ListStaleRuns(ctx context.Context, staleDuration time.Duration) ([]*domain.Run, error) {
	// Get all running runs
	runningStatus := domain.RunStatusRunning
	runs, err := o.runs.List(ctx, repository.RunListFilter{
		Status: &runningStatus,
	})
	if err != nil {
		return nil, err
	}

	// Filter to stale runs
	var staleRuns []*domain.Run
	for _, run := range runs {
		if run.IsStale(staleDuration) {
			staleRuns = append(staleRuns, run)
		}
	}

	return staleRuns, nil
}

// NOTE: Approval operations (ApproveRun, RejectRun, PartialApprove)
// are implemented in approval.go for cognitive load reduction.

// -----------------------------------------------------------------------------
// Event Operations
// -----------------------------------------------------------------------------

func (o *Orchestrator) GetRunEvents(ctx context.Context, runID uuid.UUID, opts event.GetOptions) ([]*domain.RunEvent, error) {
	if o.events == nil {
		return nil, domain.NewConfigMissingError("eventStore", "not configured", nil)
	}
	return o.events.Get(ctx, runID, opts)
}

func (o *Orchestrator) StreamRunEvents(ctx context.Context, runID uuid.UUID, opts event.StreamOptions) (<-chan *domain.RunEvent, error) {
	if o.events == nil {
		return nil, domain.NewConfigMissingError("eventStore", "not configured", nil)
	}
	return o.events.Stream(ctx, runID, opts)
}

// -----------------------------------------------------------------------------
// Diff Operations
// -----------------------------------------------------------------------------

func (o *Orchestrator) GetRunDiff(ctx context.Context, runID uuid.UUID) (*sandbox.DiffResult, error) {
	run, err := o.GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}

	if run.SandboxID == nil {
		return nil, &domain.ValidationError{Field: "sandboxId", Message: "run has no sandbox"}
	}

	if o.sandbox == nil {
		return nil, domain.NewConfigMissingError("sandbox", "provider not configured", nil)
	}

	return o.sandbox.GetDiff(ctx, *run.SandboxID)
}

// -----------------------------------------------------------------------------
// Model Policy Operations
// -----------------------------------------------------------------------------

func (o *Orchestrator) GetModelHealthSnapshot(ctx context.Context) (health.Snapshot, error) {
	if o.healthStore == nil {
		return health.Snapshot{
			Models:  map[string]map[string]health.ModelEntry{},
			Runners: map[string]health.RunnerEntry{},
		}, nil
	}
	return o.healthStore.Snapshot(ctx)
}

// ExplainProfilePolicy resolves the profile against the active catalog without
// creating a run. The returned snapshot includes the same availability
// preflight and precedence explanation run creation would persist.
func (o *Orchestrator) ExplainProfilePolicy(ctx context.Context, profileID uuid.UUID) (*domain.ExecutionPolicySnapshot, error) {
	cfg, _, err := o.resolveRunConfig(ctx, CreateRunRequest{AgentProfileID: &profileID})
	if err != nil {
		return nil, err
	}
	if cfg == nil || cfg.PolicySnapshot == nil {
		return nil, domain.NewValidationError("rolePolicyCatalog", "profile resolution produced no policy snapshot")
	}
	return cfg.PolicySnapshot, nil
}

// ExplainRunPolicy returns only the immutable snapshot stored with the run. It
// never reconstructs historical provenance from the current catalog.
func (o *Orchestrator) ExplainRunPolicy(ctx context.Context, runID uuid.UUID) (*domain.ExecutionPolicySnapshot, error) {
	run, err := o.GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	if run.ResolvedConfig == nil || run.ResolvedConfig.PolicySnapshot == nil {
		return nil, nil
	}
	return run.ResolvedConfig.PolicySnapshot, nil
}

// -----------------------------------------------------------------------------
// Config Accessors
// -----------------------------------------------------------------------------

func (o *Orchestrator) GetDefaultProjectRoot() string {
	return o.config.DefaultProjectRoot
}

// ValidatePath delegates path validation to the sandbox provider.
func (o *Orchestrator) ValidatePath(ctx context.Context, path string, projectRoot string) (*sandbox.PathValidationResult, error) {
	if o.sandbox == nil {
		return nil, domain.NewConfigMissingError("sandbox", "provider not configured", nil)
	}
	return o.sandbox.ValidatePath(ctx, path, projectRoot)
}

// VerifyIdentityToken validates a signed agent identity token and returns the
// embedded claims along with the current run status.
func (o *Orchestrator) VerifyIdentityToken(ctx context.Context, token string) (*IdentityVerifyResult, error) {
	if len(o.identitySecret) == 0 {
		return &IdentityVerifyResult{Valid: false, Error: "identity system not configured"}, nil
	}

	claims, err := identity.VerifyToken(token, o.identitySecret)
	if err != nil {
		return &IdentityVerifyResult{Valid: false, Error: err.Error()}, nil
	}

	// A valid signature alone is insufficient. The token must still be the
	// active token recorded for the same run and must not have been revoked at
	// run completion. This turns a signed, time-limited bearer token into a
	// live run credential rather than a 24-hour post-completion capability.
	tokenHash := identity.HashToken(token)
	run, err := o.runs.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return &IdentityVerifyResult{Valid: false, Error: "identity token is not active"}, nil
	}
	if run.ID != claims.RunID {
		return &IdentityVerifyResult{Valid: false, Error: "identity token does not match its active run"}, nil
	}
	if run.IdentityTokenRevokedAt != nil {
		return &IdentityVerifyResult{Valid: false, Error: "identity token has been revoked"}, nil
	}

	return &IdentityVerifyResult{
		Valid:     true,
		Claims:    claims,
		RunStatus: run.Status,
	}, nil
}

// Status Operations
// -----------------------------------------------------------------------------

func (o *Orchestrator) GetHealth(ctx context.Context) (*HealthStatus, error) {
	status := &HealthStatus{
		Status:    "healthy",
		Service:   "agent-manager",
		Timestamp: o.now().UTC().Format(time.RFC3339),
		Readiness: true,
		Dependencies: &HealthDependencies{
			Runners: make(map[string]*DependencyStatus),
		},
	}

	// Check database (repositories configured)
	if o.profiles != nil && o.workflows != nil && o.tasks != nil && o.runs != nil {
		status.Dependencies.Database = &DependencyStatus{Connected: true, Storage: o.storageLabel}
	} else {
		msg := "core or workflow repository not configured"
		status.Dependencies.Database = &DependencyStatus{
			Connected: false,
			Error:     &msg,
			Storage:   o.storageLabel,
		}
	}
	if o.workflows != nil && o.workflowExecutions != nil && o.workflowEngine != nil {
		status.Dependencies.WorkflowRuntime = &DependencyStatus{Connected: true, Storage: o.storageLabel}
	} else {
		msg := "workflow catalog, execution repository, or interpreter is not configured"
		status.Dependencies.WorkflowRuntime = &DependencyStatus{Connected: false, Error: &msg, Storage: o.storageLabel}
		status.Readiness = false
		status.Status = "degraded"
	}

	// Check sandbox
	if o.sandbox != nil {
		available, msg := o.sandbox.IsAvailable(ctx)
		status.Dependencies.Sandbox = &DependencyStatus{
			Connected: available,
		}
		if !available && msg != "" {
			status.Dependencies.Sandbox.Error = &msg
		}
	} else {
		msg := "not configured"
		status.Dependencies.Sandbox = &DependencyStatus{
			Connected: false,
			Error:     &msg,
		}
	}

	// Check runners
	if o.runners != nil {
		for _, r := range o.runners.List() {
			available, msg := r.IsAvailable(ctx)
			depStatus := &DependencyStatus{
				Connected: available,
			}
			if !available && msg != "" {
				depStatus.Error = &msg
			}
			status.Dependencies.Runners[string(r.Type())] = depStatus
		}
	}

	// Count active runs
	if o.runs != nil {
		runningStatus := domain.RunStatusRunning
		runs, _ := o.runs.List(ctx, repository.RunListFilter{Status: &runningStatus})
		status.ActiveRuns = len(runs)
	}

	// Count queued tasks
	if o.tasks != nil {
		tasks, _ := o.tasks.List(ctx, repository.ListFilter{})
		var queued int
		for _, t := range tasks {
			if t.Status == domain.TaskStatusQueued {
				queued++
			}
		}
		status.QueuedTasks = queued
	}

	return status, nil
}
