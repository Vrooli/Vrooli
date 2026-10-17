package focus

import (
	"context"
	"testing"
)

type fakeFederationHealth struct{}

func (fakeFederationHealth) FederationHealth(context.Context) ([]FederationHealthFinding, error) {
	return []FederationHealthFinding{{ID: "provider.one", Kind: "stuck_provider", Value: "true", Evidence: "stuck"}}, nil
}

func TestFederationHealthGapSourceEmitsSearchHubFocusGap(t *testing.T) {
	gaps, err := NewFederationHealthGapSource(fakeFederationHealth{}).DerivedGaps(context.Background())
	if err != nil {
		t.Fatalf("DerivedGaps: %v", err)
	}
	if len(gaps) != 1 || gaps[0].EvidenceSource != "search-hub" || gaps[0].ID != "condition/federation/provider.one" {
		t.Fatalf("unexpected gaps: %+v", gaps)
	}
	if gaps[0].ConditionStatus != "degraded" {
		t.Fatalf("condition status = %q", gaps[0].ConditionStatus)
	}
}
