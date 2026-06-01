package autosteer

import (
	"context"
	"strings"
	"testing"

	"github.com/ecosystem-manager/api/pkg/effectiveness"
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
		effectiveness.NewMemoryStore(),
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

// skippedAudit simulates an inconclusive DIAGNOSE: phases planned but none
// executed (a cold/unwarmed test-genie), which must not be read as "clean".
func skippedAudit() *findings.Audit {
	return &findings.Audit{
		ScenarioName: "demo",
		Phases: []findings.AuditPhase{
			{Name: "standards", Status: "skipped"},
			{Name: "docs", Status: "pending"},
		},
	}
}

func TestFullAudit_RetriesInconclusiveAudit(t *testing.T) {
	prev := auditConclusiveBackoff
	auditConclusiveBackoff = 0 // no sleep in tests
	defer func() { auditConclusiveBackoff = prev }()

	// First audit is inconclusive (all skipped), second has real findings. The
	// controller must retry and end up steering on the conclusive result rather
	// than mistaking the empty first result for an objective-met scenario.
	runner := &shrinkingRunner{audits: []*findings.Audit{skippedAudit(), standardsAudit(2)}}
	orch, _, _ := loopOrchestrator(runner)

	state, err := orch.StartExecution("task-retry", "demo", "demo")
	if err != nil {
		t.Fatalf("StartExecution error: %v", err)
	}
	if state.Findings.TotalScore == 0 {
		t.Fatal("expected the retry to recover real findings, got an empty (inconclusive) state")
	}
	if state.CurrentSkill != "refactor" {
		t.Fatalf("expected a skill selected from the conclusive audit, got %q", state.CurrentSkill)
	}
}

func TestEvaluateStart_ObjectiveMetSkipsFirstRun(t *testing.T) {
	// Initial audit is clean (no findings) and the demo profile has no
	// operational-target gate → the objective is already met at start. The
	// controller must NOT run a blind agent pass; it finalizes immediately.
	runner := &shrinkingRunner{audits: []*findings.Audit{standardsAudit(0)}}
	orch, stateRepo, _ := loopOrchestrator(runner)

	taskID := "task-met-at-start"
	if _, err := orch.StartExecution(taskID, "demo", "demo"); err != nil {
		t.Fatalf("StartExecution error: %v", err)
	}

	proceed, reason, err := orch.EvaluateStart(taskID, "demo")
	if err != nil {
		t.Fatalf("EvaluateStart error: %v", err)
	}
	if proceed {
		t.Fatal("expected proceed=false: objective already met at start")
	}
	if !strings.Contains(reason, "objective met") {
		t.Fatalf("expected objective-met reason, got %q", reason)
	}
	if got, _ := stateRepo.Get(taskID); got != nil {
		t.Fatal("expected state finalized/deleted when objective met at start")
	}
}

func TestEvaluateStart_FindingsPresentProceeds(t *testing.T) {
	// Initial audit has open ERROR findings → a skill is selected and the agent
	// run is warranted.
	runner := &shrinkingRunner{audits: []*findings.Audit{standardsAudit(2)}}
	orch, _, _ := loopOrchestrator(runner)

	taskID := "task-has-work"
	if _, err := orch.StartExecution(taskID, "demo", "demo"); err != nil {
		t.Fatalf("StartExecution error: %v", err)
	}

	proceed, reason, err := orch.EvaluateStart(taskID, "demo")
	if err != nil {
		t.Fatalf("EvaluateStart error: %v", err)
	}
	if !proceed {
		t.Fatalf("expected proceed=true with open findings, got stop (%s)", reason)
	}
}

func TestEvaluateStart_UnmetTargetsButNothingSteerableStops(t *testing.T) {
	// The bookmark-intelligence-hub shape: the audit is clean (no findings to
	// steer) but operational targets are unmet (50% < 90%). The objective is NOT
	// met, yet there is no skill to select — so the controller must stop with
	// "nothing actionable" rather than launch an unsteered pass.
	stateRepo := NewMockExecutionStateRepository()
	profileRepo := NewMockProfileRepository()
	_ = profileRepo.CreateProfile(&AutoSteerProfile{
		ID:   "ot-demo",
		Name: "OT Demo",
		Objective: Objective{
			DimensionWeights: map[string]float64{"standards": 1.0},
			Targets:          ObjectiveTargets{MaxOpenSeverity: "warning", OperationalTargetsPct: 90},
		},
		AllowedSkills: []string{"refactor"},
		Budget:        Budget{MaxIterations: 10, ReauditCadence: 1},
		AuditPreset:   "comprehensive",
	})
	catalog := &skillmap.FakeCatalog{Declarations: []skillmap.SkillDeclaration{
		{ID: "refactor", Dimensions: []string{"standards"}},
	}}
	metrics := NewMockMetricsProvider()
	metrics.Metrics.OperationalTargetsTotal = 10  // targets declared…
	metrics.Metrics.OperationalTargetsPassing = 5 // …but only half pass (50% < 90%)
	metrics.Metrics.OperationalTargetsPercentage = 50
	orch := NewExecutionOrchestrator(
		stateRepo, profileRepo, &shrinkingRunner{audits: []*findings.Audit{standardsAudit(0)}},
		catalog, NewMockPromptEnhancerAPI(), metrics, NewTraceStore(nil),
		effectiveness.NewMemoryStore(),
	)

	taskID := "task-unmet-but-nothing"
	if _, err := orch.StartExecution(taskID, "ot-demo", "demo"); err != nil {
		t.Fatalf("StartExecution error: %v", err)
	}

	proceed, reason, err := orch.EvaluateStart(taskID, "demo")
	if err != nil {
		t.Fatalf("EvaluateStart error: %v", err)
	}
	if proceed {
		t.Fatal("expected proceed=false: nothing steerable even though targets unmet")
	}
	if reason != StopNothingActionable {
		t.Fatalf("expected reason %q, got %q", StopNothingActionable, reason)
	}
	if got, _ := stateRepo.Get(taskID); got != nil {
		t.Fatal("expected state finalized/deleted")
	}
}

// growingRunner returns an ever-larger distinct findings set on each call, so
// the open set always changes (no fingerprint cycle) and net findings flow is
// always negative (no net-progress stall) — isolating the budget cap as the
// halting reason. The objective is never met (findings grow).
type growingRunner struct{ call int }

func (r *growingRunner) Audit(_ context.Context, _ findings.AuditRequest) (*findings.Audit, error) {
	r.call++
	return standardsAudit(r.call + 1), nil // call 1 → {a,b}, call 2 → {a,b,c}, …
}

func TestController_BudgetCapHalts(t *testing.T) {
	// A target that keeps growing: not a cycle, not a net-progress stall, never
	// meets the objective → only the budget cap stops it.
	runner := &growingRunner{}
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
