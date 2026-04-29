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
	"fmt"
	"time"

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

	r, err := in.Runners.Get(runnerType)
	if err != nil {
		if fb := tryFallbackRunner(ctx, in, runnerType); fb != nil {
			return fb, nil
		}
		alternative := findFallbackAlternative(in.Runners, in.Run, runnerType)
		return nil, &domain.RunnerError{
			RunnerType:  runnerType,
			Operation:   "acquire",
			Cause:       err,
			IsTransient: false,
			Alternative: alternative,
		}
	}

	available, msg := r.IsAvailable(ctx)
	if !available {
		if fb := tryFallbackRunner(ctx, in, runnerType); fb != nil {
			return fb, nil
		}
		alternative := findFallbackAlternative(in.Runners, in.Run, runnerType)
		return nil, &domain.RunnerError{
			RunnerType:  runnerType,
			Operation:   "availability_check",
			Cause:       errors.New(msg),
			IsTransient: true,
			Alternative: alternative,
		}
	}

	return r, nil
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

// FindAlternativeRunner attempts to find another available runner.
// Returns the runner type as a string, or empty string if none available.
func FindAlternativeRunner(registry runner.Registry, current domain.RunnerType) string {
	if registry == nil {
		return ""
	}
	alternatives := []domain.RunnerType{
		domain.RunnerTypeClaudeCode,
		domain.RunnerTypeCodex,
		domain.RunnerTypeOpenCode,
	}
	for _, rt := range alternatives {
		if rt == current {
			continue
		}
		if r, err := registry.Get(rt); err == nil {
			if available, _ := r.IsAvailable(context.Background()); available {
				return string(rt)
			}
		}
	}
	return ""
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

func findFallbackAlternative(registry runner.Registry, run *domain.Run, primary domain.RunnerType) string {
	if registry == nil {
		return ""
	}
	for _, rt := range runnerFallbackCandidates(run, primary) {
		if r, err := registry.Get(rt); err == nil {
			if available, _ := r.IsAvailable(context.Background()); available {
				return string(rt)
			}
		}
	}
	return FindAlternativeRunner(registry, primary)
}

func tryFallbackRunner(ctx context.Context, in AcquireRunnerInput, primary domain.RunnerType) runner.Runner {
	if in.Runners == nil {
		return nil
	}
	for _, rt := range runnerFallbackCandidates(in.Run, primary) {
		r, err := in.Runners.Get(rt)
		if err != nil {
			continue
		}
		available, _ := r.IsAvailable(ctx)
		if !available {
			continue
		}
		applyRunnerFallback(ctx, in, primary, rt)
		return r
	}
	return nil
}

func applyRunnerFallback(ctx context.Context, in AcquireRunnerInput, from, to domain.RunnerType) {
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
	EmitSystemEvent(ctx, in.Deps, in.Run.ID, "warn",
		fmt.Sprintf("runner fallback: %s -> %s", from, to))
}
