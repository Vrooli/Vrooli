package dependencies

import (
	"context"
	"reflect"
	"testing"
)

// applyPkgCorpus seeds a corpus where the same package is used by several
// scenarios (with deliberately inconsistent vuln annotations to exercise the
// union/max aggregation), plus a clean package in one scenario.
func applyPkgCorpus(t *testing.T) *Store {
	t.Helper()
	s := newTestStore(t)
	recs := []DependencyRecord{
		// Same package (go|golang.org/x/net|v0.17.0) across three scenarios,
		// with differing vuln_ids + severities to verify union + max-fold.
		{Scenario: "alpha", Ecosystem: EcosystemGo, Name: "golang.org/x/net", Version: "v0.17.0", VulnIDs: []string{"GO-2024-1"}, MaxSeverity: "moderate"},
		{Scenario: "beta", Ecosystem: EcosystemGo, Name: "golang.org/x/net", Version: "v0.17.0", VulnIDs: []string{"GO-2024-1", "GO-2024-2"}, MaxSeverity: "high"},
		{Scenario: "gamma", Ecosystem: EcosystemGo, Name: "golang.org/x/net", Version: "v0.17.0", VulnIDs: []string{"GO-2024-2"}, MaxSeverity: "low"},
		// Clean package, single scenario.
		{Scenario: "alpha", Ecosystem: EcosystemNPM, Name: "react", Version: "18.0.0"},
	}
	if err := s.Apply(context.Background(), "", recs, "2026-06-01T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestPackageItems_DedupAndAggregate(t *testing.T) {
	ctx := context.Background()
	s := applyPkgCorpus(t)

	items, err := s.PackageItems(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 distinct packages, got %d: %+v", len(items), items)
	}

	byKey := map[string]PackageItem{}
	for _, it := range items {
		byKey[it.PkgKey()] = it
	}

	net, ok := byKey["go|golang.org/x/net|v0.17.0"]
	if !ok {
		t.Fatalf("x/net package missing: %+v", byKey)
	}
	// Union of vuln_ids, sorted + de-duped.
	if !reflect.DeepEqual(net.VulnIDs, []string{"GO-2024-1", "GO-2024-2"}) {
		t.Errorf("vuln union wrong: %+v", net.VulnIDs)
	}
	// Max severity across the rows (high beats moderate/low).
	if net.MaxSeverity != "high" {
		t.Errorf("max severity = %q, want high", net.MaxSeverity)
	}

	react, ok := byKey["npm|react|18.0.0"]
	if !ok {
		t.Fatalf("react package missing")
	}
	if len(react.VulnIDs) != 0 || react.MaxSeverity != "" {
		t.Errorf("clean package should carry no vulns: %+v", react)
	}
}

func TestRecordsByPackages_FansOut(t *testing.T) {
	ctx := context.Background()
	s := applyPkgCorpus(t)

	byPkg, err := s.RecordsByPackages(ctx, []string{"go|golang.org/x/net|v0.17.0", "npm|react|18.0.0", "go|missing|0"})
	if err != nil {
		t.Fatal(err)
	}
	net := byPkg["go|golang.org/x/net|v0.17.0"]
	if len(net) != 3 {
		t.Fatalf("x/net should fan out to 3 scenarios, got %d", len(net))
	}
	// Deterministic dep_key order: alpha < beta < gamma.
	if net[0].Scenario != "alpha" || net[1].Scenario != "beta" || net[2].Scenario != "gamma" {
		t.Errorf("fan-out not dep_key-ordered: %+v", net)
	}
	if len(byPkg["npm|react|18.0.0"]) != 1 {
		t.Errorf("react should fan out to 1 scenario")
	}
	if _, ok := byPkg["go|missing|0"]; ok {
		t.Errorf("missing package must be absent, not empty")
	}
}

func TestDistinctPackageCount(t *testing.T) {
	ctx := context.Background()
	s := applyPkgCorpus(t)
	n, err := s.DistinctPackageCount(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("DistinctPackageCount = %d, want 2", n)
	}
}
