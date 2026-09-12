package journeys

import (
	"context"
	"testing"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

func TestPlanDeclaresTwelveBoundedChapters(t *testing.T) {
	plan := Plan()
	if len(plan.Steps) != 12 {
		t.Fatalf("chapter count = %d", len(plan.Steps))
	}
	for _, step := range plan.Steps {
		if step.Readiness.Timeout == 0 || step.Settle.Maximum == 0 || step.Assertion == nil || step.Arguments["required_capability"] == "" {
			t.Fatalf("incomplete chapter: %+v", step)
		}
	}
}

func TestDriverNeverPassesWithoutAppleRuntime(t *testing.T) {
	result, err := (Driver{GOOS: "linux"}).Execute(context.Background(), deliveryramp.DriverRequest{RunID: "run-1", Plan: Plan()})
	if err != nil {
		t.Fatal(err)
	}
	if result.Disposition != deliveryramp.DispositionUnavailable || len(result.Steps) != 12 {
		t.Fatalf("result = %+v", result)
	}
	for _, step := range result.Steps {
		if step.Disposition != deliveryramp.StepUnavailable || step.Error == "" {
			t.Fatalf("dishonest step: %+v", step)
		}
	}
}
