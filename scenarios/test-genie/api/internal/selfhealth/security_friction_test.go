package selfhealth

import (
	"context"
	"testing"
	"time"

	"test-genie/internal/execution"
)

type frictionSource struct{ observations []execution.PhaseObservation }

func (s frictionSource) AggregatePhaseObservations(context.Context, time.Time, int) ([]execution.PhaseObservation, error) {
	return s.observations, nil
}

func (frictionSource) CountRunOutcomes(context.Context, time.Time, int) ([]execution.RunOutcomeCount, error) {
	return nil, nil
}

func TestSecurityFrictionKeepsActionableFailureTaxonomy(t *testing.T) {
	now := time.Now().UTC()
	source := frictionSource{observations: []execution.PhaseObservation{
		{ScenarioName: "demo", PhaseName: "security", Status: "failed", Classification: "unknown-provider-unavailable-blocked", RunnabilityReason: "repair required", CompletedAt: now},
		{ScenarioName: "demo", PhaseName: "security", Status: "passed", Classification: "repair-applied", CompletedAt: now.Add(time.Minute)},
	}}
	ledger, err := NewBuilder(source, time.Hour).Build(context.Background(), map[string]PhaseMeta{"security": {FindingSource: "security"}})
	if err != nil {
		t.Fatal(err)
	}
	friction := ledger.Phases[0].SecurityFriction
	if friction.UnknownFindings == 0 || friction.BlockedActions == 0 || friction.RepairAttempts == 0 || friction.RepairSuccesses == 0 || friction.ProviderOutages == 0 {
		t.Fatalf("friction taxonomy = %+v", friction)
	}
}
