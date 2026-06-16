package baseline

import (
	"context"
	"sync"
	"testing"

	"connectrpc.com/connect"

	bl "git-control-tower/internal/baseline"
	"git-control-tower/internal/git"

	"github.com/vrooli/api-core/storage"
	baselinesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/baselines"
)

type fakeRepos struct{ dir string }

func (f fakeRepos) Resolve(_ context.Context, _ int64) (int64, string, error) {
	return 1, f.dir, nil
}

func newTestServer(t *testing.T) *Server {
	t.Helper()
	resolver, err := storage.NewResolver(storage.ResolverConfig{AppID: "vrooli"})
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}
	svc := bl.NewService(bl.Deps{
		Storage:    bl.NewStorageAt(resolver, t.TempDir()),
		CaptureGit: func(context.Context, string) (git.State, error) { return git.State{Branch: "agi", Sha: "abc123"}, nil },
	})
	return NewServer(Deps{Service: svc, Repos: fakeRepos{dir: "/repo"}})
}

// TestBaselinesServiceRoundTrip exercises the create→get→list→diff→delete RPC
// surface end-to-end through the Connect handler (proto↔domain mapping), using
// an empty-capture baseline so no external subsystem is touched.
func TestBaselinesServiceRoundTrip(t *testing.T) {
	srv := newTestServer(t)
	ctx := context.Background()

	createResp, err := srv.CreateBaseline(ctx, connect.NewRequest(&baselinesv1.CreateBaselineRequest{
		Scenario: "foo", Name: "plan-1", Branch: "agi",
	}))
	if err != nil {
		t.Fatalf("CreateBaseline: %v", err)
	}
	if createResp.Msg.GetBaseline().GetName() != "plan-1" {
		t.Fatalf("unexpected created baseline: %+v", createResp.Msg.GetBaseline())
	}
	if createResp.Msg.GetBaseline().GetGit().GetSha() != "abc123" {
		t.Fatalf("git state not mapped: %+v", createResp.Msg.GetBaseline().GetGit())
	}

	getResp, err := srv.GetBaseline(ctx, connect.NewRequest(&baselinesv1.GetBaselineRequest{
		Scenario: "foo", Name: "plan-1", Branch: "agi",
	}))
	if err != nil {
		t.Fatalf("GetBaseline: %v", err)
	}
	if getResp.Msg.GetBaseline().GetBranch() != "agi" {
		t.Fatalf("branch not mapped: %q", getResp.Msg.GetBaseline().GetBranch())
	}

	listResp, err := srv.ListBaselines(ctx, connect.NewRequest(&baselinesv1.ListBaselinesRequest{
		Scenario: "foo", Branch: "agi",
	}))
	if err != nil {
		t.Fatalf("ListBaselines: %v", err)
	}
	if len(listResp.Msg.GetBaselines()) != 1 {
		t.Fatalf("expected 1 baseline, got %d", len(listResp.Msg.GetBaselines()))
	}

	// An empty (create-only) baseline has no captured run, so a diff fails fast
	// rather than wasting a comprehensive run that has nothing to compare.
	if _, err := srv.StartDiff(ctx, connect.NewRequest(&baselinesv1.StartDiffRequest{
		Scenario: "foo", Name: "plan-1", Branch: "agi",
	})); err == nil {
		t.Fatal("StartDiff on an empty baseline should fail fast")
	}

	if _, err := srv.DeleteBaseline(ctx, connect.NewRequest(&baselinesv1.DeleteBaselineRequest{
		Scenario: "foo", Name: "plan-1", Branch: "agi",
	})); err != nil {
		t.Fatalf("DeleteBaseline: %v", err)
	}

	_, err = srv.GetBaseline(ctx, connect.NewRequest(&baselinesv1.GetBaselineRequest{
		Scenario: "foo", Name: "plan-1", Branch: "agi",
	}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("expected NotFound after delete, got %v", err)
	}
}

// --- tail-decoupling fakes (Phase 5) -------------------------------------

// cancelOnExecExecutor cancels the supplied context as soon as the run starts,
// modeling a client that disconnects the instant the heavy run begins. It still
// reports a successful run, so the cheap pin + manifest tail is what's at risk if
// the handler tied it to the request context.
type cancelOnExecExecutor struct {
	cancel func()
	mu     sync.Mutex
	calls  int
}

func (e *cancelOnExecExecutor) StartRun(_ context.Context, _ string) (bl.RunHandle, error) {
	e.mu.Lock()
	e.calls++
	e.mu.Unlock()
	e.cancel() // client "disconnects" the instant the run starts
	return bl.RunHandle{RunID: "run-xyz", EstimatedTotalSeconds: 60, EtaKnown: true}, nil
}

func (e *cancelOnExecExecutor) AwaitResult(_ context.Context, _, runID string) (bl.ExecResult, error) {
	return bl.ExecResult{RunID: runID, Success: true, Phases: []bl.PhaseStatus{{Name: "unit", Status: "passed"}}}, nil
}

func (e *cancelOnExecExecutor) RunStatus(_ context.Context, _, _ string) (bl.RunStatusInfo, error) {
	return bl.RunStatusInfo{Status: "passed", Terminal: true, Success: true}, nil
}

func (e *cancelOnExecExecutor) FindReusableRun(_ context.Context, _, _ string) (bl.ReusableRun, bool, error) {
	return bl.ReusableRun{}, false, nil
}

type recordingRuns struct {
	mu   sync.Mutex
	pins int
}

func (r *recordingRuns) PinRun(_ context.Context, _, _, _, _ string) error {
	r.mu.Lock()
	r.pins++
	r.mu.Unlock()
	return nil
}
func (r *recordingRuns) UnpinRun(_ context.Context, _, _, _ string) error { return nil }
func (r *recordingRuns) CompareRuns(_ context.Context, _, _, _, _ string) (bl.CompareResult, error) {
	return bl.CompareResult{}, nil
}

func (r *recordingRuns) ListRunVisuals(_ context.Context, _, _ string) ([]bl.RunVisual, error) {
	return nil, nil
}

func (r *recordingRuns) CompareRunVisuals(_ context.Context, _, _, _ string) ([]bl.VisualDelta, error) {
	return nil, nil
}

// SnapshotForBaseline returns the run handle immediately and finalizes (pin +
// manifest) on a server-owned context. The pin + manifest must NOT be abandoned
// when the client context is canceled the moment the durable run starts — the
// finalize tail detaches via context.WithoutCancel. This is the reported
// silent-hang/abandonment fix, now on the return-fast flow.
func TestSnapshotTailSurvivesClientCancel(t *testing.T) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{AppID: "vrooli"})
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}
	store := bl.NewStorageAt(resolver, t.TempDir())

	ctx, cancel := context.WithCancel(context.Background())
	exec := &cancelOnExecExecutor{cancel: cancel}
	runs := &recordingRuns{}
	svc := bl.NewService(bl.Deps{
		Storage:    store,
		Exec:       exec,
		Runs:       runs,
		CaptureGit: func(context.Context, string) (git.State, error) { return git.State{Branch: "agi", Sha: "abc123"}, nil },
	})
	srv := NewServer(Deps{Service: svc, Repos: fakeRepos{dir: "/repo"}})
	// Run the finalize tail synchronously (still detached from the request ctx
	// via WithoutCancel) so the durability assertion is deterministic.
	done := make(chan error, 1)
	srv.finalize = func(reqCtx context.Context, pending bl.PendingCapture) {
		_, ferr := svc.FinalizeCapture(context.WithoutCancel(reqCtx), pending)
		done <- ferr
	}

	resp, err := srv.SnapshotForBaseline(ctx, connect.NewRequest(&baselinesv1.SnapshotForBaselineRequest{
		Scenario: "foo", Name: "snap-1", Branch: "agi",
	}))
	if err != nil {
		t.Fatalf("SnapshotForBaseline (cancel mid-run): %v", err)
	}
	if ferr := <-done; ferr != nil {
		t.Fatalf("finalize failed: %v", ferr)
	}
	if exec.calls != 1 {
		t.Fatalf("expected exactly one run start, got %d", exec.calls)
	}
	if runs.pins != 1 {
		t.Fatalf("pin must survive client cancel, got %d pins", runs.pins)
	}
	if got := resp.Msg.GetRunId(); got != "run-xyz" {
		t.Fatalf("snapshot must return the run id, got %q", got)
	}
	if got := resp.Msg.GetName(); got != "snap-1" {
		t.Fatalf("snapshot must echo the baseline name, got %q", got)
	}
	// The manifest is durable on disk despite the canceled client context.
	if _, err := svc.Get(context.Background(), 1, "foo", "agi", "snap-1"); err != nil {
		t.Fatalf("manifest must be durably persisted: %v", err)
	}
}

// StartDiff returns the current run handle immediately and finalizes the verdict
// on a server-owned tail; GetDiffResult then returns the cached verdict. This is
// the durable, return-fast diff contract (mirror of the snapshot tail test).
func TestStartDiffThenGetResult(t *testing.T) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{AppID: "vrooli"})
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}
	store := bl.NewStorageAt(resolver, t.TempDir())
	exec := &cancelOnExecExecutor{cancel: func() {}}
	runs := &recordingRuns{}
	svc := bl.NewService(bl.Deps{
		Storage:    store,
		Exec:       exec,
		Runs:       runs,
		CaptureGit: func(context.Context, string) (git.State, error) { return git.State{Branch: "agi", Sha: "abc123"}, nil },
	})
	srv := NewServer(Deps{Service: svc, Repos: fakeRepos{dir: "/repo"}})
	// Run the diff finalize tail synchronously (still detached) for determinism.
	done := make(chan error, 1)
	srv.finalizeDiffFn = func(reqCtx context.Context, pending bl.PendingDiff) {
		_, ferr := svc.FinalizeDiff(context.WithoutCancel(reqCtx), pending)
		done <- ferr
	}
	ctx := context.Background()

	// Seed a captured baseline (one shared run) synchronously so the diff has a
	// base run to compare against.
	if _, err := svc.Create(ctx, bl.CreateRequest{
		RepoID: 1, RepoDir: "/repo", Scenario: "foo", Name: "plan-1", Branch: "agi", Capture: true,
	}); err != nil {
		t.Fatalf("seed baseline: %v", err)
	}

	startResp, err := srv.StartDiff(ctx, connect.NewRequest(&baselinesv1.StartDiffRequest{
		Scenario: "foo", Name: "plan-1", Branch: "agi",
	}))
	if err != nil {
		t.Fatalf("StartDiff: %v", err)
	}
	if startResp.Msg.GetRunId() == "" {
		t.Fatal("StartDiff must return a run id to follow")
	}
	if ferr := <-done; ferr != nil {
		t.Fatalf("diff finalize failed: %v", ferr)
	}

	getResp, err := srv.GetDiffResult(ctx, connect.NewRequest(&baselinesv1.GetDiffResultRequest{
		Scenario: "foo", Name: "plan-1", Branch: "agi", RunId: startResp.Msg.GetRunId(),
	}))
	if err != nil {
		t.Fatalf("GetDiffResult: %v", err)
	}
	if getResp.Msg.GetStatus() != "ready" {
		t.Fatalf("expected ready status, got %q", getResp.Msg.GetStatus())
	}
	if getResp.Msg.GetDiff() == nil {
		t.Fatal("ready result must carry the diff payload")
	}
}

func TestCreateDuplicateReturnsAlreadyExists(t *testing.T) {
	srv := newTestServer(t)
	ctx := context.Background()
	req := func() *connect.Request[baselinesv1.CreateBaselineRequest] {
		return connect.NewRequest(&baselinesv1.CreateBaselineRequest{Scenario: "foo", Name: "dup", Branch: "agi"})
	}
	if _, err := srv.CreateBaseline(ctx, req()); err != nil {
		t.Fatalf("first create: %v", err)
	}
	_, err := srv.CreateBaseline(ctx, req())
	if connect.CodeOf(err) != connect.CodeAlreadyExists {
		t.Fatalf("expected AlreadyExists, got %v", err)
	}
}
