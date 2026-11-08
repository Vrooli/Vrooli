package assessment

import (
	"testing"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// scopeSpec has two capabilities: one that applies everywhere and one scoped
// to scenarios, so a package run can be distinguished from a scenario run.
func scopeSpec() Spec {
	return Spec{
		Version:  "2.0.0",
		Provider: "storage-manager",
		Phase:    "storage",
		Capabilities: []CapabilitySpec{
			{
				ID:    "declaration_accountability",
				Label: "Declaration Accountability",
				Levels: []Level{
					{ID: "L0", Name: "Unknown", CapabilitySummary: "Unknown."},
					{ID: "L1", Name: "Declared", CapabilitySummary: "Declared.", NextUnlock: "Reconcile."},
					{ID: "L2", Name: "Reconciled", CapabilitySummary: "Reconciled.", NextUnlock: "Govern."},
					{ID: "L3", Name: "Governed", CapabilitySummary: "Governed end to end."},
				},
			},
			{
				ID:        "scenario_conventions",
				Label:     "Scenario Conventions",
				AppliesTo: []string{"scenario"},
				Levels: []Level{
					{ID: "L0", Name: "Unknown", CapabilitySummary: "Unknown."},
					{ID: "L1", Name: "Present", CapabilitySummary: "Conventions present.", NextUnlock: "Clean."},
					{ID: "L2", Name: "Clean", CapabilitySummary: "Conventions clean."},
				},
			},
		},
		Findings: map[string]FindingMapping{
			"STORAGE_ACCOUNTABILITY_UNDECLARED": {
				CapabilityID:     "declaration_accountability",
				LocalLevelImpact: "L1",
				GlobalImpact:     ImpactAdvisory,
				Dimension:        "storage",
				SeverityDefault:  "SEVERITY_INFO",
				CleanRequirement: string(CleanRequirementRequired),
			},
			"MAKEFILE_QUALITY_GATES": {
				CapabilityID:     "scenario_conventions",
				LocalLevelImpact: "L1",
				GlobalImpact:     ImpactAdvisory,
				Dimension:        "storage",
				SeverityDefault:  "SEVERITY_ERROR",
				CleanRequirement: string(CleanRequirementRequired),
			},
		},
		Fallback: FallbackPolicy{
			CapabilityID:     "declaration_accountability",
			LocalLevelImpact: "L1",
			GlobalImpact:     ImpactAdvisory,
			Dimension:        "storage",
			SeverityDefault:  "SEVERITY_WARNING",
			CleanRequirement: string(CleanRequirementRequired),
		},
	}
}

func target(kind commonv1.ValidationTargetKind, id string) *commonv1.ValidationTarget {
	return &commonv1.ValidationTarget{Kind: kind, Id: id}
}

func levelOf(t *testing.T, results []LocalResult, capabilityID string) string {
	t.Helper()
	for _, result := range results {
		if result.CapabilityID == capabilityID {
			return result.CurrentLevel
		}
	}
	return ""
}

// The live regression this scoping exists to prevent: storage-manager reports
// on every resource, tool, and safeguard inside its own scenario run. One
// safeguard that declared no storage entries pulled storage-manager's own
// declaration_accountability ladder from L3 down to L0, even though
// storage-manager's own storage was fully governed.
func TestForeignSubjectDoesNotMoveTheRunTargetLadder(t *testing.T) {
	spec := scopeSpec()
	run := target(commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SCENARIO, "storage-manager")
	findings := []Finding{{
		Code:     "STORAGE_ACCOUNTABILITY_UNDECLARED",
		Severity: "SEVERITY_INFO",
		Subject:  target(commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SAFEGUARD, "clock"),
	}}

	got := CapabilityMaturityForTarget(spec, findings, run)
	if level := levelOf(t, got, "declaration_accountability"); level != "L3" {
		t.Fatalf("level = %q, want L3: a safeguard's declaration must not score the scenario's ladder", level)
	}
}

// The same finding about the run's own target must still score, or scoping
// would have traded a false pass for a different false pass.
func TestOwnSubjectStillMovesTheLadder(t *testing.T) {
	spec := scopeSpec()
	run := target(commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SCENARIO, "storage-manager")
	findings := []Finding{{
		Code:     "STORAGE_ACCOUNTABILITY_UNDECLARED",
		Severity: "SEVERITY_INFO",
		Subject:  target(commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SCENARIO, "storage-manager"),
	}}

	got := CapabilityMaturityForTarget(spec, findings, run)
	if level := levelOf(t, got, "declaration_accountability"); level != "L0" {
		t.Fatalf("level = %q, want L0", level)
	}
}

// A nil subject means "the run's own target" and must score. Providers that
// predate the subject field emit nothing else.
func TestNilSubjectScoresTheRunTarget(t *testing.T) {
	spec := scopeSpec()
	run := target(commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SCENARIO, "storage-manager")
	findings := []Finding{{Code: "STORAGE_ACCOUNTABILITY_UNDECLARED", Severity: "SEVERITY_INFO"}}

	got := CapabilityMaturityForTarget(spec, findings, run)
	if level := levelOf(t, got, "declaration_accountability"); level != "L0" {
		t.Fatalf("level = %q, want L0", level)
	}
}

// Same id, different kind: a resource named like the scenario is not the
// scenario. Comparing ids alone would silently re-introduce the conflation.
func TestSubjectMatchRequiresKindAndID(t *testing.T) {
	spec := scopeSpec()
	run := target(commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SCENARIO, "ollama")
	findings := []Finding{{
		Code:     "STORAGE_ACCOUNTABILITY_UNDECLARED",
		Severity: "SEVERITY_INFO",
		Subject:  target(commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_RESOURCE, "ollama"),
	}}

	got := CapabilityMaturityForTarget(spec, findings, run)
	if level := levelOf(t, got, "declaration_accountability"); level != "L3" {
		t.Fatalf("level = %q, want L3: resource:ollama is not scenario:ollama", level)
	}
}

// A scenario-only capability must not appear in a package run's standing. The
// live case: quality-health scored packages/api-core against MAKEFILE_QUALITY_GATES
// and coverage/testing.json, neither of which a shared library has.
func TestScenarioOnlyCapabilityIsOmittedForAPackageTarget(t *testing.T) {
	spec := scopeSpec()
	pkg := target(commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_PACKAGE, "api-core")

	got := CapabilityMaturityForTarget(spec, nil, pkg)
	if levelOf(t, got, "scenario_conventions") != "" {
		t.Fatalf("scenario_conventions must be omitted for a package target, got %+v", got)
	}
	if levelOf(t, got, "declaration_accountability") == "" {
		t.Fatal("declaration_accountability applies to every kind and must survive scoping")
	}
}

func TestScenarioOnlyCapabilityIsKeptForAScenarioTarget(t *testing.T) {
	spec := scopeSpec()
	run := target(commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SCENARIO, "demo")

	got := CapabilityMaturityForTarget(spec, nil, run)
	if levelOf(t, got, "scenario_conventions") == "" {
		t.Fatal("scenario_conventions must be scored for a scenario target")
	}
}

// Scoping must never produce an empty standing: that would read as "nothing to
// check here" when the real condition is a mis-declared spec.
func TestScopingNeverEmptiesTheCapabilitySet(t *testing.T) {
	spec := scopeSpec()
	for i := range spec.Capabilities {
		spec.Capabilities[i].AppliesTo = []string{"scenario"}
	}
	got := CapabilityMaturityForTarget(spec, nil, target(commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_DOCS, "docs"))
	if len(got) != len(spec.Capabilities) {
		t.Fatalf("got %d capabilities, want all %d retained rather than an empty standing", len(got), len(spec.Capabilities))
	}
}

// BuildProtoAssessment must scope by default. Every provider that predates the
// target model calls it without a Target, so the implicit scenario target is
// what actually fixes the fleet.
func TestBuildProtoAssessmentDerivesTheScenarioTarget(t *testing.T) {
	spec := scopeSpec()
	out, err := BuildProtoAssessment(BuildInput{
		Scenario: "storage-manager",
		Spec:     spec,
		Findings: []Finding{{
			Code:     "STORAGE_ACCOUNTABILITY_UNDECLARED",
			Severity: "SEVERITY_INFO",
			Subject:  target(commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SAFEGUARD, "clock"),
		}},
	})
	if err != nil {
		t.Fatalf("BuildProtoAssessment error = %v", err)
	}
	for _, capability := range out.GetCapabilities() {
		if capability.GetId() == "declaration_accountability" && capability.GetCurrentLevel() != "L3" {
			t.Fatalf("level = %q, want L3 with no explicit target passed", capability.GetCurrentLevel())
		}
	}
	// The excluded finding is still reported: scoping changes scoring, never
	// visibility.
	if len(out.GetFindings()) != 1 {
		t.Fatalf("findings = %d, want the out-of-scope finding still reported", len(out.GetFindings()))
	}
}

// A nil target cannot discriminate, so behavior must be identical to the
// pre-scoping engine rather than silently dropping every subjected finding.
func TestNilTargetPreservesLegacyScoring(t *testing.T) {
	spec := scopeSpec()
	findings := []Finding{{
		Code:     "STORAGE_ACCOUNTABILITY_UNDECLARED",
		Severity: "SEVERITY_INFO",
		Subject:  target(commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SAFEGUARD, "clock"),
	}}
	got := CapabilityMaturity(spec, findings)
	if level := levelOf(t, got, "declaration_accountability"); level != "L0" {
		t.Fatalf("level = %q, want L0 (legacy unscoped behavior)", level)
	}
}

func TestTargetKindNameRoundTripsEveryDeclaredKind(t *testing.T) {
	for _, name := range []string{"scenario", "resource", "tool", "safeguard", "team", "package", "control-plane", "docs", "project"} {
		kinds := descriptorTargetKinds([]string{name})
		if len(kinds) != 1 {
			t.Fatalf("descriptorTargetKinds(%q) = %v", name, kinds)
		}
		if got := TargetKindName(kinds[0]); got != name {
			t.Errorf("TargetKindName round-trip for %q = %q", name, got)
		}
	}
}

func outOfScopeAssessment(t *testing.T) *commonv1.MaturityAssessment {
	t.Helper()
	out, err := BuildProtoAssessment(BuildInput{
		Scenario: "storage-manager",
		Spec:     scopeSpec(),
		Findings: []Finding{
			{Code: "STORAGE_ACCOUNTABILITY_UNDECLARED", Severity: "SEVERITY_INFO", Subject: target(commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SAFEGUARD, "clock")},
			{Code: "STORAGE_ACCOUNTABILITY_UNDECLARED", Severity: "SEVERITY_WARNING"},
		},
	})
	if err != nil {
		t.Fatalf("BuildProtoAssessment error = %v", err)
	}
	return out
}

func capabilityCount(t *testing.T, a *commonv1.MaturityAssessment, id, severity string) int32 {
	t.Helper()
	for _, capability := range a.GetCapabilities() {
		if capability.GetId() == id {
			return capability.GetFindingsBySeverity()[severity]
		}
	}
	t.Fatalf("capability %q missing", id)
	return 0
}

// Wrapping an assessment in a validation response must not undo scoping: a
// capability's counts describe the findings that scored it, and the
// out-of-scope finding must not reappear in them.
func TestBuildValidationResponseKeepsScopedCapabilityCounts(t *testing.T) {
	response, err := BuildValidationResponse("storage-manager", outOfScopeAssessment(t), nil, nil)
	if err != nil {
		t.Fatalf("BuildValidationResponse error = %v", err)
	}
	got := response.GetAssessment()
	if n := capabilityCount(t, got, "declaration_accountability", "FINDING_SEVERITY_INFO"); n != 0 {
		t.Fatalf("out-of-scope INFO finding counted %d time(s) in the capability", n)
	}
	if n := capabilityCount(t, got, "declaration_accountability", "FINDING_SEVERITY_WARNING"); n != 1 {
		t.Fatalf("in-scope WARNING finding counted %d time(s), want 1", n)
	}
	if n := got.GetFindingsBySeverity()["FINDING_SEVERITY_INFO"]; n != 1 {
		t.Fatalf("assessment-level INFO count = %d, want the out-of-scope finding still visible", n)
	}
}

// The validator cannot know the run's target, so it bounds capability counts:
// subjectless findings always score, subject-attributed ones may not.
func TestValidateAssessmentBoundsCapabilityCountsByScoping(t *testing.T) {
	for name, tc := range map[string]struct {
		severity string
		count    int32
	}{
		"subjectless finding dropped":       {severity: "FINDING_SEVERITY_WARNING", count: 0},
		"count exceeds attributed findings": {severity: "FINDING_SEVERITY_INFO", count: 2},
	} {
		t.Run(name, func(t *testing.T) {
			a := outOfScopeAssessment(t)
			for _, capability := range a.GetCapabilities() {
				if capability.GetId() == "declaration_accountability" {
					capability.FindingsBySeverity[tc.severity] = tc.count
				}
			}
			if err := ValidateAssessment(a); err == nil {
				t.Fatalf("ValidateAssessment accepted %s=%d", tc.severity, tc.count)
			}
		})
	}
	a := outOfScopeAssessment(t)
	for _, capability := range a.GetCapabilities() {
		if capability.GetId() == "declaration_accountability" {
			capability.FindingsBySeverity["FINDING_SEVERITY_INFO"] = 1
		}
	}
	if err := ValidateAssessment(a); err != nil {
		t.Fatalf("a subject-attributed finding that scored (explicit target) must validate: %v", err)
	}
}
