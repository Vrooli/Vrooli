package export

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/enums"
	"github.com/vrooli/browser-automation-studio/internal/typeconv"
	bastelemetry "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/domain"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	"google.golang.org/protobuf/encoding/protojson"
)

// TimelineLoader loads execution timeline data from various sources.
type TimelineLoader struct {
	repo ExecutionRepository
}

// NewTimelineLoader creates a new TimelineLoader.
func NewTimelineLoader(repo ExecutionRepository) *TimelineLoader {
	return &TimelineLoader{repo: repo}
}

// LoadTimelineProto reads the on-disk proto timeline (preferred) for a given execution.
// Falls back to a minimal proto timeline if no file exists yet.
func (l *TimelineLoader) LoadTimelineProto(ctx context.Context, executionID uuid.UUID) (*bastimeline.ExecutionTimeline, error) {
	execution, err := l.repo.GetExecution(ctx, executionID)
	if err != nil {
		return nil, err
	}

	pb := &bastimeline.ExecutionTimeline{
		ExecutionId: execution.ID.String(),
		WorkflowId:  execution.WorkflowID.String(),
		Status:      enums.StringToExecutionStatus(execution.Status),
		Progress:    0,
		StartedAt:   autocontracts.TimeToTimestamp(execution.StartedAt),
	}
	if execution.CompletedAt != nil {
		pb.CompletedAt = autocontracts.TimePtrToTimestamp(execution.CompletedAt)
	}

	if strings.TrimSpace(execution.ResultPath) == "" {
		return pb, nil
	}

	timelinePath := filepath.Join(filepath.Dir(execution.ResultPath), "timeline.proto.json")
	raw, err := os.ReadFile(timelinePath)
	if err != nil {
		if os.IsNotExist(err) {
			return pb, nil
		}
		return nil, fmt.Errorf("read proto timeline: %w", err)
	}
	var parsed bastimeline.ExecutionTimeline
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("parse proto timeline: %w", err)
	}

	// Ensure key fields reflect current index data.
	if strings.TrimSpace(parsed.ExecutionId) == "" {
		parsed.ExecutionId = pb.ExecutionId
	}
	if strings.TrimSpace(parsed.WorkflowId) == "" {
		parsed.WorkflowId = pb.WorkflowId
	}
	parsed.Status = pb.Status
	parsed.StartedAt = pb.StartedAt
	parsed.CompletedAt = pb.CompletedAt
	return &parsed, nil
}

// LoadTimeline assembles replay-ready timeline data for a given execution.
// Reads execution data from the result JSON file stored on disk.
func (l *TimelineLoader) LoadTimeline(ctx context.Context, executionID uuid.UUID) (*ExecutionTimeline, error) {
	execution, err := l.repo.GetExecution(ctx, executionID)
	if err != nil {
		return nil, err
	}

	// If no result file exists yet, return a minimal timeline
	if execution.ResultPath == "" {
		return &ExecutionTimeline{
			ExecutionID: execution.ID,
			WorkflowID:  execution.WorkflowID,
			Status:      execution.Status,
			StartedAt:   execution.StartedAt,
			CompletedAt: execution.CompletedAt,
			Frames:      []TimelineFrame{},
			Logs:        []TimelineLog{},
		}, nil
	}

	pbTimeline, err := l.LoadTimelineProto(ctx, executionID)
	if err != nil {
		if os.IsNotExist(err) {
			return &ExecutionTimeline{
				ExecutionID: execution.ID,
				WorkflowID:  execution.WorkflowID,
				Status:      execution.Status,
				StartedAt:   execution.StartedAt,
				CompletedAt: execution.CompletedAt,
				Frames:      []TimelineFrame{},
				Logs:        []TimelineLog{},
			}, nil
		}
		return nil, fmt.Errorf("read timeline proto: %w", err)
	}

	return timelineProtoToExport(execution, pbTimeline), nil
}

// timelineProtoToExport converts a proto timeline to the export timeline format.
func timelineProtoToExport(execution *database.ExecutionIndex, pb *bastimeline.ExecutionTimeline) *ExecutionTimeline {
	frames := make([]TimelineFrame, 0, len(pb.Entries))
	for _, entry := range pb.Entries {
		if entry == nil {
			continue
		}
		frame := timelineEntryToFrame(entry)
		frames = append(frames, frame)
	}
	sort.Slice(frames, func(i, j int) bool {
		if frames[i].StepIndex != frames[j].StepIndex {
			return frames[i].StepIndex < frames[j].StepIndex
		}
		return frames[i].NodeID < frames[j].NodeID
	})

	logs := make([]TimelineLog, 0, len(pb.Logs))
	for _, log := range pb.Logs {
		if log == nil {
			continue
		}
		entry := TimelineLog{
			ID:      log.Id,
			Level:   strings.ToLower(log.Level.String()),
			Message: log.Message,
		}
		if log.Timestamp != nil {
			entry.Timestamp = autocontracts.TimestampToTime(log.Timestamp)
		}
		if log.StepName != nil {
			entry.StepName = *log.StepName
		}
		logs = append(logs, entry)
	}

	startedAt := execution.StartedAt
	if pb.StartedAt != nil {
		startedAt = autocontracts.TimestampToTime(pb.StartedAt)
	}
	completedAt := execution.CompletedAt
	if pb.CompletedAt != nil {
		completedAt = autocontracts.TimestampToTimePtr(pb.CompletedAt)
	}

	return &ExecutionTimeline{
		ExecutionID: execution.ID,
		WorkflowID:  execution.WorkflowID,
		Status:      execution.Status,
		Progress:    int(pb.Progress),
		StartedAt:   startedAt,
		CompletedAt: completedAt,
		Frames:      frames,
		Logs:        logs,
	}
}

// timelineEntryToFrame converts a proto timeline entry to a TimelineFrame.
func timelineEntryToFrame(entry *bastimeline.TimelineEntry) TimelineFrame {
	frame := TimelineFrame{}

	if entry.StepIndex != nil {
		frame.StepIndex = int(*entry.StepIndex)
	} else {
		frame.StepIndex = int(entry.SequenceNum)
	}
	if entry.NodeId != nil {
		frame.NodeID = *entry.NodeId
	}
	if entry.Action != nil {
		frame.StepType = enums.ActionTypeToString(entry.Action.Type)
	}
	if entry.DurationMs != nil {
		frame.DurationMs = int(*entry.DurationMs)
	}
	if entry.TotalDurationMs != nil {
		frame.TotalDurationMs = int(*entry.TotalDurationMs)
	}
	if entry.Timestamp != nil {
		started := autocontracts.TimestampToTime(entry.Timestamp)
		frame.StartedAt = &started
		if entry.DurationMs != nil {
			completed := started.Add(time.Duration(*entry.DurationMs) * time.Millisecond)
			frame.CompletedAt = &completed
		}
	}
	if entry.Context != nil {
		if entry.Context.Success != nil {
			frame.Success = *entry.Context.Success
		}
		if entry.Context.Error != nil {
			frame.Error = *entry.Context.Error
		}
		if entry.Context.Assertion != nil {
			frame.Assertion = &autocontracts.AssertionOutcome{
				Mode:          entry.Context.Assertion.Mode.String(),
				Selector:      entry.Context.Assertion.Selector,
				Success:       entry.Context.Assertion.Success,
				Negated:       entry.Context.Assertion.Negated,
				CaseSensitive: entry.Context.Assertion.CaseSensitive,
			}
			if entry.Context.Assertion.Message != nil {
				frame.Assertion.Message = *entry.Context.Assertion.Message
			}
			if entry.Context.Assertion.Expected != nil {
				frame.Assertion.Expected = typeconv.JsonValueToAny(entry.Context.Assertion.Expected)
			}
			if entry.Context.Assertion.Actual != nil {
				frame.Assertion.Actual = typeconv.JsonValueToAny(entry.Context.Assertion.Actual)
			}
		}
	}
	if entry.Telemetry != nil {
		frame.FinalURL = entry.Telemetry.Url
		frame.ElementBoundingBox = entry.Telemetry.ElementBoundingBox
		frame.ClickPosition = entry.Telemetry.ClickPosition
		frame.CursorTrail = entry.Telemetry.CursorTrail
		frame.HighlightRegions = entry.Telemetry.HighlightRegions
		frame.MaskRegions = entry.Telemetry.MaskRegions
		if entry.Telemetry.ZoomFactor != nil {
			frame.ZoomFactor = *entry.Telemetry.ZoomFactor
		}
		if entry.Telemetry.Screenshot != nil {
			frame.Screenshot = timelineScreenshotFromProto(entry.Telemetry.Screenshot)
		}
		if entry.Telemetry.ConsoleLogArtifact != nil {
			if entries, err := loadTelemetryJSONSlice(entry.Telemetry.ConsoleLogArtifact.Path); err == nil {
				frame.ConsoleLogCount = len(entries)
				frame.Artifacts = append(frame.Artifacts, telemetryArtifactToExport("console", entry.Telemetry.ConsoleLogArtifact, map[string]any{
					"entries": entries,
				}))
			}
		}
		if entry.Telemetry.NetworkEventArtifact != nil {
			if events, err := loadTelemetryJSONSlice(entry.Telemetry.NetworkEventArtifact.Path); err == nil {
				frame.NetworkEventCount = len(events)
				frame.Artifacts = append(frame.Artifacts, telemetryArtifactToExport("network", entry.Telemetry.NetworkEventArtifact, map[string]any{
					"events": events,
				}))
			}
		}
		if entry.Telemetry.DomSnapshot != nil {
			frame.Artifacts = append(frame.Artifacts, telemetryArtifactToExport("dom_snapshot", entry.Telemetry.DomSnapshot, map[string]any{}))
		}
	}
	if entry.Aggregates != nil {
		frame.Status = enums.StepStatusToString(entry.Aggregates.Status)
		if entry.Aggregates.FinalUrl != nil {
			frame.FinalURL = *entry.Aggregates.FinalUrl
		}
		if entry.Aggregates.Progress != nil {
			frame.Progress = int(*entry.Aggregates.Progress)
		}
		if entry.Aggregates.ConsoleLogCount != 0 {
			frame.ConsoleLogCount = int(entry.Aggregates.ConsoleLogCount)
		}
		if entry.Aggregates.NetworkEventCount != 0 {
			frame.NetworkEventCount = int(entry.Aggregates.NetworkEventCount)
		}
		if entry.Aggregates.ExtractedDataPreview != nil {
			frame.ExtractedDataPreview = typeconv.JsonValueToAny(entry.Aggregates.ExtractedDataPreview)
		}
		if entry.Aggregates.FocusedElement != nil {
			frame.FocusedElement = entry.Aggregates.FocusedElement
		}
	}
	if frame.Status == "" {
		if frame.Success {
			frame.Status = "completed"
		} else {
			frame.Status = "failed"
		}
	}
	return frame
}

// timelineScreenshotFromProto converts a proto screenshot to the export format.
func timelineScreenshotFromProto(shot *bastelemetry.TimelineScreenshot) *TimelineScreenshot {
	if shot == nil {
		return nil
	}
	return &TimelineScreenshot{
		ArtifactID:   shot.ArtifactId,
		URL:          shot.Url,
		ThumbnailURL: shot.ThumbnailUrl,
		Width:        int(shot.Width),
		Height:       int(shot.Height),
		ContentType:  shot.ContentType,
		SizeBytes:    shot.SizeBytes,
	}
}

// loadTelemetryJSONSlice loads a JSON array from a file path.
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

// telemetryArtifactToExport converts a proto telemetry artifact to the export format.
func telemetryArtifactToExport(kind string, artifact *bastelemetry.TelemetryArtifact, payload map[string]any) TimelineArtifact {
	out := TimelineArtifact{
		Type:    kind,
		Label:   kind,
		Payload: payload,
	}
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
	}
	return out
}
