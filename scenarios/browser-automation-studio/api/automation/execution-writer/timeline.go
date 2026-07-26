package executionwriter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
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

func (r *FileWriter) protoTimelineFilePath(executionID uuid.UUID) string {
	return filepath.Join(r.dataDir, executionID.String(), protoTimelineFileName)
}

func (r *FileWriter) writeProtoTimelineFile(executionID uuid.UUID, timeline *executionTimelineData) error {
	if timeline == nil || timeline.pb == nil {
		return nil
	}
	timeline.mu.Lock()
	defer timeline.mu.Unlock()
	path := r.protoTimelineFilePath(executionID)
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
	if err := os.WriteFile(path, indented, 0o644); err != nil {
		return fmt.Errorf("write proto timeline file: %w", err)
	}
	return nil
}
