package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"test-genie/internal/shared"
)

var validScenarioName = regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`)

// Environment exposes scenario paths so phase runners can inspect files without shell scripts.
type Environment struct {
	ScenarioName string
	ScenarioDir  string
	TestDir      string
	AppRoot      string
}

// ScenarioWorkspace captures the canonical paths for a scenario so the orchestrator
// doesn't have to re-derive them on every call.
type ScenarioWorkspace struct {
	Name        string
	ScenarioDir string
	TestDir     string
	PhaseDir    string
	AppRoot     string

	artifactDir string
}

// New loads and validates the file-system layout for a scenario.
func New(scenariosRoot, scenario string) (*ScenarioWorkspace, error) {
	name := strings.TrimSpace(scenario)
	if name == "" {
		return nil, shared.NewValidationError("scenarioName is required")
	}
	if !validScenarioName.MatchString(name) {
		return nil, shared.NewValidationError("scenarioName may only contain letters, numbers, hyphens, or underscores")
	}

	scenarioDir := filepath.Join(scenariosRoot, name)
	info, err := os.Stat(scenarioDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, shared.NewValidationError(fmt.Sprintf("scenario '%s' was not found under %s", name, scenariosRoot))
		}
		return nil, fmt.Errorf("failed to read scenario '%s': %w", name, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("scenario path is not a directory: %s", scenarioDir)
	}

	testDir := filepath.Join(scenarioDir, "test")
	if err := EnsureDir(testDir); err != nil {
		return nil, err
	}

	phaseDir := filepath.Join(testDir, "phases")

	return &ScenarioWorkspace{
		Name:        name,
		ScenarioDir: scenarioDir,
		TestDir:     testDir,
		PhaseDir:    phaseDir,
		AppRoot:     AppRootFromScenario(scenarioDir),
	}, nil
}

// Environment returns the phase environment bound to this workspace.
func (w *ScenarioWorkspace) Environment() Environment {
	if w == nil {
		return Environment{}
	}
	return Environment{
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

// AppRootFromScenario returns the repository root given a scenario directory.
func AppRootFromScenario(scenarioDir string) string {
	return filepath.Clean(filepath.Join(scenarioDir, "..", ".."))
}
