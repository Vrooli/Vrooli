package artifacts

import (
	"fmt"
	"path/filepath"
	"time"

	"test-genie/internal/playbooks/types"
	sharedartifacts "test-genie/internal/shared/artifacts"
)

// Writer defines the interface for writing playbook artifacts.
type Writer interface {
	// WriteTimeline writes a timeline dump for a workflow execution.
	WriteTimeline(workflowFile string, timelineData []byte) (string, error)
	// WritePhaseResults writes the phase results JSON.
	WritePhaseResults(results []types.Result) error
}

// FileWriter writes artifacts to the filesystem.
type FileWriter struct {
	*sharedartifacts.BaseWriter
	appRoot string
}

// NewWriter creates a new artifact writer.
func NewWriter(scenarioDir, scenarioName, appRoot string, opts ...sharedartifacts.BaseWriterOption) *FileWriter {
	return &FileWriter{
		BaseWriter: sharedartifacts.NewBaseWriter(scenarioDir, scenarioName, opts...),
		appRoot:    appRoot,
	}
}

// WriteTimeline writes a timeline dump for a workflow execution.
// Returns the relative path to the artifact.
func (w *FileWriter) WriteTimeline(workflowFile string, timelineData []byte) (string, error) {
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

// WritePhaseResults writes the phase results JSON.
func (w *FileWriter) WritePhaseResults(results []types.Result) error {
	if len(results) == 0 {
		return nil
	}
	return sharedartifacts.WritePhaseResults(w.BaseWriter, sharedartifacts.PhaseResultsPlaybooks, results, buildPhaseOutput)
}

// buildPhaseOutput constructs the phase results structure.
func buildPhaseOutput(scenarioName string, results []types.Result) map[string]any {
	output := map[string]any{
		"phase":      "playbooks",
		"scenario":   scenarioName,
		"tests":      len(results),
		"errors":     0,
		"status":     "passed",
		"updated_at": time.Now().UTC().Format(time.RFC3339),
	}

	var requirementEntries []map[string]any
	errorCount := 0

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
			errorCount++
		}

		if result.ArtifactPath != "" {
			evidence = fmt.Sprintf("%s (artifact: %s)", evidence, result.ArtifactPath)
		}

		for _, reqID := range result.Entry.Requirements {
			requirementEntries = append(requirementEntries, map[string]any{
				"id":         reqID,
				"status":     status,
				"phase":      "playbooks",
				"evidence":   evidence,
				"updated_at": time.Now().UTC().Format(time.RFC3339),
			})
		}
	}

	output["errors"] = errorCount
	if errorCount > 0 {
		output["status"] = "failed"
	}
	output["requirements"] = requirementEntries

	return output
}
