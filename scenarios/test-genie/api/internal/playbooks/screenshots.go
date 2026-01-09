package playbooks

import (
	"context"

	"test-genie/internal/playbooks/artifacts"
	"test-genie/internal/playbooks/execution"
	"test-genie/internal/shared"
)

// downloadScreenshots downloads screenshot images from BAS for a workflow execution.
// It extracts screenshot references from the parsed timeline and downloads each one.
// Download failures are logged but don't stop the process - partial results are returned.
func (r *Runner) downloadScreenshots(ctx context.Context, workflowFile string, parsed *execution.ParsedTimeline) []artifacts.ScreenshotData {
	var screenshots []artifacts.ScreenshotData

	// Extract screenshot references from parsed timeline
	refs := artifacts.ExtractScreenshotsFromTimeline(parsed)
	if len(refs) == 0 {
		return screenshots
	}

	shared.LogStep(r.logWriter, "downloading %d screenshots for %s", len(refs), workflowFile)

	for _, ref := range refs {
		if ref.URL == "" {
			continue
		}

		data, err := r.basClient.DownloadAsset(ctx, ref.URL)
		if err != nil {
			shared.LogWarn(r.logWriter, "failed to download screenshot for step %d: %v", ref.StepIndex, err)
			continue
		}

		screenshots = append(screenshots, artifacts.ScreenshotData{
			StepIndex: ref.StepIndex,
			StepName:  ref.StepType,
			Filename:  artifacts.GenerateScreenshotFilename(ref),
			Data:      data,
		})
	}

	return screenshots
}
