package export

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	exportsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/exports"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestBuildReplaySpecUsesGeneratedContractAndPreservesBaseline(t *testing.T) {
	id := uuid.New()
	started := timestamppb.New(time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC))
	baseline := &exportsv1.ReplaySpec{
		Version: "v1",
		Execution: &exportsv1.ReplayExecutionMetadata{
			ExecutionId: id.String(), WorkflowId: uuid.NewString(), WorkflowName: "Workflow",
			Status: "completed", StartedAt: started,
		},
		Frames: []*exportsv1.ReplayFrame{{Index: 4, DurationMs: 1200, ScreenshotAssetId: "screen-1"}},
		Assets: []*exportsv1.ReplayAsset{{Id: "screen-1", Source: "asset://screen-1"}},
	}
	incoming := &exportsv1.ReplaySpec{Execution: &exportsv1.ReplayExecutionMetadata{ExecutionId: id.String()}}
	result, err := BuildReplaySpec(baseline, incoming, id)
	if err != nil {
		t.Fatal(err)
	}
	if result == incoming || result.Frames[0] == baseline.Frames[0] {
		t.Fatal("builder must return cloned protobuf messages")
	}
	if result.Execution.GetWorkflowName() != "Workflow" || result.Execution.GetStartedAt().AsTime() != started.AsTime() {
		t.Fatalf("baseline execution metadata was not retained: %v", result.Execution)
	}
	if result.Summary.GetTotalDurationMs() != 1200 || result.Summary.GetScreenshotCount() != 1 {
		t.Fatalf("summary defaults were not derived from frames: %v", result.Summary)
	}
	if result.Cursor.GetScale() != 0.5 {
		t.Fatalf("cursor scale clamp was not applied: %v", result.Cursor)
	}
	if !proto.Equal(incoming, &exportsv1.ReplaySpec{Execution: &exportsv1.ReplayExecutionMetadata{ExecutionId: id.String()}}) {
		t.Fatal("builder mutated the caller's generated message")
	}
	if baseline.Frames[0].Index != 4 {
		t.Fatal("builder mutated the baseline frame")
	}
}

func TestBuildReplaySpecAppliesStandaloneDefaults(t *testing.T) {
	id := uuid.New()
	result, err := BuildReplaySpec(nil, &exportsv1.ReplaySpec{
		Execution: &exportsv1.ReplayExecutionMetadata{ExecutionId: id.String(), WorkflowName: "Workflow"},
		Frames:    []*exportsv1.ReplayFrame{{DurationMs: 800}},
	}, id)
	if err != nil {
		t.Fatal(err)
	}
	if result.Theme.GetBrowserChrome().GetTitle() != "Workflow" || result.Cursor.GetInitialPosition() != "center" {
		t.Fatalf("standalone theme/cursor defaults missing: theme=%v cursor=%v", result.Theme, result.Cursor)
	}
}

func TestBuildReplaySpecValidatesIdentityAndFrames(t *testing.T) {
	id := uuid.New()
	t.Run("execution mismatch", func(t *testing.T) {
		_, err := BuildReplaySpec(nil, &exportsv1.ReplaySpec{Execution: &exportsv1.ReplayExecutionMetadata{ExecutionId: uuid.NewString()}, Frames: []*exportsv1.ReplayFrame{{}}}, id)
		if err == nil || !strings.Contains(err.Error(), "execution_id mismatch") {
			t.Fatalf("expected execution mismatch, got %v", err)
		}
	})
	t.Run("workflow mismatch", func(t *testing.T) {
		base := &exportsv1.ReplaySpec{Execution: &exportsv1.ReplayExecutionMetadata{WorkflowId: uuid.NewString()}, Frames: []*exportsv1.ReplayFrame{{}}}
		incoming := &exportsv1.ReplaySpec{Execution: &exportsv1.ReplayExecutionMetadata{ExecutionId: id.String(), WorkflowId: uuid.NewString()}, Frames: []*exportsv1.ReplayFrame{{}}}
		_, err := BuildReplaySpec(base, incoming, id)
		if err == nil || !strings.Contains(err.Error(), "workflow_id mismatch") {
			t.Fatalf("expected workflow mismatch, got %v", err)
		}
	})
	t.Run("missing frames", func(t *testing.T) {
		_, err := BuildReplaySpec(nil, &exportsv1.ReplaySpec{Execution: &exportsv1.ReplayExecutionMetadata{ExecutionId: id.String()}}, id)
		if err == nil || !strings.Contains(err.Error(), "missing frames") {
			t.Fatalf("expected missing frames error, got %v", err)
		}
	})
	if _, err := BuildReplaySpec(nil, nil, id); err != ErrMovieSpecUnavailable {
		t.Fatalf("expected missing-spec error, got %v", err)
	}
}
