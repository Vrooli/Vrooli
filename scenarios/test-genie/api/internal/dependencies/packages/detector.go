package packages

import (
	"io"
	"os"
	"path/filepath"

	"test-genie/internal/dependencies/commands"
)

// Manager represents a detected package manager requirement.
type Manager struct {
	Name   string // Manager name (e.g., "pnpm", "npm", "yarn")
	Reason string // Why it's required
}

// Detector identifies which package managers are needed for a scenario.
type Detector interface {
	// Detect returns the list of required package managers based on lockfiles.
	Detect() []Manager

	// HasNodeWorkspace returns true if a Node.js workspace is detected.
	HasNodeWorkspace() bool
}

// FileChecker abstracts file existence checks for testing.
type FileChecker interface {
	// Exists returns true if the file exists and is not a directory.
	Exists(path string) bool
}

// detector is the default implementation of Detector.
type detector struct {
	scenarioDir string
	fileChecker FileChecker
	logWriter   io.Writer
}

// New creates a new package manager detector.
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

// lockfileMapping defines the mapping from lockfile paths to package managers.
type lockfileMapping struct {
	Path    string
	Manager string
}

// Detect implements Detector.
func (d *detector) Detect() []Manager {
	lockfiles := []lockfileMapping{
		{filepath.Join(d.scenarioDir, "pnpm-lock.yaml"), "pnpm"},
		{filepath.Join(d.scenarioDir, "ui", "pnpm-lock.yaml"), "pnpm"},
		{filepath.Join(d.scenarioDir, "package-lock.json"), "npm"},
		{filepath.Join(d.scenarioDir, "ui", "package-lock.json"), "npm"},
		{filepath.Join(d.scenarioDir, "yarn.lock"), "yarn"},
		{filepath.Join(d.scenarioDir, "ui", "yarn.lock"), "yarn"},
	}

	var managers []Manager
	seen := make(map[string]bool)

	for _, lf := range lockfiles {
		if d.fileChecker.Exists(lf.Path) && !seen[lf.Manager] {
			seen[lf.Manager] = true
			managers = append(managers, Manager{
				Name:   lf.Manager,
				Reason: "package manager required to install JavaScript dependencies",
			})
		}
	}

	// Default to pnpm if Node workspace exists but no lockfile found
	if len(managers) == 0 && d.HasNodeWorkspace() {
		managers = append(managers, Manager{
			Name:   "pnpm",
			Reason: "package manager required to install JavaScript dependencies (defaulting to pnpm)",
		})
	}

	return managers
}

// HasNodeWorkspace implements Detector.
func (d *detector) HasNodeWorkspace() bool {
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

// osFileChecker is the default FileChecker using os package.
type osFileChecker struct{}

func (c *osFileChecker) Exists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// ToCommandRequirements converts managers to command requirements.
func ToCommandRequirements(managers []Manager) []commands.CommandRequirement {
	reqs := make([]commands.CommandRequirement, len(managers))
	for i, m := range managers {
		reqs[i] = commands.CommandRequirement{
			Name:   m.Name,
			Reason: m.Reason,
		}
	}
	return reqs
}
