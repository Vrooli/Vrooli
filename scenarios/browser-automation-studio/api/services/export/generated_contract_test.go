package export

import (
	"testing"
	"time"

	exportsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/exports"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestReplaySpecProtoProjectionPreservesStableReplayFields(t *testing.T) {
	generatedAt := time.Date(2026, 9, 15, 12, 30, 0, 0, time.UTC)
	spec := &exportsv1.ReplaySpec{
		Version: "2025-11-07", GeneratedAt: timestamppb.New(generatedAt),
		Execution: &exportsv1.ReplayExecutionMetadata{ExecutionId: "execution-1", WorkflowId: "workflow-1", StartedAt: timestamppb.New(generatedAt.Add(-time.Minute))},
		Frames:    []*exportsv1.ReplayFrame{{Index: 2, NodeId: "step-3", DurationMs: 840, Enter: &exportsv1.ReplayTransition{Type: "fade", DurationMs: 120, Easing: "ease-out"}, Viewport: &exportsv1.ReplayDimensions{Width: 1440, Height: 900}}},
		Assets:    []*exportsv1.ReplayAsset{{Id: "screen-1", Type: "screenshot", Source: "asset://screen-1"}},
	}
	if spec.GetExecution().GetExecutionId() != "execution-1" || spec.GetExecution().GetWorkflowId() != "workflow-1" {
		t.Fatalf("generated execution identity = (%q, %q)", spec.GetExecution().GetExecutionId(), spec.GetExecution().GetWorkflowId())
	}
	if !spec.GetGeneratedAt().AsTime().Equal(generatedAt) {
		t.Fatalf("generated timestamp = %s", spec.GetGeneratedAt().AsTime())
	}
	if len(spec.GetFrames()) != 1 || spec.GetFrames()[0].GetIndex() != 2 || spec.GetFrames()[0].GetEnter().GetType() != "fade" {
		t.Fatalf("generated frame projection lost fields: %v", spec.GetFrames())
	}
}
