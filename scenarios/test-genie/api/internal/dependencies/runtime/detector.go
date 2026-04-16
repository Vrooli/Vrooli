package runtime

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"

	"test-genie/internal/dependencies/commands"
)

// Runtime represents a detected language runtime requirement.
type Runtime struct {
	Name    string // Runtime name (e.g., "Go", "Node.js", "Python")
	Command string // Command to check in PATH (e.g., "go", "node", "python3")
	Reason  string // Why it's required
}

// Detector identifies which language runtimes are needed for a scenario.
type Detector interface {
	// Detect returns the list of required runtimes based on scenario files.
	Detect() []Runtime
}

// FileChecker abstracts file existence checks for testing.
type FileChecker interface {
	// Exists returns true if the file exists and is not a directory.
	Exists(path string) bool

	// ReadFile returns the file contents.
	ReadFile(path string) ([]byte, error)

	// GlobMatch returns true if any files match the pattern.
	GlobMatch(pattern string) bool
}

// detector is the default implementation of Detector.
type detector struct {
	scenarioDir string
	fileChecker FileChecker
	logWriter   io.Writer
}

// New creates a new runtime detector.
func New(scenarioDir string, logWriter io.Writer, opts ...Option) Detector {
	d := &detector{
		scenarioDir: scenarioDir,
		fileChecker: &osFileChecker{},
		logWriter:   logWriter,
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

// Option configures a detector.
type Option func(*detector)

// WithFileChecker sets a custom file checker (for testing).
func WithFileChecker(fc FileChecker) Option {
	return func(d *detector) {
		d.fileChecker = fc
	}
}

// Detect implements Detector.
func (d *detector) Detect() []Runtime {
	var runtimes []Runtime

	if d.hasGo() {
		runtimes = append(runtimes, Runtime{
			Name:    "Go",
			Command: "go",
			Reason:  "Go runtime required to compile or test scenario code",
		})
	}

	if d.hasNode() {
		runtimes = append(runtimes, Runtime{
			Name:    "Node.js",
			Command: "node",
			Reason:  "Node.js runtime required to build or test UI code",
		})
	}

	if d.hasPython() {
		runtimes = append(runtimes, Runtime{
			Name:    "Python",
			Command: "python3",
			Reason:  "Python runtime required to run scenario scripts or tests",
		})
	}

	return runtimes
}

// hasGo checks for Go project indicators.
func (d *detector) hasGo() bool {
	candidates := []string{filepath.Join(d.scenarioDir, "api", "go.mod")}
	for _, path := range candidates {
		if d.fileChecker.Exists(path) {
			return true
		}
	}

	patterns := []string{filepath.Join(d.scenarioDir, "api", "*.go")}
	for _, pattern := range patterns {
		if d.fileChecker.GlobMatch(pattern) {
			return true
		}
	}

	if d.hasGoModuleCLI() {
		return true
	}

	return false
}

type serviceManifest struct {
	CLI *serviceCLIConfig `json:"cli"`
}

type serviceCLIConfig struct {
	Enabled bool              `json:"enabled"`
	Adapter serviceCLIAdapter `json:"adapter"`
}

type serviceCLIAdapter struct {
	Kind      string `json:"kind"`
	ModuleDir string `json:"module_dir"`
}

func (d *detector) hasGoModuleCLI() bool {
	data, err := d.fileChecker.ReadFile(filepath.Join(d.scenarioDir, ".vrooli", "service.json"))
	if err != nil {
		return false
	}

	var manifest serviceManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return false
	}
	if manifest.CLI == nil || !manifest.CLI.Enabled {
		return false
	}
	if strings.TrimSpace(manifest.CLI.Adapter.Kind) != "go_module" {
		return false
	}

	moduleDir := strings.TrimSpace(manifest.CLI.Adapter.ModuleDir)
	if moduleDir == "" {
		return false
	}
	moduleRoot := filepath.Join(d.scenarioDir, filepath.FromSlash(moduleDir))
	if d.fileChecker.Exists(filepath.Join(moduleRoot, "go.mod")) {
		return true
	}
	return d.fileChecker.GlobMatch(filepath.Join(moduleRoot, "*.go"))
}

// hasNode checks for Node.js project indicators.
func (d *detector) hasNode() bool {
	candidates := []string{
		filepath.Join(d.scenarioDir, "package.json"),
		filepath.Join(d.scenarioDir, "ui", "package.json"),
	}
	for _, path := range candidates {
		if d.fileChecker.Exists(path) {
			return true
		}
	}
	return false
}

// hasPython checks for Python project indicators.
func (d *detector) hasPython() bool {
	candidates := []string{
		filepath.Join(d.scenarioDir, "requirements.txt"),
		filepath.Join(d.scenarioDir, "pyproject.toml"),
	}
	for _, path := range candidates {
		if d.fileChecker.Exists(path) {
			return true
		}
	}
	return false
}

// osFileChecker is the default FileChecker using os package.
type osFileChecker struct{}

func (c *osFileChecker) Exists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func (c *osFileChecker) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (c *osFileChecker) GlobMatch(pattern string) bool {
	matches, _ := filepath.Glob(pattern)
	return len(matches) > 0
}

// ToCommandRequirements converts runtimes to command requirements.
func ToCommandRequirements(runtimes []Runtime) []commands.CommandRequirement {
	reqs := make([]commands.CommandRequirement, len(runtimes))
	for i, r := range runtimes {
		reqs[i] = commands.CommandRequirement{
			Name:   r.Command,
			Reason: r.Reason,
		}
	}
	return reqs
}
