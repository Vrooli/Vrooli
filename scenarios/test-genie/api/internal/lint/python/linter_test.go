package python

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLinter_Lint_NoPythonFiles(t *testing.T) {
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
		t.Error("expected skipped when no Python files")
	}
	if !result.Success {
		t.Error("expected success (skipped is success)")
	}
	if result.SkipReason != "no Python files found" {
		t.Errorf("unexpected skip reason: %s", result.SkipReason)
	}
}

func TestLinter_HasPythonFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lint-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	config := Config{Dir: tmpDir}
	linter := New(config)

	// No Python files initially
	if linter.hasPythonFiles() {
		t.Error("expected no Python files")
	}

	// Create a Python file
	if err := os.WriteFile(filepath.Join(tmpDir, "test.py"), []byte("print('hello')"), 0644); err != nil {
		t.Fatal(err)
	}

	if !linter.hasPythonFiles() {
		t.Error("expected Python files to be found")
	}
}

func TestLinter_HasPythonFiles_SkipsDirectories(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lint-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create .venv directory with Python file
	venvDir := filepath.Join(tmpDir, ".venv")
	if err := os.MkdirAll(venvDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(venvDir, "test.py"), []byte("print('hello')"), 0644); err != nil {
		t.Fatal(err)
	}

	config := Config{Dir: tmpDir}
	linter := New(config)

	// Should not find Python files in .venv
	if linter.hasPythonFiles() {
		t.Error("expected no Python files (should skip .venv)")
	}
}

func TestLinter_HasMypyConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lint-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	config := Config{Dir: tmpDir}
	linter := New(config)

	// No mypy config
	if linter.hasMypyConfig() {
		t.Error("expected no mypy config")
	}

	// Create mypy.ini
	if err := os.WriteFile(filepath.Join(tmpDir, "mypy.ini"), []byte("[mypy]"), 0644); err != nil {
		t.Fatal(err)
	}

	if !linter.hasMypyConfig() {
		t.Error("expected mypy config to be found")
	}
}

func TestLinter_HasMypyConfig_PyprojectToml(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lint-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	config := Config{Dir: tmpDir}
	linter := New(config)

	// Create pyproject.toml with [tool.mypy]
	pyprojectContent := `[project]
name = "test"

[tool.mypy]
strict = true
`
	if err := os.WriteFile(filepath.Join(tmpDir, "pyproject.toml"), []byte(pyprojectContent), 0644); err != nil {
		t.Fatal(err)
	}

	if !linter.hasMypyConfig() {
		t.Error("expected mypy config to be found in pyproject.toml")
	}
}

func TestLinter_HasCommand(t *testing.T) {
	config := Config{
		Dir: "/tmp",
		CommandLookup: func(name string) (string, error) {
			if name == "ruff" {
				return "/path/to/ruff", nil
			}
			return "", errors.New("not found")
		},
	}
	linter := New(config)

	if !linter.hasCommand("ruff") {
		t.Error("expected hasCommand to return true for ruff")
	}
	if linter.hasCommand("flake8") {
		t.Error("expected hasCommand to return false for flake8")
	}
}

func TestLinter_ParseRuffOutput(t *testing.T) {
	linter := New(Config{Dir: "/tmp"})

	ruffJSON := `[
		{
			"code": "E501",
			"message": "Line too long",
			"filename": "test.py",
			"location": {"row": 10, "column": 80}
		},
		{
			"code": "F401",
			"message": "Unused import",
			"filename": "utils.py",
			"location": {"row": 1, "column": 1}
		}
	]`

	issues := linter.parseRuffOutput([]byte(ruffJSON))

	if len(issues) != 2 {
		t.Errorf("expected 2 issues, got %d", len(issues))
	}

	// Check first issue
	if issues[0].File != "test.py" {
		t.Errorf("expected file 'test.py', got %s", issues[0].File)
	}
	if issues[0].Line != 10 {
		t.Errorf("expected line 10, got %d", issues[0].Line)
	}
	if issues[0].Column != 80 {
		t.Errorf("expected column 80, got %d", issues[0].Column)
	}
	if issues[0].Rule != "E501" {
		t.Errorf("expected rule 'E501', got %s", issues[0].Rule)
	}
	if issues[0].Source != "ruff" {
		t.Errorf("expected source 'ruff', got %s", issues[0].Source)
	}
	if issues[0].Severity != SeverityWarning {
		t.Errorf("expected SeverityWarning, got %v", issues[0].Severity)
	}
}

func TestLinter_ParseRuffOutput_InvalidJSON(t *testing.T) {
	linter := New(Config{Dir: "/tmp"})

	issues := linter.parseRuffOutput([]byte("not json"))

	if len(issues) != 0 {
		t.Errorf("expected 0 issues for invalid JSON, got %d", len(issues))
	}
}

func TestLinter_ParseFlake8Output(t *testing.T) {
	linter := New(Config{Dir: "/tmp"})

	flake8Output := `test.py:10:80: E501 line too long (120 > 88 characters)
utils.py:1:1: F401 'os' imported but unused
invalid line without proper format`

	issues := linter.parseFlake8Output(flake8Output)

	if len(issues) != 2 {
		t.Errorf("expected 2 issues, got %d", len(issues))
	}

	// Check first issue
	if issues[0].File != "test.py" {
		t.Errorf("expected file 'test.py', got %s", issues[0].File)
	}
	if issues[0].Line != 10 {
		t.Errorf("expected line 10, got %d", issues[0].Line)
	}
	if issues[0].Column != 80 {
		t.Errorf("expected column 80, got %d", issues[0].Column)
	}
	if issues[0].Rule != "E501" {
		t.Errorf("expected rule 'E501', got %s", issues[0].Rule)
	}
	if issues[0].Source != "flake8" {
		t.Errorf("expected source 'flake8', got %s", issues[0].Source)
	}
}

func TestLinter_ParseMypyOutput(t *testing.T) {
	linter := New(Config{Dir: "/tmp"})

	mypyOutput := `test.py:10: error: Incompatible types in assignment
utils.py:20: error: Name 'foo' is not defined
Some note about the error
test.py:30: error: Missing return statement`

	issues := linter.parseMypyOutput(mypyOutput)

	if len(issues) != 3 {
		t.Errorf("expected 3 issues, got %d", len(issues))
	}

	// Check first issue
	if issues[0].File != "test.py" {
		t.Errorf("expected file 'test.py', got %s", issues[0].File)
	}
	if issues[0].Line != 10 {
		t.Errorf("expected line 10, got %d", issues[0].Line)
	}
	if issues[0].Source != "mypy" {
		t.Errorf("expected source 'mypy', got %s", issues[0].Source)
	}
	if issues[0].Severity != SeverityError {
		t.Errorf("expected SeverityError, got %v", issues[0].Severity)
	}
}

func TestLinter_WithLogger(t *testing.T) {
	config := Config{Dir: "/tmp"}
	linter := New(config, WithLogger(os.Stdout))

	if linter.logWriter != os.Stdout {
		t.Error("expected logWriter to be set to stdout")
	}
}

func TestLinter_Lint_NoToolsAvailable(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lint-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a Python file
	if err := os.WriteFile(filepath.Join(tmpDir, "test.py"), []byte("print('hello')"), 0644); err != nil {
		t.Fatal(err)
	}

	config := Config{
		Dir: tmpDir,
		CommandLookup: func(name string) (string, error) {
			return "", errors.New("not found")
		},
	}
	linter := New(config)

	result := linter.Lint(context.Background())

	if !result.Skipped {
		t.Error("expected skipped when no tools available")
	}
	if result.SkipReason != "no Python linting tools available" {
		t.Errorf("unexpected skip reason: %s", result.SkipReason)
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
