// This file coordinates quiescing orchestration before controlled shutdown.
package orchestration

import (
	"context"
	"fmt"
	"slices"
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
	// MaxQuiesceTimeout is the longest drain the server accepts (decision P-09).
	// A longer wait holds a promote, and its HTTP response, open for too long.
	MaxQuiesceTimeout = 30 * time.Minute
	// DefaultQuiescePoll is the cadence for re-checking in-flight runs.
	DefaultQuiescePoll = 2 * time.Second

	// QuiesceInstanceLive is the instance a promote restarts. A run that does not
	// declare its target instance counts as live (decision P-09).
	QuiesceInstanceLive = "live"
	// quiesceTargetReferenceKind and quiesceTargetRelationship identify the work
	// reference a run uses to declare the scenario instance it changes:
	// {kind: scenario-instance, id: <scenario>@<instance>, relationship: targets}.
	quiesceTargetReferenceKind = "scenario-instance"
	quiesceTargetRelationship  = "targets"
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

	// ExcludeRunID removes one run from the drain set: typically the promoting
	// run, or a run the caller knows does not use the instance being restarted.
	// An excluded active run is reported in QuiesceResult.Excluded. Only a
	// promote of agent-manager itself rejects it, because that restart ends the
	// excluded run's owner.
	ExcludeRunID *uuid.UUID

	// Instance is the scenario instance being restarted. Empty means live. Runs
	// that declare a different instance of the scenario are reported in
	// QuiesceResult.NotDrained instead of being drained; undeclared runs count
	// as live. The request field that sets it arrives with the wave-2 proto.
	Instance string

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
	// Instances are the instances of the scenario the run declares it targets;
	// empty means undeclared (counted as live).
	Instances []string `json:"instances,omitempty"`
}

// QuiesceResult reports the outcome of a promote-quiesce drain.
type QuiesceResult struct {
	Scenario  string          `json:"scenario"`
	Drained   bool            `json:"drained"`             // scenario is now quiet (no in-flight runs)
	Aborted   bool            `json:"aborted"`             // timed out without Force; in-flight work left untouched
	Initial   int             `json:"initial"`             // in-flight count when the drain started (after exclusion)
	InFlight  []QuiesceRunRef `json:"inFlight,omitempty"`  // runs still active at the end (abort case)
	Cancelled []QuiesceRunRef `json:"cancelled,omitempty"` // runs force-cancelled
	// Excluded lists the active run removed by ExcludeRunID. It was not waited
	// on and keeps running through the restart.
	Excluded []QuiesceRunRef `json:"excluded,omitempty"`
	// NotDrained lists active runs that declare another instance of the
	// scenario (for example a shadow), so the restart does not affect them.
	NotDrained []QuiesceRunRef `json:"notDrained,omitempty"`
	WaitedMs   int64           `json:"waitedMs"`
	Reason     string          `json:"reason"` // human guidance / next action
}

// quiesceActiveStatuses are the run states that hold a live OS process executing
// in the scenario's working tree — the states a promote restart must not
// interrupt. pending (queued, no process yet), needs_review (paused, process
// already exited), and parked (suspended on external async work, process exited)
// are intentionally excluded: none is actively writing, so none blocks a safe
// re-point + restart. A parked run will simply wake later against the promoted
// tree. CanStopRun additionally permits stopping parked runs (no process), so if
// a future policy wants to wake-to-cancel parked runs during a drain it can,
// without changing this active-set definition.
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
//   - Default (Force=false): wait up to Timeout (at most MaxQuiesceTimeout); on
//     timeout, ABORT and report the in-flight runs without touching them
//     (promote is re-runnable).
//   - Force=true: on timeout, cancel survivors via the graceful-first StopRun.
//   - Exclusion: ExcludeRunID is removed from the drain set and reported in
//     Excluded. Only agent-manager's own promote rejects an active excluded run,
//     because restarting agent-manager ends that run's owner.
//   - Instances: runs declaring another instance of the scenario are reported in
//     NotDrained; undeclared runs count as live.
func (o *Orchestrator) QuiesceScenario(ctx context.Context, opts QuiesceOptions) (*QuiesceResult, error) {
	scenario := strings.TrimSpace(opts.Scenario)
	if scenario == "" {
		return nil, domain.NewValidationError("scenario", "scenario is required")
	}
	if opts.Timeout < 0 {
		return nil, domain.NewValidationError("timeout", "timeout must be a positive duration")
	}
	if opts.Timeout > MaxQuiesceTimeout {
		return nil, domain.NewValidationErrorWithHint(
			"timeout",
			fmt.Sprintf("timeout %s exceeds the %s maximum", opts.Timeout, MaxQuiesceTimeout),
			"retry with a shorter --timeout; a promote that cannot drain in 30 minutes should abort and retry later",
		)
	}
	scopePrefix := strings.TrimRight(strings.TrimSpace(opts.ScopePrefix), "/")
	if scopePrefix == "" {
		scopePrefix = "scenarios/" + scenario
	}
	tagPrefix := strings.TrimSpace(opts.TagPrefix)
	instance := strings.TrimSpace(opts.Instance)
	if instance == "" {
		instance = QuiesceInstanceLive
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = DefaultQuiesceTimeout
	}
	poll := opts.PollInterval
	if poll <= 0 {
		poll = DefaultQuiescePoll
	}

	result := &QuiesceResult{Scenario: scenario}
	drainSet := func() ([]QuiesceRunRef, error) {
		refs, err := o.activeRunsForScenario(ctx, scenario, scopePrefix, tagPrefix)
		if err != nil {
			return nil, err
		}
		drain, _ := splitQuiesceInstance(excludeRun(refs, opts.ExcludeRunID), instance)
		return drain, nil
	}

	initial, err := o.activeRunsForScenario(ctx, scenario, scopePrefix, tagPrefix)
	if err != nil {
		return nil, err
	}
	if opts.ExcludeRunID != nil {
		if ref, isMember := findRunRef(initial, *opts.ExcludeRunID); isMember {
			if selfidentity.Is(scenario) {
				return nil, domain.NewValidationErrorWithHint(
					"exclude_run_id",
					fmt.Sprintf("cannot promote %q while run %s is active against it: restarting agent-manager ends the owner of every run it manages, including that one", scenario, ref.ID),
					"run the agent-manager promote from an operator session",
				)
			}
			result.Excluded = []QuiesceRunRef{ref}
		}
	}
	initialDrain, notDrained := splitQuiesceInstance(excludeRun(initial, opts.ExcludeRunID), instance)
	result.NotDrained = notDrained
	result.Initial = len(initialDrain)
	if len(initialDrain) == 0 {
		result.Drained = true
		result.Reason = fmt.Sprintf("no in-flight runs target %q — safe to promote", scenario) + quiesceSetAsideNote(result, instance)
		return result, nil
	}

	start := o.now()
	deadline := start.Add(timeout)
	for {
		remaining, err := drainSet()
		if err != nil {
			return nil, err
		}
		if len(remaining) == 0 {
			result.Drained = true
			result.WaitedMs = time.Since(start).Milliseconds()
			result.Reason = fmt.Sprintf("%q drained — safe to promote", scenario) + quiesceSetAsideNote(result, instance)
			return result, nil
		}

		if !o.now().Before(deadline) {
			result.WaitedMs = o.now().Sub(start).Milliseconds()
			if opts.Force {
				return o.forceCancel(ctx, result, remaining, scenario, instance, drainSet), nil
			}
			// Default: abort, never destroy others' in-flight work.
			result.Aborted = true
			result.InFlight = remaining
			result.Reason = fmt.Sprintf(
				"%d run(s) still in-flight against %q after %s; retry once they finish, or pass --force to cancel them",
				len(remaining), scenario, timeout,
			) + quiesceSetAsideNote(result, instance)
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
	scenario, instance string,
	drainSet func() ([]QuiesceRunRef, error),
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

	if after, err := drainSet(); err == nil {
		result.InFlight = append(result.InFlight, after...)
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
	result.Reason += quiesceSetAsideNote(result, instance)
	return result
}

// quiesceSetAsideNote names the active runs the drain deliberately did not wait
// on, so a caller never mistakes an excluded or other-instance run for a gap.
func quiesceSetAsideNote(result *QuiesceResult, instance string) string {
	var note strings.Builder
	for _, ref := range result.Excluded {
		fmt.Fprintf(&note, "; excluded run %s [%s] was not waited on and keeps running", ref.ID, ref.Status)
	}
	for _, ref := range result.NotDrained {
		fmt.Fprintf(&note, "; run %s [%s] declares instance %s, not %s, and was not drained", ref.ID, ref.Status, strings.Join(ref.Instances, ","), instance)
	}
	return note.String()
}

// splitQuiesceInstance separates runs that use the instance being restarted
// (declared, or undeclared and the instance is live) from runs that declare
// only other instances of the scenario.
func splitQuiesceInstance(refs []QuiesceRunRef, instance string) (drain, other []QuiesceRunRef) {
	for _, ref := range refs {
		if len(ref.Instances) == 0 {
			if instance == QuiesceInstanceLive {
				drain = append(drain, ref)
			} else {
				other = append(other, ref)
			}
			continue
		}
		if slices.Contains(ref.Instances, instance) {
			drain = append(drain, ref)
		} else {
			other = append(other, ref)
		}
	}
	return drain, other
}

// declaredQuiesceInstances returns the instances of scenario that a run declares
// it targets through {kind: scenario-instance, id: <scenario>@<instance>,
// relationship: targets} work references.
func declaredQuiesceInstances(run *domain.Run, scenario string) []string {
	var instances []string
	for _, ref := range run.WorkReferences {
		if ref.GetKind() != quiesceTargetReferenceKind || ref.GetRelationship() != quiesceTargetRelationship {
			continue
		}
		name, instance, ok := strings.Cut(ref.GetId(), "@")
		if !ok || strings.TrimSpace(name) != scenario || strings.TrimSpace(instance) == "" {
			continue
		}
		if instance = strings.TrimSpace(instance); !slices.Contains(instances, instance) {
			instances = append(instances, instance)
		}
	}
	return instances
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
			refs = append(refs, QuiesceRunRef{ID: run.ID.String(), Tag: run.Tag, Status: string(run.Status), ScopePath: scope, Instances: declaredQuiesceInstances(run, scenario)})
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
			refs = append(refs, QuiesceRunRef{ID: run.ID.String(), Tag: run.Tag, Status: string(run.Status), Instances: declaredQuiesceInstances(run, scenario)})
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
