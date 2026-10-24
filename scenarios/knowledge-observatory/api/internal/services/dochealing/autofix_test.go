package dochealing

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"knowledge-observatory/internal/services/dochealth"
)

// memFS is an in-memory FileSystem for testing.
type memFS struct {
	files   map[string]bool
	dirs    map[string]bool
	renamed []renamePair
	mkdirs  []string
}

type renamePair struct{ from, to string }

func newMemFS() *memFS {
	return &memFS{
		files: map[string]bool{},
		dirs:  map[string]bool{},
	}
}

func (m *memFS) MkdirAll(path string, _ os.FileMode) error {
	m.mkdirs = append(m.mkdirs, path)
	m.dirs[path] = true
	return nil
}

func (m *memFS) Rename(oldpath, newpath string) error {
	m.renamed = append(m.renamed, renamePair{oldpath, newpath})
	delete(m.files, oldpath)
	m.files[newpath] = true
	return nil
}

func (m *memFS) Stat(path string) (os.FileInfo, error) {
	if m.files[path] || m.dirs[path] {
		return nil, nil
	}
	return nil, os.ErrNotExist
}

// failFS always fails rename operations.
type failFS struct {
	*memFS
}

func newFailFS() *failFS {
	return &failFS{memFS: newMemFS()}
}

func (f *failFS) Rename(_, _ string) error {
	return fmt.Errorf("disk full")
}

// setupAutoFixScenario creates a temp scenario with misplaced docs.
func setupAutoFixScenario(t *testing.T, files map[string]string) (scenariosRoot string, scenarioName string) {
	t.Helper()
	scenariosRoot = t.TempDir()
	scenarioName = "test-scenario"
	scenarioPath := filepath.Join(scenariosRoot, scenarioName)

	for relPath, content := range files {
		absPath := filepath.Join(scenarioPath, relPath)
		if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	return scenariosRoot, scenarioName
}

func newAutoFixService(t *testing.T, scenariosRoot string, fs FileSystem) *Service {
	t.Helper()
	health, err := dochealth.NewService(scenariosRoot)
	if err != nil {
		t.Fatalf("dochealth.NewService: %v", err)
	}
	svc := &Service{
		scenariosRoot: scenariosRoot,
		repoRoot:      filepath.Dir(scenariosRoot),
		health:        health,
		fs:            fs,
	}
	return svc
}

func TestAutoFix_HappyPath(t *testing.T) {
	// Create a scenario with a misplaced ARCHITECTURE.md in root instead of docs/concepts/
	scenariosRoot, name := setupAutoFixScenario(t, map[string]string{
		"ARCHITECTURE.md": "# Architecture",
		"README.md":       "# Test",
	})

	mfs := newMemFS()
	// Mark the source file as existing
	scenarioPath := filepath.Join(scenariosRoot, name)
	mfs.files[filepath.Join(scenarioPath, "ARCHITECTURE.md")] = true

	svc := newAutoFixService(t, scenariosRoot, mfs)
	result, err := svc.AutoFix(context.Background(), name, false)
	if err != nil {
		t.Fatalf("AutoFix: %v", err)
	}

	if result.ScenarioName != name {
		t.Errorf("expected scenario %s, got %s", name, result.ScenarioName)
	}
	if len(result.Moved) == 0 {
		t.Fatal("expected at least one moved file")
	}
	if result.Moved[0].DocType != "architecture" {
		t.Errorf("expected architecture doc type, got %s", result.Moved[0].DocType)
	}
	if len(result.Skipped) != 0 {
		t.Errorf("expected no skipped files, got %d", len(result.Skipped))
	}
}

func TestAutoFix_DestinationExists(t *testing.T) {
	scenariosRoot, name := setupAutoFixScenario(t, map[string]string{
		"ARCHITECTURE.md":               "# Architecture (root)",
		"docs/concepts/ARCHITECTURE.md": "# Architecture (canonical)",
		"README.md":                     "# Test",
	})

	mfs := newMemFS()
	scenarioPath := filepath.Join(scenariosRoot, name)
	// Mark both source and dest as existing
	mfs.files[filepath.Join(scenarioPath, "ARCHITECTURE.md")] = true
	mfs.files[filepath.Join(scenarioPath, "docs", "concepts", "ARCHITECTURE.md")] = true

	svc := newAutoFixService(t, scenariosRoot, mfs)
	result, err := svc.AutoFix(context.Background(), name, false)
	if err != nil {
		t.Fatalf("AutoFix: %v", err)
	}

	if len(result.Skipped) == 0 {
		t.Fatal("expected skipped file when destination exists")
	}
	if result.Skipped[0].Reason != "destination already exists" {
		t.Errorf("expected 'destination already exists', got %q", result.Skipped[0].Reason)
	}
	if len(result.Moved) != 0 {
		t.Errorf("expected no moved files, got %d", len(result.Moved))
	}
}

func TestAutoFix_NoMisplacedDocs(t *testing.T) {
	scenariosRoot, name := setupAutoFixScenario(t, map[string]string{
		"README.md":                     "# Test",
		"docs/concepts/ARCHITECTURE.md": "# Architecture",
	})

	mfs := newMemFS()
	svc := newAutoFixService(t, scenariosRoot, mfs)
	result, err := svc.AutoFix(context.Background(), name, false)
	if err != nil {
		t.Fatalf("AutoFix: %v", err)
	}

	if len(result.Moved) != 0 {
		t.Errorf("expected no moved files, got %d", len(result.Moved))
	}
	if len(result.Skipped) != 0 {
		t.Errorf("expected no skipped files, got %d", len(result.Skipped))
	}
}

func TestAutoFix_DryRun(t *testing.T) {
	scenariosRoot, name := setupAutoFixScenario(t, map[string]string{
		"ARCHITECTURE.md": "# Architecture",
		"README.md":       "# Test",
	})

	mfs := newMemFS()
	svc := newAutoFixService(t, scenariosRoot, mfs)
	result, err := svc.AutoFix(context.Background(), name, true)
	if err != nil {
		t.Fatalf("AutoFix: %v", err)
	}

	if len(result.Moved) == 0 {
		t.Fatal("expected dry-run to report moves")
	}
	// Verify no actual renames happened
	if len(mfs.renamed) != 0 {
		t.Errorf("dry run should not rename files, got %d renames", len(mfs.renamed))
	}
	if len(mfs.mkdirs) != 0 {
		t.Errorf("dry run should not create dirs, got %d mkdirs", len(mfs.mkdirs))
	}
}

func TestAutoFix_RenameFails(t *testing.T) {
	scenariosRoot, name := setupAutoFixScenario(t, map[string]string{
		"ARCHITECTURE.md": "# Architecture",
		"README.md":       "# Test",
	})

	ffs := newFailFS()
	svc := newAutoFixService(t, scenariosRoot, ffs)
	result, err := svc.AutoFix(context.Background(), name, false)
	if err != nil {
		t.Fatalf("AutoFix: %v", err)
	}

	if len(result.Skipped) == 0 {
		t.Fatal("expected skipped file on rename failure")
	}
	if result.Skipped[0].Reason == "" {
		t.Error("expected non-empty skip reason")
	}
}

func TestAutoFix_EmptyScenarioName(t *testing.T) {
	svc := &Service{health: &dochealth.Service{}}
	_, err := svc.AutoFix(context.Background(), "", false)
	if err != ErrScenarioRequired {
		t.Errorf("expected ErrScenarioRequired, got %v", err)
	}
}

func TestAutoFix_NilHealth(t *testing.T) {
	svc := &Service{}
	_, err := svc.AutoFix(context.Background(), "test", false)
	if err != ErrHealthUnavailable {
		t.Errorf("expected ErrHealthUnavailable, got %v", err)
	}
}
