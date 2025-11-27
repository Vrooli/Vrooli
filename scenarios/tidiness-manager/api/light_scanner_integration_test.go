package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLightScanner_Integration_WithLanguageMetrics(t *testing.T) {
	// Create a temporary scenario structure
	tmpDir := t.TempDir()

	// Create Makefile (minimal, targets won't actually work but scanner should handle gracefully)
	makefileContent := `lint:
	@echo "Linting..."

type:
	@echo "Type checking..."
`
	if err := os.WriteFile(filepath.Join(tmpDir, "Makefile"), []byte(makefileContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create api/ directory with Go files
	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatal(err)
	}

	goCode := `package main

import "fmt"

// TODO: refactor this
func main() {
	fmt.Println("hello")
}

func helper() {
	// FIXME: needs work
}
`
	if err := os.WriteFile(filepath.Join(apiDir, "main.go"), []byte(goCode), 0644); err != nil {
		t.Fatal(err)
	}

	// Create ui/src/ directory with TypeScript files
	uiSrcDir := filepath.Join(tmpDir, "ui", "src")
	if err := os.MkdirAll(uiSrcDir, 0755); err != nil {
		t.Fatal(err)
	}

	tsCode := `import React from 'react';

// TODO: add tests
export default function App() {
  return <div>Hello</div>;
}
`
	if err := os.WriteFile(filepath.Join(uiSrcDir, "App.tsx"), []byte(tsCode), 0644); err != nil {
		t.Fatal(err)
	}

	// Run light scan
	scanner := NewLightScanner(tmpDir, 30*time.Second)
	result, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Light scan failed: %v", err)
	}

	// Verify basic scan results
	if result.Scenario == "" {
		t.Error("Expected scenario name to be set")
	}
	if result.TotalFiles < 2 {
		t.Errorf("Expected at least 2 files, got %d", result.TotalFiles)
	}

	// Verify language metrics were collected
	if result.LanguageMetrics == nil {
		t.Fatal("Expected language metrics to be collected")
	}

	// Check Go metrics
	goMetrics, hasGo := result.LanguageMetrics[LanguageGo]
	if !hasGo {
		t.Error("Expected Go language to be detected")
	} else {
		if goMetrics.FileCount != 1 {
			t.Errorf("Expected 1 Go file, got %d", goMetrics.FileCount)
		}

		// Check code metrics
		if goMetrics.CodeMetrics == nil {
			t.Error("Expected Go code metrics")
		} else {
			if goMetrics.CodeMetrics.TodoCount != 1 {
				t.Errorf("Expected 1 TODO in Go code, got %d", goMetrics.CodeMetrics.TodoCount)
			}
			if goMetrics.CodeMetrics.FixmeCount != 1 {
				t.Errorf("Expected 1 FIXME in Go code, got %d", goMetrics.CodeMetrics.FixmeCount)
			}
		}

		// Complexity analysis may or may not run depending on gocyclo availability
		if goMetrics.Complexity != nil {
			if !goMetrics.Complexity.Skipped {
				// If gocyclo ran, we should have some results
				if goMetrics.Complexity.TotalFunctions < 1 {
					t.Errorf("Expected at least 1 function in complexity analysis, got %d", goMetrics.Complexity.TotalFunctions)
				}
			}
		}
	}

	// Check TypeScript metrics
	tsMetrics, hasTS := result.LanguageMetrics[LanguageTypeScript]
	if !hasTS {
		t.Error("Expected TypeScript language to be detected")
	} else {
		if tsMetrics.FileCount != 1 {
			t.Errorf("Expected 1 TypeScript file, got %d", tsMetrics.FileCount)
		}

		// Check code metrics
		if tsMetrics.CodeMetrics == nil {
			t.Error("Expected TypeScript code metrics")
		} else {
			if tsMetrics.CodeMetrics.TodoCount != 1 {
				t.Errorf("Expected 1 TODO in TypeScript code, got %d", tsMetrics.CodeMetrics.TodoCount)
			}
		}

		// Complexity should be skipped for TypeScript (not implemented)
		if tsMetrics.Complexity != nil && !tsMetrics.Complexity.Skipped {
			t.Error("Expected TypeScript complexity to be skipped")
		}
	}

	// Verify scan completed
	if result.CompletedAt.IsZero() {
		t.Error("Expected completed timestamp")
	}
	if result.Duration == 0 {
		t.Error("Expected non-zero duration")
	}
}

func TestLightScanner_Integration_NoLanguages(t *testing.T) {
	// Create empty scenario
	tmpDir := t.TempDir()

	scanner := NewLightScanner(tmpDir, 30*time.Second)
	result, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("Light scan failed: %v", err)
	}

	// Should complete without error even if no languages detected
	if result.TotalFiles != 0 {
		t.Errorf("Expected 0 files for empty scenario, got %d", result.TotalFiles)
	}

	// Language metrics should be empty or nil
	if result.LanguageMetrics != nil && len(result.LanguageMetrics) > 0 {
		t.Errorf("Expected no language metrics for empty scenario, got %d", len(result.LanguageMetrics))
	}
}
