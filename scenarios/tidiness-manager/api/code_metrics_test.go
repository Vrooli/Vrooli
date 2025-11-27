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
