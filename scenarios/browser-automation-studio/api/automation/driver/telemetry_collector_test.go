package driver

import (
	"context"
	"testing"

	"github.com/vrooli/browser-automation-studio/automation/contracts"
)

func TestNoOpCollector_IsInert(t *testing.T) {
	var c TelemetryCollector = NoOpCollector{}
	ctx := context.Background()
	instr := contracts.CompiledInstruction{NodeID: "n1"}

	// BeforeStep must return the same context unchanged.
	got := c.BeforeStep(ctx, instr)
	if got != ctx {
		t.Fatalf("NoOpCollector.BeforeStep must return ctx unchanged")
	}

	// AfterStep must not panic for any outcome/error combination.
	c.AfterStep(ctx, instr, contracts.StepOutcome{Success: true}, nil)
	c.AfterStep(ctx, instr, contracts.StepOutcome{}, context.Canceled)
}
