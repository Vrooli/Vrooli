// Responsibility: validate and execute retained-session continuations with bounded authority.
package orchestration

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/adapters/webconsole"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/orchestration/phases"
	"agent-manager/internal/promptmanager"
	"agent-manager/internal/runstate"

	"github.com/google/uuid"
)

func continuationRequestHash(req ContinueRunRequest) []byte {
	data, _ := json.Marshal(req)
	sum := sha256.Sum256(data)
	return sum[:]
}

func (o *Orchestrator) ContinueRun(ctx context.Context, req ContinueRunRequest) (_ *domain.Run, returnErr error) {
	effectsPossible := false
	defer func() {
		if !effectsPossible {
			returnErr = domain.RefuseBeforeEffects(returnErr)
		}
	}()
	if req.IdempotencyKey != "" && o.idempotency != nil {
		effectsPossible = true // An unreadable or pending receipt may belong to accepted dispatch.
		existing, err := o.idempotency.Check(ctx, req.IdempotencyKey)
		if err != nil {
			return nil, err
		}
		if existing != nil && existing.Status == domain.IdempotencyStatusComplete && existing.EntityID != nil {
			if existing.EntityType != "Continuation" || *existing.EntityID != req.RunID || string(existing.Response) != string(continuationRequestHash(req)) {
				return nil, domain.RefuseBeforeEffects(domain.NewValidationErrorWithCode("idempotencyKey", "key belongs to a different operation or continuation request", domain.ErrCodeValidationConflict))
			}
			return o.GetRun(ctx, *existing.EntityID)
		}
		if existing != nil && existing.Status == domain.IdempotencyStatusPending {
			return nil, domain.NewStateError("Run", "continuing", "continue", "a continuation with this idempotency key is already in progress")
		}
		effectsPossible = false
	}
	releaseAdmission, err := o.admitMaintenance(ctx)
	if err != nil {
		return nil, err
	}
	defer releaseAdmission()
	// Validate message
	o.wakeMu.Lock()
	defer o.wakeMu.Unlock()
	if strings.TrimSpace(req.Message) == "" {
		return nil, domain.NewValidationError("message", "message is required")
	}

	// Get the run
	run, err := o.GetRun(ctx, req.RunID)
	if err != nil {
		return nil, err
	}
	if run.ExecutionMode.Normalized() == domain.ExecutionModeImported {
		return nil, importedRunLifecycleError("continue")
	}
	if run.ExecutionMode.Normalized() == domain.ExecutionModeInteractive && run.ResolvedConfig != nil && strings.TrimSpace(run.ResolvedConfig.Until) != "" {
		if run.ObservedGoalStatus == string(runner.GoalStatusPaused) {
			return nil, domain.NewStateError("Run", "goal_paused", "continue", "native goal is paused; input was rejected before transport and requires explicit owner reconciliation")
		}
	}
	typed, err := o.typeIntoRunningSession(ctx, run, req)
	if typed {
		// Input reached (or may have reached) the session, even when the
		// submit was not confirmed: never report this as a pre-effect refusal.
		effectsPossible = true
	}
	if err != nil {
		return nil, err
	}
	if typed {
		return o.attachRunActions(ctx, run), nil
	}

	if allowed, reason := domain.CanContinueRun(run); !allowed {
		return nil, domain.NewStateError("Run", string(run.Status), "continue", reason)
	}
	if err := validateExecutionModel(run.ResolvedConfig); err != nil {
		return nil, err
	}
	if err := o.validateCurrentExecutionModel(ctx, run.ResolvedConfig); err != nil {
		return nil, err
	}
	if err := o.validateContinuationSession(ctx, run); err != nil {
		return nil, err
	}
	if req.IdempotencyKey != "" && o.idempotency != nil {
		if _, err := o.idempotency.Reserve(ctx, req.IdempotencyKey, time.Hour); err != nil {
			effectsPossible = true // Another caller won; its outcome is not a refusal.
			return nil, domain.NewStateError("Run", "continuing", "continue", "a continuation with this idempotency key is already in progress")
		}
	}

	// Interactive runs continue by typing the follow-up into the still-live
	// web-console session (never a process respawn) and reattaching a tailer to
	// drive the new turn to completion — see continueInteractiveRun.
	if run.ExecutionMode.Normalized() == domain.ExecutionModeInteractive {
		if req.MaxTurns != nil || req.MaxToolCalls != nil || req.Timeout != nil || req.ResultSpec != nil {
			o.markIdempotencyFailed(ctx, req.IdempotencyKey)
			return nil, domain.NewValidationError("continuationOverrides", "interactive continuation does not support per-turn workflow overrides")
		}
		effectsPossible = true
		continued, err := o.continueInteractiveRun(ctx, run, req.Message, req.AttachmentIDs, req.ReinstallGoal)
		if err != nil {
			return nil, err
		}
		if err := o.completeContinuationReceipt(ctx, req); err != nil {
			return nil, err
		}
		return continued, nil
	}
	effectsPossible = true
	continued, err := o.resumeConversation(ctx, run, req.Message, req.AttachmentIDs, "Continuation requested", continuationOverrides{MaxTurns: req.MaxTurns, MaxToolCalls: req.MaxToolCalls, Timeout: req.Timeout, ResultSpec: req.ResultSpec})
	if err != nil {
		if domain.IsPreEffectRefusal(err) {
			o.markIdempotencyFailed(ctx, req.IdempotencyKey)
		}
		return nil, err
	}
	if err := o.completeContinuationReceipt(ctx, req); err != nil {
		return nil, err
	}
	return continued, nil
}

func (o *Orchestrator) observeInteractiveRun(ctx context.Context, run *domain.Run) error {
	if run == nil || run.ExecutionMode.Normalized() != domain.ExecutionModeInteractive || run.ResolvedConfig == nil || strings.TrimSpace(run.ResolvedConfig.Until) == "" || o.events == nil {
		return nil
	}
	events, err := o.allRunEvents(ctx, run.ID, event.GetOptions{AfterSequence: -1, EventTypes: []domain.RunEventType{
		domain.EventTypeGoalStatusChanged, domain.EventTypeMessage, domain.EventTypeToolCall, domain.EventTypeToolResult,
	}})
	if err != nil {
		return err
	}
	for _, observed := range events {
		if observed == nil {
			continue
		}
		if observed.EventType == domain.EventTypeGoalStatusChanged {
			goal, ok := observed.Data.(*domain.GoalStatusChangedEventData)
			if ok && goal != nil && observedGoalMatchesRun(run, goal.Objective) && runner.GoalStatus(goal.Status).Valid() {
				status := goal.Status
				run.ObservedGoalStatus = status
				t := observed.Timestamp
				run.ObservedGoalAt = &t
			}
			continue
		}
		if observed.EventType == domain.EventTypeMessage || observed.EventType == domain.EventTypeToolCall || observed.EventType == domain.EventTypeToolResult {
			t := observed.Timestamp
			run.ProviderActivityAt = &t
		}
	}
	return nil
}

func observedGoalMatchesRun(run *domain.Run, objective string) bool {
	objective = strings.TrimSpace(objective)
	if objective == "" {
		return false
	}
	if run == nil || run.ResolvedConfig == nil || strings.TrimSpace(run.ResolvedConfig.Until) == "" {
		return true
	}
	want := strings.TrimSpace(run.ResolvedConfig.Until)
	// Interactive launch may append a pasted-text attachment path after the
	// installed objective. A different prefix belongs to another objective.
	return objective == want || strings.HasPrefix(objective, want+"\n")
}

// typeIntoRunningSession delivers a continuation to a running interactive
// session as its next user message; the harness queues it behind the current
// turn. It reports false for every other run, which continues between turns,
// and true whenever input was sent to the session, including when the submit
// failed or could not be confirmed. A delivered directive is recorded as a
// user message event so mid-turn continuations stay auditable.
func (o *Orchestrator) typeIntoRunningSession(ctx context.Context, run *domain.Run, req ContinueRunRequest) (bool, error) {
	if run.Status != domain.RunStatusRunning || run.ExecutionMode.Normalized() != domain.ExecutionModeInteractive || run.WebConsoleSessionID == "" || o.interactiveSessions == nil {
		return false, nil
	}
	if len(req.AttachmentIDs) > 0 || req.MaxTurns != nil || req.MaxToolCalls != nil || req.Timeout != nil || req.ResultSpec != nil || req.ReinstallGoal {
		return false, domain.NewValidationError("continuationOverrides", "a running session accepts message text only")
	}
	if req.IdempotencyKey != "" && o.idempotency != nil {
		if _, err := o.idempotency.Reserve(ctx, req.IdempotencyKey, time.Hour); err != nil {
			return false, domain.NewStateError("Run", "continuing", "continue", "a continuation with this idempotency key is already in progress")
		}
	}
	text := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(req.Message, "\r\n", " "), "\n", " "))
	// Web Console's SendText forwards LF literally to the PTY. Codex's TUI
	// requires a named Enter key after a paste; otherwise the directive remains
	// composed but unsubmitted. SendPrompt owns that paste+Enter contract.
	submission, err := o.interactiveSessions.SendPrompt(ctx, run.WebConsoleSessionID, text, interactiveRunSource(run.ID))
	if errors.Is(err, webconsole.ErrPromptNotSubmitted) {
		// The directive is visibly composed but unsent. Keep the pending
		// idempotency reservation: a same-key retry would paste a second copy.
		o.appendRunLogEvent(ctx, run.ID, "warn", fmt.Sprintf("continuation typed into web-console session %s was not submitted after %d Enter presses; it remains in the composer", run.WebConsoleSessionID, submission.EnterPresses))
		return true, domain.NewStateError("Run", string(run.Status), "continue", fmt.Sprintf(
			"the directive was typed into web-console session %s but not submitted after %d Enter presses and still sits in the agent's composer; submit or clear it there before retrying",
			run.WebConsoleSessionID, submission.EnterPresses))
	}
	if err != nil {
		// A failed response can occur after paste or Enter. Keep the pending
		// idempotency reservation: retrying could submit the same directive twice.
		return true, fmt.Errorf("interactive input transport outcome unknown; reconcile the session before retrying the same key: %w", err)
	}
	if aerr := o.appendAndBroadcastEvents(ctx, run.ID, domain.NewMessageEvent(run.ID, "user", req.Message)); aerr != nil {
		obs.Component("interactive").Warn("continuation message event append failed", obs.KeyRunID, run.ID.String(), obs.KeyError, aerr.Error())
	}
	if !submission.Verified {
		o.appendRunLogEvent(ctx, run.ID, "warn", fmt.Sprintf("continuation typed into web-console session %s; submission unverified (the composer could not be observed)", run.WebConsoleSessionID))
	}
	return true, o.completeContinuationReceipt(ctx, req)
}

// appendRunLogEvent records a best-effort log event on the run.
func (o *Orchestrator) appendRunLogEvent(ctx context.Context, runID uuid.UUID, level, message string) {
	if o.events == nil {
		return
	}
	if err := o.appendAndBroadcastEvents(ctx, runID, domain.NewLogEvent(runID, level, message)); err != nil {
		obs.Component("interactive").Warn("run log event append failed", obs.KeyRunID, runID.String(), obs.KeyError, err.Error())
	}
}

func (o *Orchestrator) completeContinuationReceipt(ctx context.Context, req ContinueRunRequest) error {
	if req.IdempotencyKey == "" || o.idempotency == nil {
		return nil
	}
	return o.idempotency.Complete(context.WithoutCancel(ctx), req.IdempotencyKey, req.RunID, "Continuation", continuationRequestHash(req))
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
	MaxTurns     *int
	MaxToolCalls *int
	Timeout      *time.Duration
	ResultSpec   *domain.ResultSpec
}

func (o *Orchestrator) validateContinuationSession(ctx context.Context, run *domain.Run) error {
	if run.ExecutionMode.Normalized() == domain.ExecutionModeInteractive {
		return nil // The interactive substrate owns its live web-console session.
	}
	if run.ResolvedConfig == nil || o.runners == nil {
		return nil // Existing config/capability gates report these failures.
	}
	r, err := o.runners.Get(run.ResolvedConfig.RunnerType)
	if err != nil {
		return err
	}
	checker, ok := r.(runner.ContinuationPreflighter)
	if !ok {
		return nil
	}
	env := make(map[string]string, len(run.CustomEnv)+1)
	for key, value := range run.CustomEnv {
		env[key] = value
	}
	if run.ResolvedConfig.RunnerType == domain.RunnerTypeOpenCode {
		root, err := o.resolveRunStateRoot(ctx)
		if err != nil {
			return err
		}
		runDir, err := runstate.RunDir(root, run.ID)
		if err != nil {
			return err
		}
		env["XDG_DATA_HOME"] = openCodeRunDataHome(runDir)
	}
	return checker.ValidateContinuation(ctx, runner.ContinueRequest{RunID: run.ID, SessionID: run.SessionID, Environment: env, ResolvedConfig: run.ResolvedConfig, SandboxID: run.SandboxID})
}

func (o *Orchestrator) resumeConversation(ctx context.Context, run *domain.Run, message string, attachmentIDs []string, reason string, overrides continuationOverrides) (_ *domain.Run, returnErr error) {
	effectsPossible := false
	defer func() {
		if !effectsPossible {
			returnErr = domain.RefuseBeforeEffects(returnErr)
		}
	}()
	// The immutable resolved config is the sole execution authority.
	var runnerType domain.RunnerType
	if run.ResolvedConfig != nil {
		runnerType = run.ResolvedConfig.RunnerType
	}

	if runnerType == "" {
		return nil, domain.NewStateError("Run", string(run.Status), "continue",
			"cannot determine runner type for this run")
	}
	if err := validateExecutionModel(run.ResolvedConfig); err != nil {
		return nil, err
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
	// Wake also enters here; check before sandbox, status, event, transcript,
	// credential, or runtime-root writes. Continue checks before reservation too.
	if err := o.validateContinuationSession(ctx, run); err != nil {
		return nil, err
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

	effectsPossible = true
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
		if overrides.MaxToolCalls != nil {
			if *overrides.MaxToolCalls < 0 {
				return nil, domain.NewValidationError("maxToolCalls", "must be zero or positive")
			}
			resolved.MaxToolCalls = *overrides.MaxToolCalls
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
	repoRoot := ""
	if task != nil {
		scopePath = task.ScopePath
		repoRoot = task.ProjectRoot
	}
	identityToken := ""
	if len(o.identitySecret) > 0 {
		identityToken = phases.GenerateIdentityToken(ctx, phases.GenerateIdentityTokenInput{
			Deps:            phases.Deps{Runs: o.runs, Events: o.events, Broadcaster: o.broadcaster},
			Run:             run,
			Profile:         profile,
			Task:            task,
			Secret:          o.identitySecret,
			Meta:            workflowIdentityMeta(run.CustomEnv),
			AccountScopes:   run.OwnerScopes,
			RequestedScopes: run.RequestedScopes,
		})
	}
	env := phases.AssembleRunEnv(phases.AssembleRunEnvInput{
		Custom:        run.CustomEnv,
		RunMode:       run.RunMode,
		SandboxID:     run.SandboxID,
		WorkDir:       workDir,
		RepoRoot:      repoRoot,
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
	runtimeEnv, err := PrepareRunnerRuntimeRoot(runStateRoot, run.ID)
	if err != nil {
		return nil, err
	}
	sessionEnv, err := PrepareCodecSessionHome(runStateRoot, run.ID, run.ResolvedConfig.RunnerType)
	if err != nil {
		return nil, err
	}
	skillEnv, err := PrepareRunnerSkillScope(runStateRoot, run.ID, run.ResolvedConfig.RunnerType)
	if err != nil {
		return nil, err
	}
	for _, envSet := range []map[string]string{runtimeEnv, sessionEnv, skillEnv} {
		for key, value := range envSet {
			if env == nil {
				env = make(map[string]string)
			}
			env[key] = value
		}
	}
	if len(run.ResolvedConfig.SkillPack) > 0 {
		runtimeRoot := ""
		for _, key := range []string{"CLAUDE_CONFIG_DIR", "CODEX_HOME", "GROK_HOME", "OPENCODE_CONFIG_DIR", "ANTIGRAVITY_STATE_DIR"} {
			if value := env[key]; value != "" {
				runtimeRoot = value
				break
			}
		}
		source, ok := o.promptClient.(promptmanager.SourceClient)
		if !ok || source == nil {
			return nil, fmt.Errorf("continuation profile skill pack requires prompt-manager source")
		}
		var projectErr error
		if run.ResolvedConfig.SkillExperimentID != "" {
			_, projectErr = ProjectProfileSkillsAssigned(ctx, runStateRoot, run.ID, runtimeRoot, run.ResolvedConfig.SkillExperimentID, run.ResolvedConfig.SkillPack, source, defaultProfileSkillTokenCeiling)
		} else {
			_, projectErr = ProjectProfileSkills(ctx, runStateRoot, run.ID, runtimeRoot, run.ResolvedConfig.SkillPack, source, defaultProfileSkillTokenCeiling)
		}
		if projectErr != nil {
			return nil, projectErr
		}
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
	runStateRoot, err := o.resolveRunStateRoot(ctx)
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
			RunStateRoot:             runStateRoot,
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
	if run.OwnerIdentity == "" {
		run.OwnerIdentity = o.runtimeOwnerIdentity
	}
	if run.OwnerEpoch == 0 {
		run.OwnerEpoch = 1
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
		RootDir: runStateRoot, RunnerType: run.ResolvedConfig.RunnerType, WorkingDir: workDir, StartedAt: startedAt,
		RunnerIdentity: fmt.Sprintf("%s:%s:%d", run.OwnerIdentity, run.ID, run.OwnerEpoch), OwnerEpoch: run.OwnerEpoch,
		OnWrite: func() { o.recordRunStateWrite(ctx) },
	})
	if err != nil {
		return nil, nil, err
	}

	snap := state.Snapshot()
	run.TranscriptPath = snap.TranscriptPath
	// This owner prepares a new continuation, whose stdout appends to the
	// original transcript. Persist the old file boundary before dispatch so
	// restart recovery cannot treat an earlier terminal marker as its result.
	info, err := state.TranscriptWriter().Stat()
	if err != nil {
		_ = state.Close()
		return nil, nil, err
	}
	run.TranscriptCursor = info.Size()
	if err := state.PersistCursor(run.TranscriptCursor, run.TranscriptLastSeq); err != nil {
		_ = state.Close()
		return nil, nil, err
	}
	if err := o.runs.Update(ctx, run); err != nil {
		_ = state.Close()
		return nil, nil, err
	}

	// Callbacks own their stream snapshot; they must never mutate the returned
	// run or persist its stale lifecycle/heartbeat projection.
	streamRun := *run
	run = &streamRun
	persistStream := func() error {
		updated, err := o.runs.UpdateRunnerStreamState(context.Background(), run)
		if err == nil && !updated {
			return domain.NewStateError("Run", "superseded", "stream", "continuation no longer owns this lifecycle")
		}
		return err
	}
	cfg := &runner.TranscriptConfig{
		TranscriptPath: snap.TranscriptPath,
		StderrPath:     snap.StderrPath,
		StdoutFile:     state.TranscriptWriter(),
		StderrFile:     state.StderrWriter(),
		OnProcessStart: func(pid, pgid int) error {
			run.RunnerPID = pid
			run.RunnerPGID = pgid
			if err := persistStream(); err != nil {
				return err
			}
			if err := state.PersistProcess(pid, pgid); err != nil {
				return err
			}
			return nil
		},
		OnAdvance: func(cursor, lastSeq int64) error {
			if cursor > run.TranscriptCursor {
				run.TranscriptCursor = cursor
			}
			if lastSeq > run.TranscriptLastSeq {
				run.TranscriptLastSeq = lastSeq
			}
			if err := persistStream(); err != nil {
				return err
			}
			if err := state.PersistCursor(run.TranscriptCursor, run.TranscriptLastSeq); err != nil {
				return err
			}
			return nil
		},
		OnSessionID: func(sessionID string) error {
			if sessionID == "" || run.SessionID == sessionID {
				return nil
			}
			run.SessionID = sessionID
			if err := persistStream(); err != nil {
				return err
			}
			if err := state.PersistSessionID(sessionID); err != nil {
				return err
			}
			return nil
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
		RunID:  run.ID,
		Tag:    run.GetTag(),
		State:  phases.HeartbeatState{LastRunHeartbeat: run.SafeLastHeartbeat()},
		Levers: levers,
		Stop:   heartbeatStop,
		Done:   heartbeatDone,
	})
	// stopHeartbeat is idempotent (sync.Once) and drains the owner writes before
	// the terminal transition. The loop holds copied identity/timestamps, never
	// the executor's mutable run. The defer covers early-return paths.
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
		Tag:            run.GetTag(),
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

	// Rebuild owner policy from persisted authority on every fresh process;
	// runtime-home files are not a source of authority after a continuation.
	var result *runner.ExecuteResult
	root, err := o.resolveRunStateRoot(execCtx)
	if err == nil {
		continueReq.PolicyFiles, err = prepareRunnerPolicy(root, run.ID, run.ResolvedConfig)
	}
	if err == nil {
		result, err = r.Continue(execCtx, continueReq)
	}
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
		if sessionLostOnContinue(err, result) {
			run.TerminalClass = domain.RunTerminalClassInterruption
			run.StopReason = domain.RunStopReasonSessionLost
		}
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
		if sessionLostOnContinue(nil, result) {
			run.TerminalClass = domain.RunTerminalClassInterruption
			run.StopReason = domain.RunStopReasonSessionLost
		}
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
	if preservedPartialResult && o.events != nil {
		preservedTurn := 1
		if previousSummary != nil && previousSummary.TurnsUsed > 0 {
			preservedTurn = previousSummary.TurnsUsed
		}
		// Persist the preservation evidence before publishing the terminal
		// status. Readers must never observe a failed continuation without the
		// event that explains which successful structured result was retained.
		phases.EmitSystemEvent(ctx, phases.Deps{Events: o.events, Broadcaster: o.broadcaster}, run.ID, "info",
			fmt.Sprintf("continuation failed on turn %d; preserved structured result from successful turn %d", preservedTurn+1, preservedTurn))
	}

	// A continuation mints a fresh live credential at admission. Retire that
	// generation with this terminal turn, just as the initial execution does.
	// The parked path returned above and retains its separate wake lifecycle.
	phases.RevokeIdentityToken(run)
	updatedRun, err := o.applyRunStatusTransition(ctx, transition)
	if err != nil {
		obs.Component("continuation").Error("continuation status transition failed",
			obs.KeyRunID, run.ID.String(),
			obs.KeyError, err.Error(),
		)
		return
	}
	run = updatedRun
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
		if cleanupErr := CleanupRunnerSkillScope(runStateRoot, run.ID, run.ResolvedConfig.RunnerType); cleanupErr != nil {
			obs.Component("continuation").Warn("failed to clean run-scoped skill scope",
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

// sessionLostOnContinue reports whether a continuation turn failed because the
// runner-side session no longer exists. It inspects both shapes: the typed
// error the codec-pipe path returns directly and the typed TerminalError the
// durable-transcript path now preserves on the result. A session-lost
// continuation is an interruption, not a terminal failure, so the run is typed
// interruption/session_lost and Swarm's until-allowance sweeper starts a fresh
// run instead of finalizing.
func sessionLostOnContinue(err error, result *runner.ExecuteResult) bool {
	if runnerErrorIsSessionLost(err) {
		return true
	}
	if result != nil {
		return runnerErrorIsSessionLost(result.TerminalError)
	}
	return false
}

func runnerErrorIsSessionLost(err error) bool {
	var runnerErr *domain.RunnerError
	if !errors.As(err, &runnerErr) {
		return false
	}
	switch runnerErr.Code() {
	case domain.ErrCodeRunnerSessionExpired, domain.ErrCodeRunnerSessionStateLost:
		return true
	default:
		return false
	}
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
	// Continuations need the same detached, generation-checked cleanup and
	// final projection as initial turns, including when apply is deferred.
	phases.Finalize(phases.FinalizeInput{
		Deps: phases.Deps{Runs: o.runs, Events: o.events, Broadcaster: o.broadcaster, Levers: o.runLevers(), WorkspaceSandbox: o.workspaceSandbox},
		Run:  run, SandboxID: run.SandboxID, Sandbox: o.sandbox,
		Event: phases.TurnLifecycleEventForOutcome(outcome),
	})
}
