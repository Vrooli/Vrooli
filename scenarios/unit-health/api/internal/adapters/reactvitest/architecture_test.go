package reactvitest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestArchitectureReportsEveryStructuralDriftAndRecognizesSharedHelper(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "tests"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Join(root, "src", "a.test.tsx"), filepath.Join(root, "src", "b.spec.ts"), filepath.Join(root, "tests", "c.test.tsx")} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("import { render } from '@testing-library/react'\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	seen := map[string]bool{}
	for _, drift := range AnalyzeArchitecture(root) {
		seen[drift.Code] = true
	}
	if !seen["TEST_UTIL_MISSING"] || !seen["TEST_NOT_COLOCATED"] {
		t.Fatalf("drifts = %+v", seen)
	}
	utils := filepath.Join(root, "src", "test-utils")
	if err := os.MkdirAll(utils, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(utils, "index.ts"), []byte("export { renderWithProviders } from '@vrooli/api-base/testing'\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !HasSharedRenderHelper("import { renderWithProviders } from '@vrooli/api-base/testing'") {
		t.Fatal("shared helper not recognized")
	}
	if got := formatCount(3); got != "3" {
		t.Fatalf("formatCount = %q", got)
	}
}
