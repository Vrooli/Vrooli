package focus

import (
	"context"
	"testing"
)

type fakeDurabilityReader struct{ observations []DurabilityObservation }

func (f fakeDurabilityReader) ReadDurability(context.Context) ([]DurabilityObservation, error) {
	return f.observations, nil
}

func TestDurabilityGapSourceGatesThinSamplesAndReportsLaneCoverage(t *testing.T) {
	source := NewDurabilityGapSource(fakeDurabilityReader{observations: []DurabilityObservation{
		{RunID: "verified-run", Sample: 3, Lane: LaneVerified, Reference: "agent-manager://verified"},
		{RunID: "observed-run", Sample: 4, Lane: LaneObserved, Reference: "agent-manager://observed"},
		{RunID: "unlinked-run", Sample: 5, Lane: LaneUnlinked, Reference: "agent-manager://unlinked"},
	}})
	gaps, err := source.DerivedGaps(context.Background())
	if err != nil || len(gaps) != 1 {
		t.Fatalf("gaps=%#v err=%v", gaps, err)
	}
	if gaps[0].Recurrence != 3 || gaps[0].EvidenceLocator == "" || len(gaps[0].Notes) != 1 {
		t.Fatalf("gap=%#v", gaps[0])
	}
}

func TestDurabilityGapSourceOmitsThinSamples(t *testing.T) {
	source := NewDurabilityGapSource(fakeDurabilityReader{observations: []DurabilityObservation{{RunID: "one", Sample: 9}}})
	gaps, err := source.DerivedGaps(context.Background())
	if err != nil || len(gaps) != 0 {
		t.Fatalf("gaps=%#v err=%v", gaps, err)
	}
}
