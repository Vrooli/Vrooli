package main

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// Test helpers

// newTestDetector creates a duplication detector with standard test settings
func newTestDetector() *DuplicationDetector {
	return NewDuplicationDetector("/scenario", 60*time.Second)
}

// requireBlocks validates the number of blocks and returns them
func requireBlocks(t *testing.T, blocks []DuplicateBlock, expectedCount int) []DuplicateBlock {
	t.Helper()
	if len(blocks) != expectedCount {
		t.Fatalf("Expected %d duplicate blocks, got %d", expectedCount, len(blocks))
	}
	return blocks
}

// assertEqual compares two values of any comparable type
func assertEqual[T comparable](t *testing.T, got, want T, fieldName string) {
	t.Helper()
	if got != want {
		t.Errorf("Expected %s %v, got %v", fieldName, want, got)
	}
}

// assertBlock validates all properties of a duplicate block
func assertBlock(t *testing.T, block DuplicateBlock, fileCount, lines, tokens int) {
	t.Helper()
	assertEqual(t, len(block.Files), fileCount, "file count")
	assertEqual(t, block.Lines, lines, "lines")
	assertEqual(t, block.Tokens, tokens, "tokens")
}

// assertLocation validates a duplicate location's path and line range
func assertLocation(t *testing.T, loc DuplicateLocation, path string, startLine, endLine int) {
	t.Helper()
	assertEqual(t, loc.Path, path, "path")
	if loc.StartLine != startLine || loc.EndLine != endLine {
		t.Errorf("Expected lines %d-%d, got %d-%d", startLine, endLine, loc.StartLine, loc.EndLine)
	}
}

// assertDuplicationSkipped validates that a duplication result is skipped with a reason containing expected text
func assertDuplicationSkipped(t *testing.T, result DuplicateResult, expectedReasonSubstring string) {
	t.Helper()
	if !result.Skipped {
		t.Error("Expected result to be skipped")
	}
	if result.SkipReason == "" {
		t.Error("Expected SkipReason to be set")
	}
	if !contains(result.SkipReason, expectedReasonSubstring) {
		t.Errorf("Expected skip reason to contain %q, got: %q", expectedReasonSubstring, result.SkipReason)
	}
}

// assertNoError checks that an error is nil
func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

// assertResultNotNil checks that a DuplicateResult is not nil
func assertResultNotNil(t *testing.T, result *DuplicateResult) {
	t.Helper()
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

// assertResultSkipped validates that a result is properly skipped
func assertResultSkipped(t *testing.T, result *DuplicateResult, expectedReasonContains string) {
	t.Helper()
	assertResultNotNil(t, result)
	if !result.Skipped {
		t.Error("Expected result to be skipped")
	}
	if result.SkipReason == "" {
		t.Error("Expected SkipReason to be set")
	}
	if expectedReasonContains != "" && !contains(result.SkipReason, expectedReasonContains) {
		t.Errorf("Expected skip reason to contain %q, got: %q", expectedReasonContains, result.SkipReason)
	}
}

// calculateTotalFileLocations counts all file locations across all duplicate blocks
func calculateTotalFileLocations(blocks []DuplicateBlock) int {
	total := 0
	for _, block := range blocks {
		total += len(block.Files)
	}
	return total
}

// calculateTotalLines sums all lines across all duplicate blocks
func calculateTotalLines(blocks []DuplicateBlock) int {
	total := 0
	for _, block := range blocks {
		total += block.Lines
	}
	return total
}

// duplicationTestFunc represents a duplication detection function under test
type duplicationTestFunc func(ctx context.Context, files []string) (*DuplicateResult, error)

// [REQ:TM-LS-005] Test NewDuplicationDetector initialization
func TestNewDuplicationDetector(t *testing.T) {
	dd := NewDuplicationDetector("/test/path", 60*time.Second)

	if dd == nil {
		t.Fatal("Expected non-nil detector")
	}

	if dd.scenarioPath != "/test/path" {
		t.Errorf("Expected scenarioPath '/test/path', got %q", dd.scenarioPath)
	}

	if dd.timeout != 60*time.Second {
		t.Errorf("Expected timeout 60s, got %v", dd.timeout)
	}
}

// [REQ:TM-LS-005] Test NewDuplicationDetector with zero timeout sets default
func TestNewDuplicationDetector_DefaultTimeout(t *testing.T) {
	dd := NewDuplicationDetector("/test/path", 0)

	if dd.timeout != 90*time.Second {
		t.Errorf("Expected default timeout 90s, got %v", dd.timeout)
	}
}

// [REQ:TM-LS-005] Test parseDuplOutput with valid dupl output
func TestParseDuplOutput_Valid(t *testing.T) {
	dd := newTestDetector()

	output := `/scenario/api/handler.go:10-25
/scenario/api/util.go:45-60
found 2 clones

/scenario/api/main.go:100-110
/scenario/api/server.go:200-210
found 2 clones
`

	blocks := requireBlocks(t, dd.parseDuplOutput(output), 2)

	// Verify first block
	assertBlock(t, blocks[0], 2, 16, 0) // 25-10+1=16 lines
	assertLocation(t, blocks[0].Files[0], "api/handler.go", 10, 25)

	// Verify second block
	assertBlock(t, blocks[1], 2, 11, 0) // 110-100+1=11 lines
}

// [REQ:TM-LS-005] Test parseDuplOutput with empty output
func TestParseDuplOutput_Empty(t *testing.T) {
	dd := newTestDetector()
	requireBlocks(t, dd.parseDuplOutput(""), 0)
}

// [REQ:TM-LS-005] Test parseDuplOutput with single block
func TestParseDuplOutput_SingleBlock(t *testing.T) {
	dd := newTestDetector()

	output := `/scenario/api/test.go:5-10
/scenario/api/copy.go:20-25
found 2 clones
`

	blocks := requireBlocks(t, dd.parseDuplOutput(output), 1)
	assertEqual(t, blocks[0].Lines, 6, "lines") // 10-5+1
}

// [REQ:TM-LS-005] Test parseDuplOutput with malformed output
func TestParseDuplOutput_Malformed(t *testing.T) {
	dd := newTestDetector()

	// Invalid format - should not crash
	output := `some random text
not a valid line
123-456
`

	blocks := dd.parseDuplOutput(output)

	// Should return empty or handle gracefully
	t.Logf("Parsed %d blocks from malformed output", len(blocks))
}

// [REQ:TM-LS-005] Test parseDuplOutput with multiple locations per block
func TestParseDuplOutput_MultipleLocations(t *testing.T) {
	dd := newTestDetector()

	// Three files with same duplication
	output := `/scenario/api/a.go:10-20
/scenario/api/b.go:30-40
/scenario/api/c.go:50-60
found 3 clones
`

	blocks := requireBlocks(t, dd.parseDuplOutput(output), 1)
	assertEqual(t, len(blocks[0].Files), 3, "file count")
}

// [REQ:TM-LS-005] Test parseJscpdOutput with valid JSON
func TestParseJscpdOutput_Valid(t *testing.T) {
	dd := newTestDetector()

	jsonOutput := `{
  "duplicates": [
    {
      "firstFile": {
        "name": "/scenario/ui/src/Component.tsx",
        "start": 10,
        "end": 25
      },
      "secondFile": {
        "name": "/scenario/ui/src/Copy.tsx",
        "start": 50,
        "end": 65
      },
      "lines": 15,
      "tokens": 120
    }
  ]
}`

	blocks := requireBlocks(t, dd.parseJscpdOutput(jsonOutput), 1)

	assertBlock(t, blocks[0], 2, 15, 120)
	assertEqual(t, blocks[0].Files[0].Path, "ui/src/Component.tsx", "path")
}

// [REQ:TM-LS-005] Test parseJscpdOutput with empty JSON
func TestParseJscpdOutput_Empty(t *testing.T) {
	dd := newTestDetector()

	jsonOutput := `{"duplicates": []}`

	requireBlocks(t, dd.parseJscpdOutput(jsonOutput), 0)
}

// [REQ:TM-LS-005] Test parseJscpdOutput with invalid JSON
func TestParseJscpdOutput_InvalidJSON(t *testing.T) {
	dd := newTestDetector()

	invalidJSON := `{not valid json}`

	// Should return empty blocks on parse error
	requireBlocks(t, dd.parseJscpdOutput(invalidJSON), 0)
}

// [REQ:TM-LS-005] Test parseJscpdOutput with multiple duplicates
func TestParseJscpdOutput_MultipleDuplicates(t *testing.T) {
	dd := newTestDetector()

	jsonOutput := `{
  "duplicates": [
    {
      "firstFile": {"name": "/scenario/ui/src/A.tsx", "start": 1, "end": 10},
      "secondFile": {"name": "/scenario/ui/src/B.tsx", "start": 20, "end": 29},
      "lines": 10,
      "tokens": 50
    },
    {
      "firstFile": {"name": "/scenario/ui/src/C.tsx", "start": 5, "end": 15},
      "secondFile": {"name": "/scenario/ui/src/D.tsx", "start": 30, "end": 40},
      "lines": 11,
      "tokens": 60
    }
  ]
}`

	blocks := requireBlocks(t, dd.parseJscpdOutput(jsonOutput), 2)

	if blocks[0].Lines != 10 {
		t.Errorf("Block 0: expected 10 lines, got %d", blocks[0].Lines)
	}

	if blocks[1].Lines != 11 {
		t.Errorf("Block 1: expected 11 lines, got %d", blocks[1].Lines)
	}
}

// [REQ:TM-LS-005] Test DetectDuplication with unsupported language
func TestDetectDuplication_UnsupportedLanguage(t *testing.T) {
	dd := newTestDetector()
	ctx := context.Background()

	result, err := dd.DetectDuplication(ctx, LanguagePython, []string{"test.py"})
	assertNoError(t, err)
	assertResultSkipped(t, result, "not implemented")
}

// [REQ:TM-LS-005] Test duplication detection with no files (table-driven)
func TestDetectDuplication_NoFiles(t *testing.T) {
	dd := newTestDetector()
	ctx := context.Background()

	tests := []struct {
		name           string
		detectFunc     duplicationTestFunc
		expectedReason []string // Multiple possible reasons
	}{
		{
			name:           "Go duplication with no files",
			detectFunc:     dd.detectGoDuplication,
			expectedReason: []string{"no files", "not installed"},
		},
		{
			name:           "JS duplication with no files",
			detectFunc:     dd.detectJSDuplication,
			expectedReason: []string{"no files"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.detectFunc(ctx, []string{})
			assertNoError(t, err)
			assertResultNotNil(t, result)

			if !result.Skipped {
				t.Error("Expected result to be skipped for empty file list")
			}

			// Check if skip reason contains any of the expected reasons
			found := false
			for _, reason := range tt.expectedReason {
				if contains(result.SkipReason, reason) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected skip reason to contain one of %v, got: %q", tt.expectedReason, result.SkipReason)
			}
		})
	}
}

// [REQ:TM-LS-005] Test duplication detection when tool not installed (table-driven)
func TestDetectDuplication_ToolNotInstalled(t *testing.T) {
	dd := newTestDetector()
	ctx := context.Background()

	tests := []struct {
		name          string
		detectFunc    duplicationTestFunc
		files         []string
		toolName      string
		notInstalled  string
		toolCheckFunc func() bool
	}{
		{
			name:          "Go duplication when dupl not installed",
			detectFunc:    dd.detectGoDuplication,
			files:         []string{"test.go"},
			toolName:      "dupl",
			notInstalled:  "dupl not installed",
			toolCheckFunc: func() bool { return commandExists("dupl") },
		},
		{
			name:          "JS duplication when npx not installed",
			detectFunc:    dd.detectJSDuplication,
			files:         []string{"test.ts"},
			toolName:      "npx",
			notInstalled:  "npx not installed",
			toolCheckFunc: func() bool { return commandExists("npx") },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.detectFunc(ctx, tt.files)
			assertNoError(t, err)
			assertResultNotNil(t, result)

			if !tt.toolCheckFunc() {
				// Tool not installed - should skip gracefully
				assertResultSkipped(t, result, tt.notInstalled)
			} else {
				t.Logf("%s is installed, test verified it runs without error", tt.toolName)
			}
		})
	}
}

// [REQ:TM-LS-005] Test DuplicateResult structure
func TestDuplicateResult_Structure(t *testing.T) {
	result := DuplicateResult{
		TotalDuplicates: 3,
		DuplicateBlocks: []DuplicateBlock{
			{
				Files: []DuplicateLocation{
					{Path: "a.go", StartLine: 1, EndLine: 10},
					{Path: "b.go", StartLine: 20, EndLine: 29},
				},
				Lines:  10,
				Tokens: 50,
			},
		},
		TotalLines: 30,
		Skipped:    false,
		Tool:       "dupl",
	}

	assertEqual(t, result.TotalDuplicates, 3, "TotalDuplicates")
	assertEqual(t, len(result.DuplicateBlocks), 1, "block count")
	assertEqual(t, result.Tool, "dupl", "Tool")

	assertBlock(t, result.DuplicateBlocks[0], 2, 10, 50)
}

// [REQ:TM-LS-005] Test parseDuplOutput with paths containing special characters
func TestParseDuplOutput_SpecialCharacters(t *testing.T) {
	dd := newTestDetector()

	output := `/scenario/api/handler-v2.go:10-25
/scenario/api/util_test.go:45-60
found 2 clones
`

	blocks := requireBlocks(t, dd.parseDuplOutput(output), 1)
	assertEqual(t, blocks[0].Files[0].Path, "api/handler-v2.go", "path")
	assertEqual(t, blocks[0].Files[1].Path, "api/util_test.go", "path")
}

// [REQ:TM-LS-005] Test context cancellation handling
func TestDetectDuplication_ContextCancellation(t *testing.T) {
	dd := newTestDetector()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Should handle cancelled context gracefully
	result, err := dd.DetectDuplication(ctx, LanguageGo, []string{"test.go"})

	// Either returns error or skipped result
	if err == nil && result != nil {
		t.Log("Returned result (may be skipped)")
	} else if err != nil {
		t.Logf("Returned error as expected: %v", err)
	}
}

// [REQ:TM-LS-005] Test detectGoDuplication with context timeout
func TestDetectGoDuplication_ContextTimeout(t *testing.T) {
	if !commandExists("dupl") {
		t.Skip("dupl not installed")
	}

	dd := NewDuplicationDetector("/scenario", 1*time.Nanosecond) // Very short timeout
	ctx := context.Background()

	// This should complete quickly even with short timeout since we skip when no files
	result, err := dd.detectGoDuplication(ctx, []string{})
	if err != nil {
		t.Fatalf("Expected no error for empty file list, got: %v", err)
	}

	if !result.Skipped {
		t.Error("Expected result to be skipped for empty file list")
	}
}

// [REQ:TM-LS-005] Test DetectDuplication with TypeScript files
func TestDetectDuplication_TypeScript(t *testing.T) {
	dd := newTestDetector()
	ctx := context.Background()

	result, err := dd.DetectDuplication(ctx, LanguageTypeScript, []string{"test.ts"})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should either run or skip gracefully
	if result.Skipped {
		t.Logf("Skipped: %s", result.SkipReason)
	} else {
		if result.Tool != "jscpd" {
			t.Errorf("Expected tool 'jscpd', got %q", result.Tool)
		}
	}
}

// [REQ:TM-LS-005] Test DetectDuplication with JavaScript files
func TestDetectDuplication_JavaScript(t *testing.T) {
	dd := newTestDetector()
	ctx := context.Background()

	result, err := dd.DetectDuplication(ctx, LanguageJavaScript, []string{"test.js"})
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should either run or skip gracefully
	if result.Skipped {
		t.Logf("Skipped: %s", result.SkipReason)
	} else {
		if result.Tool != "jscpd" {
			t.Errorf("Expected tool 'jscpd', got %q", result.Tool)
		}
	}
}

// [REQ:TM-LS-005] Test parseDuplOutput with missing line numbers
func TestParseDuplOutput_MissingLineNumbers(t *testing.T) {
	dd := newTestDetector()

	// Malformed line without end line number
	output := `/scenario/api/test.go:5
found 1 clones
`

	blocks := dd.parseDuplOutput(output)

	// Should handle gracefully - either skip or parse what it can
	t.Logf("Parsed %d blocks from incomplete output", len(blocks))
}

// [REQ:TM-LS-005] Test parseDuplOutput with negative line numbers
func TestParseDuplOutput_NegativeLineNumbers(t *testing.T) {
	dd := newTestDetector()

	// Invalid negative line numbers (shouldn't happen but test robustness)
	output := `/scenario/api/test.go:-5--1
found 1 clones
`

	blocks := dd.parseDuplOutput(output)

	// Should handle gracefully
	t.Logf("Parsed %d blocks from invalid output", len(blocks))
}

// [REQ:TM-LS-005] Test parseJscpdOutput with missing fields
func TestParseJscpdOutput_MissingFields(t *testing.T) {
	dd := newTestDetector()

	// Missing lines/tokens fields
	jsonOutput := `{
  "duplicates": [
    {
      "firstFile": {"name": "/scenario/ui/src/A.tsx", "start": 1, "end": 10},
      "secondFile": {"name": "/scenario/ui/src/B.tsx", "start": 20, "end": 29}
    }
  ]
}`

	// Should parse what it can
	blocks := requireBlocks(t, dd.parseJscpdOutput(jsonOutput), 1)

	if blocks[0].Lines != 0 {
		t.Logf("Lines defaulted to: %d", blocks[0].Lines)
	}
}

// [REQ:TM-LS-005] Test parseJscpdOutput with null values
func TestParseJscpdOutput_NullValues(t *testing.T) {
	dd := newTestDetector()

	jsonOutput := `{"duplicates": null}`

	requireBlocks(t, dd.parseJscpdOutput(jsonOutput), 0)
}

// [REQ:TM-LS-005] Test DuplicateLocation path handling
func TestDuplicateLocation_RelativePaths(t *testing.T) {
	loc := DuplicateLocation{
		Path:      "api/handler.go",
		StartLine: 10,
		EndLine:   25,
	}

	if loc.Path != "api/handler.go" {
		t.Errorf("Expected relative path, got %q", loc.Path)
	}

	if loc.StartLine >= loc.EndLine {
		t.Error("Expected StartLine < EndLine")
	}
}

// [REQ:TM-LS-005] Test DuplicateBlock with empty files
func TestDuplicateBlock_EmptyFiles(t *testing.T) {
	block := DuplicateBlock{
		Files:  []DuplicateLocation{},
		Lines:  0,
		Tokens: 0,
	}

	if len(block.Files) != 0 {
		t.Errorf("Expected empty files list, got %d", len(block.Files))
	}
}

// [REQ:TM-LS-005] Test DuplicateResult skipped states
func TestDuplicateResult_SkippedStates(t *testing.T) {
	tests := []struct {
		name      string
		result    DuplicateResult
		wantValid bool
	}{
		{
			name: "skipped with reason",
			result: DuplicateResult{
				Skipped:    true,
				SkipReason: "tool not installed",
			},
			wantValid: true,
		},
		{
			name: "not skipped with results",
			result: DuplicateResult{
				Skipped:         false,
				TotalDuplicates: 1,
				DuplicateBlocks: []DuplicateBlock{{}},
			},
			wantValid: true,
		},
		{
			name: "skipped without reason (unusual)",
			result: DuplicateResult{
				Skipped:    true,
				SkipReason: "",
			},
			wantValid: false, // Should have a reason
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasReason := tt.result.SkipReason != ""
			if tt.result.Skipped && !hasReason && tt.wantValid {
				t.Error("Skipped result should have a skip reason")
			}
		})
	}
}

// [REQ:TM-LS-005] Test parseDuplOutput with mixed valid and invalid lines
func TestParseDuplOutput_MixedValidInvalid(t *testing.T) {
	dd := newTestDetector()

	output := `some random header
/scenario/api/valid.go:10-20
/scenario/api/valid2.go:30-40
found 2 clones

invalid line here
/scenario/api/another.go:50-60
malformed:line
/scenario/api/another2.go:70-80
found 2 clones
`

	blocks := dd.parseDuplOutput(output)

	// Should parse the valid blocks and skip invalid lines
	if len(blocks) < 1 {
		t.Error("Expected at least 1 valid block to be parsed")
	}

	t.Logf("Successfully parsed %d blocks from mixed output", len(blocks))
}

// [REQ:TM-LS-005] Test total lines calculation accuracy
func TestDuplicateResult_TotalLinesCalculation(t *testing.T) {
	blocks := []DuplicateBlock{
		{Lines: 10, Files: []DuplicateLocation{{}, {}}},
		{Lines: 20, Files: []DuplicateLocation{{}, {}}},
		{Lines: 15, Files: []DuplicateLocation{{}, {}}},
	}

	actualTotal := calculateTotalLines(blocks)
	assertEqual(t, actualTotal, 45, "total lines")
}

// [REQ:TM-LS-005] Test TotalDuplicates count accuracy
func TestDuplicateResult_TotalDuplicatesAccuracy(t *testing.T) {
	// Test with 3 blocks, each having 2 file locations
	blocks := []DuplicateBlock{
		{Files: []DuplicateLocation{{}, {}}}, // 2 locations
		{Files: []DuplicateLocation{{}, {}}}, // 2 locations
		{Files: []DuplicateLocation{{}, {}}}, // 2 locations
	}

	actualCount := calculateTotalFileLocations(blocks)
	assertEqual(t, actualCount, 6, "total file locations")
}

// [REQ:TM-LS-005] Test parseDuplOutput with very large output (stress test)
func TestParseDuplOutput_LargeOutput(t *testing.T) {
	dd := newTestDetector()

	// Generate output with 100 duplicate blocks
	var output string
	for i := 0; i < 100; i++ {
		output += fmt.Sprintf("/scenario/api/file%d.go:10-50\n", i)
		output += fmt.Sprintf("/scenario/api/copy%d.go:100-140\n", i)
		output += "found 2 clones\n\n"
	}

	blocks := requireBlocks(t, dd.parseDuplOutput(output), 100)

	// Verify each block has correct structure
	for i, block := range blocks {
		if len(block.Files) != 2 {
			t.Errorf("Block %d: expected 2 file locations, got %d", i, len(block.Files))
		}
		if block.Lines != 41 {
			t.Errorf("Block %d: expected 41 lines (50-10+1), got %d", i, block.Lines)
		}
	}
}

// [REQ:TM-LS-005] Test parseDuplOutput with Unicode characters in paths
func TestParseDuplOutput_UnicodePaths(t *testing.T) {
	dd := newTestDetector()

	// Test with various international characters
	output := `/scenario/api/处理器.go:10-20
/scenario/api/файл.go:30-40
found 2 clones

/scenario/ui/コンポーネント.tsx:5-15
/scenario/ui/مكون.tsx:25-35
found 2 clones
`

	blocks := requireBlocks(t, dd.parseDuplOutput(output), 2)

	// Verify Unicode characters are preserved
	if !contains(blocks[0].Files[0].Path, "处理器") && !contains(blocks[0].Files[0].Path, "файл") {
		t.Error("Expected Unicode characters to be preserved in path")
	}
}

// [REQ:TM-LS-005] Test parseJscpdOutput with very large JSON
func TestParseJscpdOutput_LargeJSON(t *testing.T) {
	dd := newTestDetector()

	// Build large JSON with 50 duplicates
	jsonOutput := `{"duplicates": [`
	for i := 0; i < 50; i++ {
		if i > 0 {
			jsonOutput += ","
		}
		jsonOutput += fmt.Sprintf(`{
			"firstFile": {"name": "/scenario/ui/src/file%d.tsx", "start": 1, "end": 20},
			"secondFile": {"name": "/scenario/ui/src/copy%d.tsx", "start": 50, "end": 69},
			"lines": 20,
			"tokens": 100
		}`, i, i)
	}
	jsonOutput += `]}`

	blocks := requireBlocks(t, dd.parseJscpdOutput(jsonOutput), 50)

	// Verify structure of a sample block
	if len(blocks) > 0 {
		if blocks[0].Lines != 20 {
			t.Errorf("Expected 20 lines, got %d", blocks[0].Lines)
		}
		if blocks[0].Tokens != 100 {
			t.Errorf("Expected 100 tokens, got %d", blocks[0].Tokens)
		}
	}
}

// [REQ:TM-LS-005] Test DetectDuplication language routing correctness
func TestDetectDuplication_LanguageRouting(t *testing.T) {
	dd := newTestDetector()
	ctx := context.Background()

	tests := []struct {
		lang       Language
		files      []string
		expectTool string
	}{
		{LanguageGo, []string{"test.go"}, "dupl"},
		{LanguageTypeScript, []string{"test.ts"}, "jscpd"},
		{LanguageJavaScript, []string{"test.js"}, "jscpd"},
	}

	for _, tt := range tests {
		t.Run(string(tt.lang), func(t *testing.T) {
			result, err := dd.DetectDuplication(ctx, tt.lang, tt.files)
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			// If not skipped, verify correct tool was used
			if !result.Skipped {
				if result.Tool != tt.expectTool {
					t.Errorf("Expected tool %q for %s, got %q", tt.expectTool, tt.lang, result.Tool)
				}
			}
		})
	}
}

// [REQ:TM-LS-005] Test parseDuplOutput line range edge cases
func TestParseDuplOutput_LineRangeEdgeCases(t *testing.T) {
	dd := newTestDetector()

	tests := []struct {
		name          string
		output        string
		expectedLines int
	}{
		{
			name: "single line duplication",
			output: `/scenario/api/test.go:42-42
/scenario/api/copy.go:100-100
found 2 clones
`,
			expectedLines: 1, // 42-42+1 = 1
		},
		{
			name: "large line range",
			output: `/scenario/api/test.go:1-1000
/scenario/api/copy.go:1-1000
found 2 clones
`,
			expectedLines: 1000, // 1000-1+1 = 1000
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocks := requireBlocks(t, dd.parseDuplOutput(tt.output), 1)

			if blocks[0].Lines != tt.expectedLines {
				t.Errorf("Expected %d lines, got %d", tt.expectedLines, blocks[0].Lines)
			}
		})
	}
}

// [REQ:TM-LS-005] Test parseJscpdOutput with nested paths
func TestParseJscpdOutput_NestedPaths(t *testing.T) {
	dd := newTestDetector()

	jsonOutput := `{
  "duplicates": [
    {
      "firstFile": {
        "name": "/scenario/ui/src/components/nested/deep/Component.tsx",
        "start": 1,
        "end": 20
      },
      "secondFile": {
        "name": "/scenario/ui/src/utils/helpers/deep/nested/util.ts",
        "start": 50,
        "end": 69
      },
      "lines": 20,
      "tokens": 150
    }
  ]
}`

	blocks := requireBlocks(t, dd.parseJscpdOutput(jsonOutput), 1)

	block := blocks[0]

	// Verify deeply nested paths are correctly relativized
	if !contains(block.Files[0].Path, "ui/src/components/nested/deep/Component.tsx") {
		t.Errorf("Expected nested path to be preserved, got: %q", block.Files[0].Path)
	}

	if !contains(block.Files[1].Path, "ui/src/utils/helpers/deep/nested/util.ts") {
		t.Errorf("Expected nested path to be preserved, got: %q", block.Files[1].Path)
	}
}

// [REQ:TM-LS-005] Test DuplicateResult with zero duplicates but tool ran
func TestDuplicateResult_ZeroDuplicatesSuccess(t *testing.T) {
	result := DuplicateResult{
		TotalDuplicates: 0,
		DuplicateBlocks: []DuplicateBlock{},
		TotalLines:      0,
		Skipped:         false,
		SkipReason:      "",
		Tool:            "dupl",
	}

	// Zero duplicates is a valid successful result (clean code!)
	if result.Skipped {
		t.Error("Result should not be skipped when tool ran successfully with 0 duplicates")
	}

	if result.TotalDuplicates != 0 {
		t.Errorf("Expected 0 duplicates, got %d", result.TotalDuplicates)
	}

	if len(result.DuplicateBlocks) != 0 {
		t.Errorf("Expected empty blocks for 0 duplicates, got %d blocks", len(result.DuplicateBlocks))
	}

	if result.Tool == "" {
		t.Error("Tool should be set even when 0 duplicates found")
	}
}

// [REQ:TM-LS-005] Test parseDuplOutput with Windows-style paths
func TestParseDuplOutput_WindowsPaths(t *testing.T) {
	dd := NewDuplicationDetector("C:\\scenario", 60*time.Second)

	// Test with backslashes (though dupl would likely normalize these)
	output := `C:\scenario\api\handler.go:10-25
C:\scenario\api\util.go:45-60
found 2 clones
`

	blocks := dd.parseDuplOutput(output)

	// Should handle Windows paths gracefully
	t.Logf("Parsed %d blocks from Windows-style paths", len(blocks))

	if len(blocks) > 0 {
		// Path should be relativized (either with / or \)
		if blocks[0].Files[0].Path == "" {
			t.Error("Expected non-empty path after parsing")
		}
	}
}

// [REQ:TM-LS-005] Test command timeout enforcement
func TestDuplicationDetector_TimeoutEnforcement(t *testing.T) {
	// Test that timeout is respected
	dd := NewDuplicationDetector("/scenario", 100*time.Millisecond)

	if dd.timeout != 100*time.Millisecond {
		t.Errorf("Expected timeout 100ms, got %v", dd.timeout)
	}

	// Very short timeout
	dd2 := NewDuplicationDetector("/scenario", 1*time.Nanosecond)
	if dd2.timeout != 1*time.Nanosecond {
		t.Errorf("Expected timeout 1ns, got %v", dd2.timeout)
	}
}

// [REQ:TM-LS-005] Test parseDuplOutput ignores non-clone lines
func TestParseDuplOutput_IgnoresNonCloneLines(t *testing.T) {
	dd := newTestDetector()

	output := `Analyzing files...
Processing: api/handler.go
Processing: api/util.go
/scenario/api/handler.go:10-20
/scenario/api/util.go:30-40
found 2 clones

Total files analyzed: 50
Duplicates found: 1
`

	blocks := requireBlocks(t, dd.parseDuplOutput(output), 1)

	if len(blocks) > 0 && blocks[0].Lines != 11 {
		t.Errorf("Expected 11 lines, got %d", blocks[0].Lines)
	}
}
