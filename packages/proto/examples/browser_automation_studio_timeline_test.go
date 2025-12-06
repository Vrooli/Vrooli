package examples_test

import (
	"testing"
	"time"

	basv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestTimelineSerialization(t *testing.T) {
	started := timestamppb.New(time.Date(2024, 12, 1, 12, 0, 0, 0, time.UTC))

	timeline := &basv1.ExecutionTimeline{
		ExecutionId: "abc-123",
		WorkflowId:  "00000000-0000-0000-0000-000000000000",
		Status:      basv1.ExecutionStatus_EXECUTION_STATUS_COMPLETED,
		Progress:    100,
		StartedAt:   started,
		CompletedAt: started,
		Frames: []*basv1.TimelineFrame{
			{
				StepIndex:  0,
				NodeId:     "navigate-1",
				StepType:   basv1.StepType_STEP_TYPE_NAVIGATE,
				Status:     basv1.StepStatus_STEP_STATUS_COMPLETED,
				Success:    true,
				DurationMs: 1200,
			},
		},
	}

	jsonOpts := protojson.MarshalOptions{
		UseProtoNames: true, // BAS APIs use snake_case JSON fields.
	}
	data, err := jsonOpts.Marshal(timeline)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var parsed basv1.ExecutionTimeline
	if err := protojson.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if parsed.ExecutionId != timeline.ExecutionId {
		t.Errorf("execution_id mismatch")
	}

	if len(parsed.Frames) != 1 || parsed.Frames[0].Status != basv1.StepStatus_STEP_STATUS_COMPLETED {
		t.Fatalf("unexpected frame data after round-trip: %+v", parsed.Frames)
	}
}
