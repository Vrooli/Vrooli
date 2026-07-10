package baseline

import (
	"context"

	"git-control-tower/internal/git"
)

// fakeExecutor returns a canned ExecResult and records how many comprehensive
// runs it was asked to start. Each StartRun synthesizes a unique runID so a
// capture run and a diff's current run differ; AwaitResult replays the canned
// result (or err) for that runID.
type fakeExecutor struct {
	result   ExecResult
	err      error // AwaitResult error (the run failed)
	startErr error // StartRun error (could not start the run)
	calls    int
	// reusable / reusableHit / findErr script FindReusableRun (clean-tree reuse).
	reusable    ReusableRun
	reusableHit bool
	findErr     error
	// statusInfo, when set, scripts RunStatus; otherwise a terminal-passed
	// snapshot is returned.
	statusInfo *RunStatusInfo
}

func (f *fakeExecutor) StartRun(_ context.Context, _ string) (RunHandle, error) {
	if f.startErr != nil {
		return RunHandle{}, f.startErr
	}
	f.calls++
	return RunHandle{RunID: "run-" + itoa(f.calls), EstimatedTotalSeconds: 60, EtaKnown: true}, nil
}

func (f *fakeExecutor) AwaitResult(_ context.Context, _, runID string) (ExecResult, error) {
	if f.err != nil {
		return ExecResult{}, f.err
	}
	r := f.result
	r.RunID = runID
	return r, nil
}

func (f *fakeExecutor) RunStatus(_ context.Context, _, _ string) (RunStatusInfo, error) {
	if f.statusInfo != nil {
		return *f.statusInfo, nil
	}
	return RunStatusInfo{Status: "passed", Terminal: true, Success: true}, nil
}

func (f *fakeExecutor) FindReusableRun(_ context.Context, _, _ string) (ReusableRun, bool, error) {
	if f.findErr != nil {
		return ReusableRun{}, false, f.findErr
	}
	return f.reusable, f.reusableHit, nil
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	return string(b)
}

type pinCall struct{ scenario, runID, by, reason string }

type fakeRuns struct {
	pins       []pinCall
	unpins     []pinCall
	compare    CompareResult
	compareErr error
	catalogs   map[string]ArtifactCatalog
	catalogErr error
	// visualDeltas is the canned CompareRunVisuals result; compareVisualsErr
	// forces an error.
	visualDeltas     []VisualDelta
	compareVisualErr error
}

func (f *fakeRuns) PinRun(_ context.Context, scenario, runID, by, reason string) error {
	f.pins = append(f.pins, pinCall{scenario, runID, by, reason})
	return nil
}

func (f *fakeRuns) UnpinRun(_ context.Context, scenario, runID, by string) error {
	f.unpins = append(f.unpins, pinCall{scenario, runID, by, ""})
	return nil
}

func (f *fakeRuns) CompareRuns(_ context.Context, _, _, _, _ string) (CompareResult, error) {
	return f.compare, f.compareErr
}

func (f *fakeRuns) ListRunArtifacts(_ context.Context, _, runID string) (ArtifactCatalog, error) {
	if f.catalogErr != nil {
		return ArtifactCatalog{}, f.catalogErr
	}
	catalog := f.catalogs[runID]
	catalog.RunID = runID
	return catalog, nil
}

func (f *fakeRuns) CompareRunVisuals(_ context.Context, _, _, _ string) ([]VisualDelta, error) {
	if f.compareVisualErr != nil {
		return nil, f.compareVisualErr
	}
	return f.visualDeltas, nil
}

// fakeReachability returns a canned reachability result and records how many
// times it was probed.
type fakeReachability struct {
	err    error
	probes int
}

func (f *fakeReachability) Probe(_ context.Context) error {
	f.probes++
	return f.err
}

// fixedGit returns a CaptureGit func yielding a fixed state.
func fixedGit(st git.State) func(context.Context, string) (git.State, error) {
	return func(context.Context, string) (git.State, error) { return st, nil }
}
