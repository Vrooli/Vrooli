package services

import (
	"context"
	"testing"

	"github.com/google/uuid"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	autorecorder "github.com/vrooli/browser-automation-studio/automation/recorder"
	"github.com/vrooli/browser-automation-studio/database"
)

func TestUnsupportedAutomationNodes(t *testing.T) {
	tests := []struct {
		name string
		flow database.JSONMap
	}{
		{
			name: "simple linear nodes allowed",
			flow: database.JSONMap{
				"nodes": []any{
					map[string]any{"id": "1", "type": "navigate"},
					map[string]any{"id": "2", "type": "click"},
					map[string]any{"id": "3", "type": "assert"},
				},
			},
		},
		{
			name: "condition node allowed",
			flow: database.JSONMap{
				"nodes": []any{
					map[string]any{"id": "1", "type": "condition"},
					map[string]any{"id": "2", "type": "click"},
				},
			},
		},
		{
			name: "repeat loop allowed",
			flow: database.JSONMap{
				"nodes": []any{
					map[string]any{"id": "1", "type": "loop", "data": map[string]any{"loopType": "repeat", "loopCount": 2}},
					map[string]any{"id": "2", "type": "click"},
				},
			},
		},
		{
			name: "missing nodes ignored",
			flow: database.JSONMap{},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()
			unsupported := unsupportedAutomationNodes(tt.flow)
			if len(unsupported) > 0 {
				t.Fatalf("expected no unsupported nodes, got %v", unsupported)
			}
		})
	}
}

type markerRecorder struct {
	marked []autocontracts.StepFailure
}

func (m *markerRecorder) MarkCrash(ctx context.Context, executionID uuid.UUID, failure autocontracts.StepFailure) error {
	m.marked = append(m.marked, failure)
	return nil
}

func (m *markerRecorder) RecordStepOutcome(ctx context.Context, plan autocontracts.ExecutionPlan, outcome autocontracts.StepOutcome) (autorecorder.RecordResult, error) {
	return autorecorder.RecordResult{}, nil
}

func (m *markerRecorder) RecordTelemetry(ctx context.Context, plan autocontracts.ExecutionPlan, telemetry autocontracts.StepTelemetry) error {
	return nil
}

// Ensure recordExecutionMarker uses recorder without panicking when absent.
func TestRecordExecutionMarker(t *testing.T) {
	service := &WorkflowService{}
	// No recorder configured should be a no-op.
	service.recordExecutionMarker(context.Background(), uuid.New(), autocontracts.StepFailure{Kind: autocontracts.FailureKindTimeout})

	rec := &markerRecorder{}
	service.artifactRecorder = rec
	service.recordExecutionMarker(context.Background(), uuid.New(), autocontracts.StepFailure{Kind: autocontracts.FailureKindTimeout})

	if len(rec.marked) != 1 {
		t.Fatalf("expected one crash marker, got %d", len(rec.marked))
	}
	if rec.marked[0].Kind != autocontracts.FailureKindTimeout {
		t.Fatalf("expected timeout failure kind, got %+v", rec.marked[0])
	}
}
