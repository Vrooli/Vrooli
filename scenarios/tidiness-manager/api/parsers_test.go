package main

import (
	"fmt"
	"strings"
	"testing"
)

// [REQ:TM-LS-003] Test eslint format parsing
func TestParseLintOutput_ESLint(t *testing.T) {
	output := `
src/components/App.tsx:10:5: Unexpected console statement [no-console]
src/utils/helpers.ts:25:12: 'value' is assigned a value but never used [no-unused-vars]
`
	issues := ParseLintOutput("test-scenario", "eslint", strings.TrimSpace(output))

	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}

	// Check first issue
	issue := issues[0]
	if issue.File != "src/components/App.tsx" {
		t.Errorf("expected file 'src/components/App.tsx', got %q", issue.File)
	}
	if issue.Line != 10 {
		t.Errorf("expected line 10, got %d", issue.Line)
	}
	if issue.Column != 5 {
		t.Errorf("expected column 5, got %d", issue.Column)
	}
	if issue.Rule != "no-console" {
		t.Errorf("expected rule 'no-console', got %q", issue.Rule)
	}
	if issue.Category != "lint" {
		t.Errorf("expected category 'lint', got %q", issue.Category)
	}
	if issue.Tool != "eslint" {
		t.Errorf("expected tool 'eslint', got %q", issue.Tool)
	}

	// Check second issue
	issue2 := issues[1]
	if issue2.File != "src/utils/helpers.ts" {
		t.Errorf("expected file 'src/utils/helpers.ts', got %q", issue2.File)
	}
	if issue2.Rule != "no-unused-vars" {
		t.Errorf("expected rule 'no-unused-vars', got %q", issue2.Rule)
	}
}

// [REQ:TM-LS-003] Test golangci-lint format parsing
func TestParseLintOutput_GolangCILint(t *testing.T) {
	output := `
api/main.go:42:1: exported function NewServer should have comment or be unexported (revive)
api/handlers.go:15:8: G104: Errors unhandled (gosec)
`
	issues := ParseLintOutput("test-scenario", "golangci-lint", strings.TrimSpace(output))

	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}

	issue := issues[0]
	if issue.File != "api/main.go" {
		t.Errorf("expected file 'api/main.go', got %q", issue.File)
	}
	if issue.Line != 42 {
		t.Errorf("expected line 42, got %d", issue.Line)
	}
	if issue.Tool != "golangci-lint" {
		t.Errorf("expected tool 'golangci-lint', got %q", issue.Tool)
	}
}

// [REQ:TM-LS-004] Test TypeScript error parsing
func TestParseTypeOutput_TypeScript(t *testing.T) {
	output := `
src/App.tsx(10,5): error TS2304: Cannot find name 'unknown'
src/utils/api.ts(25,12): error TS2322: Type 'string' is not assignable to type 'number'
ui/components/Button.tsx(8,3): warning TS6133: 'props' is declared but never used
`
	issues := ParseTypeOutput("test-scenario", "tsc", strings.TrimSpace(output))

	if len(issues) != 3 {
		t.Fatalf("expected 3 issues, got %d", len(issues))
	}

	// Check error issue
	issue := issues[0]
	if issue.File != "src/App.tsx" {
		t.Errorf("expected file 'src/App.tsx', got %q", issue.File)
	}
	if issue.Line != 10 {
		t.Errorf("expected line 10, got %d", issue.Line)
	}
	if issue.Column != 5 {
		t.Errorf("expected column 5, got %d", issue.Column)
	}
	if issue.Severity != "error" {
		t.Errorf("expected severity 'error', got %q", issue.Severity)
	}
	if issue.Rule != "TS2304" {
		t.Errorf("expected rule 'TS2304', got %q", issue.Rule)
	}
	if issue.Category != "type" {
		t.Errorf("expected category 'type', got %q", issue.Category)
	}

	// Check warning issue
	issue3 := issues[2]
	if issue3.Severity != "warning" {
		t.Errorf("expected severity 'warning', got %q", issue3.Severity)
	}
	if issue3.Rule != "TS6133" {
		t.Errorf("expected rule 'TS6133', got %q", issue3.Rule)
	}
}

// [REQ:TM-LS-004] Test Go type error parsing
func TestParseTypeOutput_Go(t *testing.T) {
	output := `
api/main.go:42:8: undefined: unknownFunc
api/handlers.go:15:2: cannot use "string" (type string) as type int in assignment
`
	issues := ParseTypeOutput("test-scenario", "go", strings.TrimSpace(output))

	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}

	issue := issues[0]
	if issue.File != "api/main.go" {
		t.Errorf("expected file 'api/main.go', got %q", issue.File)
	}
	if issue.Line != 42 {
		t.Errorf("expected line 42, got %d", issue.Line)
	}
	if issue.Column != 8 {
		t.Errorf("expected column 8, got %d", issue.Column)
	}
	if issue.Severity != "error" {
		t.Errorf("expected severity 'error', got %q", issue.Severity)
	}
	if issue.Category != "type" {
		t.Errorf("expected category 'type', got %q", issue.Category)
	}
	if issue.Tool != "go" {
		t.Errorf("expected tool 'go', got %q", issue.Tool)
	}
}

// [REQ:TM-LS-003,TM-LS-004] Test handling empty output
func TestParseOutput_Empty(t *testing.T) {
	lintIssues := ParseLintOutput("test", "lint", "")
	if len(lintIssues) != 0 {
		t.Errorf("expected 0 lint issues for empty output, got %d", len(lintIssues))
	}

	typeIssues := ParseTypeOutput("test", "type", "")
	if len(typeIssues) != 0 {
		t.Errorf("expected 0 type issues for empty output, got %d", len(typeIssues))
	}
}

// [REQ:TM-LS-003,TM-LS-004] Test handling malformed output
func TestParseOutput_Malformed(t *testing.T) {
	malformed := `
This is not a valid lint output
Random text without structure
Another line of gibberish
`
	lintIssues := ParseLintOutput("test", "lint", malformed)
	// Should not crash, may return empty or partial results
	if lintIssues == nil {
		t.Error("expected non-nil slice even for malformed input")
	}

	typeIssues := ParseTypeOutput("test", "type", malformed)
	if typeIssues == nil {
		t.Error("expected non-nil slice even for malformed input")
	}
}

// [REQ:TM-LS-003] Test parsing with special characters
func TestParseLintOutput_SpecialCharacters(t *testing.T) {
	output := `src/file-name.tsx:10:5: Error with "quotes" and 'apostrophes' [rule-name]`
	issues := ParseLintOutput("test", "eslint", output)

	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}

	issue := issues[0]
	if !strings.Contains(issue.Message, "quotes") {
		t.Error("expected message to contain special characters")
	}
}

// [REQ:TM-LS-003] Test parsing with unicode and emoji in output
func TestParseLintOutput_Unicode(t *testing.T) {
	output := `src/日本語.tsx:10:5: Error with emoji 🔥 and special chars 你好 [rule-name]`
	issues := ParseLintOutput("test", "eslint", output)

	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}

	issue := issues[0]
	if !strings.Contains(issue.File, "日本語") {
		t.Errorf("expected file name to preserve unicode, got %q", issue.File)
	}
	if !strings.Contains(issue.Message, "🔥") {
		t.Error("expected message to preserve emoji")
	}
}

// [REQ:TM-LS-003,TM-LS-004] Test parsing with extremely long lines
func TestParseOutput_LongLines(t *testing.T) {
	// Create a line with extremely long message
	longMessage := strings.Repeat("A", 5000)
	output := fmt.Sprintf("file.go:10:5: %s [rule]", longMessage)
	issues := ParseLintOutput("test", "lint", output)

	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}

	if len(issues[0].Message) == 0 {
		t.Error("expected message to be preserved even if very long")
	}
}

// [REQ:TM-LS-003] Test parsing with invalid line/column numbers
func TestParseLintOutput_InvalidNumbers(t *testing.T) {
	testCases := []string{
		"file.go:abc:5: Message [rule]",              // Invalid line
		"file.go:10:xyz: Message [rule]",             // Invalid column
		"file.go:999999999999999:5: Message [rule]",  // Overflow line number
		"file.go:10:999999999999999: Message [rule]", // Overflow column
	}

	for _, output := range testCases {
		issues := ParseLintOutput("test", "lint", output)
		// Parser should handle gracefully (either parse or skip)
		if issues == nil {
			t.Error("expected non-nil result for invalid numbers")
		}
	}
}

// [REQ:TM-LS-003] Test parsing with missing components
func TestParseLintOutput_MissingComponents(t *testing.T) {
	testCases := []struct {
		name   string
		output string
	}{
		{"no line number", "file.go:: Message [rule]"},
		{"no column", "file.go:10: Message [rule]"},
		{"no message", "file.go:10:5: [rule]"},
		{"no rule", "file.go:10:5: Message"},
		{"only filename", "file.go"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			issues := ParseLintOutput("test", "lint", tc.output)
			// Should not crash, result may be empty or partial
			if issues == nil {
				t.Error("expected non-nil slice")
			}
		})
	}
}

// [REQ:TM-LS-004] Test TypeScript with multiple errors on same line
func TestParseTypeOutput_MultipleErrorsSameLine(t *testing.T) {
	output := `
src/App.tsx(10,5): error TS2304: Cannot find name 'foo'
src/App.tsx(10,12): error TS2304: Cannot find name 'bar'
src/App.tsx(10,20): error TS2322: Type mismatch
`
	issues := ParseTypeOutput("test", "tsc", strings.TrimSpace(output))

	if len(issues) != 3 {
		t.Fatalf("expected 3 issues, got %d", len(issues))
	}

	// All should be from same file and line
	for _, issue := range issues {
		if issue.File != "src/App.tsx" {
			t.Errorf("expected all issues from src/App.tsx, got %s", issue.File)
		}
		if issue.Line != 10 {
			t.Errorf("expected all issues on line 10, got %d", issue.Line)
		}
	}

	// Columns should be different
	columns := make(map[int]bool)
	for _, issue := range issues {
		columns[issue.Column] = true
	}
	if len(columns) != 3 {
		t.Error("expected 3 different columns for 3 issues on same line")
	}
}

// [REQ:TM-LS-004] Test Go error parsing with build errors
func TestParseTypeOutput_GoBuildErrors(t *testing.T) {
	output := `
# command-line-arguments
./main.go:10:8: undefined: unknownFunc
./main.go:15:2: cannot use "string" (type string) as type int
./handlers.go:20:10: missing return
`
	issues := ParseTypeOutput("test", "go", strings.TrimSpace(output))

	// Should skip the comment line and parse 3 errors
	if len(issues) != 3 {
		t.Fatalf("expected 3 issues, got %d", len(issues))
	}

	// Verify all are error severity
	for _, issue := range issues {
		if issue.Severity != "error" {
			t.Errorf("expected severity 'error', got %q", issue.Severity)
		}
		if issue.Category != "type" {
			t.Errorf("expected category 'type', got %q", issue.Category)
		}
	}
}

// [REQ:TM-LS-003,TM-LS-004] Test parsing with Windows-style paths
func TestParseOutput_WindowsPaths(t *testing.T) {
	output := `C:\Users\user\project\src\file.ts:10:5: Error message [rule]`
	issues := ParseLintOutput("test", "lint", output)

	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}

	issue := issues[0]
	if !strings.Contains(issue.File, "file.ts") {
		t.Errorf("expected Windows path to be preserved, got %q", issue.File)
	}
}

// [REQ:TM-LS-003] Test parsing with ANSI color codes
func TestParseLintOutput_ANSIColors(t *testing.T) {
	// Output with ANSI color codes (common in terminal output)
	output := "\033[31mfile.go:10:5: Error message\033[0m [rule]"
	issues := ParseLintOutput("test", "lint", output)

	// Parser should handle ANSI codes gracefully
	if len(issues) == 0 {
		t.Error("expected parser to extract issue despite ANSI codes")
	}
}

// [REQ:TM-LS-003,TM-LS-004] Test parsing with mixed severity levels
func TestParseOutput_MixedSeverity(t *testing.T) {
	output := `
file.go:10:5: fatal error: cannot compile
file.go:15:2: error: undefined variable
file.go:20:3: warning: unused variable
file.go:25:4: note: consider using const
`
	issues := ParseLintOutput("test", "lint", strings.TrimSpace(output))

	if len(issues) == 0 {
		t.Fatal("expected at least one issue")
	}

	// Verify severity detection works for various keywords
	hasError := false
	for _, issue := range issues {
		if issue.Severity == "error" {
			hasError = true
		}
	}

	if !hasError {
		t.Error("expected at least one error severity issue")
	}
}

// [REQ:TM-LS-004] Test TypeScript with suggestions
func TestParseTypeOutput_WithSuggestions(t *testing.T) {
	output := `
src/App.tsx(10,5): error TS2304: Cannot find name 'Foo'. Did you mean 'foo'?
src/App.tsx(15,3): warning TS6133: 'unused' is declared but its value is never read.
`
	issues := ParseTypeOutput("test", "tsc", strings.TrimSpace(output))

	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}

	// First issue should contain suggestion in message
	if !strings.Contains(issues[0].Message, "Did you mean") {
		t.Error("expected suggestion to be preserved in message")
	}

	// Verify one error and one warning
	if issues[0].Severity != "error" || issues[1].Severity != "warning" {
		t.Error("expected one error and one warning")
	}
}

// [REQ:TM-LS-003] Test parsing output with relative and absolute paths
func TestParseLintOutput_MixedPaths(t *testing.T) {
	output := `
./src/file1.go:10:5: Error in relative path [rule1]
/home/user/project/src/file2.go:20:10: Error in absolute path [rule2]
src/file3.go:30:15: Error in simple relative [rule3]
`
	issues := ParseLintOutput("test", "lint", strings.TrimSpace(output))

	if len(issues) != 3 {
		t.Fatalf("expected 3 issues, got %d", len(issues))
	}

	// All paths should be preserved as-is
	expectedFiles := []string{"./src/file1.go", "/home/user/project/src/file2.go", "src/file3.go"}
	for i, expected := range expectedFiles {
		if issues[i].File != expected {
			t.Errorf("Issue %d: expected file %q, got %q", i, expected, issues[i].File)
		}
	}
}

// [REQ:TM-LS-003,TM-LS-004] Test concurrent parsing safety
func TestParseOutput_Concurrent(t *testing.T) {
	lintOutput := `
file1.go:10:5: Error 1 [rule1]
file2.go:20:10: Error 2 [rule2]
file3.go:30:15: Error 3 [rule3]
`
	typeOutput := `
file1.ts(10,5): error TS2304: Error 1
file2.ts(20,10): error TS2322: Error 2
file3.ts(30,15): warning TS6133: Error 3
`

	// Run 10 concurrent parse operations
	done := make(chan bool, 20)
	for i := 0; i < 10; i++ {
		go func() {
			issues := ParseLintOutput("test", "lint", lintOutput)
			if len(issues) != 3 {
				t.Errorf("concurrent lint parse: expected 3 issues, got %d", len(issues))
			}
			done <- true
		}()
		go func() {
			issues := ParseTypeOutput("test", "tsc", typeOutput)
			if len(issues) != 3 {
				t.Errorf("concurrent type parse: expected 3 issues, got %d", len(issues))
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}
}

// [REQ:TM-LS-003] Test parsing very large output (stress test)
func TestParseLintOutput_LargeOutput(t *testing.T) {
	// Generate output with 1000 issues
	var builder strings.Builder
	for i := 1; i <= 1000; i++ {
		builder.WriteString(fmt.Sprintf("file%d.go:%d:5: Error message %d [rule%d]\n", i, i, i, i%10))
	}

	issues := ParseLintOutput("test", "lint", builder.String())

	if len(issues) != 1000 {
		t.Fatalf("expected 1000 issues, got %d", len(issues))
	}

	// Verify first and last issues are correct
	if issues[0].File != "file1.go" || issues[0].Line != 1 {
		t.Error("first issue incorrectly parsed")
	}
	if issues[999].File != "file1000.go" || issues[999].Line != 1000 {
		t.Error("last issue incorrectly parsed")
	}
}

// [REQ:TM-LS-004] Test TypeScript parsing with multiple files and nested paths
func TestParseTypeOutput_NestedPaths(t *testing.T) {
	output := `
src/components/ui/Button.tsx(10,5): error TS2304: Cannot find name 'ButtonProps'
src/components/ui/forms/Input.tsx(20,10): warning TS6133: 'onChange' is declared but never used
src/utils/api/client/fetch.ts(30,15): error TS2322: Type 'string' is not assignable to type 'number'
`
	issues := ParseTypeOutput("test", "tsc", strings.TrimSpace(output))

	if len(issues) != 3 {
		t.Fatalf("expected 3 issues, got %d", len(issues))
	}

	// Verify nested paths are preserved correctly
	expectedPaths := []string{
		"src/components/ui/Button.tsx",
		"src/components/ui/forms/Input.tsx",
		"src/utils/api/client/fetch.ts",
	}

	for i, expected := range expectedPaths {
		if issues[i].File != expected {
			t.Errorf("Issue %d: expected file %q, got %q", i, expected, issues[i].File)
		}
	}
}

// [REQ:TM-LS-003] Test parsing with multiline error messages
func TestParseLintOutput_MultilineMessages(t *testing.T) {
	// Some linters output multiline messages - ensure we handle gracefully
	output := `file.go:10:5: Error on line 1
  Additional context on line 2
  Suggestion on line 3 [rule1]
file2.go:20:10: Single line error [rule2]`

	issues := ParseLintOutput("test", "lint", output)

	// Should parse at least the valid issues
	if len(issues) == 0 {
		t.Error("expected at least one issue to be parsed")
	}

	// Verify we got the second single-line issue
	foundSingleLine := false
	for _, issue := range issues {
		if issue.File == "file2.go" && issue.Line == 20 {
			foundSingleLine = true
			if issue.Rule != "rule2" {
				t.Errorf("expected rule 'rule2', got %q", issue.Rule)
			}
		}
	}

	if !foundSingleLine {
		t.Error("expected to find single-line issue")
	}
}

// [REQ:TM-LS-003,TM-LS-004] Test parsing with null bytes and control characters
func TestParseOutput_ControlCharacters(t *testing.T) {
	// Test with various control characters that might appear in corrupted output
	testCases := []struct {
		name   string
		output string
	}{
		{"null byte", "file.go:10:5: Error\x00message [rule]"},
		{"tab character", "file.go:10:5: Error\tmessage [rule]"},
		{"carriage return", "file.go:10:5: Error\rmessage [rule]"},
		{"form feed", "file.go:10:5: Error\fmessage [rule]"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			issues := ParseLintOutput("test", "lint", tc.output)
			// Should not crash, might parse successfully or skip
			if issues == nil {
				t.Error("expected non-nil result")
			}
		})
	}
}

// [REQ:TM-LS-003] Test scenario name preservation
func TestParseLintOutput_ScenarioName(t *testing.T) {
	scenarioName := "my-test-scenario"
	output := "file.go:10:5: Error message [rule]"

	issues := ParseLintOutput(scenarioName, "lint", output)

	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}

	if issues[0].Scenario != scenarioName {
		t.Errorf("expected scenario %q, got %q", scenarioName, issues[0].Scenario)
	}
}

// [REQ:TM-LS-004] Test tool name preservation
func TestParseTypeOutput_ToolName(t *testing.T) {
	toolName := "my-custom-tsc"
	output := "file.ts(10,5): error TS2304: Cannot find name 'foo'"

	issues := ParseTypeOutput("test", toolName, output)

	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}

	if issues[0].Tool != toolName {
		t.Errorf("expected tool %q, got %q", toolName, issues[0].Tool)
	}
}

// [REQ:TM-LS-003] Benchmark lint parsing performance
func BenchmarkParseLintOutput_Small(b *testing.B) {
	output := `
src/components/App.tsx:10:5: Unexpected console statement [no-console]
src/utils/helpers.ts:25:12: 'value' is assigned a value but never used [no-unused-vars]
`
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseLintOutput("test", "eslint", strings.TrimSpace(output))
	}
}

// [REQ:TM-LS-003] Benchmark lint parsing with large output
func BenchmarkParseLintOutput_Large(b *testing.B) {
	var builder strings.Builder
	for i := 1; i <= 1000; i++ {
		builder.WriteString(fmt.Sprintf("file%d.go:%d:5: Error message %d [rule%d]\n", i, i, i, i%10))
	}
	output := builder.String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseLintOutput("test", "lint", output)
	}
}

// [REQ:TM-LS-004] Benchmark type output parsing
func BenchmarkParseTypeOutput_TypeScript(b *testing.B) {
	output := `
src/App.tsx(10,5): error TS2304: Cannot find name 'unknown'
src/utils/api.ts(25,12): error TS2322: Type 'string' is not assignable to type 'number'
ui/components/Button.tsx(8,3): warning TS6133: 'props' is declared but never used
`
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseTypeOutput("test", "tsc", strings.TrimSpace(output))
	}
}

// [REQ:TM-LS-003,TM-LS-004] Test that all Issue struct fields are properly populated
func TestParseOutput_CompleteIssueFields(t *testing.T) {
	testCases := []struct {
		name      string
		parseFunc func(string, string, string) []Issue
		scenario  string
		tool      string
		output    string
		expected  Issue
	}{
		{
			name:      "lint issue complete fields",
			parseFunc: ParseLintOutput,
			scenario:  "test-scenario",
			tool:      "eslint",
			output:    "src/App.tsx:10:5: Unexpected console [no-console]",
			expected: Issue{
				Scenario: "test-scenario",
				Tool:     "eslint",
				Category: "lint",
				File:     "src/App.tsx",
				Line:     10,
				Column:   5,
				Rule:     "no-console",
				Severity: "", // May not be set for basic lint output
			},
		},
		{
			name:      "type issue complete fields",
			parseFunc: ParseTypeOutput,
			scenario:  "my-scenario",
			tool:      "tsc",
			output:    "src/App.tsx(10,5): error TS2304: Cannot find name 'foo'",
			expected: Issue{
				Scenario: "my-scenario",
				Tool:     "tsc",
				Category: "type",
				File:     "src/App.tsx",
				Line:     10,
				Column:   5,
				Rule:     "TS2304",
				Severity: "error",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			issues := tc.parseFunc(tc.scenario, tc.tool, tc.output)

			if len(issues) != 1 {
				t.Fatalf("expected 1 issue, got %d", len(issues))
			}

			issue := issues[0]

			// Verify all expected fields are populated
			if issue.Scenario != tc.expected.Scenario {
				t.Errorf("Scenario: expected %q, got %q", tc.expected.Scenario, issue.Scenario)
			}
			if issue.Tool != tc.expected.Tool {
				t.Errorf("Tool: expected %q, got %q", tc.expected.Tool, issue.Tool)
			}
			if issue.Category != tc.expected.Category {
				t.Errorf("Category: expected %q, got %q", tc.expected.Category, issue.Category)
			}
			if issue.File != tc.expected.File {
				t.Errorf("File: expected %q, got %q", tc.expected.File, issue.File)
			}
			if issue.Line != tc.expected.Line {
				t.Errorf("Line: expected %d, got %d", tc.expected.Line, issue.Line)
			}
			if issue.Column != tc.expected.Column {
				t.Errorf("Column: expected %d, got %d", tc.expected.Column, issue.Column)
			}
			if issue.Rule != tc.expected.Rule {
				t.Errorf("Rule: expected %q, got %q", tc.expected.Rule, issue.Rule)
			}
			// Severity check only if expected value is set
			if tc.expected.Severity != "" && issue.Severity != tc.expected.Severity {
				t.Errorf("Severity: expected %q, got %q", tc.expected.Severity, issue.Severity)
			}
			// Message should always be non-empty
			if issue.Message == "" {
				t.Error("Message should not be empty")
			}
		})
	}
}

// [REQ:TM-LS-003] Fuzz test for lint parser robustness
func FuzzParseLintOutput(f *testing.F) {
	// Seed corpus with known formats
	f.Add("test", "eslint", "file.go:10:5: Error [rule]")
	f.Add("scenario", "golangci-lint", "api/main.go:42:1: Comment missing (revive)")
	f.Add("test", "lint", "")
	f.Add("test", "lint", "malformed input without structure")

	f.Fuzz(func(t *testing.T, scenario, tool, output string) {
		// Parser must not panic on any input
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("ParseLintOutput panicked on input: scenario=%q tool=%q output=%q", scenario, tool, output)
			}
		}()

		issues := ParseLintOutput(scenario, tool, output)

		// Result should never be nil
		if issues == nil {
			t.Error("ParseLintOutput returned nil")
		}

		// All returned issues must have required fields
		for i, issue := range issues {
			if issue.Scenario != scenario {
				t.Errorf("Issue %d: scenario mismatch: expected %q, got %q", i, scenario, issue.Scenario)
			}
			if issue.Tool != tool {
				t.Errorf("Issue %d: tool mismatch: expected %q, got %q", i, tool, issue.Tool)
			}
			if issue.Category != "lint" {
				t.Errorf("Issue %d: expected category 'lint', got %q", i, issue.Category)
			}
			// File path should be non-empty for valid issues
			if issue.File == "" {
				t.Errorf("Issue %d: File should not be empty", i)
			}
		}
	})
}

// [REQ:TM-LS-004] Fuzz test for type parser robustness
func FuzzParseTypeOutput(f *testing.F) {
	// Seed corpus with known formats
	f.Add("test", "tsc", "src/App.tsx(10,5): error TS2304: Cannot find name 'foo'")
	f.Add("scenario", "go", "api/main.go:42:8: undefined: unknownFunc")
	f.Add("test", "type", "")
	f.Add("test", "type", "random text")

	f.Fuzz(func(t *testing.T, scenario, tool, output string) {
		// Parser must not panic on any input
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("ParseTypeOutput panicked on input: scenario=%q tool=%q output=%q", scenario, tool, output)
			}
		}()

		issues := ParseTypeOutput(scenario, tool, output)

		// Result should never be nil
		if issues == nil {
			t.Error("ParseTypeOutput returned nil")
		}

		// All returned issues must have required fields
		for i, issue := range issues {
			if issue.Scenario != scenario {
				t.Errorf("Issue %d: scenario mismatch: expected %q, got %q", i, scenario, issue.Scenario)
			}
			if issue.Tool != tool {
				t.Errorf("Issue %d: tool mismatch: expected %q, got %q", i, tool, issue.Tool)
			}
			if issue.Category != "type" {
				t.Errorf("Issue %d: expected category 'type', got %q", i, issue.Category)
			}
			// File path should be non-empty for valid issues
			if issue.File == "" {
				t.Errorf("Issue %d: File should not be empty", i)
			}
		}
	})
}
