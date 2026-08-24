package runs

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/freshness-go/treedigest"

	"test-genie/internal/execution"
	"test-genie/internal/orchestrator"
	"test-genie/internal/orchestrator/phases"
	"test-genie/internal/runmanager"
	sharedruns "test-genie/internal/shared/runs"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
	"google.golang.org/protobuf/proto"
)

// rpcFakeExecutor drives a controllable run for the lifecycle RPC tests.
type rpcFakeExecutor struct {
	scenarioDir string
	blockOnCtx  bool
	release     chan struct{}
	startedOnce sync.Once
	started     chan struct{}
	result      *orchestrator.SuiteExecutionResult
}

type countingRPCPlanner struct {
	mu       sync.Mutex
	calls    int
	preview  *execution.ExecutionPlanPreview
	blockCtx bool
}

func (p *countingRPCPlanner) Preview(ctx context.Context, _ orchestrator.SuiteExecutionRequest) (*execution.ExecutionPlanPreview, error) {
	p.mu.Lock()
	p.calls++
	p.mu.Unlock()
	if p.blockCtx {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return p.preview, nil
}

func (p *countingRPCPlanner) callCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.calls
}

func newRPCFake(scenarioDir string) *rpcFakeExecutor {
	_ = os.MkdirAll(scenarioDir, 0o755)
	return &rpcFakeExecutor{
		scenarioDir: scenarioDir,
		release:     make(chan struct{}),
		started:     make(chan struct{}),
		result:      &orchestrator.SuiteExecutionResult{ScenarioName: "demo", Success: true, Verdict: "PASS", CompletedAt: time.Now().UTC()},
	}
}

func TestHistoricalStandingWireDataDoesNotDecodeAsPresentation(t *testing.T) {
	legacy := &runspb.RunEvent{MaturityStanding: &runspb.PhaseMaturityStanding{
		Provider:     "structure-health",
		Phase:        "structure",
		CurrentLevel: "L1",
	}}
	wire, err := proto.Marshal(legacy)
	if err != nil {
		t.Fatalf("marshal historical event: %v", err)
	}
	decoded := &runspb.RunEvent{}
	if err := proto.Unmarshal(wire, decoded); err != nil {
		t.Fatalf("unmarshal historical event: %v", err)
	}
	if decoded.GetPhasePresentation() != nil {
		t.Fatalf("historical standing must not decode as a presentation: %+v", decoded.GetPhasePresentation())
	}
	if decoded.GetMaturityStanding().GetProvider() != "structure-health" {
		t.Fatalf("historical standing was not retained: %+v", decoded.GetMaturityStanding())
	}
}

func TestPreparationStageKeepsPreparingLifecycle(t *testing.T) {
	got := testGenieLifecycle(runmanager.LiveStatus{
		Status:      sharedruns.StatusInProgress,
		Active:      true,
		ActivePhase: "preparing:provider_readiness",
	})
	if got != "preparing" {
		t.Fatalf("lifecycle = %q, want preparing", got)
	}
}

func TestCanonicalPresentationUsesItsOwnWireField(t *testing.T) {
	presentation := &commonv1.PhasePresentation{ContractVersion: "v1", Provider: "ui-health", Phase: "ui-health", CurrentLevel: "L2"}
	current := &runspb.RunEvent{PhasePresentation: presentation}
	wire, err := proto.Marshal(current)
	if err != nil {
		t.Fatalf("marshal current event: %v", err)
	}
	decoded := &runspb.RunEvent{}
	if err := proto.Unmarshal(wire, decoded); err != nil {
		t.Fatalf("unmarshal current event: %v", err)
	}
	if !proto.Equal(decoded.GetPhasePresentation(), presentation) || decoded.GetMaturityStanding() != nil {
		t.Fatalf("canonical presentation wire contract drifted: %+v", decoded)
	}
}

func (f *rpcFakeExecutor) ExecuteWithEvents(ctx context.Context, input execution.SuiteExecutionInput, emit orchestrator.ExecutionEventCallback) (*orchestrator.SuiteExecutionResult, error) {
	descriptorSnapshot, _ := sharedruns.NewDescriptorSnapshot([]sharedruns.PhaseDescriptorSnapshot{{
		Phase: "architecture", DisplayName: "Architecture", Provider: "architecture-health",
		Applicability: sharedruns.ApplicabilityDecisionSnapshot{Status: "applies", Planned: true},
	}})
	_ = sharedruns.WriteDescriptorSnapshot(f.scenarioDir, input.Request.RunID, descriptorSnapshot)
	_ = sharedruns.NewIndex(f.scenarioDir).Append(sharedruns.RunRecord{
		RunID: input.Request.RunID, Scenario: input.Request.ScenarioName, StartedAt: time.Now().UTC(), Status: sharedruns.StatusInProgress,
		DescriptorSnapshotSchemaVersion: descriptorSnapshot.SchemaVersion, DescriptorSnapshotDigest: descriptorSnapshot.Digest,
	})
	f.startedOnce.Do(func() { close(f.started) })
	if f.blockOnCtx {
		<-ctx.Done()
	} else {
		<-f.release
	}
	_ = sharedruns.NewIndex(f.scenarioDir).Update(input.Request.RunID, func(r *sharedruns.RunRecord) error {
		r.Status = sharedruns.StatusPassed
		r.CompletedAt = time.Now().UTC()
		return nil
	})
	return f.result, nil
}

func TestLifecycleRPC_StartWaitStatus(t *testing.T) { // [REQ:TESTGENIE-RUN-SNAPSHOT-P0]
	root := newFleetRoot(t)
	fake := newRPCFake(root + "/demo")
	fake.result.Phases = []phases.ExecutionResult{{
		Name: "architecture", Status: "passed", DurationSeconds: 7,
		PhasePresentation: &commonv1.PhasePresentation{
			Provider:             "architecture-health",
			Phase:                "architecture",
			CurrentLevel:         "L2",
			NextLevel:            "L3",
			BlockingFindingCodes: []string{"arch.primitive_unverified"},
			NextAction:           "Prove each command primitive.",
		},
		FindingsSummary: &runspb.PhaseFindingsSummary{Errors: 1, Total: 1},
	}}
	svc := NewService(root, runmanager.New(fake, root), nil, nil)
	ctx := context.Background()

	start, err := svc.StartRun(ctx, connect.NewRequest(&runspb.StartRunRequest{Target: "demo"}))
	if err != nil {
		t.Fatalf("StartRun: %v", err)
	}
	runID := start.Msg.GetRunId()
	if runID == "" {
		t.Fatal("StartRun returned empty run id")
	}
	<-fake.started

	st, err := svc.GetRunStatus(ctx, connect.NewRequest(&runspb.GetRunStatusRequest{Target: "demo", RunId: runID}))
	if err != nil {
		t.Fatalf("GetRunStatus: %v", err)
	}
	if st.Msg.GetStatus() != sharedruns.StatusInProgress {
		t.Fatalf("status = %q, want in_progress", st.Msg.GetStatus())
	}
	if !st.Msg.GetActive() {
		t.Fatal("expected active=true for a live run")
	}
	if got := st.Msg.GetStanding(); got == nil || got.GetLifecycle() != "preparing" || got.GetDirective() != "wait" || got.GetActivePhase() != "" {
		t.Fatalf("preparing run standing = %#v; a missing active phase must remain waitable", got)
	}

	// Wait with a short timeout returns a non-terminal snapshot (run continues).
	wr, err := svc.WaitRun(ctx, connect.NewRequest(&runspb.WaitRunRequest{Target: "demo", RunId: runID, TimeoutSeconds: 1}))
	if err != nil {
		t.Fatalf("WaitRun(timeout): %v", err)
	}
	if !wr.Msg.GetTimedOut() {
		t.Fatal("expected timed_out=true on short wait")
	}
	if wr.Msg.GetStatus().GetStatus() != sharedruns.StatusInProgress {
		t.Fatalf("timed-out wait status = %q, want in_progress", wr.Msg.GetStatus().GetStatus())
	}

	close(fake.release)

	wr2, err := svc.WaitRun(ctx, connect.NewRequest(&runspb.WaitRunRequest{Target: "demo", RunId: runID}))
	if err != nil {
		t.Fatalf("WaitRun(terminal): %v", err)
	}
	if wr2.Msg.GetTimedOut() {
		t.Fatal("expected timed_out=false on terminal wait")
	}
	if wr2.Msg.GetStatus().GetStatus() != sharedruns.StatusPassed {
		t.Fatalf("terminal wait status = %q, want passed", wr2.Msg.GetStatus().GetStatus())
	}
	terminalRun := wr2.Msg.GetTerminalRun()
	if terminalRun == nil || wr2.Msg.GetTerminalSnapshotSchemaVersion() != sharedruns.TerminalSnapshotSchemaVersion {
		t.Fatalf("terminal snapshot metadata = run:%+v schema:%d", terminalRun, wr2.Msg.GetTerminalSnapshotSchemaVersion())
	}
	if len(wr2.Msg.GetDegradedReasons()) != 0 || len(terminalRun.GetPhases()) != 1 {
		t.Fatalf("terminal projection = %+v degraded=%v", terminalRun, wr2.Msg.GetDegradedReasons())
	}
	if snapshot := terminalRun.GetDescriptorSnapshot(); snapshot == nil || snapshot.GetDigest() == "" || len(snapshot.GetPhases()) != 1 || snapshot.GetPhases()[0].GetDisplayName() != "Architecture" {
		t.Fatalf("terminal descriptor snapshot = %+v", snapshot)
	}
	if phase := terminalRun.GetPhases()[0]; phase.GetName() != "architecture" || phase.GetStatus() != "passed" || phase.GetDurationSeconds() != 7 {
		t.Fatalf("terminal phase = %+v", phase)
	}
	show, err := svc.GetRun(ctx, connect.NewRequest(&runspb.GetRunRequest{Target: "demo", RunId: runID}))
	if err != nil {
		t.Fatalf("GetRun(terminal): %v", err)
	}
	showPhase := show.Msg.GetRun().GetPhases()[0]
	if show.Msg.GetTerminalSnapshotSchemaVersion() != wr2.Msg.GetTerminalSnapshotSchemaVersion() ||
		showPhase.GetName() != terminalRun.GetPhases()[0].GetName() ||
		showPhase.GetStatus() != terminalRun.GetPhases()[0].GetStatus() ||
		showPhase.GetDurationSeconds() != terminalRun.GetPhases()[0].GetDurationSeconds() {
		t.Fatalf("wait/show terminal mismatch: wait=%+v show=%+v", terminalRun, show.Msg.GetRun())
	}
	standings := wr2.Msg.GetStatus().GetTerminalPresentations()
	if len(standings) != 1 {
		t.Fatalf("terminal standings = %d, want 1", len(standings))
	}
	if got := standings[0].GetBlockingFindingCodes(); len(got) != 1 || got[0] != "arch.primitive_unverified" {
		t.Fatalf("terminal standing blocking codes = %v", got)
	}
	summaries := wr2.Msg.GetStatus().GetTerminalFindingsSummaries()
	if len(summaries) != 1 || summaries[0].GetErrors() != 1 {
		t.Fatalf("terminal findings summaries = %+v", summaries)
	}
}

func TestLifecycleRPC_StartRunPreviewsOnceForAdmission(t *testing.T) {
	root := newFleetRoot(t)
	fake := newRPCFake(root + "/demo")
	planner := &countingRPCPlanner{preview: &execution.ExecutionPlanPreview{
		ScenarioName: "demo", PhaseSetDigest: "phase:one", DescriptorSnapshotDigest: "descriptor:one", ConfigurationFingerprint: "config:one",
	}}
	manager := runmanager.New(fake, root)
	defer manager.Shutdown()
	svc := NewService(root, manager, planner, nil)

	if _, err := svc.StartRun(context.Background(), connect.NewRequest(&runspb.StartRunRequest{Target: "demo"})); err != nil {
		t.Fatalf("StartRun: %v", err)
	}
	if got := planner.callCount(); got != 1 {
		t.Fatalf("planner calls = %d, want 1", got)
	}
	close(fake.release)
}

func TestPrepareAdmissionProjectsPlanTimingIntoDurableRequest(t *testing.T) {
	root := newFleetRoot(t)
	if err := os.MkdirAll(root+"/demo", 0o755); err != nil {
		t.Fatal(err)
	}
	planner := &countingRPCPlanner{preview: &execution.ExecutionPlanPreview{
		ScenarioName: "demo",
		Phases: []execution.PlannedPhase{
			{Name: "structure", EstimatedDurationSeconds: 1},
			{Name: "docs", EstimatedDurationSeconds: 43},
		},
		PhaseSetDigest:           "phase:one",
		DescriptorSnapshotDigest: "descriptor:one",
		ConfigurationFingerprint: "config:one",
	}}
	svc := NewService(root, nil, planner, nil)
	req := &orchestrator.SuiteExecutionRequest{ScenarioName: "demo"}
	if _, _, err := svc.prepareAdmission(context.Background(), req); err != nil {
		t.Fatalf("prepareAdmission: %v", err)
	}
	// The selection lands on ResolvedPhases, never on Phases. Phases means the
	// operator named them, which records no preset and made every durable run
	// ineligible for the baseline reuse it had just earned.
	if len(req.Phases) != 0 {
		t.Fatalf("admission pinned Phases %v; a preset request must stay a preset request", req.Phases)
	}
	if got, want := strings.Join(req.ResolvedPhases, ","), "structure,docs"; got != want {
		t.Fatalf("resolved phases = %q, want %q", got, want)
	}
	if got, want := req.PredictedPhaseDurationsMilliseconds["docs"], int64(43_000); got != want {
		t.Fatalf("docs prediction = %d, want %d", got, want)
	}
	if got, want := req.PredictedPhaseDurationsMilliseconds["structure"], int64(1_000); got != want {
		t.Fatalf("structure prediction = %d, want %d", got, want)
	}
}

func TestPrepareAdmissionHonorsCancellation(t *testing.T) {
	root := newFleetRoot(t)
	if err := os.MkdirAll(root+"/demo", 0o755); err != nil {
		t.Fatal(err)
	}
	planner := &countingRPCPlanner{blockCtx: true}
	svc := NewService(root, nil, planner, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	started := time.Now()
	_, _, err := svc.prepareAdmission(ctx, &orchestrator.SuiteExecutionRequest{ScenarioName: "demo"})
	if err == nil {
		t.Fatal("expected cancelled admission to fail")
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("admission ignored cancellation for %s", elapsed)
	}
}

func TestPrepareAdmissionUsesCustomScenarioPathForTreeDigest(t *testing.T) {
	root := newFleetRoot(t)
	if err := os.MkdirAll(root+"/demo", 0o755); err != nil {
		t.Fatal(err)
	}
	scratch := t.TempDir()
	if err := os.WriteFile(scratch+"/marker.txt", []byte("scratch source"), 0o644); err != nil {
		t.Fatal(err)
	}
	svc := NewService(root, nil, nil, nil)
	req := &orchestrator.SuiteExecutionRequest{ScenarioName: "demo", ScenarioPath: scratch}
	if _, _, err := svc.prepareAdmission(context.Background(), req); err != nil {
		t.Fatalf("prepareAdmission: %v", err)
	}
	want, err := treedigest.Compute(scratch)
	if err != nil {
		t.Fatalf("compute scratch digest: %v", err)
	}
	if req.AdmissionTreeDigest != want {
		t.Fatalf("admission digest = %q, want custom scenario digest %q", req.AdmissionTreeDigest, want)
	}
}

func TestLifecycleRPC_PreservesArtifactCatalogFailureAsDegradedEvidence(t *testing.T) {
	root := newFleetRoot(t)
	scenarioDir := root + "/demo"
	idx := sharedruns.NewIndex(scenarioDir)
	if err := idx.Append(sharedruns.RunRecord{RunID: "catalog-failed", Scenario: "demo", StartedAt: time.Now().UTC(), Status: sharedruns.StatusInProgress}); err != nil {
		t.Fatal(err)
	}
	result := &orchestrator.SuiteExecutionResult{RunID: "catalog-failed", ScenarioName: "demo", CompletedAt: time.Now().UTC(), Success: true, Verdict: "PASS", Warnings: []string{"artifact catalog unavailable: disk full"}}
	if err := idx.Finalize("catalog-failed", result, func(r *sharedruns.RunRecord) error {
		r.Status = sharedruns.StatusPassed
		r.CompletedAt = result.CompletedAt
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	svc := NewService(root, nil, nil, nil)
	show, err := svc.GetRun(context.Background(), connect.NewRequest(&runspb.GetRunRequest{Target: "demo", RunId: "catalog-failed"}))
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if got := show.Msg.GetDegradedReasons(); !containsString(got, "artifact catalog unavailable: disk full") {
		t.Fatalf("GetRun catalog degradation = %v", got)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestLifecycleRPC_LegacyTerminalReadIsExplicitlyDegraded(t *testing.T) { // [REQ:TESTGENIE-RUN-SNAPSHOT-P0]
	root := newFleetRoot(t)
	idx := sharedruns.NewIndex(root + "/demo")
	if err := idx.Append(sharedruns.RunRecord{
		RunID: "legacy", Scenario: "demo", StartedAt: time.Now().UTC(), CompletedAt: time.Now().UTC(), Status: sharedruns.StatusPassed,
	}); err != nil {
		t.Fatalf("append legacy run: %v", err)
	}
	svc := NewService(root, runmanager.New(nil, root), nil, nil)
	wait, err := svc.WaitRun(context.Background(), connect.NewRequest(&runspb.WaitRunRequest{Target: "demo", RunId: "legacy"}))
	if err != nil {
		t.Fatalf("WaitRun legacy: %v", err)
	}
	if wait.Msg.GetTerminalRun() != nil || len(wait.Msg.GetDegradedReasons()) != 2 || len(wait.Msg.GetStatus().GetDegradedReasons()) != 2 {
		t.Fatalf("legacy wait must be degraded without canonical record: %+v", wait.Msg)
	}
	show, err := svc.GetRun(context.Background(), connect.NewRequest(&runspb.GetRunRequest{Target: "demo", RunId: "legacy"}))
	if err != nil {
		t.Fatalf("GetRun legacy: %v", err)
	}
	if show.Msg.GetTerminalSnapshotSchemaVersion() != 0 || len(show.Msg.GetDegradedReasons()) != 2 {
		t.Fatalf("legacy show must preserve degraded reason: %+v", show.Msg)
	}
}

func TestLifecycleRPC_Abort(t *testing.T) {
	root := newFleetRoot(t)
	fake := newRPCFake(root + "/demo")
	fake.blockOnCtx = true
	fake.result = &orchestrator.SuiteExecutionResult{ScenarioName: "demo", Success: false, Verdict: "FAIL", CompletedAt: time.Now().UTC()}
	svc := NewService(root, runmanager.New(fake, root), nil, nil)
	ctx := context.Background()

	start, err := svc.StartRun(ctx, connect.NewRequest(&runspb.StartRunRequest{Target: "demo"}))
	if err != nil {
		t.Fatalf("StartRun: %v", err)
	}
	runID := start.Msg.GetRunId()
	<-fake.started

	ab, err := svc.AbortRun(ctx, connect.NewRequest(&runspb.AbortRunRequest{Target: "demo", RunId: runID}))
	if err != nil {
		t.Fatalf("AbortRun: %v", err)
	}
	if ab.Msg.GetStatus().GetStatus() != sharedruns.StatusAborted {
		t.Fatalf("abort status = %q, want aborted", ab.Msg.GetStatus().GetStatus())
	}
}

// TestKeepFollowEvent proves the per-follower heartbeat filter drops ONLY
// phase_heartbeat events (and only when suppression is on), never phase
// transitions or the terminal run_completed.
func TestKeepFollowEvent(t *testing.T) {
	cases := []struct {
		kind     string
		suppress bool
		want     bool
	}{
		{runmanager.EventPhaseHeartbeat, true, false}, // dropped when suppressing
		{runmanager.EventPhaseHeartbeat, false, true}, // kept for an interactive follower
		{runmanager.EventPhaseStarted, true, true},    // phase transitions always survive
		{runmanager.EventPhaseCompleted, true, true},  // ...
		{runmanager.EventPhaseFailed, true, true},     // ...
		{runmanager.EventRunCompleted, true, true},    // the verdict always survives
		{runmanager.EventPhaseProgress, true, true},   // progress always survives
	}
	for _, tc := range cases {
		if got := keepFollowEvent(runmanager.Event{Kind: tc.kind}, tc.suppress); got != tc.want {
			t.Errorf("keepFollowEvent(%q, suppress=%v) = %v, want %v", tc.kind, tc.suppress, got, tc.want)
		}
	}
}

func TestLifecycleRPC_StartRejectsEmptyScenario(t *testing.T) {
	svc := NewService(t.TempDir(), runmanager.New(newRPCFake(t.TempDir()), t.TempDir()), nil, nil)
	if _, err := svc.StartRun(context.Background(), connect.NewRequest(&runspb.StartRunRequest{})); err == nil {
		t.Fatal("expected error for empty scenario")
	}
}
