// Package workflow provides workflow execution services.
// Timeline loading has been moved to services/export/timeline_loader.go.
// This file provides backward-compatible delegation methods.
package workflow

import (
	"context"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/internal/typeconv"
	"github.com/vrooli/browser-automation-studio/services/export"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
)

// Type aliases for backward compatibility with existing workflow package code.
type (
	ExecutionTimeline  = export.ExecutionTimeline
	TimelineFrame      = export.TimelineFrame
	TimelineLog        = export.TimelineLog
	RetryHistoryEntry  = typeconv.RetryHistoryEntry
	TimelineScreenshot = typeconv.TimelineScreenshot
	TimelineArtifact   = typeconv.TimelineArtifact
)

// timelineLoader is lazily initialized to avoid circular dependencies.
func (s *WorkflowService) getTimelineLoader() *export.TimelineLoader {
	// Create a timeline loader using the repo
	return export.NewTimelineLoader(s.repo)
}

// GetExecutionTimelineProto reads the on-disk proto timeline (preferred) for a given execution.
// Falls back to a minimal proto timeline if no file exists yet.
// Delegates to services/export.TimelineLoader.
func (s *WorkflowService) GetExecutionTimelineProto(ctx context.Context, executionID uuid.UUID) (*bastimeline.ExecutionTimeline, error) {
	return s.getTimelineLoader().LoadTimelineProto(ctx, executionID)
}

// GetExecutionTimeline assembles replay-ready artifacts for a given execution.
// Reads execution data from the result JSON file stored on disk.
// Delegates to services/export.TimelineLoader.
func (s *WorkflowService) GetExecutionTimeline(ctx context.Context, executionID uuid.UUID) (*ExecutionTimeline, error) {
	return s.getTimelineLoader().LoadTimeline(ctx, executionID)
}
