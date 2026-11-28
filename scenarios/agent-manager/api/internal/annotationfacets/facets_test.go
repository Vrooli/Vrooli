package annotationfacets

import (
	"testing"

	"agent-manager/internal/runannotations"
)

func ptr(v float64) *float64 { return &v }

func TestCrossesSearchThresholdNeverTreatsMissingProbabilityAsCertain(t *testing.T) {
	label := runannotations.Label{Search: runannotations.Search{Facet: "label:agent.stalled", MinProbability: ptr(0.6)}}
	if crossesSearchThreshold(label, runannotations.AnnotationView{State: runannotations.StateObserved}) {
		t.Fatal("an observed label without a probability crossed a positive threshold")
	}
	if crossesSearchThreshold(label, runannotations.AnnotationView{State: runannotations.StateObserved, Probability: ptr(0.59)}) {
		t.Fatal("a below-threshold probability was tagged")
	}
	if !crossesSearchThreshold(label, runannotations.AnnotationView{State: runannotations.StateObserved, Probability: ptr(0.6)}) {
		t.Fatal("a threshold-equal probability was not tagged")
	}
	if crossesSearchThreshold(label, runannotations.AnnotationView{State: runannotations.StateCompatible, Probability: ptr(0.9)}) == false {
		t.Fatal("a compatible observation at the threshold was not tagged")
	}
	if crossesSearchThreshold(label, runannotations.AnnotationView{State: runannotations.StateNotObserved, Probability: ptr(0.9)}) {
		t.Fatal("a not-observed label was tagged")
	}
}

func TestCrossesSearchThresholdWithoutAThresholdAllowsObserved(t *testing.T) {
	label := runannotations.Label{Search: runannotations.Search{Facet: "label:run.type"}}
	if !crossesSearchThreshold(label, runannotations.AnnotationView{State: runannotations.StateObserved}) {
		t.Fatal("a deterministic label with no probability and no threshold was not tagged")
	}
}
