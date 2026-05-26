package baseline

import (
	"context"
	"testing"

	"github.com/vrooli/api-core/storage"

	"git-control-tower/internal/git"
)

func newTestSnapshotStore(t *testing.T) *SnapshotStore {
	t.Helper()
	resolver, err := storage.NewResolver(storage.ResolverConfig{AppID: "vrooli"})
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}
	return NewSnapshotStoreAt(resolver, t.TempDir())
}

// fakeExecutor returns a canned ExecResult and records the phases/diagnostics
// it was asked to run.
type fakeExecutor struct {
	result   ExecResult
	err      error
	calls    int
	lastDiag string
	lastPh   []string
}

func (f *fakeExecutor) Execute(_ context.Context, _ string, phases []string, diag string) (ExecResult, error) {
	f.calls++
	f.lastPh = phases
	f.lastDiag = diag
	if f.err != nil {
		return ExecResult{}, f.err
	}
	// Synthesize a unique runID per call so capture vs diff differ.
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

// fakeAuditor returns a queue of scan results (capture, then diff).
type fakeAuditor struct {
	results [][]Issue
	idx     int
	err     error
}

func (f *fakeAuditor) Scan(_ context.Context, _, _ string) ([]Issue, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.idx >= len(f.results) {
		return nil, nil
	}
	r := f.results[f.idx]
	f.idx++
	return r, nil
}

type fakeVisual struct {
	capture []VisualSnapshot
	idx     int
	getSnap VisualSnapshot
	getOK   bool
}

func (f *fakeVisual) Capture(_ context.Context, _ int64, _, _ string) (VisualSnapshot, error) {
	if f.idx >= len(f.capture) {
		return VisualSnapshot{}, nil
	}
	r := f.capture[f.idx]
	f.idx++
	return r, nil
}

func (f *fakeVisual) Get(_ context.Context, _ int64, _, _ string) (VisualSnapshot, bool, error) {
	return f.getSnap, f.getOK, nil
}

// fixedGit returns a CaptureGit func yielding a fixed state.
func fixedGit(st git.State) func(context.Context, string) (git.State, error) {
	return func(context.Context, string) (git.State, error) { return st, nil }
}
