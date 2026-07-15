package operatingmode

import (
	"context"
	"testing"
)

type fakePlanContainment struct {
	scope ContainmentScope
	found bool
	err   error
}

func (f fakePlanContainment) ContainmentForPlan(string) (ContainmentScope, bool, error) {
	return f.scope, f.found, f.err
}

// TestPlanExecutionSpawnInheritsContainmentScope proves the generic containment
// seam: when the plan-execution target adapter resolves a write-scope (from the
// backlog item that owns the plan_ref), the engine threads it verbatim into the
// agent spawn — so a plan-execution drain runs sandbox-scoped exactly as the
// item's legacy execution did.
func TestPlanExecutionSpawnInheritsContainmentScope(t *testing.T) {
	agent := &fakeAgent{}
	svc := newTestService(t, t.TempDir(), agent, &fakePrompts{})
	svc.SetPlanContainmentResolver(fakePlanContainment{
		scope: ContainmentScope{
			AcceptanceAllow: []string{"scenarios/foo/**"},
			AcceptanceDeny:  []string{"scenarios/foo/secret/**"},
			Creates:         []string{"scenarios/foo/new.go"},
		},
		found: true,
	})
	if _, err := svc.StartTargetPhase(context.Background(), StartTargetPhaseRequest{
		Mode: string(ModePhasedPlanDrain), TargetRef: "scoped-plan",
	}); err != nil {
		t.Fatalf("StartTargetPhase: %v", err)
	}
	if len(agent.spawned) != 1 {
		t.Fatalf("expected 1 spawn, got %d", len(agent.spawned))
	}
	got := agent.spawned[0]
	if len(got.AcceptanceAllow) != 1 || got.AcceptanceAllow[0] != "scenarios/foo/**" {
		t.Fatalf("acceptance_allow not threaded: %#v", got.AcceptanceAllow)
	}
	if len(got.AcceptanceDeny) != 1 || got.AcceptanceDeny[0] != "scenarios/foo/secret/**" {
		t.Fatalf("acceptance_deny not threaded: %#v", got.AcceptanceDeny)
	}
	if len(got.Creates) != 1 || got.Creates[0] != "scenarios/foo/new.go" {
		t.Fatalf("creates not threaded: %#v", got.Creates)
	}
}

// TestScenarioSpecSyncSpawnScopedToScenarioDir proves the scenario target adapter
// projects a non-zero write-scope from the ref alone (no resolver seam), so a
// spec-sync run is sandbox-scoped to the scenario's own directory exactly as the
// legacy spawn's ScopePath=ScenarioPath was. This fail-safe is load-bearing: the
// archive flow RemoveAll's the scenario directory on completion, so the agent must
// never be able to write repo-wide.
func TestScenarioSpecSyncSpawnScopedToScenarioDir(t *testing.T) {
	agent := &fakeAgent{}
	svc := newTestService(t, t.TempDir(), agent, &fakePrompts{})
	if _, err := svc.StartTargetPhase(context.Background(), StartTargetPhaseRequest{
		Mode: "scenario-spec-sync", TargetRef: "target-scenario",
	}); err != nil {
		t.Fatalf("StartTargetPhase: %v", err)
	}
	if len(agent.spawned) != 1 {
		t.Fatalf("expected 1 spawn, got %d", len(agent.spawned))
	}
	got := agent.spawned[0]
	if len(got.AcceptanceAllow) != 1 || got.AcceptanceAllow[0] != "scenarios/target-scenario/**" {
		t.Fatalf("spec-sync spawn not scoped to the scenario dir: allow=%#v", got.AcceptanceAllow)
	}
}

// TestPlanExecutionSpawnUnconstrainedWithoutContainment proves a scopeless
// target produces an unconstrained spawn identical to today: no resolver wired
// (or no owning item) means the engine sets no acceptance scope.
func TestPlanExecutionSpawnUnconstrainedWithoutContainment(t *testing.T) {
	agent := &fakeAgent{}
	svc := newTestService(t, t.TempDir(), agent, &fakePrompts{})
	if _, err := svc.StartTargetPhase(context.Background(), StartTargetPhaseRequest{
		Mode: string(ModePhasedPlanDrain), TargetRef: "unscoped-plan",
	}); err != nil {
		t.Fatalf("StartTargetPhase: %v", err)
	}
	if len(agent.spawned) != 1 {
		t.Fatalf("expected 1 spawn, got %d", len(agent.spawned))
	}
	if got := agent.spawned[0]; len(got.AcceptanceAllow) != 0 || len(got.AcceptanceDeny) != 0 || len(got.Creates) != 0 {
		t.Fatalf("expected unconstrained spawn, got allow=%#v deny=%#v creates=%#v", got.AcceptanceAllow, got.AcceptanceDeny, got.Creates)
	}
}
