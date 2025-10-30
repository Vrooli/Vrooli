package runtime

import (
	"context"
	"testing"

	"github.com/vrooli/browser-automation-studio/browserless/compiler"
	"github.com/vrooli/browser-automation-studio/internal/scenarioport"
)

func TestInstructionFromStepScenario(t *testing.T) {
	restore := scenarioport.SetPortLookupFuncForTests(func(ctx context.Context, scenario string, port string) (int, error) {
		return 4242, nil
	})
	defer restore()

	step := compiler.ExecutionStep{
		Index:  0,
		NodeID: "navigate-1",
		Type:   compiler.StepNavigate,
		Params: map[string]any{
			"destinationType": "scenario",
			"scenario":        "app-monitor",
			"scenarioPath":    "/dashboard",
		},
	}

	instruction, err := instructionFromStep(context.Background(), step)
	if err != nil {
		t.Fatalf("instructionFromStep returned error: %v", err)
	}

	expectedURL := "http://localhost:4242/dashboard"
	if instruction.Params.URL != expectedURL {
		t.Fatalf("expected resolved URL %q, got %q", expectedURL, instruction.Params.URL)
	}
}

func TestInstructionFromStepScenarioMissingName(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  0,
		NodeID: "navigate-2",
		Type:   compiler.StepNavigate,
		Params: map[string]any{
			"destinationType": "scenario",
		},
	}

	if _, err := instructionFromStep(context.Background(), step); err == nil {
		t.Fatalf("expected error when scenario name is missing")
	}
}

func TestInstructionFromStepURLFallback(t *testing.T) {
	step := compiler.ExecutionStep{
		Index:  0,
		NodeID: "navigate-3",
		Type:   compiler.StepNavigate,
		Params: map[string]any{
			"url": " https://example.com ",
		},
	}

	instruction, err := instructionFromStep(context.Background(), step)
	if err != nil {
		t.Fatalf("unexpected error converting navigate step: %v", err)
	}

	if instruction.Params.URL != "https://example.com" {
		t.Fatalf("expected URL to be trimmed, got %q", instruction.Params.URL)
	}
}
