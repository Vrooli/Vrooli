package nodejs

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
	if result.SkipReason != "ui/ directory not found" {
		t.Errorf("unexpected skip reason: %s", result.SkipReason)
	}
}

func TestLinter_Lint_NoPackageJson(t *testing.T) {
	// Create temp dir without package.json
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
		t.Error("expected skipped when no package.json")
	}
	if result.SkipReason != "no package.json found" {
		t.Errorf("unexpected skip reason: %s", result.SkipReason)
	}
}

func TestLinter_HasCommand(t *testing.T) {
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

func TestLinter_HasTsConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lint-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	config := Config{Dir: tmpDir}
	linter := New(config)

	// No tsconfig
	if linter.hasTsConfig() {
		t.Error("expected no tsconfig")
	}

	// Create tsconfig.json
	if err := os.WriteFile(filepath.Join(tmpDir, "tsconfig.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	if !linter.hasTsConfig() {
		t.Error("expected tsconfig to be found")
	}
}

func TestLinter_HasEslintConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lint-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	config := Config{Dir: tmpDir}
	linter := New(config)

	// No eslint config
	if linter.hasEslintConfig() {
		t.Error("expected no eslint config")
	}

	// Create .eslintrc.js
	if err := os.WriteFile(filepath.Join(tmpDir, ".eslintrc.js"), []byte("module.exports = {}"), 0644); err != nil {
		t.Fatal(err)
	}

	if !linter.hasEslintConfig() {
		t.Error("expected eslint config to be found")
	}
}

func TestLinter_ParseTscOutput(t *testing.T) {
	linter := New(Config{Dir: "/tmp"})

	tscOutput := `src/App.tsx(10,5): error TS2322: Type 'string' is not assignable to type 'number'.
src/utils.ts(20,10): error TS2304: Cannot find name 'foo'.
Some other line without error format`

	issues := linter.parseTscOutput(tscOutput)

	if len(issues) != 2 {
		t.Errorf("expected 2 issues, got %d", len(issues))
	}

	// Check first issue
	if issues[0].File != "src/App.tsx" {
		t.Errorf("expected file 'src/App.tsx', got %s", issues[0].File)
	}
	if issues[0].Line != 10 {
		t.Errorf("expected line 10, got %d", issues[0].Line)
	}
	if issues[0].Column != 5 {
		t.Errorf("expected column 5, got %d", issues[0].Column)
	}
	if issues[0].Rule != "TS2322" {
		t.Errorf("expected rule 'TS2322', got %s", issues[0].Rule)
	}
	if issues[0].Source != "tsc" {
		t.Errorf("expected source 'tsc', got %s", issues[0].Source)
	}
	if issues[0].Severity != SeverityError {
		t.Errorf("expected SeverityError, got %v", issues[0].Severity)
	}
}

func TestLinter_ParseEslintOutput(t *testing.T) {
	linter := New(Config{Dir: "/tmp"})

	eslintJSON := `[
		{
			"filePath": "/project/src/App.tsx",
			"messages": [
				{
					"line": 10,
					"column": 5,
					"message": "Unexpected any",
					"ruleId": "@typescript-eslint/no-explicit-any",
					"severity": 2
				},
				{
					"line": 20,
					"column": 1,
					"message": "Missing return type",
					"ruleId": "@typescript-eslint/explicit-function-return-type",
					"severity": 1
				}
			]
		}
	]`

	issues := linter.parseEslintOutput([]byte(eslintJSON))

	if len(issues) != 2 {
		t.Errorf("expected 2 issues, got %d", len(issues))
	}

	// Check first issue (severity 2 = error)
	if issues[0].Severity != SeverityError {
		t.Errorf("expected SeverityError, got %v", issues[0].Severity)
	}
	if issues[0].Source != "eslint" {
		t.Errorf("expected source 'eslint', got %s", issues[0].Source)
	}

	// Check second issue (severity 1 = warning)
	if issues[1].Severity != SeverityWarning {
		t.Errorf("expected SeverityWarning, got %v", issues[1].Severity)
	}
}

func TestLinter_ParseEslintOutput_InvalidJSON(t *testing.T) {
	linter := New(Config{Dir: "/tmp"})

	issues := linter.parseEslintOutput([]byte("not json"))

	if len(issues) != 0 {
		t.Errorf("expected 0 issues for invalid JSON, got %d", len(issues))
	}
}

func TestLinter_FindTsc(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lint-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create node_modules/.bin/tsc
	binDir := filepath.Join(tmpDir, "node_modules", ".bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	tscPath := filepath.Join(binDir, "tsc")
	if err := os.WriteFile(tscPath, []byte("#!/bin/bash\n"), 0755); err != nil {
		t.Fatal(err)
	}

	config := Config{Dir: tmpDir}
	linter := New(config)

	found := linter.findTsc()
	if found != tscPath {
		t.Errorf("expected tsc path %s, got %s", tscPath, found)
	}
}

func TestLinter_FindTsc_NotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lint-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	config := Config{
		Dir: tmpDir,
		CommandLookup: func(name string) (string, error) {
			return "", errors.New("not found")
		},
	}
	linter := New(config)

	found := linter.findTsc()
	if found != "" {
		t.Errorf("expected empty string for not found tsc, got %s", found)
	}
}

func TestLinter_WithLogger(t *testing.T) {
	config := Config{Dir: "/tmp"}
	linter := New(config, WithLogger(os.Stdout))

	if linter.logWriter != os.Stdout {
		t.Error("expected logWriter to be set to stdout")
	}
}

func TestLinter_Lint_NoTools(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lint-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create package.json but no tsconfig or eslint config
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{"name":"test"}`), 0644); err != nil {
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
		t.Error("expected skipped when no tools configured")
	}
	if result.SkipReason != "no TypeScript or ESLint configuration found" {
		t.Errorf("unexpected skip reason: %s", result.SkipReason)
	}
}
