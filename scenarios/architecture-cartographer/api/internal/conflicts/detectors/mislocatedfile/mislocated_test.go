package mislocatedfile_test

import (
	"context"
	"testing"

	"architecture-cartographer/internal/conflicts"
	"architecture-cartographer/internal/conflicts/detectors/mislocatedfile"
	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/graph"
)

type stubVerdictProvider struct {
	v conflicts.Verdict
}

func (s stubVerdictProvider) VerdictFor(_ context.Context, _ string, _ graph.Chunk) (conflicts.Verdict, error) {
	return s.v, nil
}

func TestDetect_EmitsWhenVerdictDisagreesWithDomainMap(t *testing.T) {
	snap := graph.GraphSnapshot{
		Files: []graph.FileNode{
			{ID: "file:a", Path: "internal/graph/service.go", PackageID: "pkg:graph"},
		},
	}
	m := domains.DerivedDomainMap{
		Domains: []domains.DerivedDomain{
			{Name: "graph", Paths: []string{"internal/graph/**"}},
			{Name: "conflicts", Paths: []string{"internal/conflicts/**"}},
		},
	}
	in := conflicts.DetectInput{
		Scenario: "demo", Snapshot: snap, DomainMap: m,
		VerdictProvider: stubVerdictProvider{v: conflicts.Verdict{
			Tier:      "auto_place",
			TopDomain: "conflicts",
			TopValue:  0.92,
		}},
	}
	got, err := mislocatedfile.New().Detect(context.Background(), in)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 conflict, got %d", len(got))
	}
	if got[0].Type != "mislocated_file" {
		t.Fatalf("unexpected type %s", got[0].Type)
	}
	if len(got[0].SuggestedFixes) == 0 || got[0].SuggestedFixes[0].Kind != conflicts.FixKindMoveFile {
		t.Fatalf("expected move_file fix, got %+v", got[0].SuggestedFixes)
	}
}

func TestDetect_NoConflictWhenVerdictMatchesManifest(t *testing.T) {
	snap := graph.GraphSnapshot{
		Files: []graph.FileNode{
			{ID: "file:a", Path: "internal/graph/service.go"},
		},
	}
	m := domains.DerivedDomainMap{
		Domains: []domains.DerivedDomain{{Name: "graph", Paths: []string{"internal/graph/**"}}},
	}
	in := conflicts.DetectInput{
		Scenario: "demo", Snapshot: snap, DomainMap: m,
		VerdictProvider: stubVerdictProvider{v: conflicts.Verdict{Tier: "auto_place", TopDomain: "graph"}},
	}
	got, _ := mislocatedfile.New().Detect(context.Background(), in)
	if len(got) != 0 {
		t.Fatalf("expected no conflicts, got %+v", got)
	}
}

func TestDetect_SkipsNonAutoPlaceTiers(t *testing.T) {
	snap := graph.GraphSnapshot{
		Files: []graph.FileNode{
			{ID: "file:a", Path: "internal/graph/service.go"},
		},
	}
	m := domains.DerivedDomainMap{
		Domains: []domains.DerivedDomain{
			{Name: "graph", Paths: []string{"internal/graph/**"}},
			{Name: "conflicts", Paths: []string{"internal/conflicts/**"}},
		},
	}
	in := conflicts.DetectInput{
		Scenario: "demo", Snapshot: snap, DomainMap: m,
		VerdictProvider: stubVerdictProvider{v: conflicts.Verdict{Tier: "suggest", TopDomain: "conflicts"}},
	}
	got, _ := mislocatedfile.New().Detect(context.Background(), in)
	if len(got) != 0 {
		t.Fatalf("non-auto_place tier must not emit, got %+v", got)
	}
}

func TestDetect_NilVerdictProviderReturnsNothing(t *testing.T) {
	got, err := mislocatedfile.New().Detect(context.Background(), conflicts.DetectInput{
		Snapshot: graph.GraphSnapshot{Files: []graph.FileNode{{ID: "file:a", Path: "x.go"}}},
	})
	if err != nil || len(got) != 0 {
		t.Fatalf("nil provider should be a no-op; got %+v err=%v", got, err)
	}
}
