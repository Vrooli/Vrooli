package examples_test

import (
	"testing"
	"time"

	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestTimelineSerialization(t *testing.T) {
	started := timestamppb.New(time.Date(2024, 12, 1, 12, 0, 0, 0, time.UTC))
	stepIndex := int32(0)
	nodeId := "navigate-1"
	success := true

	timeline := &bastimeline.ExecutionTimeline{
		ExecutionId: "abc-123",
		WorkflowId:  "00000000-0000-0000-0000-000000000000",
		Status:      basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED,
		Progress:    100,
		StartedAt:   started,
		CompletedAt: started,
		Entries: []*bastimeline.TimelineEntry{
			{
				Id:        "entry-1",
				StepIndex: &stepIndex,
				NodeId:    &nodeId,
				Timestamp: started,
				Action: &basactions.ActionDefinition{
					Type: basactions.ActionType_ACTION_TYPE_NAVIGATE,
					Params: &basactions.ActionDefinition_Navigate{
						Navigate: &basactions.NavigateParams{
							Url: "https://example.com",
						},
					},
				},
				Context: &basbase.EventContext{
					Success: &success,
				},
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

	var parsed bastimeline.ExecutionTimeline
	if err := protojson.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if parsed.ExecutionId != timeline.ExecutionId {
		t.Errorf("execution_id mismatch")
	}

	if len(parsed.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(parsed.Entries))
	}

	entry := parsed.Entries[0]
	if entry.Context == nil || entry.Context.Success == nil || !*entry.Context.Success {
		t.Fatalf("unexpected entry data after round-trip: %+v", entry)
	}
}
