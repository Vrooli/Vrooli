package runtime

import (
	"io"
	"testing"
)

// mockFileChecker implements FileChecker for testing.
type mockFileChecker struct {
	existingFiles map[string]bool
	globMatches   map[string]bool
}

func (m *mockFileChecker) Exists(path string) bool {
	return m.existingFiles[path]
}

func (m *mockFileChecker) GlobMatch(pattern string) bool {
	return m.globMatches[pattern]
}

func TestDetectorDetectsGo(t *testing.T) {
	scenarioDir := "/scenarios/demo"
	d := New(scenarioDir, io.Discard, WithFileChecker(&mockFileChecker{
		existingFiles: map[string]bool{
			"/scenarios/demo/api/go.mod": true,
		},
	}))

	runtimes := d.Detect()
	if len(runtimes) != 1 {
		t.Fatalf("expected 1 runtime, got %d", len(runtimes))
	}
	if runtimes[0].Command != "go" {
		t.Fatalf("expected go runtime, got %s", runtimes[0].Command)
	}
}

func TestDetectorDetectsGoFromGlobPattern(t *testing.T) {
	scenarioDir := "/scenarios/demo"
	d := New(scenarioDir, io.Discard, WithFileChecker(&mockFileChecker{
		globMatches: map[string]bool{
			"/scenarios/demo/api/*.go": true,
		},
	}))

	runtimes := d.Detect()
	if len(runtimes) != 1 {
		t.Fatalf("expected 1 runtime, got %d", len(runtimes))
	}
	if runtimes[0].Command != "go" {
		t.Fatalf("expected go runtime, got %s", runtimes[0].Command)
	}
}

func TestDetectorDetectsNode(t *testing.T) {
	scenarioDir := "/scenarios/demo"
	d := New(scenarioDir, io.Discard, WithFileChecker(&mockFileChecker{
		existingFiles: map[string]bool{
			"/scenarios/demo/ui/package.json": true,
		},
	}))

	runtimes := d.Detect()
	if len(runtimes) != 1 {
		t.Fatalf("expected 1 runtime, got %d", len(runtimes))
	}
	if runtimes[0].Command != "node" {
		t.Fatalf("expected node runtime, got %s", runtimes[0].Command)
	}
}

func TestDetectorDetectsPython(t *testing.T) {
	scenarioDir := "/scenarios/demo"
	d := New(scenarioDir, io.Discard, WithFileChecker(&mockFileChecker{
		existingFiles: map[string]bool{
			"/scenarios/demo/requirements.txt": true,
		},
	}))

	runtimes := d.Detect()
	if len(runtimes) != 1 {
		t.Fatalf("expected 1 runtime, got %d", len(runtimes))
	}
	if runtimes[0].Command != "python3" {
		t.Fatalf("expected python3 runtime, got %s", runtimes[0].Command)
	}
}

func TestDetectorDetectsMultipleRuntimes(t *testing.T) {
	scenarioDir := "/scenarios/demo"
	d := New(scenarioDir, io.Discard, WithFileChecker(&mockFileChecker{
		existingFiles: map[string]bool{
			"/scenarios/demo/api/go.mod":       true,
			"/scenarios/demo/ui/package.json":  true,
			"/scenarios/demo/requirements.txt": true,
		},
	}))

	runtimes := d.Detect()
	if len(runtimes) != 3 {
		t.Fatalf("expected 3 runtimes, got %d", len(runtimes))
	}

	commands := make(map[string]bool)
	for _, r := range runtimes {
		commands[r.Command] = true
	}

	for _, expected := range []string{"go", "node", "python3"} {
		if !commands[expected] {
			t.Fatalf("expected runtime '%s' not found", expected)
		}
	}
}

func TestDetectorNoRuntimes(t *testing.T) {
	scenarioDir := "/scenarios/demo"
	d := New(scenarioDir, io.Discard, WithFileChecker(&mockFileChecker{}))

	runtimes := d.Detect()
	if len(runtimes) != 0 {
		t.Fatalf("expected no runtimes, got %d", len(runtimes))
	}
}

func TestToCommandRequirements(t *testing.T) {
	runtimes := []Runtime{
		{Name: "Go", Command: "go", Reason: "test"},
		{Name: "Node.js", Command: "node", Reason: "test2"},
	}

	reqs := ToCommandRequirements(runtimes)
	if len(reqs) != 2 {
		t.Fatalf("expected 2 requirements, got %d", len(reqs))
	}
	if reqs[0].Name != "go" {
		t.Fatalf("expected 'go', got '%s'", reqs[0].Name)
	}
	if reqs[1].Name != "node" {
		t.Fatalf("expected 'node', got '%s'", reqs[1].Name)
	}
}
