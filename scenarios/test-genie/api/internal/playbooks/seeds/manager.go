package seeds

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	// SeedsFolder is the name of the seeds folder within playbooks.
	SeedsFolder = "__seeds"
	// ApplyScript is the name of the apply script.
	ApplyScript = "apply.sh"
	// CleanupScript is the name of the cleanup script.
	CleanupScript = "cleanup.sh"
)

// Manager defines the interface for seed script management.
type Manager interface {
	// Apply runs the apply seed script and returns a cleanup function.
	Apply(ctx context.Context) (cleanup func(), err error)
	// HasSeeds returns true if the seeds directory exists with an apply script.
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
	return filepath.Join(m.testDir, "playbooks", SeedsFolder)
}

// ApplyPath returns the path to the apply script.
func (m *FileManager) ApplyPath() string {
	return filepath.Join(m.SeedsDir(), ApplyScript)
}

// CleanupPath returns the path to the cleanup script.
func (m *FileManager) CleanupPath() string {
	return filepath.Join(m.SeedsDir(), CleanupScript)
}

// HasSeeds returns true if the seeds directory exists with an apply script.
func (m *FileManager) HasSeeds() bool {
	_, err := os.Stat(m.ApplyPath())
	return err == nil
}

// Apply runs the apply seed script and returns a cleanup function.
// If no apply script exists, returns nil cleanup and no error.
func (m *FileManager) Apply(ctx context.Context) (func(), error) {
	applyPath := m.ApplyPath()
	if _, err := os.Stat(applyPath); err != nil {
		// No apply script, nothing to do
		return nil, nil
	}

	if err := m.runScript(ctx, applyPath); err != nil {
		return nil, fmt.Errorf("apply seed script failed: %w", err)
	}

	// Create cleanup function
	cleanupPath := m.CleanupPath()
	cleanup := func() {}
	if _, err := os.Stat(cleanupPath); err == nil {
		cleanup = func() {
			// Use a background context for cleanup since the original context may be cancelled
			_ = m.runScript(context.Background(), cleanupPath)
		}
	}

	return cleanup, nil
}

// runScript executes a seed script with the appropriate environment.
func (m *FileManager) runScript(ctx context.Context, scriptPath string) error {
	cmd := exec.CommandContext(ctx, "bash", scriptPath)
	cmd.Dir = m.scenarioDir
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("TEST_GENIE_SCENARIO_DIR=%s", m.scenarioDir),
		fmt.Sprintf("TEST_GENIE_APP_ROOT=%s", m.appRoot),
	)
	cmd.Stdout = m.logWriter
	cmd.Stderr = m.logWriter

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("script %s failed: %w", scriptPath, err)
	}
	return nil
}
