package rules_test

import (
	"os"
	"path/filepath"
	"testing"

	"quality-health/internal/autofix"
	"quality-health/internal/rules"
	"quality-health/internal/surfaces"

	"github.com/stretchr/testify/require"
)

func TestRegistryDeclaresFixClassEvaluatorAndFixers(t *testing.T) {
	// [REQ:QH-FIX-001]
	seen := map[string]bool{}
	for _, rule := range rules.Registry() {
		require.NotEmpty(t, rule.ID)
		require.False(t, seen[rule.ID], "duplicate rule id %s", rule.ID)
		seen[rule.ID] = true
		require.NotEmpty(t, rule.Title, rule.ID)
		require.NotEmpty(t, rule.Category, rule.ID)
		require.NotEmpty(t, rule.Severity, rule.ID)
		require.NotEmpty(t, rule.ContractID, rule.ID)
		require.NotEmpty(t, rule.FixClass, rule.ID)
		require.NotNil(t, rule.Evaluate, rule.ID)

		switch rule.FixClass {
		case rules.FixClassAutofix:
			root, path := defaultFindingPath(t, rule.ID)
			require.True(t, autofix.CanFix(root, rule.ID, path), rule.ID)
		case rules.FixClassDetectionOnly:
			require.NotEmpty(t, rule.FixReason, rule.ID)
			require.False(t, autofix.CanFix(t.TempDir(), rule.ID, ""), rule.ID)
		default:
			t.Fatalf("unknown fix class %q for %s", rule.FixClass, rule.ID)
		}
	}
}

func TestRegistryDerivesSurfaceAndScenarioApplicability(t *testing.T) {
	// [REQ:QH-FIX-001]
	ts := rules.SurfaceRules(surfaces.Surface{ID: "worker", Kind: "worker", Language: "typescript"})
	require.Equal(t, []string{
		rules.RuleTSConfigStrict,
		rules.RuleESLintSafetyRules,
		rules.RuleTSDangerousPatterns,
		rules.RuleESLintTypedConfig,
		rules.RuleTypecheckPlannerCoverage,
		rules.RuleUILazyChunkRecovery,
	}, ids(ts))

	js := rules.SurfaceRules(surfaces.Surface{ID: "web", Kind: "ui", Language: "javascript"})
	require.Equal(t, ids(ts), ids(js))

	goRules := rules.SurfaceRules(surfaces.Surface{ID: "worker", Kind: "worker", Language: "go"})
	require.Equal(t, []string{
		rules.RuleGoModPresent,
		rules.RuleGoLintConfigPresent,
		rules.RuleGoLintRequiredLinters,
		rules.RuleGoDangerousPatterns,
		rules.RuleScenarioPrivilegeBoundary,
		rules.RuleScenarioInteractiveBoundary,
	}, ids(goRules))

	scenarioRules := rules.ScenarioRules()
	require.Equal(t, []string{rules.RuleTestingConfigStrict, rules.RuleMakefileQualityGates, rules.RuleShellSyntaxLint}, ids(scenarioRules))
}

func ids(ruleList []rules.Rule) []string {
	out := make([]string, 0, len(ruleList))
	for _, rule := range ruleList {
		out = append(out, rule.ID)
	}
	return out
}

func defaultFindingPath(t *testing.T, ruleID string) (string, string) {
	t.Helper()
	root := t.TempDir()
	switch ruleID {
	case rules.RuleTSConfigStrict:
		path := filepath.Join(root, "tsconfig.json")
		write(t, path, `{"compilerOptions":{}}`)
		return root, path
	case rules.RuleESLintSafetyRules, rules.RuleESLintTypedConfig:
		path := filepath.Join(root, "eslint.config.js")
		write(t, path, `export default { rules: {} }`)
		return root, path
	case rules.RuleTypecheckPlannerCoverage:
		path := filepath.Join(root, "package.json")
		write(t, path, `{"scripts":{"build":"vite build"}}`)
		return root, path
	case rules.RuleTestingConfigStrict:
		write(t, filepath.Join(root, "ui", "package.json"), `{"scripts":{"build":"vite build"}}`)
		path := filepath.Join(root, ".vrooli", "testing.json")
		write(t, path, `{}`)
		return root, path
	case rules.RuleGoModPresent:
		write(t, filepath.Join(root, "api", "main.go"), "package main\nfunc main(){}\n")
		return root, filepath.Join(root, "api", "go.mod")
	case rules.RuleGoLintConfigPresent, rules.RuleGoLintRequiredLinters:
		write(t, filepath.Join(root, "api", "main.go"), "package main\nfunc main(){}\n")
		path := filepath.Join(root, "api", ".golangci.yml")
		write(t, path, "linters:\n  enable:\n    - govet\n")
		return root, path
	case rules.RuleScenarioInteractiveBoundary:
		path := filepath.Join(root, "main.go")
		write(t, path, "package main\nimport \"os\"\nfunc main() { os.Open(\"/dev/tty\") }\n")
		return root, path
	case rules.RuleMakefileQualityGates:
		write(t, filepath.Join(root, "ui", "package.json"), `{"scripts":{"build":"vite build"}}`)
		path := filepath.Join(root, "Makefile")
		write(t, path, "lint-ui:\n\t@echo TODO\n")
		return root, path
	default:
		return root, ""
	}
}

func TestLazyChunkRecoveryRule(t *testing.T) {
	rule, ok := rules.ByID(rules.RuleUILazyChunkRecovery)
	require.True(t, ok)
	evaluate := func(root string) []rules.Finding {
		return rule.Evaluate(rules.EvalContext{Surface: surfaces.Surface{ID: "ui", Kind: "ui", Language: "typescript", RootPath: root}})
	}
	appTsx := "import { lazy } from \"react\";\nconst Page = lazy(() => import(\"./Page\"));\n"

	t.Run("flags a Vite UI that code-splits without a guard", func(t *testing.T) {
		root := t.TempDir()
		write(t, filepath.Join(root, "vite.config.ts"), "export default {}\n")
		write(t, filepath.Join(root, "src", "App.tsx"), appTsx)

		findings := evaluate(root)
		require.Len(t, findings, 1)
		require.Equal(t, rules.RuleUILazyChunkRecovery, findings[0].RuleID)
		require.Contains(t, findings[0].Evidence, "App.tsx")
	})

	t.Run("passes once the reload guard is installed", func(t *testing.T) {
		root := t.TempDir()
		write(t, filepath.Join(root, "vite.config.ts"), "export default {}\n")
		write(t, filepath.Join(root, "src", "App.tsx"), appTsx)
		write(t, filepath.Join(root, "src", "main.tsx"), "import { installChunkReloadGuard } from \"@vrooli/api-base\";\ninstallChunkReloadGuard();\n")

		require.Empty(t, evaluate(root))
	})

	t.Run("accepts a direct vite:preloadError handler", func(t *testing.T) {
		root := t.TempDir()
		write(t, filepath.Join(root, "vite.config.ts"), "export default {}\n")
		write(t, filepath.Join(root, "src", "App.tsx"), appTsx)
		write(t, filepath.Join(root, "src", "main.tsx"), "window.addEventListener(\"vite:preloadError\", () => window.location.reload());\n")

		require.Empty(t, evaluate(root))
	})

	t.Run("ignores surfaces without lazy imports, test-only lazy, or non-Vite builds", func(t *testing.T) {
		noLazy := t.TempDir()
		write(t, filepath.Join(noLazy, "vite.config.ts"), "export default {}\n")
		write(t, filepath.Join(noLazy, "src", "App.tsx"), "export const App = () => null;\n")
		require.Empty(t, evaluate(noLazy))

		testOnly := t.TempDir()
		write(t, filepath.Join(testOnly, "vite.config.ts"), "export default {}\n")
		write(t, filepath.Join(testOnly, "src", "App.test.tsx"), appTsx)
		require.Empty(t, evaluate(testOnly))

		noVite := t.TempDir()
		write(t, filepath.Join(noVite, "src", "App.tsx"), appTsx)
		require.Empty(t, evaluate(noVite))
	})
}

func write(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}
