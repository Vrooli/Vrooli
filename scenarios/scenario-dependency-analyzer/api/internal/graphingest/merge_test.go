package graphingest

import (
	"testing"
	"time"

	types "scenario-dependency-analyzer/internal/types"
)

func TestMergeDedupKeepsHighestConfidenceAndUnionsEvidence(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	edges := Merge([]Contribution{
		{From: "a", To: "b", Kind: KindScenario, Source: SourceDeclared, Required: true, Evidence: types.UnifiedEdgeEvidence{Source: SourceDeclared}},
		{From: "a", To: "b", Kind: KindScenario, Source: SourceProtoImport, Evidence: types.UnifiedEdgeEvidence{Source: SourceProtoImport, FromFile: "a.proto"}},
		{From: "a", To: "b", Kind: KindScenario, Source: SourceGoImport, Evidence: types.UnifiedEdgeEvidence{Source: SourceGoImport, ImportPath: "x/b"}},
	}, now)

	if len(edges) != 1 {
		t.Fatalf("expected 1 merged edge, got %d", len(edges))
	}
	edge := edges[0]
	if edge.Source != SourceProtoImport {
		t.Fatalf("expected highest-confidence source proto_import, got %q", edge.Source)
	}
	if edge.Confidence != ConfidenceFor(SourceProtoImport) {
		t.Fatalf("confidence = %v, want %v", edge.Confidence, ConfidenceFor(SourceProtoImport))
	}
	if !edge.Required {
		t.Fatalf("required should be OR-ed true from the declared contribution")
	}
	if len(edge.Evidence) != 3 {
		t.Fatalf("expected 3 unioned evidence entries, got %d", len(edge.Evidence))
	}
}

func TestMergeDropsSelfAndEmptyEdges(t *testing.T) {
	edges := Merge([]Contribution{
		{From: "a", To: "a", Kind: KindScenario, Source: SourceProtoImport},
		{From: "", To: "b", Kind: KindScenario, Source: SourceProtoImport},
		{From: "a", To: "", Kind: KindScenario, Source: SourceProtoImport},
	}, time.Now())
	if len(edges) != 0 {
		t.Fatalf("expected no edges, got %d", len(edges))
	}
}

func TestMergeConfidenceRanking(t *testing.T) {
	ranks := []struct {
		higher string
		lower  string
	}{
		{SourceProtoImport, SourceGoImport},
		{SourceGoImport, SourceResource},
		{SourceResource, SourceDeclared},
		{SourceDeclared, SourceVrooliCLI},
	}
	for _, r := range ranks {
		if ConfidenceFor(r.higher) <= ConfidenceFor(r.lower) {
			t.Fatalf("expected %s > %s", r.higher, r.lower)
		}
	}
}

func TestMergeDeterministicOrder(t *testing.T) {
	now := time.Now()
	edges := Merge([]Contribution{
		{From: "z", To: "a", Kind: KindScenario, Source: SourceDeclared},
		{From: "a", To: "z", Kind: KindScenario, Source: SourceDeclared},
		{From: "a", To: "b", Kind: KindScenario, Source: SourceDeclared},
	}, now)
	if len(edges) != 3 {
		t.Fatalf("expected 3 edges, got %d", len(edges))
	}
	if edges[0].From != "a" || edges[0].To != "b" {
		t.Fatalf("unexpected first edge %+v", edges[0])
	}
	if edges[2].From != "z" {
		t.Fatalf("unexpected last edge %+v", edges[2])
	}
}
