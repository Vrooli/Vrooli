package gates

import (
	"math"
	"testing"
)

func TestScoreCompositionUsesOwnRenderedTreeAndReasonedEscapes(t *testing.T) {
	root := &axObservation{
		DOM: axDOM{Tag: "section", Attributes: map[string]string{"data-rcl-asset": "components.card"}},
		Children: []axObservation{
			{DOM: axDOM{Tag: "div", Attributes: map[string]string{}}},
			{DOM: axDOM{Tag: "div", Attributes: map[string]string{"data-bespoke": "third-party slot"}}},
			{DOM: axDOM{Tag: "div", Attributes: map[string]string{"data-rcl-asset": "components.button"}}, Children: []axObservation{
				{DOM: axDOM{Tag: "span", Attributes: map[string]string{}}},
			}},
		},
	}
	score, total, raw, escapes, missing := scoreComposition(root, "components.card")
	if total != 3 || raw != 1 {
		t.Fatalf("composition counts = total:%d raw:%d, want total:3 raw:1", total, raw)
	}
	if math.Abs(score-2.0/3.0) > 0.000001 {
		t.Fatalf("composition score = %v, want %v", score, 2.0/3.0)
	}
	if len(escapes) != 1 || escapes[0].Reason != "third-party slot" {
		t.Fatalf("escapes = %+v, want one reasoned escape", escapes)
	}
	if len(missing) != 0 {
		t.Fatalf("missing bespoke reasons = %v", missing)
	}
}

func TestScoreCompositionRequiresBespokeReason(t *testing.T) {
	root := &axObservation{DOM: axDOM{Tag: "section", Attributes: map[string]string{"data-rcl-asset": "components.card"}}, Children: []axObservation{
		{DOM: axDOM{Tag: "div", Attributes: map[string]string{"data-bespoke": ""}}},
	}}
	_, _, _, _, missing := scoreComposition(root, "components.card")
	if len(missing) != 1 {
		t.Fatalf("missing bespoke reasons = %v, want one", missing)
	}
}

func TestScoreCompositionRecognizesSharedPrimitiveMarkers(t *testing.T) {
	root := &axObservation{DOM: axDOM{Tag: "section", Attributes: map[string]string{"data-rcl-asset": "components.card"}}, Children: []axObservation{
		{DOM: axDOM{Tag: "span", Attributes: map[string]string{"data-text-style": "label"}}},
		{DOM: axDOM{Tag: "span", Attributes: map[string]string{"data-tone": "success"}}},
		{DOM: axDOM{Tag: "button", Attributes: map[string]string{"data-rcl-control": "true"}}},
		{DOM: axDOM{Tag: "div", Attributes: map[string]string{}}},
	}}
	score, total, raw, _, _ := scoreComposition(root, "components.card")
	if total != 5 || raw != 1 {
		t.Fatalf("composition counts = total:%d raw:%d, want total:5 raw:1", total, raw)
	}
	if math.Abs(score-0.8) > 0.000001 {
		t.Fatalf("composition score = %v, want 0.8", score)
	}
}

func TestCompositionMetadataRoundTripsScoreAndEscapes(t *testing.T) {
	raw := CompositionScoreMetadataJSON(Result{
		CompositionScores: map[string]float64{"components.card": 0.75},
		BespokeEscapes:    []CompositionEscape{{AssetID: "components.card", Reason: "legacy slot"}},
	}, "components.card")
	score, ok, reasons := CompositionScoreMetadata(raw)
	if !ok || score != 0.75 || len(reasons) != 1 || reasons[0] != "legacy slot" {
		t.Fatalf("metadata = %q -> score:%v ok:%v reasons:%v", raw, score, ok, reasons)
	}
}
