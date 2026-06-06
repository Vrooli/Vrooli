package orchestration

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"
	"agent-manager/internal/selfidentity"
)

// Baseline Modes P6 — promote-quiesce drain.
//
// Before the platform re-points and restarts a scenario's LIVE instance during
// a `git-control-tower baseline promote`, every agent run actively executing
// against that scenario's working tree must reach a terminal state — otherwise
// the restart kills an in-flight run (potentially the very run doing the
// editing). This file is the net-new drain primitive the promote sequence
// calls. It is named "quiesce" deliberately: "drain" already names the swarm
// queue poller's behavior and the phased-plan-drain, so the promote surface is
// kept distinct to avoid collision.

const (
	// DefaultQuiesceTimeout bounds how long the drain waits for in-flight runs to
	// finish on their own before it aborts (or, with Force, cancels them).
	DefaultQuiesceTimeout = 5 * time.Minute
	// DefaultQuiescePoll is the cadence for re-checking in-flight runs.
	DefaultQuiescePoll = 2 * time.Second
)

// QuiesceOptions parameterizes a promote-quiesce drain: "make scenario <X> quiet
// enough that the platform can re-point and restart its live instance without
// killing in-flight agent runs."
type QuiesceOptions struct {
	// Scenario is the target scenario slug. It drives the default scope, the
	// self-deadlock guard, and the human-facing messaging.
	Scenario string

	// ScopePrefix overrides the working-tree scope used to find runs targeting
	// the scenario. Empty ⇒ "scenarios/<Scenario>". Sandboxed/scoped runs carry a
	// scenario-specific task ScopePath; whole-repo orchestrator runs (ecosystem-
	// manager scopes its runs to the vrooli root) are matched via TagPrefix.
	ScopePrefix string

	// TagPrefix optionally enumerates additional runs by run tag (EM tags its
	// runs), unioned with the scope match. Use it to catch whole-repo runs whose
	// task scope is the repo root rather than scenarios/<X>.
	TagPrefix string

	// ExcludeRunID is the promoting run's own ID, removed from the drain set so a
	// promote never trivially waits on itself. If that run is itself active
	// against the target scenario, the self-guard rejects the promote.
	ExcludeRunID *uuid.UUID

	// Timeout bounds the wait for in-flight runs to terminate. 0 ⇒ DefaultQuiesceTimeout.
	Timeout time.Duration

	// PollInterval is the re-check cadence. 0 ⇒ DefaultQuiescePoll.
	PollInterval time.Duration

	// Force, on timeout, cancels survivors via the graceful-first StopRun instead
	// of aborting. Default (false) aborts and leaves in-flight work untouched —
	// promote is terminal and re-runnable, so it must never destroy others' work
	// silently.
	Force bool
}

// QuiesceRunRef is a compact description of one run in the drain set.
type QuiesceRunRef struct {
	ID        string `json:"id"`
	Tag       string `json:"tag,omitempty"`
	Status    string `json:"status"`
	ScopePath string `json:"scopePath,omitempty"`
}

// QuiesceResult reports the outcome of a promote-quiesce drain.
type QuiesceResult struct {
	Scenario  string          `json:"scenario"`
	Drained   bool            `json:"drained"`             // scenario is now quiet (no in-flight runs)
	Aborted   bool            `json:"aborted"`             // timed out without Force; in-flight work left untouched
	Initial   int             `json:"initial"`             // in-flight count when the drain started (after exclusion)
	InFlight  []QuiesceRunRef `json:"inFlight,omitempty"`  // runs still active at the end (abort case)
	Cancelled []QuiesceRunRef `json:"cancelled,omitempty"` // runs force-cancelled
	WaitedMs  int64           `json:"waitedMs"`
	Reason    string          `json:"reason"` // human guidance / next action
}

// quiesceActiveStatuses are the run states that hold a live OS process executing
// in the scenario's working tree — the states a promote restart must not
// interrupt. pending (queued, no process yet) and needs_review (paused, process
// already exited) are intentionally excluded: neither is actively writing, so
// neither blocks a safe re-point + restart. This matches CanStopRun, so every
// run we enumerate for Force is also stoppable.
var quiesceActiveStatuses = []domain.RunStatus{
	domain.RunStatusRunning,
	domain.RunStatusStarting,
}

// QuiesceScenario drains in-flight agent runs targeting a scenario so the
// platform can re-point and restart its live instance. It is idempotent and
// re-runnable: calling it on an already-quiet scenario returns Drained=true
// immediately.
//
// Policy:
//   - Default (Force=false): wait up to Timeout; on timeout, ABORT and report
//     the in-flight runs without touching them (promote is re-runnable).
//   - Force=true: on timeout, cancel survivors via the graceful-first StopRun.
//   - Self-guard: if ExcludeRunID (the promoting run) is itself active against
//     the target scenario, draining can never complete (it would wait on the run
//     requesting the promote) — reject and point to the external one-shot path.
func (o *Orchestrator) QuiesceScenario(ctx context.Context, opts QuiesceOptions) (*QuiesceResult, error) {
	scenario := strings.TrimSpace(opts.Scenario)
	if scenario == "" {
		return nil, domain.NewValidationError("scenario", "scenario is required")
	}
	scopePrefix := strings.TrimRight(strings.TrimSpace(opts.ScopePrefix), "/")
	if scopePrefix == "" {
		scopePrefix = "scenarios/" + scenario
	}
	tagPrefix := strings.TrimSpace(opts.TagPrefix)
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = DefaultQuiesceTimeout
	}
	poll := opts.PollInterval
	if poll <= 0 {
		poll = DefaultQuiescePoll
	}

	result := &QuiesceResult{Scenario: scenario}

	// Snapshot the active set once to run the self-deadlock guard.
	initial, err := o.activeRunsForScenario(ctx, scenario, scopePrefix, tagPrefix)
	if err != nil {
		return nil, err
	}
	if opts.ExcludeRunID != nil {
		if _, isMember := findRunRef(initial, *opts.ExcludeRunID); isMember {
			who := "the promoting run " + opts.ExcludeRunID.String()
			if selfidentity.Is(scenario) {
				who = "agent-manager run " + opts.ExcludeRunID.String() + " (this orchestrator's own scenario)"
			}
			return nil, domain.NewValidationError(
				"scenario",
				fmt.Sprintf("cannot quiesce %q: %s is itself executing against it — draining would deadlock on the run requesting the promote. Run the promote from an external one-shot.", scenario, who),
			)
		}
	}

	drainSet := excludeRun(initial, opts.ExcludeRunID)
	result.Initial = len(drainSet)
	if len(drainSet) == 0 {
		result.Drained = true
		result.Reason = fmt.Sprintf("no in-flight runs target %q — safe to promote", scenario)
		return result, nil
	}

	start := time.Now()
	deadline := start.Add(timeout)
	for {
		remaining, err := o.activeRunsForScenario(ctx, scenario, scopePrefix, tagPrefix)
		if err != nil {
			return nil, err
		}
		remaining = excludeRun(remaining, opts.ExcludeRunID)
		if len(remaining) == 0 {
			result.Drained = true
			result.WaitedMs = time.Since(start).Milliseconds()
			result.Reason = fmt.Sprintf("%q drained — safe to promote", scenario)
			return result, nil
		}

		if !time.Now().Before(deadline) {
			result.WaitedMs = time.Since(start).Milliseconds()
			if opts.Force {
				return o.forceCancel(ctx, result, remaining, scenario, scopePrefix, tagPrefix, opts.ExcludeRunID), nil
			}
			// Default: abort, never destroy others' in-flight work.
			result.Aborted = true
			result.InFlight = remaining
			result.Reason = fmt.Sprintf(
				"%d run(s) still in-flight against %q after %s; retry once they finish, or pass --force to cancel them",
				len(remaining), scenario, timeout,
			)
			return result, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(poll):
		}
	}
}

// forceCancel cancels the survivors via the graceful-first StopRun, then
// re-checks and finalizes the result.
func (o *Orchestrator) forceCancel(
	ctx context.Context,
	result *QuiesceResult,
	remaining []QuiesceRunRef,
	scenario, scopePrefix, tagPrefix string,
	excludeRunID *uuid.UUID,
) *QuiesceResult {
	for _, ref := range remaining {
		id, perr := uuid.Parse(ref.ID)
		if perr != nil {
			result.InFlight = append(result.InFlight, ref)
			continue
		}
		if serr := o.StopRun(ctx, id); serr != nil {
			// Could not cancel it (e.g. it already moved on) — leave it visible.
			result.InFlight = append(result.InFlight, ref)
			continue
		}
		result.Cancelled = append(result.Cancelled, ref)
	}

	after, err := o.activeRunsForScenario(ctx, scenario, scopePrefix, tagPrefix)
	if err == nil {
		result.InFlight = append(result.InFlight, excludeRun(after, excludeRunID)...)
	}
	result.Drained = len(result.InFlight) == 0
	if result.Drained {
		result.Reason = fmt.Sprintf("%q drained after force-cancelling %d run(s)", scenario, len(result.Cancelled))
	} else {
		result.Reason = fmt.Sprintf(
			"force-cancelled %d run(s) but %d still active against %q — retry",
			len(result.Cancelled), len(result.InFlight), scenario,
		)
	}
	return result
}

// activeRunsForScenario enumerates the runs holding a live process against the
// scenario: scope-matched runs (task ScopePath under scenarios/<X>, refined to
// the exact directory boundary) unioned with tag-matched runs (whole-repo
// orchestrator runs identified by tag prefix). Results are de-duplicated by run
// ID.
func (o *Orchestrator) activeRunsForScenario(ctx context.Context, scenario, scopePrefix, tagPrefix string) ([]QuiesceRunRef, error) {
	seen := make(map[uuid.UUID]struct{})
	var refs []QuiesceRunRef

	for _, status := range quiesceActiveStatuses {
		st := status

		scoped, err := o.runs.List(ctx, repository.RunListFilter{Status: &st, ScopePrefix: scopePrefix})
		if err != nil {
			return nil, err
		}
		for _, run := range scoped {
			if _, ok := seen[run.ID]; ok {
				continue
			}
			// Refine the SQL LIKE prefix to the exact scenario directory boundary
			// (guards scenarios/foo vs scenarios/foo-bar). Fetch the task scope; on
			// a lookup miss, fall back to including the run (the SQL already
			// prefix-matched it) rather than silently dropping live work.
			scope := ""
			if task, terr := o.tasks.Get(ctx, run.TaskID); terr == nil && task != nil {
				scope = task.ScopePath
				if !scopePathTargetsScope(task.ScopePath, scopePrefix) {
					continue
				}
			}
			seen[run.ID] = struct{}{}
			refs = append(refs, QuiesceRunRef{ID: run.ID.String(), Tag: run.Tag, Status: string(run.Status), ScopePath: scope})
		}

		if tagPrefix == "" {
			continue
		}
		tagged, err := o.runs.List(ctx, repository.RunListFilter{Status: &st, TagPrefix: tagPrefix})
		if err != nil {
			return nil, err
		}
		for _, run := range tagged {
			if _, ok := seen[run.ID]; ok {
				continue
			}
			seen[run.ID] = struct{}{}
			refs = append(refs, QuiesceRunRef{ID: run.ID.String(), Tag: run.Tag, Status: string(run.Status)})
		}
	}

	return refs, nil
}

// scopePathTargetsScope reports whether a task scope path is exactly the target
// scope directory or a descendant of it — the boundary the SQL LIKE cannot
// express (scenarios/foo must not match scenarios/foo-bar).
func scopePathTargetsScope(scopePath, scopePrefix string) bool {
	sp := strings.TrimRight(strings.TrimSpace(scopePath), "/")
	prefix := strings.TrimRight(scopePrefix, "/")
	if prefix == "" {
		return false
	}
	return sp == prefix || strings.HasPrefix(sp, prefix+"/")
}

func findRunRef(refs []QuiesceRunRef, id uuid.UUID) (QuiesceRunRef, bool) {
	target := id.String()
	for _, r := range refs {
		if r.ID == target {
			return r, true
		}
	}
	return QuiesceRunRef{}, false
}

// excludeRun returns a new slice without the given run ID (no-op if id is nil or
// absent); it never mutates the input.
func excludeRun(refs []QuiesceRunRef, id *uuid.UUID) []QuiesceRunRef {
	if id == nil {
		return refs
	}
	target := id.String()
	out := make([]QuiesceRunRef, 0, len(refs))
	for _, r := range refs {
		if r.ID == target {
			continue
		}
		out = append(out, r)
	}
	return out
}
