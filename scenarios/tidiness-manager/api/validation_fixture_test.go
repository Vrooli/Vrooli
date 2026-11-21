package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeValidationTargetFixture creates a small, deterministic target with one
// source-backed long-file finding. Keeping this fixture shared lets validation
// contract tests compare API projections against the same canonical scan.
func writeValidationTargetFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "api"), 0o755); err != nil {
		t.Fatalf("mkdir fixture api: %v", err)
	}
	content := strings.Builder{}
	content.WriteString("package main\n\nfunc main() {}\n")
	for i := 0; i < 520; i++ {
		content.WriteString("// maintained source context\n")
	}
	if err := os.WriteFile(filepath.Join(root, "api", "main.go"), []byte(content.String()), 0o644); err != nil {
		t.Fatalf("write fixture source: %v", err)
	}
	return root
}
