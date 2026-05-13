package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"flow-verifier/internal/flows/layout"
)

func TestWriteRejectsExistingFlowDir(t *testing.T) {
	root := t.TempDir()
	parent := "ui/src/features/foo"
	flowDir := filepath.Join(root, filepath.FromSlash(parent), "flow")
	if err := os.MkdirAll(flowDir, 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := Write(Options{
		Root:      root,
		ParentDir: parent,
		FlowID:    "demo.foo.ui",
		Language:  layout.LanguageTypeScript,
	})
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected already-exists rejection, got %v", err)
	}
}

func TestWriteInfersLanguageFromUIPath(t *testing.T) {
	root := t.TempDir()
	parent := "ui/src/features/foo"
	if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(parent)), 0o755); err != nil {
		t.Fatal(err)
	}
	flowDir, err := Write(Options{
		Root:      root,
		ParentDir: parent,
		FlowID:    "demo.foo.ui",
	})
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if flowDir != parent+"/flow" {
		t.Fatalf("flowDir = %s", flowDir)
	}
	for _, name := range []string{"flow.json", "transition.ts", "fixtures.ts", "flow.test.ts"} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(flowDir), name)); err != nil {
			t.Fatalf("expected %s in scaffold: %v", name, err)
		}
	}
}

func TestWriteInfersLanguageFromAPIPath(t *testing.T) {
	root := t.TempDir()
	parent := "api/internal/foo"
	if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(parent)), 0o755); err != nil {
		t.Fatal(err)
	}
	flowDir, err := Write(Options{
		Root:      root,
		ParentDir: parent,
		FlowID:    "demo.foo.api",
	})
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	for _, name := range []string{"flow.json", "transition.go", "flow_test.go"} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(flowDir), name)); err != nil {
			t.Fatalf("expected %s in scaffold: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(flowDir), "fixtures.ts")); err == nil {
		t.Fatalf("Go scaffold should not emit fixtures.ts")
	}
}

func TestWriteRequiresThreeSegmentFlowID(t *testing.T) {
	root := t.TempDir()
	parent := "ui/src/features/foo"
	if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(parent)), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := Write(Options{
		Root:      root,
		ParentDir: parent,
		FlowID:    "twoseg.only",
	})
	if err == nil || !strings.Contains(err.Error(), "three dotted segments") {
		t.Fatalf("expected three-segment rejection, got %v", err)
	}
}
