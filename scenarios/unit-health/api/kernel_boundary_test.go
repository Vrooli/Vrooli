package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestFrameworkNeutralKernelHasNoAdapterVocabulary protects the dependency
// boundary described by the v2 contract. Adapter-specific terms belong in the
// adapter registry or adapter-owned validation; executor, evidence, and
// readiness must remain reusable across ecosystems.
func TestFrameworkNeutralKernelHasNoAdapterVocabulary(t *testing.T) {
	forbidden := []string{"Vitest", "React Testing Library", "pytest", "Cargo", "Bats", "Pester"}
	for _, dir := range []string{"internal/executor", "internal/evidence", "internal/readiness", "internal/validation"} {
		err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
				return nil
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			for _, term := range forbidden {
				if strings.Contains(string(raw), term) {
					t.Errorf("framework term %q found in kernel file %s", term, path)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("scan %s: %v", dir, err)
		}
	}
}
