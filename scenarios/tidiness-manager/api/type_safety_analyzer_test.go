package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTypeSafetyAnalyzer_TSConfig_AllPresent(t *testing.T) {
	tmpDir := t.TempDir()
	uiDir := filepath.Join(tmpDir, "ui")
	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		t.Fatal(err)
	}

	tsconfig := `{
  "compilerOptions": {
    // ╔══════════════════════════════════════════════════════════════════════════╗
    // ║  SAFETY-CRITICAL RULES - DO NOT REMOVE OR WEAKEN                         ║
    // ║  If you encounter type errors from these rules:                          ║
    // ║  ❌ DON'T: Use type assertions (as X) to silence errors                  ║
    // ║  These rules exist because UI crashes are the #1 production issue.       ║
    // ╚══════════════════════════════════════════════════════════════════════════╝
    "strict": true,
    "noUncheckedIndexedAccess": true
  }
}`
	if err := os.WriteFile(filepath.Join(uiDir, "tsconfig.json"), []byte(tsconfig), 0o644); err != nil {
		t.Fatal(err)
	}

	analyzer := NewTypeSafetyAnalyzer(tmpDir)
	result := analyzer.Analyze()

	if !result.TSConfigFound {
		t.Error("expected tsconfig found")
	}
	if !result.TSConfigStrict {
		t.Error("expected strict true")
	}
	if !result.TSConfigNoUnchecked {
		t.Error("expected noUncheckedIndexedAccess true")
	}
	if !result.TSConfigHasProtectiveComment {
		t.Error("expected protective comment detected")
	}
	// No TS_CONFIG_STRICT violations
	for _, v := range result.Violations {
		if v.RuleID == "TS_CONFIG_STRICT" {
			t.Errorf("unexpected TS_CONFIG_STRICT violation: %s", v.Title)
		}
	}
}

func TestTypeSafetyAnalyzer_TSConfig_MissingComment(t *testing.T) {
	tmpDir := t.TempDir()
	uiDir := filepath.Join(tmpDir, "ui")
	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		t.Fatal(err)
	}

	tsconfig := `{
  "compilerOptions": {
    "strict": true,
    "noUncheckedIndexedAccess": true
  }
}`
	if err := os.WriteFile(filepath.Join(uiDir, "tsconfig.json"), []byte(tsconfig), 0o644); err != nil {
		t.Fatal(err)
	}

	analyzer := NewTypeSafetyAnalyzer(tmpDir)
	result := analyzer.Analyze()

	if !result.TSConfigStrict {
		t.Error("expected strict true")
	}
	if result.TSConfigHasProtectiveComment {
		t.Error("expected protective comment NOT detected")
	}

	foundCommentViolation := false
	for _, v := range result.Violations {
		if v.RuleID == "TS_CONFIG_STRICT" && strings.Contains(v.Title, "protective comment") {
			foundCommentViolation = true
		}
	}
	if !foundCommentViolation {
		t.Error("expected violation about missing protective comments")
	}
}

func TestTypeSafetyAnalyzer_TSConfig_MissingStrict(t *testing.T) {
	tmpDir := t.TempDir()
	uiDir := filepath.Join(tmpDir, "ui")
	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		t.Fatal(err)
	}

	tsconfig := `{"compilerOptions": {"target": "ES2020"}}`
	if err := os.WriteFile(filepath.Join(uiDir, "tsconfig.json"), []byte(tsconfig), 0o644); err != nil {
		t.Fatal(err)
	}

	analyzer := NewTypeSafetyAnalyzer(tmpDir)
	result := analyzer.Analyze()

	if result.TSConfigStrict {
		t.Error("expected strict false")
	}

	foundStrictViolation := false
	for _, v := range result.Violations {
		if v.RuleID == "TS_CONFIG_STRICT" && strings.Contains(v.Title, "strict") {
			foundStrictViolation = true
		}
	}
	if !foundStrictViolation {
		t.Error("expected violation about missing strict mode")
	}
}

func TestTypeSafetyAnalyzer_TSConfig_JSONC(t *testing.T) {
	tmpDir := t.TempDir()
	uiDir := filepath.Join(tmpDir, "ui")
	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		t.Fatal(err)
	}

	tsconfig := `{
  // This is a JSONC comment
  "compilerOptions": {
    /* block comment */
    "strict": true,
    "noUncheckedIndexedAccess": true, // inline comment
    "target": "ES2020"
  }
}`
	if err := os.WriteFile(filepath.Join(uiDir, "tsconfig.json"), []byte(tsconfig), 0o644); err != nil {
		t.Fatal(err)
	}

	analyzer := NewTypeSafetyAnalyzer(tmpDir)
	result := analyzer.Analyze()

	if !result.TSConfigFound {
		t.Error("expected tsconfig found")
	}
	if !result.TSConfigStrict {
		t.Error("expected strict true after JSONC parsing")
	}
	if !result.TSConfigNoUnchecked {
		t.Error("expected noUncheckedIndexedAccess true after JSONC parsing")
	}
}

func TestTypeSafetyAnalyzer_ESLint_AllRulesPresent(t *testing.T) {
	tmpDir := t.TempDir()
	uiDir := filepath.Join(tmpDir, "ui")
	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		t.Fatal(err)
	}

	eslintConfig := `import tseslint from "typescript-eslint";

export default tseslint.config({
  rules: {
    // ════════════════════════════════════════════════════════════════════════
    // SAFETY-CRITICAL RULES - DO NOT REMOVE, DISABLE, OR WEAKEN
    // ════════════════════════════════════════════════════════════════════════

    // CRITICAL: Catches React Error #310
    "react-hooks/rules-of-hooks": "error",

    // CRITICAL: Prevents non-null assertion
    "@typescript-eslint/no-non-null-assertion": "error",

    // CRITICAL: Catches operations on any
    "@typescript-eslint/no-unsafe-member-access": "warn",
    "@typescript-eslint/no-unsafe-call": "warn",
    "@typescript-eslint/no-unsafe-argument": "warn",
    "@typescript-eslint/no-unsafe-assignment": "warn",
    "@typescript-eslint/no-unsafe-return": "warn",

    "@typescript-eslint/no-explicit-any": "error",

    // CRITICAL: Detects circular deps
    "import/no-cycle": "error",
  },
});`
	if err := os.WriteFile(filepath.Join(uiDir, "eslint.config.js"), []byte(eslintConfig), 0o644); err != nil {
		t.Fatal(err)
	}

	analyzer := NewTypeSafetyAnalyzer(tmpDir)
	result := analyzer.Analyze()

	if !result.ESLintConfigFound {
		t.Error("expected eslint config found")
	}
	if !result.ESLintHasHeaderComment {
		t.Error("expected header comment detected")
	}

	for _, rule := range result.ESLintSafetyRules {
		if !rule.Found {
			t.Errorf("expected rule %s to be found", rule.Rule)
		}
	}
}

func TestTypeSafetyAnalyzer_ESLint_MissingRules(t *testing.T) {
	tmpDir := t.TempDir()
	uiDir := filepath.Join(tmpDir, "ui")
	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		t.Fatal(err)
	}

	eslintConfig := `export default { rules: { "react-hooks/rules-of-hooks": "error" } };`
	if err := os.WriteFile(filepath.Join(uiDir, "eslint.config.js"), []byte(eslintConfig), 0o644); err != nil {
		t.Fatal(err)
	}

	analyzer := NewTypeSafetyAnalyzer(tmpDir)
	result := analyzer.Analyze()

	if !result.ESLintConfigFound {
		t.Error("expected eslint config found")
	}

	foundMissingViolation := false
	for _, v := range result.Violations {
		if v.RuleID == "ESLINT_SAFETY_RULES" && strings.Contains(v.Title, "Missing ESLint safety rules") {
			foundMissingViolation = true
		}
	}
	if !foundMissingViolation {
		t.Error("expected violation about missing rules")
	}
}

func TestTypeSafetyAnalyzer_ESLint_MissingHeaderComment(t *testing.T) {
	tmpDir := t.TempDir()
	uiDir := filepath.Join(tmpDir, "ui")
	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		t.Fatal(err)
	}

	eslintConfig := `export default {
  rules: {
    "react-hooks/rules-of-hooks": "error",
    "@typescript-eslint/no-non-null-assertion": "error",
    "@typescript-eslint/no-explicit-any": "error",
    "@typescript-eslint/no-unsafe-member-access": "warn",
    "@typescript-eslint/no-unsafe-call": "warn",
    "@typescript-eslint/no-unsafe-argument": "warn",
    "@typescript-eslint/no-unsafe-assignment": "warn",
    "@typescript-eslint/no-unsafe-return": "warn",
    "import/no-cycle": "error",
  },
};`
	if err := os.WriteFile(filepath.Join(uiDir, "eslint.config.js"), []byte(eslintConfig), 0o644); err != nil {
		t.Fatal(err)
	}

	analyzer := NewTypeSafetyAnalyzer(tmpDir)
	result := analyzer.Analyze()

	if result.ESLintHasHeaderComment {
		t.Error("expected header comment NOT detected")
	}

	foundHeaderViolation := false
	for _, v := range result.Violations {
		if v.RuleID == "ESLINT_SAFETY_RULES" && strings.Contains(v.Title, "header comment") {
			foundHeaderViolation = true
		}
	}
	if !foundHeaderViolation {
		t.Error("expected violation about missing header comment")
	}
}

func TestTypeSafetyAnalyzer_NoUIDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	analyzer := NewTypeSafetyAnalyzer(tmpDir)
	result := analyzer.Analyze()

	if result.TSConfigFound {
		t.Error("expected TSConfigFound=false when no ui/ dir")
	}
	if result.ESLintConfigFound {
		t.Error("expected ESLintConfigFound=false when no ui/ dir")
	}
	if len(result.Violations) != 0 {
		t.Errorf("expected no violations when no ui/ dir, got %d", len(result.Violations))
	}
}

func TestTypeSafetyAnalyzer_FixTSConfig(t *testing.T) {
	tmpDir := t.TempDir()
	uiDir := filepath.Join(tmpDir, "ui")
	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Start with minimal tsconfig
	tsconfig := `{
  "compilerOptions": {
    "target": "ES2020"
  }
}`
	tsconfigPath := filepath.Join(uiDir, "tsconfig.json")
	if err := os.WriteFile(tsconfigPath, []byte(tsconfig), 0o644); err != nil {
		t.Fatal(err)
	}

	analyzer := NewTypeSafetyAnalyzer(tmpDir)
	result, err := analyzer.FixTSConfig()
	if err != nil {
		t.Fatalf("FixTSConfig failed: %v", err)
	}

	if !result.TSConfigStrict {
		t.Error("expected strict true after fix")
	}
	if !result.TSConfigNoUnchecked {
		t.Error("expected noUncheckedIndexedAccess true after fix")
	}
	if !result.TSConfigHasProtectiveComment {
		t.Error("expected protective comment after fix")
	}

	// Read file and verify content
	fixed, err := os.ReadFile(tsconfigPath)
	if err != nil {
		t.Fatalf("cannot read fixed tsconfig: %v", err)
	}
	fixedContent := string(fixed)
	if !strings.Contains(fixedContent, "SAFETY-CRITICAL RULES") {
		t.Error("expected protective comment in fixed file")
	}
	if !strings.Contains(fixedContent, `"strict": true`) && !strings.Contains(fixedContent, `"strict":true`) {
		t.Error("expected strict: true in fixed file")
	}
}

func TestTypeSafetyAnalyzer_GoLintConfigViolations(t *testing.T) {
	tmpDir := t.TempDir()
	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := NewTypeSafetyAnalyzer(tmpDir).Analyze()

	var foundLintConfig, foundGoMod bool
	for _, v := range result.Violations {
		if v.RuleID == "GO_LINT_CONFIG_PRESENT" {
			foundLintConfig = true
		}
		if v.RuleID == "GO_MOD_PRESENT_FOR_API_OR_CLI" {
			foundGoMod = true
		}
	}
	if !foundLintConfig {
		t.Error("expected GO_LINT_CONFIG_PRESENT violation")
	}
	if !foundGoMod {
		t.Error("expected GO_MOD_PRESENT_FOR_API_OR_CLI violation")
	}
}

func TestTypeSafetyAnalyzer_GoLintRequiredLintersViolation(t *testing.T) {
	tmpDir := t.TempDir()
	apiDir := filepath.Join(tmpDir, "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "go.mod"), []byte("module example.com/test\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, ".golangci.yml"), []byte("linters:\n  enable:\n    - govet\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := NewTypeSafetyAnalyzer(tmpDir).Analyze()

	found := false
	for _, v := range result.Violations {
		if v.RuleID == "GO_LINT_REQUIRED_LINTERS" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected GO_LINT_REQUIRED_LINTERS violation")
	}
}

func TestTypeSafetyAnalyzer_TestingConfigStrictViolation(t *testing.T) {
	tmpDir := t.TempDir()
	uiDir := filepath.Join(tmpDir, "ui")
	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(uiDir, "package.json"), []byte(`{"scripts":{"build":"vite build"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, ".vrooli", "testing.json"), []byte(`{"lint":{"languages":{"node":{"enabled":true,"strict":false}}}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	result := NewTypeSafetyAnalyzer(tmpDir).Analyze()

	var strictViolation, buildViolation bool
	for _, v := range result.Violations {
		if v.RuleID == "TESTING_CONFIG_LINT_STRICT" {
			strictViolation = true
		}
		if v.RuleID == "NODE_BUILD_TYPECHECK" {
			buildViolation = true
		}
	}
	if !strictViolation {
		t.Error("expected TESTING_CONFIG_LINT_STRICT violation")
	}
	if !buildViolation {
		t.Error("expected NODE_BUILD_TYPECHECK violation")
	}
}

func TestTypeSafetyAnalyzer_ESLintTypedConfigViolation(t *testing.T) {
	tmpDir := t.TempDir()
	uiDir := filepath.Join(tmpDir, "ui")
	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		t.Fatal(err)
	}

	eslintConfig := `export default {
  rules: {
    // ════════════════════════════════════════════════════════════════════════
    // SAFETY-CRITICAL RULES - DO NOT REMOVE, DISABLE, OR WEAKEN
    // ════════════════════════════════════════════════════════════════════════
    // CRITICAL: Catches React Error #310
    "react-hooks/rules-of-hooks": "error",
    // CRITICAL: Prevents non-null assertion
    "@typescript-eslint/no-non-null-assertion": "error",
    "@typescript-eslint/no-explicit-any": "error",
    // CRITICAL: Catches operations on any
    "@typescript-eslint/no-unsafe-member-access": "warn",
    "@typescript-eslint/no-unsafe-call": "warn",
    "@typescript-eslint/no-unsafe-argument": "warn",
    "@typescript-eslint/no-unsafe-assignment": "warn",
    "@typescript-eslint/no-unsafe-return": "warn",
    // CRITICAL: Detects circular deps
    "import/no-cycle": "error"
  }
};`
	if err := os.WriteFile(filepath.Join(uiDir, "eslint.config.js"), []byte(eslintConfig), 0o644); err != nil {
		t.Fatal(err)
	}

	result := NewTypeSafetyAnalyzer(tmpDir).Analyze()

	found := false
	for _, v := range result.Violations {
		if v.RuleID == "ESLINT_TYPED_CONFIG" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected ESLINT_TYPED_CONFIG violation")
	}
}

func TestTypeSafetyAnalyzer_MakefileQualityViolation(t *testing.T) {
	tmpDir := t.TempDir()
	uiDir := filepath.Join(tmpDir, "ui")
	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(uiDir, "package.json"), []byte(`{"scripts":{"build":"tsc --noEmit && vite build"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "Makefile"), []byte("fmt-ui:\n\t@echo missing\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := NewTypeSafetyAnalyzer(tmpDir).Analyze()

	found := false
	for _, v := range result.Violations {
		if v.RuleID == "MAKEFILE_QUALITY_GATES" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected MAKEFILE_QUALITY_GATES violation")
	}
}

func TestStripJSONCComments(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single line comment",
			input:    `{"key": "value" // comment}`,
			expected: `{"key": "value" `,
		},
		{
			name:     "block comment",
			input:    `{"key": /* comment */ "value"}`,
			expected: `{"key":  "value"}`,
		},
		{
			name:     "string with slashes",
			input:    `{"url": "http://example.com"}`,
			expected: `{"url": "http://example.com"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripJSONCComments(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
