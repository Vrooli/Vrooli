package investigation

import "testing"

func TestProjectionLagIsExplicitUntilCompatibleWatermark(t *testing.T) {
	cut := ProjectionCut{Plane: "events", SourceGeneration: "source-1", ClassifierVersion: "events.v1", ThroughRef: "event-10", ThroughSequence: 10}
	lagging := ReadProjectionAtCut(cut, ProjectionObservation{Plane: "events", SourceGeneration: "source-1", ClassifierVersion: "events.v1", ThroughRef: "event-8", ThroughSequence: 8})
	if lagging.Coverage.State != CoverageLagging || lagging.Coverage.Reason == "" {
		t.Fatalf("lagging read=%+v", lagging)
	}
	ready := ReadProjectionAtCut(cut, ProjectionObservation{Plane: "events", SourceGeneration: "source-1", ClassifierVersion: "events.v1", ThroughRef: "event-10", ThroughSequence: 10})
	if ready.Coverage.State != CoverageComplete {
		t.Fatalf("compatible read=%+v", ready)
	}
}

func TestHistoricalCutExcludesLaterProjectionRecords(t *testing.T) {
	cut := ProjectionCut{Plane: "events", SourceGeneration: "source-1", ClassifierVersion: "events.v1", ThroughRef: "event-10", ThroughSequence: 10}
	read := ReadProjectionAtCut(cut, ProjectionObservation{Plane: "events", SourceGeneration: "source-1", ClassifierVersion: "events.v1", ThroughRef: "event-12", ThroughSequence: 12, Records: []ProjectedRecord{{Ref: "event-10", Sequence: 10}, {Ref: "event-11", Sequence: 11}, {Ref: "event-12", Sequence: 12}}})
	if read.Coverage.State != CoverageComplete || len(read.Records) != 1 || len(read.ExcludedAfterCut) != 2 || read.Coverage.Reason == "" {
		t.Fatalf("historical read=%+v", read)
	}
}

func TestReconciliationStopsAfterOneAllowanceAndKeepsIdentities(t *testing.T) {
	result := ReconcileObservations([]ProjectionObservation{
		{Plane: "events", SourceGeneration: "generation-1", ClassifierVersion: "events.v1", ThroughRef: "event-1"},
		{Plane: "events", SourceGeneration: "generation-2", ClassifierVersion: "events.v1", ThroughRef: "event-2"},
		{Plane: "events", SourceGeneration: "generation-3", ClassifierVersion: "events.v1", ThroughRef: "event-3"},
	}, 1)
	if result.Coverage.State != CoveragePartial || result.Reconciliations != 1 || len(result.Identities) != 3 {
		t.Fatalf("reconciliation=%+v", result)
	}
}

func TestRetentionGapDoesNotInventPrunedEvents(t *testing.T) {
	gap := RetentionGap(10, 15)
	if gap.State != CoverageRetentionGap || gap.Reason == "" {
		t.Fatalf("gap=%+v", gap)
	}
	if retained := RetentionGap(15, 15); retained.State != CoverageComplete {
		t.Fatalf("retained=%+v", retained)
	}
}
