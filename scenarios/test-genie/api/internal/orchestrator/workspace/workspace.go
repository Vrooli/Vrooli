package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"test-genie/internal/shared"
	sharedartifacts "test-genie/internal/shared/artifacts"
)

var validScenarioName = regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`)

// Environment exposes scenario paths and runtime URLs so phase runners can inspect files
// and connect to running services without shell scripts.
type Environment struct {
	ScenarioName string
	ScenarioDir  string
	// TestDir is the legacy "testing workspace" root. Vrooli no longer requires
	// scenarios to include a top-level test/ directory; this now defaults to
	// coverage/ to keep all test-related artifacts and optional configs together.
	TestDir string
	AppRoot      string

	// Runtime URLs for phases that need to connect to running services.
	// These are optional and may be empty if the scenario isn't running.
	UIURL          string // Base URL for the scenario UI (e.g., "http://localhost:3000")
	APIURL         string // Base URL for the scenario API (e.g., "http://localhost:8080")
	BrowserlessURL string // URL for Browserless service (e.g., "http://localhost:4110")
}

// ScenarioWorkspace captures the canonical paths for a scenario so the orchestrator
// doesn't have to re-derive them on every call.
type ScenarioWorkspace struct {
	Name        string
	ScenarioDir string
	// TestDir is the legacy "testing workspace" root (now coverage/ by default).
	TestDir  string
	PhaseDir string
	AppRoot     string

	artifactDir string

	// Runtime URLs (set via SetRuntimeURLs)
	uiURL          string
	apiURL         string
	browserlessURL string
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

	testDir := filepath.Join(scenarioDir, "coverage")
	if err := os.MkdirAll(testDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create coverage directory: %w", err)
	}
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
		ScenarioName:   w.Name,
		ScenarioDir:    w.ScenarioDir,
		TestDir:        w.TestDir,
		AppRoot:        w.AppRoot,
		UIURL:          w.uiURL,
		APIURL:         w.apiURL,
		BrowserlessURL: w.browserlessURL,
	}
}

// SetRuntimeURLs configures the runtime service URLs for phases that need to connect
// to running services (e.g., Lighthouse audits, integration tests).
func (w *ScenarioWorkspace) SetRuntimeURLs(uiURL, apiURL, browserlessURL string) {
	if w == nil {
		return
	}
	w.uiURL = uiURL
	w.apiURL = apiURL
	w.browserlessURL = browserlessURL
}

// EnsureArtifactDir lazily creates the artifact directory and returns its path.
func (w *ScenarioWorkspace) EnsureArtifactDir() (string, error) {
	if w == nil {
		return "", fmt.Errorf("workspace is not configured")
	}
	if w.artifactDir != "" {
		return w.artifactDir, nil
	}
	dir := filepath.Join(w.ScenarioDir, sharedartifacts.LogsDir)
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
