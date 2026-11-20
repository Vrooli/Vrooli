package runannotations

import (
	"path/filepath"
	"testing"
)

func TestAnnotationIsPositiveHonorsSignedScorePolarity(t *testing.T) {
	confidence := 0.9
	negative := -1.5
	positive := 1.5
	ceiling := -1.0
	threshold := 0.6
	label := Label{
		ID: "operator.valence",
		Friction: Friction{
			Enabled: true, MinProbability: &threshold, MaxScore: &ceiling,
		},
	}
	cases := []struct {
		name        string
		score       *float64
		probability *float64
		want        bool
	}{
		{name: "strong negative is friction", score: &negative, probability: &confidence, want: true},
		{name: "strong positive is not friction", score: &positive, probability: &confidence, want: false},
		{name: "low confidence is not friction", score: &negative, probability: floatPtr(0.3), want: false},
		{name: "missing score is not friction", score: nil, probability: &confidence, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			annotation := AnnotationView{Value: Value{Kind: "score", Score: tc.score}, Probability: tc.probability}
			if got := annotationIsPositive(label, annotation); got != tc.want {
				t.Fatalf("positive = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAnnotationIsPositiveWithoutScoreCeilingUsesConfidence(t *testing.T) {
	confidence := 0.9
	threshold := 0.6
	label := Label{ID: "agent.spinning", Friction: Friction{Enabled: true, MinProbability: &threshold}}
	if !annotationIsPositive(label, AnnotationView{Probability: &confidence}) {
		t.Fatal("a confident probability-only label was not positive")
	}
	if annotationIsPositive(label, AnnotationView{Probability: floatPtr(0.2)}) {
		t.Fatal("a low-confidence probability-only label was positive")
	}
}

func floatPtr(value float64) *float64 { return &value }

func TestAnnotationViewCarriesTheLabelSelectionRule(t *testing.T) {
	pack, err := LoadCompiled(filepath.Join("..", "..", "..", ".vrooli", "annotations", "default.json"))
	if err != nil {
		t.Fatal(err)
	}
	service := &Service{pack: pack}
	score := -1.5
	view := service.annotationView(Annotation{
		LabelID: "operator.valence", State: StateObserved,
		Value: Value{Kind: "score", Score: &score},
	})
	if !view.Friction.Enabled || view.Friction.MaxScore == nil || *view.Friction.MaxScore != -1 {
		t.Fatalf("view friction = %+v, want the declared signed-score ceiling", view.Friction)
	}
}

func TestAnnotationIsPositiveReadsMassInTheDeclaredScoreDirection(t *testing.T) {
	threshold := 0.6
	floor := 2.0
	label := Label{
		ID: "agent.spinning",
		Question: &Question{
			Type: QuestionScore, Offset: 0,
			Levels: []string{"0", "1", "2", "3", "4"},
		},
		Friction: Friction{Enabled: true, MinProbability: &threshold, MinScore: &floor},
	}
	spinning := 2.4
	// P(score >= 2) = 0.8: clearly in the friction direction even though no
	// single level carries a high probability.
	distribution := map[string]float64{"0": 0.05, "1": 0.15, "2": 0.30, "3": 0.30, "4": 0.20}
	if !annotationIsPositive(label, AnnotationView{Value: Value{Kind: "score", Score: &spinning, Probabilities: distribution}}) {
		t.Fatal("mass above the declared score floor was not friction")
	}
	// P(score >= 2) = 0.2: a mild score is not friction however confident the
	// top level is.
	mild := 1.8
	low := map[string]float64{"0": 0.40, "1": 0.40, "2": 0.10, "3": 0.06, "4": 0.04}
	if annotationIsPositive(label, AnnotationView{Value: Value{Kind: "score", Score: &mild, Probabilities: low}, Probability: floatPtr(0.9)}) {
		t.Fatal("a mild score counted as friction because its top level was confident")
	}
}

func TestAnnotationIsPositiveMapsSignedOffsetIntoLevelIndices(t *testing.T) {
	threshold := 0.6
	ceiling := -1.0
	label := Label{
		ID: "operator.valence",
		Question: &Question{
			Type: QuestionScore, Offset: -2,
			Levels: []string{"-2", "-1", "0", "+1", "+2"},
		},
		Friction: Friction{Enabled: true, MinProbability: &threshold, MaxScore: &ceiling},
	}
	negative := -1.6
	// Indices 0 and 1 (scores -2 and -1) carry 0.7 of the mass.
	distribution := map[string]float64{"0": 0.40, "1": 0.30, "2": 0.20, "3": 0.06, "4": 0.04}
	if !annotationIsPositive(label, AnnotationView{Value: Value{Kind: "score", Score: &negative, Probabilities: distribution}}) {
		t.Fatal("mass at or below the signed ceiling was not friction")
	}
	positive := 1.6
	if annotationIsPositive(label, AnnotationView{Value: Value{Kind: "score", Score: &positive, Probabilities: distribution}, Probability: floatPtr(0.9)}) {
		t.Fatal("praise counted as friction because a level was confident")
	}
}

func TestCoverageCountsTheFrictionRegionMassFromStoredViews(t *testing.T) {
	pack, err := LoadCompiled(filepath.Join("..", "..", "..", ".vrooli", "annotations", "default.json"))
	if err != nil {
		t.Fatal(err)
	}
	service := &Service{pack: pack}
	negative := -1.6
	views := []RunView{{RunID: "run-1", Labels: []AnnotationView{
		{ID: "operator.valence", State: StateObserved, Value: Value{
			Kind: "score", Score: &negative,
			// Levels -2 and -1 carry 0.75 of the mass, above the 0.6 threshold.
			Probabilities: map[string]float64{"0": 0.55, "1": 0.20, "2": 0.15, "3": 0.06, "4": 0.04},
		}},
		{ID: "operator.satisfaction", State: StateObserved, Probability: floatPtr(0.9)},
	}}}
	coverage := service.coverageFromViews(views)
	for _, label := range coverage {
		switch label.ID {
		case "operator.valence":
			if label.Positives != 1 {
				t.Fatalf("valence positives = %d, want the mass at or below max_score", label.Positives)
			}
		case "operator.satisfaction":
			if label.Positives != 0 {
				t.Fatalf("satisfaction positives = %d, want 0 for a label with friction disabled", label.Positives)
			}
		}
	}
}

func TestCoverageNeverRatesANonFrictionLabel(t *testing.T) {
	label := Label{
		ID:       "run.type",
		Friction: Friction{Enabled: false},
		Gate: Gate{
			Precision: 0.9, Recall: 0.9, MinGold: 1,
			Measured: &GateMeasurement{Precision: 0.9, Recall: 0.9, Gold: 10},
		},
	}
	row := coverageLabel(label, StateCounts{Observed: 4}, 3)
	if row.Rate != nil {
		t.Fatalf("a classification label reported a friction rate: %v", *row.Rate)
	}
	if annotationIsPositive(label, AnnotationView{Probability: floatPtr(0.95)}) {
		t.Fatal("a confident classification answer counted as a friction positive")
	}
}
