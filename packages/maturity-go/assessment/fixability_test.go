package assessment

import (
	"strings"
	"testing"
)

// withFix returns validSpec with its single finding's fixability fields set.
func withFix(class FixClass, status FixerStatus, reason string) Spec {
	spec := validSpec()
	m := spec.Findings["measures.uncovered-domain"]
	m.FixClass = class
	m.FixerStatus = status
	m.FixReason = reason
	spec.Findings["measures.uncovered-domain"] = m
	return spec
}

func TestValidateSpecAcceptsFixabilityVocabulary(t *testing.T) {
	cases := []struct {
		name   string
		class  FixClass
		status FixerStatus
		reason string
	}{
		{"auto pending", FixClassAuto, FixerStatusPending, ""},
		{"auto implemented", FixClassAuto, FixerStatusImplemented, ""},
		{"external default status", FixClassExternal, "", ""},
		{"manual with reason", FixClassManual, "", "needs human judgment"},
		{"absent (backward compatible)", "", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateSpec(withFix(tc.class, tc.status, tc.reason)); err != nil {
				t.Fatalf("ValidateSpec rejected valid fixability: %v", err)
			}
		})
	}
}

func TestValidateSpecRejectsInvalidFixability(t *testing.T) {
	cases := []struct {
		name   string
		class  FixClass
		status FixerStatus
		reason string
		want   string
	}{
		{"bad fix_class", FixClass("sometimes"), "", "", "fix_class"},
		{"bad fixer_status", FixClassAuto, FixerStatus("soon"), "", "fixer_status"},
		{"fixer_status on manual", FixClassManual, FixerStatusPending, "x", "fixer_status is only valid"},
		{"manual without reason", FixClassManual, "", "", "reason is required"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateSpec(withFix(tc.class, tc.status, tc.reason))
			if err == nil {
				t.Fatalf("ValidateSpec accepted invalid fixability")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q does not mention %q", err, tc.want)
			}
		})
	}
}

func TestEffectiveFixDefaults(t *testing.T) {
	// Absent fix_class is the conservative manual default with no fixer status.
	var m FindingMapping
	if got := m.EffectiveFixClass(); got != FixClassManual {
		t.Fatalf("absent EffectiveFixClass = %q, want manual", got)
	}
	if got := m.EffectiveFixerStatus(); got != "" {
		t.Fatalf("manual EffectiveFixerStatus = %q, want empty", got)
	}
	// Auto without an explicit status defaults to pending (a visible gap).
	auto := FindingMapping{FixClass: FixClassAuto}
	if got := auto.EffectiveFixerStatus(); got != FixerStatusPending {
		t.Fatalf("auto EffectiveFixerStatus = %q, want pending", got)
	}
}

func TestComputeAutofixCoverage(t *testing.T) {
	spec := validSpec()
	spec.Findings = map[string]FindingMapping{
		"a": {GlobalImpact: ImpactAdvisory, FixClass: FixClassAuto, FixerStatus: FixerStatusImplemented},
		"b": {GlobalImpact: ImpactAdvisory, FixClass: FixClassAuto, FixerStatus: FixerStatusPending},
		"c": {GlobalImpact: ImpactAdvisory, FixClass: FixClassExternal}, // defaults pending
		"d": {GlobalImpact: ImpactAdvisory, FixClass: FixClassManual, FixReason: "judgment"},
		"e": {GlobalImpact: ImpactAdvisory}, // absent → manual, not declared
	}
	cov := ComputeAutofixCoverage(spec)
	if cov.Total != 5 {
		t.Fatalf("Total = %d, want 5", cov.Total)
	}
	if cov.FixableUniverse != 3 {
		t.Fatalf("FixableUniverse = %d, want 3", cov.FixableUniverse)
	}
	if cov.Implemented != 1 {
		t.Fatalf("Implemented = %d, want 1", cov.Implemented)
	}
	if cov.Pending != 2 { // b + c(default pending)
		t.Fatalf("Pending = %d, want 2", cov.Pending)
	}
	if cov.Manual != 2 { // d + e(default)
		t.Fatalf("Manual = %d, want 2", cov.Manual)
	}
	if cov.Declared != 4 { // e is undeclared
		t.Fatalf("Declared = %d, want 4", cov.Declared)
	}
	if cov.DeclarationComplete {
		t.Fatalf("DeclarationComplete = true, want false (e is undeclared)")
	}
	// Implementation rate keeps pending in the denominator: 1/(1+2).
	if got := cov.ImplementationRate(); got < 0.33 || got > 0.34 {
		t.Fatalf("ImplementationRate = %v, want ~0.333", got)
	}
}

func TestComputeAutofixCoverageDeclarationComplete(t *testing.T) {
	spec := validSpec()
	spec.Findings = map[string]FindingMapping{
		"a": {GlobalImpact: ImpactAdvisory, FixClass: FixClassAuto, FixerStatus: FixerStatusImplemented},
		"b": {GlobalImpact: ImpactAdvisory, FixClass: FixClassManual, FixReason: "judgment"},
	}
	cov := ComputeAutofixCoverage(spec)
	if !cov.DeclarationComplete {
		t.Fatalf("DeclarationComplete = false, want true (all findings declared)")
	}
}

func TestConsistencyWarnings(t *testing.T) {
	spec := validSpec()
	spec.Findings = map[string]FindingMapping{
		"AUTO_DONE":    {GlobalImpact: ImpactAdvisory, FixClass: FixClassAuto, FixerStatus: FixerStatusImplemented},
		"AUTO_PENDING": {GlobalImpact: ImpactAdvisory, FixClass: FixClassAuto, FixerStatus: FixerStatusPending},
		"MANUAL_ONE":   {GlobalImpact: ImpactAdvisory, FixClass: FixClassManual, FixReason: "judgment"},
	}
	findings := []Finding{
		{Code: "AUTO_DONE", AutofixAvailable: true},     // consistent — no warning
		{Code: "AUTO_PENDING", AutofixAvailable: true},  // claims fix but pending → warn
		{Code: "MANUAL_ONE", AutofixAvailable: true},    // claims fix but manual → warn
		{Code: "AUTO_PENDING", AutofixAvailable: false}, // no claim → no warning
	}
	warnings := ConsistencyWarnings(spec, findings)
	if len(warnings) != 2 {
		t.Fatalf("got %d warnings, want 2: %v", len(warnings), warnings)
	}
	joined := strings.Join(warnings, "\n")
	if !strings.Contains(joined, "AUTO_PENDING") || !strings.Contains(joined, "MANUAL_ONE") {
		t.Fatalf("warnings missing expected codes: %v", warnings)
	}
}
