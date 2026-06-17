package validation

import (
	"path/filepath"
	"testing"
)

func TestAnalyzeTSArchitectureTestUtilMissing(t *testing.T) {
	root := t.TempDir()
	// Three co-located component tests under src/, but no src/test-utils module.
	for _, n := range []string{"A", "B", "C"} {
		writeFile(t, filepath.Join(root, "src", n+".test.tsx"), "import {render} from '@testing-library/react';\nit('x',()=>{expect(1).toBe(1)});\n")
	}
	ws := Workspace{ID: "ui", Language: "typescript", RootPath: root}
	findings := analyzeArchitecture("demo", []Workspace{ws}, fixedNowStr)
	if _, ok := findingByCode(findings, codeTestUtilMissing); !ok {
		t.Errorf("expected TEST_UTIL_MISSING for a UI surface with tests but no test-utils, got %v", codes(findings))
	}
}

func TestAnalyzeTSArchitectureNotColocated(t *testing.T) {
	root := t.TempDir()
	// A test outside src/ in a __tests__ dir is not co-located.
	writeFile(t, filepath.Join(root, "__tests__", "App.test.tsx"), "it('x',()=>{expect(1).toBe(1)});\n")
	ws := Workspace{ID: "ui", Language: "typescript", RootPath: root}
	findings := analyzeArchitecture("demo", []Workspace{ws}, fixedNowStr)
	if _, ok := findingByCode(findings, codeTestNotColocated); !ok {
		t.Errorf("expected TEST_NOT_COLOCATED for a test outside src/, got %v", codes(findings))
	}
}

func TestAnalyzeTSArchitectureCleanWithTestUtils(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "src", "test-utils", "render.tsx"), "export const x = 1;\n")
	writeFile(t, filepath.Join(root, "src", "A.test.tsx"), "import {x} from './test-utils/render';\nit('x',()=>{expect(x).toBe(1)});\n")
	ws := Workspace{ID: "ui", Language: "typescript", RootPath: root}
	findings := analyzeArchitecture("demo", []Workspace{ws}, fixedNowStr)
	if _, ok := findingByCode(findings, codeTestUtilMissing); ok {
		t.Errorf("a UI surface with src/test-utils must not be flagged TEST_UTIL_MISSING, got %v", codes(findings))
	}
}
