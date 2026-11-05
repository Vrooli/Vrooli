package artifacts

import (
	"encoding/json"
	"fmt"
	"path/filepath"
)

// Note: PhaseResultsDir and other path constants are defined in paths.go

// BaseWriter provides common artifact writing functionality. All artifact
// paths it produces are keyed by RunID under coverage/runs/<runID>/.
type BaseWriter struct {
	ScenarioDir  string
	ScenarioName string
	RunID        string
	FS           FileSystem
}

// BaseWriterOption configures a BaseWriter.
type BaseWriterOption func(*BaseWriter)

// NewBaseWriter creates a new BaseWriter with the given options. runID keys all
// artifact paths under coverage/runs/<runID>/.
func NewBaseWriter(scenarioDir, scenarioName, runID string, opts ...BaseWriterOption) *BaseWriter {
	w := &BaseWriter{
		ScenarioDir:  scenarioDir,
		ScenarioName: scenarioName,
		RunID:        runID,
		FS:           OSFileSystem{},
	}
	for _, opt := range opts {
		opt(w)
	}
	return w
}

// WithFileSystem sets a custom filesystem implementation.
func WithFileSystem(fs FileSystem) BaseWriterOption {
	return func(w *BaseWriter) {
		w.FS = fs
	}
}

// EnsureDir creates a directory if it doesn't exist.
func (w *BaseWriter) EnsureDir(path string) error {
	return w.FS.MkdirAll(path, 0o755)
}

// WriteJSON writes data as pretty-printed JSON to the given path.
func (w *BaseWriter) WriteJSON(path string, data any) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return w.FS.WriteFile(path, jsonData, 0o644)
}

// WriteFile writes raw data to the given path.
func (w *BaseWriter) WriteFile(path string, data []byte) error {
	return w.FS.WriteFile(path, data, 0o644)
}

// PhaseResultsPath returns the path to a phase results file for this run.
func (w *BaseWriter) PhaseResultsPath(filename string) string {
	return PhaseResultsPath(w.ScenarioDir, w.RunID, filename)
}

// WritePhaseResults writes phase results to the run-keyed location.
// The buildOutput function constructs the phase-specific output structure.
func WritePhaseResults[T any](
	w *BaseWriter,
	filename string,
	results T,
	buildOutput func(scenarioName string, results T) map[string]any,
) error {
	phaseDir := RunPhaseResultsDir(w.ScenarioDir, w.RunID)
	if err := w.EnsureDir(phaseDir); err != nil {
		return fmt.Errorf("failed to create phase results dir: %w", err)
	}

	output := buildOutput(w.ScenarioName, results)

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal phase results: %w", err)
	}

	path := filepath.Join(phaseDir, filename)
	if err := w.FS.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write phase results: %w", err)
	}

	return nil
}

// AbsPath returns the absolute path, cleaning up any relative components.
// If the path cannot be made absolute, the original path is returned.
func AbsPath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}

// RelPath returns a relative path from the base to the target.
// If the relative path cannot be computed, the target path is returned.
func RelPath(base, target string) string {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return target
	}
	return rel
}

// PrettyPrintJSON attempts to pretty-print JSON data.
// If the data is not valid JSON, it is returned unchanged.
func PrettyPrintJSON(data []byte) []byte {
	var obj any
	if err := json.Unmarshal(data, &obj); err == nil {
		if pretty, err := json.MarshalIndent(obj, "", "  "); err == nil {
			return pretty
		}
	}
	return data
}
