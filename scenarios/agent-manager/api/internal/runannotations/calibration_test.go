package runannotations

import (
	"math"
	"math/rand"
	"testing"
)

// overconfidentSamples builds noul samples from a model that reports
// sigmoid(k * logit(q)) when the true event probability is q: k > 1 is an
// overconfident model, whose ideal temperature is k.
func overconfidentSamples(n int, k float64, seed int64) []Sample {
	random := rand.New(rand.NewSource(seed))
	samples := make([]Sample, 0, n)
	for range n {
		q := 0.02 + 0.96*random.Float64()
		reported := 1 / (1 + math.Exp(-k*math.Log(q/(1-q))))
		samples = append(samples, Sample{
			Probs: []float64{1 - reported, reported}, Region: []int{1}, Positive: random.Float64() < q,
		})
	}
	return samples
}

func TestFitTemperatureRecoversTheModelsOverconfidence(t *testing.T) {
	samples := overconfidentSamples(6000, 3, 1)
	temperature, fitted := FitTemperature(samples)
	if !fitted {
		t.Fatal("a mixed-outcome set was not fitted")
	}
	if temperature < 2.5 || temperature > 3.6 {
		t.Fatalf("temperature = %.3f, want about 3", temperature)
	}
	if after, before := ECE(samples, temperature), ECE(samples, 1); after >= before {
		t.Fatalf("ECE %.4f -> %.4f, want calibration to reduce it", before, after)
	}
	if after, before := Brier(samples, temperature), Brier(samples, 1); after >= before {
		t.Fatalf("Brier %.4f -> %.4f, want calibration to reduce it", before, after)
	}
}

func TestFitTemperatureLeavesAnUnderconfidentModelSharper(t *testing.T) {
	samples := overconfidentSamples(6000, 0.5, 2)
	temperature, fitted := FitTemperature(samples)
	if !fitted || temperature > 0.7 {
		t.Fatalf("temperature = %.3f (fitted %v), want about 0.5 for an underconfident model", temperature, fitted)
	}
}

func TestFitTemperatureRefusesDataThatCannotConstrainIt(t *testing.T) {
	one := []Sample{{Probs: []float64{0.2, 0.8}, Region: []int{1}, Positive: true}, {Probs: []float64{0.3, 0.7}, Region: []int{1}, Positive: true}}
	if temperature, fitted := FitTemperature(one); fitted || temperature != 1 {
		t.Fatalf("a single outcome class fitted T=%v (%v)", temperature, fitted)
	}
	fixed := []Sample{
		{Probs: []float64{0, 1}, Region: []int{1}, Fixed: true, Positive: true},
		{Probs: []float64{1, 0}, Region: []int{1}, Fixed: true},
	}
	if temperature, fitted := FitTemperature(fixed); fitted || temperature != 1 {
		t.Fatalf("fixed rule answers fitted T=%v (%v)", temperature, fitted)
	}
	if _, fitted := FitTemperature(nil); fitted {
		t.Fatal("no rows were fitted")
	}
}

func TestTwoClassScoreIsTheTemperedSigmoid(t *testing.T) {
	sample := Sample{Probs: []float64{0.1, 0.9}, Region: []int{1}}
	logit := math.Log(0.9 / 0.1)
	for _, temperature := range []float64{0.5, 1, 2, 4} {
		want := 1 / (1 + math.Exp(-logit/temperature))
		if got := sample.Score(temperature); math.Abs(got-want) > 1e-9 {
			t.Fatalf("score at T=%v = %v, want %v", temperature, got, want)
		}
	}
}

func TestRegionMassOfAMultiLevelVectorIsTempered(t *testing.T) {
	sample := Sample{Probs: []float64{0.05, 0.15, 0.30, 0.30, 0.20}, Region: []int{2, 3, 4}}
	if mass := sample.Score(1); math.Abs(mass-0.8) > 1e-9 {
		t.Fatalf("raw mass = %v, want 0.8", mass)
	}
	flatter := sample.Score(3)
	if flatter >= 0.8 || flatter <= 0.5 {
		t.Fatalf("mass at T=3 = %v, want it pulled toward the uniform 0.6", flatter)
	}
	tempered := sample.Tempered(3)
	total := 0.0
	for _, value := range tempered {
		total += value
	}
	if math.Abs(total-1) > 1e-9 {
		t.Fatalf("tempered vector sums to %v", total)
	}
}

func TestFixedAnswersIgnoreTemperature(t *testing.T) {
	sample := Sample{Probs: []float64{0, 1}, Region: []int{1}, Fixed: true}
	if sample.Score(5) != 1 || sample.Score(0.2) != 1 {
		t.Fatal("a rule's answer moved with temperature")
	}
}

func TestChooseThresholdMeetsBothTargetsWhenTheCurveAllowsIt(t *testing.T) {
	var samples []Sample
	for _, score := range []float64{0.95, 0.9, 0.85, 0.8, 0.7} {
		samples = append(samples, Sample{Probs: []float64{1 - score, score}, Region: []int{1}, Positive: true})
	}
	for _, score := range []float64{0.45, 0.3, 0.2, 0.1} {
		samples = append(samples, Sample{Probs: []float64{1 - score, score}, Region: []int{1}})
	}
	threshold, ok := ChooseThreshold(samples, 1, 0.8, 0.8)
	if !ok {
		t.Fatal("no threshold chosen")
	}
	if threshold <= 0.45 || threshold > 0.7 {
		t.Fatalf("threshold = %v, want one between the classes (the midpoint 0.575)", threshold)
	}
	confusion := Evaluate(samples, 1, threshold)
	if confusion.Precision() != 1 || confusion.Recall() != 1 {
		t.Fatalf("at %v: %+v", threshold, confusion)
	}
}

func TestChooseThresholdFallsBackToBestF1WhenNoPointMeetsTheTargets(t *testing.T) {
	samples := []Sample{
		{Probs: []float64{0.3, 0.7}, Region: []int{1}, Positive: true},
		{Probs: []float64{0.4, 0.6}, Region: []int{1}},
		{Probs: []float64{0.5, 0.5}, Region: []int{1}, Positive: true},
		{Probs: []float64{0.8, 0.2}, Region: []int{1}},
	}
	threshold, ok := ChooseThreshold(samples, 1, 0.99, 0.99)
	if !ok || threshold <= 0 || threshold > 1 {
		t.Fatalf("threshold = %v (%v)", threshold, ok)
	}
	if _, ok := ChooseThreshold([]Sample{{Probs: []float64{0.3, 0.7}, Region: []int{1}, Positive: true}}, 1, 0.8, 0.7); ok {
		t.Fatal("a single outcome class chose a threshold")
	}
}

func TestReliabilityKeepsEveryBinAndConservesTheRows(t *testing.T) {
	samples := overconfidentSamples(500, 1, 3)
	bins := Reliability(samples, 1)
	if len(bins) != ReliabilityBins {
		t.Fatalf("bins = %d, want a fixed-shape table of %d", len(bins), ReliabilityBins)
	}
	total := 0
	for _, bin := range bins {
		total += bin.Count
		if bin.Count > 0 && (bin.MeanPredicted < bin.Lo-1e-9 || bin.MeanPredicted > bin.Hi+1e-9) {
			t.Fatalf("bin %+v has a mean outside its range", bin)
		}
	}
	if total != len(samples) {
		t.Fatalf("binned %d of %d rows", total, len(samples))
	}
	top := Reliability([]Sample{{Probs: []float64{0, 1}, Region: []int{1}, Fixed: true, Positive: true}}, 1)
	if top[ReliabilityBins-1].Count != 1 {
		t.Fatalf("a score of exactly 1 was not placed in the last bin: %+v", top)
	}
}

func TestSampleForReadsEachLabelShape(t *testing.T) {
	score := Label{ID: "s", Question: &Question{Type: QuestionScore, Levels: []string{"0", "1", "2"}}, Friction: Friction{Enabled: true, MinScore: floatPtr(1)}}
	row := Annotation{State: StateObserved, Method: "DECISION_MODEL", Value: Value{Kind: "score", Probabilities: map[string]float64{"0": 0.2, "1": 0.3, "2": 0.5}}}
	sample, ok := sampleFor(score, row)
	if !ok || len(sample.Region) != 2 || sample.Fixed {
		t.Fatalf("directional score sample = %+v (%v)", sample, ok)
	}
	if mass := sample.Score(1); math.Abs(mass-0.8) > 1e-9 {
		t.Fatalf("friction mass = %v, want 0.8", mass)
	}
	choice := Label{ID: "c", Question: &Question{Type: QuestionChoice, Options: []Option{{Label: "a"}, {Label: "b"}}}}
	pick := Annotation{State: StateObserved, Method: "DECISION_MODEL", Value: Value{Kind: "choice", Choice: "b", Probabilities: map[string]float64{"a": 0.3, "b": 0.7}}}
	if sample, ok := sampleFor(choice, pick); !ok || len(sample.Region) != 1 || sample.Region[0] != 1 {
		t.Fatalf("choice sample = %+v (%v), want the winning option as the event", sample, ok)
	}
	rule := Annotation{State: StateObserved, Method: "deterministic", Value: Value{Kind: "bool", Bool: boolPointer(true)}, Probability: floatPtr(1)}
	if sample, ok := sampleFor(Label{ID: "r"}, rule); !ok || !sample.Fixed {
		t.Fatalf("rule sample = %+v (%v), want a fixed answer", sample, ok)
	}
	enum := Annotation{State: StateObserved, Method: "LLM_ENUM", Value: Value{Kind: "choice", Choice: "a"}}
	if _, ok := sampleFor(choice, enum); ok {
		t.Fatal("an LLM enum answer with no distribution was scored")
	}
	unavailable := Annotation{State: StateUnavailable, Probability: floatPtr(0.9)}
	if _, ok := sampleFor(choice, unavailable); ok {
		t.Fatal("an unavailable row was scored")
	}
}

func boolPointer(value bool) *bool { return &value }
