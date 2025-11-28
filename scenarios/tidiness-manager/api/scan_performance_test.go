package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// [REQ:TM-LS-007] Test light scan performance for small scenarios (<50 files)
func TestScanPerformance_SmallScenario(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a small scenario with <50 files
	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatalf("failed to create api directory: %v", err)
	}

	// Create 30 test files with unique names and realistic content
	for i := 0; i < 30; i++ {
		var testFile string
		if i < 26 {
			testFile = filepath.Join(apiDir, "file_"+string(rune('a'+i))+".go")
		} else {
			testFile = filepath.Join(apiDir, "file_"+string(rune('a'+(i-26)))+string(rune('a'))+".go")
		}
		content := `package main

import "fmt"

func test() {
  x := 1
  y := 2
  z := x + y
  fmt.Println(z)
  return
}

func helper() string {
  return "test"
}
`
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
	}

	// Create simple Makefile with fast targets
	makefileContent := `
.PHONY: lint
lint:
	@echo "lint complete"

.PHONY: type
type:
	@echo "type complete"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "Makefile"), []byte(makefileContent), 0644); err != nil {
		t.Fatalf("failed to create Makefile: %v", err)
	}

	scanner := NewLightScanner(tmpDir, 60*time.Second)
	ctx := context.Background()

	start := time.Now()
	result, err := scanner.Scan(ctx)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	// Verify it completes in under 60 seconds (requirement threshold)
	if duration > 60*time.Second {
		t.Errorf("small scenario scan took %v, expected <60s (requirement violation)", duration)
	}

	// Verify file count
	if len(result.FileMetrics) < 30 {
		t.Errorf("expected at least 30 files, got %d", len(result.FileMetrics))
	}

	// Verify total file count matches
	if result.TotalFiles != len(result.FileMetrics) {
		t.Errorf("expected TotalFiles=%d, got %d", len(result.FileMetrics), result.TotalFiles)
	}

	t.Logf("Small scenario (30 files) scan completed in %v", duration)
}

// [REQ:TM-LS-008] Test light scan performance for medium scenarios (<200 files)
func TestScanPerformance_MediumScenario(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a medium scenario with ~150 files across multiple directories
	dirs := []string{"api", "ui/src/components", "ui/src/pages", "cli"}
	for _, dir := range dirs {
		fullDir := filepath.Join(tmpDir, dir)
		if err := os.MkdirAll(fullDir, 0755); err != nil {
			t.Fatalf("failed to create directory %s: %v", dir, err)
		}
	}

	// Create 150 test files across directories with realistic content
	for i := 0; i < 150; i++ {
		var dir string
		switch i % 4 {
		case 0:
			dir = "api"
		case 1:
			dir = "ui/src/components"
		case 2:
			dir = "ui/src/pages"
		case 3:
			dir = "cli"
		}

		fileName := "file" + string(rune('a'+i%26)) + string(rune('a'+(i/26)%26)) + ".go"
		testFile := filepath.Join(tmpDir, dir, fileName)

		content := `package main

import (
	"context"
	"fmt"
	"time"
)

type Handler struct {
	name string
	ctx  context.Context
}

func NewHandler(name string) *Handler {
	return &Handler{
		name: name,
		ctx:  context.Background(),
	}
}

func (h *Handler) Process() error {
	fmt.Printf("Processing %s\n", h.name)
	time.Sleep(10 * time.Millisecond)
	return nil
}

func (h *Handler) Close() {
	fmt.Println("Handler closed")
}
`
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file %s: %v", testFile, err)
		}
	}

	// Create simple Makefile with fast targets
	makefileContent := `
.PHONY: lint
lint:
	@echo "lint complete"

.PHONY: type
type:
	@echo "type complete"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "Makefile"), []byte(makefileContent), 0644); err != nil {
		t.Fatalf("failed to create Makefile: %v", err)
	}

	scanner := NewLightScanner(tmpDir, 120*time.Second)
	ctx := context.Background()

	start := time.Now()
	result, err := scanner.Scan(ctx)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	// Verify it completes in under 120 seconds (requirement threshold)
	if duration > 120*time.Second {
		t.Errorf("medium scenario scan took %v, expected <120s (requirement violation)", duration)
	}

	// Verify file count
	if len(result.FileMetrics) < 150 {
		t.Errorf("expected at least 150 files, got %d", len(result.FileMetrics))
	}

	// Verify total file count matches
	if result.TotalFiles != len(result.FileMetrics) {
		t.Errorf("expected TotalFiles=%d, got %d", len(result.FileMetrics), result.TotalFiles)
	}

	// Verify total lines were counted
	if result.TotalLines == 0 {
		t.Error("expected TotalLines to be greater than 0")
	}

	t.Logf("Medium scenario (150 files) scan completed in %v", duration)
}

// [REQ:TM-LS-007] Test scan performance with mixed file sizes (small scenario)
func TestScanPerformance_SmallScenarioMixedFileSizes(t *testing.T) {
	tmpDir := t.TempDir()

	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatalf("failed to create api directory: %v", err)
	}

	// Create files with varying sizes
	testCases := []struct {
		name      string
		lineCount int
	}{
		{"tiny.go", 5},
		{"small.go", 50},
		{"medium.go", 200},
		{"large.go", 400},
	}

	// Create multiple instances of each size
	fileCount := 0
	for _, tc := range testCases {
		for i := 0; i < 10; i++ {
			fileName := filepath.Join(apiDir, tc.name[:len(tc.name)-3]+"_"+string(rune('a'+i))+".go")
			content := "package main\n\n"
			for j := 0; j < tc.lineCount; j++ {
				content += "// Line comment to reach target size\n"
			}
			if err := os.WriteFile(fileName, []byte(content), 0644); err != nil {
				t.Fatalf("failed to create %s: %v", fileName, err)
			}
			fileCount++
		}
	}

	makefileContent := `
.PHONY: lint
lint:
	@echo "lint complete"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "Makefile"), []byte(makefileContent), 0644); err != nil {
		t.Fatalf("failed to create Makefile: %v", err)
	}

	scanner := NewLightScanner(tmpDir, 60*time.Second)
	ctx := context.Background()

	start := time.Now()
	result, err := scanner.Scan(ctx)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	// Verify performance requirement
	if duration > 60*time.Second {
		t.Errorf("small scenario with mixed file sizes took %v, expected <60s", duration)
	}

	// Verify all files were scanned
	if len(result.FileMetrics) != fileCount {
		t.Errorf("expected %d files, got %d", fileCount, len(result.FileMetrics))
	}

	t.Logf("Small scenario (%d mixed-size files) scan completed in %v", fileCount, duration)
}

// [REQ:TM-LS-008] Test scan performance with deeply nested directory structure
func TestScanPerformance_MediumScenarioDeeplyNested(t *testing.T) {
	tmpDir := t.TempDir()

	// Create deeply nested directory structure within api/ (where scanner looks)
	apiDir := filepath.Join(tmpDir, "api")
	for depth := 0; depth < 5; depth++ {
		for i := 0; i < 3; i++ {
			dirPath := filepath.Join(apiDir, "level"+string(rune('0'+depth)), "dir"+string(rune('a'+i)))
			if err := os.MkdirAll(dirPath, 0755); err != nil {
				t.Fatalf("failed to create nested directory: %v", err)
			}

			// Create 10 files in each directory
			for j := 0; j < 10; j++ {
				fileName := filepath.Join(dirPath, "file"+string(rune('a'+j))+".go")
				content := `package main

func test() {
  x := 1
  y := 2
  return
}
`
				if err := os.WriteFile(fileName, []byte(content), 0644); err != nil {
					t.Fatalf("failed to create file: %v", err)
				}
			}
		}
	}

	makefileContent := `
.PHONY: lint
lint:
	@echo "lint complete"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "Makefile"), []byte(makefileContent), 0644); err != nil {
		t.Fatalf("failed to create Makefile: %v", err)
	}

	scanner := NewLightScanner(tmpDir, 120*time.Second)
	ctx := context.Background()

	start := time.Now()
	result, err := scanner.Scan(ctx)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	// Verify performance requirement
	if duration > 120*time.Second {
		t.Errorf("medium scenario with nested directories took %v, expected <120s", duration)
	}

	// Verify files were found
	if len(result.FileMetrics) == 0 {
		t.Error("expected files to be found in nested structure")
	}

	t.Logf("Medium scenario (nested structure, %d files) scan completed in %v", len(result.FileMetrics), duration)
}

// [REQ:TM-LS-007] [REQ:TM-LS-008] Test scan performance consistency across multiple runs
func TestScanPerformance_Consistency(t *testing.T) {
	tmpDir := t.TempDir()

	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatalf("failed to create api directory: %v", err)
	}

	// Create 50 files
	for i := 0; i < 50; i++ {
		fileName := filepath.Join(apiDir, "file"+string(rune('a'+i%26))+string(rune('a'+i/26))+".go")
		content := `package main

func test() {
  x := 1
  return
}
`
		if err := os.WriteFile(fileName, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
	}

	makefileContent := `
.PHONY: lint
lint:
	@echo "lint complete"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "Makefile"), []byte(makefileContent), 0644); err != nil {
		t.Fatalf("failed to create Makefile: %v", err)
	}

	// Run scan 5 times and collect durations
	var durations []time.Duration
	for run := 0; run < 5; run++ {
		scanner := NewLightScanner(tmpDir, 60*time.Second)
		ctx := context.Background()

		start := time.Now()
		result, err := scanner.Scan(ctx)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("scan run %d failed: %v", run, err)
		}

		if len(result.FileMetrics) != 50 {
			t.Errorf("run %d: expected 50 files, got %d", run, len(result.FileMetrics))
		}

		durations = append(durations, duration)
	}

	// Calculate average and max duration
	var total time.Duration
	var maxDuration time.Duration
	for _, d := range durations {
		total += d
		if d > maxDuration {
			maxDuration = d
		}
	}
	avgDuration := total / time.Duration(len(durations))

	// All runs should be under the threshold
	if maxDuration > 60*time.Second {
		t.Errorf("slowest scan took %v, expected all runs <60s", maxDuration)
	}

	t.Logf("Scan performance across 5 runs: avg=%v, max=%v", avgDuration, maxDuration)
}
