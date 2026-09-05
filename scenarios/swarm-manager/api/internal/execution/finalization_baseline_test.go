package execution

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func newBaselineDiffTestService(t *testing.T, fake BaselineClient, enabled bool) (*Service, string) {
	t.Helper()
	storePath := filepath.Join(t.TempDir(), "exec.json")
	svc := &Service{
		store:          NewStore(storePath),
		baselineClient: fake,
		finalizationCfg: FinalizationConfig{
			BaselineDiffEnabled: enabled,
			BaselineDiffTimeout: 30 * time.Second,
		},
	}
	rec := Record{
		ExecutionID: "e1",
		Status:      StatusValidating,
		Finalization: &Finalization{
			Eligible:          true,
			Status:            FinalizationStatusRunning,
			Phase:             FinalizationPhaseRestarting,
			AffectedScenarios: []string{"alpha", "beta"},
			Scenarios: []ScenarioFinalization{
				{ScenarioName: "alpha"},
				{ScenarioName: "beta"},
			},
		},
		PreExecBaselines: map[string]string{"alpha": "preexec-alpha-abc123"},
	}
	if err := svc.store.Save([]Record{rec}); err != nil {
		t.Fatalf("seed record: %v", err)
	}
	return svc, "e1"
}

func loadScenarioDiff(t *testing.T, svc *Service, executionID, scenario string) *BaselineDiffResult {
	t.Helper()
	sf, err := svc.loadScenarioFinalization(executionID, scenario)
	if err != nil {
		t.Fatalf("load scenario %s: %v", scenario, err)
	}
	return sf.BaselineDiff
}

func TestRunBaselineDiffs_RegressionAndScopeExpansion(t *testing.T) {
	fake := newFakeBaselineClient()
	fake.diffs["alpha"] = BaselineDiffResult{
		ScenarioName: "alpha",
		Verdict:      baselineVerdictRegression,
		ExitCode:     1,
		Comparable:   true,
		Regressions:  []SurfaceFinding{{Surface: "tests", Detail: "TestNew"}},
	}
	svc, execID := newBaselineDiffTestService(t, fake, true)

	scope := finalizationScope{affectedScenarios: []string{"alpha", "beta"}}
	preExec := map[string]string{"alpha": "preexec-alpha-abc123"} // beta intentionally absent

	if err := svc.runBaselineDiffs(context.Background(), execID, scope, preExec); err != nil {
		t.Fatalf("runBaselineDiffs: %v", err)
	}

	// alpha: real diff with a regression.
	alpha := loadScenarioDiff(t, svc, execID, "alpha")
	if alpha == nil || alpha.Verdict != baselineVerdictRegression || !alpha.HasNewRegressions() {
		t.Fatalf("alpha diff = %+v, want regression with new regressions", alpha)
	}

	// beta: no pre-exec baseline → not_comparable.
	beta := loadScenarioDiff(t, svc, execID, "beta")
	if beta == nil || beta.Comparable || beta.Verdict != baselineVerdictNotComparable {
		t.Fatalf("beta diff = %+v, want not_comparable", beta)
	}

	// A scope-expansion warning should have been emitted for beta.
	rec, _, err := svc.loadRecordLocked(execID)
	if err != nil {
		t.Fatalf("load record: %v", err)
	}
	foundWarning := false
	foundRegressionWarning := false
	for _, w := range rec[0].Finalization.Warnings {
		if w.Code == finalizationWarningBaselineScopeExpanded && w.ScenarioName == "beta" {
			foundWarning = true
		}
		if w.Code == finalizationWarningBaselineRegression && w.ScenarioName == "alpha" {
			foundRegressionWarning = true
		}
	}
	if !foundWarning {
		t.Errorf("expected scope-expanded warning for beta, warnings=%+v", rec[0].Finalization.Warnings)
	}
	// alpha's regression must surface as a first-class, attributable warning
	// (the audit/UI signal paired with the summarizeFinalization gate).
	if !foundRegressionWarning {
		t.Errorf("expected baseline-regression warning for alpha, warnings=%+v", rec[0].Finalization.Warnings)
	}

	// Phase should have advanced to baseline_diff.
	if rec[0].Finalization.Phase != FinalizationPhaseBaselineDiff {
		t.Errorf("phase = %q, want %q", rec[0].Finalization.Phase, FinalizationPhaseBaselineDiff)
	}
}

func TestRunBaselineDiffs_DiffErrorIsNotComparable(t *testing.T) {
	fake := newFakeBaselineClient()
	fake.diffErr = context.DeadlineExceeded
	svc, execID := newBaselineDiffTestService(t, fake, true)

	scope := finalizationScope{affectedScenarios: []string{"alpha"}}
	preExec := map[string]string{"alpha": "preexec-alpha-abc123"}

	if err := svc.runBaselineDiffs(context.Background(), execID, scope, preExec); err != nil {
		t.Fatalf("runBaselineDiffs should swallow diff errors, got %v", err)
	}
	alpha := loadScenarioDiff(t, svc, execID, "alpha")
	if alpha == nil || alpha.Comparable {
		t.Fatalf("diff error should yield not_comparable, got %+v", alpha)
	}
}

func TestCleanupPreExecBaselines_DeletesUnlessRetained(t *testing.T) {
	preExec := map[string]string{
		"alpha": "preexec-alpha-abc",
		"beta":  "preexec-beta-def",
	}

	// Default: delete every captured baseline.
	fake := newFakeBaselineClient()
	svc := &Service{baselineClient: fake, finalizationCfg: FinalizationConfig{BaselineDiffEnabled: true}}
	svc.cleanupPreExecBaselines(context.Background(), preExec)
	if got := len(fake.deletedKeys()); got != 2 {
		t.Fatalf("expected 2 deletes, got %d (%v)", got, fake.deletedKeys())
	}

	// Retain: delete nothing.
	retainFake := newFakeBaselineClient()
	retainSvc := &Service{baselineClient: retainFake, finalizationCfg: FinalizationConfig{BaselineDiffEnabled: true, BaselineRetainAfterFinalization: true}}
	retainSvc.cleanupPreExecBaselines(context.Background(), preExec)
	if got := len(retainFake.deletedKeys()); got != 0 {
		t.Fatalf("retain flag should skip deletes, got %d (%v)", got, retainFake.deletedKeys())
	}

	// Nil client: no panic, no-op.
	nilSvc := &Service{finalizationCfg: FinalizationConfig{BaselineDiffEnabled: true}}
	nilSvc.cleanupPreExecBaselines(context.Background(), preExec)
}

func TestRunBaselineDiffs_DisabledOrNoClientSkips(t *testing.T) {
	// Disabled feature: no phase change, no diff recorded.
	fake := newFakeBaselineClient()
	svc, execID := newBaselineDiffTestService(t, fake, false)
	if err := svc.runBaselineDiffs(context.Background(), execID, finalizationScope{affectedScenarios: []string{"alpha"}}, map[string]string{"alpha": "x"}); err != nil {
		t.Fatalf("disabled runBaselineDiffs: %v", err)
	}
	if d := loadScenarioDiff(t, svc, execID, "alpha"); d != nil {
		t.Errorf("disabled feature must not record a diff, got %+v", d)
	}
	rec, _, _ := svc.loadRecordLocked(execID)
	if rec[0].Finalization.Phase == FinalizationPhaseBaselineDiff {
		t.Errorf("disabled feature must not advance to baseline_diff phase")
	}

	// Nil client also skips.
	svc2, execID2 := newBaselineDiffTestService(t, nil, true)
	svc2.baselineClient = nil
	if err := svc2.runBaselineDiffs(context.Background(), execID2, finalizationScope{affectedScenarios: []string{"alpha"}}, map[string]string{"alpha": "x"}); err != nil {
		t.Fatalf("nil-client runBaselineDiffs: %v", err)
	}
	if d := loadScenarioDiff(t, svc2, execID2, "alpha"); d != nil {
		t.Errorf("nil client must not record a diff, got %+v", d)
	}
}
