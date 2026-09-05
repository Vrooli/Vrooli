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

func TestAuditFailsGoLinterDisabledNotJustMentioned(t *testing.T) {
	// [REQ:QH-AUDIT-006]
	root := t.TempDir()
	api := filepath.Join(root, "api")
	require.NoError(t, os.MkdirAll(api, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(api, "main.go"), []byte("package main\nfunc main(){}\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(api, ".golangci.yml"), []byte(`linters:
  enable:
    - govet
    - gofumpt
    - staticcheck
    - typecheck
    - unused
    - ineffassign
  disable:
    - errcheck # mentioning errcheck here must not satisfy the contract
`), 0o644))

	report := runAuditWithSurfaces(t, root, []surfaces.Surface{
		{ID: "api", Kind: "api", Language: "go", RootPath: api, Status: "known"},
	}, []string{contracts.RuleGoLintRequiredLinters})

	f := findingForRule(report, contracts.RuleGoLintRequiredLinters)
	require.Equal(t, contracts.RuleGoLintRequiredLinters, f.RuleID)
	require.Contains(t, f.Observed, "errcheck")
}

func TestAuditFailsCommentedMakefileGate(t *testing.T) {
	// [REQ:QH-AUDIT-006]
	root := t.TempDir()
	ui := filepath.Join(root, "ui")
	require.NoError(t, os.MkdirAll(ui, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(ui, "package.json"), []byte(`{"scripts":{"build":"vite build"}}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "Makefile"), []byte(`# lint-ui:
#	pnpm run lint
#	pnpm run type-check
fmt-ui:
	@echo lint:fix
`), 0o644))

	report := runAuditWithSurfaces(t, root, []surfaces.Surface{
		{ID: "ui", Kind: "ui", Language: "typescript", RootPath: ui, Status: "known"},
	}, []string{contracts.RuleMakefileQualityGates})

	f := findingForRule(report, contracts.RuleMakefileQualityGates)
	require.Equal(t, contracts.RuleMakefileQualityGates, f.RuleID)
	require.Contains(t, f.Observed, "lint-ui")
}

func TestAuditFailsCommentedESLintTypedConfig(t *testing.T) {
	// [REQ:QH-AUDIT-006]
	root := t.TempDir()
	ui := filepath.Join(root, "ui")
	require.NoError(t, os.MkdirAll(ui, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(ui, "eslint.config.js"), []byte(`export default {
  // strictTypeChecked
  // parserOptions: { project: true },
  rules: { "import/no-cycle": "error" },
}`), 0o644))

	report := runAuditWithSurfaces(t, root, []surfaces.Surface{
		{ID: "ui", Kind: "ui", Language: "typescript", RootPath: ui, Status: "known"},
	}, []string{contracts.RuleESLintTypedConfig})

	f := findingForRule(report, contracts.RuleESLintTypedConfig)
	require.Equal(t, contracts.RuleESLintTypedConfig, f.RuleID)
	require.Contains(t, f.Observed, "strictTypeChecked")
	require.Contains(t, f.Observed, "parserOptions.project")
}

func TestAuditReportsBareGoNolintSuppression(t *testing.T) {
	// [REQ:QH-SUP-001]
	root := t.TempDir()
	api := filepath.Join(root, "api")
	require.NoError(t, os.MkdirAll(api, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(api, "main.go"), []byte("package main\n\nfunc main() { //nolint:errcheck\n}\n"), 0o644))

	report := runAuditWithSurfaces(t, root, []surfaces.Surface{
		{ID: "api", Kind: "api", Language: "go", RootPath: api, Status: "known"},
	}, []string{contracts.RuleGoDangerousPatterns})

	f := findingForRule(report, contracts.RuleGoDangerousPatterns)
	require.Equal(t, contracts.RuleGoDangerousPatterns, f.RuleID)
	require.Equal(t, "warning", f.Severity)
	require.False(t, f.AutofixAvailable)
}

func TestAuditAllowsReasonedGoNolintSuppression(t *testing.T) {
	// [REQ:QH-SUP-001]
	root := t.TempDir()
	api := filepath.Join(root, "api")
	require.NoError(t, os.MkdirAll(api, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(api, "main.go"), []byte("package main\n\nfunc main() { //nolint:errcheck // third-party cleanup intentionally ignored here\n}\n"), 0o644))

	report := runAuditWithSurfaces(t, root, []surfaces.Surface{
		{ID: "api", Kind: "api", Language: "go", RootPath: api, Status: "known"},
	}, []string{contracts.RuleGoDangerousPatterns})

	require.Equal(t, Finding{}, findingForRule(report, contracts.RuleGoDangerousPatterns))
}

func TestAuditReportsBareTSSuppression(t *testing.T) {
	// [REQ:QH-SUP-001]
	root := t.TempDir()
	ui := filepath.Join(root, "ui")
	require.NoError(t, os.MkdirAll(ui, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(ui, "component.ts"), []byte("const value = {};\n// @ts-expect-error\nvalue.missing();\n"), 0o644))

	report := runAuditWithSurfaces(t, root, []surfaces.Surface{
		{ID: "ui", Kind: "ui", Language: "typescript", RootPath: ui, Status: "known"},
	}, []string{contracts.RuleTSDangerousPatterns})

	f := findingForRule(report, contracts.RuleTSDangerousPatterns)
	require.Equal(t, contracts.RuleTSDangerousPatterns, f.RuleID)
	require.Equal(t, "warning", f.Severity)
	require.Contains(t, f.Evidence, "bare TS suppressions=1")
	require.False(t, f.AutofixAvailable)
}

func TestAuditAllowsReasonedTSSuppression(t *testing.T) {
	// [REQ:QH-SUP-001]
	root := t.TempDir()
	ui := filepath.Join(root, "ui")
	require.NoError(t, os.MkdirAll(ui, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(ui, "component.ts"), []byte("const value = {};\n// @ts-expect-error third-party theme package lacks types\nvalue.missing();\n"), 0o644))

	report := runAuditWithSurfaces(t, root, []surfaces.Surface{
		{ID: "ui", Kind: "ui", Language: "typescript", RootPath: ui, Status: "known"},
	}, []string{contracts.RuleTSDangerousPatterns})

	require.Equal(t, Finding{}, findingForRule(report, contracts.RuleTSDangerousPatterns))
}

func TestAuditAutofixHonestyForMissingTSConfig(t *testing.T) {
	// [REQ:QH-FIX-003]
	root := t.TempDir()
	ui := filepath.Join(root, "ui")
	require.NoError(t, os.MkdirAll(ui, 0o755))

	report := runAuditWithSurfaces(t, root, []surfaces.Surface{
		{ID: "ui", Kind: "ui", Language: "typescript", RootPath: ui, Status: "known"},
	}, []string{contracts.RuleTSConfigStrict})

	f := findingForRule(report, contracts.RuleTSConfigStrict)
	require.Equal(t, contracts.RuleTSConfigStrict, f.RuleID)
	require.False(t, f.AutofixAvailable)
	require.Empty(t, f.AutofixCommand)
	require.NotContains(t, report.Summary, "autofixable")
}

func TestAuditRoutesTSSurfaceByLanguageNotName(t *testing.T) {
	// A TypeScript surface named "worker" (not "ui") must still be evaluated by
	// the TS pack — routing keys on language, not surface name.
	root := t.TempDir()
	worker := filepath.Join(root, "worker")
	require.NoError(t, os.MkdirAll(worker, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(worker, "tsconfig.json"), []byte(`{"compilerOptions":{"strict":true,"noUncheckedIndexedAccess":true}}`), 0o644))

	report := runAuditWithSurfaces(t, root, []surfaces.Surface{
		{ID: "worker", Kind: "worker", Language: "typescript", RootPath: worker, Status: "known"},
	}, []string{contracts.RuleTSConfigStrict})

	require.Len(t, report.Findings, 1)
	require.Equal(t, contracts.RuleTSConfigStrict, report.Findings[0].RuleID)
	require.Equal(t, "worker", report.Findings[0].SurfaceID)
	require.Equal(t, "typescript-static-quality", contractStatusFor(report, "worker").ContractID)
}

func TestAuditRoutesGoSurfaceByLanguageNotName(t *testing.T) {
	// A Go surface named "worker" (not api/cli) must still be evaluated by the
	// Go pack.
	root := t.TempDir()
	worker := filepath.Join(root, "worker")
	require.NoError(t, os.MkdirAll(worker, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(worker, "main.go"), []byte("package main\nfunc main(){}\n"), 0o644))

	report := runAuditWithSurfaces(t, root, []surfaces.Surface{
		{ID: "worker", Kind: "worker", Language: "go", RootPath: worker, Status: "known"},
	}, []string{contracts.RuleGoModPresent})

	require.Len(t, report.Findings, 1)
	require.Equal(t, contracts.RuleGoModPresent, report.Findings[0].RuleID)
	require.Equal(t, "go-static-quality", contractStatusFor(report, "worker").ContractID)
}

func TestAuditUncoveredSurfaceReportsGapNotPass(t *testing.T) {
	// A discovered surface with no applicable contract (e.g. rust) must report
	// `uncovered` + a coverage-gap info finding, never a clean pass.
	root := t.TempDir()
	rsvc := filepath.Join(root, "rsvc")
	require.NoError(t, os.MkdirAll(rsvc, 0o755))
	// Keep scenario-level gates clean (no node/go surfaces here) so we isolate
	// the coverage-gap behavior in the run status.
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".vrooli", "testing.json"), []byte(`{"lint":{"handlers":{}}}`), 0o644))

	report := runAuditWithSurfaces(t, root, []surfaces.Surface{
		{ID: "rsvc", Kind: "api", Language: "rust", RootPath: rsvc, Status: "known"},
	}, nil)

	c := contractStatusFor(report, "rsvc")
	require.Equal(t, "uncovered", c.Status)
	require.Equal(t, "", c.ContractID)

	gap := findingForRule(report, contracts.RuleCoverageGap)
	require.Equal(t, contracts.RuleCoverageGap, gap.RuleID)
	require.Equal(t, "info", gap.Severity)
	require.Equal(t, "coverage", gap.Category)
	require.Contains(t, gap.Message, "rsvc")
	require.Contains(t, gap.Message, "language=rust")
	// Info-only: run status is not failed.
	require.Equal(t, "passed", report.Status)
	require.Contains(t, report.Summary, "1 surface(s) uncovered")
}

func TestAuditNonGoApiSurfaceDoesNotRunGoEvaluator(t *testing.T) {
	// A non-Go surface named "api" (rust) must not run the Go evaluator.
	root := t.TempDir()
	api := filepath.Join(root, "api")
	require.NoError(t, os.MkdirAll(api, 0o755))

	report := runAuditWithSurfaces(t, root, []surfaces.Surface{
		{ID: "api", Kind: "api", Language: "rust", RootPath: api, Status: "known"},
	}, nil)

	for _, f := range report.Findings {
		require.NotEqual(t, contracts.RuleGoModPresent, f.RuleID)
		require.NotEqual(t, contracts.RuleGoLintConfigPresent, f.RuleID)
		require.NotEqual(t, contracts.RuleGoLintRequiredLinters, f.RuleID)
	}
	require.Equal(t, "uncovered", contractStatusFor(report, "api").Status)
}

func TestAuditJSOnlySurfaceSkipsTSConfig(t *testing.T) {
	// A JavaScript-only surface (no tsconfig) gets ESLint/build/suppression
	// rules; TS_CONFIG_STRICT self-skips without a spurious "not found" error.
	root := t.TempDir()
	web := filepath.Join(root, "web")
	require.NoError(t, os.MkdirAll(web, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(web, "package.json"), []byte(`{"scripts":{"build":"vite build"}}`), 0o644))

	report := runAuditWithSurfaces(t, root, []surfaces.Surface{
		{ID: "web", Kind: "ui", Language: "javascript", RootPath: web, Status: "known"},
	}, nil)

	var ruleIDs []string
	for _, f := range report.Findings {
		ruleIDs = append(ruleIDs, f.RuleID)
		require.NotEqual(t, contracts.RuleTSConfigStrict, f.RuleID, "TS_CONFIG_STRICT must self-skip for JS-only surfaces")
	}
	// ESLint-missing and planner-coverage rules still apply.
	require.Contains(t, ruleIDs, contracts.RuleESLintSafetyRules)
	require.Contains(t, ruleIDs, contracts.RuleTypecheckPlannerCoverage)
	require.Equal(t, "typescript-static-quality", contractStatusFor(report, "web").ContractID)
}

func TestAuditMaturityCappedWhenUncovered(t *testing.T) {
	// Mixed inventory: one clean covered surface + one uncovered surface. Run is
	// passed, but maturity is capped at L2 and names the uncovered surface.
	root := t.TempDir()
	api := filepath.Join(root, "api")
	require.NoError(t, os.MkdirAll(api, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(api, "main.go"), []byte("package main\nfunc main(){}\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(api, "go.mod"), []byte("module x\n\ngo 1.25\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(api, ".golangci.yml"), []byte("linters:\n  enable:\n    - errcheck\n    - gofumpt\n    - govet\n    - ineffassign\n    - staticcheck\n    - typecheck\n    - unused\n"), 0o644))
	// Keep the scenario-level Go gates clean so the run stays passed and we
	// isolate the maturity cap caused purely by the uncovered surface.
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".vrooli", "testing.json"), []byte(`{"lint":{"handlers":{"go_module":{"enabled":true,"strict":true}}}}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "Makefile"), []byte("lint-go:\n\tgolangci-lint run\nfmt-go:\n\tgofumpt -w .\n"), 0o644))
	rust := filepath.Join(root, "rustsvc")
	require.NoError(t, os.MkdirAll(rust, 0o755))

	report := runAuditWithSurfaces(t, root, []surfaces.Surface{
		{ID: "api", Kind: "api", Language: "go", RootPath: api, Status: "known"},
		{ID: "rustsvc", Kind: "api", Language: "rust", RootPath: rust, Status: "known"},
	}, nil)

	require.Equal(t, "passed", report.Status)
	require.Equal(t, 2, report.Maturity.Rung)
	require.Equal(t, "L2", report.Maturity.Label)
	require.Contains(t, report.Maturity.Rationale, "rustsvc")
	require.Contains(t, report.Summary, "1 surface(s) uncovered")
}

func TestAuditParityForUiApiCliFixture(t *testing.T) {
	// Parity: the classic ui/+api/+cli layout still fires every existing rule
	// (no findings dropped after the language-first loosening).
	root := t.TempDir()
	ui := filepath.Join(root, "ui")
	api := filepath.Join(root, "api")
	cli := filepath.Join(root, "cli")
	for _, d := range []string{ui, api, cli} {
		require.NoError(t, os.MkdirAll(d, 0o755))
	}
	// UI: tsconfig without protective comments -> TS_CONFIG_STRICT fires.
	require.NoError(t, os.WriteFile(filepath.Join(ui, "tsconfig.json"), []byte(`{"compilerOptions":{"strict":true,"noUncheckedIndexedAccess":true}}`), 0o644))
	// api/cli: go files without go.mod -> GO_MOD_PRESENT fires.
	require.NoError(t, os.WriteFile(filepath.Join(api, "main.go"), []byte("package main\nfunc main(){}\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(cli, "main.go"), []byte("package main\nfunc main(){}\n"), 0o644))

	report := runAuditWithSurfaces(t, root, []surfaces.Surface{
		{ID: "ui", Kind: "ui", Language: "typescript", Framework: "react-vite", RootPath: ui, Status: "known"},
		{ID: "api", Kind: "api", Language: "go", RootPath: api, Status: "known"},
		{ID: "cli", Kind: "cli", Language: "go", RootPath: cli, Status: "known"},
	}, []string{contracts.RuleTSConfigStrict, contracts.RuleGoModPresent})

	bySurface := map[string][]string{}
	for _, f := range report.Findings {
		bySurface[f.SurfaceID] = append(bySurface[f.SurfaceID], f.RuleID)
	}
	require.Contains(t, bySurface["ui"], contracts.RuleTSConfigStrict)
	require.Contains(t, bySurface["api"], contracts.RuleGoModPresent)
	require.Contains(t, bySurface["cli"], contracts.RuleGoModPresent)
	require.Equal(t, "typescript-static-quality", contractStatusFor(report, "ui").ContractID)
	require.Equal(t, "go-static-quality", contractStatusFor(report, "api").ContractID)
	require.Equal(t, "go-static-quality", contractStatusFor(report, "cli").ContractID)
}

func findingForRule(report Response, ruleID string) Finding {
	for _, f := range report.Findings {
		if f.RuleID == ruleID {
			return f
		}
	}
	return Finding{}
}

func contractStatusFor(report Response, surfaceID string) ContractEvaluation {
	for _, c := range report.Contracts {
		if c.SurfaceID == surfaceID {
			return c
		}
	}
	return ContractEvaluation{}
}

func runAuditWithSurfaces(t *testing.T, root string, surfs []surfaces.Surface, rules []string) Response {
	t.Helper()
	svc := &Service{
		Discoverer: fakeDiscoverer{inv: surfaces.Inventory{
			Scenario:   "fixture",
			TargetKind: "path",
			RootPath:   root,
			Surfaces:   surfs,
		}},
		Now: func() time.Time { return time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC) },
	}
	report, err := svc.Audit(context.Background(), Request{Scenario: "fixture", RuleIDs: rules})
	require.NoError(t, err)
	return report
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
