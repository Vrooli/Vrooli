package baseline

import (
	"context"

	"git-control-tower/internal/git"
)

// fakeExecutor returns a canned ExecResult and records how many comprehensive
// runs it was asked to trigger. Each call synthesizes a unique runID so a
// capture run and a diff's current run differ.
type fakeExecutor struct {
	result ExecResult
	err    error
	calls  int
}

func (f *fakeExecutor) Execute(_ context.Context, _ string) (ExecResult, error) {
	f.calls++
	if f.err != nil {
		return ExecResult{}, f.err
	}
	r := f.result
	if r.RunID == "" {
		r.RunID = "run"
	}
	r.RunID = r.RunID + "-" + itoa(f.calls)
	return r, nil
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
	// visuals is keyed by runID; a missing key returns an empty slice.
	visuals    map[string][]RunVisual
	visualsErr error
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

func (f *fakeRuns) ListRunVisuals(_ context.Context, _, runID string) ([]RunVisual, error) {
	if f.visualsErr != nil {
		return nil, f.visualsErr
	}
	return f.visuals[runID], nil
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
