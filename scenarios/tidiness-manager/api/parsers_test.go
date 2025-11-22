package main

import (
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
