package lifecycle

import (
	"errors"
	"time"
)

// This file is the single wait/poll primitive for scenario lifecycle
// orchestration (docs/plans/scenario-lifecycle-start-wait-contract-plan.md,
// Phase 1). Every "poll until condition or deadline" site in
// internal/lifecycle and internal/orchestrator goes through Await with a row
// from the policy table below; bespoke for/sleep loops are prohibited (they
// drift on timeout semantics and are invisible to tests).

// ErrAwaitExpired is returned by Await when the policy bound (deadline or
// attempt budget) is exhausted before the condition reports done. Callers
// shape their own domain error/result from the state their condition closure
// recorded; the sentinel exists so that shaping is explicit, never inferred
// from message text.
var ErrAwaitExpired = errors.New("await: condition not met before policy bound")

// AwaitPolicy is one row of the lifecycle timeout-policy table: how long to
// keep re-evaluating a condition and how often. Exactly one of Timeout or
// MaxAttempts is normally set; setting both bounds by whichever trips first.
type AwaitPolicy struct {
	// Timeout bounds the wait by wall clock. The deadline is anchored at the
	// first evaluation (Await entry), not per attempt. Zero means "no time
	// bound" (MaxAttempts must then be set).
	Timeout time.Duration
	// Interval is the base sleep between evaluations.
	Interval time.Duration
	// MaxInterval, when > Interval, enables capped exponential backoff: the
	// sleep doubles after each evaluation up to MaxInterval. Zero keeps a
	// fixed interval.
	MaxInterval time.Duration
	// MaxAttempts bounds the wait by evaluation count. Zero means "no count
	// bound" (Timeout must then be set).
	MaxAttempts int
	// ExpireStrictlyAfter selects the deadline comparison. False (default)
	// expires when now >= deadline; true expires only when now > deadline, so
	// an evaluation landing exactly on the deadline gets one more sleep+retry.
	// WaitForHealth has always used the strict form; changing it would shift
	// its degraded-grace boundary by one tick.
	ExpireStrictlyAfter bool
}

// AwaitClock is the injectable time seam for Await. Production passes the
// runner's lifecycleDeps now/sleep pair; tests pass a fake that records
// sleeps and advances virtual time.
type AwaitClock struct {
	Now   func() time.Time
	Sleep func(time.Duration)
}

// Await evaluates cond once per tick until it reports done, returns an error,
// or the policy bound trips. Contract:
//
//   - cond is ALWAYS evaluated at least once, immediately (no leading sleep).
//   - cond returning (true, nil) ends the wait with nil.
//   - cond returning a non-nil error ends the wait immediately with that
//     error (fatal). Transient errors a caller wants to retry through are
//     handled INSIDE the condition closure (record and return false, nil).
//   - Bound checks happen after each evaluation, before the sleep, so the
//     bound never eats an evaluation.
//   - On bound expiry Await returns ErrAwaitExpired; the condition's recorded
//     state tells the caller what the world looked like at the last look.
func Await(clock AwaitClock, policy AwaitPolicy, cond func() (bool, error)) error {
	var deadline time.Time
	if policy.Timeout > 0 {
		deadline = clock.Now().Add(policy.Timeout)
	}
	interval := policy.Interval
	for attempt := 1; ; attempt++ {
		done, err := cond()
		if err != nil {
			return err
		}
		if done {
			return nil
		}
		if policy.MaxAttempts > 0 && attempt >= policy.MaxAttempts {
			return ErrAwaitExpired
		}
		if policy.Timeout > 0 {
			now := clock.Now()
			if policy.ExpireStrictlyAfter {
				if now.After(deadline) {
					return ErrAwaitExpired
				}
			} else if !now.Before(deadline) {
				return ErrAwaitExpired
			}
		}
		clock.Sleep(interval)
		if policy.MaxInterval > policy.Interval {
			interval *= 2
			if interval > policy.MaxInterval {
				interval = policy.MaxInterval
			}
		}
	}
}

// awaitClock returns the runner's injectable clock pair for Await calls.
func (r *Runner) awaitClock() AwaitClock {
	deps := r.runtimeDeps()
	return AwaitClock{Now: deps.now, Sleep: deps.sleep}
}

// Lifecycle timeout-policy table. Every wait site's bounds live here so
// tuning is a one-line diff and the values are visible in one place.
// WaitForHealth's Timeout/Interval rows are defaults the manifest may
// override (see healthAwaitPolicy).
var (
	// healthWaitDefaultPolicy bounds WaitForHealth when the manifest declares
	// health checks but no explicit timeout/interval. Strict-after expiry is
	// load-bearing: the degraded-grace decision happens one tick past an
	// exact-deadline evaluation.
	healthWaitDefaultPolicy = AwaitPolicy{
		Timeout:             30 * time.Second,
		Interval:            500 * time.Millisecond,
		ExpireStrictlyAfter: true,
	}
	// healthWaitMaxInterval caps manifest-declared poll intervals so a huge
	// interval cannot starve the health deadline of evaluations.
	healthWaitMaxInterval = 2 * time.Second
	// registryHealthRetryPolicy is the bounded data-plane probe retry used
	// before condemning a registry-authoritative running instance: a single
	// dropped probe must not trigger a stop+rebuild+restart.
	registryHealthRetryPolicy = AwaitPolicy{
		MaxAttempts: 3,
		Interval:    1 * time.Second,
	}
	// resourceReadyPolicy bounds the post-start readiness wait for a resource
	// dependency.
	resourceReadyPolicy = AwaitPolicy{
		Timeout:  30 * time.Second,
		Interval: 500 * time.Millisecond,
	}
	// dependencyLockPolicy bounds how long a start waits for a transitive
	// dependency's lifecycle lock held by a concurrent invocation.
	dependencyLockPolicy = AwaitPolicy{
		Timeout:  2 * time.Minute,
		Interval: 500 * time.Millisecond,
	}
	// SandboxStartPolicy bounds the orchestrator's wait for a scenario to
	// report running after a sandbox host-lifecycle proxy start. Exported for
	// internal/orchestrator, which owns no wait loops of its own.
	SandboxStartPolicy = AwaitPolicy{
		Timeout:             30 * time.Second,
		Interval:            500 * time.Millisecond,
		ExpireStrictlyAfter: true,
	}
)

// stopSettleDelay is the pause between stopping an instance and starting its
// replacement. Stop verifies fixed-port release with a condition wait
// (verifyPortsReleased), but dynamically allocated ports, process-record
// removal, and registry claim release settle asynchronously after the kill
// signal; an immediate restart can race a dying process's lingering sockets
// or claims. A condition wait would need per-port probes for ports that are
// not known until the new environment is built, so a short named delay is
// the honest trade-off.
const stopSettleDelay = 1 * time.Second

// healthAwaitPolicy resolves the effective WaitForHealth policy for a
// manifest health config: defaults from the policy table, overridden by the
// manifest's explicit timeout/interval, with the interval capped.
func healthAwaitPolicy(timeoutMillis, intervalMillis int) AwaitPolicy {
	policy := healthWaitDefaultPolicy
	if timeoutMillis > 0 {
		policy.Timeout = time.Duration(timeoutMillis) * time.Millisecond
	}
	if intervalMillis > 0 {
		policy.Interval = time.Duration(intervalMillis) * time.Millisecond
		if policy.Interval > healthWaitMaxInterval {
			policy.Interval = healthWaitMaxInterval
		}
	}
	return policy
}
