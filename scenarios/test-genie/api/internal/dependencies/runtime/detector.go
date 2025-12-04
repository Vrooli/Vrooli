package runtime

import (
	"io"
	"os"
	"path/filepath"

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
	candidates := []string{
		filepath.Join(d.scenarioDir, "api", "go.mod"),
		filepath.Join(d.scenarioDir, "cli", "go.mod"),
	}
	for _, path := range candidates {
		if d.fileChecker.Exists(path) {
			return true
		}
	}

	// Check for loose .go files
	patterns := []string{
		filepath.Join(d.scenarioDir, "api", "*.go"),
		filepath.Join(d.scenarioDir, "cli", "*.go"),
	}
	for _, pattern := range patterns {
		if d.fileChecker.GlobMatch(pattern) {
			return true
		}
	}

	return false
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
