// This file parks eligible runs while preserving their resumable state.
package orchestration

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/interactive"
	"agent-manager/internal/orchestration/obs"
)

// DefaultParkTurnEndGrace is how long agent-manager waits after parking a run
// (and returning the clean tool-result to the in-run command) before it
// terminates the agent process group to end the turn. The grace lets the
// HTTP response flush and the in-run CLI exit so the agent records the clean
// "parked" outcome before its process group is signalled. Kept small: the
// turn is already logically over once the run is parked.
const DefaultParkTurnEndGrace = 2 * time.Second

// Park/wake orchestration core (durable park/resume, Phase 2).
//
// A run that issues an externally-owned async request (a test-genie run, a
// git-control-tower baseline diff) is *parked*: the agent process exits (zero
// tokens burned) and the run enters the non-terminal RunStatusParked, recording
// an await-handle describing the work it is blocked on. agent-manager (which
// owns the process) performs the blocking wait on the agent's behalf and *wakes*
// the run — resuming the conversation with the resolved result injected as the
// next user turn — once the work completes (or the park deadline elapses).
//
// This file holds the orchestration-core transitions. The per-producer Waiter
// seam + await-handle registry that actually drives wake (and re-spawns waiters
// after an agent-manager restart) land in Phase 3; the park/wake REST+CLI
// surface and the in-run turn-ending mechanism land in Phase 4. ParkRun/WakeRun
// are written so those layers are thin callers over this core.

// parkSameKeyStreakLimit bounds how many times in a row a run may re-park on the
// SAME await key without forward progress before agent-manager refuses the park.
//
// This is the structural protection against a coding-agent limitation: a woken
// agent that is handed an awaited result sometimes (especially weaker local
// models) re-runs the very command it just awaited instead of using the result —
// which would park again, wake again, and loop forever burning the wake budget.
// Park exists precisely to absorb agents that cannot reliably wait; this guard
// absorbs agents that cannot reliably *stop* waiting.
//
// The limit is 1 (not 0) deliberately: TranscriptLastSeq — the progress signal —
// is advanced by the asynchronous transcript scanner and can lag the live turn,
// so a single legitimate "edit, then re-run the same check" can momentarily look
// like no-progress. Tolerating one same-key re-park gives that case the benefit
// of the doubt; the second consecutive no-progress re-park is unambiguously the
// degenerate loop and is refused. Refusal never withholds data — the agent is
// handed the cached result and the cheap re-fetch command — it only declines to
// re-run the blocking work. Tunable.
const parkSameKeyStreakLimit = 1

// awaitKeyString renders the stable "producer:key" identity used to compare a new
// park request against the most recently resolved await (Run.LastAwaitKey). It
// mirrors the provenance string formatWakeMessage / formatParkMessage print so
// the guard, the wake message, and the steer message all speak the same key.
func awaitKeyString(producer, key string) string {
	producer = strings.TrimSpace(producer)
	key = strings.TrimSpace(key)
	if producer == "" && key == "" {
		return ""
	}
	return producer + ":" + key
}

const (
	// DefaultParkTTL bounds how long a parked run waits on its await-handle
	// before agent-manager wakes it with a typed timeout result rather than
	// hanging forever. Generous by default — most awaited work (a test-genie
	// suite, a baseline diff) completes well within it; a handle that never
	// resolves is the failure this guards against. Callers may supply an
	// explicit per-handle deadline to ParkRun.
	DefaultParkTTL = 30 * time.Minute
)

// ParkRunInput parameterizes a park (running→parked) transition.
type ParkRunInput struct {
	// RunID is the running run to park. It must be the live, running run that
	// owns the await (enforced by CanParkRun).
	RunID uuid.UUID
	// Producer identifies the Waiter that will resolve the handle (e.g.
	// "test-genie", "git-control-tower").
	Producer string
	// Key is the producer-scoped identifier of the awaited work.
	Key string
	// Deadline optionally overrides the wait bound. Nil ⇒ now + DefaultParkTTL.
	Deadline *time.Time
	// SameKeyParkStreak is the consecutive-same-key-park count to persist on this
	// park (computed by the re-park guard in ParkRunFromAgent). It is carried on
	// ParkRunInput so it is written atomically with the park transition.
	SameKeyParkStreak int
}

// ParkRun suspends a running run on externally-owned async work: it records the
// await-handle on the run and transitions running→parked. The run keeps its
// sandbox (mirroring needs_review) and is NOT routed through any terminal
// handler, so the identity token is not revoked and the conversation can be
// woken later. Parking does not itself terminate the agent process — the
// in-run turn-ending mechanism (Phase 4) does that after this returns; the
// reconciler will not reap the parked run in the meantime (LivenessPolicy:
// scanned, no heartbeat/process expectation).
//
// Only one open handle is permitted per run: parking an already-parked run is
// rejected by CanParkRun.
func (o *Orchestrator) ParkRun(ctx context.Context, in ParkRunInput) (*domain.Run, error) {
	updated, err := o.parkRun(ctx, in, true)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (o *Orchestrator) parkRun(ctx context.Context, in ParkRunInput, registerWaiter bool) (*domain.Run, error) {
	producer := strings.TrimSpace(in.Producer)
	if producer == "" {
		return nil, domain.NewValidationError("producer", "producer is required to park a run")
	}
	key := strings.TrimSpace(in.Key)
	if key == "" {
		return nil, domain.NewValidationError("key", "await key is required to park a run")
	}
	if o.awaitRegistry != nil && !o.awaitRegistry.SupportsProducer(producer) {
		return nil, domain.NewValidationError("producer", fmt.Sprintf("no waiter is registered for producer %q", producer))
	}

	run, err := o.GetRun(ctx, in.RunID)
	if err != nil {
		return nil, err
	}
	if run.ExecutionMode.Normalized() == domain.ExecutionModeImported {
		return nil, importedRunLifecycleError("park")
	}
	if allowed, reason := domain.CanParkRun(run); !allowed {
		return nil, domain.NewStateError("Run", string(run.Status), "park", reason)
	}

	now := o.now()
	deadline := in.Deadline
	if deadline == nil {
		d := now.Add(o.parkTTL())
		deadline = &d
	}

	// Record the handle on the run BEFORE the transition so applyRunStatusTransition
	// persists it atomically with the status change (its Update writes the whole row).
	handle := &domain.AwaitHandle{
		Producer:     producer,
		Key:          key,
		Deadline:     deadline,
		RegisteredAt: now,
	}
	run.AwaitHandle = handle
	// Persist the re-park guard's streak decision atomically with the transition.
	run.SameKeyParkStreak = in.SameKeyParkStreak

	updated, err := o.applyRunStatusTransition(ctx, RunStatusTransitionInput{
		Run:       run,
		NewStatus: domain.RunStatusParked,
		Phase:     domain.RunPhaseExecuting,
		Reason:    fmt.Sprintf("Parked waiting on %s:%s (ETA %s)", producer, key, deadline.Format(time.RFC3339)),
	})
	if err != nil {
		if updated == nil || updated.Status != domain.RunStatusParked || updated.AwaitHandle == nil ||
			updated.AwaitHandle.Producer != producer || updated.AwaitHandle.Key != key {
			// Leave the run un-parked: clear the handle we speculatively set so
			// the in-memory run matches its un-persisted status.
			run.AwaitHandle = nil
			return nil, err
		}
	}

	// Spawn the waiter that performs the blocking wait on the agent's behalf and
	// wakes the run when the handle resolves (or its deadline elapses). Nil-safe:
	// without a registry the run is still durably parked (handle persisted) and
	// will be picked up by restart recovery once a registry is wired.
	if registerWaiter && o.awaitRegistry != nil {
		o.awaitRegistry.Register(in.RunID, updated.AwaitHandle)
	}
	return updated, err
}

// WakeRunInput parameterizes a wake (parked→running) transition.
type WakeRunInput struct {
	// RunID is the parked run to wake.
	RunID uuid.UUID
	// Result is the resolved awaited result, injected as the next user turn.
	Result string
	// TimedOut indicates the park deadline elapsed without resolution; the
	// injected message is framed as a typed timeout so the agent stays in
	// control (it can retry, wait again, or proceed) rather than hanging.
	TimedOut bool
	// EffortSignalReceipt is an internal check-in watermark committed atomically
	// with this exact parked→running transition. Public wake handlers never set it.
	EffortSignalReceipt *domain.EffortSignalReceipt
}

// WakeRun resumes a parked run with the awaited result injected as the next
// user turn. It is idempotent: a run that is no longer parked (already woken by
// a prior resolve, or cancelled while parked) is returned unchanged rather than
// re-resumed — so a waiter double-resolve never double-wakes. The await-handle
// is cleared as part of the wake; resumeConversation transitions parked→running,
// resets the heartbeat, and re-injects the full env + a fresh identity token.
func (o *Orchestrator) WakeRun(ctx context.Context, in WakeRunInput) (_ *domain.Run, returnErr error) {
	effectsPossible := false
	defer func() {
		if !effectsPossible {
			returnErr = domain.RefuseBeforeEffects(returnErr)
		}
	}()
	// Wake progresses an already-admitted parked run, which remains counted by
	// maintenance inventory. It must not attempt a new maintenance admission.
	// Claim the parked state under one process-wide lifecycle lock before
	// clearing the handle or launching continuation. A second notification then
	// re-reads the now-running row and becomes the documented idempotent no-op.
	if err := o.awaitParkedTurnEnd(ctx, in.RunID); err != nil {
		return nil, err
	}
	if err := o.lockForWake(ctx, in.RunID); err != nil {
		return nil, err
	}
	defer o.wakeMu.Unlock()

	run, err := o.GetRun(ctx, in.RunID)
	if err != nil {
		return nil, err
	}
	if run.ExecutionMode.Normalized() == domain.ExecutionModeImported {
		return nil, importedRunLifecycleError("wake")
	}

	// Idempotency / replay-safety: only a parked run is woken. If it already
	// moved on, this is a no-op (the waiter resolving twice, or a stop racing a
	// resolve) — return the current run so callers treat it as already-woken.
	if run.Status != domain.RunStatusParked {
		return o.attachRunActions(ctx, run), nil
	}
	if o.deliveryEfforts != nil {
		pressure, err := o.deliveryEfforts.DeliveryRunContext(ctx, run)
		if err != nil {
			return nil, err
		}
		in.Result += pressure
	}
	if receipt := in.EffortSignalReceipt; receipt != nil {
		if strings.TrimSpace(receipt.EffortRef) == "" || receipt.Through <= 0 || !runHasEffortOrchestratorReference(run, receipt.EffortRef) {
			return nil, domain.RefuseBeforeEffects(domain.NewValidationError("effort_signal_receipt", "check-in receipt must identify a positive sequence for this run's active public effort orchestrator reference"))
		}
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
	interactiveWake := run.ExecutionMode.Normalized() == domain.ExecutionModeInteractive
	var tailContext interactive.TailContext
	if interactiveWake {
		// Codex transcript discovery must remain scoped to this run's durable
		// run-state directory. Resolve it before cancelling the parked driver,
		// clearing the await, or recording any wake state.
		if run.ResolvedConfig != nil && run.ResolvedConfig.RunnerType == domain.RunnerTypeCodex {
			var contextErr error
			tailContext, contextErr = loadInteractiveTailContextForRun(ctx, run, o.resolveRunStateRoot)
			if contextErr != nil {
				obs.Component("interactive").Warn("interactive wake could not load run-state metadata",
					obs.KeyRunID, run.ID.String(), obs.KeyError, contextErr.Error())
			}
			if strings.TrimSpace(tailContext.RunDir) == "" {
				return nil, domain.RefuseBeforeEffects(contextErr)
			}
		}
		// Prove the retained Web Console session before cancelling the parked
		// waiter or the prior interactive tail. Interactive runs continue in
		// their existing session; they must never fall through to codec-pipe
		// resumeConversation.
		if err := o.verifyInteractiveSession(ctx, run); err != nil {
			return nil, domain.RefuseBeforeEffects(err)
		}
	}
	// Parked legacy runs may predate the Task.current_run_id projection. Bind
	// only this exact requested Run, and only when no competing nonterminal
	// attempt exists, before cancelling the waiter or continuing the runner.
	// This also fences cancelled Tasks and Tasks now owned by another Run.
	if err := o.ensureCurrentTaskRun(ctx, run); err != nil {
		return nil, err
	}
	effectsPossible = true
	if interactiveWake {
		// The old interactive coordinator owns the parked turn's tail. Drain it
		// before attaching a new tailer, then re-read the exact durable owner so
		// a concurrent terminal transition cannot be overwritten by this wake.
		o.interactiveDrivers.cancelAndWait(run.ID)
		current, err := o.GetRun(ctx, run.ID)
		if err != nil {
			return nil, err
		}
		if current.Status != domain.RunStatusParked {
			return o.attachRunActions(ctx, current), nil
		}
		if current.WebConsoleSessionID != run.WebConsoleSessionID {
			return nil, domain.NewStateError("Run", string(current.Status), "wake",
				"parked interactive run changed its retained web-console session")
		}
		if err := o.ensureCurrentTaskRun(ctx, current); err != nil {
			return nil, err
		}
		run = current
	}

	// Cancel the background watcher before resuming. When wake is driven by the
	// watcher itself this just clears the (already-resolved) entry; when driven
	// externally (ops wake), it stops the still-blocked watcher so it cannot
	// also wake — defence in depth on top of WakeRun's own idempotency.
	if o.awaitRegistry != nil {
		o.awaitRegistry.Cancel(in.RunID)
	}

	message := formatWakeMessage(run.ID, run.AwaitHandle, in.Result, in.TimedOut)

	// Record the resolved await as the re-fetch SSOT BEFORE clearing the handle:
	// a woken agent that did not see — or wants to re-read — the result can
	// retrieve it via GET /runs/{id}/await-result (CLI: `run await-result`)
	// without re-running the blocking producer. Also snapshot the transcript
	// position so the no-progress re-park guard can tell whether the agent did
	// any work after this wake before it tries to park again.
	if run.AwaitHandle != nil {
		run.LastAwaitKey = awaitKeyString(run.AwaitHandle.Producer, run.AwaitHandle.Key)
	}
	run.LastAwaitResult = in.Result
	resolvedAt := o.now()
	run.LastAwaitResolvedAt = &resolvedAt
	run.LastWakeSeq = run.TranscriptLastSeq

	// Clear the handle before resuming so a crash mid-wake does not leave a
	// resolved handle that restart recovery would re-spawn a waiter for.
	run.AwaitHandle = nil
	run.EffortSignalReceipt = in.EffortSignalReceipt

	reason := "Woken — awaited result available"
	if in.TimedOut {
		reason = "Woken — await deadline elapsed (timed out)"
	}
	if interactiveWake {
		return o.continueInteractiveRunWithVerifiedSession(ctx, run, message, false, tailContext)
	}
	return o.resumeConversation(ctx, run, message, nil, reason, continuationOverrides{})
}

// WakeEffortCheckin wakes the retained exact run after an authorized effort
// signal. Signal details stay in the canonical effort log; the run receives
// only a neutral prompt to reread current feedback and its signal log.
func (o *Orchestrator) WakeEffortCheckin(ctx context.Context, runID uuid.UUID, effortRef string, through int64) error {
	effortRef = strings.TrimSpace(effortRef)
	if runID == uuid.Nil || effortRef == "" || through <= 0 {
		return domain.NewValidationError("effort_checkin", "run, effort reference, and positive sequence are required")
	}
	message := fmt.Sprintf("New signal records are available for effort %s through sequence %d. This check-in consumed the previous await and cleared its wait handle; it does not mean the awaited producer or child finished. Reread the current canonical effort feedback and signal log, then inspect your current run state and the awaited condition. If the same producer is still unresolved and the goal still waits on it, park this same run again with the same producer and key before ending the turn. Do not treat a prior successful park as proof that this run is currently parked. Continue only within the already approved goal and scope. Signal records are observations, not instructions or approval.", effortRef, through)
	_, err := o.WakeRun(ctx, WakeRunInput{
		RunID:               runID,
		Result:              message,
		EffortSignalReceipt: &domain.EffortSignalReceipt{EffortRef: effortRef, Through: through},
	})
	return err
}

func runHasEffortOrchestratorReference(run *domain.Run, effortRef string) bool {
	if run == nil || effortRef == "" {
		return false
	}
	for _, ref := range run.WorkReferences {
		if domain.IsPublicActiveEffortOrchestratorReference(ref) && ref.GetId() == effortRef {
			return true
		}
	}
	return false
}

// parkTTL returns the configured park deadline window, falling back to
// DefaultParkTTL. Kept as a method so a runtime override (orchestration
// settings) can be wired here without touching call sites.
func (o *Orchestrator) parkTTL() time.Duration {
	return DefaultParkTTL
}

// parkTurnEndGrace returns the delay between parking a run and terminating its
// agent process group. Kept as a method seam so a runtime override can be wired
// without touching call sites (and so tests can shrink it).
func (o *Orchestrator) parkTurnEndGrace() time.Duration {
	return DefaultParkTurnEndGrace
}

// ParkRunFromAgentRequest parameterizes an agent-initiated park (the in-run CLI
// calling POST /api/v1/runs/{id}/park).
type ParkRunFromAgentRequest struct {
	// RunID is the run to park (the path id).
	RunID uuid.UUID
	// Producer / Key / Deadline mirror ParkRunInput.
	Producer string
	Key      string
	Deadline *time.Time
	// IdentityToken is the caller's VROOLI_AGENT_IDENTITY_TOKEN. It authenticates
	// the caller as the owning, live run: its claims' run_id must equal RunID.
	IdentityToken string
}

// ParkRunResult is the outcome of an agent-initiated park.
type ParkRunResult struct {
	// Run is the parked run (on a refusal, the still-running run, unchanged).
	Run *domain.Run
	// Message is the tool-result text the in-run command should print. On a
	// successful park it is the clean "parked, resuming later" message; on a
	// refusal it is the steer message (which embeds the cached result).
	Message string
	// Refused is true when agent-manager declined to park because this is a
	// no-progress re-park on the same await key (the degenerate wake→re-run loop).
	// The run is NOT parked and the turn is NOT ended — the agent keeps running so
	// it can use the cached result the steer message carries.
	Refused bool
	// Result is the cached awaited result echoed back on a refusal so the agent
	// has the data inline even if it never saw the original wake injection.
	Result string
}

// ParkRunFromAgent is the agent-facing park entry point behind the park
// endpoint. It (1) authenticates the caller as the owning live run via its
// identity token, (2) parks the run (running→parked + register the waiter), and
// (3) ends the turn by terminating the agent process group after a short grace
// — so the suspended run burns zero tokens. The grace lets the HTTP response
// reach the in-run CLI (a child of the same process group) so the agent records
// the clean "parked" outcome before the group is signalled; the executeRun /
// executeContinuation goroutine that owns the now-killed process detects the
// parked status and declines to apply any terminal transition (park owns the
// lifecycle from here).
func (o *Orchestrator) ParkRunFromAgent(ctx context.Context, req ParkRunFromAgentRequest) (*ParkRunResult, error) {
	token := strings.TrimSpace(req.IdentityToken)
	if token == "" {
		return nil, parkAuthError("an identity token is required to park a run")
	}

	verified, err := o.VerifyIdentityToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if verified == nil || !verified.Valid || verified.Claims == nil {
		return nil, parkAuthError("identity token is invalid or expired")
	}
	if verified.Claims.RunID != req.RunID {
		// The token belongs to a different run — a run may only park itself.
		return nil, parkAuthError("identity token does not own this run")
	}

	// No-progress re-park guard. Decide BEFORE parking whether this is the
	// degenerate "woke with a result → re-ran the same blocking command → re-park"
	// loop. The decision needs the run's current state (last resolved await +
	// transcript progress since the last wake), so fetch it here; ParkRun re-reads
	// the run, so this read is purely for the guard.
	current, err := o.GetRun(ctx, req.RunID)
	if err != nil {
		return nil, err
	}
	streak, refuse := o.evaluateReparkGuard(current, req.Producer, req.Key)
	if refuse {
		// Refuse: do not transition, do not end the turn. Hand back the cached
		// result + an explicit steer so the agent uses the result it already has
		// instead of re-running the blocking work.
		return &ParkRunResult{
			Run:     o.attachRunActions(ctx, current),
			Message: formatReparkSteerMessage(current),
			Refused: true,
			Result:  current.LastAwaitResult,
		}, nil
	}

	if refusal := o.checkGoalHomePark(ctx, current, req.Producer, req.Deadline); refusal != "" {
		return &ParkRunResult{Run: o.attachRunActions(ctx, current), Message: refusal, Refused: true}, nil
	}

	// Install the turn-end barrier before the parked status becomes visible.
	// A producer can complete immediately, and another caller can observe the
	// persisted parked row before this request returns.
	turnEnd, err := o.prepareParkedTurnEnd(current)
	if err != nil {
		return nil, err
	}
	run, err := o.parkRun(ctx, ParkRunInput{
		RunID:             req.RunID,
		Producer:          req.Producer,
		Key:               req.Key,
		Deadline:          req.Deadline,
		SameKeyParkStreak: streak,
	}, false)
	if err != nil {
		if run == nil || run.Status != domain.RunStatusParked || run.AwaitHandle == nil ||
			run.AwaitHandle.Producer != strings.TrimSpace(req.Producer) || run.AwaitHandle.Key != strings.TrimSpace(req.Key) {
			o.failParkedTurnEnd(req.RunID, turnEnd, err)
			return nil, err
		}
		// A matching nonnil run means the status write committed but its event
		// publication failed. Keep that error visible after finishing the same
		// turn-end and waiter obligations as the successful publication path.
	}

	// End the turn out of band so the HTTP response (the clean tool-result) can
	// flush to the in-run CLI before the process group is signalled. The barrier
	// is already installed; now start its stop/drain and expose the waiter.
	o.endParkedTurn(run, turnEnd)
	if o.awaitRegistry != nil {
		o.awaitRegistry.Register(run.ID, run.AwaitHandle)
	}
	o.scheduleParkCompaction(run)
	if err != nil {
		return nil, err
	}

	return &ParkRunResult{
		Run:     run,
		Message: formatParkMessage(run.AwaitHandle),
	}, nil
}

// evaluateReparkGuard decides whether an agent-initiated park should be refused
// as a no-progress re-park, and returns the same-key-park streak value to persist
// if the park proceeds.
//
// The loop it guards: a woken agent re-runs the exact command it just awaited
// (same producer:key) without doing any work in between, which would park again
// and wake again forever. We detect "same key" via Run.LastAwaitKey (set on the
// last wake) and "no work in between" via the transcript advancing past the
// snapshot taken at that wake (best-effort — the scanner can lag, which is why
// the streak limit tolerates one re-park rather than refusing immediately).
//
// A park with forward progress, or on a different key, is never a loop: it resets
// the streak and proceeds.
func (o *Orchestrator) evaluateReparkGuard(run *domain.Run, producer, key string) (streak int, refuse bool) {
	if run == nil {
		return 0, false
	}
	incomingKey := awaitKeyString(producer, key)
	sameKey := run.LastAwaitKey != "" && incomingKey == run.LastAwaitKey
	progressed := run.TranscriptLastSeq > run.LastWakeSeq

	if progressed || !sameKey {
		// Genuine forward progress or a different await — not a loop. Reset.
		return 0, false
	}

	// Same key, no detected progress since the wake that delivered this key.
	if run.SameKeyParkStreak >= parkSameKeyStreakLimit {
		// Already tolerated the limit — this is the degenerate loop. Refuse; the
		// streak is left as-is (the run does not transition).
		return run.SameKeyParkStreak, true
	}
	// Benefit of the doubt (scanner lag): admit, but count it.
	return run.SameKeyParkStreak + 1, false
}

// endParkedTurn terminates the agent process group of a just-parked run after a
// short grace, ending the turn. It does not block the park response. If process
// exit cannot be confirmed, the retained barrier prevents a wake from starting
// a continuation beside the original turn.
type parkedTurnEnd struct {
	done   chan struct{}
	runner runner.Runner
	err    error
}

func (o *Orchestrator) prepareParkedTurnEnd(run *domain.Run) (*parkedTurnEnd, error) {
	if run == nil {
		return nil, domain.NewValidationError("run", "run is required to prepare a park turn-end barrier")
	}
	runnerType := runnerTypeOrEmpty(run)
	if runnerType == "" {
		return nil, domain.NewStateError("Run", string(run.Status), "park", "runner type is unavailable; cannot safely end the parked turn")
	}
	if o.runners == nil {
		return nil, domain.NewStateError("Run", string(run.Status), "park", "runner registry is unavailable; cannot safely end the parked turn")
	}
	r, err := o.runners.Get(runnerType)
	if err != nil {
		return nil, fmt.Errorf("resolve runner %q for safe park turn-end: %w", runnerType, err)
	}
	if r == nil {
		return nil, domain.NewStateError("Run", string(run.Status), "park", fmt.Sprintf("runner %q is unavailable; cannot safely end the parked turn", runnerType))
	}
	if _, ok := r.(interface{ PID(uuid.UUID) int }); !ok {
		return nil, domain.NewStateError("Run", string(run.Status), "park", fmt.Sprintf("runner %q cannot report process exit; cannot safely end the parked turn", runnerType))
	}
	barrier := &parkedTurnEnd{done: make(chan struct{}), runner: r}
	if _, loaded := o.parkTurnEnds.LoadOrStore(run.ID, barrier); loaded {
		return nil, domain.NewStateError("Run", string(run.Status), "park", "a prior turn-end barrier is still active")
	}
	return barrier, nil
}

func (o *Orchestrator) endParkedTurn(run *domain.Run, barrier *parkedTurnEnd) {
	if run == nil || barrier == nil {
		return
	}
	runID := run.ID
	grace := o.parkTurnEndGrace()
	go func() {
		defer func() {
			close(barrier.done)
			if barrier.err == nil {
				o.parkTurnEnds.CompareAndDelete(runID, barrier)
			}
		}()
		defer obs.RecoverToFailure("park turn-end stop", func(failure obs.PanicFailure) {
			barrier.err = failure
		})
		if grace > 0 {
			time.Sleep(grace)
		}
		// Use a fresh, bounded context: the request ctx is already done by now.
		stopCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		stopErr := barrier.runner.Stop(stopCtx, runID)
		if stopErr != nil {
			obs.Component("park").Debug("park turn-end stop returned",
				obs.KeyRunID, runID.String(), obs.KeyError, stopErr.Error())
		}
		exitErr := waitForProcessExit(stopCtx, barrier.runner, runID)
		if exitErr != nil {
			barrier.err = errors.Join(stopErr, exitErr)
		} else if stopErr != nil {
			if reporter, ok := barrier.runner.(interface{ PID(uuid.UUID) int }); !ok || reporter.PID(runID) != 0 {
				barrier.err = stopErr
			}
		}
		if barrier.err != nil {
			obs.Component("park").Warn("parked turn exit is unconfirmed; retaining wake barrier",
				obs.KeyRunID, runID.String(), obs.KeyError, barrier.err.Error())
		}
	}()
}

func (o *Orchestrator) failParkedTurnEnd(runID uuid.UUID, barrier *parkedTurnEnd, cause error) {
	if barrier == nil {
		return
	}
	barrier.err = fmt.Errorf("park transition failed before turn-end stop: %w", cause)
	close(barrier.done)
	o.parkTurnEnds.CompareAndDelete(runID, barrier)
}

// awaitParkedTurnEnd blocks until a recently parked run's turn-end stop has
// finished. A wake can arrive inside the turn-end grace (the awaited result was
// already available); resuming then would run two agent processes on one
// conversation, and the delayed stop would hit the new continuation.
func (o *Orchestrator) awaitParkedTurnEnd(ctx context.Context, runID uuid.UUID) error {
	ended, ok := o.parkTurnEnds.Load(runID)
	if !ok {
		return nil
	}
	barrier := ended.(*parkedTurnEnd)
	select {
	case <-barrier.done:
	case <-ctx.Done():
		return ctx.Err()
	}
	if barrier.err == nil {
		return nil
	}
	if reporter, ok := barrier.runner.(interface{ PID(uuid.UUID) int }); ok && reporter.PID(runID) == 0 {
		o.parkTurnEnds.CompareAndDelete(runID, barrier)
		return nil
	}
	return barrier.err
}

// waitForProcessExit waits until a runner that reports process IDs no longer
// has a live process for runID. A live process at the context deadline is an
// error because the wake barrier must not treat an unconfirmed drain as success.
func waitForProcessExit(ctx context.Context, r runner.Runner, runID uuid.UUID) error {
	reporter, ok := r.(interface{ PID(uuid.UUID) int })
	if !ok {
		return fmt.Errorf("runner cannot report process exit for parked run %s", runID)
	}
	for reporter.PID(runID) != 0 {
		select {
		case <-ctx.Done():
			if reporter.PID(runID) == 0 {
				return nil
			}
			return fmt.Errorf("waiting for parked run %s process exit: %w", runID, ctx.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}
	return nil
}

// AwaitResult is the most recently resolved await for a run — the durable result
// behind the re-fetch path. It lets a woken agent re-read the result of the work
// it parked on without re-running the blocking producer.
type AwaitResult struct {
	// Found is true when the run has a recorded resolved await.
	Found bool
	// Key is the "producer:key" identity of the resolved await.
	Key string
	// Result is the full result string that was injected into the woken turn.
	Result string
	// ResolvedAt records when the await resolved (nil when none recorded).
	ResolvedAt *time.Time
}

// GetAwaitResult returns the run's most recently resolved await result. It is a
// pure read — it never parks, blocks, or mutates the run — so a woken agent can
// retrieve the result cheaply and repeatedly. Powers GET
// /api/v1/runs/{id}/await-result and the `agent-manager run await-result` CLI.
func (o *Orchestrator) GetAwaitResult(ctx context.Context, runID uuid.UUID) (*AwaitResult, error) {
	run, err := o.GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	res := &AwaitResult{
		Key:        run.LastAwaitKey,
		Result:     run.LastAwaitResult,
		ResolvedAt: run.LastAwaitResolvedAt,
	}
	res.Found = run.LastAwaitResolvedAt != nil || strings.TrimSpace(run.LastAwaitResult) != ""
	return res, nil
}

// isRunParked reports whether the run's currently-persisted status is parked.
// Used by the execution goroutines to detect a mid-turn park and decline a
// terminal transition. Best-effort: a read error returns false so the normal
// terminal-handling path proceeds (failing safe toward finishing the run rather
// than silently leaving it stuck).
func (o *Orchestrator) isRunParked(ctx context.Context, id uuid.UUID) bool {
	if o.runs == nil {
		return false
	}
	cur, err := o.runs.Get(ctx, id)
	if err != nil || cur == nil {
		return false
	}
	return cur.Status == domain.RunStatusParked
}

// parkAuthError builds the 403-mapped error returned when park authentication
// fails (invalid token / wrong owner). PolicyViolationError{Rule:"scope_denied"}
// maps to ErrCodePolicyScope → HTTP 403.
func parkAuthError(message string) error {
	return &domain.PolicyViolationError{
		PolicyName: "run_identity",
		Rule:       "scope_denied",
		Message:    message,
		RequiredBy: "security",
	}
}

// formatParkMessage renders the clean tool-result the in-run command prints when
// a park succeeds. It is explicit that the run is suspended and will resume
// automatically, so the agent stops working rather than spinning.
func formatParkMessage(handle *domain.AwaitHandle) string {
	work := "the requested async work"
	eta := ""
	if handle != nil {
		if strings.TrimSpace(handle.Producer) != "" {
			work = fmt.Sprintf("%s:%s", handle.Producer, handle.Key)
		}
		if handle.Deadline != nil {
			eta = " (resuming by " + handle.Deadline.Format(time.RFC3339) + " at the latest)"
		}
	}
	return fmt.Sprintf(
		"PARKED — agent-manager is now waiting on %s on your behalf%s. This run is suspended (zero tokens) and will resume automatically with the result once it completes. Stop here; do not continue working — you will be woken with the result.",
		work, eta,
	)
}

// formatWakeMessage renders the user-turn message injected when a parked run is
// woken. It is deliberately explicit about provenance (which handle resolved)
// and, on timeout, about the result being unknown — so the agent reasons about
// the outcome rather than assuming success. It also carries an explicit, cheap
// re-fetch command so that even if the agent somehow did not receive the result
// inline, it has a non-blocking way to retrieve it rather than re-running the
// blocking producer (which would just make it wait again).
func formatWakeMessage(runID uuid.UUID, handle *domain.AwaitHandle, result string, timedOut bool) string {
	if handle != nil && handle.Producer == ProducerChildren {
		return formatChildrenWakeMessage(result, timedOut)
	}
	if handle != nil && handle.Producer == ProducerOperatorDecision {
		return formatOperatorDecisionWakeMessage(result, timedOut)
	}
	work := "the async work you were waiting on"
	if handle != nil && strings.TrimSpace(handle.Producer) != "" {
		work = fmt.Sprintf("the async work you parked on (%s:%s)", handle.Producer, handle.Key)
	}

	if timedOut {
		deadline := ""
		if handle != nil && handle.Deadline != nil {
			deadline = " (deadline " + handle.Deadline.Format(time.RFC3339) + ")"
		}
		msg := fmt.Sprintf(
			"[awaited result timed out] %s did not complete within the park deadline%s. Its result is unknown. Decide whether to retry the wait, proceed without the result, or report the stall — do not assume it succeeded.",
			work, deadline,
		)
		if strings.TrimSpace(result) != "" {
			msg += "\n\nPartial/last-known status:\n" + result
		}
		return msg
	}

	return fmt.Sprintf(
		"[awaited result available] %s has completed. Result:\n\n%s\n\nContinue from here with this result. Do NOT re-run the command you parked on — it has already completed and re-running it will only make you wait again. %s",
		work, strings.TrimSpace(result), reFetchHint(runID),
	)
}

// formatChildrenWakeMessage frames an orchestrator's wake briefly. A parent
// parks on its children many times a day and Codex keeps every wake message
// verbatim through compaction, so the frame carries only what changed. A timer
// wake is routine for a parent (its child is still working), not a stall.
func formatChildrenWakeMessage(result string, timedOut bool) string {
	result = strings.TrimSpace(result)
	if timedOut {
		return "[wake: timer] No child run ended since your last wake. " + result
	}
	return "[wake] " + result
}

// formatOperatorDecisionWakeMessage frames the wake of a run that parked on a
// pending operator decision. On resolution the result IS the already-composed
// answer summary (see handlers.formatDecisionWakeMessage), so it is passed
// through under a short, unambiguous banner — without the generic "do not re-run
// the command you parked on" line, which does not fit a decision wait. On the
// deadline path the result is the waiter's neutral "still pending" status.
func formatOperatorDecisionWakeMessage(result string, timedOut bool) string {
	result = strings.TrimSpace(result)
	if timedOut {
		return "[operator decision — still pending] " + result
	}
	return "[operator decision resolved] " + result
}

// reFetchHint renders the one-line instruction pointing at the non-blocking
// re-fetch command. Included in both the wake message and the re-park steer so
// the agent always has a cheap, idempotent way to re-read the awaited result
// without re-running the producer.
func reFetchHint(runID uuid.UUID) string {
	return fmt.Sprintf(
		"If you did not receive the result above or need it again, run `agent-manager run await-result %s` — it returns the result immediately without re-running the work.",
		runID.String(),
	)
}

// formatReparkSteerMessage renders the tool-result returned when agent-manager
// REFUSES a no-progress re-park. It does not withhold data: it echoes the cached
// result and points at the re-fetch command, while telling the agent to stop
// re-running the blocking producer and continue from the result it already has.
func formatReparkSteerMessage(run *domain.Run) string {
	work := "the work you just awaited"
	if run != nil && strings.TrimSpace(run.LastAwaitKey) != "" {
		work = fmt.Sprintf("the work you just awaited (%s)", run.LastAwaitKey)
	}
	msg := fmt.Sprintf(
		"NOT PARKED — agent-manager declined to wait again on %s because it already completed and you have not made progress since. Re-running it would just loop. Use the result below and continue; do not re-run that command.",
		work,
	)
	if run != nil {
		if strings.TrimSpace(run.LastAwaitResult) != "" {
			msg += "\n\nResult:\n\n" + strings.TrimSpace(run.LastAwaitResult)
		}
		msg += "\n\n" + reFetchHint(run.ID)
	}
	return msg
}
