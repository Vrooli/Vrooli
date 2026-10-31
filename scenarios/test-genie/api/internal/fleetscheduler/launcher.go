package fleetscheduler

import (
	"context"
	"errors"
	"strings"

	"test-genie/internal/execution"
	"test-genie/internal/orchestrator"
	"test-genie/internal/runmanager"
)

// runStarter is the slice of runmanager.Manager the launcher needs. Keeping it
// narrow makes the launcher trivially fakeable and documents the exact coupling:
// admit a run (honoring the per-scenario invariant) and block until it ends.
type runStarter interface {
	Start(opts runmanager.StartOptions) (runmanager.StartResult, error)
	Wait(ctx context.Context, scenario, runID string) (runmanager.LiveStatus, error)
}

// managerLauncher drives full suites through the durable run manager. The
// manager owns the one-in-progress-per-scenario invariant: a divergent in-flight
// run yields *runmanager.BusyError, which the launcher maps to ErrScenarioBusy
// so the scheduler skips the scenario this cycle. An identical in-flight run
// coalesces (no second suite) and the launcher simply awaits it.
type managerLauncher struct {
	mgr    runStarter
	preset string
}

// NewManagerLauncher wraps the run manager. preset selects the suite shape the
// scheduler cycles ("comprehensive" for full fleet coverage); empty uses the
// orchestrator default.
func NewManagerLauncher(mgr runStarter, preset string) Launcher {
	return &managerLauncher{mgr: mgr, preset: strings.TrimSpace(preset)}
}

func (l *managerLauncher) Launch(ctx context.Context, scenario string) (string, error) {
	res, err := l.mgr.Start(runmanager.StartOptions{
		Input: execution.SuiteExecutionInput{
			Request: orchestrator.SuiteExecutionRequest{
				ScenarioName: strings.TrimSpace(scenario),
				Preset:       l.preset,
			},
		},
	})
	if err != nil {
		var busy *runmanager.BusyError
		if errors.As(err, &busy) {
			return "", ErrScenarioBusy
		}
		return "", err
	}
	return res.RunID, nil
}

func (l *managerLauncher) Await(ctx context.Context, scenario, runID string) (string, error) {
	st, err := l.mgr.Wait(ctx, scenario, runID)
	if err != nil {
		return "", err
	}
	// Prefer the explicit verdict; fall back to the lifecycle status.
	if v := strings.TrimSpace(st.Verdict); v != "" {
		return v, nil
	}
	return strings.TrimSpace(st.Status), nil
}
