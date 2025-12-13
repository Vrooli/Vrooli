package main

import (
	"os"
	"path/filepath"
	"testing"
)

// assertCodeMetricsTechDebt validates TODO/FIXME/HACK counts in CodeMetrics
func assertCodeMetricsTechDebt(t *testing.T, metrics *CodeMetrics, todoCount, fixmeCount, hackCount int) {
	t.Helper()
	if metrics.TodoCount != todoCount {
		t.Errorf("Expected %d TODO(s), got %d", todoCount, metrics.TodoCount)
	}
	if metrics.FixmeCount != fixmeCount {
		t.Errorf("Expected %d FIXME(s), got %d", fixmeCount, metrics.FixmeCount)
	}
	if metrics.HackCount != hackCount {
		t.Errorf("Expected %d HACK(s), got %d", hackCount, metrics.HackCount)
	}
}

// assertAvgImportsAndFunctions validates average imports and functions per file
func assertAvgImportsAndFunctions(t *testing.T, metrics *CodeMetrics, avgImports, avgFunctions float64) {
	t.Helper()
	if metrics.AvgImportsPerFile != avgImports {
		t.Errorf("Expected %.1f import(s) per file, got %f", avgImports, metrics.AvgImportsPerFile)
	}
	if metrics.AvgFunctionsPerFile != avgFunctions {
		t.Errorf("Expected %.1f function(s) per file, got %f", avgFunctions, metrics.AvgFunctionsPerFile)
	}
}

// writeCodeFile writes a test file with the given content to tmpDir/filename
func writeCodeFile(t *testing.T, tmpDir, filename, content string) {
	t.Helper()
	filePath := filepath.Join(tmpDir, filename)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// codeMetricsLanguageTestCase defines test data for language-specific code metrics analysis
type codeMetricsLanguageTestCase struct {
	name            string
	language        Language
	filename        string
	code            string
	expectedTodo    int
	expectedFixme   int
	expectedHack    int
	expectedImports float64
	minFunctions    float64
}

func TestCodeMetricsAnalyzer_AnalyzeFiles_Languages(t *testing.T) {
	testCases := []codeMetricsLanguageTestCase{
		{
			name:     "Go",
			language: LanguageGo,
			filename: "test.go",
			code: `package main

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
`,
			expectedTodo:    1,
			expectedFixme:   1,
			expectedHack:    1,
			expectedImports: 1.0, // Go import block counts as single import
			minFunctions:    2.0,
		},
		{
			name:     "TypeScript",
			language: LanguageTypeScript,
			filename: "test.tsx",
			code: `import React from 'react';
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
`,
			expectedTodo:    1,
			expectedFixme:   1,
			expectedHack:    1,
			expectedImports: 2.0,
			minFunctions:    1.0,
		},
		{
			name:     "Python",
			language: LanguagePython,
			filename: "test.py",
			code: `import os
from pathlib import Path

# TODO: add error handling
def process():
    # FIXME: validate input
    pass

def helper():
    # HACK: temporary solution
    return None
`,
			expectedTodo:    1,
			expectedFixme:   1,
			expectedHack:    1,
			expectedImports: 2.0,
			minFunctions:    2.0,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			writeCodeFile(t, tmpDir, tc.filename, tc.code)

			analyzer := NewCodeMetricsAnalyzer(tmpDir)
			metrics, err := analyzer.AnalyzeFiles([]string{tc.filename}, tc.language)
			if err != nil {
				t.Fatalf("AnalyzeFiles failed: %v", err)
			}

			assertCodeMetricsTechDebt(t, metrics, tc.expectedTodo, tc.expectedFixme, tc.expectedHack)

			if metrics.AvgImportsPerFile != tc.expectedImports {
				t.Errorf("Expected %f imports per file, got %f", tc.expectedImports, metrics.AvgImportsPerFile)
			}
			if metrics.AvgFunctionsPerFile < tc.minFunctions {
				t.Errorf("Expected at least %f functions per file, got %f", tc.minFunctions, metrics.AvgFunctionsPerFile)
			}
		})
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

	writeCodeFile(t, tmpDir, "test.go", code)

	analyzer := NewCodeMetricsAnalyzer(tmpDir)
	metrics, err := analyzer.AnalyzeFiles([]string{"test.go"}, LanguageGo)
	if err != nil {
		t.Fatalf("AnalyzeFiles failed: %v", err)
	}

	// Should detect all case variations
	assertCodeMetricsTechDebt(t, metrics, 3, 2, 2)
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

// commentDensityTestCase defines test data for comment density analysis
type commentDensityTestCase struct {
	name                  string
	language              Language
	filename              string
	code                  string
	expectedCommentLines  int
	expectedCodeLines     int
	expectedRatio         float64
	validateExactCodeLine bool
	validateExactRatio    bool
}

// Tests for comment density
func TestCodeMetricsAnalyzer_CommentDensity(t *testing.T) {
	testCases := []commentDensityTestCase{
		{
			name:     "Go",
			language: LanguageGo,
			filename: "test.go",
			code: `package main

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
`,
			expectedCommentLines:  6,
			expectedCodeLines:     6,
			expectedRatio:         1.0,
			validateExactCodeLine: true,
			validateExactRatio:    true,
		},
		{
			name:     "TypeScript",
			language: LanguageTypeScript,
			filename: "test.tsx",
			code: `// File header comment
import React from 'react';

/**
 * Component documentation
 * Multi-line JSDoc
 */
export function Component() {
  // Implementation comment
  return <div>Hello</div>;
}
`,
			expectedCommentLines:  6,
			expectedCodeLines:     0, // Will validate > 0
			expectedRatio:         0, // Will validate > 0
			validateExactCodeLine: false,
			validateExactRatio:    false,
		},
		{
			name:     "Python",
			language: LanguagePython,
			filename: "test.py",
			code: `# File header
import os

"""
Module documentation
Multi-line docstring
"""

def process():
    # Implementation comment
    pass
`,
			expectedCommentLines:  6,
			expectedCodeLines:     0, // Will validate > 0
			expectedRatio:         0, // Will validate > 0
			validateExactCodeLine: false,
			validateExactRatio:    false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			filePath := filepath.Join(tmpDir, tc.filename)
			if err := os.WriteFile(filePath, []byte(tc.code), 0644); err != nil {
				t.Fatal(err)
			}

			analyzer := NewCodeMetricsAnalyzer(tmpDir)
			metrics, err := analyzer.AnalyzeFiles([]string{tc.filename}, tc.language)
			if err != nil {
				t.Fatalf("AnalyzeFiles failed: %v", err)
			}

			if metrics.TotalCommentLines != tc.expectedCommentLines {
				t.Errorf("Expected %d comment lines, got %d", tc.expectedCommentLines, metrics.TotalCommentLines)
			}

			if tc.validateExactCodeLine {
				if metrics.TotalCodeLines != tc.expectedCodeLines {
					t.Errorf("Expected %d code lines, got %d", tc.expectedCodeLines, metrics.TotalCodeLines)
				}
			} else {
				if metrics.TotalCodeLines == 0 {
					t.Errorf("Expected code lines > 0, got %d", metrics.TotalCodeLines)
				}
			}

			if tc.validateExactRatio {
				if metrics.CommentToCodeRatio != tc.expectedRatio {
					t.Errorf("Expected ratio %f, got %f", tc.expectedRatio, metrics.CommentToCodeRatio)
				}
			} else {
				if metrics.CommentToCodeRatio <= 0 {
					t.Errorf("Expected positive comment ratio, got %f", metrics.CommentToCodeRatio)
				}
			}
		})
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
