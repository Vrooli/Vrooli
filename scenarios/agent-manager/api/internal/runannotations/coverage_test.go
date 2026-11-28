package runannotations

import (
	"testing"
)

func floatPtr(value float64) *float64 { return &value }

// scoreFrictionPack is a two-label pack with a signed score friction label and a
// noul label whose friction is off: the shapes the coverage rules read. It is a
// fixture on purpose; the default pack no longer declares a score label.
func scoreFrictionPack(t *testing.T) *CompiledPack {
	t.Helper()
	pack, err := LoadCompiled(packPathForTest(t, `[
	{"id":"operator.valence","group":"operator","scope":"run","method":"decision",
	 "question":{"type":"score","instructions":"x","levels":["-2","-1","0","+1","+2"],"offset":-2},
	 "evidence":[],"gate":{"precision":0.8,"recall":0.7,"min_gold":40},
	 "search":{"facet":"label:operator.valence","min_probability":0.6},
	 "friction":{"enabled":true,"min_probability":0.6,"cost_unit":"run","importance":"high","max_score":-1},
	 "compatibility":{"change":"breaking"}},
	{"id":"operator.satisfaction","group":"operator","scope":"run","method":"decision",
	 "question":{"type":"noul","instructions":"x","true_description":"t","false_description":"f"},
	 "evidence":[],"gate":{"precision":0.8,"recall":0.7,"min_gold":40},
	 "search":{"facet":"label:operator.satisfaction","min_probability":0.6},
	 "friction":{"enabled":false,"cost_unit":"run","importance":"low"},
	 "compatibility":{"change":"breaking"}}]`))
	if err != nil {
		t.Fatal(err)
	}
	return pack
}

func observedScore(score float64, probability *float64, distribution map[string]float64) Annotation {
	return Annotation{
		State: StateObserved, Method: "DECISION_MODEL", Model: "m", ModelVersion: "r",
		Value:       Value{Kind: "score", Score: &score, Probabilities: distribution},
		Probability: probability,
	}
}

func TestFrictionPositiveHonorsSignedScorePolarity(t *testing.T) {
	ceiling, threshold := -1.0, 0.6
	label := Label{ID: "operator.valence", Friction: Friction{Enabled: true, MinProbability: &threshold, MaxScore: &ceiling}}
	cases := []struct {
		name  string
		score float64
		prob  float64
		want  bool
	}{
		{name: "strong negative is friction", score: -1.5, prob: 0.9, want: true},
		{name: "strong positive is not friction", score: 1.5, prob: 0.9, want: false},
		{name: "low confidence is not friction", score: -1.5, prob: 0.3, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			annotation := observedScore(tc.score, floatPtr(tc.prob), nil)
			if got := isFrictionPositive(label, annotation, 1, 0); got != tc.want {
				t.Fatalf("positive = %v, want %v", got, tc.want)
			}
		})
	}
	missing := Annotation{State: StateObserved, Method: "DECISION_MODEL", Value: Value{Kind: "score"}, Probability: floatPtr(0.9)}
	if isFrictionPositive(label, missing, 1, 0) {
		t.Fatal("a score label with no score counted as friction")
	}
}

func TestFrictionPositiveWithoutScoreCeilingUsesConfidence(t *testing.T) {
	threshold := 0.6
	label := Label{ID: "agent.stalled", Friction: Friction{Enabled: true, MinProbability: &threshold}}
	confident := Annotation{State: StateObserved, Method: "DECISION_MODEL", Value: Value{Kind: "noul", Noul: floatPtr(0.9)}, Probability: floatPtr(0.9)}
	if !isFrictionPositive(label, confident, 1, 0) {
		t.Fatal("a confident probability-only label was not positive")
	}
	weak := Annotation{State: StateObserved, Method: "DECISION_MODEL", Value: Value{Kind: "noul", Noul: floatPtr(0.2)}, Probability: floatPtr(0.2)}
	if isFrictionPositive(label, weak, 1, 0) {
		t.Fatal("a low-confidence probability-only label was positive")
	}
}

func TestMeasuredThresholdReplacesTheDeclaredOne(t *testing.T) {
	declared := 0.6
	label := Label{ID: "agent.stalled", Friction: Friction{Enabled: true, MinProbability: &declared}}
	annotation := Annotation{State: StateObserved, Method: "DECISION_MODEL", Value: Value{Kind: "noul", Noul: floatPtr(0.45)}, Probability: floatPtr(0.45)}
	if isFrictionPositive(label, annotation, 1, 0) {
		t.Fatal("0.45 reached the declared 0.6")
	}
	if !isFrictionPositive(label, annotation, 1, 0.4) {
		t.Fatal("0.45 did not reach the measured 0.4 threshold")
	}
}

func TestTemperatureIsAppliedToTheStoredVectorAtReadTime(t *testing.T) {
	threshold := 0.6
	label := Label{ID: "agent.stalled", Question: &Question{Type: QuestionNoul}, Friction: Friction{Enabled: true, MinProbability: &threshold}}
	// An overconfident model: raw 0.9 is only 0.75 after T = 2 (logit 2.197 / 2).
	annotation := Annotation{State: StateObserved, Method: "DECISION_MODEL", Value: Value{Kind: "noul", Noul: floatPtr(0.9)}, Probability: floatPtr(0.9)}
	sample, ok := sampleFor(label, annotation)
	if !ok {
		t.Fatal("no sample")
	}
	if got := sample.Score(2); got < 0.74 || got > 0.76 {
		t.Fatalf("score at T=2 = %v, want ~0.75", got)
	}
	if got := sample.Score(1); got != 0.9 {
		t.Fatalf("score at T=1 = %v, want the stored 0.9", got)
	}
	if annotation.Value.Noul == nil || *annotation.Value.Noul != 0.9 {
		t.Fatal("reading at a temperature rewrote the stored vector")
	}
	if !isFrictionPositive(label, annotation, 1, 0) || isFrictionPositive(label, annotation, 8, 0.6) {
		t.Fatal("temperature did not move the positive decision")
	}
}

func TestAnnotationViewCarriesTheLabelSelectionRule(t *testing.T) {
	pack := scoreFrictionPack(t)
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

func TestFrictionPositiveReadsMassInTheDeclaredScoreDirection(t *testing.T) {
	threshold, floor := 0.6, 2.0
	label := Label{
		ID: "agent.stalled",
		Question: &Question{
			Type: QuestionScore, Levels: []string{"0", "1", "2", "3", "4"},
		},
		Friction: Friction{Enabled: true, MinProbability: &threshold, MinScore: &floor},
	}
	// P(score >= 2) = 0.8: clearly in the friction direction even though no
	// single level carries a high probability.
	spinning := observedScore(2.4, nil, map[string]float64{"0": 0.05, "1": 0.15, "2": 0.30, "3": 0.30, "4": 0.20})
	if !isFrictionPositive(label, spinning, 1, 0) {
		t.Fatal("mass above the declared score floor was not friction")
	}
	// P(score >= 2) = 0.2: a mild score is not friction however confident the
	// top level is.
	mild := observedScore(1.8, floatPtr(0.9), map[string]float64{"0": 0.40, "1": 0.40, "2": 0.10, "3": 0.06, "4": 0.04})
	if isFrictionPositive(label, mild, 1, 0) {
		t.Fatal("a mild score counted as friction because its top level was confident")
	}
}

func TestFrictionPositiveMapsSignedOffsetIntoLevelIndices(t *testing.T) {
	threshold, ceiling := 0.6, -1.0
	label := Label{
		ID: "operator.valence",
		Question: &Question{
			Type: QuestionScore, Offset: -2, Levels: []string{"-2", "-1", "0", "+1", "+2"},
		},
		Friction: Friction{Enabled: true, MinProbability: &threshold, MaxScore: &ceiling},
	}
	// Indices 0 and 1 (scores -2 and -1) carry 0.7 of the mass.
	distribution := map[string]float64{"0": 0.40, "1": 0.30, "2": 0.20, "3": 0.06, "4": 0.04}
	if !isFrictionPositive(label, observedScore(-1.6, nil, distribution), 1, 0) {
		t.Fatal("mass at or below the signed ceiling was not friction")
	}
	praise := map[string]float64{"0": 0.02, "1": 0.03, "2": 0.10, "3": 0.30, "4": 0.55}
	if isFrictionPositive(label, observedScore(1.4, floatPtr(0.55), praise), 1, 0) {
		t.Fatal("praise counted as friction")
	}
}

func TestCoverageCountsTheFrictionRegionMassFromStoredViews(t *testing.T) {
	pack := scoreFrictionPack(t)
	service := &Service{pack: pack}
	hash := pack.LabelHashes["operator.valence"]
	satisfaction := pack.LabelHashes["operator.satisfaction"]
	negative := -1.6
	views := []RunView{{RunID: "run-1", Labels: []AnnotationView{
		{ID: "operator.valence", State: StateObserved, DefinitionHash: hash, EvaluatorMethod: "DECISION_MODEL", Model: "m", ResourceRole: "r", Value: Value{
			Kind: "score", Score: &negative,
			// Levels -2 and -1 carry 0.75 of the mass, above the 0.6 threshold.
			Probabilities: map[string]float64{"0": 0.55, "1": 0.20, "2": 0.15, "3": 0.06, "4": 0.04},
		}},
		{ID: "operator.satisfaction", State: StateObserved, DefinitionHash: satisfaction, EvaluatorMethod: "DECISION_MODEL", Probability: floatPtr(0.9)},
	}}}
	for _, label := range service.coverageFromViews(views) {
		switch label.ID {
		case "operator.valence":
			if label.Positives != 1 {
				t.Fatalf("valence positives = %d, want the mass at or below max_score", label.Positives)
			}
			if label.Rate != nil {
				t.Fatal("an unmeasured model produced a rate")
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
		ID: "run.type", Friction: Friction{Enabled: false},
		Gate: Gate{Precision: 0.9, Recall: 0.9, MinGold: 1},
	}
	service := &Service{pack: &CompiledPack{LabelHashes: map[string]string{"run.type": "h"}}}
	acc := &labelAccumulator{
		counts: StateCounts{Observed: 4},
		models: map[ModelKey]*modelAccumulator{{Model: RuleModel}: {observed: 4, positives: 3}},
	}
	label.Gate.Measurements = []Measurement{{
		DefinitionHash: "h", Model: RuleModel, Precision: 0.9, Recall: 0.9, Gold: 10, Split: SplitHoldout,
	}}
	row := service.coverageLabel(label, acc)
	if row.Rate != nil {
		t.Fatalf("a classification label reported a friction rate: %v", *row.Rate)
	}
}
