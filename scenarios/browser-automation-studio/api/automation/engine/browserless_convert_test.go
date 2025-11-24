package engine

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/browserless/runtime"
)

func TestBuildStepOutcomeFromRuntimeCarriesExecutionID(t *testing.T) {
	execID := uuid.New()
	start := time.Now().UTC().Add(-time.Second)
	resp := &runtime.ExecutionResponse{
		Steps: []runtime.StepResult{
			{
				Index:            3,
				NodeID:           "node-3",
				Type:             "navigate",
				Success:          true,
				DurationMs:       1200,
				FinalURL:         "https://example.test",
				ExtractedData:    map[string]any{"ok": true},
				ScreenshotBase64: "",
			},
		},
	}

	outcome, err := buildStepOutcomeFromRuntime(execID, contracts.CompiledInstruction{Index: 3, NodeID: "node-3", Type: "navigate"}, start, resp)
	if err != nil {
		t.Fatalf("buildStepOutcomeFromRuntime returned error: %v", err)
	}
	if outcome.ExecutionID != execID {
		t.Fatalf("expected execution_id %s, got %s", execID, outcome.ExecutionID)
	}
	if outcome.StepIndex != 3 || outcome.NodeID != "node-3" || outcome.StepType != "navigate" {
		t.Fatalf("unexpected step identifiers: %+v", outcome)
	}
	if outcome.StartedAt.IsZero() || outcome.CompletedAt == nil || outcome.DurationMs == 0 {
		t.Fatalf("expected timing fields to be populated, got %+v", outcome)
	}
}
