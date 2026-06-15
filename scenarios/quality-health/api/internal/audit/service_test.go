package audit

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"quality-health/internal/contracts"
	"quality-health/internal/surfaces"

	"github.com/stretchr/testify/require"
)

type fakeDiscoverer struct{ inv surfaces.Inventory }

func (f fakeDiscoverer) Discover(context.Context, string, string, bool) (surfaces.Inventory, error) {
	return f.inv, nil
}

func TestAuditPreservesTSConfigProtectiveCommentContract(t *testing.T) {
	root := t.TempDir()
	ui := filepath.Join(root, "ui")
	require.NoError(t, os.MkdirAll(ui, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(ui, "tsconfig.json"), []byte(`{"compilerOptions":{"strict":true,"noUncheckedIndexedAccess":true}}`), 0o644))

	report := runFixtureAudit(t, root, []string{contracts.RuleTSConfigStrict})

	require.Equal(t, "failed", report.Status)
	require.Len(t, report.Findings, 1)
	require.Equal(t, contracts.RuleTSConfigStrict, report.Findings[0].RuleID)
	require.Contains(t, report.Findings[0].Message, "Missing protective comment")
}

func TestAuditFailsWeakESLintRuleLevels(t *testing.T) {
	root := t.TempDir()
	ui := filepath.Join(root, "ui")
	require.NoError(t, os.MkdirAll(ui, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(ui, "eslint.config.js"), []byte(`export default {
  rules: {
    // SAFETY-CRITICAL RULES - DO NOT REMOVE, DISABLE, OR WEAKEN
    // CRITICAL: hooks
    "react-hooks/rules-of-hooks": "warn",
    // CRITICAL: non null
    "@typescript-eslint/no-non-null-assertion": "off",
    // CRITICAL: any
    "@typescript-eslint/no-explicit-any": "error",
    // CRITICAL: unsafe
    "@typescript-eslint/no-unsafe-member-access": "warn",
    "@typescript-eslint/no-unsafe-call": "warn",
    "@typescript-eslint/no-unsafe-argument": "warn",
    "@typescript-eslint/no-unsafe-assignment": "warn",
    "@typescript-eslint/no-unsafe-return": "warn",
    // CRITICAL: cycle
    "import/no-cycle": "error",
  }
}`), 0o644))

	report := runFixtureAudit(t, root, []string{contracts.RuleESLintSafetyRules})

	require.Equal(t, "failed", report.Status)
	require.Len(t, report.Findings, 1)
	require.Equal(t, contracts.RuleESLintSafetyRules, report.Findings[0].RuleID)
	require.Contains(t, report.Findings[0].Observed, "weak react-hooks/rules-of-hooks=warn")
	require.Contains(t, report.Findings[0].Observed, "weak @typescript-eslint/no-non-null-assertion=off")
}

func TestAuditDetectsScenarioLevelQualityGateRules(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "ui"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "ui", "package.json"), []byte(`{"scripts":{"build":"vite build"}}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".vrooli", "testing.json"), []byte(`{"lint":{"handlers":{"node_package":{"enabled":true,"strict":false}}}}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "Makefile"), []byte("lint-ui:\n\t@echo TODO\n"), 0o644))

	report := runFixtureAudit(t, root, []string{contracts.RuleTestingConfigStrict, contracts.RuleMakefileQualityGates})

	var got []string
	for _, f := range report.Findings {
		got = append(got, f.RuleID)
	}
	require.Contains(t, got, contracts.RuleTestingConfigStrict)
	require.Contains(t, got, contracts.RuleMakefileQualityGates)
}

func TestAuditDetectsGoLintBaselineRules(t *testing.T) {
	root := t.TempDir()
	api := filepath.Join(root, "api")
	require.NoError(t, os.MkdirAll(api, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(api, "main.go"), []byte("package main\nfunc main(){}\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(api, ".golangci.yml"), []byte("linters:\n  enable:\n    - govet\n"), 0o644))

	report := runFixtureAudit(t, root, []string{contracts.RuleGoModPresent, contracts.RuleGoLintRequiredLinters})

	var got []string
	for _, f := range report.Findings {
		got = append(got, f.RuleID)
	}
	require.Contains(t, got, contracts.RuleGoModPresent)
	require.Contains(t, got, contracts.RuleGoLintRequiredLinters)
}

func runFixtureAudit(t *testing.T, root string, rules []string) Response {
	t.Helper()
	svc := &Service{
		Discoverer: fakeDiscoverer{inv: surfaces.Inventory{
			Scenario:   "fixture",
			TargetKind: "path",
			RootPath:   root,
			Surfaces: []surfaces.Surface{
				{ID: "ui", Kind: "ui", Language: "typescript", Framework: "react-vite", RootPath: filepath.Join(root, "ui"), Status: "known"},
				{ID: "api", Kind: "api", Language: "go", RootPath: filepath.Join(root, "api"), Status: "known"},
			},
		}},
		Now: func() time.Time { return time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC) },
	}
	report, err := svc.Audit(context.Background(), Request{Scenario: "fixture", RuleIDs: rules})
	require.NoError(t, err)
	return report
}
