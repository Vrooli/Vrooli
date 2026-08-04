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

// makefileGateFixture lays out a tree whose Makefile is missing the UI quality
// gates, which is what MAKEFILE_QUALITY_GATES exists to catch.
func makefileGateFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	ui := filepath.Join(root, "ui")
	require.NoError(t, os.MkdirAll(ui, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(ui, "package.json"), []byte(`{"scripts":{"build":"vite build"}}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "Makefile"), []byte("fmt-ui:\n\t@echo lint:fix\n"), 0o644))
	return root
}

func auditForTargetKind(t *testing.T, root, validationKind string) Response {
	t.Helper()
	svc := &Service{
		Discoverer: fakeDiscoverer{inv: surfaces.Inventory{
			Scenario:   "fixture",
			TargetKind: "path",
			RootPath:   root,
			Surfaces: []surfaces.Surface{
				{ID: "ui", Kind: "ui", Language: "typescript", RootPath: filepath.Join(root, "ui"), Status: "known"},
			},
		}},
		Now: func() time.Time { return time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC) },
	}
	report, err := svc.Audit(context.Background(), Request{
		Scenario:             "fixture",
		ValidationTargetKind: validationKind,
		RuleIDs:              []string{contracts.RuleMakefileQualityGates},
	})
	require.NoError(t, err)
	return report
}

func hasRule(report Response, ruleID string) bool {
	for _, f := range report.Findings {
		if f.RuleID == ruleID {
			return true
		}
	}
	return false
}

// A shared package has no Makefile quality gates by construction, so asserting
// them reports a defect the target cannot have. This failed a real
// `test-genie execute package:api-core quality` run at ERROR severity.
func TestScenarioContractRulesDoNotRunAgainstAPackageTarget(t *testing.T) {
	report := auditForTargetKind(t, makefileGateFixture(t), "package")
	require.False(t, hasRule(report, contracts.RuleMakefileQualityGates),
		"a package target must not be judged against scenario Makefile gates")
}

func TestScenarioContractRulesDoNotRunAgainstAControlPlaneTarget(t *testing.T) {
	report := auditForTargetKind(t, makefileGateFixture(t), "control-plane")
	require.False(t, hasRule(report, contracts.RuleMakefileQualityGates),
		"a control-plane target must not be judged against scenario Makefile gates")
}

// The guard must not cost scenario coverage: this is the case the rule is for.
func TestScenarioContractRulesStillRunAgainstAScenarioTarget(t *testing.T) {
	report := auditForTargetKind(t, makefileGateFixture(t), "scenario")
	require.True(t, hasRule(report, contracts.RuleMakefileQualityGates),
		"a scenario target must still be judged against scenario Makefile gates")
}

// An unresolved kind must never narrow coverage. Code Facts reports "path" for
// a scenario supplied by path, so a caller that cannot name the governance kind
// has to keep the pre-target-model behavior or scenarios silently lose rules.
func TestUnknownTargetKindKeepsEveryRule(t *testing.T) {
	report := auditForTargetKind(t, makefileGateFixture(t), "")
	require.True(t, hasRule(report, contracts.RuleMakefileQualityGates),
		"an unresolved target kind must not silently drop scenario rules")
}
