package convergencedrift_test

import (
	"context"
	"testing"
	"time"

	"architecture-cartographer/internal/conflicts"
	"architecture-cartographer/internal/conflicts/detectors/convergencedrift"
	"architecture-cartographer/internal/domains"
)

func mapWith(t *testing.T, doc, folders []string) domains.DerivedDomainMap {
	t.Helper()
	mk := func(src domains.Source, names []string) domains.Extraction {
		ex := domains.Extraction{Source: src}
		for _, n := range names {
			ex.Domains = append(ex.Domains, domains.ExtractedDomain{Name: n, Paths: []string{n + "/"}})
		}
		return ex
	}
	m, err := domains.Resolve("demo", []domains.Extraction{
		mk(domains.SourceDomainsDoc, doc),
		mk(domains.SourceAPIFolders, folders),
	}, time.Time{})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	return m
}

func TestDetect_NoDriftNoConflicts(t *testing.T) {
	in := conflicts.DetectInput{Scenario: "demo", DomainMap: mapWith(t, []string{"graph"}, []string{"graph"})}
	got, err := convergencedrift.New().Detect(context.Background(), in)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no conflicts, got %+v", got)
	}
}

func TestDetect_EmitsMissingImplementation(t *testing.T) {
	// "conflicts" declared in doc, absent from folders.
	in := conflicts.DetectInput{Scenario: "demo", DomainMap: mapWith(t, []string{"graph", "conflicts"}, []string{"graph"})}
	got, err := convergencedrift.New().Detect(context.Background(), in)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	var found *conflicts.Conflict
	for i := range got {
		if got[i].Subtype == domains.FindingMissingImplementation {
			found = &got[i]
		}
	}
	if found == nil {
		t.Fatalf("expected missing_implementation conflict, got %+v", got)
	}
	if found.Type != "convergence_drift" || found.Severity != conflicts.SeverityWarn {
		t.Fatalf("unexpected conflict shape: %+v", found)
	}
	if len(found.Domains) != 1 || found.Domains[0] != "conflicts" {
		t.Fatalf("unexpected domains: %v", found.Domains)
	}
}

// TestDetect_EveryConflictHasSuggestedFix asserts no convergence_drift
// finding ships without at least one templated fix. Without this, the
// workbench is a finder, not a fixer.
func TestDetect_EveryConflictHasSuggestedFix(t *testing.T) {
	in := conflicts.DetectInput{Scenario: "demo", DomainMap: mapWith(t, []string{"graph", "conflicts"}, []string{"graph", "audit"})}
	got, err := convergencedrift.New().Detect(context.Background(), in)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(got) == 0 {
		t.Fatalf("expected drift conflicts to assert on, got none")
	}
	for _, c := range got {
		if len(c.SuggestedFixes) < 1 {
			t.Errorf("subtype %q: missing SuggestedFixes", c.Subtype)
			continue
		}
		if c.SuggestedFixes[0].Summary == "" {
			t.Errorf("subtype %q: first fix has empty Summary", c.Subtype)
		}
	}
}

func TestDetect_NameAndTypes(t *testing.T) {
	d := convergencedrift.New()
	if d.Name() != "convergence_drift" {
		t.Fatalf("name = %q", d.Name())
	}
	if len(d.EmitsTypes()) != 1 || d.EmitsTypes()[0] != "convergence_drift" {
		t.Fatalf("emits = %v", d.EmitsTypes())
	}
}
