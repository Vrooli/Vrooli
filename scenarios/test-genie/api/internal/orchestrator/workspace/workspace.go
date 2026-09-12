package workspace

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vrooli/vrooli/packages/artifactpaths"
	"test-genie/internal/shared"
	"test-genie/internal/targetmodel"

	sharedartifacts "test-genie/internal/shared/artifacts"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

var validScenarioName = regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`)

// Environment exposes scenario paths and runtime URLs so phase runners can inspect files
// and connect to running services without shell scripts.
type Environment struct {
	// RunID keys all artifact writes for this execution under
	// coverage/runs/<RunID>/. The orchestrator mints it once per run and
	// threads it into every phase writer.
	RunID string

	// DiagnosticsPreset, when non-empty ("none"|"light"|"full"), records the
	// requested provider-evidence depth for this run.
	DiagnosticsPreset string

	// CaptureProfile is the capture-depth dial. "" (default) keeps routine runs
	// at default visual-artifact depth; "baseline" requests all-pages visual
	// capture + video. See internal/captureprofile.
	CaptureProfile string
	// Cache identity is frozen during planning. Empty values make a phase
	// observational rather than guessable.
	DescriptorSnapshotDigest     string
	ExecutionConfigurationDigest string
	SchedulerDecision            string
	// CapabilitySubset is an execution-time override used only by mixed
	// determinism phases. It is kept on the workspace environment so the
	// orchestrator can request the uncached capabilities without changing the
	// provider-owned phase runner contract.
	CapabilitySubset []string

	ScenarioName string
	TargetKind   string
	TargetID     string
	TargetRoot   string
	// Exclude contains contract-owned repository-relative globs for the target.
	Exclude     []string
	ScenarioDir string
	// ArtifactRoot is the durable evidence owner. It is the source directory
	// for scenarios and a runtime-home target directory otherwise.
	ArtifactRoot   string
	PhaseCacheRoot string
	// CoverageDir is the testing workspace root. Vrooli no longer requires
	// scenarios to include a top-level test/ directory; this defaults to
	// coverage/ to keep all test-related artifacts and optional configs together.
	CoverageDir     string
	AppRoot         string
	PhysicalAppRoot string
	Mapping         Mapping

	// Runtime URLs for phases that need to connect to running services.
	// These are optional and may be empty if the scenario isn't running.
	UIURL         string // Base URL for the scenario UI (e.g., "http://localhost:3000")
	APIURL        string // Base URL for the scenario API (e.g., "http://localhost:8080")
	TargetRuntime TargetRuntime
}

// EffectivePhaseCacheRoot preserves explicit test seams while production uses
// the independently resolved cache-class root.
func (e Environment) EffectivePhaseCacheRoot() string {
	return firstNonEmpty(e.PhaseCacheRoot, e.ArtifactRoot)
}

// TargetRuntime manages the lifecycle of the scenario under test when a phase
// needs the target restarted with temporary resources.
type TargetRuntime interface {
	RestartWithEnv(ctx context.Context, env map[string]string, logWriter io.Writer) error
	Restore(ctx context.Context, logWriter io.Writer) error
}

// TargetWorkspace captures canonical paths for a validation target so the orchestrator
// doesn't have to re-derive them on every call.
type TargetWorkspace struct {
	Name        string
	TargetKind  string
	TargetID    string
	TargetRoot  string
	Exclude     []string
	ScenarioDir string
	// CoverageDir is the target's governed test-genie artifact workspace.
	CoverageDir     string
	PhaseDir        string
	AppRoot         string
	PhysicalAppRoot string
	Mapping         Mapping

	artifactDir    string
	artifactRoot   string
	phaseCacheRoot string

	// Runtime URLs (set via SetRuntimeURLs)
	uiURL         string
	apiURL        string
	targetRuntime TargetRuntime
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
func New(scenariosRoot, scenario string) (*TargetWorkspace, error) {
	return NewWithOptions(scenariosRoot, scenario, Options{})
}

// NewTarget creates a workspace for a non-scenario repository target. The
// source directory remains the target itself, while all run artifacts are
// owned by the runtime-home target store rather than being written into the
// source tree. Such targets intentionally have no lifecycle runtime.
func NewTarget(repoRoot string, target targetmodel.Target) (*TargetWorkspace, error) {
	if target.HasRuntime() {
		return nil, fmt.Errorf("NewTarget is only for non-scenario targets")
	}
	if strings.TrimSpace(repoRoot) == "" || !filepath.IsAbs(repoRoot) {
		return nil, shared.NewValidationError("repository root must be absolute")
	}
	info, err := os.Stat(target.Path)
	if err != nil {
		return nil, fmt.Errorf("target path is not accessible: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("target path is not a directory: %s", target.Path)
	}
	mapping, err := NewMapping(target.Path, repoRoot, "", "", target.ID)
	if err != nil {
		return nil, err
	}
	artifactRoot, err := targetmodel.ArtifactRoot(repoRoot, target)
	if err != nil {
		return nil, fmt.Errorf("resolve target artifacts: %w", err)
	}
	phaseCacheRoot, err := targetmodel.PhaseCacheRoot(target)
	if err != nil {
		return nil, fmt.Errorf("resolve target phase cache: %w", err)
	}
	testDir := artifactpaths.ScenarioPath(artifactRoot, artifactpaths.CoverageRoot)
	if err := os.MkdirAll(testDir, 0o755); err != nil {
		return nil, fmt.Errorf("create target artifact workspace: %w", err)
	}
	return &TargetWorkspace{
		Name:            target.ID,
		TargetKind:      targetKindName(target.Kind),
		TargetID:        target.ID,
		TargetRoot:      target.Root,
		Exclude:         append([]string(nil), target.Exclude...),
		ScenarioDir:     target.Path,
		artifactRoot:    artifactRoot,
		phaseCacheRoot:  phaseCacheRoot,
		CoverageDir:     testDir,
		PhaseDir:        artifactpaths.ScenarioPath(artifactRoot, artifactpaths.CoverageRoot, "phases"),
		AppRoot:         repoRoot,
		PhysicalAppRoot: repoRoot,
		Mapping:         mapping,
	}, nil
}

func targetKindName(kind commonv1.ValidationTargetKind) string {
	name := strings.TrimPrefix(strings.ToLower(kind.String()), "validation_target_kind_")
	return strings.ReplaceAll(name, "_", "-")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// NewWithOptions loads and validates the file-system layout for a scenario.
func NewWithOptions(scenariosRoot, scenario string, opts Options) (*TargetWorkspace, error) {
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

	physicalAppRoot := AppRootFromScenario(scenarioDir)
	mapping, err := NewMapping(scenarioDir, physicalAppRoot, opts.LogicalRepoRoot, opts.LogicalScenarioRelPath, name)
	if err != nil {
		return nil, err
	}
	target := targetmodel.Target{Kind: commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SCENARIO, ID: name, Path: scenarioDir}
	artifactRoot, err := targetmodel.ArtifactRoot(physicalAppRoot, target)
	if err != nil {
		return nil, fmt.Errorf("resolve scenario artifacts: %w", err)
	}
	phaseCacheRoot, err := targetmodel.PhaseCacheRoot(target)
	if err != nil {
		return nil, fmt.Errorf("resolve scenario phase cache: %w", err)
	}
	testDir := artifactpaths.ScenarioPath(artifactRoot, artifactpaths.CoverageRoot)
	if err := os.MkdirAll(testDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create governed artifact directory: %w", err)
	}
	if err := EnsureDir(testDir); err != nil {
		return nil, err
	}
	phaseDir := filepath.Join(testDir, "phases")

	return &TargetWorkspace{
		Name:            name,
		TargetKind:      "scenario",
		TargetID:        name,
		TargetRoot:      filepath.ToSlash(filepath.Join("scenarios", name)),
		ScenarioDir:     scenarioDir,
		artifactRoot:    artifactRoot,
		phaseCacheRoot:  phaseCacheRoot,
		CoverageDir:     testDir,
		PhaseDir:        phaseDir,
		AppRoot:         physicalAppRoot,
		PhysicalAppRoot: physicalAppRoot,
		Mapping:         mapping,
	}, nil
}

// Environment returns the phase environment bound to this workspace.
func (w *TargetWorkspace) Environment() Environment {
	if w == nil {
		return Environment{}
	}
	return Environment{
		ScenarioName:    w.Name,
		TargetKind:      w.TargetKind,
		TargetID:        w.TargetID,
		TargetRoot:      w.TargetRoot,
		Exclude:         append([]string(nil), w.Exclude...),
		ScenarioDir:     w.ScenarioDir,
		ArtifactRoot:    firstNonEmpty(w.artifactRoot, w.ScenarioDir),
		PhaseCacheRoot:  firstNonEmpty(w.phaseCacheRoot, w.artifactRoot, w.ScenarioDir),
		CoverageDir:     w.CoverageDir,
		AppRoot:         w.AppRoot,
		PhysicalAppRoot: w.PhysicalAppRoot,
		Mapping:         w.Mapping,
		UIURL:           w.uiURL,
		APIURL:          w.apiURL,
		TargetRuntime:   w.targetRuntime,
	}
}

// SetRuntimeURLs configures the runtime service URLs for phases that need to connect
// to running services (e.g., Lighthouse audits, integration tests).
func (w *TargetWorkspace) SetRuntimeURLs(uiURL, apiURL string) {
	if w == nil {
		return
	}
	w.uiURL = uiURL
	w.apiURL = apiURL
}

// SetTargetRuntime configures the lifecycle manager used by phases that need to
// restart the scenario under test.
func (w *TargetWorkspace) SetTargetRuntime(runtime TargetRuntime) {
	if w == nil {
		return
	}
	w.targetRuntime = runtime
}

// EnsureArtifactDir lazily creates the artifact directory and returns its path.
func (w *TargetWorkspace) EnsureArtifactDir() (string, error) {
	if w == nil {
		return "", fmt.Errorf("workspace is not configured")
	}
	if w.artifactDir != "" {
		return w.artifactDir, nil
	}
	dir := filepath.Join(firstNonEmpty(w.artifactRoot, w.ScenarioDir), sharedartifacts.LogsDir)
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
