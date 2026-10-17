package mislocatedfile_test

import (
	"context"
	"testing"

	"architecture-cartographer/internal/conflicts"
	"architecture-cartographer/internal/conflicts/detectors/mislocatedfile"
	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/graph"
)

// TestPositiveTrigger_FileOwnedByWrongDomain documents the canonical
// drift the mislocated_file detector must catch: a file at
// api/internal/foo/bar.go whose verdict says it actually belongs in
// the `baz` domain. Closes Plan Problem 3 ("detector coverage opaque")
// with a clear, intent-driven trigger fixture.
func TestPositiveTrigger_FileOwnedByWrongDomain(t *testing.T) {
	snap := graph.GraphSnapshot{
		Files: []graph.FileNode{
			{ID: "file:bar", Path: "api/internal/foo/bar.go", PackageID: "pkg:foo"},
		},
	}
	m := domains.DerivedDomainMap{
		Domains: []domains.DerivedDomain{
			{Name: "foo", Paths: []string{"api/internal/foo/**"}},
			{Name: "baz", Paths: []string{"api/internal/baz/**"}},
		},
	}
	in := conflicts.DetectInput{
		Scenario: "demo", Snapshot: snap, DomainMap: m,
		VerdictProvider: stubVerdictProvider{v: conflicts.Verdict{
			Tier:      "auto_place",
			TopDomain: "baz",
			TopValue:  0.91,
		}},
	}
	got, err := mislocatedfile.New().Detect(context.Background(), in)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 mislocated conflict, got %d (%+v)", len(got), got)
	}
	c := got[0]
	if c.Type != "mislocated_file" {
		t.Fatalf("want type=mislocated_file, got %s", c.Type)
	}
	if c.Subtype != "foo-to-baz" {
		t.Fatalf("subtype must reflect from/to domains, got %q", c.Subtype)
	}
	if len(c.Locations) != 1 || c.Locations[0] != "api/internal/foo/bar.go" {
		t.Fatalf("want file path as location, got %v", c.Locations)
	}
	if len(c.Evidence) < 2 {
		t.Fatalf("want verdict + derived-location evidence, got %+v", c.Evidence)
	}
	if len(c.SuggestedFixes) == 0 || c.SuggestedFixes[0].Kind != conflicts.FixKindMoveFile {
		t.Fatalf("want move_file fix, got %+v", c.SuggestedFixes)
	}
}
