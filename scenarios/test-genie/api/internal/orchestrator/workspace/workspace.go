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
	TestDir         string
	AppRoot         string
	PhysicalAppRoot string
	Mapping         Mapping

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
	TestDir         string
	PhaseDir        string
	AppRoot         string
	PhysicalAppRoot string
	Mapping         Mapping

	artifactDir string

	// Runtime URLs (set via SetRuntimeURLs)
	uiURL          string
	apiURL         string
	browserlessURL string
}

// Options configures scenario workspace resolution.
type Options struct {
	// ScenarioPath is the physical scenario directory to read and write.
	ScenarioPath string
	// LogicalRepoRoot is the repo root used for repo-relative validation.
	LogicalRepoRoot string
	// LogicalScenarioRelPath is the logical scenario directory relative to LogicalRepoRoot.
	LogicalScenarioRelPath string
}

// Mapping captures how a scenario's physical files map to logical repo placement.
type Mapping struct {
	PhysicalScenarioDir    string
	PhysicalAppRoot        string
	LogicalRepoRoot        string
	LogicalScenarioRelPath string
}

// ResolvedLink reports a mapping-aware local link resolution.
type ResolvedLink struct {
	Exists          bool
	PhysicalPath    string
	LogicalPath     string
	OutsideScenario bool
	EscapesRoot     bool
}

// New loads and validates the file-system layout for a scenario.
func New(scenariosRoot, scenario string) (*ScenarioWorkspace, error) {
	return NewWithOptions(scenariosRoot, scenario, Options{})
}

// NewWithOptions loads and validates the file-system layout for a scenario.
func NewWithOptions(scenariosRoot, scenario string, opts Options) (*ScenarioWorkspace, error) {
	name := strings.TrimSpace(scenario)
	if name == "" {
		return nil, shared.NewValidationError("scenarioName is required")
	}
	if !validScenarioName.MatchString(name) {
		return nil, shared.NewValidationError("scenarioName may only contain letters, numbers, hyphens, or underscores")
	}

	scenarioDir := filepath.Join(scenariosRoot, name)
	if override := strings.TrimSpace(opts.ScenarioPath); override != "" {
		if !filepath.IsAbs(override) {
			return nil, shared.NewValidationError("scenarioPath must be absolute when provided")
		}
		scenarioDir = filepath.Clean(override)
		if filepath.Base(scenarioDir) != name {
			return nil, shared.NewValidationError("scenarioPath must match scenarioName")
		}
	}
	info, err := os.Stat(scenarioDir)
	if err != nil {
		if os.IsNotExist(err) {
			if strings.TrimSpace(opts.ScenarioPath) != "" {
				return nil, shared.NewValidationError(fmt.Sprintf("scenarioPath was not found: %s", scenarioDir))
			}
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
	physicalAppRoot := AppRootFromScenario(scenarioDir)
	mapping, err := NewMapping(scenarioDir, physicalAppRoot, opts.LogicalRepoRoot, opts.LogicalScenarioRelPath, name)
	if err != nil {
		return nil, err
	}

	return &ScenarioWorkspace{
		Name:            name,
		ScenarioDir:     scenarioDir,
		TestDir:         testDir,
		PhaseDir:        phaseDir,
		AppRoot:         physicalAppRoot,
		PhysicalAppRoot: physicalAppRoot,
		Mapping:         mapping,
	}, nil
}

// Environment returns the phase environment bound to this workspace.
func (w *ScenarioWorkspace) Environment() Environment {
	if w == nil {
		return Environment{}
	}
	return Environment{
		ScenarioName:    w.Name,
		ScenarioDir:     w.ScenarioDir,
		TestDir:         w.TestDir,
		AppRoot:         w.AppRoot,
		PhysicalAppRoot: w.PhysicalAppRoot,
		Mapping:         w.Mapping,
		UIURL:           w.uiURL,
		APIURL:          w.apiURL,
		BrowserlessURL:  w.browserlessURL,
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

// NewMapping validates and constructs a physical/logical workspace mapping.
func NewMapping(physicalScenarioDir, physicalAppRoot, logicalRepoRoot, logicalScenarioRelPath, scenarioName string) (Mapping, error) {
	mapping := Mapping{
		PhysicalScenarioDir: filepath.Clean(physicalScenarioDir),
		PhysicalAppRoot:     filepath.Clean(physicalAppRoot),
	}
	logicalRepoRoot = strings.TrimSpace(logicalRepoRoot)
	logicalScenarioRelPath = strings.TrimSpace(logicalScenarioRelPath)
	if logicalRepoRoot == "" && logicalScenarioRelPath == "" {
		return mapping, nil
	}
	if logicalRepoRoot == "" || logicalScenarioRelPath == "" {
		return Mapping{}, shared.NewValidationError("logicalRepoRoot and logicalScenarioRelPath must be provided together")
	}
	if !filepath.IsAbs(logicalRepoRoot) {
		return Mapping{}, shared.NewValidationError("logicalRepoRoot must be absolute")
	}
	if filepath.IsAbs(logicalScenarioRelPath) {
		return Mapping{}, shared.NewValidationError("logicalScenarioRelPath must be relative")
	}
	cleanRel := filepath.Clean(logicalScenarioRelPath)
	if cleanRel == "." || cleanRel == "" {
		return Mapping{}, shared.NewValidationError("logicalScenarioRelPath must not be empty")
	}
	if cleanRel == ".." || strings.HasPrefix(cleanRel, ".."+string(filepath.Separator)) {
		return Mapping{}, shared.NewValidationError("logicalScenarioRelPath must not escape the logical repo root")
	}
	if filepath.Base(cleanRel) != scenarioName {
		return Mapping{}, shared.NewValidationError("logicalScenarioRelPath must match scenarioName")
	}
	mapping.LogicalRepoRoot = filepath.Clean(logicalRepoRoot)
	mapping.LogicalScenarioRelPath = cleanRel
	return mapping, nil
}

func (m Mapping) HasLogicalPlacement() bool {
	return m.LogicalRepoRoot != "" && m.LogicalScenarioRelPath != ""
}

func (m Mapping) LogicalScenarioDir() string {
	if !m.HasLogicalPlacement() {
		return m.PhysicalScenarioDir
	}
	return filepath.Join(m.LogicalRepoRoot, m.LogicalScenarioRelPath)
}

func (m Mapping) PhysicalPath(rel string) string {
	if filepath.IsAbs(rel) {
		return filepath.Clean(rel)
	}
	return filepath.Join(m.PhysicalScenarioDir, rel)
}

func (m Mapping) LogicalPath(rel string) string {
	if filepath.IsAbs(rel) {
		return filepath.Clean(rel)
	}
	return filepath.Join(m.LogicalScenarioDir(), rel)
}

func (m Mapping) PhysicalToLogical(absPhysical string) (string, bool) {
	absPhysical = filepath.Clean(absPhysical)
	rel, ok := relativeWithin(m.PhysicalScenarioDir, absPhysical)
	if !ok {
		return "", false
	}
	return m.LogicalPath(rel), true
}

func (m Mapping) LogicalToPhysical(absLogical string) (string, bool) {
	absLogical = filepath.Clean(absLogical)
	if !m.HasLogicalPlacement() {
		rel, ok := relativeWithin(m.PhysicalScenarioDir, absLogical)
		if !ok {
			return "", false
		}
		return m.PhysicalPath(rel), true
	}
	rel, ok := relativeWithin(m.LogicalScenarioDir(), absLogical)
	if !ok {
		return "", false
	}
	return m.PhysicalPath(rel), true
}

func (m Mapping) ResolveLocalLink(fromPhysicalFile, href string) ResolvedLink {
	if filepath.IsAbs(href) {
		info, err := os.Stat(href)
		return ResolvedLink{Exists: err == nil && !info.IsDir(), PhysicalPath: filepath.Clean(href), LogicalPath: filepath.Clean(href)}
	}
	if !m.HasLogicalPlacement() {
		target := filepath.Clean(filepath.Join(filepath.Dir(fromPhysicalFile), href))
		info, err := os.Stat(target)
		return ResolvedLink{Exists: err == nil && !info.IsDir(), PhysicalPath: target, LogicalPath: target}
	}
	fromLogical, ok := m.PhysicalToLogical(fromPhysicalFile)
	if !ok {
		fromLogical = filepath.Join(m.LogicalScenarioDir(), filepath.Base(fromPhysicalFile))
	}
	logicalTarget := filepath.Clean(filepath.Join(filepath.Dir(fromLogical), href))
	resolved := ResolvedLink{LogicalPath: logicalTarget}
	if _, ok := relativeWithin(m.LogicalRepoRoot, logicalTarget); !ok {
		resolved.EscapesRoot = true
		return resolved
	}
	if physicalTarget, ok := m.LogicalToPhysical(logicalTarget); ok {
		resolved.PhysicalPath = physicalTarget
		info, err := os.Stat(physicalTarget)
		resolved.Exists = err == nil && !info.IsDir()
		return resolved
	}
	resolved.OutsideScenario = true
	resolved.PhysicalPath = logicalTarget
	info, err := os.Stat(logicalTarget)
	resolved.Exists = err == nil && !info.IsDir()
	return resolved
}

func relativeWithin(base, target string) (string, bool) {
	base = filepath.Clean(base)
	target = filepath.Clean(target)
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return "", false
	}
	if rel == "." {
		return "", true
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return rel, true
}
