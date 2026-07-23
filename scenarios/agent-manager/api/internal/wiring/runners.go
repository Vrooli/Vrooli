// Package wiring owns production composition of Agent Manager services.
package wiring

import (
	"context"
	"fmt"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/runner/codecs"
	runnercore "agent-manager/internal/adapters/runner/core"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/obs"
)

// Runners contains the production registry and concrete runners that need
// follow-up composition (for example protected-sandbox launchers).
type Runners struct {
	Registry *runner.DefaultRegistry
	Claude   *runnercore.Runner
	Codex    *runnercore.Runner
	OpenCode *runnercore.Runner
	Grok     *runnercore.Runner
}

// NewRunners registers every supported coding-agent runner. Codec creation
// failures become unavailable stub runners so the API can start and accurately
// report the missing runtime instead of failing bootstrap.
func NewRunners() Runners {
	registry := runner.NewRegistry()
	hostLauncher := runner.NewHostLauncher()
	result := Runners{Registry: registry}
	register := func(name string, runnerType domain.RunnerType, build func() (*runnercore.Runner, error), target **runnercore.Runner) {
		built, err := build()
		if err != nil {
			obs.Logger().Warn(name+" codec construction failed", obs.KeyRunnerType, string(runnerType), obs.KeyError, err.Error())
			if err := registry.Register(runner.NewStubRunner(runnerType, fmt.Sprintf("%s runner failed to initialize: %v", name, err))); err != nil {
				obs.Logger().Warn("stub "+name+" runner registration failed", obs.KeyRunnerType, string(runnerType), obs.KeyError, err.Error())
			}
			return
		}
		*target = built
		if err := registry.Register(built); err != nil {
			obs.Logger().Warn(name+" runner registration failed", obs.KeyRunnerType, string(runnerType), obs.KeyError, err.Error())
		}
		if available, message := built.IsAvailable(context.Background()); available {
			obs.Logger().Info("runner available", obs.KeyRunnerType, string(runnerType))
		} else {
			obs.Logger().Warn("runner unavailable", obs.KeyRunnerType, string(runnerType), obs.KeyMessage, message)
		}
	}
	register("Claude Code", domain.RunnerTypeClaudeCode, func() (*runnercore.Runner, error) {
		codec, err := codecs.NewClaude()
		if err != nil {
			return nil, err
		}
		return runnercore.NewRunner(codec, hostLauncher, nil), nil
	}, &result.Claude)
	register("Codex", domain.RunnerTypeCodex, func() (*runnercore.Runner, error) {
		codec, err := codecs.NewCodex()
		if err != nil {
			return nil, err
		}
		return runnercore.NewRunner(codec, hostLauncher, nil), nil
	}, &result.Codex)
	register("OpenCode", domain.RunnerTypeOpenCode, func() (*runnercore.Runner, error) {
		codec, err := codecs.NewOpenCode()
		if err != nil {
			return nil, err
		}
		return runnercore.NewRunner(codec, hostLauncher, nil), nil
	}, &result.OpenCode)
	register("Grok", domain.RunnerTypeGrok, func() (*runnercore.Runner, error) {
		codec, err := codecs.NewGrok()
		if err != nil {
			return nil, err
		}
		return runnercore.NewRunner(codec, hostLauncher, nil), nil
	}, &result.Grok)
	return result
}
