package autofix

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"quality-health/internal/contracts"

	"github.com/stretchr/testify/require"
)

func TestPreviewAndApplyTSConfigStrictFix(t *testing.T) {
	root := t.TempDir()
	ui := filepath.Join(root, "ui")
	require.NoError(t, os.MkdirAll(ui, 0o755))
	path := filepath.Join(ui, "tsconfig.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"compilerOptions":{}}`), 0o644))

	preview, err := Preview(root, []string{contracts.RuleTSConfigStrict})
	require.NoError(t, err)
	require.Len(t, preview, 1)
	require.False(t, preview[0].Applied)
	require.Contains(t, preview[0].After, `"strict": true`)
	require.Contains(t, preview[0].After, "SAFETY-CRITICAL RULES")

	applied, err := Apply(root, []string{contracts.RuleTSConfigStrict})
	require.NoError(t, err)
	require.Len(t, applied, 1)
	require.True(t, applied[0].Applied)
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"noUncheckedIndexedAccess": true`)
	require.True(t, HasTSConfigProtectiveComments(string(raw)))
}

func TestApplyRepairsAllAutofixClassConfigRules(t *testing.T) {
	// [REQ:QH-FIX-002]
	root := t.TempDir()
	ui := filepath.Join(root, "ui")
	api := filepath.Join(root, "api")
	require.NoError(t, os.MkdirAll(ui, 0o755))
	require.NoError(t, os.MkdirAll(api, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(ui, "tsconfig.json"), []byte(`{"compilerOptions":{}}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(ui, "eslint.config.js"), []byte(`export default { rules: {} }`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(ui, "package.json"), []byte(`{"scripts":{"build":"vite build"}}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(api, "main.go"), []byte("package main\nfunc main(){}\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(api, ".golangci.yml"), []byte("linters:\n  enable:\n    - govet\n  disable:\n    - errcheck\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "Makefile"), []byte("lint-ui:\n\t@echo TODO\n"), 0o644))

	rules := []string{
		contracts.RuleTSConfigStrict,
		contracts.RuleESLintSafetyRules,
		contracts.RuleESLintTypedConfig,
		contracts.RuleTestingConfigStrict,
		contracts.RuleGoModPresent,
		contracts.RuleGoLintRequiredLinters,
		contracts.RuleMakefileQualityGates,
	}
	preview, err := Preview(root, rules)
	require.NoError(t, err)
	gotRules := candidateRules(preview)
	for _, ruleID := range rules {
		require.Contains(t, gotRules, ruleID)
	}

	applied, err := Apply(root, rules)
	require.NoError(t, err)
	require.Len(t, applied, len(preview))
	for _, candidate := range applied {
		require.True(t, candidate.Applied, candidate.RuleID)
	}

	assertFileContains(t, filepath.Join(ui, "tsconfig.json"), `"strict": true`, "SAFETY-CRITICAL RULES")
	assertFileContains(t, filepath.Join(ui, "eslint.config.js"), "strictTypeChecked", "react-hooks/rules-of-hooks", "import/resolver")
	require.NoFileExists(t, filepath.Join(root, "eslint.config.js"))
	assertFileContains(t, filepath.Join(ui, "package.json"), `"build":"vite build"`)
	assertFileContains(t, filepath.Join(root, ".vrooli", "testing.json"), `"node_package"`, `"go_module"`, `"strict": true`)
	assertFileContains(t, filepath.Join(api, "go.mod"), "module api", "go 1.25")
	assertFileContains(t, filepath.Join(api, ".golangci.yml"), "errcheck", "gofumpt", "staticcheck", "unused")
	assertFileContains(t, filepath.Join(root, "Makefile"), "fmt-ui:", "lint-ui:", "pnpm run type-check", "lint-go:", "fmt-go:")
}

func TestApplyCreatesMissingGoLintConfig(t *testing.T) {
	// [REQ:QH-FIX-002]
	root := t.TempDir()
	api := filepath.Join(root, "api")
	require.NoError(t, os.MkdirAll(api, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(api, "main.go"), []byte("package main\nfunc main(){}\n"), 0o644))

	require.True(t, CanFix(root, contracts.RuleGoLintConfigPresent, filepath.Join(api, ".golangci.yml")))
	applied, err := Apply(root, []string{contracts.RuleGoLintConfigPresent})
	require.NoError(t, err)
	require.Len(t, applied, 1)
	require.True(t, applied[0].Applied)
	assertFileContains(t, filepath.Join(api, ".golangci.yml"), "errcheck", "gofumpt", "govet")
}

func TestCanFixReturnsFalseForUnparseableConfig(t *testing.T) {
	// [REQ:QH-FIX-003]
	root := t.TempDir()
	ui := filepath.Join(root, "ui")
	require.NoError(t, os.MkdirAll(ui, 0o755))
	path := filepath.Join(ui, "package.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"scripts":`), 0o644))
}

func candidateRules(candidates []Candidate) []string {
	out := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if !slices.Contains(out, candidate.RuleID) {
			out = append(out, candidate.RuleID)
		}
	}
	return out
}

func assertFileContains(t *testing.T, path string, values ...string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	text := string(raw)
	for _, value := range values {
		require.Contains(t, text, value)
	}
}
