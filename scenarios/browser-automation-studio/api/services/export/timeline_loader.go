package export

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/internal/enums"
	basevidence "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/evidence"
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
	redactSensitiveTimelineEntries(parsed.Entries)

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

// LoadReplayPackage loads the writer-owned renderer-neutral package. A missing
// package is an expected legacy-execution condition; callers decide whether to
// use their compatible timeline fallback.
func (l *TimelineLoader) LoadReplayPackage(ctx context.Context, executionID uuid.UUID) (*basevidence.ReplayPackage, error) {
	execution, err := l.repo.GetExecution(ctx, executionID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(execution.ResultPath) == "" {
		return nil, os.ErrNotExist
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(execution.ResultPath), "evidence.proto.json"))
	if err != nil {
		return nil, err
	}
	var pack basevidence.ReplayPackage
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(raw, &pack); err != nil {
		return nil, fmt.Errorf("parse replay package: %w", err)
	}
	redactSensitiveTimelineEntries(pack.Timeline)
	return &pack, nil
}

func redactSensitiveTimelineEntries(entries []*bastimeline.TimelineEntry) {
	for _, entry := range entries {
		driver.RedactSensitiveTimelineEntry(entry)
	}
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

	return BuildExecutionTimelinePresentation(execution, pbTimeline), nil
}
