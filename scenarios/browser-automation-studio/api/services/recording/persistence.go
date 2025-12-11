package recording

import (
	"archive/zip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/browser-automation-studio/database"
)

// framePersistResult summarizes the outcome of persisting recording frames.
type framePersistResult struct {
	frameCount      int
	assetCount      int
	totalDurationMs int
	lastNodeID      string
}

// persistFrames processes each frame in the recording manifest, converts it to a step outcome,
// and persists it using the automation recorder. Returns summary statistics.
func (s *RecordingService) persistFrames(
	ctx context.Context,
	zr *zip.Reader,
	project *database.Project,
	workflow *database.Workflow,
	execution *database.Execution,
	manifest *recordingManifest,
) (*framePersistResult, error) {
	// Build a map of archive files for efficient lookup
	files := map[string]*zip.File{}
	for i := range zr.File {
		entry := zr.File[i]
		normalized := normalizeArchiveName(entry.Name)
		if normalized == "" {
			continue
		}
		files[normalized] = entry
	}

	// Determine the effective start time from the manifest or execution
	startTime := execution.StartedAt
	if manifest.RecordedAt != "" {
		if parsed, err := time.Parse(time.RFC3339, manifest.RecordedAt); err == nil {
			startTime = parsed
			execution.StartedAt = parsed
		}
	}

	// Create the adapter to convert recording frames into contract step outcomes
	adapter := newRecordingAdapter(execution.ID, workflow.ID, manifest, files, s.log)
	plan := adapter.executionPlan(startTime)

	totalDuration := 0
	assetCount := 0
	lastNodeID := ""

	// Process each frame sequentially
	for index, frame := range manifest.Frames {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		durationMs := frame.DurationMs
		if durationMs <= 0 {
			durationMs = deriveFrameDuration(manifest.Frames, index)
		}
		totalDuration += durationMs
		lastNodeID = frame.NodeID()

		outcome, hasScreenshot, err := adapter.outcomeForFrame(index, frame, startTime, durationMs)
		if err != nil {
			return nil, err
		}
		if hasScreenshot {
			assetCount++
		}

		if _, err := s.recorder.RecordStepOutcome(ctx, plan, outcome); err != nil {
			return nil, fmt.Errorf("failed to persist recording frame %d: %w", index, err)
		}
	}

	return &framePersistResult{
		frameCount:      len(manifest.Frames),
		assetCount:      assetCount,
		totalDurationMs: totalDuration,
		lastNodeID:      lastNodeID,
	}, nil
}

// cleanupRecordingArtifacts removes partially created database records and filesystem artifacts
// when a recording import fails. It attempts to clean up execution, workflow, and project in order.
func (s *RecordingService) cleanupRecordingArtifacts(baseCtx context.Context, projectCreated, workflowCreated bool, project *database.Project, workflow *database.Workflow, execution *database.Execution) {
	ctx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
	defer cancel()

	// Clean up execution and its associated files
	if execution != nil {
		if err := s.repo.DeleteExecution(ctx, execution.ID); err != nil && s.log != nil {
			s.log.WithError(err).WithField("execution_id", execution.ID).Warn("Failed to delete partial execution during recording cleanup")
		}
		execDir := strings.TrimSpace(s.recordingsRoot)
		if execDir != "" {
			path := filepath.Join(execDir, execution.ID.String())
			if err := os.RemoveAll(path); err != nil && s.log != nil {
				s.log.WithError(err).WithField("path", path).Warn("Failed to remove partial recording assets")
			}
		}
	}

	// Clean up workflow if it was created during this import
	if workflowCreated && workflow != nil {
		if err := s.repo.DeleteWorkflow(ctx, workflow.ID); err != nil && s.log != nil {
			s.log.WithError(err).WithField("workflow_id", workflow.ID).Warn("Failed to delete temporary workflow during recording cleanup")
		}
	}

	// Clean up project if it was created during this import
	if projectCreated && project != nil {
		if err := s.repo.DeleteProject(ctx, project.ID); err != nil && s.log != nil {
			s.log.WithError(err).WithField("project_id", project.ID).Warn("Failed to delete temporary project during recording cleanup")
		}
	}
}

// deriveFrameDuration calculates the duration of a frame by looking at the time delta
// to the next frame. Falls back to the default duration if no next frame exists.
func deriveFrameDuration(frames []recordingFrame, index int) int {
	if index < len(frames)-1 {
		current := frames[index]
		next := frames[index+1]
		delta := next.TimestampMs() - current.TimestampMs()
		if delta > 0 {
			return delta
		}
	}
	return recordingDefaultFrameDurationMs()
}

// intPointer creates a pointer to an int value (helper for optional fields).
func intPointer(v int) *int {
	value := v
	return &value
}
