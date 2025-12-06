package artifacts

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"test-genie/internal/playbooks/execution"
	"test-genie/internal/playbooks/types"
	sharedartifacts "test-genie/internal/shared/artifacts"

	basv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1"
)

// WorkflowArtifacts contains paths to all artifacts for a workflow execution.
type WorkflowArtifacts struct {
	// Dir is the workflow-specific directory under coverage/automation/
	Dir string `json:"dir"`
	// Timeline is the path to the full timeline JSON
	Timeline string `json:"timeline,omitempty"`
	// Console is the path to console logs JSON
	Console string `json:"console,omitempty"`
	// DOM is the path to the final DOM snapshot HTML
	DOM string `json:"dom,omitempty"`
	// Assertions is the path to assertions JSON
	Assertions string `json:"assertions,omitempty"`
	// Latest is the path to the result summary JSON
	Latest string `json:"latest,omitempty"`
	// Readme is the path to the README.md
	Readme string `json:"readme,omitempty"`
	// Screenshots is a list of screenshot file paths
	Screenshots []string `json:"screenshots,omitempty"`
	// Proto contains the parsed proto timeline for error diagnostics.
	// This is not serialized to JSON but available in-memory for error context.
	Proto *basv1.ExecutionTimeline `json:"-"`
}

// WorkflowResult contains the execution result for README generation.
type WorkflowResult struct {
	WorkflowFile  string
	Description   string
	Requirements  []string
	ExecutionID   string
	Success       bool
	Status        string
	Error         string
	DurationMs    int64
	Timestamp     time.Time
	Summary       execution.TimelineSummary
	ParsedSummary *execution.ParsedTimeline
	Artifacts     WorkflowArtifacts
}

// WriteWorkflowArtifacts writes all artifacts for a single workflow execution.
// It creates a workflow-specific directory and writes:
// - timeline.json (full BAS timeline)
// - console.json (extracted logs)
// - dom.html (final DOM snapshot)
// - assertions.json (assertion results)
// - latest.json (result summary)
// - README.md (human-readable summary)
// - screenshots/*.png (step screenshots)
func (w *FileWriter) WriteWorkflowArtifacts(
	workflowFile string,
	timelineData []byte,
	parsed *execution.ParsedTimeline,
	screenshots []ScreenshotData,
	result *WorkflowResult,
) (*WorkflowArtifacts, error) {
	// Create workflow-specific directory
	workflowDir := w.workflowDir(workflowFile)
	if err := w.EnsureDir(workflowDir); err != nil {
		return nil, fmt.Errorf("failed to create workflow dir: %w", err)
	}

	artifacts := &WorkflowArtifacts{
		Dir: sharedartifacts.RelPath(w.appRoot, workflowDir),
	}

	// Write timeline JSON
	if len(timelineData) > 0 {
		timelinePath := filepath.Join(workflowDir, "timeline.json")
		prettyTimeline := sharedartifacts.PrettyPrintJSON(timelineData)
		if err := w.WriteFile(timelinePath, prettyTimeline); err != nil {
			return artifacts, fmt.Errorf("failed to write timeline: %w", err)
		}
		artifacts.Timeline = sharedartifacts.RelPath(w.appRoot, timelinePath)
	}

	// Write extracted data from parsed timeline
	if parsed != nil {
		// Write console logs
		if len(parsed.Logs) > 0 {
			consolePath := filepath.Join(workflowDir, "console.json")
			if err := w.writeJSON(consolePath, parsed.Logs); err != nil {
				// Log error but continue
				fmt.Printf("Warning: failed to write console.json: %v\n", err)
			} else {
				artifacts.Console = sharedartifacts.RelPath(w.appRoot, consolePath)
			}
		}

		// Write assertions
		if len(parsed.Assertions) > 0 {
			assertionsPath := filepath.Join(workflowDir, "assertions.json")
			if err := w.writeJSON(assertionsPath, parsed.Assertions); err != nil {
				fmt.Printf("Warning: failed to write assertions.json: %v\n", err)
			} else {
				artifacts.Assertions = sharedartifacts.RelPath(w.appRoot, assertionsPath)
			}
		}

		// Write final DOM
		if parsed.FinalDOM != "" {
			domPath := filepath.Join(workflowDir, "dom.html")
			if err := w.WriteFile(domPath, []byte(parsed.FinalDOM)); err != nil {
				fmt.Printf("Warning: failed to write dom.html: %v\n", err)
			} else {
				artifacts.DOM = sharedartifacts.RelPath(w.appRoot, domPath)
			}
		}
	}

	// Write screenshots
	if len(screenshots) > 0 {
		screenshotsDir := filepath.Join(workflowDir, "screenshots")
		if err := w.EnsureDir(screenshotsDir); err != nil {
			fmt.Printf("Warning: failed to create screenshots dir: %v\n", err)
		} else {
			for _, ss := range screenshots {
				filename := ss.Filename
				if filename == "" {
					filename = fmt.Sprintf("step-%02d.png", ss.StepIndex)
				}
				ssPath := filepath.Join(screenshotsDir, filename)
				if err := w.WriteFile(ssPath, ss.Data); err != nil {
					fmt.Printf("Warning: failed to write screenshot %s: %v\n", filename, err)
				} else {
					artifacts.Screenshots = append(artifacts.Screenshots, sharedartifacts.RelPath(w.appRoot, ssPath))
				}
			}
		}
	}

	// Write latest.json (result summary)
	if result != nil {
		latestPath := filepath.Join(workflowDir, "latest.json")
		latestData := buildLatestJSON(result, artifacts)
		if err := w.writeJSON(latestPath, latestData); err != nil {
			fmt.Printf("Warning: failed to write latest.json: %v\n", err)
		} else {
			artifacts.Latest = sharedartifacts.RelPath(w.appRoot, latestPath)
		}
	}

	// Write README.md
	if result != nil {
		readmePath := filepath.Join(workflowDir, "README.md")
		result.Artifacts = *artifacts
		readmeContent := GenerateWorkflowReadme(result)
		if err := w.WriteFile(readmePath, []byte(readmeContent)); err != nil {
			fmt.Printf("Warning: failed to write README.md: %v\n", err)
		} else {
			artifacts.Readme = sharedartifacts.RelPath(w.appRoot, readmePath)
		}
	}

	return artifacts, nil
}

// ScreenshotData contains downloaded screenshot data.
type ScreenshotData struct {
	StepIndex int
	StepName  string
	Filename  string
	Data      []byte
}

// workflowDir returns the directory path for a workflow's artifacts.
func (w *FileWriter) workflowDir(workflowFile string) string {
	// Sanitize workflow file path to create a valid directory name
	sanitized := sharedartifacts.SanitizeFilenameWithoutExtension(workflowFile)
	return filepath.Join(w.ScenarioDir, sharedartifacts.AutomationDir, sanitized)
}

// writeJSON writes data as pretty-printed JSON.
func (w *FileWriter) writeJSON(path string, data any) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return w.WriteFile(path, jsonData)
}

// buildLatestJSON constructs the latest.json content.
func buildLatestJSON(result *WorkflowResult, artifacts *WorkflowArtifacts) map[string]any {
	data := map[string]any{
		"workflow":     result.WorkflowFile,
		"status":       result.Status,
		"success":      result.Success,
		"timestamp":    result.Timestamp.Format(time.RFC3339),
		"updated_at":   time.Now().UTC().Format(time.RFC3339),
	}

	if result.Description != "" {
		data["description"] = result.Description
	}
	if result.ExecutionID != "" {
		data["execution_id"] = result.ExecutionID
	}
	if result.DurationMs > 0 {
		data["duration_ms"] = result.DurationMs
	}
	if result.Error != "" {
		data["error"] = result.Error
	}
	if len(result.Requirements) > 0 {
		data["requirements"] = result.Requirements
	}

	// Add summary stats
	summary := map[string]int{
		"total_steps":     result.Summary.TotalSteps,
		"total_asserts":   result.Summary.TotalAsserts,
		"asserts_passed":  result.Summary.AssertsPassed,
	}
	data["summary"] = summary

	// Add artifact paths
	if artifacts != nil {
		data["artifacts"] = artifacts
	}

	return data
}

// ExtractScreenshotsFromTimeline extracts screenshot references from parsed timeline
// for later download.
func ExtractScreenshotsFromTimeline(parsed *execution.ParsedTimeline) []ScreenshotRef {
	var refs []ScreenshotRef
	if parsed == nil {
		return refs
	}

	for _, frame := range parsed.Frames {
		if frame.Screenshot != nil && frame.Screenshot.URL != "" {
			refs = append(refs, ScreenshotRef{
				StepIndex:  frame.StepIndex,
				NodeID:     frame.NodeID,
				StepType:   frame.StepType,
				URL:        frame.Screenshot.URL,
				ArtifactID: frame.Screenshot.ArtifactID,
			})
		}
	}

	return refs
}

// ScreenshotRef contains info needed to download a screenshot.
type ScreenshotRef struct {
	StepIndex  int
	NodeID     string
	StepType   string
	URL        string
	ArtifactID string
}

// GenerateScreenshotFilename creates a filename for a screenshot.
func GenerateScreenshotFilename(ref ScreenshotRef) string {
	// Create descriptive filename: step-01-navigate.png
	stepType := ref.StepType
	if stepType == "" {
		stepType = "step"
	}
	// Sanitize step type
	stepType = strings.ReplaceAll(stepType, "/", "-")
	stepType = strings.ReplaceAll(stepType, " ", "-")
	return fmt.Sprintf("step-%02d-%s.png", ref.StepIndex, stepType)
}

// Legacy method for backward compatibility
// WriteTimeline writes a timeline dump to the automation directory (flat structure).
// Deprecated: Use WriteWorkflowArtifacts for full artifact collection.
func (w *FileWriter) WriteTimelineLegacy(workflowFile string, timelineData []byte) (string, error) {
	targetDir := filepath.Join(w.ScenarioDir, sharedartifacts.AutomationDir)
	if err := w.EnsureDir(targetDir); err != nil {
		return "", fmt.Errorf("failed to create timeline dir: %w", err)
	}

	filename := sharedartifacts.SanitizeFilenameWithoutExtension(workflowFile)
	path := filepath.Join(targetDir, filename+".timeline.json")

	if err := w.WriteFile(path, timelineData); err != nil {
		return "", fmt.Errorf("failed to write timeline: %w", err)
	}

	return sharedartifacts.RelPath(w.appRoot, path), nil
}

// RequirementCoverageEntry represents a single requirement's coverage from playbooks.
type RequirementCoverageEntry struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	Phase     string `json:"phase"`
	Evidence  string `json:"evidence"`
	UpdatedAt string `json:"updated_at"`
}

// BuildRequirementCoverage builds requirement coverage entries from workflow results.
func BuildRequirementCoverage(results []types.Result) []RequirementCoverageEntry {
	var entries []RequirementCoverageEntry

	for _, result := range results {
		status := "passed"
		evidence := result.Entry.File

		if result.Outcome != nil {
			if result.Outcome.Stats != "" {
				evidence = fmt.Sprintf("%s%s", result.Entry.File, result.Outcome.Stats)
			}
			if result.Outcome.Duration > 0 {
				evidence = fmt.Sprintf("%s in %s", evidence, result.Outcome.Duration.Truncate(time.Millisecond))
			}
		}

		if result.Err != nil {
			status = "failed"
			evidence = fmt.Sprintf("%s failed: %v", result.Entry.File, result.Err)
		}

		for _, reqID := range result.Entry.Requirements {
			entries = append(entries, RequirementCoverageEntry{
				ID:        reqID,
				Status:    status,
				Phase:     "playbooks",
				Evidence:  evidence,
				UpdatedAt: time.Now().UTC().Format(time.RFC3339),
			})
		}
	}

	return entries
}
