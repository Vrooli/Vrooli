package playbooks

import (
	"context"
	"io"

	"test-genie/internal/playbooks/artifacts"
	"test-genie/internal/playbooks/execution"
	"test-genie/internal/shared"
)

const maxScreenshotArtifactBytes int64 = 32 * 1024 * 1024

// streamScreenshots persists each screenshot before requesting the next one.
// It retains only the resulting artifact reference in WorkflowArtifacts.
func (r *Runner) streamScreenshots(ctx context.Context, workflowFile string, parsed *execution.ParsedTimeline, writer *artifacts.FileWriter, output *artifacts.WorkflowArtifacts) {
	if writer == nil || output == nil {
		return
	}

	// Extract screenshot references from parsed timeline
	refs := artifacts.ExtractScreenshotsFromTimeline(parsed)
	if len(refs) == 0 {
		return
	}

	shared.LogStep(r.logWriter, "downloading %d screenshots for %s", len(refs), workflowFile)

	for _, ref := range refs {
		if ref.URL == "" {
			continue
		}

		path, err := writer.StreamScreenshot(workflowFile, artifacts.GenerateScreenshotFilename(ref), maxScreenshotArtifactBytes, func(destination io.Writer, maxBytes int64) (int64, error) {
			return r.basClient.StreamAsset(ctx, ref.URL, destination, maxBytes)
		})
		if err != nil {
			shared.LogWarn(r.logWriter, "failed to download screenshot for step %d: %v", ref.StepIndex, err)
			continue
		}
		output.Screenshots = append(output.Screenshots, path)
	}
}
