package intentalignment_test

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"architecture-cartographer/internal/conflicts"
	"architecture-cartographer/internal/conflicts/detectors/intentalignment"
	"architecture-cartographer/internal/domains"

	intent "intent-go"
)

func TestDetectorDetectsRequirementDomainJoinFindings(t *testing.T) {
	d := intentalignment.New()
	claims := []intent.CapabilityClaim{
		{ID: "OT-P0-001", Altitude: intent.Outcome, Text: "Analyze image content for operator review", Anchor: "PRD.md:10"},
		{ID: "OT-P0-002", Altitude: intent.Outcome, Text: "Route analysis requests through the API", Anchor: "PRD.md:11"},
		{
			ID:       "REQ-1",
			Altitude: intent.Requirement,
			Text:     "Image analysis runs through the analysis domain",
			Anchor:   "requirements/analysis/module.json",
			Refs: []intent.Ref{
				{Raw: "OT-P0-001", Path: "OT-P0-001", Kind: intent.RefDoc},
				{Raw: "api/internal/analysis/run.go", Path: "api/internal/analysis/run.go", Kind: intent.RefCode},
			},
		},
		{
			ID:       "REQ-2",
			Altitude: intent.Requirement,
			Text:     "Transport exposes request routing",
			Anchor:   "requirements/transport/module.json",
			Refs: []intent.Ref{
				{Raw: "OT-P0-002", Path: "OT-P0-002", Kind: intent.RefDoc},
				{Raw: "api/handlers/analyze.go", Path: "api/handlers/analyze.go", Kind: intent.RefCode},
			},
		},
		{
			ID:       "REQ-3",
			Altitude: intent.Requirement,
			Text:     "AI evidence is still aspirational",
			Anchor:   "requirements/orphan/module.json",
			Refs: []intent.Ref{
				{Raw: "api/internal/ai/run.go", Path: "api/internal/ai/run.go", Kind: intent.RefCode},
			},
		},
	}
	dmap := domains.DerivedDomainMap{
		Scenario: "demo",
		Domains: []domains.DerivedDomain{
			{Name: "analysis", Paths: []string{"api/internal/analysis/**"}, Glossary: []string{"image", "ledger"}},
			{Name: "billing", Paths: []string{"api/internal/billing/**"}},
		},
		SharedSubstrate: []string{"api/handlers/**"},
		NonDomains:      []string{"handlers"},
	}

	got, err := d.Detect(context.Background(), conflicts.DetectInput{
		Scenario:      "demo",
		DomainMap:     dmap,
		ClaimProvider: conflicts.StaticClaimProvider{ClaimsOut: claims},
	})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}

	counts := countByType(got)
	want := expectedPositiveFixtureCounts()
	for code, n := range want {
		if counts[code] != n {
			t.Fatalf("count for %s = %d, want %d; all counts=%v", code, counts[code], n, counts)
		}
	}
	for _, finding := range got {
		if finding.Detector != "intent_alignment" {
			t.Fatalf("detector = %q", finding.Detector)
		}
		if finding.Type == intent.CodeReqUnownedDomain && finding.Severity != conflicts.SeverityError {
			t.Fatalf("unowned severity = %s", finding.Severity)
		}
		if finding.Type == intent.CodeReqTransportOwned && finding.Severity != conflicts.SeverityInfo {
			t.Fatalf("transport severity = %s", finding.Severity)
		}
		if finding.Type == intent.CodeVocabDrift && finding.Severity != conflicts.SeverityWarn {
			t.Fatalf("vocab severity = %s", finding.Severity)
		}
	}
}

func TestLexicalMatcherSkipsAlignedGlossary(t *testing.T) {
	matcher := intentalignment.LexicalMatcher{}
	domain := intent.CapabilityClaim{ID: "analysis", Altitude: intent.Domain, Text: "analysis image review"}
	req := intent.CapabilityClaim{ID: "REQ-1", Text: "Image review is validated here"}

	got, err := matcher.Match(context.Background(), intentalignment.MatchInput{
		RequirementsByDomain: map[string][]intent.CapabilityClaim{"analysis": {req}},
		Domains:              []intent.CapabilityClaim{domain},
	})
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("got %d matches, want none: %+v", len(got), got)
	}
}

func TestDeferredMatchersAreOffByDefault(t *testing.T) {
	for _, matcher := range []intentalignment.Matcher{
		intentalignment.EmbeddingMatcher{},
		intentalignment.LLMMatcher{},
	} {
		got, err := matcher.Match(context.Background(), intentalignment.MatchInput{})
		if err != nil {
			t.Fatalf("%s Match: %v", matcher.Name(), err)
		}
		if len(got) != 0 {
			t.Fatalf("%s emitted %d matches while off by default", matcher.Name(), len(got))
		}
	}
}

func TestIntentAlignmentDetectorCodesAreDocumentedAndExercised(t *testing.T) {
	// INVARIANT: intentFindingCodesMatchDoctrine
	doctrineCodes := readDoctrineInvariantCodes(t)
	emittedCodes := setFromSlice(intentalignment.New().EmitsTypes())
	testedCodes := setFromCountMap(expectedPositiveFixtureCounts())

	assertSameStringSet(t, "emitted detector codes", emittedCodes, "positive fixture codes", testedCodes)
	for code := range emittedCodes {
		if !doctrineCodes[code] {
			t.Fatalf("detector emits %s, but docs/reference/intent-alignment.md does not list it in the invariant registry", code)
		}
	}
}

func expectedPositiveFixtureCounts() map[string]int {
	return map[string]int{
		intent.CodeReqUnownedDomain:  1,
		intent.CodeReqTransportOwned: 1,
		intent.CodeDomainUnrequired:  1,
		intent.CodeOTNoDomain:        1,
		intent.CodeVocabDrift:        1,
	}
}

func TestDetectorSkipsWithoutClaimProvider(t *testing.T) {
	got, err := intentalignment.New().Detect(context.Background(), conflicts.DetectInput{Scenario: "demo"})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("got %d findings, want none", len(got))
	}
}

func readDoctrineInvariantCodes(t *testing.T) map[string]bool {
	t.Helper()
	path := filepath.Join("..", "..", "..", "..", "..", "..", "..", "docs", "reference", "intent-alignment.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read doctrine: %v", err)
	}
	re := regexp.MustCompile("`(intent\\.[a-z_]+)`")
	out := map[string]bool{}
	for _, match := range re.FindAllStringSubmatch(string(data), -1) {
		out[match[1]] = true
	}
	return out
}

func countByType(findings []conflicts.Conflict) map[string]int {
	out := map[string]int{}
	for _, finding := range findings {
		out[finding.Type]++
	}
	return out
}

func setFromCountMap(values map[string]int) map[string]bool {
	out := map[string]bool{}
	for key := range values {
		out[key] = true
	}
	return out
}

func setFromSlice(values []string) map[string]bool {
	out := map[string]bool{}
	for _, value := range values {
		out[value] = true
	}
	return out
}

func assertSameStringSet(t *testing.T, leftName string, left map[string]bool, rightName string, right map[string]bool) {
	t.Helper()
	for value := range left {
		if !right[value] {
			t.Fatalf("%s contains %s, but %s does not", leftName, value, rightName)
		}
	}
	for value := range right {
		if !left[value] {
			t.Fatalf("%s contains %s, but %s does not", rightName, value, leftName)
		}
	}
}
