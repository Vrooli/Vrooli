package executionwriter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	corestorage "github.com/vrooli/api-core/storage"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	"google.golang.org/protobuf/encoding/protojson"
)

// getOrCreateTimeline and writeProtoTimelineFile own the durable timeline
// projection. Outcome and telemetry modules only supply entries/logs.
func (r *FileWriter) getOrCreateTimeline(plan contracts.ExecutionPlan) *executionTimelineData {
	key := plan.ExecutionID.String()
	if value, ok := r.timelines.Load(key); ok {
		return value.(*executionTimelineData)
	}
	timeline := &executionTimelineData{pb: &bastimeline.ExecutionTimeline{ExecutionId: plan.ExecutionID.String(), WorkflowId: plan.WorkflowID.String(), Status: basbase.ExecutionStatus_EXECUTION_STATUS_PENDING}}
	r.timelines.Store(key, timeline)
	return timeline
}

func (r *FileWriter) protoTimelineFilePath(ctx context.Context, executionID uuid.UUID) (string, error) {
	if r == nil || r.root == nil {
		return "", fmt.Errorf("execution artifact root is unavailable")
	}
	root, err := r.root.Root(ctx)
	if err != nil {
		return "", fmt.Errorf("resolve execution artifact root: %w", err)
	}
	return filepath.Join(root, executionID.String(), protoTimelineFileName), nil
}

func (r *FileWriter) writeProtoTimelineFile(ctx context.Context, executionID uuid.UUID, timeline *executionTimelineData) error {
	if timeline == nil || timeline.pb == nil {
		return nil
	}
	timeline.mu.Lock()
	defer timeline.mu.Unlock()
	path, err := r.protoTimelineFilePath(ctx, executionID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create proto timeline directory: %w", err)
	}
	raw, err := protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: false}.Marshal(timeline.pb)
	if err != nil {
		return fmt.Errorf("marshal proto timeline: %w", err)
	}
	indented := raw
	var buf bytes.Buffer
	if json.Indent(&buf, raw, "", "  ") == nil {
		indented = buf.Bytes()
	}
	if err := corestorage.WriteFileAtomic(path, indented, 0o644); err != nil {
		return fmt.Errorf("write proto timeline file: %w", err)
	}
	r.root.RecordWrite(ctx)
	return nil
}
