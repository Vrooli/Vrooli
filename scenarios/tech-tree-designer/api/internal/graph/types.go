package graph

import (
	"context"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/graph"
)

const (
	SourceSDA = "scenario-dependency-analyzer"
)

type SourceRequest struct {
	ScenarioFilter  []string
	Limit           int32
	StabilityFilter string
	MaxScenarioHops int32
}

type GraphSource interface {
	Graph(ctx context.Context, req SourceRequest) (*graphv1.TechTreeGraph, error)
}

type PlannedGraphSource interface {
	PlannedGraph(ctx context.Context) (*graphv1.TechTreeGraph, error)
}
