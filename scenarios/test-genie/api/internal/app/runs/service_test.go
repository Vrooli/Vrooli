package runs

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/vrooli/freshness-go/treedigest"
	"test-genie/internal/execution"
	"test-genie/internal/executionevidence"
	"test-genie/internal/orchestrator"
	"test-genie/internal/orchestrator/phases"
	sharedartifacts "test-genie/internal/shared/artifacts"
	sharedruns "test-genie/internal/shared/runs"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

type fixedExecutionPlanner struct {
	preview *execution.ExecutionPlanPreview
}

func (p fixedExecutionPlanner) Preview(context.Context, orchestrator.SuiteExecutionRequest) (*execution.ExecutionPlanPreview, error) {
	return p.preview, nil
}

func newTestService(t *testing.T) (*Service, string) {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "demo"), 0o755); err != nil {
		t.Fatal(err)
	}
	// The read-only index RPCs under test need neither the run manager nor the
	// planner.
	return NewService(root, nil, nil, nil), root
}

func seedRecord(t *testing.T, root string, rec sharedruns.RunRecord) {
	seedRunRecord(t, root, rec, true)
}

func seedLegacyRecord(t *testing.T, root string, rec sharedruns.RunRecord) {
	seedRunRecord(t, root, rec, false)
}

// seedRunRecord makes ordinary fixtures canonical terminal runs. Tests that
// exercise historical degradation opt out explicitly instead of accidentally
// relying on an incomplete snapshot.
func seedRunRecord(t *testing.T, root string, rec sharedruns.RunRecord, terminalSnapshot bool) {
	t.Helper()
	if rec.TreeDigest == "" {
		rec.TreeDigest = "td:test"
	}
	index := sharedruns.NewIndex(filepath.Join(root, "demo"))
	if err := index.Append(rec); err != nil {
		t.Fatalf("seed %s: %v", rec.RunID, err)
	}
	if terminalSnapshot && (rec.Status == sharedruns.StatusPassed || rec.Status == sharedruns.StatusFailed || rec.Status == sharedruns.StatusAborted) {
		if err := index.Finalize(rec.RunID, &orchestrator.SuiteExecutionResult{GateQuality: true}, func(*sharedruns.RunRecord) error { return nil }); err != nil {
			t.Fatalf("finalize fixture %s: %v", rec.RunID, err)
		}
	}
}

func seedDescriptorSnapshot(t *testing.T, root, runID string, phases ...sharedruns.PhaseDescriptorSnapshot) {
	t.Helper()
	snapshot, err := sharedruns.NewDescriptorSnapshot(phases)
	if err != nil {
		t.Fatalf("build descriptor snapshot for %s: %v", runID, err)
	}
	scenarioDir := filepath.Join(root, "demo")
	if err := sharedruns.WriteDescriptorSnapshot(scenarioDir, runID, snapshot); err != nil {
		t.Fatalf("write descriptor snapshot for %s: %v", runID, err)
	}
	if err := sharedruns.NewIndex(scenarioDir).Update(runID, func(record *sharedruns.RunRecord) error {
		record.DescriptorSnapshotSchemaVersion = snapshot.SchemaVersion
		record.DescriptorSnapshotDigest = snapshot.Digest
		return nil
	}); err != nil {
		t.Fatalf("stamp descriptor snapshot for %s: %v", runID, err)
	}
}

func capturedPhase(phase, displayName string) sharedruns.PhaseDescriptorSnapshot {
	return sharedruns.PhaseDescriptorSnapshot{
		Phase: phase, DisplayName: displayName,
		Applicability: sharedruns.ApplicabilityDecisionSnapshot{Status: "applies", Planned: true},
	}
}

func TestListAndGetRun(t *testing.T) {
	svc, root := newTestService(t)
	seedRecord(t, root, sharedruns.RunRecord{RunID: "r1", Scenario: "demo", StartedAt: time.Now().UTC(), Status: sharedruns.StatusPassed})
	seedRecord(t, root, sharedruns.RunRecord{RunID: "r2", Scenario: "demo", StartedAt: time.Now().UTC().Add(time.Minute), Status: sharedruns.StatusFailed})

	list, err := svc.ListRuns(context.Background(), connect.NewRequest(&runspb.ListRunsRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	if len(list.Msg.GetRuns()) != 2 {
		t.Fatalf("expected 2 runs, got %d", len(list.Msg.GetRuns()))
	}
	if list.Msg.GetRuns()[0].GetRunId() != "r2" {
		t.Fatalf("expected newest-first (r2), got %s", list.Msg.GetRuns()[0].GetRunId())
	}

	// Status filter.
	failed, err := svc.ListRuns(context.Background(), connect.NewRequest(&runspb.ListRunsRequest{Scenario: "demo", Status: "failed"}))
	if err != nil {
		t.Fatalf("ListRuns(failed): %v", err)
	}
	if len(failed.Msg.GetRuns()) != 1 || failed.Msg.GetRuns()[0].GetRunId() != "r2" {
		t.Fatalf("status filter failed: %v", failed.Msg.GetRuns())
	}

	got, err := svc.GetRun(context.Background(), connect.NewRequest(&runspb.GetRunRequest{Scenario: "demo", RunId: "r1"}))
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if got.Msg.GetRun().GetStatus() != sharedruns.StatusPassed {
		t.Fatalf("unexpected status: %s", got.Msg.GetRun().GetStatus())
	}

	// Missing run → NotFound.
	if _, err := svc.GetRun(context.Background(), connect.NewRequest(&runspb.GetRunRequest{Scenario: "demo", RunId: "nope"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("expected NotFound, got %v", err)
	}
}

func TestPinUnpinAndForceDelete(t *testing.T) {
	svc, root := newTestService(t)
	seedRecord(t, root, sharedruns.RunRecord{RunID: "r1", Scenario: "demo", StartedAt: time.Now().UTC(), Status: sharedruns.StatusPassed})

	pinned, err := svc.PinRun(context.Background(), connect.NewRequest(&runspb.PinRunRequest{Scenario: "demo", RunId: "r1", PinnedBy: "gct:baseline:x", Reason: "baseline"}))
	if err != nil {
		t.Fatalf("PinRun: %v", err)
	}
	if len(pinned.Msg.GetRun().GetPins()) != 1 {
		t.Fatalf("expected 1 pin, got %v", pinned.Msg.GetRun().GetPins())
	}

	// Deleting a pinned run without force → FailedPrecondition.
	if _, err := svc.DeleteRun(context.Background(), connect.NewRequest(&runspb.DeleteRunRequest{Scenario: "demo", RunId: "r1"})); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("expected FailedPrecondition for pinned delete, got %v", err)
	}

	if _, err := svc.UnpinRun(context.Background(), connect.NewRequest(&runspb.UnpinRunRequest{Scenario: "demo", RunId: "r1", PinnedBy: "gct:baseline:x"})); err != nil {
		t.Fatalf("UnpinRun: %v", err)
	}

	del, err := svc.DeleteRun(context.Background(), connect.NewRequest(&runspb.DeleteRunRequest{Scenario: "demo", RunId: "r1"}))
	if err != nil {
		t.Fatalf("DeleteRun: %v", err)
	}
	if !del.Msg.GetDeleted() {
		t.Fatal("expected deleted=true")
	}
}

func TestCompareRunsClassification(t *testing.T) {
	svc, root := newTestService(t)
	// Baseline: workflows + tests + structure all pass.
	seedRecord(t, root, sharedruns.RunRecord{
		RunID: "base", Scenario: "demo", StartedAt: time.Now().UTC(), Status: sharedruns.StatusPassed,
		Phases: []sharedruns.PhaseRecord{
			{Name: "playbooks", Status: "passed"},
			{Name: "unit", Status: "failed"}, // preexisting failure
			{Name: "structure", Status: "passed"},
		},
	})
	seedDescriptorSnapshot(t, root, "base",
		capturedPhase("playbooks", "Workflows"), capturedPhase("unit", "Unit"), capturedPhase("structure", "Structure"),
	)
	// Current: workflows regressed, a new phase fails, unit still fails, structure clean.
	seedRecord(t, root, sharedruns.RunRecord{
		RunID: "cur", Scenario: "demo", StartedAt: time.Now().UTC().Add(time.Minute), Status: sharedruns.StatusFailed,
		Phases: []sharedruns.PhaseRecord{
			{Name: "playbooks", Status: "failed"},   // regression (passed → failed)
			{Name: "unit", Status: "failed"},        // preexisting
			{Name: "structure", Status: "passed"},   // clean
			{Name: "integration", Status: "failed"}, // new failure (absent in base)
		},
	})
	seedDescriptorSnapshot(t, root, "cur",
		capturedPhase("playbooks", "Workflow Evidence"), capturedPhase("unit", "Unit"),
		capturedPhase("structure", "Structure"), capturedPhase("integration", "Integration"),
	)

	resp, err := svc.CompareRuns(context.Background(), connect.NewRequest(&runspb.CompareRunsRequest{Scenario: "demo", RunIdA: "base", RunIdB: "cur"}))
	if err != nil {
		t.Fatalf("CompareRuns: %v", err)
	}
	got := map[string]string{}
	for _, p := range resp.Msg.GetPhases() {
		got[p.GetPhase()] = p.GetVerdict()
	}
	if got["playbooks"] != verdictRegression {
		t.Errorf("playbooks: want regression, got %s", got["playbooks"])
	}
	if got["unit"] != verdictPreexisting {
		t.Errorf("unit: want preexisting, got %s", got["unit"])
	}
	if got["structure"] != verdictClean {
		t.Errorf("structure: want clean, got %s", got["structure"])
	}
	if got["integration"] != verdictNewFailure {
		t.Errorf("integration: want new-failure, got %s", got["integration"])
	}
	var integration *runspb.PhaseDiff
	for _, phase := range resp.Msg.GetPhases() {
		if phase.GetPhase() == "integration" {
			integration = phase
		}
	}
	if integration == nil || len(integration.GetReasons()) != 1 || integration.GetReasons()[0].GetCode() != runspb.PhaseComparisonReasonCode_PHASE_COMPARISON_REASON_CODE_NEW_PHASE {
		t.Fatalf("integration comparison reason = %+v", integration)
	}
	if gotDisplay := resp.Msg.GetPhases()[0].GetDescriptorB().GetDisplayName(); gotDisplay != "Workflow Evidence" {
		t.Fatalf("captured current display name = %q", gotDisplay)
	}
	// Overall verdict is the worst (regression).
	if resp.Msg.GetVerdict() != verdictRegression {
		t.Errorf("overall verdict: want regression, got %s", resp.Msg.GetVerdict())
	}

	// Phase filter restricts output.
	filtered, err := svc.CompareRuns(context.Background(), connect.NewRequest(&runspb.CompareRunsRequest{Scenario: "demo", RunIdA: "base", RunIdB: "cur", Phase: "structure"}))
	if err != nil {
		t.Fatalf("CompareRuns(filter): %v", err)
	}
	if len(filtered.Msg.GetPhases()) != 1 || filtered.Msg.GetPhases()[0].GetPhase() != "structure" {
		t.Fatalf("phase filter failed: %v", filtered.Msg.GetPhases())
	}
}

func TestCompareRunsDifferentGitSHAsRemainBehaviorallyComparable(t *testing.T) { // [REQ:TESTGENIE-DESCRIPTOR-SNAPSHOT-P0]
	svc, root := newTestService(t)
	seedRecord(t, root, sharedruns.RunRecord{
		RunID: "base-sha", Scenario: "demo", GitSha: "1111111", StartedAt: time.Now().UTC(), Status: sharedruns.StatusPassed,
		Phases: []sharedruns.PhaseRecord{{Name: "unit", Status: "passed"}},
	})
	seedRecord(t, root, sharedruns.RunRecord{
		RunID: "current-sha", Scenario: "demo", GitSha: "2222222", StartedAt: time.Now().UTC().Add(time.Minute), Status: sharedruns.StatusPassed,
		Phases: []sharedruns.PhaseRecord{{Name: "unit", Status: "passed"}},
	})
	seedDescriptorSnapshot(t, root, "base-sha", capturedPhase("unit", "Unit"))
	seedDescriptorSnapshot(t, root, "current-sha", capturedPhase("unit", "Unit"))

	resp, err := svc.CompareRuns(context.Background(), connect.NewRequest(&runspb.CompareRunsRequest{Scenario: "demo", RunIdA: "base-sha", RunIdB: "current-sha"}))
	if err != nil {
		t.Fatalf("CompareRuns: %v", err)
	}
	if resp.Msg.GetVerdict() != verdictClean || len(resp.Msg.GetPhases()) != 1 || resp.Msg.GetPhases()[0].GetVerdict() != verdictClean {
		t.Fatalf("different SHA comparison = %+v", resp.Msg)
	}
}

func TestCompareRunsUsesCapturedCatalogEvolutionAndTypedReasons(t *testing.T) { // [REQ:TESTGENIE-DESCRIPTOR-SNAPSHOT-P0]
	svc, root := newTestService(t)
	seedRecord(t, root, sharedruns.RunRecord{
		RunID: "old", Scenario: "demo", StartedAt: time.Now().UTC(), Status: sharedruns.StatusPassed,
		Phases: []sharedruns.PhaseRecord{
			{Name: "stable", Status: "passed"},
			{Name: "retired", Status: "passed"},
			{Name: "conditional", Status: "passed"},
			{Name: "runtime", Status: "passed"},
		},
	})
	oldStable := capturedPhase("stable", "Original Label")
	oldStable.Provider = "old-provider"
	seedDescriptorSnapshot(t, root, "old",
		oldStable, capturedPhase("retired", "Retired Phase"), capturedPhase("conditional", "Conditional"), capturedPhase("runtime", "Runtime"),
	)

	seedRecord(t, root, sharedruns.RunRecord{
		RunID: "new", Scenario: "demo", StartedAt: time.Now().UTC().Add(time.Minute), Status: sharedruns.StatusPassed,
		Phases: []sharedruns.PhaseRecord{
			{Name: "new-phase", Status: "passed"},
			{Name: "stable", Status: "passed"},
			{Name: "runtime", Status: "provider_unavailable"},
		},
	})
	newStable := capturedPhase("stable", "Renamed Label")
	newStable.Provider = "new-provider"
	inapplicable := capturedPhase("conditional", "Conditional")
	inapplicable.Applicability = sharedruns.ApplicabilityDecisionSnapshot{Status: "not_applicable", Planned: false}
	seedDescriptorSnapshot(t, root, "new",
		capturedPhase("new-phase", "New Phase"), newStable, inapplicable, capturedPhase("runtime", "Runtime"),
	)

	resp, err := svc.CompareRuns(context.Background(), connect.NewRequest(&runspb.CompareRunsRequest{Scenario: "demo", RunIdA: "old", RunIdB: "new"}))
	if err != nil {
		t.Fatalf("CompareRuns: %v", err)
	}
	if got := resp.Msg.GetPhases()[0].GetPhase(); got != "new-phase" {
		t.Fatalf("comparison order starts with %q, want current captured order", got)
	}
	byPhase := map[string]*runspb.PhaseDiff{}
	for _, diff := range resp.Msg.GetPhases() {
		byPhase[diff.GetPhase()] = diff
	}
	if got := byPhase["stable"].GetDescriptorA().GetDisplayName(); got != "Original Label" {
		t.Fatalf("baseline historical label = %q", got)
	}
	if diff := byPhase["stable"]; diff.GetDescriptorB().GetDisplayName() != "Renamed Label" || diff.GetDescriptorB().GetProvider() != "new-provider" {
		t.Fatalf("current historical descriptor = %+v", diff.GetDescriptorB())
	}
	assertComparisonReason(t, byPhase["new-phase"], runspb.PhaseComparisonReasonCode_PHASE_COMPARISON_REASON_CODE_NEW_PHASE)
	assertComparisonReason(t, byPhase["retired"], runspb.PhaseComparisonReasonCode_PHASE_COMPARISON_REASON_CODE_RETIRED_PHASE)
	assertComparisonReason(t, byPhase["conditional"], runspb.PhaseComparisonReasonCode_PHASE_COMPARISON_REASON_CODE_INAPPLICABLE)
	assertComparisonReason(t, byPhase["runtime"], runspb.PhaseComparisonReasonCode_PHASE_COMPARISON_REASON_CODE_PROVIDER_UNAVAILABLE)
}

func inapplicablePhase(phase, displayName string) sharedruns.PhaseDescriptorSnapshot {
	descriptor := capturedPhase(phase, displayName)
	descriptor.Applicability = sharedruns.ApplicabilityDecisionSnapshot{Status: "not_applicable", Planned: false}
	return descriptor
}

// A conditional phase that is not applicable in BOTH runs is a matched state
// with no regression signal: it stays visible in the phase table (with
// Symmetric inapplicability is an honest coverage gap, not a hidden clean
// result. Comparable phases still retain their independent behavior outcome.
func TestCompareRunsSymmetricInapplicableIsExplicitlyUnmeasured(t *testing.T) {
	svc, root := newTestService(t)
	seedRecord(t, root, sharedruns.RunRecord{
		RunID: "base", Scenario: "demo", StartedAt: time.Now().UTC(), Status: sharedruns.StatusFailed,
		Phases: []sharedruns.PhaseRecord{
			{Name: "unit", Status: "failed"}, // preexisting failure
			{Name: "structure", Status: "passed"},
		},
	})
	seedDescriptorSnapshot(t, root, "base",
		capturedPhase("unit", "Unit"), capturedPhase("structure", "Structure"), inapplicablePhase("ai-conformance", "AI Conformance"),
	)
	seedRecord(t, root, sharedruns.RunRecord{
		RunID: "cur", Scenario: "demo", StartedAt: time.Now().UTC().Add(time.Minute), Status: sharedruns.StatusFailed,
		Phases: []sharedruns.PhaseRecord{
			{Name: "unit", Status: "failed"},
			{Name: "structure", Status: "passed"},
		},
	})
	seedDescriptorSnapshot(t, root, "cur",
		capturedPhase("unit", "Unit"), capturedPhase("structure", "Structure"), inapplicablePhase("ai-conformance", "AI Conformance"),
	)

	resp, err := svc.CompareRuns(context.Background(), connect.NewRequest(&runspb.CompareRunsRequest{Scenario: "demo", RunIdA: "base", RunIdB: "cur"}))
	if err != nil {
		t.Fatalf("CompareRuns: %v", err)
	}
	byPhase := map[string]*runspb.PhaseDiff{}
	for _, diff := range resp.Msg.GetPhases() {
		byPhase[diff.GetPhase()] = diff
	}
	// The inapplicable phase remains visible with its reason and explicit
	// unmeasured coverage; it is never silently promoted to clean.
	assertComparisonReason(t, byPhase["ai-conformance"], runspb.PhaseComparisonReasonCode_PHASE_COMPARISON_REASON_CODE_INAPPLICABLE)
	if got := byPhase["ai-conformance"].GetCoverage(); got != "unmeasured" {
		t.Errorf("ai-conformance coverage: want unmeasured, got %s", got)
	}
	if got := resp.Msg.GetBehavior(); got != "preexisting" {
		t.Errorf("overall behavior: want preexisting, got %s", got)
	}
	if got := resp.Msg.GetVerdict(); got != verdictPreexisting {
		t.Errorf("legacy verdict: want %s, got %s", verdictPreexisting, got)
	}
}

func TestCompareRunsSymmetricBestEffortProviderUnavailableIsUnmeasured(t *testing.T) {
	svc, root := newTestService(t)
	for _, runID := range []string{"base", "cur"} {
		seedRecord(t, root, sharedruns.RunRecord{RunID: runID, Scenario: "demo", StartedAt: time.Now().UTC(), Status: sharedruns.StatusPassed, Phases: []sharedruns.PhaseRecord{{Name: "architecture", Status: "provider_unavailable"}}})
		descriptor := capturedPhase("architecture", "Architecture")
		descriptor.Policy.Unavailable = "skip_without_failing"
		seedDescriptorSnapshot(t, root, runID, descriptor)
	}
	resp, err := svc.CompareRuns(context.Background(), connect.NewRequest(&runspb.CompareRunsRequest{Scenario: "demo", RunIdA: "base", RunIdB: "cur"}))
	if err != nil {
		t.Fatalf("CompareRuns: %v", err)
	}
	if got := resp.Msg.GetVerdict(); got != verdictNotComparable {
		t.Fatalf("legacy verdict: want %s, got %s", verdictNotComparable, got)
	}
	if got := resp.Msg.GetPhases()[0].GetCoverage(); got != "unmeasured" {
		t.Fatalf("phase coverage: want unmeasured, got %s", got)
	}
	assertComparisonReason(t, resp.Msg.GetPhases()[0], runspb.PhaseComparisonReasonCode_PHASE_COMPARISON_REASON_CODE_PROVIDER_UNAVAILABLE)
}

func TestCompareRunsSameKeyContractChangeRequiresExplicitCompatibility(t *testing.T) {
	svc, root := newTestService(t)
	seedRecord(t, root, sharedruns.RunRecord{RunID: "base", Scenario: "demo", StartedAt: time.Now().UTC(), Status: sharedruns.StatusPassed, Phases: []sharedruns.PhaseRecord{{Name: "unit", Status: "passed"}}})
	base := capturedPhase("unit", "Unit")
	base.Provider = "unit-health-v1"
	seedDescriptorSnapshot(t, root, "base", base)
	seedRecord(t, root, sharedruns.RunRecord{RunID: "cur", Scenario: "demo", StartedAt: time.Now().UTC().Add(time.Minute), Status: sharedruns.StatusFailed, Phases: []sharedruns.PhaseRecord{{Name: "unit", Status: "failed"}}})
	changed := capturedPhase("unit", "Unit")
	changed.Provider = "unit-health-v2"
	seedDescriptorSnapshot(t, root, "cur", changed)

	resp, err := svc.CompareRuns(context.Background(), connect.NewRequest(&runspb.CompareRunsRequest{Scenario: "demo", RunIdA: "base", RunIdB: "cur"}))
	if err != nil {
		t.Fatal(err)
	}
	phase := resp.Msg.GetPhases()[0]
	if phase.GetCompatibility() != "changed-unreviewed" || phase.GetBehavior() != "unknown" || phase.GetVerdict() != verdictNotComparable {
		t.Fatalf("unreviewed contract change = %+v", phase)
	}

	changed.ComparisonMode = "compatible"
	seedDescriptorSnapshot(t, root, "cur", changed)
	resp, err = svc.CompareRuns(context.Background(), connect.NewRequest(&runspb.CompareRunsRequest{Scenario: "demo", RunIdA: "base", RunIdB: "cur"}))
	if err != nil {
		t.Fatal(err)
	}
	phase = resp.Msg.GetPhases()[0]
	if phase.GetCompatibility() != "compatible" || phase.GetBehavior() != "regression" || phase.GetVerdict() != verdictRegression {
		t.Fatalf("compatible contract change = %+v", phase)
	}
}

func TestCompareRunsUnknownPhaseStatusCannotLookClean(t *testing.T) {
	svc, root := newTestService(t)
	for _, runID := range []string{"base", "cur"} {
		seedRecord(t, root, sharedruns.RunRecord{RunID: runID, Scenario: "demo", StartedAt: time.Now().UTC(), Status: sharedruns.StatusPassed, Phases: []sharedruns.PhaseRecord{{Name: "unit", Status: "future_terminal_state"}}})
		seedDescriptorSnapshot(t, root, runID, capturedPhase("unit", "Unit"))
	}
	resp, err := svc.CompareRuns(context.Background(), connect.NewRequest(&runspb.CompareRunsRequest{Scenario: "demo", RunIdA: "base", RunIdB: "cur"}))
	if err != nil {
		t.Fatal(err)
	}
	phase := resp.Msg.GetPhases()[0]
	if phase.GetBehavior() != "unknown" || phase.GetVerdict() != verdictNotComparable {
		t.Fatalf("unknown terminal status = %+v", phase)
	}
}

func TestComparisonProvenanceRequiresGateQualityTerminalEvidence(t *testing.T) {
	verified := runProjection{record: sharedruns.RunRecord{TreeDigest: "td:verified"}, result: &orchestrator.SuiteExecutionResult{GateQuality: true}}
	if got := comparisonProvenance(verified, verified); got != "strict" {
		t.Fatalf("gate-quality provenance = %q", got)
	}
	volatile := verified
	volatile.result = &orchestrator.SuiteExecutionResult{}
	if got := comparisonProvenance(verified, volatile); got != "volatile" {
		t.Fatalf("exploratory provenance = %q", got)
	}
	if got := comparisonProvenance(verified, runProjection{record: sharedruns.RunRecord{TreeDigest: "td:legacy"}}); got != "legacy" {
		t.Fatalf("snapshotless provenance = %q", got)
	}
}

// Asymmetric applicability (applicable in one run, not applicable in the
// other) means the tested surface changed between the runs, so it still
// forces the run-level verdict to not-comparable.
func TestCompareRunsAsymmetricInapplicableStillNotComparable(t *testing.T) {
	svc, root := newTestService(t)
	seedRecord(t, root, sharedruns.RunRecord{
		RunID: "base", Scenario: "demo", StartedAt: time.Now().UTC(), Status: sharedruns.StatusPassed,
		Phases: []sharedruns.PhaseRecord{
			{Name: "structure", Status: "passed"},
			{Name: "conditional", Status: "passed"},
		},
	})
	seedDescriptorSnapshot(t, root, "base",
		capturedPhase("structure", "Structure"), capturedPhase("conditional", "Conditional"),
	)
	seedRecord(t, root, sharedruns.RunRecord{
		RunID: "cur", Scenario: "demo", StartedAt: time.Now().UTC().Add(time.Minute), Status: sharedruns.StatusPassed,
		Phases: []sharedruns.PhaseRecord{
			{Name: "structure", Status: "passed"},
		},
	})
	seedDescriptorSnapshot(t, root, "cur",
		capturedPhase("structure", "Structure"), inapplicablePhase("conditional", "Conditional"),
	)

	resp, err := svc.CompareRuns(context.Background(), connect.NewRequest(&runspb.CompareRunsRequest{Scenario: "demo", RunIdA: "base", RunIdB: "cur"}))
	if err != nil {
		t.Fatalf("CompareRuns: %v", err)
	}
	if got := resp.Msg.GetVerdict(); got != verdictNotComparable {
		t.Errorf("overall verdict: want %s, got %s", verdictNotComparable, got)
	}
}

func assertComparisonReason(t *testing.T, diff *runspb.PhaseDiff, code runspb.PhaseComparisonReasonCode) {
	t.Helper()
	if diff == nil {
		t.Fatalf("missing phase diff for reason %s", code)
	}
	for _, reason := range diff.GetReasons() {
		if reason.GetCode() == code {
			return
		}
	}
	t.Fatalf("phase %s reasons = %+v, want %s", diff.GetPhase(), diff.GetReasons(), code)
}

func TestGetPhaseArtifact(t *testing.T) {
	svc, root := newTestService(t)
	scenarioDir := filepath.Join(root, "demo")
	seedRecord(t, root, sharedruns.RunRecord{RunID: "r1", Scenario: "demo", StartedAt: time.Now().UTC(), Status: sharedruns.StatusPassed})

	phaseDir := sharedartifacts.RunPhaseResultsDir(scenarioDir, "r1")
	if err := os.MkdirAll(phaseDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(phaseDir, "unit.json"), []byte(`{"phase":"unit","status":"passed"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	resp, err := svc.GetPhaseArtifact(context.Background(), connect.NewRequest(&runspb.GetPhaseArtifactRequest{Scenario: "demo", RunId: "r1", Phase: "unit"}))
	if err != nil {
		t.Fatalf("GetPhaseArtifact: %v", err)
	}
	if resp.Msg.GetContentType() != "application/json" {
		t.Errorf("unexpected content type: %s", resp.Msg.GetContentType())
	}
	if resp.Msg.GetContent() == "" {
		t.Error("expected non-empty content")
	}

	// Missing artifact → NotFound.
	if _, err := svc.GetPhaseArtifact(context.Background(), connect.NewRequest(&runspb.GetPhaseArtifactRequest{Scenario: "demo", RunId: "r1", Phase: "missing"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("expected NotFound, got %v", err)
	}
	if err := os.WriteFile(filepath.Join(phaseDir, "oversized.json"), make([]byte, maxPhaseArtifactResponseBytes+1), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GetPhaseArtifact(context.Background(), connect.NewRequest(&runspb.GetPhaseArtifactRequest{Scenario: "demo", RunId: "r1", Phase: "oversized"})); connect.CodeOf(err) != connect.CodeResourceExhausted {
		t.Fatalf("oversized artifact code = %s err=%v", connect.CodeOf(err), err)
	}
}

func TestListAndGetRunArtifactsUseOpaqueTypedReferences(t *testing.T) { // [REQ:TESTGENIE-TYPED-EVIDENCE-P0]
	svc, root := newTestService(t)
	scenarioDir := filepath.Join(root, "demo")
	runID := "artifacts"
	seedRecord(t, root, sharedruns.RunRecord{RunID: runID, Scenario: "demo", StartedAt: time.Now().UTC(), Status: sharedruns.StatusPassed})
	seedDescriptorSnapshot(t, root, runID, sharedruns.PhaseDescriptorSnapshot{
		Phase: "future-visual", EvidenceKinds: []string{sharedartifacts.ArtifactKindScreenshot},
		Applicability: sharedruns.ApplicabilityDecisionSnapshot{Status: "applies", Planned: true},
	})
	pageDir := filepath.Join(sharedartifacts.RunUISmokePagesDir(scenarioDir, runID), "home")
	if err := os.MkdirAll(pageDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pageDir, "page.json"), []byte(`{"page":"/home","label":"Home"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pageDir, "screenshot.png"), []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	catalog, err := sharedartifacts.DiscoverArtifactCatalog(
		scenarioDir, runID, artifactPhaseDeclarations(scenarioDir, runID), time.Unix(100, 0), false,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := sharedartifacts.WriteArtifactCatalog(scenarioDir, catalog); err != nil {
		t.Fatal(err)
	}

	listed, err := svc.ListRunArtifacts(context.Background(), connect.NewRequest(&runspb.ListRunArtifactsRequest{
		Scenario: "demo", RunId: runID, Kinds: []string{sharedartifacts.ArtifactKindScreenshot},
	}))
	if err != nil {
		t.Fatalf("ListRunArtifacts: %v", err)
	}
	if listed.Msg.GetLegacyDiscovered() || len(listed.Msg.GetArtifacts()) != 1 {
		t.Fatalf("artifact list = %+v", listed.Msg)
	}
	artifact := listed.Msg.GetArtifacts()[0]
	if artifact.GetProducingPhase() != "future-visual" || artifact.GetKind() != sharedartifacts.ArtifactKindScreenshot {
		t.Fatalf("artifact metadata = %+v", artifact)
	}
	if strings.Contains(artifact.GetAccessPath(), "ui-smoke") || strings.Contains(artifact.GetId(), "screenshot") {
		t.Fatalf("public reference leaks storage path: %+v", artifact)
	}
	if artifact.GetComparison().GetSemantics() != "advisory" {
		t.Fatalf("visual comparison semantics = %+v", artifact.GetComparison())
	}
	got, err := svc.GetRunArtifact(context.Background(), connect.NewRequest(&runspb.GetRunArtifactRequest{
		Scenario: "demo", RunId: runID, ArtifactId: artifact.GetId(),
	}))
	if err != nil {
		t.Fatalf("GetRunArtifact: %v", err)
	}
	if got.Msg.GetArtifact().GetId() != artifact.GetId() {
		t.Fatalf("detail = %+v", got.Msg)
	}
}

func TestRunArtifactCatalogLegacyAndCrossRunErrorsAreExplicit(t *testing.T) { // [REQ:TESTGENIE-TYPED-EVIDENCE-P0]
	svc, root := newTestService(t)
	scenarioDir := filepath.Join(root, "demo")
	for _, runID := range []string{"legacy-a", "legacy-b"} {
		seedRecord(t, root, sharedruns.RunRecord{RunID: runID, Scenario: "demo", StartedAt: time.Now().UTC(), Status: sharedruns.StatusPassed})
		runDir := sharedartifacts.RunDir(scenarioDir, runID)
		if err := os.MkdirAll(runDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(runDir, "future.bin"), []byte(runID), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	legacy, err := svc.ListRunArtifacts(context.Background(), connect.NewRequest(&runspb.ListRunArtifactsRequest{Scenario: "demo", RunId: "legacy-a"}))
	if err != nil {
		t.Fatal(err)
	}
	if !legacy.Msg.GetLegacyDiscovered() || len(legacy.Msg.GetDegradedReasons()) == 0 {
		t.Fatalf("legacy projection = %+v", legacy.Msg)
	}
	foreign, err := svc.ListRunArtifacts(context.Background(), connect.NewRequest(&runspb.ListRunArtifactsRequest{Scenario: "demo", RunId: "legacy-b"}))
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.GetRunArtifact(context.Background(), connect.NewRequest(&runspb.GetRunArtifactRequest{
		Scenario: "demo", RunId: "legacy-a", ArtifactId: foreign.Msg.GetArtifacts()[0].GetId(),
	}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("foreign artifact code = %s err=%v", connect.CodeOf(err), err)
	}
	_, err = svc.ListRunArtifacts(context.Background(), connect.NewRequest(&runspb.ListRunArtifactsRequest{Scenario: "../demo", RunId: "legacy-a"}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("scenario traversal code = %s err=%v", connect.CodeOf(err), err)
	}
}

func TestGetRunFindings(t *testing.T) {
	svc, root := newTestService(t)
	scenarioDir := filepath.Join(root, "demo")
	seedRecord(t, root, sharedruns.RunRecord{RunID: "r1", Scenario: "demo", StartedAt: time.Now().UTC(), Status: sharedruns.StatusPassed})

	runDir := sharedartifacts.RunDir(scenarioDir, "r1")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// The detailed findings document is deliberately opaque to the summary RPC.
	findings := `{
	  "scenario": "demo",
	  "runId": "r1",
	  "verdict": "pass",
	  "phases": [
	    {"name": "contracts", "status": "passed", "findingSource": "cli",
	     "phasePresentation": {"contract_version": "v1", "provider": "cli-health", "phase": "contracts", "current_level": "L3", "next_level": "L4", "ceiling_level": "L4", "north_star": "Verified primitives.", "next_action": "Prove the primitive.", "blocking_finding_codes": ["arch.primitive_unverified"]},
	     "findingsSummary": {"warnings": 1, "total": 1}}
	  ]
	}`
	if err := os.WriteFile(sharedartifacts.RunFindingsArtifactPath(scenarioDir, "r1"), []byte(findings), 0o644); err != nil {
		t.Fatal(err)
	}
	writer, err := executionevidence.NewWriter(runDir)
	if err != nil {
		t.Fatal(err)
	}
	ref, err := writer.ReferenceExisting("findings", "findings.document", sharedartifacts.FindingsArtifactFile, "application/json", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteManifest(executionevidence.Manifest{
		SchemaVersion: executionevidence.SchemaVersion, RunID: "r1", Scenario: "demo", Verdict: "pass", CreatedAt: time.Now().UTC(), Findings: ref,
		Phases: []executionevidence.PhaseSummary{{
			Name: "contracts", Status: "passed", FindingSource: "cli", FindingsSummary: &runspb.PhaseFindingsSummary{Warnings: 1, Total: 1},
			PhasePresentation: &commonv1.PhasePresentation{ContractVersion: "v1", Provider: "cli-health", Phase: "contracts", CurrentLevel: "L3", NextLevel: "L4", CeilingLevel: "L4", NorthStar: "Verified primitives.", NextAction: "Prove the primitive.", BlockingFindingCodes: []string{"arch.primitive_unverified"}},
		}},
	}); err != nil {
		t.Fatal(err)
	}
	// The summary projection is self-contained. Removing the detailed payload
	// proves this RPC cannot accidentally hydrate it to render phase standing.
	if err := os.Remove(sharedartifacts.RunFindingsArtifactPath(scenarioDir, "r1")); err != nil {
		t.Fatal(err)
	}

	resp, err := svc.GetRunFindings(context.Background(), connect.NewRequest(&runspb.GetRunFindingsRequest{Scenario: "demo", RunId: "r1"}))
	if err != nil {
		t.Fatalf("GetRunFindings: %v", err)
	}
	if len(resp.Msg.GetPhases()) != 1 {
		t.Fatalf("want 1 phase, got %d", len(resp.Msg.GetPhases()))
	}
	st := resp.Msg.GetPhases()[0].GetPhasePresentation()
	if st == nil || st.GetCurrentLevel() != "L3" || st.GetNextLevel() != "L4" || st.GetNorthStar() != "Verified primitives." {
		t.Fatalf("standing not read back from evidence manifest: %+v", st)
	}
	if st.GetNextAction() != "Prove the primitive." || len(st.GetBlockingFindingCodes()) != 1 {
		t.Fatalf("standing detail missing: %+v", st)
	}

	// "latest" resolves through the lightweight latest run manifest; findings
	// are not copied under coverage/latest.
	if err := os.MkdirAll(sharedartifacts.LatestDirPath(scenarioDir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sharedartifacts.LatestManifestPath(scenarioDir), []byte(`{"run_id":"r1"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GetRunFindings(context.Background(), connect.NewRequest(&runspb.GetRunFindingsRequest{Scenario: "demo", RunId: "latest"})); err != nil {
		t.Fatalf("GetRunFindings latest: %v", err)
	}

	// Missing artifact → NotFound.
	if _, err := svc.GetRunFindings(context.Background(), connect.NewRequest(&runspb.GetRunFindingsRequest{Scenario: "demo", RunId: "nope"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("expected NotFound, got %v", err)
	}
}

func TestGetRunFindingsDoesNotReadArtifactWithoutManifest(t *testing.T) {
	svc, root := newTestService(t)
	scenarioDir := filepath.Join(root, "demo")
	seedRecord(t, root, sharedruns.RunRecord{RunID: "historical", Scenario: "demo", StartedAt: time.Now().UTC(), Status: sharedruns.StatusPassed})
	if err := os.MkdirAll(sharedartifacts.RunDir(scenarioDir, "historical"), 0o755); err != nil {
		t.Fatal(err)
	}
	// A manifest-less historical artifact must not be decoded by the new runtime.
	findings := `{
	  "scenario": "demo",
	  "runId": "historical",
	  "verdict": "pass",
	  "phases": [{
	    "name": "ui-health",
	    "status": "passed",
	    "findingSource": "ui",
	    "maturityStanding": {"provider": "ui-health", "phase": "ui-health", "current_level": "L2", "next_level": "L3"}
	  }]
	}`
	if err := os.WriteFile(sharedartifacts.RunFindingsArtifactPath(scenarioDir, "historical"), []byte(findings), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := svc.GetRunFindings(context.Background(), connect.NewRequest(&runspb.GetRunFindingsRequest{Scenario: "demo", RunId: "historical"}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("manifest-less artifact code = %s err=%v", connect.CodeOf(err), err)
	}
}

// FindRun returns the newest completed run that matches every shape filter, and
// found=false when none does. It is the reuse primitive git-control-tower
// queries: only a clean, comprehensive+baseline run at the requested sha should
// match (so a quick run, a dirty run, or a different sha is never reused).
func TestFindRun(t *testing.T) {
	svc, root := newTestService(t)
	base := time.Now().UTC()
	comprehensivePhases := phases.DefaultPresets()[phases.PresetComprehensive.String()]
	comprehensiveDigest := phases.PhaseSetDigest(comprehensivePhases)
	// A matching clean comprehensive+baseline run at sha "abc".
	seedRecord(t, root, sharedruns.RunRecord{
		RunID: "match", Scenario: "demo", StartedAt: base, Status: sharedruns.StatusPassed,
		GitSha: "abc", Preset: "comprehensive", CaptureProfile: "baseline",
		PlannedPhases: comprehensivePhases, PhaseSetDigest: comprehensiveDigest,
	})
	// A newer run that should NOT match: different preset.
	seedRecord(t, root, sharedruns.RunRecord{
		RunID: "wrong-preset", Scenario: "demo", StartedAt: base.Add(time.Minute), Status: sharedruns.StatusPassed,
		GitSha: "abc", Preset: "quick", CaptureProfile: "baseline",
	})
	// A newer run that should NOT match: dirty tree.
	seedRecord(t, root, sharedruns.RunRecord{
		RunID: "dirty", Scenario: "demo", StartedAt: base.Add(2 * time.Minute), Status: sharedruns.StatusPassed,
		GitSha: "abc", Preset: "comprehensive", CaptureProfile: "baseline", GitDirty: true,
	})
	// A newer run that should NOT match: different sha.
	seedRecord(t, root, sharedruns.RunRecord{
		RunID: "other-sha", Scenario: "demo", StartedAt: base.Add(3 * time.Minute), Status: sharedruns.StatusPassed,
		GitSha: "def", Preset: "comprehensive", CaptureProfile: "baseline",
	})

	resp, err := svc.FindRun(context.Background(), connect.NewRequest(&runspb.FindRunRequest{
		Scenario: "demo", GitSha: "abc", Preset: "comprehensive", CaptureProfile: "baseline",
		Status: "passed", RequireClean: true,
	}))
	if err != nil {
		t.Fatalf("FindRun: %v", err)
	}
	if !resp.Msg.GetFound() {
		t.Fatal("expected a matching run")
	}
	if got := resp.Msg.GetRun().GetRunId(); got != "match" {
		t.Fatalf("FindRun matched %q, want the clean comprehensive+baseline run \"match\"", got)
	}

	// No run at this sha → found=false.
	miss, err := svc.FindRun(context.Background(), connect.NewRequest(&runspb.FindRunRequest{
		Scenario: "demo", GitSha: "zzz", Preset: "comprehensive", CaptureProfile: "baseline", RequireClean: true,
	}))
	if err != nil {
		t.Fatalf("FindRun(miss): %v", err)
	}
	if miss.Msg.GetFound() {
		t.Fatalf("expected no match for an unknown sha, got %q", miss.Msg.GetRun().GetRunId())
	}
}

func TestFindRunRequiresCurrentComprehensivePhaseSetDigest(t *testing.T) {
	svc, root := newTestService(t)
	base := time.Now().UTC()
	currentPhases := phases.DefaultPresets()[phases.PresetComprehensive.String()]
	currentDigest := phases.PhaseSetDigest(currentPhases)

	seedRecord(t, root, sharedruns.RunRecord{
		RunID: "stale-shape", Scenario: "demo", StartedAt: base, Status: sharedruns.StatusPassed,
		GitSha: "abc", Preset: "comprehensive", CaptureProfile: "baseline",
		PlannedPhases: []string{"structure", "unit"}, PhaseSetDigest: phases.PhaseSetDigest([]string{"structure", "unit"}),
	})
	seedRecord(t, root, sharedruns.RunRecord{
		RunID: "match-shape", Scenario: "demo", StartedAt: base.Add(time.Minute), Status: sharedruns.StatusPassed,
		GitSha: "abc", Preset: "comprehensive", CaptureProfile: "baseline",
		PlannedPhases: currentPhases, PhaseSetDigest: currentDigest,
	})

	resp, err := svc.FindRun(context.Background(), connect.NewRequest(&runspb.FindRunRequest{
		Scenario: "demo", GitSha: "abc", Preset: "comprehensive", CaptureProfile: "baseline",
		Status: "passed", RequireClean: true,
	}))
	if err != nil {
		t.Fatalf("FindRun: %v", err)
	}
	if !resp.Msg.GetFound() || resp.Msg.GetRun().GetRunId() != "match-shape" {
		t.Fatalf("FindRun matched %#v, want current phase-set run", resp.Msg.GetRun())
	}
	if got := resp.Msg.GetRun().GetPhaseSetDigest(); got != currentDigest {
		t.Fatalf("RunInfo phase_set_digest = %q, want %q", got, currentDigest)
	}
}

func TestFindRunGateQualityRejectsCleanButUnprovenHistory(t *testing.T) {
	svc, root := newTestService(t)
	seedLegacyRecord(t, root, sharedruns.RunRecord{RunID: "clean-history", Scenario: "demo", StartedAt: time.Now().UTC(), Status: sharedruns.StatusPassed, GitSha: "abc", Preset: "comprehensive", CaptureProfile: "baseline"})
	resp, err := svc.FindRun(context.Background(), connect.NewRequest(&runspb.FindRunRequest{Scenario: "demo", GitSha: "abc", Preset: "comprehensive", CaptureProfile: "baseline", RequireClean: true, RequireGateQuality: true}))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Msg.GetFound() {
		t.Fatalf("gate-quality reuse accepted historical run: %+v", resp.Msg.GetRun())
	}
}

// [REQ:TG-SHARED-WORKSPACE-CACHE-P0] Reuse keys are the declared scenario
// scope plus validation configuration, not the whole repository's dirty bit.
func TestFindRunMatchesCurrentScopedSourceAndConfiguration(t *testing.T) {
	svc, root := newTestService(t)
	scenarioDir := filepath.Join(root, "demo")
	if err := os.WriteFile(filepath.Join(scenarioDir, "relevant.txt"), []byte("before"), 0o644); err != nil {
		t.Fatal(err)
	}
	digest, err := treedigest.Compute(scenarioDir)
	if err != nil {
		t.Fatal(err)
	}
	planned := phases.DefaultPresets()[phases.PresetComprehensive.String()]
	phaseDigest := phases.PhaseSetDigest(planned)
	seedRecord(t, root, sharedruns.RunRecord{RunID: "matching", Scenario: "demo", StartedAt: time.Now().UTC(), Status: sharedruns.StatusPassed, TreeDigest: digest, Preset: "comprehensive", CaptureProfile: "baseline", PlannedPhases: planned, PhaseSetDigest: phaseDigest})
	index := sharedruns.NewIndex(scenarioDir)
	if err := index.Finalize("matching", &orchestrator.SuiteExecutionResult{SourceFingerprint: digest, SourceScope: "scenario:demo", SourceStable: true, ConfigurationFingerprint: "cfg:one"}, func(*sharedruns.RunRecord) error { return nil }); err != nil {
		t.Fatal(err)
	}
	svc.planner = fixedExecutionPlanner{preview: &execution.ExecutionPlanPreview{ScenarioName: "demo", PhaseSetDigest: phaseDigest, ConfigurationFingerprint: "cfg:one"}}
	find := func() *runspb.FindRunResponse {
		resp, err := svc.FindRun(context.Background(), connect.NewRequest(&runspb.FindRunRequest{Scenario: "demo", Preset: "comprehensive", CaptureProfile: "baseline", MatchCurrentSource: true}))
		if err != nil {
			t.Fatal(err)
		}
		return resp.Msg
	}
	if got := find(); !got.GetFound() || got.GetRun().GetRunId() != "matching" {
		t.Fatalf("matching shared scope was not reused: %+v", got)
	}
	if err := os.WriteFile(filepath.Join(root, "unrelated.txt"), []byte("other agent"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := find(); !got.GetFound() {
		t.Fatalf("unrelated workspace edit invalidated cache: %+v", got)
	}
	svc.planner = fixedExecutionPlanner{preview: &execution.ExecutionPlanPreview{ScenarioName: "demo", PhaseSetDigest: phaseDigest, ConfigurationFingerprint: "cfg:changed"}}
	if got := find(); got.GetFound() {
		t.Fatalf("validation configuration change incorrectly reused %q", got.GetRun().GetRunId())
	}
	svc.planner = fixedExecutionPlanner{preview: &execution.ExecutionPlanPreview{ScenarioName: "demo", PhaseSetDigest: phaseDigest, ConfigurationFingerprint: "cfg:one"}}
	if err := os.WriteFile(filepath.Join(scenarioDir, "relevant.txt"), []byte("after"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := find(); got.GetFound() {
		t.Fatalf("relevant source edit incorrectly reused %q", got.GetRun().GetRunId())
	}
}
