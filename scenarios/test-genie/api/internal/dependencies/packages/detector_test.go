package packages

import (
	"io"
	"testing"
)

// mockFileChecker implements FileChecker for testing.
type mockFileChecker struct {
	existingFiles map[string]bool
}

func (m *mockFileChecker) Exists(path string) bool {
	return m.existingFiles[path]
}

func TestDetectorDetectsPnpm(t *testing.T) {
	scenarioDir := "/scenarios/demo"
	d := New(scenarioDir, io.Discard, WithFileChecker(&mockFileChecker{
		existingFiles: map[string]bool{
			"/scenarios/demo/pnpm-lock.yaml": true,
		},
	}))

	managers := d.Detect()
	if len(managers) != 1 {
		t.Fatalf("expected 1 manager, got %d", len(managers))
	}
	if managers[0].Name != "pnpm" {
		t.Fatalf("expected pnpm, got %s", managers[0].Name)
	}
}

func TestDetectorDetectsNpm(t *testing.T) {
	scenarioDir := "/scenarios/demo"
	d := New(scenarioDir, io.Discard, WithFileChecker(&mockFileChecker{
		existingFiles: map[string]bool{
			"/scenarios/demo/package-lock.json": true,
		},
	}))

	managers := d.Detect()
	if len(managers) != 1 {
		t.Fatalf("expected 1 manager, got %d", len(managers))
	}
	if managers[0].Name != "npm" {
		t.Fatalf("expected npm, got %s", managers[0].Name)
	}
}

func TestDetectorDetectsYarn(t *testing.T) {
	scenarioDir := "/scenarios/demo"
	d := New(scenarioDir, io.Discard, WithFileChecker(&mockFileChecker{
		existingFiles: map[string]bool{
			"/scenarios/demo/yarn.lock": true,
		},
	}))

	managers := d.Detect()
	if len(managers) != 1 {
		t.Fatalf("expected 1 manager, got %d", len(managers))
	}
	if managers[0].Name != "yarn" {
		t.Fatalf("expected yarn, got %s", managers[0].Name)
	}
}

func TestDetectorDetectsUILockfile(t *testing.T) {
	scenarioDir := "/scenarios/demo"
	d := New(scenarioDir, io.Discard, WithFileChecker(&mockFileChecker{
		existingFiles: map[string]bool{
			"/scenarios/demo/ui/pnpm-lock.yaml": true,
		},
	}))

	managers := d.Detect()
	if len(managers) != 1 {
		t.Fatalf("expected 1 manager, got %d", len(managers))
	}
	if managers[0].Name != "pnpm" {
		t.Fatalf("expected pnpm, got %s", managers[0].Name)
	}
}

func TestDetectorDeduplicatesManagers(t *testing.T) {
	scenarioDir := "/scenarios/demo"
	d := New(scenarioDir, io.Discard, WithFileChecker(&mockFileChecker{
		existingFiles: map[string]bool{
			"/scenarios/demo/pnpm-lock.yaml":    true,
			"/scenarios/demo/ui/pnpm-lock.yaml": true,
		},
	}))

	managers := d.Detect()
	if len(managers) != 1 {
		t.Fatalf("expected 1 manager (deduplicated), got %d", len(managers))
	}
}

func TestDetectorDefaultsToPnpmWithNodeWorkspace(t *testing.T) {
	scenarioDir := "/scenarios/demo"
	d := New(scenarioDir, io.Discard, WithFileChecker(&mockFileChecker{
		existingFiles: map[string]bool{
			"/scenarios/demo/ui/package.json": true,
		},
	}))

	managers := d.Detect()
	if len(managers) != 1 {
		t.Fatalf("expected 1 manager (default pnpm), got %d", len(managers))
	}
	if managers[0].Name != "pnpm" {
		t.Fatalf("expected pnpm as default, got %s", managers[0].Name)
	}
}

func TestDetectorNoManagersWithoutNodeWorkspace(t *testing.T) {
	scenarioDir := "/scenarios/demo"
	d := New(scenarioDir, io.Discard, WithFileChecker(&mockFileChecker{}))

	managers := d.Detect()
	if len(managers) != 0 {
		t.Fatalf("expected no managers, got %d", len(managers))
	}
}

func TestHasNodeWorkspace(t *testing.T) {
	scenarioDir := "/scenarios/demo"

	tests := []struct {
		name     string
		files    map[string]bool
		expected bool
	}{
		{
			name:     "root package.json",
			files:    map[string]bool{"/scenarios/demo/package.json": true},
			expected: true,
		},
		{
			name:     "ui package.json",
			files:    map[string]bool{"/scenarios/demo/ui/package.json": true},
			expected: true,
		},
		{
			name:     "no package.json",
			files:    map[string]bool{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New(scenarioDir, io.Discard, WithFileChecker(&mockFileChecker{
				existingFiles: tt.files,
			}))

			got := d.HasNodeWorkspace()
			if got != tt.expected {
				t.Fatalf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestToCommandRequirements(t *testing.T) {
	managers := []Manager{
		{Name: "pnpm", Reason: "test"},
		{Name: "npm", Reason: "test2"},
	}

	reqs := ToCommandRequirements(managers)
	if len(reqs) != 2 {
		t.Fatalf("expected 2 requirements, got %d", len(reqs))
	}
	if reqs[0].Name != "pnpm" {
		t.Fatalf("expected 'pnpm', got '%s'", reqs[0].Name)
	}
	if reqs[1].Name != "npm" {
		t.Fatalf("expected 'npm', got '%s'", reqs[1].Name)
	}
}
