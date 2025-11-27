package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestComplexityAnalyzer_AnalyzeComplexity_Go_ToolNotInstalled(t *testing.T) {
	tmpDir := t.TempDir()

	analyzer := NewComplexityAnalyzer(tmpDir, 30*time.Second)

	// If gocyclo is not installed, should return skipped result
	result, err := analyzer.AnalyzeComplexity(context.Background(), LanguageGo, []string{"dummy.go"})
	if err != nil {
		t.Fatalf("AnalyzeComplexity failed: %v", err)
	}

	// Check if gocyclo is available
	if !commandExists("gocyclo") {
		if !result.Skipped {
			t.Error("Expected result to be skipped when gocyclo not installed")
		}
		if result.SkipReason == "" {
			t.Error("Expected skip reason when gocyclo not installed")
		}
	}
}

func TestComplexityAnalyzer_AnalyzeComplexity_Go_WithGocyclo(t *testing.T) {
	// Skip if gocyclo not installed
	if !commandExists("gocyclo") {
		t.Skip("gocyclo not installed, skipping test")
	}

	tmpDir := t.TempDir()

	// Create a simple Go file with low complexity
	simpleCode := `package main

func simple() {
	println("hello")
}

func add(a, b int) int {
	return a + b
}
`

	simpleFile := filepath.Join(tmpDir, "simple.go")
	if err := os.WriteFile(simpleFile, []byte(simpleCode), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a complex Go file with high cyclomatic complexity
	complexCode := `package main

func complex(n int) int {
	result := 0
	if n > 0 {
		if n > 10 {
			if n > 20 {
				result = 1
			} else if n > 15 {
				result = 2
			} else {
				result = 3
			}
		} else if n > 5 {
			if n == 7 {
				result = 4
			} else {
				result = 5
			}
		} else {
			result = 6
		}
	} else if n < 0 {
		if n < -10 {
			result = 7
		} else {
			result = 8
		}
	}
	return result
}
`

	complexFile := filepath.Join(tmpDir, "complex.go")
	if err := os.WriteFile(complexFile, []byte(complexCode), 0644); err != nil {
		t.Fatal(err)
	}

	analyzer := NewComplexityAnalyzer(tmpDir, 30*time.Second)
	result, err := analyzer.AnalyzeComplexity(context.Background(), LanguageGo, []string{"simple.go", "complex.go"})
	if err != nil {
		t.Fatalf("AnalyzeComplexity failed: %v", err)
	}

	// Should not be skipped
	if result.Skipped {
		t.Errorf("Expected analysis to run, got skipped: %s", result.SkipReason)
	}

	// Should have detected functions
	if result.TotalFunctions < 2 {
		t.Errorf("Expected at least 2 functions, got %d", result.TotalFunctions)
	}

	// Should have reasonably high complexity function (at least 8)
	if result.MaxComplexity < 8 {
		t.Errorf("Expected max complexity >= 8 for complex function, got %d", result.MaxComplexity)
	}

	// Average complexity should be reasonable
	if result.AverageComplexity < 1.0 {
		t.Errorf("Expected average complexity > 1, got %f", result.AverageComplexity)
	}

	// Tool should be gocyclo
	if result.Tool != "gocyclo" {
		t.Errorf("Expected tool 'gocyclo', got '%s'", result.Tool)
	}
}

func TestComplexityAnalyzer_AnalyzeComplexity_TypeScript_NotImplemented(t *testing.T) {
	tmpDir := t.TempDir()

	analyzer := NewComplexityAnalyzer(tmpDir, 30*time.Second)
	result, err := analyzer.AnalyzeComplexity(context.Background(), LanguageTypeScript, []string{"test.ts"})
	if err != nil {
		t.Fatalf("AnalyzeComplexity failed: %v", err)
	}

	// TypeScript complexity should be skipped for now
	if !result.Skipped {
		t.Error("Expected TypeScript complexity to be skipped")
	}
	if result.SkipReason == "" {
		t.Error("Expected skip reason for TypeScript")
	}
}

func TestComplexityAnalyzer_EmptyFileList(t *testing.T) {
	tmpDir := t.TempDir()

	analyzer := NewComplexityAnalyzer(tmpDir, 30*time.Second)
	result, err := analyzer.AnalyzeComplexity(context.Background(), LanguageGo, []string{})
	if err != nil {
		t.Fatalf("AnalyzeComplexity failed: %v", err)
	}

	// Should skip if no files to analyze
	if !result.Skipped {
		t.Error("Expected analysis to be skipped for empty file list")
	}
}

func TestCommandExists(t *testing.T) {
	// Test with a command that should exist
	if !commandExists("go") {
		t.Error("Expected 'go' command to exist")
	}

	// Test with a command that shouldn't exist
	if commandExists("nonexistent-command-12345") {
		t.Error("Expected 'nonexistent-command-12345' to not exist")
	}
}
