package focus

import (
	"context"
	"testing"
)

func TestRouterQualityGapSourceCarriesAStableSharedCause(t *testing.T) {
	source := NewRouterQualityGapSource(func(context.Context) ([]RouterQualityFinding, error) {
		return []RouterQualityFinding{{Projection: ProjectionAnswer, CellID: "7", Owner: "ui-health.surfaces", Message: "router quality debt"}}, nil
	})
	gaps, err := source.DerivedGaps(context.Background())
	if err != nil {
		t.Fatalf("DerivedGaps: %v", err)
	}
	if len(gaps) != 1 || gaps[0].CauseKey != "search-hub/router_quality_debt" || gaps[0].SourceCellID != "7" {
		t.Fatalf("unexpected router quality gap: %+v", gaps)
	}
}

func TestSubstrateGapSourceNamesUnhealthyResource(t *testing.T) {
	source := NewSubstrateGapSource(func(context.Context) ([]SubstrateObservation, error) {
		return []SubstrateObservation{{Name: "reranker", Healthy: false, Reason: "leg unavailable"}, {Name: "qdrant", Healthy: true}}, nil
	})
	gaps, err := source.DerivedGaps(context.Background())
	if err != nil {
		t.Fatalf("DerivedGaps: %v", err)
	}
	if len(gaps) != 1 || gaps[0].ID != "condition/substrate/reranker" || gaps[0].Notes[0] != "leg unavailable" {
		t.Fatalf("unexpected substrate gaps: %+v", gaps)
	}
}
