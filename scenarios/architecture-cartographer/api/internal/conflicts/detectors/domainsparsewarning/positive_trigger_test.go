package domainsparsewarning_test

import (
	"context"
	"strings"
	"testing"

	"architecture-cartographer/internal/conflicts"
	"architecture-cartographer/internal/conflicts/detectors/domainsparsewarning"
	"architecture-cartographer/internal/domains"
)

// TestPositiveTrigger_MalformedDomainsRow documents the canonical drift
// the parse-warning detector must catch: a DOMAINS.md with a row that
// the structured parser silently skipped (wrong column count). Closes
// Plan Problem 3 ("detector coverage opaque") with an intent-driven
// trigger fixture; without this, a row drop disappears entirely.
func TestPositiveTrigger_MalformedDomainsRow(t *testing.T) {
	m := domains.DerivedDomainMap{
		Warnings: []domains.ExtractionWarning{{
			Kind:    "domains_doc.row_shape",
			Path:    "docs/concepts/DOMAINS.md",
			Line:    42,
			Summary: "row has 5 columns, header has 8",
		}},
	}
	got, err := domainsparsewarning.New().Detect(context.Background(), conflicts.DetectInput{
		Scenario: "demo", DomainMap: m,
	})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 parse-warning conflict, got %d (%+v)", len(got), got)
	}
	c := got[0]
	if c.Type != "domains_doc_parse_warning" {
		t.Fatalf("want type=domains_doc_parse_warning, got %s", c.Type)
	}
	if c.Subtype != "domains_doc.row_shape" {
		t.Fatalf("want subtype=domains_doc.row_shape, got %s", c.Subtype)
	}
	if c.Severity != conflicts.SeverityWarn {
		t.Fatalf("want warn severity, got %s", c.Severity)
	}
	if len(c.Locations) != 1 || !strings.Contains(c.Locations[0], "DOMAINS.md:42") {
		t.Fatalf("want path:line locator, got %v", c.Locations)
	}
	if len(c.SuggestedFixes) == 0 || c.SuggestedFixes[0].Summary == "" {
		t.Fatalf("want templated fix summary, got %+v", c.SuggestedFixes)
	}
}
