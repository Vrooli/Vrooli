package autosteer

import (
	"testing"

	"github.com/ecosystem-manager/api/pkg/effectiveness"
	"github.com/ecosystem-manager/api/pkg/findings"
	"github.com/ecosystem-manager/api/pkg/skillmap"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

// mixedAudit builds an audit with standards findings and (failing) unit-test
// findings, so credit assignment can be checked across dimensions.
func mixedAudit(standardsIDs, testsIDs []string) *findings.Audit {
	errSev := int32(architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR)
	stdF := make([]findings.AuditFinding, 0, len(standardsIDs))
	for _, id := range standardsIDs {
		stdF = append(stdF, findings.AuditFinding{
			Source:   int32(architecturev1.FindingSource_FINDING_SOURCE_STANDARDS),
			Severity: errSev, Code: "STD", StableID: id,
		})
	}
	testF := make([]findings.AuditFinding, 0, len(testsIDs))
	for _, id := range testsIDs {
		// Source unspecified → dimension resolves from the "unit" phase → tests.
		testF = append(testF, findings.AuditFinding{Severity: errSev, Code: "TST", StableID: id})
	}
	status := func(n int) string {
		if n == 0 {
			return "pass"
		}
		return "fail"
	}
	return &findings.Audit{
		ScenarioName: "demo",
		Phases: []findings.AuditPhase{
			{Name: "standards", Status: status(len(standardsIDs)), Findings: stdF},
			{Name: "unit", Status: status(len(testsIDs)), Findings: testF},
		},
	}
}

// creditOrchestrator builds an orchestrator over a held effectiveness store.
func creditOrchestrator(runner findings.AuditRunner) (*ExecutionOrchestrator, *effectiveness.MemoryStore) {
	stateRepo := NewMockExecutionStateRepository()
	profileRepo := NewMockProfileRepository()
	_ = profileRepo.CreateProfile(&AutoSteerProfile{
		ID:   "demo",
		Name: "Demo",
		Objective: Objective{
			DimensionWeights: map[string]float64{"standards": 1.0},
			Targets:          ObjectiveTargets{MaxOpenSeverity: "warning"},
		},
		AllowedSkills: []string{"refactor"},
		Budget:        Budget{MaxIterations: 10, ReauditCadence: 1},
		AuditPreset:   "comprehensive",
	})
	catalog := &skillmap.FakeCatalog{Declarations: []skillmap.SkillDeclaration{
		{ID: "refactor", Dimensions: []string{"standards"}},
	}}
	store := effectiveness.NewMemoryStore()
	orch := NewExecutionOrchestrator(
		stateRepo, profileRepo, runner, catalog,
		NewMockPromptEnhancerAPI(), NewMockCompletenessProvider(), NewTraceStore(nil), store,
	)
	return orch, store
}

func TestCreditAssignment_ClosedAndCollateral(t *testing.T) { // [REQ:EM-P1-003]
	// Diagnose: standards {a,b}. After iter 1: standards {a} (b closed),
	// tests {t1} introduced as collateral damage from the standards skill.
	runner := &shrinkingRunner{audits: []*findings.Audit{
		mixedAudit([]string{"a", "b"}, nil),
		mixedAudit([]string{"a"}, []string{"t1"}),
	}}
	orch, store := creditOrchestrator(runner)

	taskID := "credit-task"
	if _, err := orch.StartExecution(taskID, "demo", "demo"); err != nil {
		t.Fatalf("StartExecution: %v", err)
	}
	orch.RecordRunCost(taskID, RunCost{TotalTokens: 2000})

	if _, err := orch.EvaluateIteration(taskID, "demo"); err != nil {
		t.Fatalf("EvaluateIteration: %v", err)
	}

	// Target dimension (standards): 1 closed, 1 run, full token cost.
	std, ok, _ := store.Get("refactor", "standards")
	if !ok || std.ClosedCount != 1 || std.IntroducedCount != 0 || std.TotalRuns != 1 || std.TotalTokens != 2000 {
		t.Fatalf("standards credit wrong: %+v (ok=%v)", std, ok)
	}

	// Collateral dimension (tests): 1 introduced, but no run/token attributed.
	tst, ok, _ := store.Get("refactor", "tests")
	if !ok || tst.IntroducedCount != 1 || tst.ClosedCount != 0 || tst.TotalRuns != 0 || tst.TotalTokens != 0 {
		t.Fatalf("tests collateral credit wrong: %+v (ok=%v)", tst, ok)
	}

	// The trace's first iteration records the split and token cost.
	state, err := orch.GetExecutionState(taskID)
	if err != nil || state == nil {
		t.Fatalf("GetExecutionState: %v", err)
	}
	e := state.Trace[0]
	if e.ClosedByDimension["standards"] != 1 {
		t.Fatalf("trace closed_by_dimension wrong: %+v", e.ClosedByDimension)
	}
	if e.IntroducedByDimension["tests"] != 1 {
		t.Fatalf("trace introduced_by_dimension wrong: %+v", e.IntroducedByDimension)
	}
	if e.TokensUsed != 2000 {
		t.Fatalf("trace tokens_used wrong: %d", e.TokensUsed)
	}
}
