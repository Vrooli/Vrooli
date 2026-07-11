// Runner acquisition for historical snapshot-less runs.
//
// AcquireRunner returns the single configured runner. Policy-backed runs use
// ExecuteWithModelFallback and their immutable candidate snapshot instead.
// Returns a typed
// *domain.RunnerError so the executor can route to failWithError without
// re-classifying.

package phases

import (
	"context"
	"errors"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
)

// AcquireRunnerInput is the explicit input to AcquireRunner.
type AcquireRunnerInput struct {
	Deps    Deps
	Run     *domain.Run
	Profile *domain.AgentProfile
	Runners runner.Registry
}

// AcquireRunner selects the runner for this run and verifies it is
// available. On failure, returns a typed *domain.RunnerError carrying
// the alternative runner name (if any) so the operator-facing error is
// actionable.
//
// Emits runner.fallback.attempted on the successful walk step and
// runner.fallback.exhausted when candidates were tried but none became
// available. The walk classifies each rejection via fallback.Reason so
// the persisted health store and stats engine see structured outcomes
// instead of freeform strings.
func AcquireRunner(ctx context.Context, in AcquireRunnerInput) (runner.Runner, error) {
	runnerType := GetRunnerType(in.Run, in.Profile)

	if in.Runners == nil {
		return nil, &domain.RunnerError{
			RunnerType:  runnerType,
			Operation:   "acquire",
			Cause:       domain.NewConfigMissingError("runnerRegistry", "not configured", nil),
			IsTransient: false,
		}
	}

	r, getErr := in.Runners.Get(runnerType)
	if getErr == nil {
		available, msg := r.IsAvailable(ctx)
		if available {
			return r, nil
		}
		return nil, &domain.RunnerError{
			RunnerType: runnerType, Operation: "availability_check", Cause: errors.New(msg),
			IsTransient: true, Alternative: lastResortAlternative(in.Runners, runnerType),
		}
	}

	return nil, &domain.RunnerError{
		RunnerType: runnerType, Operation: "acquire", Cause: getErr,
		IsTransient: false, Alternative: lastResortAlternative(in.Runners, runnerType),
	}
}

// GetRunnerType returns the run-owned resolved runner before consulting the
// source profile. Policy preflight may select a later cross-runner candidate,
// so the profile is historical input rather than execution authority.
func GetRunnerType(run *domain.Run, profile *domain.AgentProfile) domain.RunnerType {
	if run != nil && run.ResolvedConfig != nil {
		return run.ResolvedConfig.RunnerType
	}
	return domain.RunnerTypeClaudeCode
}

// lastResortAlternative returns the name of any other registered,
// available runner so RunnerError.Alternative carries an actionable
// suggestion even when the primary's configured chain was exhausted
// (or empty). Returns "" when no alternative is available.
func lastResortAlternative(registry runner.Registry, current domain.RunnerType) string {
	if registry == nil {
		return ""
	}
	candidates := []domain.RunnerType{
		domain.RunnerTypeClaudeCode,
		domain.RunnerTypeCodex,
		domain.RunnerTypeOpenCode,
	}
	for _, rt := range candidates {
		if rt == current {
			continue
		}
		r, err := registry.Get(rt)
		if err != nil {
			continue
		}
		if available, _ := r.IsAvailable(context.Background()); available {
			return string(rt)
		}
	}
	return ""
}
