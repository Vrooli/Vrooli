package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCodeMetricsAnalyzer_AnalyzeFiles_Go(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a Go file with various markers
	goCode := `package main

import (
	"fmt"
	"strings"
)

// TODO: refactor this function
func processData() {
	// FIXME: handle edge cases
	fmt.Println("processing")
	// HACK: temporary workaround
}

func helper() {
	// another function
}
`

	filePath := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(filePath, []byte(goCode), 0644); err != nil {
		t.Fatal(err)
	}

	analyzer := NewCodeMetricsAnalyzer(tmpDir)
	metrics, err := analyzer.AnalyzeFiles([]string{"test.go"}, LanguageGo)
	if err != nil {
		t.Fatalf("AnalyzeFiles failed: %v", err)
	}

	// Check TODO/FIXME/HACK counts
	if metrics.TodoCount != 1 {
		t.Errorf("Expected 1 TODO, got %d", metrics.TodoCount)
	}
	if metrics.FixmeCount != 1 {
		t.Errorf("Expected 1 FIXME, got %d", metrics.FixmeCount)
	}
	if metrics.HackCount != 1 {
		t.Errorf("Expected 1 HACK, got %d", metrics.HackCount)
	}

	// Check import count (1 import block with 2 packages)
	// Note: Go's import block counts as a single import line
	if metrics.AvgImportsPerFile != 1.0 {
		t.Errorf("Expected 1 import per file, got %f", metrics.AvgImportsPerFile)
	}

	// Check function count (2 functions: processData, helper)
	if metrics.AvgFunctionsPerFile != 2.0 {
		t.Errorf("Expected 2 functions per file, got %f", metrics.AvgFunctionsPerFile)
	}
}

func TestCodeMetricsAnalyzer_AnalyzeFiles_TypeScript(t *testing.T) {
	tmpDir := t.TempDir()

	tsCode := `import React from 'react';
import { useState } from 'react';

// TODO: add prop validation
export default function Component() {
  // FIXME: optimize rendering
  const [state, setState] = useState(0);
  return <div>{state}</div>;
}

const helper = () => {
  // HACK: workaround for browser bug
  return 42;
};
`

	filePath := filepath.Join(tmpDir, "test.tsx")
	if err := os.WriteFile(filePath, []byte(tsCode), 0644); err != nil {
		t.Fatal(err)
	}

	analyzer := NewCodeMetricsAnalyzer(tmpDir)
	metrics, err := analyzer.AnalyzeFiles([]string{"test.tsx"}, LanguageTypeScript)
	if err != nil {
		t.Fatalf("AnalyzeFiles failed: %v", err)
	}

	// Check markers
	if metrics.TodoCount != 1 {
		t.Errorf("Expected 1 TODO, got %d", metrics.TodoCount)
	}
	if metrics.FixmeCount != 1 {
		t.Errorf("Expected 1 FIXME, got %d", metrics.FixmeCount)
	}
	if metrics.HackCount != 1 {
		t.Errorf("Expected 1 HACK, got %d", metrics.HackCount)
	}

	// Check import count (2 imports)
	if metrics.AvgImportsPerFile != 2.0 {
		t.Errorf("Expected 2 imports per file, got %f", metrics.AvgImportsPerFile)
	}

	// Check function count (should detect both the component and helper)
	if metrics.AvgFunctionsPerFile < 1.0 {
		t.Errorf("Expected at least 1 function per file, got %f", metrics.AvgFunctionsPerFile)
	}
}

func TestCodeMetricsAnalyzer_AnalyzeFiles_Python(t *testing.T) {
	tmpDir := t.TempDir()

	pyCode := `import os
from pathlib import Path

# TODO: add error handling
def process():
    # FIXME: validate input
    pass

def helper():
    # HACK: temporary solution
    return None
`

	filePath := filepath.Join(tmpDir, "test.py")
	if err := os.WriteFile(filePath, []byte(pyCode), 0644); err != nil {
		t.Fatal(err)
	}

	analyzer := NewCodeMetricsAnalyzer(tmpDir)
	metrics, err := analyzer.AnalyzeFiles([]string{"test.py"}, LanguagePython)
	if err != nil {
		t.Fatalf("AnalyzeFiles failed: %v", err)
	}

	// Check markers
	if metrics.TodoCount != 1 {
		t.Errorf("Expected 1 TODO, got %d", metrics.TodoCount)
	}
	if metrics.FixmeCount != 1 {
		t.Errorf("Expected 1 FIXME, got %d", metrics.FixmeCount)
	}
	if metrics.HackCount != 1 {
		t.Errorf("Expected 1 HACK, got %d", metrics.HackCount)
	}

	// Check import count (2 import lines)
	if metrics.AvgImportsPerFile != 2.0 {
		t.Errorf("Expected 2 imports per file, got %f", metrics.AvgImportsPerFile)
	}

	// Check function count (2 functions)
	if metrics.AvgFunctionsPerFile != 2.0 {
		t.Errorf("Expected 2 functions per file, got %f", metrics.AvgFunctionsPerFile)
	}
}

func TestCodeMetricsAnalyzer_CaseInsensitiveMarkers(t *testing.T) {
	tmpDir := t.TempDir()

	code := `package main

// todo: lowercase
// TODO: uppercase
// Todo: mixed case
// FIXME: fix this
// fixme: also this
// hack: workaround
// HACK: another workaround

func test() {}
`

	filePath := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(filePath, []byte(code), 0644); err != nil {
		t.Fatal(err)
	}

	analyzer := NewCodeMetricsAnalyzer(tmpDir)
	metrics, err := analyzer.AnalyzeFiles([]string{"test.go"}, LanguageGo)
	if err != nil {
		t.Fatalf("AnalyzeFiles failed: %v", err)
	}

	// Should detect all case variations
	if metrics.TodoCount != 3 {
		t.Errorf("Expected 3 TODOs (case-insensitive), got %d", metrics.TodoCount)
	}
	if metrics.FixmeCount != 2 {
		t.Errorf("Expected 2 FIXMEs (case-insensitive), got %d", metrics.FixmeCount)
	}
	if metrics.HackCount != 2 {
		t.Errorf("Expected 2 HACKs (case-insensitive), got %d", metrics.HackCount)
	}
}

func TestCodeMetricsAnalyzer_EmptyFiles(t *testing.T) {
	tmpDir := t.TempDir()

	analyzer := NewCodeMetricsAnalyzer(tmpDir)
	metrics, err := analyzer.AnalyzeFiles([]string{}, LanguageGo)
	if err != nil {
		t.Fatalf("AnalyzeFiles failed: %v", err)
	}

	// Should return zero metrics for empty file list
	if metrics.TodoCount != 0 {
		t.Errorf("Expected 0 TODOs for empty list, got %d", metrics.TodoCount)
	}
	if metrics.AvgImportsPerFile != 0 {
		t.Errorf("Expected 0 imports per file for empty list, got %f", metrics.AvgImportsPerFile)
	}
}

// Tests for comment density
func TestCodeMetricsAnalyzer_CommentDensity_Go(t *testing.T) {
	tmpDir := t.TempDir()

	goCode := `package main

// Single line comment
import "fmt"

/*
 * Multi-line comment
 * across several lines
 */
func main() {
	x := 5 // inline comment (counts as code)
	// Another single-line comment
	fmt.Println(x)
}
`

	filePath := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(filePath, []byte(goCode), 0644); err != nil {
		t.Fatal(err)
	}

	analyzer := NewCodeMetricsAnalyzer(tmpDir)
	metrics, err := analyzer.AnalyzeFiles([]string{"test.go"}, LanguageGo)
	if err != nil {
		t.Fatalf("AnalyzeFiles failed: %v", err)
	}

	// Verify comment lines: 1 (single) + 4 (multi-line: /*, 2 middle, */) + 1 (another single) = 6
	if metrics.TotalCommentLines != 6 {
		t.Errorf("Expected 6 comment lines, got %d", metrics.TotalCommentLines)
	}

	// Verify code lines (non-empty, non-comment lines)
	// package main, import, func main(), {, x := 5, fmt.Println, } = 6 lines
	// Note: x := 5 // inline counts as code (line starts with code)
	if metrics.TotalCodeLines != 6 {
		t.Errorf("Expected 6 code lines, got %d", metrics.TotalCodeLines)
	}

	// Verify ratio (6 comments / 6 code = 1.0)
	if metrics.CommentToCodeRatio != 1.0 {
		t.Errorf("Expected ratio 1.0, got %f", metrics.CommentToCodeRatio)
	}
}

func TestCodeMetricsAnalyzer_CommentDensity_TypeScript(t *testing.T) {
	tmpDir := t.TempDir()

	tsCode := `// File header comment
import React from 'react';

/**
 * Component documentation
 * Multi-line JSDoc
 */
export function Component() {
  // Implementation comment
  return <div>Hello</div>;
}
`

	filePath := filepath.Join(tmpDir, "test.tsx")
	if err := os.WriteFile(filePath, []byte(tsCode), 0644); err != nil {
		t.Fatal(err)
	}

	analyzer := NewCodeMetricsAnalyzer(tmpDir)
	metrics, err := analyzer.AnalyzeFiles([]string{"test.tsx"}, LanguageTypeScript)
	if err != nil {
		t.Fatalf("AnalyzeFiles failed: %v", err)
	}

	// Comment lines: 1 (header) + 4 (JSDoc: /**, 2 middle, */) + 1 (implementation) = 6
	if metrics.TotalCommentLines != 6 {
		t.Errorf("Expected 6 comment lines, got %d", metrics.TotalCommentLines)
	}

	// Code lines should be > 0
	if metrics.TotalCodeLines == 0 {
		t.Errorf("Expected code lines > 0, got %d", metrics.TotalCodeLines)
	}

	// Ratio should be reasonable
	if metrics.CommentToCodeRatio <= 0 {
		t.Errorf("Expected positive comment ratio, got %f", metrics.CommentToCodeRatio)
	}
}

func TestCodeMetricsAnalyzer_CommentDensity_Python(t *testing.T) {
	tmpDir := t.TempDir()

	pyCode := `# File header
import os

"""
Module documentation
Multi-line docstring
"""

def process():
    # Implementation comment
    pass
`

	filePath := filepath.Join(tmpDir, "test.py")
	if err := os.WriteFile(filePath, []byte(pyCode), 0644); err != nil {
		t.Fatal(err)
	}

	analyzer := NewCodeMetricsAnalyzer(tmpDir)
	metrics, err := analyzer.AnalyzeFiles([]string{"test.py"}, LanguagePython)
	if err != nil {
		t.Fatalf("AnalyzeFiles failed: %v", err)
	}

	// Comment lines: 1 (header) + 4 (docstring: """, 2 middle, """) + 1 (implementation) = 6
	if metrics.TotalCommentLines != 6 {
		t.Errorf("Expected 6 comment lines, got %d", metrics.TotalCommentLines)
	}

	if metrics.TotalCodeLines == 0 {
		t.Errorf("Expected code lines > 0, got %d", metrics.TotalCodeLines)
	}
}

// Tests for test coverage indicators
func TestCodeMetricsAnalyzer_TestCoverage_Go(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source file
	sourceCode := `package main

func Add(a, b int) int {
	return a + b
}
`
	sourcePath := filepath.Join(tmpDir, "math.go")
	if err := os.WriteFile(sourcePath, []byte(sourceCode), 0644); err != nil {
		t.Fatal(err)
	}

	// Create test file
	testCode := `package main

import "testing"

func TestAdd(t *testing.T) {
	if Add(1, 2) != 3 {
		t.Error("Add failed")
	}
}
`
	testPath := filepath.Join(tmpDir, "math_test.go")
	if err := os.WriteFile(testPath, []byte(testCode), 0644); err != nil {
		t.Fatal(err)
	}

	analyzer := NewCodeMetricsAnalyzer(tmpDir)
	metrics, err := analyzer.AnalyzeFiles([]string{"math.go", "math_test.go"}, LanguageGo)
	if err != nil {
		t.Fatalf("AnalyzeFiles failed: %v", err)
	}

	// Should detect that math.go has a test
	if metrics.FilesWithTests != 1 {
		t.Errorf("Expected 1 file with tests, got %d", metrics.FilesWithTests)
	}
	if metrics.FilesWithoutTests != 0 {
		t.Errorf("Expected 0 files without tests, got %d", metrics.FilesWithoutTests)
	}
	if metrics.TestCoverageRatio != 1.0 {
		t.Errorf("Expected 100%% test coverage, got %f", metrics.TestCoverageRatio)
	}
}

func TestCodeMetricsAnalyzer_TestCoverage_TypeScript(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source file
	sourceCode := `export function add(a: number, b: number): number {
  return a + b;
}
`
	sourcePath := filepath.Join(tmpDir, "math.ts")
	if err := os.WriteFile(sourcePath, []byte(sourceCode), 0644); err != nil {
		t.Fatal(err)
	}

	// Create test file (same directory, .test.ts pattern)
	testCode := `import { add } from './math';

test('add function', () => {
  expect(add(1, 2)).toBe(3);
});
`
	testPath := filepath.Join(tmpDir, "math.test.ts")
	if err := os.WriteFile(testPath, []byte(testCode), 0644); err != nil {
		t.Fatal(err)
	}

	analyzer := NewCodeMetricsAnalyzer(tmpDir)
	metrics, err := analyzer.AnalyzeFiles([]string{"math.ts", "math.test.ts"}, LanguageTypeScript)
	if err != nil {
		t.Fatalf("AnalyzeFiles failed: %v", err)
	}

	if metrics.FilesWithTests != 1 {
		t.Errorf("Expected 1 file with tests, got %d", metrics.FilesWithTests)
	}
	if metrics.TestCoverageRatio != 1.0 {
		t.Errorf("Expected 100%% test coverage, got %f", metrics.TestCoverageRatio)
	}
}

func TestCodeMetricsAnalyzer_TestCoverage_TypeScript_TestsDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source file
	sourceCode := `export function multiply(a: number, b: number): number {
  return a * b;
}
`
	sourcePath := filepath.Join(tmpDir, "math.ts")
	if err := os.WriteFile(sourcePath, []byte(sourceCode), 0644); err != nil {
		t.Fatal(err)
	}

	// Create __tests__ directory
	testsDir := filepath.Join(tmpDir, "__tests__")
	if err := os.MkdirAll(testsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create test file in __tests__ directory
	testCode := `import { multiply } from '../math';

test('multiply function', () => {
  expect(multiply(2, 3)).toBe(6);
});
`
	testPath := filepath.Join(testsDir, "math.test.ts")
	if err := os.WriteFile(testPath, []byte(testCode), 0644); err != nil {
		t.Fatal(err)
	}

	analyzer := NewCodeMetricsAnalyzer(tmpDir)
	metrics, err := analyzer.AnalyzeFiles([]string{"math.ts", "__tests__/math.test.ts"}, LanguageTypeScript)
	if err != nil {
		t.Fatalf("AnalyzeFiles failed: %v", err)
	}

	if metrics.FilesWithTests != 1 {
		t.Errorf("Expected 1 file with tests (__tests__ pattern), got %d", metrics.FilesWithTests)
	}
}

func TestCodeMetricsAnalyzer_TestCoverage_NoTests(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source file without test
	sourceCode := `package main

func Untested() {
	// No test for this
}
`
	sourcePath := filepath.Join(tmpDir, "untested.go")
	if err := os.WriteFile(sourcePath, []byte(sourceCode), 0644); err != nil {
		t.Fatal(err)
	}

	analyzer := NewCodeMetricsAnalyzer(tmpDir)
	metrics, err := analyzer.AnalyzeFiles([]string{"untested.go"}, LanguageGo)
	if err != nil {
		t.Fatalf("AnalyzeFiles failed: %v", err)
	}

	if metrics.FilesWithTests != 0 {
		t.Errorf("Expected 0 files with tests, got %d", metrics.FilesWithTests)
	}
	if metrics.FilesWithoutTests != 1 {
		t.Errorf("Expected 1 file without tests, got %d", metrics.FilesWithoutTests)
	}
	if metrics.TestCoverageRatio != 0.0 {
		t.Errorf("Expected 0%% test coverage, got %f", metrics.TestCoverageRatio)
	}
}

func TestCodeMetricsAnalyzer_TestCoverage_MixedCoverage(t *testing.T) {
	tmpDir := t.TempDir()

	// Create 3 source files
	files := []struct {
		name    string
		hasTest bool
	}{
		{"file1.go", true},
		{"file2.go", false},
		{"file3.go", true},
	}

	var fileList []string
	for _, f := range files {
		sourcePath := filepath.Join(tmpDir, f.name)
		if err := os.WriteFile(sourcePath, []byte("package main\nfunc dummy() {}\n"), 0644); err != nil {
			t.Fatal(err)
		}
		fileList = append(fileList, f.name)

		if f.hasTest {
			testName := f.name[:len(f.name)-3] + "_test.go"
			testPath := filepath.Join(tmpDir, testName)
			if err := os.WriteFile(testPath, []byte("package main\nfunc TestDummy(t *testing.T) {}\n"), 0644); err != nil {
				t.Fatal(err)
			}
			fileList = append(fileList, testName)
		}
	}

	analyzer := NewCodeMetricsAnalyzer(tmpDir)
	metrics, err := analyzer.AnalyzeFiles(fileList, LanguageGo)
	if err != nil {
		t.Fatalf("AnalyzeFiles failed: %v", err)
	}

	// 2 out of 3 files have tests
	if metrics.FilesWithTests != 2 {
		t.Errorf("Expected 2 files with tests, got %d", metrics.FilesWithTests)
	}
	if metrics.FilesWithoutTests != 1 {
		t.Errorf("Expected 1 file without tests, got %d", metrics.FilesWithoutTests)
	}

	// Test coverage should be ~0.67 (2 out of 3 files)
	if metrics.TestCoverageRatio < 0.66 || metrics.TestCoverageRatio > 0.67 {
		t.Errorf("Expected test coverage ~0.67, got %f", metrics.TestCoverageRatio)
	}
}
