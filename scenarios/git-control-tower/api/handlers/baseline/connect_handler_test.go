package baseline

import (
	"bytes"
	"context"
	"log"
	"os"
	stdexec "os/exec"
	"path/filepath"
	"strings"
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

func TestCollectionCaptureProjectsMemberCoverageAndResumes(t *testing.T) {
	exec := &recordingExecutor{}
	runs := &recordingRuns{}
	srv, svc := newServerDeps(t, exec, runs)
	srv.finalizeCollection = func(ctx context.Context, repoID int64, pending bl.PendingCollectionCapture) {
		if _, err := svc.FinalizeCollectionCapture(context.WithoutCancel(ctx), repoID, pending); err != nil {
			t.Errorf("finalize collection: %v", err)
		}
	}
	request := &baselinesv1.StartCollectionCaptureRequest{
		Name: "before", Branch: "agi",
		Targets: []*baselinesv1.CollectionTarget{
			{Scenario: "plan-manager", BaselineName: "before", Required: true},
			{Scenario: "git-control-tower", BaselineName: "before", Required: true},
		},
	}
	started, err := srv.StartCollectionCapture(context.Background(), connect.NewRequest(request))
	if err != nil {
		t.Fatalf("StartCollectionCapture: %v", err)
	}
	if started.Msg.GetResumed() || started.Msg.GetCollection().GetCoverage().GetPending() != 2 {
		t.Fatalf("initial collection = %#v", started.Msg)
	}
	got, err := srv.GetCollection(context.Background(), connect.NewRequest(&baselinesv1.GetCollectionRequest{Name: "before", Branch: "agi"}))
	if err != nil {
		t.Fatalf("GetCollection: %v", err)
	}
	if !got.Msg.GetCollection().GetCoverage().GetComplete() || got.Msg.GetCollection().GetCoverage().GetReady() != 2 {
		t.Fatalf("final collection = %#v", got.Msg.GetCollection())
	}
	srv.finalizeCollectionDiffFn = func(ctx context.Context, repoID int64, pending bl.PendingCollectionDiff) {
		if _, err := svc.FinalizeCollectionDiff(context.WithoutCancel(ctx), repoID, pending); err != nil {
			t.Errorf("finalize collection diff: %v", err)
		}
	}
	diff, err := srv.StartCollectionDiff(context.Background(), connect.NewRequest(&baselinesv1.StartCollectionDiffRequest{Name: "before", Branch: "agi", OperationId: "phase-1", Scenarios: []string{"plan-manager"}}))
	if err != nil {
		t.Fatalf("StartCollectionDiff: %v", err)
	}
	if len(diff.Msg.GetMembers()) != 1 || diff.Msg.GetMembers()[0].GetScenario() != "plan-manager" || diff.Msg.GetMembers()[0].GetStatus() != "pending" {
		t.Fatalf("narrow collection diff = %#v", diff.Msg)
	}
	settled, err := srv.GetCollectionDiff(context.Background(), connect.NewRequest(&baselinesv1.GetCollectionDiffRequest{Name: "before", Branch: "agi", OperationId: "phase-1", Wait: true}))
	if err != nil || len(settled.Msg.GetMembers()) != 1 || settled.Msg.GetMembers()[0].GetStatus() != "ready" {
		t.Fatalf("settled collection diff = %#v err=%v", settled.Msg, err)
	}
	resumed, err := srv.StartCollectionCapture(context.Background(), connect.NewRequest(request))
	if err != nil || !resumed.Msg.GetResumed() || exec.calls != 3 {
		t.Fatalf("resume = %#v err=%v calls=%d", resumed.Msg, err, exec.calls)
	}
}

func TestCollectionCaptureAsyncFinalizersProjectSiblingAndLogLifecycle(t *testing.T) { // [REQ:GCT-DURABLE-OPS-P0]
	exec := &blockingCollectionExecutor{firstAwaitStarted: make(chan struct{}), releaseFirst: make(chan struct{})}
	runs := &recordingRuns{}
	srv, _ := newServerDeps(t, exec, runs)
	logs := &lifecycleLogBuffer{terminalCommit: make(chan struct{})}
	srv.logger = log.New(logs, "", 0)

	started, err := srv.StartCollectionCapture(context.Background(), connect.NewRequest(&baselinesv1.StartCollectionCaptureRequest{
		Name: "before", Branch: "agi",
		Targets: []*baselinesv1.CollectionTarget{
			{Scenario: "blocked", BaselineName: "before", Required: true},
			{Scenario: "terminal", BaselineName: "before", Required: true},
		},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if started.Msg.GetCollection().GetCoverage().GetPending() != 2 {
		t.Fatalf("initial collection = %#v", started.Msg.GetCollection())
	}
	<-exec.firstAwaitStarted

	partial, err := srv.GetCollection(context.Background(), connect.NewRequest(&baselinesv1.GetCollectionRequest{Name: "before", Branch: "agi"}))
	if err != nil {
		t.Fatal(err)
	}
	coverage := partial.Msg.GetCollection().GetCoverage()
	if coverage.GetReady() != 1 || coverage.GetPending() != 1 || coverage.GetComplete() {
		t.Fatalf("partial coverage = %#v", coverage)
	}
	if !strings.Contains(logs.String(), "finalizer started collection=before scenario=blocked run=run-1") {
		t.Fatalf("lifecycle logs = %q", logs.String())
	}

	close(exec.releaseFirst)
	complete, err := srv.GetCollection(context.Background(), connect.NewRequest(&baselinesv1.GetCollectionRequest{Name: "before", Branch: "agi", Wait: true}))
	if err != nil || !complete.Msg.GetCollection().GetCoverage().GetComplete() {
		t.Fatalf("complete collection = %#v err=%v", complete.Msg.GetCollection(), err)
	}
	select {
	case <-logs.terminalCommit:
	case <-time.After(time.Second):
		t.Fatalf("terminal commit log missing: %q", logs.String())
	}
}

// blockingCollectionExecutor holds the first collection member at the durable
// Test Genie wait boundary. It lets the handler test assert progress through
// the real asynchronous finalizer rather than substituting its test hook.
type blockingCollectionExecutor struct {
	recordingExecutor
	firstAwaitStarted chan struct{}
	releaseFirst      chan struct{}
	firstAwaitOnce    sync.Once
}

func (e *blockingCollectionExecutor) AwaitResult(ctx context.Context, scenario, runID string) (bl.ExecResult, error) {
	if runID == "run-1" {
		e.firstAwaitOnce.Do(func() { close(e.firstAwaitStarted) })
		select {
		case <-e.releaseFirst:
		case <-ctx.Done():
			return bl.ExecResult{}, ctx.Err()
		}
	}
	return e.recordingExecutor.AwaitResult(ctx, scenario, runID)
}

func (e *blockingCollectionExecutor) RunStatus(_ context.Context, _, runID string) (bl.RunStatusInfo, error) {
	if runID == "run-1" {
		select {
		case <-e.releaseFirst:
			return bl.RunStatusInfo{Status: "passed", Terminal: true, Success: true}, nil
		default:
			return bl.RunStatusInfo{Status: "in_progress", Terminal: false}, nil
		}
	}
	return bl.RunStatusInfo{Status: "passed", Terminal: true, Success: true}, nil
}

type lifecycleLogBuffer struct {
	mu             sync.Mutex
	data           bytes.Buffer
	terminalCommit chan struct{}
	once           sync.Once
}

func (b *lifecycleLogBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	n, err := b.data.Write(p)
	if strings.Contains(string(p), "terminal commit collection=") {
		b.once.Do(func() { close(b.terminalCommit) })
	}
	return n, err
}

func (b *lifecycleLogBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.data.String()
}

func TestPathSnapshotHandlersReturnMetadataWithoutContent(t *testing.T) {
	exec := &recordingExecutor{}
	runs := &recordingRuns{}
	srv, _ := newServerDeps(t, exec, runs)
	repo := t.TempDir()
	if out, err := stdexec.Command("git", "-C", repo, "init", "--quiet").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v (%s)", err, out)
	}
	if err := os.WriteFile(filepath.Join(repo, "dirty.txt"), []byte("dirty source bytes\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	srv.repos = fakeRepos{dir: repo}
	before, err := srv.CapturePathSnapshot(context.Background(), connect.NewRequest(&baselinesv1.CapturePathSnapshotRequest{Name: "before", Branch: "agi", Selections: []string{"*.txt"}, RetentionSeconds: 2 * 60 * 60}))
	if err != nil {
		t.Fatalf("CapturePathSnapshot: %v", err)
	}
	entry := before.Msg.GetSnapshot().GetEntries()[0]
	if entry.GetDigest() == "" || entry.GetDetail() == "dirty source bytes\n" {
		t.Fatalf("source bytes leaked or digest absent: %#v", entry)
	}
	if before.Msg.GetSnapshot().GetPolicyVersion() != bl.PathSnapshotPolicyVersion {
		t.Fatalf("policy version = %d", before.Msg.GetSnapshot().GetPolicyVersion())
	}
	created, err := time.Parse(time.RFC3339Nano, before.Msg.GetSnapshot().GetCreatedAt())
	if err != nil {
		t.Fatalf("parse created_at: %v", err)
	}
	expires, err := time.Parse(time.RFC3339Nano, before.Msg.GetSnapshot().GetExpiresAt())
	if err != nil || expires.Sub(created) != 2*time.Hour {
		t.Fatalf("retention was not preserved: created=%s expires=%s err=%v", created, expires, err)
	}
	if err := os.WriteFile(filepath.Join(repo, "dirty.txt"), []byte("changed source bytes\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.CapturePathSnapshot(context.Background(), connect.NewRequest(&baselinesv1.CapturePathSnapshotRequest{Name: "after", Branch: "agi", Selections: []string{"*.txt"}})); err != nil {
		t.Fatalf("CapturePathSnapshot after: %v", err)
	}
	diff, err := srv.DiffPathSnapshots(context.Background(), connect.NewRequest(&baselinesv1.DiffPathSnapshotsRequest{BeforeName: "before", AfterName: "after", Branch: "agi"}))
	if err != nil {
		t.Fatalf("DiffPathSnapshots: %v", err)
	}
	if diff.Msg.GetClassification() != "informational-source-evidence" || len(diff.Msg.GetDeltas()) != 1 || diff.Msg.GetDeltas()[0].GetStatus() != "modified" {
		t.Fatalf("source diff = %#v", diff.Msg)
	}
	filtered, err := srv.DiffPathSnapshots(context.Background(), connect.NewRequest(&baselinesv1.DiffPathSnapshotsRequest{BeforeName: "before", AfterName: "after", Branch: "agi", Selections: []string{"other/**"}}))
	if err != nil || len(filtered.Msg.GetDeltas()) != 0 {
		t.Fatalf("filtered source diff = %#v err=%v", filtered.Msg, err)
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
