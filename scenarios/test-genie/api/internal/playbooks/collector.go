package playbooks

import (
	"context"
	"time"

	"test-genie/internal/playbooks/artifacts"
	"test-genie/internal/playbooks/execution"
	"test-genie/internal/shared"
)

// collectWorkflowArtifacts fetches and writes all artifacts for a workflow execution.
// This includes timeline data, parsed results, screenshots, and generates a README.
// It handles both success and failure cases to aid debugging.
func (r *Runner) collectWorkflowArtifacts(
	ctx context.Context,
	entry Entry,
	executionID string,
	outcome *Outcome,
	execErr error,
) (*artifacts.WorkflowArtifacts, error) {
	// Fetch timeline data from BAS
	timeline, timelineData, fetchErr := r.basClient.GetTimeline(ctx, executionID)
	if fetchErr != nil {
		shared.LogWarn(r.logWriter, "failed to fetch timeline for %s: %v", entry.File, fetchErr)
		return &artifacts.WorkflowArtifacts{}, nil
	}

	// Parse timeline for structured data
	parsed, parseErr := execution.ParseFullTimeline(timelineData)
	if parseErr != nil {
		shared.LogWarn(r.logWriter, "failed to parse timeline for %s: %v", entry.File, parseErr)
		// Continue with nil parsed - will still write raw timeline
	} else if parsed != nil && timeline != nil {
		// Reuse the already-fetched proto timeline to avoid duplicate parsing work downstream.
		parsed.Proto = timeline
	}

	// Update outcome stats from parsed timeline
	if parsed != nil {
		outcome.Stats = parsed.Summary.String()
	}

	// Download screenshots
	var screenshots []artifacts.ScreenshotData
	if parsed != nil {
		screenshots = r.downloadScreenshots(ctx, entry.File, parsed)
	}

	// Build result for README generation
	status := "passed"
	errMsg := ""
	if execErr != nil {
		status = "failed"
		errMsg = execErr.Error()
	}

	result := &artifacts.WorkflowResult{
		WorkflowFile:  entry.File,
		Description:   entry.Description,
		Requirements:  entry.Requirements,
		ExecutionID:   executionID,
		Success:       execErr == nil,
		Status:        status,
		Error:         errMsg,
		DurationMs:    outcome.Duration.Milliseconds(),
		Timestamp:     time.Now().UTC(),
		Summary:       getSummaryFromParsed(parsed),
		ParsedSummary: parsed,
	}

	// Get the FileWriter from the interface (if available)
	fileWriter, ok := r.artifactWriter.(*artifacts.FileWriter)
	if !ok {
		// Fallback to legacy timeline-only writing
		shared.LogWarn(r.logWriter, "artifact writer does not support full artifact collection, using legacy mode")
		timelinePath, _ := r.artifactWriter.WriteTimeline(entry.File, timelineData)
		return &artifacts.WorkflowArtifacts{Dir: timelinePath, Timeline: timelinePath}, parseErr
	}

	// Write all artifacts
	workflowArtifacts, writeErr := fileWriter.WriteWorkflowArtifacts(
		entry.File,
		timelineData,
		parsed,
		screenshots,
		result,
	)
	if writeErr != nil {
		shared.LogWarn(r.logWriter, "failed to write artifacts for %s: %v", entry.File, writeErr)
	}

	if workflowArtifacts != nil {
		shared.LogStep(r.logWriter, "artifacts written to %s", workflowArtifacts.Dir)
		// Attach proto timeline for error diagnostics
		workflowArtifacts.Proto = timeline
	}

	return workflowArtifacts, parseErr
}

// getSummaryFromParsed extracts TimelineSummary from parsed timeline.
func getSummaryFromParsed(parsed *execution.ParsedTimeline) execution.TimelineSummary {
	if parsed == nil {
		return execution.TimelineSummary{}
	}
	return parsed.Summary
}
