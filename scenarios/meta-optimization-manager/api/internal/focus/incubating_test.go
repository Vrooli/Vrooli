package focus

import (
	"context"
	"testing"
)

func TestIncubatingGapSourceEmitsOneProviderScopedAdoptionGap(t *testing.T) {
	source := NewIncubatingGapSource(func(context.Context) ([]IncubatingProvider, error) {
		return []IncubatingProvider{{
			ProviderID: "synthetic.records",
			DeclaredAt: "2026-08-13T20:38:00Z",
			NextAction: "run a recent passing evaluation",
		}}, nil
	})
	gaps, err := source.DerivedGaps(context.Background())
	if err != nil {
		t.Fatalf("DerivedGaps: %v", err)
	}
	if len(gaps) != 1 {
		t.Fatalf("got %d gaps, want 1", len(gaps))
	}
	got := gaps[0]
	if got.ID != "condition/incubating/synthetic.records" || got.EvidenceSource != "search-hub" {
		t.Fatalf("unexpected gap identity/provenance: %+v", got)
	}
	if len(got.ProviderIDs) != 1 || got.ProviderIDs[0] != "synthetic.records" {
		t.Fatalf("unexpected provider scope: %+v", got.ProviderIDs)
	}
	if len(got.Notes) != 2 || got.Notes[1] != "next_action=run a recent passing evaluation" {
		t.Fatalf("unexpected adoption notes: %+v", got.Notes)
	}
}
