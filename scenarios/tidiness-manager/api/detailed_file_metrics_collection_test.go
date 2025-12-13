package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// metricsAssertion represents a single metrics field assertion
type metricsAssertion struct {
	name     string
	actual   interface{}
	expected interface{}
}

// assertMetricsFields validates multiple metrics fields in a single call
func assertMetricsFields(t *testing.T, m DetailedFileMetrics, assertions []metricsAssertion) {
	t.Helper()
	for _, a := range assertions {
		switch v := a.actual.(type) {
		case string:
			if v != a.expected.(string) {
				t.Errorf("Expected %s %q, got %q", a.name, a.expected, v)
			}
		case int:
			if v != a.expected.(int) {
				t.Errorf("Expected %s %d, got %d", a.name, a.expected, v)
			}
		case float64:
			if v != a.expected.(float64) {
				t.Errorf("Expected %s %f, got %f", a.name, a.expected, v)
			}
		case bool:
			if v != a.expected.(bool) {
				t.Errorf("Expected %s %v, got %v", a.name, a.expected, v)
			}
		}
	}
}

// assertTechDebtCounts validates TODO/FIXME/HACK counts
func assertTechDebtCounts(t *testing.T, m DetailedFileMetrics, todoCount, fixmeCount, hackCount int) {
	t.Helper()
	assertMetricsFields(t, m, []metricsAssertion{
		{"TodoCount", m.TodoCount, todoCount},
		{"FixmeCount", m.FixmeCount, fixmeCount},
		{"HackCount", m.HackCount, hackCount},
	})
}

// assertLanguageAndExtension validates language and file extension fields
func assertLanguageAndExtension(t *testing.T, m DetailedFileMetrics, language, extension string) {
	t.Helper()
	// Handle language aliases (javascript vs js)
	languageMatch := m.Language == language
	if language == "javascript" && m.Language == "js" {
		languageMatch = true
	}
	if !languageMatch {
		t.Errorf("Expected language %q, got %q", language, m.Language)
	}
	if m.FileExtension != extension {
		t.Errorf("Expected extension %q, got %q", extension, m.FileExtension)
	}
}

// languageTestCase defines test data for language-specific file metrics
type languageTestCase struct {
	name             string
	language         string
	extension        string
	dirPath          string
	filename         string
	code             string
	testFile         string
	expectedTodo     int
	expectedFixme    int
	expectedHack     int
	minFunctionCount int
	expectTestFile   bool
}

// [REQ:TM-LS-005] Test CollectDetailedFileMetrics with multiple programming languages
func TestCollectDetailedFileMetrics_Languages(t *testing.T) {
	testCases := []languageTestCase{
		{
			name:      "Go",
			language:  "go",
			extension: ".go",
			dirPath:   "api",
			filename:  "main.go",
			code: `package main

import (
	"fmt"
	"strings"
)

// TODO: refactor this
func processData() {
	// FIXME: handle edge cases
	fmt.Println("processing")
	// HACK: temporary workaround
	if true {
		return
	}
}

func helper() string {
	return "test"
}
`,
			testFile: `package main

import "testing"

func TestProcessData(t *testing.T) {
	processData()
}
`,
			expectedTodo:     1,
			expectedFixme:    1,
			expectedHack:     1,
			minFunctionCount: 2,
			expectTestFile:   true,
		},
		{
			name:      "TypeScript",
			language:  "typescript",
			extension: ".tsx",
			dirPath:   "ui/src",
			filename:  "Component.tsx",
			code: `import React from 'react';
import { useState } from 'react';

// TODO: add prop validation
export function Component() {
  // FIXME: optimize rendering
  const [state, setState] = useState(0);
  return <div>{state}</div>;
}

const helper = () => {
  // HACK: workaround
  return 42;
};
`,
			expectedTodo:     1,
			expectedFixme:    1,
			expectedHack:     1,
			minFunctionCount: 1,
		},
		{
			name:      "JavaScript",
			language:  "javascript",
			extension: ".js",
			dirPath:   "ui/src",
			filename:  "Component.js",
			code: `import React from 'react';

// TODO: add propTypes
export function MyComponent({ name }) {
  // FIXME: validate name prop
  const handleClick = () => {
    console.log(name);
  };

  // HACK: temporary styling
  return <div onClick={handleClick}>{name}</div>;
}

const util = (x) => x * 2;
`,
			expectedTodo:     1,
			expectedFixme:    1,
			expectedHack:     1,
			minFunctionCount: 2,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			targetDir := filepath.Join(tmpDir, tc.dirPath)
			if err := os.MkdirAll(targetDir, 0755); err != nil {
				t.Fatal(err)
			}

			mainFile := filepath.Join(targetDir, tc.filename)
			writeTestFile(t, mainFile, tc.code)

			files := []string{filepath.Join(tc.dirPath, tc.filename)}
			expectedMetricCount := 1

			if tc.testFile != "" {
				testFileName := tc.filename[:len(tc.filename)-len(tc.extension)] + "_test" + tc.extension
				writeTestFile(t, filepath.Join(targetDir, testFileName), tc.testFile)
				files = append(files, filepath.Join(tc.dirPath, testFileName))
				expectedMetricCount = 2
			}

			metrics := collectAndValidateMetrics(t, tmpDir, files, expectedMetricCount)
			mainMetrics := requireMetricsByPath(t, metrics, files[0])

			assertLanguageAndExtension(t, *mainMetrics, tc.language, tc.extension)
			assertTechDebtCounts(t, *mainMetrics, tc.expectedTodo, tc.expectedFixme, tc.expectedHack)

			if mainMetrics.LineCount == 0 {
				t.Error("Expected non-zero line count")
			}
			if mainMetrics.ImportCount == 0 {
				t.Error("Expected non-zero import count")
			}
			if mainMetrics.FunctionCount < tc.minFunctionCount {
				t.Errorf("Expected at least %d functions, got %d", tc.minFunctionCount, mainMetrics.FunctionCount)
			}
			if tc.expectTestFile && !mainMetrics.HasTestFile {
				t.Error("Expected HasTestFile to be true")
			}
		})
	}
}

// [REQ:TM-LS-005] Test CollectDetailedFileMetrics with unknown language
func TestCollectDetailedFileMetrics_UnknownLanguage(t *testing.T) {
	tmpDir := t.TempDir()

	writeTestFile(t, filepath.Join(tmpDir, "script.sh"), "#!/bin/bash\necho 'test'\n")

	files := []string{"script.sh"}
	metrics, err := CollectDetailedFileMetrics(tmpDir, files)
	if err != nil {
		t.Fatalf("CollectDetailedFileMetrics should handle unknown languages: %v", err)
	}

	if len(metrics) != 1 {
		t.Fatalf("Expected 1 metric for unknown file, got %d", len(metrics))
	}

	m := metrics[0]
	assertLanguageAndExtension(t, m, "unknown", ".sh")
	assertTechDebtCounts(t, m, 0, 0, 0)

	if m.LineCount == 0 {
		t.Error("Expected non-zero line count even for unknown language")
	}
}

// [REQ:TM-LS-005] Test CollectDetailedFileMetrics with JSON files
func TestCollectDetailedFileMetrics_JSONFiles(t *testing.T) {
	tmpDir := t.TempDir()

	configDir := filepath.Join(tmpDir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	jsonContent := `{
  "name": "test-project",
  "version": "1.0.0",
  "dependencies": {
    "react": "^18.0.0"
  }
}
`

	writeTestFile(t, filepath.Join(configDir, "package.json"), jsonContent)

	files := []string{"config/package.json"}
	metrics, err := CollectDetailedFileMetrics(tmpDir, files)
	if err != nil {
		t.Fatalf("CollectDetailedFileMetrics should handle JSON files: %v", err)
	}

	if len(metrics) == 0 {
		t.Log("JSON files may be skipped (not a programming language)")
		return
	}

	m := metrics[0]
	if m.FileExtension != ".json" {
		t.Errorf("Expected extension '.json', got %q", m.FileExtension)
	}

	if m.LineCount == 0 {
		t.Error("Expected non-zero line count for JSON file")
	}
}

// [REQ:TM-LS-005] Test CollectDetailedFileMetrics with empty file list
func TestCollectDetailedFileMetrics_EmptyList(t *testing.T) {
	tmpDir := t.TempDir()

	metrics, err := CollectDetailedFileMetrics(tmpDir, []string{})
	if err != nil {
		t.Fatalf("CollectDetailedFileMetrics should handle empty list: %v", err)
	}

	if len(metrics) != 0 {
		t.Errorf("Expected 0 metrics for empty file list, got %d", len(metrics))
	}
}

// [REQ:TM-LS-005] Test CollectDetailedFileMetrics with non-existent files
func TestCollectDetailedFileMetrics_NonExistentFiles(t *testing.T) {
	tmpDir := t.TempDir()

	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatal(err)
	}

	files := []string{"api/nonexistent.go"}
	metrics, err := CollectDetailedFileMetrics(tmpDir, files)

	if err != nil {
		t.Logf("CollectDetailedFileMetrics returned error for non-existent file: %v", err)
		return
	}

	t.Logf("CollectDetailedFileMetrics handled non-existent file gracefully, returned %d metrics", len(metrics))
}

// [REQ:TM-LS-005] Test CollectDetailedFileMetrics with mixed files
func TestCollectDetailedFileMetrics_MixedLanguages(t *testing.T) {
	tmpDir := t.TempDir()

	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatal(err)
	}

	writeTestFile(t, filepath.Join(apiDir, "main.go"), "package main\n\nfunc main() {}\n")

	uiDir := filepath.Join(tmpDir, "ui", "src")
	if err := os.MkdirAll(uiDir, 0755); err != nil {
		t.Fatal(err)
	}

	writeTestFile(t, filepath.Join(uiDir, "util.ts"), "export function test() {\n  return 42;\n}\n")

	files := []string{"api/main.go", "ui/src/util.ts"}
	metrics, err := CollectDetailedFileMetrics(tmpDir, files)
	if err != nil {
		t.Fatalf("CollectDetailedFileMetrics failed: %v", err)
	}

	if len(metrics) != 2 {
		t.Fatalf("Expected 2 metrics, got %d", len(metrics))
	}

	languages := make(map[string]bool)
	for _, m := range metrics {
		languages[m.Language] = true
	}

	if !languages["go"] {
		t.Error("Expected to find Go language")
	}

	if !languages["typescript"] {
		t.Error("Expected to find TypeScript language")
	}
}

// [REQ:TM-LS-005] Test CollectDetailedFileMetrics comment ratio calculation
func TestCollectDetailedFileMetrics_CommentRatio(t *testing.T) {
	testCases := []struct {
		name               string
		filename           string
		code               string
		expectComments     bool
		expectNonZeroRatio bool
	}{
		{
			name:     "WithComments",
			filename: "main.go",
			code: `package main

// This is a comment
// Another comment
func main() {
	// Inline comment
	println("hello")
}
`,
			expectComments:     true,
			expectNonZeroRatio: true,
		},
		{
			name:     "NoComments",
			filename: "nocomment.go",
			code: `package main

func main() {
	println("hello")
}
`,
			expectComments:     false,
			expectNonZeroRatio: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tmpDir, apiDir := setupTestDir(t, "api")
			writeTestFile(t, filepath.Join(apiDir, tc.filename), tc.code)

			files := []string{"api/" + tc.filename}
			metrics := collectAndValidateMetrics(t, tmpDir, files, 1)
			m := metrics[0]

			if m.CodeLines == 0 {
				t.Error("Expected non-zero code lines")
			}

			if tc.expectComments {
				if m.CommentLines == 0 {
					t.Error("Expected non-zero comment lines")
				}
				if m.CommentRatio == 0 {
					t.Error("Expected non-zero comment ratio")
				}
				expectedRatio := float64(m.CommentLines) / float64(m.CodeLines)
				if m.CommentRatio != expectedRatio {
					t.Errorf("Expected comment ratio %f, got %f", expectedRatio, m.CommentRatio)
				}
			} else {
				if m.CommentLines != 0 {
					t.Errorf("Expected 0 comment lines, got %d", m.CommentLines)
				}
				if m.CommentRatio != 0 {
					t.Errorf("Expected 0 comment ratio, got %f", m.CommentRatio)
				}
			}
		})
	}
}

// [REQ:TM-LS-005] Test CollectDetailedFileMetrics handles test file detection
func TestCollectDetailedFileMetrics_TestFileDetection(t *testing.T) {
	tmpDir, apiDir := setupTestDir(t, "api")

	writeTestFile(t, filepath.Join(apiDir, "handler.go"), "package main\n\nfunc process() {}\n")
	writeTestFile(t, filepath.Join(apiDir, "handler_test.go"), "package main\n\nimport \"testing\"\n\nfunc TestProcess(t *testing.T) {}\n")
	writeTestFile(t, filepath.Join(apiDir, "orphan.go"), "package main\n\nfunc orphan() {}\n")

	files := []string{"api/handler.go", "api/handler_test.go", "api/orphan.go"}
	metrics := collectAndValidateMetrics(t, tmpDir, files, 3)

	metricMap := make(map[string]DetailedFileMetrics)
	for _, m := range metrics {
		metricMap[m.FilePath] = m
	}

	if handlerMetrics, ok := metricMap["api/handler.go"]; ok {
		if !handlerMetrics.HasTestFile {
			t.Error("handler.go should have HasTestFile=true")
		}
	}

	if testMetrics, ok := metricMap["api/handler_test.go"]; ok {
		if !testMetrics.HasTestFile {
			t.Error("handler_test.go should have HasTestFile=true (it is a test)")
		}
	}

	if orphanMetrics, ok := metricMap["api/orphan.go"]; ok {
		if orphanMetrics.HasTestFile {
			t.Error("orphan.go should have HasTestFile=false")
		}
	}
}

// [REQ:TM-LS-005] Test CollectDetailedFileMetrics with binary/non-text files
func TestCollectDetailedFileMetrics_BinaryFiles(t *testing.T) {
	tmpDir := t.TempDir()

	binaryData := []byte{0xFF, 0xFE, 0xFD, 0x00, 0x01, 0x02}
	writeTestFile(t, filepath.Join(tmpDir, "binary.dat"), string(binaryData))

	files := []string{"binary.dat"}
	metrics, err := CollectDetailedFileMetrics(tmpDir, files)

	if err != nil {
		t.Logf("CollectDetailedFileMetrics handled binary file with error: %v", err)
		return
	}

	t.Logf("CollectDetailedFileMetrics returned %d metrics for binary file", len(metrics))
}

// [REQ:TM-LS-005] Test CollectDetailedFileMetrics with very large files
func TestCollectDetailedFileMetrics_LargeFile(t *testing.T) {
	tmpDir, apiDir := setupTestDir(t, "api")

	var largeCode string
	largeCode += "package main\n\n"
	for i := 0; i < 1000; i++ {
		largeCode += "// Comment line\n"
		largeCode += "func dummy" + string(rune('A'+i%26)) + "() {}\n"
	}

	writeTestFile(t, filepath.Join(apiDir, "large.go"), largeCode)

	files := []string{"api/large.go"}
	metrics := collectAndValidateMetrics(t, tmpDir, files, 1)
	m := metrics[0]

	if m.LineCount < 2000 {
		t.Errorf("Expected at least 2000 lines, got %d", m.LineCount)
	}

	if m.FunctionCount == 0 {
		t.Error("Expected non-zero function count for large file")
	}
}

// [REQ:TM-LS-005] Test CollectDetailedFileMetrics with empty files
func TestCollectDetailedFileMetrics_EmptyFiles(t *testing.T) {
	tmpDir, apiDir := setupTestDir(t, "api")

	writeTestFile(t, filepath.Join(apiDir, "empty.go"), "")

	files := []string{"api/empty.go"}
	metrics, err := CollectDetailedFileMetrics(tmpDir, files)
	if err != nil {
		t.Fatalf("CollectDetailedFileMetrics should handle empty files: %v", err)
	}

	t.Logf("CollectDetailedFileMetrics returned %d metrics for empty file", len(metrics))

	if len(metrics) > 0 {
		m := metrics[0]
		if m.LineCount != 0 {
			t.Logf("Empty file reported line count: %d", m.LineCount)
		}
	}
}

// [REQ:TM-LS-005] Test CollectDetailedFileMetrics with special characters in filenames
func TestCollectDetailedFileMetrics_SpecialCharFilenames(t *testing.T) {
	tmpDir, apiDir := setupTestDir(t, "api")

	writeTestFile(t, filepath.Join(apiDir, "my-handler_v2.go"), "package main\n\nfunc process() {}\n")

	files := []string{"api/my-handler_v2.go"}
	metrics := collectAndValidateMetrics(t, tmpDir, files, 1)
	m := metrics[0]

	if m.FilePath != "api/my-handler_v2.go" {
		t.Errorf("Expected path to be preserved, got %q", m.FilePath)
	}
}

// [REQ:TM-LS-005] Test CollectDetailedFileMetrics with nested directories
func TestCollectDetailedFileMetrics_NestedDirs(t *testing.T) {
	tmpDir := t.TempDir()

	nestedDir := filepath.Join(tmpDir, "api", "internal", "handlers")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatal(err)
	}

	writeTestFile(t, filepath.Join(nestedDir, "user.go"), "package handlers\n\nfunc handle() {}\n")

	files := []string{"api/internal/handlers/user.go"}
	metrics := collectAndValidateMetrics(t, tmpDir, files, 1)
	m := metrics[0]

	if m.FilePath != "api/internal/handlers/user.go" {
		t.Errorf("Expected full relative path, got %q", m.FilePath)
	}
}

// [REQ:TM-LS-005] Test CollectDetailedFileMetrics with multiple TODO markers
func TestCollectDetailedFileMetrics_MultipleTodoMarkers(t *testing.T) {
	tmpDir, apiDir := setupTestDir(t, "api")

	goCode := `package main

// TODO: implement feature A
func featureA() {}

// FIXME: bug in feature B
// HACK: temporary fix
func featureB() {
	// TODO: refactor this
	// FIXME: handle error
}

// TODO: add tests
`

	writeTestFile(t, filepath.Join(apiDir, "todos.go"), goCode)

	files := []string{"api/todos.go"}
	metrics := collectAndValidateMetrics(t, tmpDir, files, 1)
	m := metrics[0]

	assertTechDebtCounts(t, m, 3, 2, 1)
}

// [REQ:TM-LS-005] Test CollectDetailedFileMetrics with all zero values file
func TestCollectDetailedFileMetrics_AllZeroValues(t *testing.T) {
	tmpDir, apiDir := setupTestDir(t, "api")

	writeTestFile(t, filepath.Join(apiDir, "minimal.go"), "package main\n")

	files := []string{"api/minimal.go"}
	metrics := collectAndValidateMetrics(t, tmpDir, files, 1)
	m := metrics[0]

	assertTechDebtCounts(t, m, 0, 0, 0)

	if m.FunctionCount != 0 {
		t.Errorf("Expected 0 functions, got %d", m.FunctionCount)
	}
	if m.ImportCount != 0 {
		t.Errorf("Expected 0 imports, got %d", m.ImportCount)
	}
}

// [REQ:TM-LS-005] Test CollectDetailedFileMetrics with exact count verification
func TestCollectDetailedFileMetrics_ExactCounts(t *testing.T) {
	tmpDir, apiDir := setupTestDir(t, "api")

	goCode := `package main

import "fmt"
import "strings"

// Function 1
func first() {
	fmt.Println("first")
}

// Function 2
func second() {
	strings.ToUpper("test")
}
`

	writeTestFile(t, filepath.Join(apiDir, "exact.go"), goCode)

	files := []string{"api/exact.go"}
	metrics := collectAndValidateMetrics(t, tmpDir, files, 1)
	m := metrics[0]

	if m.ImportCount != 2 {
		t.Errorf("Expected exactly 2 imports (fmt, strings), got %d", m.ImportCount)
	}

	if m.FunctionCount != 2 {
		t.Errorf("Expected exactly 2 functions (first, second), got %d", m.FunctionCount)
	}

	if m.CommentLines != 2 {
		t.Errorf("Expected exactly 2 comment lines, got %d", m.CommentLines)
	}

	if m.LineCount < 10 || m.LineCount > 16 {
		t.Errorf("Expected line count between 10-16, got %d", m.LineCount)
	}
}

// [REQ:TM-LS-005] Test CollectDetailedFileMetrics handles path normalization
func TestCollectDetailedFileMetrics_PathNormalization(t *testing.T) {
	tmpDir, apiDir := setupTestDir(t, "api")

	writeTestFile(t, filepath.Join(apiDir, "handler.go"), "package main\n\nfunc test() {}\n")

	testPaths := []string{
		"api/handler.go",
		"./api/handler.go",
		"api/../api/handler.go",
	}

	for _, path := range testPaths {
		metrics, err := CollectDetailedFileMetrics(tmpDir, []string{path})
		if err != nil {
			t.Errorf("CollectDetailedFileMetrics failed for path %q: %v", path, err)
			continue
		}

		if len(metrics) == 0 {
			t.Errorf("No metrics returned for path %q", path)
			continue
		}

		if metrics[0].FilePath == "" {
			t.Errorf("Empty FilePath for input %q", path)
		}
	}
}

// [REQ:TM-LS-005] Test CollectDetailedFileMetrics concurrent execution safety
func TestCollectDetailedFileMetrics_Concurrent(t *testing.T) {
	tmpDir, apiDir := setupTestDir(t, "api")

	fileCount := 10
	var files []string
	for i := 0; i < fileCount; i++ {
		filename := fmt.Sprintf("file%d.go", i)
		goCode := fmt.Sprintf("package main\n\nfunc test%d() {}\n", i)
		writeTestFile(t, filepath.Join(apiDir, filename), goCode)
		files = append(files, "api/"+filename)
	}

	done := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func(goroutineID int) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Goroutine %d panicked: %v", goroutineID, r)
				}
				done <- true
			}()

			metrics, err := CollectDetailedFileMetrics(tmpDir, files)
			if err != nil {
				t.Errorf("Goroutine %d: CollectDetailedFileMetrics failed: %v", goroutineID, err)
				return
			}

			if len(metrics) != fileCount {
				t.Errorf("Goroutine %d: Expected %d metrics, got %d", goroutineID, fileCount, len(metrics))
			}
		}(i)
	}

	for i := 0; i < 5; i++ {
		<-done
	}
}
