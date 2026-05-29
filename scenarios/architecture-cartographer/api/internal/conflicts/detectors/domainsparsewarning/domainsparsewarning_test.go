package domainsparsewarning_test

import (
	"context"
	"testing"

	"architecture-cartographer/internal/conflicts"
	"architecture-cartographer/internal/conflicts/detectors/domainsparsewarning"
	"architecture-cartographer/internal/domains"
)

func TestDetect_NoWarningsNoConflicts(t *testing.T) {
	d := domainsparsewarning.New()
	out, err := d.Detect(context.Background(), conflicts.DetectInput{
		Scenario:  "demo",
		DomainMap: domains.DerivedDomainMap{},
	})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("want 0 conflicts for empty warnings, got %d", len(out))
	}
}

func TestDetect_EmitsOneConflictPerWarning(t *testing.T) {
	d := domainsparsewarning.New()
	out, err := d.Detect(context.Background(), conflicts.DetectInput{
		Scenario: "demo",
		DomainMap: domains.DerivedDomainMap{
			Warnings: []domains.ExtractionWarning{
				{Kind: "domains_doc.row_shape", Path: "docs/concepts/DOMAINS.md", Line: 42, Summary: "row has 5 columns, header has 8"},
				{Kind: "domains_doc.empty_name", Path: "docs/concepts/DOMAINS.md", Line: 51, Summary: "row has empty Domain cell"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("want 2 conflicts, got %d", len(out))
	}
	if out[0].Severity != conflicts.SeverityWarn {
		t.Fatalf("want warn severity, got %s", out[0].Severity)
	}
	if out[0].Type != "domains_doc_parse_warning" {
		t.Fatalf("want type=domains_doc_parse_warning, got %s", out[0].Type)
	}
	if out[0].Subtype != "domains_doc.row_shape" {
		t.Fatalf("want subtype=domains_doc.row_shape, got %s", out[0].Subtype)
	}
	if out[0].Locations[0] != "docs/concepts/DOMAINS.md:42" {
		t.Fatalf("want path:line locator, got %s", out[0].Locations[0])
	}
}
