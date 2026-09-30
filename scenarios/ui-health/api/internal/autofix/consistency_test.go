package autofix

import (
	"path/filepath"
	"testing"

	"github.com/vrooli/maturity-go/assessment"
)

// TestRuntimeAutofixMatchesMaturityDeclaration is the contract guard for the
// auto-fix standard: every code the runtime can mark AutofixAvailable=true
// (FixClassFor == autofix) MUST be declared fix_class=auto + fixer_status=
// implemented in .vrooli/maturity.json. Otherwise assessment.ConsistencyWarnings
// would flag the runtime flag as contradicting the declaration (a finding that
// "reports a runtime autofix but is declared manual / pending"). Keeping the two
// in lockstep here means a future fixer addition that forgets the maturity flip
// fails this test instead of silently emitting consistency warnings in prod.
func TestRuntimeAutofixMatchesMaturityDeclaration(t *testing.T) {
	scenarioDir := scenarioDir(t)
	spec, err := assessment.LoadSpecFromScenario(scenarioDir)
	if err != nil {
		t.Fatalf("load maturity spec: %v", err)
	}

	autofixCodes := []string{
		RuleSlotDirMissing, RuleSlotParentDirMissing,
		RuleInteropHScreen, RuleInteropProtectiveComments,
		RuleStandardTSConfigStrict, RuleStandardI18nLocaleParity,
	}
	for _, code := range autofixCodes {
		if FixClassFor(code) != "autofix" {
			t.Fatalf("FixClassFor(%q) is not autofix; update this test's code list", code)
		}
		mapping, ok := spec.Findings[code]
		if !ok {
			t.Fatalf("maturity.json has no finding mapping for autofixable code %q", code)
		}
		if got := mapping.EffectiveFixClass(); got != assessment.FixClassAuto {
			t.Fatalf("maturity.json %q fix_class=%q, want auto (runtime can mark it AutofixAvailable)", code, got)
		}
		if got := mapping.EffectiveFixerStatus(); got != assessment.FixerStatusImplemented {
			t.Fatalf("maturity.json %q fixer_status=%q, want implemented (the fixer exists)", code, got)
		}
	}

	// Reverse direction: nothing in maturity.json may claim fixer_status=implemented
	// for a code the runtime has no fixer for (that would be an over-claim the
	// conformance rollup counts as implemented coverage that does not exist).
	fixable := map[string]bool{}
	for _, c := range autofixCodes {
		fixable[c] = true
	}
	for code, mapping := range spec.Findings {
		if mapping.EffectiveFixClass() == assessment.FixClassAuto &&
			mapping.EffectiveFixerStatus() == assessment.FixerStatusImplemented &&
			!fixable[code] {
			t.Fatalf("maturity.json declares %q auto/implemented but the runtime registers no fixer for it", code)
		}
	}
}

// scenarioDir resolves scenarios/ui-health from the test working directory
// (scenarios/ui-health/api/internal/autofix).
func scenarioDir(t *testing.T) string {
	t.Helper()
	wd, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("resolve working dir: %v", err)
	}
	return filepath.Clean(filepath.Join(wd, "..", "..", ".."))
}
