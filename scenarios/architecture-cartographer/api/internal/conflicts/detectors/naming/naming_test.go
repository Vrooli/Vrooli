package naming_test

import (
	"context"
	"testing"

	"architecture-cartographer/internal/conflicts"
	"architecture-cartographer/internal/conflicts/detectors/naming"
	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/graph"
)

func TestDetect_GenericDomainName(t *testing.T) {
	got, err := naming.New().Detect(context.Background(), conflicts.DetectInput{
		Scenario: "demo",
		DomainMap: domains.DerivedDomainMap{Domains: []domains.DerivedDomain{
			{Name: "utils", Paths: []string{"api/internal/utils/**"}},
		}},
	})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected one conflict, got %+v", got)
	}
	if got[0].Subtype != "generic-domain-name" || got[0].Severity != conflicts.SeverityWarn {
		t.Fatalf("unexpected conflict: %+v", got[0])
	}
}

func TestDetect_GenericPackageName(t *testing.T) {
	got, err := naming.New().Detect(context.Background(), conflicts.DetectInput{
		Scenario: "demo",
		Snapshot: graph.GraphSnapshot{Packages: []graph.PackageNode{
			{ID: "pkg:helpers", RepoPath: "api/internal/billing/helpers"},
		}},
		DomainMap: domains.DerivedDomainMap{Domains: []domains.DerivedDomain{
			{Name: "billing", Paths: []string{"api/internal/billing/**"}},
		}},
	})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected one conflict, got %+v", got)
	}
	if got[0].Subtype != "generic-package-name" || len(got[0].Domains) != 1 || got[0].Domains[0] != "billing" {
		t.Fatalf("unexpected conflict: %+v", got[0])
	}
}

func TestDetect_CustomVocabulary(t *testing.T) {
	got, err := naming.NewWithBannedVocabulary([]string{"bucket"}).Detect(context.Background(), conflicts.DetectInput{
		Scenario: "demo",
		Snapshot: graph.GraphSnapshot{Packages: []graph.PackageNode{
			{ID: "pkg:bucket", RepoPath: "api/internal/billing/bucket"},
			{ID: "pkg:helpers", RepoPath: "api/internal/billing/helpers"},
		}},
		DomainMap: domains.DerivedDomainMap{Domains: []domains.DerivedDomain{
			{Name: "billing", Paths: []string{"api/internal/billing/**"}},
		}},
	})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(got) != 1 || got[0].Locations[0] != "api/internal/billing/bucket" {
		t.Fatalf("custom vocabulary should only flag bucket, got %+v", got)
	}
}

func TestDetect_DeterministicOrder(t *testing.T) {
	in := conflicts.DetectInput{
		Scenario: "demo",
		Snapshot: graph.GraphSnapshot{Packages: []graph.PackageNode{
			{ID: "pkg:utils", RepoPath: "api/internal/billing/utils"},
			{ID: "pkg:misc", RepoPath: "api/internal/billing/misc"},
		}},
		DomainMap: domains.DerivedDomainMap{Domains: []domains.DerivedDomain{
			{Name: "billing", Paths: []string{"api/internal/billing/**"}},
		}},
	}
	first, err := naming.New().Detect(context.Background(), in)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	second, err := naming.New().Detect(context.Background(), in)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(first) != len(second) {
		t.Fatalf("count drift: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i].Locations[0] != second[i].Locations[0] {
			t.Fatalf("order drift at %d: %q vs %q", i, first[i].Locations[0], second[i].Locations[0])
		}
	}
}
