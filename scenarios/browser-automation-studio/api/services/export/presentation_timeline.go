package export

import (
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/enums"
	bastelemetry "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/domain"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
)

// BuildExecutionTimelinePresentation projects canonical proto entries at the export boundary.
func BuildExecutionTimelinePresentation(execution *database.ExecutionIndex, pb *bastimeline.ExecutionTimeline) *ExecutionTimeline {
	out := &ExecutionTimeline{ExecutionID: execution.ID, WorkflowID: execution.WorkflowID, Status: execution.Status, Progress: int(pb.Progress), StartedAt: execution.StartedAt, CompletedAt: execution.CompletedAt, Frames: make([]TimelineFrame, 0, len(pb.Entries)), Logs: make([]TimelineLog, 0, len(pb.Logs))}
	for _, log := range pb.Logs {
		if log == nil {
			continue
		}
		item := TimelineLog{ID: log.Id, Level: enums.LogLevelToString(log.Level), Message: log.Message, StepName: log.GetStepName()}
		if log.Timestamp != nil {
			item.Timestamp = log.Timestamp.AsTime()
		}
		out.Logs = append(out.Logs, item)
	}
	for _, entry := range pb.Entries {
		if entry != nil {
			out.Frames = append(out.Frames, presentationFrameForTimelineEntry(entry))
		}
	}
	return out
}

func presentationFrameForTimelineEntry(entry *bastimeline.TimelineEntry) TimelineFrame {
	frame := TimelineFrame{Status: "completed", Success: true}
	if entry.StepIndex != nil {
		frame.StepIndex = int(*entry.StepIndex)
	}
	if entry.NodeId != nil {
		frame.NodeID = *entry.NodeId
	}
	if entry.DurationMs != nil {
		frame.DurationMs = int(*entry.DurationMs)
	}
	if entry.TotalDurationMs != nil {
		frame.TotalDurationMs = int(*entry.TotalDurationMs)
	}
	if entry.Timestamp != nil {
		started := entry.Timestamp.AsTime()
		frame.StartedAt = &started
		if entry.GetTotalDurationMs() > 0 {
			completed := started.Add(time.Duration(entry.GetTotalDurationMs()) * time.Millisecond)
			frame.CompletedAt = &completed
		}
	}
	if entry.Action != nil {
		frame.StepType = entry.Action.Type.String()
	}
	if ctx := entry.GetContext(); ctx != nil {
		if ctx.Success != nil {
			frame.Success = *ctx.Success
		}
		if ctx.Error != nil {
			frame.Error = *ctx.Error
		}
		frame.Condition = contracts.ProtoToConditionOutcome(ctx.Condition)
		if retry := ctx.RetryStatus; retry != nil {
			frame.RetryAttempt = int(retry.CurrentAttempt)
			frame.RetryMaxAttempts = int(retry.MaxAttempts)
			if retry.Configured {
				frame.RetryConfigured = max(0, int(retry.MaxAttempts)-1)
			}
			frame.RetryDelayMs = int(retry.DelayMs)
			frame.RetryBackoffFactor = retry.BackoffFactor
			for _, attempt := range retry.History {
				if attempt != nil {
					frame.RetryHistory = append(frame.RetryHistory, RetryHistoryEntry{Attempt: int(attempt.Attempt), Success: attempt.Success, DurationMs: int(attempt.DurationMs), Error: attempt.GetError()})
				}
			}
		}
		if assertion := ctx.Assertion; assertion != nil {
			frame.Assertion = &contracts.AssertionOutcome{Mode: enums.AssertionModeToString(assertion.Mode), Selector: assertion.Selector, Expected: contracts.JsonValueToAny(assertion.Expected), Actual: contracts.JsonValueToAny(assertion.Actual), Success: assertion.Success, Negated: assertion.Negated, CaseSensitive: assertion.CaseSensitive, Message: assertion.GetMessage()}
		}
	}
	if entry.Aggregates != nil {
		agg := entry.Aggregates
		frame.Status = enums.StepStatusToString(agg.Status)
		if agg.FinalUrl != nil {
			frame.FinalURL = *agg.FinalUrl
		}
		frame.Progress = int(agg.GetProgress())
		frame.ConsoleLogCount = int(agg.ConsoleLogCount)
		frame.NetworkEventCount = int(agg.NetworkEventCount)
		if agg.ExtractedDataPreview != nil {
			frame.ExtractedDataPreview = contracts.JsonValueToAny(agg.ExtractedDataPreview)
		}
		frame.FocusedElement = agg.FocusedElement
		for _, artifact := range agg.Artifacts {
			if artifact != nil {
				var artifactStepIndex *int
				if artifact.StepIndex != nil {
					index := int(*artifact.StepIndex)
					artifactStepIndex = &index
				}
				payload := make(map[string]any, len(artifact.Payload))
				for key, value := range artifact.Payload {
					payload[key] = contracts.JsonValueToAny(value)
				}
				frame.Artifacts = append(frame.Artifacts, TimelineArtifact{ID: artifact.Id, Type: artifact.Type.String(), Label: artifact.GetLabel(), StorageURL: artifact.StorageUrl, ThumbnailURL: artifact.GetThumbnailUrl(), ContentType: artifact.ContentType, SizeBytes: artifact.SizeBytes, StepIndex: artifactStepIndex, Payload: payload})
			}
		}
	}
	if frame.Status == "" {
		if frame.Success {
			frame.Status = "completed"
		} else {
			frame.Status = "failed"
		}
	}
	if entry.Telemetry != nil && entry.Telemetry.Screenshot != nil {
		frame.Screenshot = timelineScreenshotFromProto(entry.Telemetry.Screenshot)
	}
	if telemetry := entry.Telemetry; telemetry != nil {
		if telemetry.DomSnapshot != nil {
			frame.Artifacts = append(frame.Artifacts, telemetryArtifactToExport("dom_snapshot", telemetry.DomSnapshot, nil))
		}
		if telemetry.ConsoleLogArtifact != nil {
			frame.Artifacts = append(frame.Artifacts, telemetryArtifactToExport("console_log", telemetry.ConsoleLogArtifact, nil))
		}
		if telemetry.NetworkEventArtifact != nil {
			frame.Artifacts = append(frame.Artifacts, telemetryArtifactToExport("network_event", telemetry.NetworkEventArtifact, nil))
		}
		frame.ElementBoundingBox = telemetry.ElementBoundingBox
		frame.ClickPosition = telemetry.ClickPosition
		frame.CursorTrail = telemetry.CursorTrail
		frame.HighlightRegions = telemetry.HighlightRegions
		frame.MaskRegions = telemetry.MaskRegions
		if telemetry.ZoomFactor != nil {
			frame.ZoomFactor = *telemetry.ZoomFactor
		}
	}
	if entry.Context != nil && entry.Context.Condition != nil && frame.Condition == nil {
		frame.Condition = contracts.ProtoToConditionOutcome(entry.Context.Condition)
	}
	return frame
}

func timelineScreenshotFromProto(shot *bastelemetry.TimelineScreenshot) *TimelineScreenshot {
	if shot == nil {
		return nil
	}
	return &TimelineScreenshot{ArtifactID: shot.ArtifactId, URL: shot.Url, ThumbnailURL: shot.ThumbnailUrl, Width: int(shot.Width), Height: int(shot.Height), ContentType: shot.ContentType, SizeBytes: shot.SizeBytes}
}

func loadTelemetryJSONSlice(path *string) ([]any, error) {
	if path == nil || strings.TrimSpace(*path) == "" {
		return nil, nil
	}
	raw, err := os.ReadFile(*path)
	if err != nil {
		return nil, err
	}
	var entries []any
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func telemetryArtifactToExport(kind string, artifact *bastelemetry.TelemetryArtifact, payload map[string]any) TimelineArtifact {
	out := TimelineArtifact{Type: kind, Label: kind, Payload: payload}
	if artifact == nil {
		return out
	}
	out.ID = artifact.ArtifactId
	out.StorageURL = artifact.StorageUrl
	out.ContentType = artifact.ContentType
	out.SizeBytes = artifact.SizeBytes
	if artifact.Path != nil {
		if out.Payload == nil {
			out.Payload = map[string]any{}
		}
		out.Payload["path"] = *artifact.Path
		if entries, err := loadTelemetryJSONSlice(artifact.Path); err == nil && entries != nil {
			out.Payload["entries"] = entries
		}
	}
	return out
}
