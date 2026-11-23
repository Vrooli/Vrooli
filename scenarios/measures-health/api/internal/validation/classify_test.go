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

type expectedDomain struct {
	status       DomainStatus
	tier         manifestscan.Tier
	measureCount int
	waiverReason string
	note         string
}

type expectedFinding struct {
	rule     string
	severity Severity
}

type classifyCase struct {
	name         string
	input        Inputs
	wantPassed   bool
	wantDomains  map[string]expectedDomain
	wantFindings []expectedFinding
}

// Classify's independent outcomes live together so maintainers can compare
// adjacent state, tier, waiver and mode contracts without repeating report
// setup and lookup assertions. Each row still supplies literal inputs and
// expected public findings; adding a case cannot derive its oracle from code.
func TestClassifyContracts(t *testing.T) {
	cases := []classifyCase{
		{
			name: "covered stateful domain passes with strongest measure tier",
			input: Inputs{
				Scenario: "swarm-manager",
				Domains:  []DerivedDomain{{Name: "backlog", Stateful: true}},
				Measures: []HarvestedMeasure{{Name: "backlog.completed", Domain: "backlog", Effect: "read", Tier: manifestscan.TierFull, QuestionCount: 3}},
			},
			wantPassed:  true,
			wantDomains: map[string]expectedDomain{"backlog": {status: StatusCovered, tier: manifestscan.TierFull, measureCount: 1}},
		},
		{
			name: "uncovered stateful domain is a blocking error",
			input: Inputs{
				Scenario: "swarm-manager",
				Domains:  []DerivedDomain{{Name: "captures", Stateful: true}},
			},
			wantDomains:  map[string]expectedDomain{"captures": {status: StatusUncovered}},
			wantFindings: []expectedFinding{{rule: "measures.uncovered-domain", severity: SeverityError}},
		},
		{
			name: "stateful domain with a reasoned waiver passes",
			input: Inputs{
				Scenario: "swarm-manager",
				Domains:  []DerivedDomain{{Name: "queue", Stateful: true}},
				Omitted:  []manifestscan.Omission{{Domain: "queue", Reason: "ephemeral; no historical value"}},
			},
			wantPassed:  true,
			wantDomains: map[string]expectedDomain{"queue": {status: StatusWaived, waiverReason: "ephemeral; no historical value"}},
		},
		{
			name: "stale waiver warns without blocking",
			input: Inputs{
				Scenario: "swarm-manager",
				Domains:  []DerivedDomain{{Name: "backlog", Stateful: true}},
				Measures: []HarvestedMeasure{{Name: "backlog.completed", Domain: "backlog", Tier: manifestscan.TierFull}},
				Omitted:  []manifestscan.Omission{{Domain: "nonexistent", Reason: "stale"}},
			},
			wantPassed: true,
			wantDomains: map[string]expectedDomain{
				"backlog":     {status: StatusCovered, tier: manifestscan.TierFull, measureCount: 1},
				"nonexistent": {status: StatusNotExpected, note: "waiver targets a non-stateful domain"},
			},
			wantFindings: []expectedFinding{{rule: "measures.stale-waiver", severity: SeverityWarning}},
		},
		{
			name: "stateless domain is not expected and stays quiet",
			input: Inputs{
				Scenario: "swarm-manager",
				Domains:  []DerivedDomain{{Name: "settings", Stateful: false, Note: "stateless configuration domain"}},
			},
			wantPassed:  true,
			wantDomains: map[string]expectedDomain{"settings": {status: StatusNotExpected, note: "stateless configuration domain"}},
		},
		{
			name: "override can downgrade a derived stateful domain",
			input: Inputs{
				Scenario:  "swarm-manager",
				Domains:   []DerivedDomain{{Name: "graph", Stateful: true}},
				Overrides: []manifestscan.DomainOverride{{Domain: "graph", Stateful: false, Reason: "structural-only, no rows"}},
			},
			wantPassed:  true,
			wantDomains: map[string]expectedDomain{"graph": {status: StatusNotExpected, note: "structural-only, no rows"}},
		},
		{
			name: "weakest measure tier governs covered domain",
			input: Inputs{
				Scenario: "s",
				Domains:  []DerivedDomain{{Name: "backlog", Stateful: true}},
				Measures: []HarvestedMeasure{
					{Name: "backlog.a", Domain: "backlog", Tier: manifestscan.TierFull},
					{Name: "backlog.b", Domain: "backlog", Tier: manifestscan.TierPartial, TierNote: "window param not canonical"},
				},
			},
			wantPassed:   true,
			wantDomains:  map[string]expectedDomain{"backlog": {status: StatusCovered, tier: manifestscan.TierPartial, measureCount: 2}},
			wantFindings: []expectedFinding{{rule: "measures.tier-partial", severity: SeverityInfo}},
		},
		{
			name: "fallback tier warns without blocking",
			input: Inputs{
				Scenario: "s",
				Domains:  []DerivedDomain{{Name: "backlog", Stateful: true}},
				Measures: []HarvestedMeasure{{Name: "backlog.a", Domain: "backlog", Tier: manifestscan.TierFallback}},
			},
			wantPassed:   true,
			wantDomains:  map[string]expectedDomain{"backlog": {status: StatusCovered, tier: manifestscan.TierFallback, measureCount: 1}},
			wantFindings: []expectedFinding{{rule: "measures.tier-fallback", severity: SeverityWarning}},
		},
		{
			name: "malformed declaration fails and leaves domain uncovered",
			input: Inputs{
				Scenario: "s",
				Domains:  []DerivedDomain{{Name: "backlog", Stateful: true}},
				Measures: []HarvestedMeasure{{Name: "backlog.broken", Domain: "backlog", AssembleErr: `manifest param "foo" has no matching field`}},
			},
			wantDomains:  map[string]expectedDomain{"backlog": {status: StatusUncovered}},
			wantFindings: []expectedFinding{{rule: "measures.malformed-declaration", severity: SeverityError}, {rule: "measures.uncovered-domain", severity: SeverityError}},
		},
		{
			name: "an extra measure is allowed even when its domain is not expected",
			input: Inputs{
				Scenario: "s",
				Measures: []HarvestedMeasure{{Name: "extra.thing", Domain: "extra", Tier: manifestscan.TierFull}},
			},
			wantPassed:  true,
			wantDomains: map[string]expectedDomain{"extra": {status: StatusCovered, tier: manifestscan.TierFull, measureCount: 1}},
		},
		{
			name: "conformant mode rejects an additive stateful declaration only once",
			input: Inputs{
				Scenario:  "swarm-manager",
				Mode:      ModeConformant,
				Domains:   []DerivedDomain{{Name: "backlog", Stateful: true}},
				Measures:  []HarvestedMeasure{{Name: "backlog.completed", Domain: "backlog", Tier: manifestscan.TierFull}},
				Overrides: []manifestscan.DomainOverride{{Domain: "smuggled", Stateful: true}},
			},
			wantDomains:  map[string]expectedDomain{"backlog": {status: StatusCovered, tier: manifestscan.TierFull, measureCount: 1}},
			wantFindings: []expectedFinding{{rule: "measures.illegal-domain-declaration", severity: SeverityError}},
		},
		{
			name: "conformant mode permits a downgrade override",
			input: Inputs{
				Scenario:  "swarm-manager",
				Mode:      ModeConformant,
				Domains:   []DerivedDomain{{Name: "graph", Stateful: true}},
				Overrides: []manifestscan.DomainOverride{{Domain: "graph", Stateful: false, Reason: "structural-only"}},
			},
			wantPassed:  true,
			wantDomains: map[string]expectedDomain{"graph": {status: StatusNotExpected, note: "structural-only"}},
		},
		{
			name: "conformant mode permits a waiver and has no fallback advisory",
			input: Inputs{
				Scenario: "swarm-manager",
				Mode:     ModeConformant,
				Domains:  []DerivedDomain{{Name: "capture", Stateful: true}},
				Omitted:  []manifestscan.Omission{{Domain: "capture", Reason: "ephemeral; no event"}},
			},
			wantPassed:  true,
			wantDomains: map[string]expectedDomain{"capture": {status: StatusWaived, waiverReason: "ephemeral; no event"}},
		},
		{
			name: "fallback declaration is covered and carries only its advisory",
			input: Inputs{
				Scenario:  "react-vite-thing",
				Mode:      ModeFallback,
				Overrides: []manifestscan.DomainOverride{{Domain: "notes", Stateful: true}},
				Measures:  []HarvestedMeasure{{Name: "notes.count", Domain: "notes", Tier: manifestscan.TierFull}},
			},
			wantPassed:   true,
			wantDomains:  map[string]expectedDomain{"notes": {status: StatusCovered, tier: manifestscan.TierFull, measureCount: 1}},
			wantFindings: []expectedFinding{{rule: "measures.architecture-fallback", severity: SeverityInfo}},
		},
		{
			name:         "empty fallback stays advisory only",
			input:        Inputs{Scenario: "bare-scenario", Mode: ModeFallback},
			wantPassed:   true,
			wantFindings: []expectedFinding{{rule: "measures.architecture-fallback", severity: SeverityInfo}},
		},
		{
			name: "undeclared persisted entity warns in fallback mode",
			input: Inputs{
				Scenario: "demo",
				Mode:     ModeFallback,
				Detected: []DetectedEntity{{Name: "notes", Evidence: "table `notes` (created_at) in scenarios/demo/api/internal/notes/schema.sql"}},
			},
			wantPassed: true,
			wantFindings: []expectedFinding{
				{rule: "measures.undeclared-substrate", severity: SeverityWarning},
				{rule: "measures.architecture-fallback", severity: SeverityInfo},
			},
		},
		{
			name: "detected entity can be waived without substrate findings",
			input: Inputs{
				Scenario: "demo",
				Mode:     ModeFallback,
				Omitted:  []manifestscan.Omission{{Domain: "notes", Reason: "ephemeral scratch"}},
				Detected: []DetectedEntity{{Name: "notes", Evidence: "table `notes` (created_at)"}},
			},
			wantPassed: true,
			wantDomains: map[string]expectedDomain{
				"notes": {status: StatusWaived, waiverReason: "ephemeral scratch"},
			},
			wantFindings: []expectedFinding{{rule: "measures.architecture-fallback", severity: SeverityInfo}},
		},
		{
			name: "plural substrate maps to covered singular domain",
			input: Inputs{
				Scenario: "demo",
				Mode:     ModeConformant,
				Domains:  []DerivedDomain{{Name: "note", Stateful: true}},
				Measures: []HarvestedMeasure{{Name: "note.count", Domain: "note", Tier: manifestscan.TierFull}},
				Detected: []DetectedEntity{{Name: "notes", Evidence: "table `notes` (created_at)"}},
			},
			wantPassed:  true,
			wantDomains: map[string]expectedDomain{"note": {status: StatusCovered, tier: manifestscan.TierFull, measureCount: 1}},
		},
		{
			name: "known uncovered substrate does not emit a duplicate warning",
			input: Inputs{
				Scenario: "demo",
				Mode:     ModeConformant,
				Domains:  []DerivedDomain{{Name: "notes", Stateful: true}},
				Detected: []DetectedEntity{{Name: "notes", Evidence: "table `notes` (created_at)"}},
			},
			wantDomains:  map[string]expectedDomain{"notes": {status: StatusUncovered}},
			wantFindings: []expectedFinding{{rule: "measures.uncovered-domain", severity: SeverityError}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			report := Classify(tc.input)
			if report.Passed != tc.wantPassed {
				t.Fatalf("Passed = %v, want %v; findings=%+v", report.Passed, tc.wantPassed, report.Findings)
			}
			if len(report.Domains) != len(tc.wantDomains) {
				t.Fatalf("classified %d domains, want %d: %+v", len(report.Domains), len(tc.wantDomains), report.Domains)
			}
			for name, want := range tc.wantDomains {
				domain := domainBy(report, name)
				if domain == nil || domain.Status != want.status || domain.Tier != want.tier || domain.MeasureCount != want.measureCount || domain.WaiverReason != want.waiverReason || domain.Note != want.note {
					t.Errorf("domain %q = %+v, want %+v", name, domain, want)
				}
			}
			if len(report.Findings) != len(tc.wantFindings) {
				t.Fatalf("got findings %+v, want %+v", report.Findings, tc.wantFindings)
			}
			for _, want := range tc.wantFindings {
				finding := findingBy(report, want.rule)
				if finding == nil || finding.Severity != want.severity {
					t.Errorf("finding %q = %+v, want severity %s", want.rule, finding, want.severity)
				}
			}
		})
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
