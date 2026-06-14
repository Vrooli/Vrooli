package roadmap

import (
	"context"
	"testing"

	graphdomain "tech-tree-designer/internal/graph"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/graph"
)

type fakeGraphProvider struct {
	graph *graphv1.TechTreeGraph
}

func (f fakeGraphProvider) Describe(context.Context, graphdomain.SourceRequest) (*graphv1.TechTreeGraph, error) {
	return f.graph, nil
}

func TestGetProgressRollsUpNodesBySectorTier(t *testing.T) {
	// [REQ:TTD-ROADMAP-001] Progress derives from graph node kind and stability.
	service := NewService(nil, fakeGraphProvider{graph: &graphv1.TechTreeGraph{Nodes: []*graphv1.TechNode{
		{Scenario: "live-one", Kind: graphv1.NodeKind_NODE_KIND_LIVE, Sector: "engineering", Tier: "foundation", Stability: []string{"stable"}},
		{Scenario: "live-two", Kind: graphv1.NodeKind_NODE_KIND_LIVE, Sector: "engineering", Tier: "foundation", Stability: []string{"beta"}},
		{Scenario: "planned-one", Kind: graphv1.NodeKind_NODE_KIND_PLANNED, Sector: "engineering", Tier: "integration", Stability: []string{"experimental"}},
	}}})

	progress, err := service.GetProgress(context.Background(), ProgressFilter{})
	if err != nil {
		t.Fatalf("GetProgress() error = %v", err)
	}
	if len(progress.GetBuckets()) != 2 {
		t.Fatalf("len(buckets) = %d, want 2", len(progress.GetBuckets()))
	}
	first := progress.GetBuckets()[0]
	if first.GetSector() != "engineering" || first.GetTier() != "foundation" {
		t.Fatalf("first bucket = %s/%s, want engineering/foundation", first.GetSector(), first.GetTier())
	}
	if first.GetLive() != 2 || first.GetBeta() != 1 || first.GetStable() != 1 || first.GetPlanned() != 0 {
		t.Fatalf("first bucket counts = planned=%d live=%d beta=%d stable=%d", first.GetPlanned(), first.GetLive(), first.GetBeta(), first.GetStable())
	}
	second := progress.GetBuckets()[1]
	if second.GetPlanned() != 1 || second.GetLive() != 0 {
		t.Fatalf("second bucket counts = planned=%d live=%d, want planned=1 live=0", second.GetPlanned(), second.GetLive())
	}
}

func TestGetProgressFiltersByTier(t *testing.T) {
	// [REQ:TTD-ROADMAP-001] Progress supports roadmap tier filters.
	service := NewService(nil, fakeGraphProvider{graph: &graphv1.TechTreeGraph{Nodes: []*graphv1.TechNode{
		{Scenario: "live-one", Kind: graphv1.NodeKind_NODE_KIND_LIVE, Sector: "engineering", Tier: "foundation"},
		{Scenario: "planned-one", Kind: graphv1.NodeKind_NODE_KIND_PLANNED, Sector: "engineering", Tier: "integration"},
	}}})

	progress, err := service.GetProgress(context.Background(), ProgressFilter{Tier: "integration"})
	if err != nil {
		t.Fatalf("GetProgress() error = %v", err)
	}
	if len(progress.GetBuckets()) != 1 || progress.GetBuckets()[0].GetTier() != "integration" {
		t.Fatalf("buckets = %+v, want one integration bucket", progress.GetBuckets())
	}
}
