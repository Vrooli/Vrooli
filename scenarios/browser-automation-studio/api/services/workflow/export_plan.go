package workflow

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/vrooli/browser-automation-studio/services/export"
)

// PlannedFile describes a single text/binary file to materialize under
// the export output directory. Content is fully resolved by the pure
// builder; WriteExportPlan only performs the I/O.
type PlannedFile struct {
	// RelPath is the file path relative to the export output directory.
	RelPath string
	// Content is the bytes to write.
	Content []byte
}

// PlannedScreenshot describes a screenshot to copy out of object storage.
// The pure builder cannot read storage, so it records the source object
// and the canonical destination filename; WriteExportPlan downloads and
// writes it (best-effort — a missing object is skipped, not fatal).
type PlannedScreenshot struct {
	// ObjectName is the storage object key (URL prefix already stripped).
	ObjectName string
	// FallbackExt is the extension to use when storage metadata does not
	// disambiguate the content type (e.g. "png"). The screenshot's own
	// declared content type wins when present.
	FallbackExt string
	// RelPath is the destination path relative to the export output dir,
	// without extension (extension resolved at write time from content
	// type). e.g. "screenshots/step-01-nav".
	RelPathNoExt string
}

// ExportPlan is the pure description of everything ExportToFolder writes.
// BuildExportPlan produces it from a timeline; WriteExportPlan executes
// it. Splitting the two makes the markdown/json generation unit-testable
// without any filesystem or storage dependency.
type ExportPlan struct {
	Files       []PlannedFile
	Screenshots []PlannedScreenshot
}

// BuildExportPlan is the PURE half of ExportToFolder: given a timeline, the
// human workflow name, and the persisted execution error, it returns the full
// set of files (with resolved content) and screenshot copy operations,
// performing no I/O.
func BuildExportPlan(timeline *export.ExecutionTimeline, workflowName, executionError string) (*ExportPlan, error) {
	if workflowName == "" {
		workflowName = "Unnamed Workflow"
	}

	plan := &ExportPlan{}

	timelineJSON, err := json.MarshalIndent(timeline, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal timeline: %w", err)
	}
	plan.Files = append(plan.Files, PlannedFile{RelPath: "timeline.json", Content: timelineJSON})

	result := buildResultSummary(timeline, executionError)
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	plan.Files = append(plan.Files, PlannedFile{RelPath: "result.json", Content: resultJSON})

	plan.Files = append(plan.Files,
		PlannedFile{RelPath: "README.md", Content: []byte(export.GenerateTimelineMarkdown(timeline, workflowName))},
		PlannedFile{RelPath: "execution-summary.md", Content: []byte(export.GenerateExecutionSummaryMarkdown(timeline))},
		PlannedFile{RelPath: "console-logs.md", Content: []byte(export.GenerateConsoleLogsMarkdown(timeline))},
		PlannedFile{RelPath: "network-activity.md", Content: []byte(export.GenerateNetworkActivityMarkdown(timeline))},
		PlannedFile{RelPath: "assertions.md", Content: []byte(export.GenerateAssertionsMarkdown(timeline))},
	)

	for i, frame := range timeline.Frames {
		if frame.Screenshot == nil || frame.Screenshot.URL == "" {
			continue
		}
		objectName := strings.TrimPrefix(frame.Screenshot.URL, "/api/v1/screenshots/")
		fallbackExt := "png"
		if frame.Screenshot.ContentType == "image/jpeg" {
			fallbackExt = "jpg"
		}
		nodeID := sanitizeFilename(frame.NodeID)
		plan.Screenshots = append(plan.Screenshots, PlannedScreenshot{
			ObjectName:   objectName,
			FallbackExt:  fallbackExt,
			RelPathNoExt: filepath.Join("screenshots", fmt.Sprintf("step-%02d-%s", i+1, nodeID)),
		})
	}

	return plan, nil
}
