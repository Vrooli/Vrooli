package baseline

import (
	"context"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	bl "git-control-tower/internal/baseline"
	"git-control-tower/internal/git"

	"github.com/vrooli/api-core/storage"
	baselinesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/baselines"
	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

type fakeRepos struct{ dir string }

func (f fakeRepos) Resolve(_ context.Context, _ int64) (int64, string, error) { return 1, f.dir, nil }

type recordingExecutor struct {
	cancel func()
	mu     sync.Mutex
	calls  int
}

func (e *recordingExecutor) StartRun(_ context.Context, _ string) (bl.RunHandle, error) {
	e.mu.Lock()
	e.calls++
	id := e.calls
	e.mu.Unlock()
	if e.cancel != nil {
		e.cancel()
	}
	return bl.RunHandle{RunID: "run-" + string(rune('0'+id)), EstimatedTotalSeconds: 60, EtaKnown: true}, nil
}

func (e *recordingExecutor) AwaitResult(_ context.Context, _, runID string) (bl.ExecResult, error) {
	return bl.ExecResult{
		RunID: runID, Success: true, CompletedAt: time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC),
		TreeDigest: "td:tree", PhaseSetDigest: "ps:set", CaptureProfile: bl.CaptureProfile,
		DescriptorSnapshotDigest: "ds:catalog", DescriptorSnapshotSchemaVersion: 1,
		Phases: []bl.PhaseStatus{{Name: "unit", Status: "passed"}},
	}, nil
}

func (e *recordingExecutor) RunStatus(_ context.Context, _, _ string) (bl.RunStatusInfo, error) {
	return bl.RunStatusInfo{Status: "passed", Terminal: true, Success: true}, nil
}

func (e *recordingExecutor) FindReusableRun(_ context.Context, _, _ string) (bl.ReusableRun, bool, error) {
	return bl.ReusableRun{}, false, nil
}

type recordingRuns struct {
	mu      sync.Mutex
	pins    int
	unpins  int
	compare bl.CompareResult
}

func (r *recordingRuns) PinRun(_ context.Context, _, _, _, _ string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pins++
	return nil
}
func (r *recordingRuns) UnpinRun(_ context.Context, _, _, _ string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.unpins++
	return nil
}
func (r *recordingRuns) CompareRuns(_ context.Context, _, _, _, _ string) (bl.CompareResult, error) {
	return r.compare, nil
}
func (r *recordingRuns) ListRunArtifacts(_ context.Context, _, runID string) (bl.ArtifactCatalog, error) {
	return bl.ArtifactCatalog{RunID: runID, SchemaVersion: 1, Digest: "catalog-" + runID, Artifacts: []*runspb.ArtifactRef{{Id: "opaque-" + runID, Kind: "application/json"}}}, nil
}
func (r *recordingRuns) CompareRunVisuals(_ context.Context, _, _, _ string) ([]bl.VisualDelta, error) {
	return []bl.VisualDelta{{Page: "/", Status: "identical"}}, nil
}

func newServerDeps(t *testing.T, exec bl.Executor, runs bl.RunsClient) (*Server, *bl.Service) {
	t.Helper()
	resolver, err := storage.NewResolver(storage.ResolverConfig{AppID: "vrooli"})
	if err != nil {
		t.Fatal(err)
	}
	svc := bl.NewService(bl.Deps{
		Storage: bl.NewStorageAt(resolver, t.TempDir()), Exec: exec, Runs: runs,
		CaptureGit: func(context.Context, string) (git.State, error) { return git.State{Branch: "agi", Sha: "abc123"}, nil },
	})
	return NewServer(Deps{Service: svc, Repos: fakeRepos{dir: "/repo"}}), svc
}

func TestSnapshotTailSurvivesClientCancelAndProjectsV2(t *testing.T) { // [REQ:GCT-BASELINE-V2-P0]
	ctx, cancel := context.WithCancel(context.Background())
	exec := &recordingExecutor{cancel: cancel}
	runs := &recordingRuns{}
	srv, svc := newServerDeps(t, exec, runs)
	done := make(chan error, 1)
	srv.finalize = func(reqCtx context.Context, pending bl.PendingCapture) {
		_, err := svc.FinalizeCapture(context.WithoutCancel(reqCtx), pending)
		done <- err
	}

	start, err := srv.SnapshotForBaseline(ctx, connect.NewRequest(&baselinesv1.SnapshotForBaselineRequest{
		Scenario: "foo", Name: "before", Branch: "agi", CreatedBy: "agent",
	}))
	if err != nil {
		t.Fatalf("SnapshotForBaseline: %v", err)
	}
	if err := <-done; err != nil {
		t.Fatalf("finalize: %v", err)
	}
	get, err := srv.GetBaseline(context.Background(), connect.NewRequest(&baselinesv1.GetBaselineRequest{Scenario: "foo", Name: "before", Branch: "agi"}))
	if err != nil {
		t.Fatal(err)
	}
	m := get.Msg.GetBaseline()
	if start.Msg.GetRunId() != "run-1" || m.GetSchemaVersion() != 2 || m.GetRun().GetRunId() != "run-1" || m.GetRun().GetDescriptorSnapshotDigest() != "ds:catalog" {
		t.Fatalf("wire projection start=%+v manifest=%+v", start.Msg, m)
	}
	if runs.pins != 1 {
		t.Fatalf("pins=%d", runs.pins)
	}
}

func TestDiffWirePreservesTestGeniePhaseAndArtifacts(t *testing.T) { // [REQ:GCT-BASELINE-V2-P0]
	phase := &runspb.PhaseDiff{
		Phase: "unregistered-future", Verdict: "not-comparable",
		DescriptorB: &runspb.RunPhaseDescriptor{Phase: "unregistered-future", DisplayName: "Future"},
		Reasons:     []*runspb.PhaseComparisonReason{{Code: runspb.PhaseComparisonReasonCode_PHASE_COMPARISON_REASON_CODE_NEW_PHASE}},
	}
	exec := &recordingExecutor{}
	runs := &recordingRuns{compare: bl.CompareResult{Verdict: "not-comparable", Phases: []*runspb.PhaseDiff{phase}}}
	srv, svc := newServerDeps(t, exec, runs)
	if _, err := svc.Create(context.Background(), bl.CreateRequest{RepoID: 1, RepoDir: "/repo", Scenario: "foo", Name: "before", Branch: "agi"}); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	srv.finalizeDiffFn = func(ctx context.Context, pending bl.PendingDiff) {
		_, err := svc.FinalizeDiff(context.WithoutCancel(ctx), pending)
		done <- err
	}
	start, err := srv.StartDiff(context.Background(), connect.NewRequest(&baselinesv1.StartDiffRequest{Scenario: "foo", Name: "before", Branch: "agi"}))
	if err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	got, err := srv.GetDiffResult(context.Background(), connect.NewRequest(&baselinesv1.GetDiffResultRequest{Scenario: "foo", Name: "before", Branch: "agi", RunId: start.Msg.GetRunId()}))
	if err != nil {
		t.Fatal(err)
	}
	diff := got.Msg.GetDiff()
	if len(diff.GetPhases()) != 1 || diff.GetPhases()[0].GetPhase() != "unregistered-future" || len(diff.GetPhases()[0].GetReasons()) != 1 {
		t.Fatalf("phase lost on wire: %+v", diff.GetPhases())
	}
	if got := diff.GetEvidence().GetCurrentCatalog().GetArtifacts()[0].GetId(); got != "opaque-"+start.Msg.GetRunId() {
		t.Fatalf("artifact id = %q", got)
	}
}
