package golang

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLinter_Lint_NoDirectory(t *testing.T) {
	config := Config{
		Dir: "/nonexistent/path",
	}
	linter := New(config)

	result := linter.Lint(context.Background())

	if !result.Skipped {
		t.Error("expected skipped when directory doesn't exist")
	}
	if !result.Success {
		t.Error("expected success (skipped is success)")
	}
}

func TestLinter_Lint_NoGoMod(t *testing.T) {
	// Create temp dir without go.mod
	tmpDir, err := os.MkdirTemp("", "lint-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	config := Config{
		Dir: tmpDir,
	}
	linter := New(config)

	result := linter.Lint(context.Background())

	if !result.Skipped {
		t.Error("expected skipped when no go.mod")
	}
	if result.SkipReason != "no go.mod found" {
		t.Errorf("unexpected skip reason: %s", result.SkipReason)
	}
}

func TestLinter_HasCommand(t *testing.T) {
	// Test with custom lookup that returns error
	config := Config{
		Dir: "/tmp",
		CommandLookup: func(name string) (string, error) {
			if name == "existing" {
				return "/path/to/existing", nil
			}
			return "", errors.New("not found")
		},
	}
	linter := New(config)

	if !linter.hasCommand("existing") {
		t.Error("expected hasCommand to return true for existing command")
	}
	if linter.hasCommand("nonexistent") {
		t.Error("expected hasCommand to return false for nonexistent command")
	}
}

func TestLinter_ParseGolangciLintOutput(t *testing.T) {
	linter := New(Config{Dir: "/tmp"})

	// Test valid JSON output
	validJSON := `{"Issues": [
		{
			"FromLinter": "typecheck",
			"Text": "undefined: foo",
			"Severity": "error",
			"Pos": {"Filename": "main.go", "Line": 10, "Column": 5}
		},
		{
			"FromLinter": "gosimple",
			"Text": "unnecessary assignment",
			"Severity": "warning",
			"Pos": {"Filename": "utils.go", "Line": 20, "Column": 1}
		}
	]}`

	issues, typeErrors := linter.parseGolangciLintOutput([]byte(validJSON))

	if len(issues) != 2 {
		t.Errorf("expected 2 issues, got %d", len(issues))
	}
	if typeErrors != 1 {
		t.Errorf("expected 1 type error, got %d", typeErrors)
	}

	// Check first issue (type error)
	if issues[0].Source != "golangci-lint" {
		t.Errorf("expected source 'golangci-lint', got %s", issues[0].Source)
	}
	if issues[0].Severity != SeverityError {
		t.Errorf("expected SeverityError, got %v", issues[0].Severity)
	}

	// Check second issue (warning)
	if issues[1].Severity != SeverityWarning {
		t.Errorf("expected SeverityWarning, got %v", issues[1].Severity)
	}
}

func TestLinter_ParseGoVetOutput(t *testing.T) {
	linter := New(Config{Dir: "/tmp"})

	// Test valid go vet output
	vetOutput := `main.go:10:5: printf: Sprintf format %d has arg str of wrong type string
utils.go:20: unreachable code`

	issues := linter.parseGoVetOutput(vetOutput)

	if len(issues) != 2 {
		t.Errorf("expected 2 issues, got %d", len(issues))
	}

	// Check first issue (with column)
	if issues[0].File != "main.go" {
		t.Errorf("expected file 'main.go', got %s", issues[0].File)
	}
	if issues[0].Line != 10 {
		t.Errorf("expected line 10, got %d", issues[0].Line)
	}
	if issues[0].Column != 5 {
		t.Errorf("expected column 5, got %d", issues[0].Column)
	}

	// Check second issue (without column)
	if issues[1].File != "utils.go" {
		t.Errorf("expected file 'utils.go', got %s", issues[1].File)
	}
	if issues[1].Line != 20 {
		t.Errorf("expected line 20, got %d", issues[1].Line)
	}
}

func TestLinter_ParseGolangciLintOutput_InvalidJSON(t *testing.T) {
	linter := New(Config{Dir: "/tmp"})

	// Test invalid JSON
	issues, typeErrors := linter.parseGolangciLintOutput([]byte("not json"))

	if len(issues) != 0 {
		t.Errorf("expected 0 issues for invalid JSON, got %d", len(issues))
	}
	if typeErrors != 0 {
		t.Errorf("expected 0 type errors for invalid JSON, got %d", typeErrors)
	}
}

func TestLinter_WithLogger(t *testing.T) {
	config := Config{Dir: "/tmp"}
	linter := New(config, WithLogger(os.Stdout))

	if linter.logWriter != os.Stdout {
		t.Error("expected logWriter to be set to stdout")
	}
}

func TestNewLinter(t *testing.T) {
	config := Config{
		Dir: "/test/path",
		CommandLookup: func(name string) (string, error) {
			return "/bin/" + name, nil
		},
	}

	linter := New(config)

	if linter.config.Dir != "/test/path" {
		t.Errorf("expected Dir '/test/path', got %s", linter.config.Dir)
	}
}

func TestLinter_Integration_WithGoMod(t *testing.T) {
	// Create temp dir with go.mod but no linter available
	tmpDir, err := os.MkdirTemp("", "lint-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create go.mod
	goModContent := `module testmodule
go 1.21
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Use command lookup that returns nothing available
	config := Config{
		Dir: tmpDir,
		CommandLookup: func(name string) (string, error) {
			return "", errors.New("not found")
		},
	}
	linter := New(config)

	result := linter.Lint(context.Background())

	// Should skip because no tools available
	if !result.Skipped {
		t.Error("expected skipped when no tools available")
	}
	if result.SkipReason != "no Go linting tools available" {
		t.Errorf("unexpected skip reason: %s", result.SkipReason)
	}
}
