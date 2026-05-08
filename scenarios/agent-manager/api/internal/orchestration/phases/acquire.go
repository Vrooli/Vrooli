// Runner acquisition + fallback chain walk.
//
// AcquireRunner returns a runnable runner.Runner for this run, walking
// the fallback chain (configured via Run.ResolvedConfig.FallbackRunnerTypes)
// when the primary runner is missing or unavailable. Returns a typed
// *domain.RunnerError so the executor can route to failWithError without
// re-classifying.

package phases

import (
	"context"
	"errors"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/eventlog"
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
		return runnerAcquireFailure(ctx, in, runnerType, "availability_check",
			errors.New(msg), msg, true)
	}

	return runnerAcquireFailure(ctx, in, runnerType, "acquire",
		getErr, getErr.Error(), false)
}

// runnerAcquireFailure attempts the configured fallback walk and, on
// success, returns the selected runner. On exhaustion it builds a typed
// *domain.RunnerError with Alternative populated so operators see an
// actionable suggestion.
func runnerAcquireFailure(ctx context.Context, in AcquireRunnerInput, primary domain.RunnerType,
	op string, cause error, primaryReason string, transient bool,
) (runner.Runner, error) {
	sel := selectFallbackRunner(ctx, in.Runners, in.Run, primary)
	if sel != nil && sel.runner != nil {
		applyRunnerFallback(ctx, in, primary, sel.runnerType, sel.attemptNo, primaryReason)
		return sel.runner, nil
	}
	if sel != nil && len(sel.tried) > 0 && in.Run != nil {
		EmitRunnerFallbackExhausted(ctx, in.Deps, in.Run.ID, eventlog.RunnerFallbackExhaustedPayload{
			Primary:         string(primary),
			CandidatesTried: sel.tried,
			LastReason:      coalesceReason(sel.lastReason, primaryReason),
		})
	}
	return nil, &domain.RunnerError{
		RunnerType:  primary,
		Operation:   op,
		Cause:       cause,
		IsTransient: transient,
		Alternative: lastResortAlternative(in.Runners, primary),
	}
}

// GetRunnerType returns the runner type, preferring profile but falling
// back to resolved config and finally to ClaudeCode.
func GetRunnerType(run *domain.Run, profile *domain.AgentProfile) domain.RunnerType {
	if profile != nil {
		return profile.RunnerType
	}
	if run != nil && run.ResolvedConfig != nil {
		return run.ResolvedConfig.RunnerType
	}
	return domain.RunnerTypeClaudeCode
}

// runnerFallbackCandidates returns the deduplicated, validated fallback
// chain for a given primary runner.
func runnerFallbackCandidates(run *domain.Run, primary domain.RunnerType) []domain.RunnerType {
	if run == nil || run.ResolvedConfig == nil || len(run.ResolvedConfig.FallbackRunnerTypes) == 0 {
		return nil
	}
	seen := make(map[domain.RunnerType]struct{}, len(run.ResolvedConfig.FallbackRunnerTypes))
	candidates := make([]domain.RunnerType, 0, len(run.ResolvedConfig.FallbackRunnerTypes))
	for _, rt := range run.ResolvedConfig.FallbackRunnerTypes {
		if !rt.IsValid() || rt == primary {
			continue
		}
		if _, exists := seen[rt]; exists {
			continue
		}
		seen[rt] = struct{}{}
		candidates = append(candidates, rt)
	}
	return candidates
}

// runnerSelection is the result of a single fallback chain walk.
// runner is non-nil when an available candidate was found; otherwise
// tried/lastReason carry the walk's audit trail for the exhausted event.
type runnerSelection struct {
	runner     runner.Runner
	runnerType domain.RunnerType
	attemptNo  int
	tried      []string
	lastReason string
}

// selectFallbackRunner walks the configured fallback chain returning
// the first available runner. The returned *runnerSelection is always
// non-nil when there is at least one configured candidate (so callers
// can read tried/lastReason for the exhausted event); it is nil when
// no fallback chain is configured at all.
//
// This is the single fallback walker — replaces the prior trio of
// FindAlternativeRunner / findFallbackAlternative / tryFallbackRunner.
func selectFallbackRunner(ctx context.Context, registry runner.Registry, run *domain.Run, primary domain.RunnerType) *runnerSelection {
	if registry == nil {
		return nil
	}
	candidates := runnerFallbackCandidates(run, primary)
	if len(candidates) == 0 {
		return nil
	}
	sel := &runnerSelection{tried: make([]string, 0, len(candidates))}
	for _, rt := range candidates {
		sel.attemptNo++
		sel.tried = append(sel.tried, string(rt))
		r, err := registry.Get(rt)
		if err != nil {
			sel.lastReason = err.Error()
			continue
		}
		available, msg := r.IsAvailable(ctx)
		if !available {
			sel.lastReason = msg
			continue
		}
		sel.runner = r
		sel.runnerType = rt
		return sel
	}
	return sel
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

func coalesceReason(last, primary string) string {
	if last != "" {
		return last
	}
	return primary
}

func applyRunnerFallback(ctx context.Context, in AcquireRunnerInput, from, to domain.RunnerType, attemptNo int, reason string) {
	if in.Run == nil {
		return
	}
	if in.Run.ResolvedConfig == nil {
		in.Run.ResolvedConfig = domain.DefaultRunConfig()
	}
	in.Run.ResolvedConfig.RunnerType = to
	in.Run.UpdatedAt = time.Now()
	if in.Deps.Runs != nil {
		if err := in.Deps.Runs.Update(ctx, in.Run); err != nil {
			EmitSystemEvent(ctx, in.Deps, in.Run.ID, "warn",
				"failed to persist runner fallback: "+err.Error())
		}
	}
	EmitRunnerFallbackAttempted(ctx, in.Deps, in.Run.ID, eventlog.RunnerFallbackAttemptedPayload{
		From:      string(from),
		To:        string(to),
		Reason:    reason,
		AttemptNo: attemptNo,
	})
}
