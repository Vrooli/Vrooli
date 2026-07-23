package orchestration

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"agent-manager/internal/domain"
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
	producer := strings.TrimSpace(in.Producer)
	if producer == "" {
		return nil, domain.NewValidationError("producer", "producer is required to park a run")
	}
	key := strings.TrimSpace(in.Key)
	if key == "" {
		return nil, domain.NewValidationError("key", "await key is required to park a run")
	}

	run, err := o.GetRun(ctx, in.RunID)
	if err != nil {
		return nil, err
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
		// Leave the run un-parked: clear the handle we speculatively set so the
		// in-memory run matches its un-persisted (still running) status.
		run.AwaitHandle = nil
		return nil, err
	}

	// Spawn the waiter that performs the blocking wait on the agent's behalf and
	// wakes the run when the handle resolves (or its deadline elapses). Nil-safe:
	// without a registry the run is still durably parked (handle persisted) and
	// will be picked up by restart recovery once a registry is wired.
	if o.awaitRegistry != nil {
		o.awaitRegistry.Register(in.RunID, handle)
	}
	return updated, nil
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
}

// WakeRun resumes a parked run with the awaited result injected as the next
// user turn. It is idempotent: a run that is no longer parked (already woken by
// a prior resolve, or cancelled while parked) is returned unchanged rather than
// re-resumed — so a waiter double-resolve never double-wakes. The await-handle
// is cleared as part of the wake; resumeConversation transitions parked→running,
// resets the heartbeat, and re-injects the full env + a fresh identity token.
func (o *Orchestrator) WakeRun(ctx context.Context, in WakeRunInput) (*domain.Run, error) {
	run, err := o.GetRun(ctx, in.RunID)
	if err != nil {
		return nil, err
	}

	// Idempotency / replay-safety: only a parked run is woken. If it already
	// moved on, this is a no-op (the waiter resolving twice, or a stop racing a
	// resolve) — return the current run so callers treat it as already-woken.
	if run.Status != domain.RunStatusParked {
		return o.attachRunActions(ctx, run), nil
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

	reason := "Woken — awaited result available"
	if in.TimedOut {
		reason = "Woken — await deadline elapsed (timed out)"
	}
	return o.resumeConversation(ctx, run, message, nil, reason, continuationOverrides{})
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

	run, err := o.ParkRun(ctx, ParkRunInput{
		RunID:             req.RunID,
		Producer:          req.Producer,
		Key:               req.Key,
		Deadline:          req.Deadline,
		SameKeyParkStreak: streak,
	})
	if err != nil {
		return nil, err
	}

	// End the turn out of band so the HTTP response (the clean tool-result) can
	// flush to the in-run CLI before the process group is signalled.
	o.endParkedTurn(run)

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
// short grace, ending the turn. Best-effort and non-blocking: the run is already
// durably parked, so a missed kill only means the agent process lingers briefly
// (it will exit on its own once its in-flight turn completes, and the terminal
// transition it would otherwise apply is suppressed by the parked-status guard).
func (o *Orchestrator) endParkedTurn(run *domain.Run) {
	if run == nil || o.runners == nil {
		return
	}
	runnerType := runnerTypeOrEmpty(run)
	if runnerType == "" {
		return
	}
	runID := run.ID
	grace := o.parkTurnEndGrace()
	go func() {
		if grace > 0 {
			time.Sleep(grace)
		}
		r, err := o.runners.Get(runnerType)
		if err != nil || r == nil {
			return
		}
		// Use a fresh, bounded context: the request ctx is already done by now.
		stopCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := r.Stop(stopCtx, runID); err != nil {
			obs.Component("park").Debug("park turn-end stop returned",
				obs.KeyRunID, runID.String(), obs.KeyError, err.Error())
		}
	}()
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
