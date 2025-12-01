package suite

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ScenarioWorkspace captures the canonical paths for a scenario so the orchestrator
// doesn't have to re-derive them on every call. This makes the structure explicit
// and easier to test as the Go runner replaces bash scripts.
type ScenarioWorkspace struct {
	Name        string
	ScenarioDir string
	TestDir     string
	PhaseDir    string
	AppRoot     string

	artifactDir string
}

func newScenarioWorkspace(scenariosRoot, scenario string) (*ScenarioWorkspace, error) {
	name := strings.TrimSpace(scenario)
	if name == "" {
		return nil, NewValidationError("scenarioName is required")
	}
	if !validScenarioName.MatchString(name) {
		return nil, NewValidationError("scenarioName may only contain letters, numbers, hyphens, or underscores")
	}

	scenarioDir := filepath.Join(scenariosRoot, name)
	info, err := os.Stat(scenarioDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, NewValidationError(fmt.Sprintf("scenario '%s' was not found under %s", name, scenariosRoot))
		}
		return nil, fmt.Errorf("failed to read scenario '%s': %w", name, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("scenario path is not a directory: %s", scenarioDir)
	}

	testDir := filepath.Join(scenarioDir, "test")
	if err := ensureDir(testDir); err != nil {
		return nil, err
	}

	phaseDir := filepath.Join(testDir, "phases")
	if err := ensureDir(phaseDir); err != nil {
		return nil, fmt.Errorf("scenario '%s' is missing test phases directory at %s", name, phaseDir)
	}

	return &ScenarioWorkspace{
		Name:        name,
		ScenarioDir: scenarioDir,
		TestDir:     testDir,
		PhaseDir:    phaseDir,
		AppRoot:     appRootFromScenario(scenarioDir),
	}, nil
}

// Environment returns the phase environment bound to this workspace.
func (w *ScenarioWorkspace) Environment() PhaseEnvironment {
	if w == nil {
		return PhaseEnvironment{}
	}
	return PhaseEnvironment{
		ScenarioName: w.Name,
		ScenarioDir:  w.ScenarioDir,
		TestDir:      w.TestDir,
		AppRoot:      w.AppRoot,
	}
}

// EnsureArtifactDir lazily creates the artifact directory and returns its path.
func (w *ScenarioWorkspace) EnsureArtifactDir() (string, error) {
	if w == nil {
		return "", fmt.Errorf("workspace is not configured")
	}
	if w.artifactDir != "" {
		return w.artifactDir, nil
	}
	dir := filepath.Join(w.TestDir, "artifacts")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create artifact directory: %w", err)
	}
	w.artifactDir = dir
	return dir, nil
}
