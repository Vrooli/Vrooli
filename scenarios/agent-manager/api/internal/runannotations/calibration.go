package runannotations

import (
	"math"
	"sort"
	"strconv"
	"strings"
)

// This file is the calibration arithmetic: pure functions over stored
// probability vectors and gold outcomes. Nothing here calls a model. A
// temperature is fitted from stored vectors, applied at read time to the model
// it was fitted for, and the raw vectors are never rewritten.

const (
	// calibrationEpsilon keeps log-probabilities finite for a model that
	// reports an exact zero.
	calibrationEpsilon = 1e-6
	// ReliabilityBins is the bin count of the ECE and the reliability table.
	ReliabilityBins = 10
	// temperature search bounds: wide enough for an overconfident small model
	// (large T) or an underconfident one (small T), narrow enough that a
	// degenerate gold set cannot fit an absurd value.
	minTemperature = 0.05
	maxTemperature = 20.0
)

// Sample is one gold row paired with the stored vector it will be scored on.
// Probs is the model's distribution over classes; Region lists the classes
// whose combined mass is the label's score. A noul answer is the two-class
// vector [1-p, p] with region {1}, so the same arithmetic covers a binary
// answer, a choice's top class and a score's friction region.
type Sample struct {
	Probs  []float64
	Region []int
	// Fixed marks an answer with no distribution to temper (a rule, a signal, or
	// a stored score that predates level probabilities). Its score is its
	// probability at any temperature.
	Fixed    bool
	Positive bool
}

// Score is the calibrated score at temperature t: the mass of the region in the
// distribution whose log-probabilities are divided by t. At t = 1 it is the
// stored mass. For a two-class vector it equals sigmoid(logit(p) / t).
func (s Sample) Score(t float64) float64 {
	if len(s.Probs) == 0 {
		return 0
	}
	if s.Fixed || t <= 0 || t == 1 {
		return regionMass(s.Probs, s.Region)
	}
	return regionMass(s.Tempered(t), s.Region)
}

// Tempered is the stored distribution read at temperature t: probabilities
// raised to 1/t and renormalized. A fixed answer has nothing to temper.
func (s Sample) Tempered(t float64) []float64 {
	if s.Fixed || t <= 0 || t == 1 || len(s.Probs) == 0 {
		return s.Probs
	}
	logs := make([]float64, len(s.Probs))
	maximum := math.Inf(-1)
	for index, probability := range s.Probs {
		logs[index] = math.Log(math.Max(probability, calibrationEpsilon)) / t
		if logs[index] > maximum {
			maximum = logs[index]
		}
	}
	out := make([]float64, len(logs))
	total := 0.0
	for index, value := range logs {
		out[index] = math.Exp(value - maximum)
		total += out[index]
	}
	for index := range out {
		out[index] /= total
	}
	return out
}

func regionMass(probs []float64, region []int) float64 {
	mass := 0.0
	for _, index := range region {
		if index >= 0 && index < len(probs) {
			mass += probs[index]
		}
	}
	return math.Min(1, math.Max(0, mass))
}

// Confusion is a thresholded outcome count.
type Confusion struct {
	TP, FP, FN, TN int
}

// Precision is TP / (TP + FP); zero when nothing was predicted positive.
func (c Confusion) Precision() float64 {
	if c.TP+c.FP == 0 {
		return 0
	}
	return float64(c.TP) / float64(c.TP+c.FP)
}

// Recall is TP / (TP + FN); zero when there are no gold positives.
func (c Confusion) Recall() float64 {
	if c.TP+c.FN == 0 {
		return 0
	}
	return float64(c.TP) / float64(c.TP+c.FN)
}

// F1 is the harmonic mean of precision and recall.
func (c Confusion) F1() float64 {
	precision, recall := c.Precision(), c.Recall()
	if precision+recall == 0 {
		return 0
	}
	return 2 * precision * recall / (precision + recall)
}

// Evaluate thresholds the calibrated scores. A score at or above the threshold
// is a predicted positive, the rule consumers apply.
func Evaluate(samples []Sample, temperature, threshold float64) Confusion {
	var c Confusion
	for _, sample := range samples {
		predicted := sample.Score(temperature) >= threshold
		switch {
		case sample.Positive && predicted:
			c.TP++
		case sample.Positive:
			c.FN++
		case predicted:
			c.FP++
		default:
			c.TN++
		}
	}
	return c
}

// ReliabilityBin is one row of a reliability table.
type ReliabilityBin struct {
	Lo            float64 `json:"lo"`
	Hi            float64 `json:"hi"`
	Count         int     `json:"count"`
	MeanPredicted float64 `json:"mean_predicted"`
	ObservedRate  float64 `json:"observed_rate"`
}

// Reliability bins scores into ReliabilityBins equal-width bins and reports the
// mean predicted score against the observed positive rate. Empty bins are kept
// (Count 0) so a table always has the same shape.
func Reliability(samples []Sample, temperature float64) []ReliabilityBin {
	bins := make([]ReliabilityBin, ReliabilityBins)
	sums := make([]float64, ReliabilityBins)
	positives := make([]int, ReliabilityBins)
	for index := range bins {
		bins[index].Lo = float64(index) / ReliabilityBins
		bins[index].Hi = float64(index+1) / ReliabilityBins
	}
	for _, sample := range samples {
		score := sample.Score(temperature)
		index := int(score * ReliabilityBins)
		if index >= ReliabilityBins {
			index = ReliabilityBins - 1
		}
		if index < 0 {
			index = 0
		}
		bins[index].Count++
		sums[index] += score
		if sample.Positive {
			positives[index]++
		}
	}
	for index := range bins {
		if bins[index].Count > 0 {
			bins[index].MeanPredicted = sums[index] / float64(bins[index].Count)
			bins[index].ObservedRate = float64(positives[index]) / float64(bins[index].Count)
		}
	}
	return bins
}

// ECE is the expected calibration error over the reliability bins.
func ECE(samples []Sample, temperature float64) float64 {
	if len(samples) == 0 {
		return 0
	}
	total := 0.0
	for _, bin := range Reliability(samples, temperature) {
		if bin.Count == 0 {
			continue
		}
		total += float64(bin.Count) / float64(len(samples)) * math.Abs(bin.MeanPredicted-bin.ObservedRate)
	}
	return total
}

// Brier is the mean squared error between the calibrated score and the gold
// outcome.
func Brier(samples []Sample, temperature float64) float64 {
	if len(samples) == 0 {
		return 0
	}
	total := 0.0
	for _, sample := range samples {
		outcome := 0.0
		if sample.Positive {
			outcome = 1
		}
		difference := sample.Score(temperature) - outcome
		total += difference * difference
	}
	return total / float64(len(samples))
}

// nll is the negative log-likelihood of the gold outcomes under the calibrated
// scores.
func nll(samples []Sample, temperature float64) float64 {
	total := 0.0
	for _, sample := range samples {
		score := math.Min(1-1e-9, math.Max(1e-9, sample.Score(temperature)))
		if sample.Positive {
			total -= math.Log(score)
		} else {
			total -= math.Log(1 - score)
		}
	}
	return total
}

// FitTemperature finds the temperature that minimizes the negative
// log-likelihood of the gold outcomes. It reports false (temperature 1) when
// the data cannot constrain a temperature: no rows, a single outcome class, or
// only fixed answers with no distribution to temper.
func FitTemperature(samples []Sample) (float64, bool) {
	positives, negatives, tempered := 0, 0, 0
	for _, sample := range samples {
		if sample.Positive {
			positives++
		} else {
			negatives++
		}
		if !sample.Fixed {
			tempered++
		}
	}
	if positives == 0 || negatives == 0 || tempered == 0 {
		return 1, false
	}
	// A coarse log-spaced scan finds the basin; golden-section refines it. The
	// scan guards against a non-convex NLL, which a region mass can produce.
	low, high := math.Log(minTemperature), math.Log(maxTemperature)
	const steps = 80
	bestIndex, bestLoss := 0, math.Inf(1)
	for index := 0; index <= steps; index++ {
		logT := low + (high-low)*float64(index)/steps
		if loss := nll(samples, math.Exp(logT)); loss < bestLoss {
			bestIndex, bestLoss = index, loss
		}
	}
	left := low + (high-low)*float64(max(bestIndex-1, 0))/steps
	right := low + (high-low)*float64(min(bestIndex+1, steps))/steps
	golden := (math.Sqrt(5) - 1) / 2
	a, b := left, right
	c, d := b-golden*(b-a), a+golden*(b-a)
	fc, fd := nll(samples, math.Exp(c)), nll(samples, math.Exp(d))
	for iteration := 0; iteration < 60; iteration++ {
		if fc < fd {
			b, d, fd = d, c, fc
			c = b - golden*(b-a)
			fc = nll(samples, math.Exp(c))
		} else {
			a, c, fc = c, d, fd
			d = a + golden*(b-a)
			fd = nll(samples, math.Exp(d))
		}
	}
	return math.Exp((a + b) / 2), true
}

// ChooseThreshold picks the operating point from the calibrated curve. Among
// thresholds that meet both targets it takes the best F1; when none does it
// takes the best F1 overall. Ties prefer the higher threshold (higher
// precision). The result is the midpoint to the next lower distinct score, so
// it does not sit on a score and flip under a rounding difference. It reports
// false when the samples have no positives or no negatives to separate.
func ChooseThreshold(samples []Sample, temperature, targetPrecision, targetRecall float64) (float64, bool) {
	type scored struct {
		score    float64
		positive bool
	}
	rows := make([]scored, 0, len(samples))
	totalPositive := 0
	for _, sample := range samples {
		rows = append(rows, scored{score: sample.Score(temperature), positive: sample.Positive})
		if sample.Positive {
			totalPositive++
		}
	}
	if totalPositive == 0 || totalPositive == len(rows) {
		return 0, false
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].score > rows[j].score })
	type candidate struct {
		score float64
		next  float64
		c     Confusion
	}
	var candidates []candidate
	tp, fp := 0, 0
	for index := 0; index < len(rows); {
		score := rows[index].score
		for index < len(rows) && rows[index].score == score {
			if rows[index].positive {
				tp++
			} else {
				fp++
			}
			index++
		}
		next := 0.0
		if index < len(rows) {
			next = rows[index].score
		}
		candidates = append(candidates, candidate{score: score, next: next, c: Confusion{TP: tp, FP: fp, FN: totalPositive - tp, TN: len(rows) - totalPositive - fp}})
	}
	best, bestMeets := -1, false
	for index, candidate := range candidates {
		meets := candidate.c.Precision() >= targetPrecision && candidate.c.Recall() >= targetRecall
		switch {
		case best < 0:
			best, bestMeets = index, meets
		case meets && !bestMeets:
			best, bestMeets = index, meets
		case meets == bestMeets && candidate.c.F1() > candidates[best].c.F1():
			best = index
		}
	}
	chosen := candidates[best]
	threshold := chosen.score
	if chosen.next > 0 && chosen.next < chosen.score {
		threshold = math.Round((chosen.score+chosen.next)/2*1e6) / 1e6
		if threshold <= chosen.next || threshold > chosen.score {
			threshold = chosen.score
		}
	}
	if threshold <= 0 {
		threshold = chosen.score
	}
	return math.Min(1, threshold), true
}

// sampleFor turns a stored observation into a scorable sample. It reports false
// for an observation that carries no usable probability, so it is excluded from
// the denominators rather than counted as a negative.
func sampleFor(label Label, annotation Annotation) (Sample, bool) {
	if annotation.State != StateObserved && annotation.State != StateCompatible {
		return Sample{}, false
	}
	ruleLike := isRuleMethod(annotation.Method)
	if question := label.Question; question != nil && !ruleLike && len(annotation.Value.Probabilities) > 0 {
		switch question.Type {
		case QuestionScore:
			vector := make([]float64, len(question.Levels))
			for key, probability := range annotation.Value.Probabilities {
				if index, err := strconv.Atoi(strings.TrimSpace(key)); err == nil && index >= 0 && index < len(vector) {
					vector[index] = probability
				}
			}
			if sum(vector) > 0 {
				return Sample{Probs: vector, Region: scoreRegion(label, vector)}, true
			}
		case QuestionChoice:
			vector := make([]float64, len(question.Options))
			for index, option := range question.Options {
				vector[index] = annotation.Value.Probabilities[option.Label]
			}
			if sum(vector) > 0 {
				return Sample{Probs: vector, Region: []int{argmax(vector)}}, true
			}
		}
	}
	probability, ok := scalarProbability(annotation)
	if !ok {
		return Sample{}, false
	}
	if (label.Friction.MinScore != nil || label.Friction.MaxScore != nil) && !ruleLike {
		// A stored score without level probabilities (written before the
		// distribution was kept, or by a method that reports none) keeps the
		// score-plus-confidence rule: the score must lie in the declared
		// direction and its probability is the confidence. A missing score has
		// no direction to judge.
		if annotation.Value.Score == nil {
			return Sample{}, false
		}
		if !scoreInRegion(*annotation.Value.Score, label) {
			probability = 0
		}
		return Sample{Probs: []float64{1 - probability, probability}, Region: []int{1}, Fixed: true}, true
	}
	return Sample{Probs: []float64{1 - probability, probability}, Region: []int{1}, Fixed: ruleLike || label.Question == nil || label.Question.Type != QuestionNoul}, true
}

// scoreRegion is the set of level indices a score label counts as the event:
// the declared friction direction, or the winning level when none is declared.
func scoreRegion(label Label, vector []float64) []int {
	bound, above, directional := frictionScoreBound(label)
	if !directional {
		return []int{argmax(vector)}
	}
	var region []int
	for index := range vector {
		if (above && index >= bound) || (!above && index <= bound) {
			region = append(region, index)
		}
	}
	return region
}

func scalarProbability(annotation Annotation) (float64, bool) {
	switch {
	case annotation.Probability != nil:
		return *annotation.Probability, true
	case annotation.Value.Noul != nil:
		return *annotation.Value.Noul, true
	case annotation.Value.Bool != nil:
		if *annotation.Value.Bool {
			return 1, true
		}
		return 0, true
	}
	return 0, false
}

func sum(values []float64) float64 {
	total := 0.0
	for _, value := range values {
		total += value
	}
	return total
}

func argmax(values []float64) int {
	best := 0
	for index, value := range values {
		if value > values[best] {
			best = index
		}
	}
	return best
}
