package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/vrooli/internal/logx"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

// Attach/wait semantics (plan Phase 4). One primitive — awaiting the
// start-operation record reaching a terminal state — serves three callers:
// a concurrent `vrooli scenario start` (attaches instead of ErrScenarioBusy),
// the `vrooli scenario wait` verb, and the agent-manager Waiter behind
// park/wake. The owning start process remains the orchestrator; attachers
// only observe, re-verify, and (for `start`) take over when the owner dies.

// Wait verdicts. Health verdicts (healthy/degraded/running) reuse the
// health-status vocabulary; the rest name why the wait ended without one.
const (
	WaitVerdictHealthy    = "healthy"
	WaitVerdictDegraded   = "degraded"
	WaitVerdictRunning    = "running" // healthy-equivalent for no-checks scenarios
	WaitVerdictFailed     = "failed"
	WaitVerdictAbandoned  = "abandoned"
	WaitVerdictNotRunning = "not_running"
	WaitVerdictTimeout    = "timeout"
)

// WaitOptions bounds a WaitScenario call.
type WaitOptions struct {
	// Variant selects the instance (empty = live).
	Variant string
	// Timeout is a CEILING, not the expected wait: on expiry the caller
	// detaches with WaitVerdictTimeout and the awaited start is unaffected.
	// Zero applies scenarioWaitDefaultTimeout.
	Timeout time.Duration
	// OnTransition, when non-nil, is invoked whenever the observed operation
	// record changes step or dependency — the attach-side heartbeat.
	OnTransition func(StartOperationView)
}

// WaitOutcome is the single return of WaitScenario.
type WaitOutcome struct {
	Scenario string
	// Attached is true when the call actually waited on an in-flight
	// operation ("attached"); false when it evaluated current state
	// immediately ("registry").
	Attached bool
	// TimedOut is true when the ceiling elapsed (verdict WaitVerdictTimeout).
	TimedOut bool
	Verdict  string
	// Error carries the failure detail for failed/abandoned verdicts.
	Error string
	// Ports are the registry-bound ports when the verdict is a health one.
	Ports map[string]int
	// Operation is the record view backing the verdict; nil when no record
	// has ever existed for the instance.
	Operation *StartOperationView
	// Waited is how long the call blocked.
	Waited time.Duration
}

// Healthy reports whether the outcome verdict means "usable now".
func (o WaitOutcome) Healthy() bool {
	switch o.Verdict {
	case WaitVerdictHealthy, WaitVerdictDegraded, WaitVerdictRunning:
		return true
	}
	return false
}

var (
	// scenarioWaitDefaultTimeout is the default `scenario wait` ceiling —
	// generous because dependency-heavy restarts legitimately run for
	// minutes; agents pass --timeout to tighten it.
	scenarioWaitDefaultTimeout = tuning.ScenarioWaitTimeout()
	// attachPollPolicy paces reads of the operation record while attached to
	// an in-flight start. Backoff caps at 2s: the record is a local SQLite
	// read, but a multi-minute start does not need sub-second sampling.
	attachPollPolicy = AwaitPolicy{
		Interval:    tuning.HealthProbeInterval(),
		MaxInterval: tuning.LifecyclePollMaxInterval(),
	}
	// attachGracePolicy bounds how long a busy-lock caller waits for the
	// lock holder's operation record to appear before concluding the holder
	// is not a start (e.g. a concurrent stop) and surfacing ErrScenarioBusy.
	attachGracePolicy = AwaitPolicy{
		Timeout:  tuning.HealthCheckTimeout(),
		Interval: tuning.FastHealthPollInterval(),
	}
)

// WaitScenario attaches to the in-flight start operation of a scenario and
// blocks until it reaches a terminal state or the ceiling elapses. With no
// in-flight operation it evaluates current registry health and returns
// immediately. Infrastructure failures (registry unreadable, scenario
// unknown) are returned as errors; every orchestration outcome — including
// timeout — is a WaitOutcome verdict.
func (r *Runner) WaitScenario(name string, opts WaitOptions) (WaitOutcome, error) {
	key, err := scenarioruntime.ParseInstanceKey(name, opts.Variant)
	if err != nil {
		return WaitOutcome{}, err
	}
	item, err := r.loadScenario(key.Scenario, "")
	if err != nil {
		return WaitOutcome{}, err
	}
	item.Variant = key.Variant
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = scenarioWaitDefaultTimeout
	}

	deps := r.runtimeDeps()
	started := deps.now()
	outcome := WaitOutcome{Scenario: key.Scenario}

	view, ok, err := r.startOperationView(item)
	if err != nil {
		return WaitOutcome{}, err
	}
	if ok && !view.Terminal() {
		outcome.Attached = true
		final, timedOut, err := r.awaitStartOperationTerminal(item, timeout, opts.OnTransition)
		if err != nil {
			return WaitOutcome{}, err
		}
		outcome.Waited = deps.now().Sub(started)
		if timedOut {
			outcome.TimedOut = true
			outcome.Verdict = WaitVerdictTimeout
			outcome.Operation = final
			return outcome, nil
		}
		view = *final
	}
	outcome.Waited = deps.now().Sub(started)
	if ok || outcome.Attached {
		outcome.Operation = &view
	}
	r.resolveWaitVerdict(&outcome, item, view, ok)
	return outcome, nil
}

// resolveWaitVerdict maps a terminal (or absent) operation record plus the
// live registry state onto the outcome verdict. Success claims are always
// re-verified against the registry fast-path rather than trusted from the
// record.
func (r *Runner) resolveWaitVerdict(outcome *WaitOutcome, item scenario.Scenario, view StartOperationView, hasRecord bool) {
	if hasRecord {
		switch view.Status {
		case scenarioruntime.StartOperationStatusFailed:
			outcome.Verdict = WaitVerdictFailed
			outcome.Error = view.Error
			return
		case scenarioruntime.StartOperationStatusAbandoned:
			outcome.Verdict = WaitVerdictAbandoned
			outcome.Error = defaultIfEmpty(view.Error, "start abandoned before completion; re-run `vrooli scenario start`")
			return
		}
	}
	// Succeeded record or no record at all: the registry is the authority on
	// what is actually running now.
	registryView, err := r.lookupRegistryRuntime(context.Background(), item)
	if err != nil {
		r.logWarn("Wait verdict registry re-verify failed", logx.AttrScenario, item.Slug, "error", err.Error())
		if hasRecord && view.Status == scenarioruntime.StartOperationStatusSucceeded && view.Verdict != "" {
			outcome.Verdict = view.Verdict
			return
		}
		outcome.Verdict = WaitVerdictNotRunning
		outcome.Error = fmt.Sprintf("runtime registry unavailable: %v", err)
		return
	}
	if !registryView.Authoritative {
		outcome.Verdict = WaitVerdictNotRunning
		outcome.Error = "scenario is not running"
		return
	}
	health := scenario.EvaluateHealth(item.Manifest.HealthConfig(), registryView.Ports)
	outcome.Ports = registryView.Ports
	switch health {
	case "healthy":
		outcome.Verdict = WaitVerdictHealthy
	case "degraded":
		outcome.Verdict = WaitVerdictDegraded
	case "unknown", "running", "":
		// No health checks configured: registry authority is the best truth.
		outcome.Verdict = WaitVerdictRunning
	default:
		outcome.Verdict = WaitVerdictFailed
		outcome.Error = fmt.Sprintf("scenario is running but health evaluates %q", health)
	}
}

// startOperationView reads and evaluates the latest operation record for an
// instance. ok=false when no record exists.
func (r *Runner) startOperationView(item scenario.Scenario) (StartOperationView, bool, error) {
	deps := r.runtimeDeps()
	ctx := context.Background()
	store, err := deps.runtimeRegistry(ctx, r.Home)
	if err != nil {
		return StartOperationView{}, false, err
	}
	defer store.Close()
	op, err := store.GetLatestStartOperation(ctx, item.Slug, item.Variant)
	if errors.Is(err, scenarioruntime.ErrNotFound) {
		return StartOperationView{}, false, nil
	}
	if err != nil {
		return StartOperationView{}, false, err
	}
	estimates, err := store.PhaseDurationEstimates(ctx, item.Slug, item.Variant)
	if err != nil {
		estimates = nil
	}
	return EvaluateStartOperation(op, deps.isPIDRunning, deps.now().UTC(), estimates), true, nil
}

// awaitStartOperationTerminal blocks (via the shared awaiter) until the
// instance's operation record reaches a terminal state — including abandoned
// via dead-initiator detection — or the ceiling elapses (timedOut=true, the
// in-flight start unaffected).
func (r *Runner) awaitStartOperationTerminal(item scenario.Scenario, timeout time.Duration, onTransition func(StartOperationView)) (*StartOperationView, bool, error) {
	policy := attachPollPolicy
	policy.Timeout = timeout
	var last StartOperationView
	var lastKey string
	err := Await(r.awaitClock(), policy, func() (bool, error) {
		view, ok, err := r.startOperationView(item)
		if err != nil {
			return false, err
		}
		if !ok {
			// The record vanished (pruned by a newer start burst); treat as
			// abandoned rather than spinning on nothing.
			last.Status = scenarioruntime.StartOperationStatusAbandoned
			return true, nil
		}
		if onTransition != nil {
			if key := view.CurrentStep + "\x00" + view.DependencyCurrent; key != lastKey {
				lastKey = key
				onTransition(view)
			}
		}
		last = view
		return view.Terminal(), nil
	})
	if errors.Is(err, ErrAwaitExpired) {
		return &last, true, nil
	}
	if err != nil {
		return nil, false, err
	}
	return &last, false, nil
}

// attachToInFlightStart handles a Start that lost the scenario lock: when the
// holder is a live in-flight start (its operation record exists), wait for
// its verdict; when the owner dies mid-flight, report takeover so the caller
// retries the lock and resumes the start. A busy lock with no start record
// within the grace window is a true conflict (e.g. a concurrent stop) and
// keeps ErrScenarioBusy.
type attachResult struct {
	takeOver bool
	result   Result
	err      error
}

func (r *Runner) attachToInFlightStart(item scenario.Scenario, busyErr error) attachResult {
	// The lock holder is the authority on whether a competing operation is live.
	// Start-operation records are written per top-level invocation, so a
	// dependency-driven start holds the lock without writing one for this
	// scenario — and the newest record on file may then be a terminal corpse
	// from an unrelated earlier run. Reading that corpse as "the holder died"
	// makes every caller attempt a takeover it can never win: the lock stays
	// held by a live process, the retry loop burns its attempts, and the
	// scenario becomes permanently unstartable while the holder works.
	holderLive := busyLockHolderProvenLive(busyErr)

	// Grace: the lock holder creates its record just after taking the lock;
	// give it a moment to appear before concluding "not a start".
	var found, abandoned bool
	graceErr := Await(r.awaitClock(), attachGracePolicy, func() (bool, error) {
		view, ok, err := r.startOperationView(item)
		if err != nil {
			return false, err
		}
		if ok && !holderLive && view.Status == scenarioruntime.StartOperationStatusAbandoned {
			// The record claims running but its initiator is dead (or it was
			// explicitly abandoned): take over rather than waiting on a corpse.
			abandoned = true
			return true, nil
		}
		found = ok && !view.Terminal()
		return found, nil
	})
	if graceErr != nil && !errors.Is(graceErr, ErrAwaitExpired) {
		return attachResult{err: graceErr}
	}
	if abandoned {
		r.logWarn("In-flight start record is abandoned (dead initiator); taking over", logx.AttrScenario, item.Slug)
		return attachResult{takeOver: true}
	}
	if !found {
		return attachResult{err: busyErr}
	}

	r.publish(ProgressEvent{Kind: EventOperationStarted, Scenario: item.Slug, Operation: "attach"})
	r.logInfo("Attaching to in-flight scenario start", logx.AttrScenario, item.Slug)
	outcome, err := r.WaitScenario(item.Slug, WaitOptions{
		Variant:      item.Variant,
		OnTransition: r.renderAttachTransition,
	})
	if err != nil {
		return attachResult{err: err}
	}
	switch {
	case outcome.Verdict == WaitVerdictAbandoned:
		// Owner died mid-flight: the caller retries the lock and takes over.
		r.logWarn("In-flight start abandoned by its owner; taking over", logx.AttrScenario, item.Slug)
		return attachResult{takeOver: true}
	case outcome.TimedOut:
		return attachResult{err: fmt.Errorf("attached to in-flight start of %q but it did not finish within %s", item.Slug, scenarioWaitDefaultTimeout)}
	case outcome.Healthy():
		return attachResult{result: Result{
			Scenario:       item,
			AllocatedPorts: outcome.Ports,
			Health:         outcome.Verdict,
			AlreadyRunning: true,
		}}
	default:
		return attachResult{err: fmt.Errorf("in-flight start of %q finished %s: %s", item.Slug, outcome.Verdict, outcome.Error)}
	}
}

// renderAttachTransition is the attacher-side heartbeat: modest console
// lines derived from the owner's operation record.
func (r *Runner) renderAttachTransition(view StartOperationView) {
	if r.Verbosity == VerbosityVerbose || r.Out == nil {
		return
	}
	if line := view.TransitionLine(); line != "" {
		fmt.Fprintln(r.Out, line)
	}
}
