package autosteer

import (
	"context"
	"strings"
	"testing"

	"github.com/ecosystem-manager/api/pkg/findings"
	"github.com/ecosystem-manager/api/pkg/skillmap"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

// shrinkingRunner returns a pre-scripted sequence of audits (one per call),
// clamping to the last once exhausted — simulating a target whose findings
// shrink as skills run.
type shrinkingRunner struct {
	audits []*findings.Audit
	call   int
}

func (r *shrinkingRunner) Audit(_ context.Context, _ findings.AuditRequest) (*findings.Audit, error) {
	a := r.audits[r.call]
	if r.call < len(r.audits)-1 {
		r.call++
	}
	return a, nil
}

func standardsAudit(errs int) *findings.Audit {
	f := make([]findings.AuditFinding, errs)
	for i := 0; i < errs; i++ {
		f[i] = findings.AuditFinding{
			Source:   int32(architecturev1.FindingSource_FINDING_SOURCE_STANDARDS),
			Severity: int32(architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR),
			Code:     "STD",
			StableID: string(rune('a' + i)),
		}
	}
	status := "fail"
	if errs == 0 {
		status = "pass"
	}
	return &findings.Audit{
		ScenarioName: "demo",
		Phases:       []findings.AuditPhase{{Name: "standards", Status: status, Findings: f}},
	}
}

func loopOrchestrator(runner findings.AuditRunner) (*ExecutionOrchestrator, *MockExecutionStateRepository, *MockProfileRepository) {
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
	orch := NewExecutionOrchestrator(
		stateRepo, profileRepo, runner, catalog,
		NewMockPromptEnhancerAPI(), NewMockMetricsProvider(), NewTraceStore(nil),
	)
	return orch, stateRepo, profileRepo
}

func TestController_MiniLoop_ShrinksAndTerminates(t *testing.T) {
	runner := &shrinkingRunner{audits: []*findings.Audit{
		standardsAudit(2), // initial diagnose: score 8
		standardsAudit(1), // after iter 1: score 4
		standardsAudit(0), // after iter 2: objective met
	}}
	orch, stateRepo, _ := loopOrchestrator(runner)

	taskID := "task-loop"
	state, err := orch.StartExecution(taskID, "demo", "demo")
	if err != nil {
		t.Fatalf("StartExecution error: %v", err)
	}
	if state.CurrentSkill != "refactor" {
		t.Fatalf("expected first skill 'refactor', got %q", state.CurrentSkill)
	}
	if state.Iteration != 1 {
		t.Fatalf("expected Iteration=1 after start, got %d", state.Iteration)
	}
	if state.Findings.TotalScore != 8 {
		t.Fatalf("expected initial score 8, got %v", state.Findings.TotalScore)
	}

	// Iteration 1: re-audit shrinks 8 → 4, still an ERROR open → continue.
	eval, err := orch.EvaluateIteration(taskID, "demo")
	if err != nil {
		t.Fatalf("EvaluateIteration #1 error: %v", err)
	}
	if eval.ShouldStop {
		t.Fatalf("expected to continue after iter 1, got stop (%s)", eval.Reason)
	}
	if eval.ChosenSkill != "refactor" {
		t.Fatalf("expected to re-select 'refactor', got %q", eval.ChosenSkill)
	}

	// Iteration 2: re-audit clears findings → objective met → stop + finalize.
	eval, err = orch.EvaluateIteration(taskID, "demo")
	if err != nil {
		t.Fatalf("EvaluateIteration #2 error: %v", err)
	}
	if !eval.ShouldStop || !strings.Contains(eval.Reason, "objective met") {
		t.Fatalf("expected objective-met stop, got stop=%v reason=%q", eval.ShouldStop, eval.Reason)
	}

	// Finalized → live state deleted.
	if got, _ := stateRepo.Get(taskID); got != nil {
		t.Fatal("expected state finalized/deleted after objective met")
	}
	if len(stateRepo.FinalizedTasks) != 1 {
		t.Fatalf("expected 1 finalized task, got %d", len(stateRepo.FinalizedTasks))
	}
}

func TestController_BudgetCapHalts(t *testing.T) {
	// Findings never shrink (stuck at score 4) → only the budget cap stops it.
	runner := &shrinkingRunner{audits: []*findings.Audit{standardsAudit(1)}}
	orch, stateRepo, profileRepo := loopOrchestrator(runner)
	// Tighten the budget for a fast cap, disable diminishing-returns so the cap
	// is the halting reason.
	p, _ := profileRepo.GetProfile("demo")
	p.Budget = Budget{MaxIterations: 3, ReauditCadence: 1}
	_ = profileRepo.UpdateProfile("demo", p)

	taskID := "task-budget"
	if _, err := orch.StartExecution(taskID, "demo", "demo"); err != nil {
		t.Fatalf("StartExecution error: %v", err)
	}

	var lastReason string
	stopped := false
	for i := 0; i < 10; i++ {
		eval, err := orch.EvaluateIteration(taskID, "demo")
		if err != nil {
			t.Fatalf("EvaluateIteration error: %v", err)
		}
		lastReason = eval.Reason
		if eval.ShouldStop {
			stopped = true
			break
		}
	}
	if !stopped {
		t.Fatal("expected the loop to halt")
	}
	if !strings.Contains(lastReason, StopBudgetExhausted) {
		t.Fatalf("expected budget_exhausted halt, got %q", lastReason)
	}
	if got, _ := stateRepo.Get(taskID); got != nil {
		t.Fatal("expected state finalized after budget cap")
	}
}
