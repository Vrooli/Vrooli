package seeds

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	// SeedsFolder is the name of the seeds folder within playbooks.
	SeedsFolder = "seeds"
	// GoEntrypoint is the preferred seed entrypoint (executed via go run).
	GoEntrypoint = "seed.go"
	// ShellEntrypoint is the legacy-compatible seed entrypoint (bash).
	ShellEntrypoint = "seed.sh"
)

// Manager defines the interface for seed script management.
type Manager interface {
	// Apply runs the seed entrypoint. Cleanup is a no-op placeholder for compatibility.
	Apply(ctx context.Context) (cleanup func(), err error)
	// HasSeeds returns true if the seeds directory exists with a supported entrypoint.
	HasSeeds() bool
}

// FileManager manages seed scripts from the filesystem.
type FileManager struct {
	scenarioDir string
	appRoot     string
	testDir     string
	logWriter   io.Writer
}

// NewManager creates a new seed manager.
func NewManager(scenarioDir, appRoot, testDir string, logWriter io.Writer) *FileManager {
	if logWriter == nil {
		logWriter = io.Discard
	}
	return &FileManager{
		scenarioDir: scenarioDir,
		appRoot:     appRoot,
		testDir:     testDir,
		logWriter:   logWriter,
	}
}

// SeedsDir returns the path to the seeds directory.
func (m *FileManager) SeedsDir() string {
	return filepath.Join(m.scenarioDir, "bas", "seeds")
}

// HasSeeds returns true if the seeds directory exists with a supported entrypoint.
func (m *FileManager) HasSeeds() bool {
	args, _ := m.entrypoint()
	return len(args) > 0
}

// Apply runs the seed entrypoint and returns a no-op cleanup function.
// If no entrypoint exists, returns nil cleanup and no error.
func (m *FileManager) Apply(ctx context.Context) (func(), error) {
	args, _ := m.entrypoint()
	if len(args) == 0 {
		// No apply script, nothing to do
		return nil, nil
	}

	if err := m.runScript(ctx, args[0], args[1:]...); err != nil {
		return nil, fmt.Errorf("seed execution failed: %w", err)
	}

	return func() {}, nil
}

// runScript executes a seed script with the appropriate environment.
func (m *FileManager) runScript(ctx context.Context, command string, args ...string) error {
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = m.scenarioDir
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("TEST_GENIE_SCENARIO_DIR=%s", m.scenarioDir),
		fmt.Sprintf("TEST_GENIE_APP_ROOT=%s", m.appRoot),
		"TEST_GENIE_SEEDS=1",
	)
	cmd.Stdout = m.logWriter
	cmd.Stderr = m.logWriter

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("seed command %s %s failed: %w", command, strings.Join(args, " "), err)
	}
	return nil
}

// entrypoint returns the command and path for the best-available seed runner.
func (m *FileManager) entrypoint() ([]string, string) {
	seedsDir := m.SeedsDir()
	goPath := filepath.Join(seedsDir, GoEntrypoint)
	if _, err := os.Stat(goPath); err == nil {
		return []string{"go", "run", goPath}, goPath
	}

	shellPath := filepath.Join(seedsDir, ShellEntrypoint)
	if _, err := os.Stat(shellPath); err == nil {
		return []string{"bash", shellPath}, shellPath
	}
	return nil, ""
}
