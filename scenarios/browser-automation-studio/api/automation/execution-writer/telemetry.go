package executionwriter

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/browser-automation-studio/automation/contracts"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// RecordTelemetry owns the telemetry artifact policy and timeline-log
// projection. Step outcomes remain independent of this optional data stream.
func (r *FileWriter) RecordTelemetry(ctx context.Context, plan contracts.ExecutionPlan, telemetry contracts.StepTelemetry) error {
	if r == nil || !r.artifactConfigForExecution(plan.ExecutionID).CollectTelemetry {
		return nil
	}
	result, timeline := r.getOrCreateResult(plan), r.getOrCreateTimeline(plan)
	data := TelemetryData{StepIndex: telemetry.StepIndex, Data: telemetry, Timestamp: time.Now().UTC()}
	result.mu.Lock()
	result.Telemetry = append(result.Telemetry, data)
	result.mu.Unlock()
	if timeline != nil {
		if entry := telemetryToTimelineLog(data); entry != nil {
			timeline.mu.Lock()
			timeline.pb.Logs = append(timeline.pb.Logs, entry)
			timeline.mu.Unlock()
			_ = r.writeProtoTimelineFile(plan.ExecutionID, timeline)
		}
	}
	return r.writeResultFile(plan.ExecutionID, result, timeline)
}

func telemetryToTimelineLog(t TelemetryData) *bastimeline.TimelineLog {
	message, level := strings.TrimSpace(t.Data.Note), basbase.LogLevel_LOG_LEVEL_INFO
	switch strings.ToLower(strings.TrimSpace(string(t.Data.Kind))) {
	case "console":
		if len(t.Data.Console) > 0 && message == "" {
			first := t.Data.Console[0]
			message = strings.TrimSpace(first.Text)
			switch strings.ToLower(strings.TrimSpace(first.Type)) {
			case "warn", "warning":
				level = basbase.LogLevel_LOG_LEVEL_WARN
			case "error":
				level = basbase.LogLevel_LOG_LEVEL_ERROR
			case "debug":
				level = basbase.LogLevel_LOG_LEVEL_DEBUG
			}
		}
	case "network":
		level = basbase.LogLevel_LOG_LEVEL_DEBUG
	case "retry":
		level = basbase.LogLevel_LOG_LEVEL_WARN
	}
	if message == "" {
		return nil
	}
	return &bastimeline.TimelineLog{Id: fmt.Sprintf("telemetry-%d-%d", t.StepIndex, t.Timestamp.UnixNano()), Level: level, Message: message, Timestamp: timestamppb.New(t.Timestamp)}
}
