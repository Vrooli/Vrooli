package validation

import (
	"testing"

	"github.com/vrooli/measures-go/manifestscan"
)

// findingBy returns the first finding with the given rule id, or nil.
func findingBy(rep Report, ruleID string) *Finding {
	for i := range rep.Findings {
		if rep.Findings[i].RuleID == ruleID {
			return &rep.Findings[i]
		}
	}
	return nil
}

func domainBy(rep Report, name string) *DomainCoverage {
	for i := range rep.Domains {
		if rep.Domains[i].Domain == name {
			return &rep.Domains[i]
		}
	}
	return nil
}

func TestClassify_CoveredDomainPasses(t *testing.T) {
	rep := Classify(Inputs{
		Scenario: "swarm-manager",
		Domains:  []DerivedDomain{{Name: "backlog", Stateful: true}},
		Measures: []HarvestedMeasure{
			{Name: "backlog.completed", Domain: "backlog", Effect: "read", Tier: manifestscan.TierFull, QuestionCount: 3},
		},
	})
	if !rep.Passed {
		t.Fatalf("expected passed; findings=%+v", rep.Findings)
	}
	d := domainBy(rep, "backlog")
	if d == nil || d.Status != StatusCovered {
		t.Fatalf("backlog: want covered, got %+v", d)
	}
	if d.MeasureCount != 1 || d.Tier != manifestscan.TierFull {
		t.Fatalf("backlog: want 1 measure / full tier, got count=%d tier=%s", d.MeasureCount, d.Tier)
	}
}

func TestClassify_UncoveredStatefulIsError(t *testing.T) {
	rep := Classify(Inputs{
		Scenario: "swarm-manager",
		Domains:  []DerivedDomain{{Name: "captures", Stateful: true}},
	})
	if rep.Passed {
		t.Fatal("expected fail (uncovered stateful domain)")
	}
	f := findingBy(rep, "measures.uncovered-domain")
	if f == nil || f.Severity != SeverityError {
		t.Fatalf("want uncovered-domain ERROR, findings=%+v", rep.Findings)
	}
	if d := domainBy(rep, "captures"); d == nil || d.Status != StatusUncovered {
		t.Fatalf("captures: want uncovered, got %+v", d)
	}
}

func TestClassify_WaivedDomainPasses(t *testing.T) {
	rep := Classify(Inputs{
		Scenario: "swarm-manager",
		Domains:  []DerivedDomain{{Name: "queue", Stateful: true}},
		Omitted:  []manifestscan.Omission{{Domain: "queue", Reason: "ephemeral; no historical value"}},
	})
	if !rep.Passed {
		t.Fatalf("expected passed (waived); findings=%+v", rep.Findings)
	}
	d := domainBy(rep, "queue")
	if d == nil || d.Status != StatusWaived || d.WaiverReason == "" {
		t.Fatalf("queue: want waived w/ reason, got %+v", d)
	}
}

func TestClassify_StaleWaiverIsWarning(t *testing.T) {
	// Waiver points at a domain that is NOT stateful (not in the derived set,
	// not overridden stateful) -> stale -> WARNING, does not fail the verdict.
	rep := Classify(Inputs{
		Scenario: "swarm-manager",
		Domains:  []DerivedDomain{{Name: "backlog", Stateful: true}},
		Measures: []HarvestedMeasure{{Name: "backlog.completed", Domain: "backlog", Tier: manifestscan.TierFull}},
		Omitted:  []manifestscan.Omission{{Domain: "nonexistent", Reason: "stale"}},
	})
	if !rep.Passed {
		t.Fatalf("stale waiver must not fail the verdict; findings=%+v", rep.Findings)
	}
	f := findingBy(rep, "measures.stale-waiver")
	if f == nil || f.Severity != SeverityWarning {
		t.Fatalf("want stale-waiver WARNING, findings=%+v", rep.Findings)
	}
}

func TestClassify_StatelessDomainNotExpected(t *testing.T) {
	rep := Classify(Inputs{
		Scenario: "swarm-manager",
		Domains: []DerivedDomain{
			{Name: "settings", Stateful: false, Note: "stateless configuration domain"},
		},
	})
	if !rep.Passed {
		t.Fatalf("stateless domain must not fail; findings=%+v", rep.Findings)
	}
	d := domainBy(rep, "settings")
	if d == nil || d.Status != StatusNotExpected {
		t.Fatalf("settings: want not_expected, got %+v", d)
	}
	if len(rep.Findings) != 0 {
		t.Fatalf("stateless domain should emit no findings, got %+v", rep.Findings)
	}
}

func TestClassify_OverrideForcesNonStateful(t *testing.T) {
	// graph is derived stateful, but an override marks it non-stateful: it must
	// become not_expected (no ERROR) rather than uncovered.
	rep := Classify(Inputs{
		Scenario:  "swarm-manager",
		Domains:   []DerivedDomain{{Name: "graph", Stateful: true}},
		Overrides: []manifestscan.DomainOverride{{Domain: "graph", Stateful: false, Reason: "structural-only, no rows"}},
	})
	if !rep.Passed {
		t.Fatalf("override to non-stateful must clear the gap; findings=%+v", rep.Findings)
	}
	d := domainBy(rep, "graph")
	if d == nil || d.Status != StatusNotExpected || d.Note == "" {
		t.Fatalf("graph: want not_expected w/ note, got %+v", d)
	}
}

func TestClassify_WorstTierAggregation(t *testing.T) {
	rep := Classify(Inputs{
		Scenario: "s",
		Domains:  []DerivedDomain{{Name: "backlog", Stateful: true}},
		Measures: []HarvestedMeasure{
			{Name: "backlog.a", Domain: "backlog", Tier: manifestscan.TierFull},
			{Name: "backlog.b", Domain: "backlog", Tier: manifestscan.TierPartial, TierNote: "window param not canonical"},
		},
	})
	d := domainBy(rep, "backlog")
	if d == nil || d.Tier != manifestscan.TierPartial {
		t.Fatalf("want worst tier=partial, got %+v", d)
	}
	// partial tier emits an INFO advisory, not a failure.
	if rep.Passed != true {
		t.Fatal("partial tier must not fail the verdict")
	}
	if f := findingBy(rep, "measures.tier-partial"); f == nil || f.Severity != SeverityInfo {
		t.Fatalf("want tier-partial INFO, findings=%+v", rep.Findings)
	}
}

func TestClassify_FallbackTierIsWarning(t *testing.T) {
	rep := Classify(Inputs{
		Scenario: "s",
		Domains:  []DerivedDomain{{Name: "backlog", Stateful: true}},
		Measures: []HarvestedMeasure{{Name: "backlog.a", Domain: "backlog", Tier: manifestscan.TierFallback}},
	})
	if !rep.Passed {
		t.Fatal("fallback tier is a warning, not a failure")
	}
	if f := findingBy(rep, "measures.tier-fallback"); f == nil || f.Severity != SeverityWarning {
		t.Fatalf("want tier-fallback WARNING, findings=%+v", rep.Findings)
	}
}

func TestClassify_MalformedDeclarationIsError(t *testing.T) {
	rep := Classify(Inputs{
		Scenario: "s",
		Domains:  []DerivedDomain{{Name: "backlog", Stateful: true}},
		Measures: []HarvestedMeasure{
			{Name: "backlog.broken", Domain: "backlog", AssembleErr: "manifest param \"foo\" has no matching field"},
		},
	})
	if rep.Passed {
		t.Fatal("malformed declaration must fail the verdict")
	}
	if f := findingBy(rep, "measures.malformed-declaration"); f == nil || f.Severity != SeverityError {
		t.Fatalf("want malformed-declaration ERROR, findings=%+v", rep.Findings)
	}
	// The broken measure leaves the (still stateful) domain uncovered, so an
	// uncovered-domain ERROR is also expected.
	if f := findingBy(rep, "measures.uncovered-domain"); f == nil {
		t.Fatalf("want uncovered-domain ERROR for the domain the broken measure failed to cover")
	}
}

func TestClassify_ExtraMeasureOnNonStatefulIsFine(t *testing.T) {
	// A measure declared on a domain not in the expected set is over-delivery:
	// covered, no finding, passes.
	rep := Classify(Inputs{
		Scenario: "s",
		Measures: []HarvestedMeasure{{Name: "extra.thing", Domain: "extra", Tier: manifestscan.TierFull}},
	})
	if !rep.Passed {
		t.Fatalf("extra measure must not fail; findings=%+v", rep.Findings)
	}
	if d := domainBy(rep, "extra"); d == nil || d.Status != StatusCovered {
		t.Fatalf("extra: want covered, got %+v", d)
	}
}

// --- Phase 2: mode-aware expectation --------------------------------------

func TestClassify_Conformant_AdditiveStatefulOverrideIsIllegal(t *testing.T) {
	// A conformant scenario (v1/domain/ folder) declares a NEW stateful domain
	// via measures.domains[] that has no proto in the folder -> ERROR.
	rep := Classify(Inputs{
		Scenario:  "swarm-manager",
		Mode:      ModeConformant,
		Domains:   []DerivedDomain{{Name: "backlog", Stateful: true}},
		Measures:  []HarvestedMeasure{{Name: "backlog.completed", Domain: "backlog", Tier: manifestscan.TierFull}},
		Overrides: []manifestscan.DomainOverride{{Domain: "smuggled", Stateful: true}},
	})
	if rep.Passed {
		t.Fatalf("additive stateful override in conformant mode must fail; findings=%+v", rep.Findings)
	}
	f := findingBy(rep, "measures.illegal-domain-declaration")
	if f == nil || f.Severity != SeverityError {
		t.Fatalf("want illegal-domain-declaration ERROR, findings=%+v", rep.Findings)
	}
	// The smuggled domain must NOT also surface as a separate uncovered/known
	// domain (the illegal declaration is the single actionable finding).
	if d := domainBy(rep, "smuggled"); d != nil {
		t.Fatalf("smuggled domain should not be classified as a known domain, got %+v", d)
	}
}

func TestClassify_Conformant_DowngradeOverrideIsLegal(t *testing.T) {
	// Marking an EXISTING folder domain non-stateful is a legal down-grade even in
	// conformant mode (a conformant scenario legitimately has non-countable protos).
	rep := Classify(Inputs{
		Scenario:  "swarm-manager",
		Mode:      ModeConformant,
		Domains:   []DerivedDomain{{Name: "graph", Stateful: true}},
		Overrides: []manifestscan.DomainOverride{{Domain: "graph", Stateful: false, Reason: "structural-only"}},
	})
	if !rep.Passed {
		t.Fatalf("down-grade override must stay legal; findings=%+v", rep.Findings)
	}
	if f := findingBy(rep, "measures.illegal-domain-declaration"); f != nil {
		t.Fatalf("down-grade must not be illegal, got %+v", f)
	}
	if d := domainBy(rep, "graph"); d == nil || d.Status != StatusNotExpected {
		t.Fatalf("graph: want not_expected, got %+v", d)
	}
}

func TestClassify_Conformant_WaiverIsLegal(t *testing.T) {
	// A waiver on a stateful folder domain is legal in conformant mode.
	rep := Classify(Inputs{
		Scenario: "swarm-manager",
		Mode:     ModeConformant,
		Domains:  []DerivedDomain{{Name: "capture", Stateful: true}},
		Omitted:  []manifestscan.Omission{{Domain: "capture", Reason: "ephemeral; no event"}},
	})
	if !rep.Passed {
		t.Fatalf("waiver must stay legal in conformant mode; findings=%+v", rep.Findings)
	}
	if f := findingBy(rep, "measures.illegal-domain-declaration"); f != nil {
		t.Fatalf("waiver must not be illegal, got %+v", f)
	}
	if d := domainBy(rep, "capture"); d == nil || d.Status != StatusWaived {
		t.Fatalf("capture: want waived, got %+v", d)
	}
}

func TestClassify_Conformant_NoFallbackAdvisory(t *testing.T) {
	rep := Classify(Inputs{
		Scenario: "swarm-manager",
		Mode:     ModeConformant,
		Domains:  []DerivedDomain{{Name: "backlog", Stateful: true}},
		Measures: []HarvestedMeasure{{Name: "backlog.completed", Domain: "backlog", Tier: manifestscan.TierFull}},
	})
	if f := findingBy(rep, "measures.architecture-fallback"); f != nil {
		t.Fatalf("conformant scenario must not carry the fallback advisory, got %+v", f)
	}
}

func TestClassify_Fallback_DeclaredDomainCovered(t *testing.T) {
	// Fallback mode: no folder; the stateful domain is declared via the override
	// and covered by a measure. Passes, with only the standing advisory.
	rep := Classify(Inputs{
		Scenario:  "react-vite-thing",
		Mode:      ModeFallback,
		Overrides: []manifestscan.DomainOverride{{Domain: "notes", Stateful: true}},
		Measures:  []HarvestedMeasure{{Name: "notes.count", Domain: "notes", Tier: manifestscan.TierFull}},
	})
	if !rep.Passed {
		t.Fatalf("fallback declared+covered must pass; findings=%+v", rep.Findings)
	}
	if d := domainBy(rep, "notes"); d == nil || d.Status != StatusCovered {
		t.Fatalf("notes: want covered, got %+v", d)
	}
	adv := findingBy(rep, "measures.architecture-fallback")
	if adv == nil || adv.Severity != SeverityInfo {
		t.Fatalf("want architecture-fallback INFO advisory, findings=%+v", rep.Findings)
	}
}

func TestClassify_Fallback_NothingDeclaredAdvisoryOnly(t *testing.T) {
	// Fallback mode with nothing declared: a standing INFO advisory, no ERROR.
	// (The anti-honor-system teeth for an undeclared real entity come in Phase 3.)
	rep := Classify(Inputs{
		Scenario: "bare-scenario",
		Mode:     ModeFallback,
	})
	if !rep.Passed {
		t.Fatalf("fallback with nothing declared must not ERROR yet; findings=%+v", rep.Findings)
	}
	e, w, _ := rep.Summary()
	if e != 0 || w != 0 {
		t.Fatalf("want only an INFO advisory, got errors=%d warnings=%d findings=%+v", e, w, rep.Findings)
	}
	if f := findingBy(rep, "measures.architecture-fallback"); f == nil {
		t.Fatalf("want standing architecture-fallback advisory, findings=%+v", rep.Findings)
	}
}

func TestReport_SummaryCounts(t *testing.T) {
	rep := Report{Findings: []Finding{
		{Severity: SeverityError},
		{Severity: SeverityError},
		{Severity: SeverityWarning},
		{Severity: SeverityInfo},
	}}
	e, w, i := rep.Summary()
	if e != 2 || w != 1 || i != 1 {
		t.Fatalf("summary want 2/1/1, got %d/%d/%d", e, w, i)
	}
}
