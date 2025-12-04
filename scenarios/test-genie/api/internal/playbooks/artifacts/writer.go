package artifacts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"test-genie/internal/playbooks/types"
)

const (
	// TimelineDir is the directory for timeline artifacts.
	TimelineDir = "coverage/automation"
	// PhaseResultsDir is the directory for phase results.
	PhaseResultsDir = "coverage/phase-results"
	// PhaseResultsFile is the filename for playbooks phase results.
	PhaseResultsFile = "playbooks.json"
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
	scenarioDir  string
	scenarioName string
	appRoot      string
}

// NewWriter creates a new artifact writer.
func NewWriter(scenarioDir, scenarioName, appRoot string) *FileWriter {
	return &FileWriter{
		scenarioDir:  scenarioDir,
		scenarioName: scenarioName,
		appRoot:      appRoot,
	}
}

// WriteTimeline writes a timeline dump for a workflow execution.
// Returns the relative path to the artifact.
func (w *FileWriter) WriteTimeline(workflowFile string, timelineData []byte) (string, error) {
	targetDir := filepath.Join(w.scenarioDir, TimelineDir)
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create timeline dir: %w", err)
	}

	filename := sanitizeArtifactName(workflowFile)
	path := filepath.Join(targetDir, filename+".timeline.json")

	if err := os.WriteFile(path, timelineData, 0o644); err != nil {
		return "", fmt.Errorf("failed to write timeline: %w", err)
	}

	// Return relative path if possible
	if rel, err := filepath.Rel(w.appRoot, path); err == nil {
		return rel, nil
	}
	return path, nil
}

// WritePhaseResults writes the phase results JSON.
func (w *FileWriter) WritePhaseResults(results []types.Result) error {
	if len(results) == 0 {
		return nil
	}

	phaseDir := filepath.Join(w.scenarioDir, PhaseResultsDir)
	if err := os.MkdirAll(phaseDir, 0o755); err != nil {
		return fmt.Errorf("failed to create phase results dir: %w", err)
	}

	output := buildPhaseOutput(w.scenarioName, results)

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal phase results: %w", err)
	}

	path := filepath.Join(phaseDir, PhaseResultsFile)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write phase results: %w", err)
	}

	return nil
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

// sanitizeArtifactName converts a workflow file path to a safe artifact name.
func sanitizeArtifactName(name string) string {
	// Replace path separators with dashes
	name = strings.ReplaceAll(name, string(filepath.Separator), "-")
	name = strings.ReplaceAll(name, "/", "-")

	// Remove extension
	name = strings.TrimSuffix(name, filepath.Ext(name))

	// Keep only alphanumeric, dash, and underscore
	name = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, name)

	return name
}
