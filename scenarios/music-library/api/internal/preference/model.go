// Package preference owns the listener's editable preference model. It keeps
// explicit controls separate from pairwise evidence and never manufactures
// feature values that are absent from the catalog.
package preference

import (
	"errors"
	"math"
	"sort"
	"strings"
	"time"
)

const ModelVersion = "taste-linear-v1"
const FeatureSpaceVersion = "music-evidence-v2"

var ErrInvalid = errors.New("invalid preference input")

type ControlMode string

const (
	ControlBoost    ControlMode = "boost"
	ControlSuppress ControlMode = "suppress"
	ControlPin      ControlMode = "pin"
	ControlBan      ControlMode = "ban"
)

// Control is an explicit operator-authored constraint. Value is in [-1,1];
// pin and ban are hard constraints and are not learned from comparisons.
type Control struct {
	Attribute string
	Mode      ControlMode
	Value     float64
}

// Feature is sourced evidence for one candidate. Confidence describes the
// evidence, not the listener's preference for it.
type Feature struct {
	Attribute  string
	Value      float64
	Confidence float64
	Source     string
}

type Candidate struct {
	ID       string
	Title    string
	Artist   string
	Genre    string
	Features []Feature
}

type Choice string

const (
	ChooseLeft    Choice = "left"
	ChooseRight   Choice = "right"
	ChooseTie     Choice = "tie"
	ChooseNeither Choice = "neither"
	ChooseSkip    Choice = "skip"
)

// Judgment retains identity and provenance so later corrections can replay
// the model from the operator's original evidence.
type Judgment struct {
	ID             string
	LeftID         string
	RightID        string
	Choice         Choice
	Context        string
	Source         string
	At             string
	Reverses       string
	Reason         string
	AttributeScope string
	LeftEvidence   []Feature
	RightEvidence  []Feature
}

type Weight struct {
	Mean     float64
	Variance float64
}

type Profile struct {
	Version      string
	FeatureSpace string
	Context      string
	Revision     int64
	UpdatedAt    time.Time
	Controls     []Control
	Judgments    []Judgment
	Weights      map[string]Weight
}

type Contribution struct {
	Attribute      string
	Value          float64
	Weight         float64
	Contribution   float64
	EvidenceSource string
}
type Score struct {
	CandidateID   string
	Mean          float64
	Uncertainty   float64
	Contributions []Contribution
	Explore       bool
	Eligible      bool
}

func NewProfile() Profile {
	return Profile{Version: ModelVersion, FeatureSpace: FeatureSpaceVersion, Weights: map[string]Weight{}}
}

func (p Profile) Validate() error {
	if p.Version != ModelVersion {
		return ErrInvalid
	}
	for _, c := range p.Controls {
		if strings.TrimSpace(c.Attribute) != c.Attribute || c.Attribute == "" || len(c.Attribute) > 120 || math.IsNaN(c.Value) || math.IsInf(c.Value, 0) || c.Value < -1 || c.Value > 1 {
			return ErrInvalid
		}
		switch c.Mode {
		case ControlBoost, ControlSuppress, ControlPin, ControlBan:
		default:
			return ErrInvalid
		}
	}
	seenControls := map[string]bool{}
	for _, c := range p.Controls {
		if seenControls[c.Attribute] {
			return ErrInvalid
		}
		seenControls[c.Attribute] = true
	}
	ids := map[string]bool{}
	for _, j := range p.Judgments {
		if j.LeftID == "" || j.RightID == "" || j.LeftID == j.RightID {
			return ErrInvalid
		}
		if j.ID != "" {
			if ids[j.ID] {
				return ErrInvalid
			}
			ids[j.ID] = true
		}
		switch j.Choice {
		case ChooseLeft, ChooseRight, ChooseTie, ChooseNeither, ChooseSkip:
		default:
			return ErrInvalid
		}
	}
	return nil
}

// SelectPair chooses the pair whose preference probability has greatest
// entropy, with a small uncertainty bonus. Ties remain valid responses.
func SelectPair(p Profile, candidates []Candidate, seen map[string]bool) (Candidate, Candidate, bool) {
	if len(candidates) < 2 {
		return Candidate{}, Candidate{}, false
	}
	best := -1.0
	var a, b Candidate
	for i := 0; i < len(candidates); i++ {
		for j := i + 1; j < len(candidates); j++ {
			left, right := candidates[i], candidates[j]
			key := pairKey(left.ID, right.ID)
			if seen[key] || left.ID == right.ID || !AreComparable(left, right) {
				continue
			}
			value := PairInformation(p, left, right)
			if value > best {
				best = value
				a = left
				b = right
			}
		}
	}
	return a, b, best >= 0
}

// AreComparable rejects pairs with identical evidence or a candidate that has
// no sourced feature values. Asking about a wholly unannotated candidate
// cannot teach this feature model a directional distinction.
func AreComparable(a, b Candidate) bool {
	if len(a.Features) == 0 || len(b.Features) == 0 {
		return false
	}
	if nearDuplicate(a, b) {
		return false
	}
	left := featureDiff(a, b)
	if len(left) == 0 {
		return false
	}
	for _, value := range left {
		if math.Abs(value) > 1e-9 {
			return true
		}
	}
	return false
}

// PairInformation approximates expected posterior log-volume reduction for a
// logistic comparison over independent Gaussian feature weights. The expected
// Bernoulli information term is p(1-p); low-confidence feature evidence enters
// squared through featureDiff, so unreliable decompositions have less pull.
func PairInformation(p Profile, a, b Candidate) float64 {
	diff := featureDiff(a, b)
	linear, variance := 0.0, 0.0
	for attribute, value := range diff {
		weight, ok := p.Weights[attribute]
		if !ok {
			weight = Weight{Variance: 1}
		}
		linear += weight.Mean * value
		variance += math.Max(0, weight.Variance) * value * value
	}
	probability := sigmoid(linear / math.Sqrt(1+variance))
	responseInformation := probability * (1 - probability)
	if responseInformation <= 0 {
		return 0
	}
	gain := 0.0
	for attribute, value := range diff {
		weight, ok := p.Weights[attribute]
		if !ok {
			weight = Weight{Variance: 1}
		}
		gain += 0.5 * math.Log1p(math.Max(0, weight.Variance)*value*value*responseInformation)
	}
	return gain
}

func nearDuplicate(a, b Candidate) bool {
	left, right := map[string]float64{}, map[string]float64{}
	for _, f := range a.Features {
		if f.Confidence > 0 {
			left[f.Attribute] = f.Value * f.Confidence
		}
	}
	for _, f := range b.Features {
		if f.Confidence > 0 {
			right[f.Attribute] = f.Value * f.Confidence
		}
	}
	if len(left) == 0 || len(right) == 0 {
		return true
	}
	shared, union := 0, len(left)
	for attribute, value := range right {
		if leftValue, ok := left[attribute]; ok && math.Abs(leftValue-value) < 1e-9 {
			shared++
		} else {
			union++
		}
	}
	// Jaccard on the explicitly represented dimensions: very close candidates
	// create low-value repeated exposure even if their scores are uncertain.
	return union > 0 && float64(shared)/float64(union) >= 0.95
}

func pairKey(a, b string) string {
	if a > b {
		a, b = b, a
	}
	return a + "\x00" + b
}

// FitPairwise replays active append-only judgments with a Gaussian prior and a
// diagonal Laplace approximation to logistic posterior uncertainty. Ties
// contribute indifference evidence; neither and skip remain nondirectional.
func FitPairwise(p Profile, candidates []Candidate, steps int) Profile {
	if p.Weights == nil {
		p.Weights = map[string]Weight{}
	}
	byID := map[string]Candidate{}
	reversed := map[string]bool{}
	for _, j := range p.Judgments {
		if j.Reverses != "" {
			reversed[j.Reverses] = true
		}
	}
	priors := make(map[string]Weight)
	for _, c := range candidates {
		byID[c.ID] = c
		for _, f := range c.Features {
			if f.Attribute != "" {
				if _, ok := p.Weights[f.Attribute]; !ok {
					p.Weights[f.Attribute] = Weight{Variance: 1}
				}
				if _, ok := priors[f.Attribute]; !ok {
					prior := p.Weights[f.Attribute]
					if prior.Variance <= 0 || math.IsNaN(prior.Variance) || math.IsInf(prior.Variance, 0) {
						prior.Variance = 1
					}
					priors[f.Attribute] = prior
				}
			}
		}
	}
	for attribute, weight := range p.Weights {
		if math.IsNaN(weight.Mean) || math.IsInf(weight.Mean, 0) {
			weight.Mean = 0
		}
		if weight.Mean < -2 || weight.Mean > 2 {
			weight.Mean = math.Max(-2, math.Min(2, weight.Mean))
		}
		if weight.Variance <= 0 || math.IsNaN(weight.Variance) || math.IsInf(weight.Variance, 0) {
			weight.Variance = 1
		}
		p.Weights[attribute] = weight
		if _, ok := priors[attribute]; !ok {
			priors[attribute] = weight
		}
	}
	type trainingPair struct {
		diff   map[string]float64
		target float64
	}
	pairs := make([]trainingPair, 0, len(p.Judgments))
	for _, j := range p.Judgments {
		if j.Choice == ChooseSkip || j.Choice == ChooseNeither || (j.ID != "" && reversed[j.ID]) {
			continue
		}
		left, lok := byID[j.LeftID]
		right, rok := byID[j.RightID]
		if !lok && len(j.LeftEvidence) > 0 {
			left, lok = Candidate{ID: j.LeftID, Features: j.LeftEvidence}, true
		}
		if !rok && len(j.RightEvidence) > 0 {
			right, rok = Candidate{ID: j.RightID, Features: j.RightEvidence}, true
		}
		if !lok || !rok {
			continue
		}
		diff := featureDiff(left, right)
		if j.AttributeScope != "" && j.AttributeScope != "track" {
			for attribute := range diff {
				if attribute != j.AttributeScope {
					delete(diff, attribute)
				}
			}
		}
		target := 0.5
		switch j.Choice {
		case ChooseLeft:
			target = 1
		case ChooseRight:
			target = 0
		case ChooseTie:
		default:
			continue
		}
		if len(diff) > 0 {
			pairs = append(pairs, trainingPair{diff: diff, target: target})
			for attribute := range diff {
				if _, ok := p.Weights[attribute]; !ok {
					p.Weights[attribute] = Weight{Variance: 1}
					priors[attribute] = p.Weights[attribute]
				}
				if _, ok := priors[attribute]; !ok {
					priors[attribute] = Weight{Variance: 1}
				}
			}
		}
	}
	if steps < 1 {
		steps = 1
	}
	if steps > 500 {
		steps = 500
	}
	for epoch := 0; epoch < steps && len(pairs) > 0; epoch++ {
		gradients := make(map[string]float64, len(p.Weights))
		for attribute, weight := range p.Weights {
			prior := priors[attribute]
			gradients[attribute] = -(weight.Mean - prior.Mean) / prior.Variance
		}
		for _, pair := range pairs {
			linear := 0.0
			for attribute, value := range pair.diff {
				linear += p.Weights[attribute].Mean * value
			}
			errorSignal := pair.target - sigmoid(linear)
			for attribute, value := range pair.diff {
				gradients[attribute] += errorSignal * value
			}
		}
		rate := 0.12 / (math.Sqrt(float64(epoch+1)) * float64(len(pairs)))
		for attribute, gradient := range gradients {
			weight := p.Weights[attribute]
			weight.Mean = math.Max(-2, math.Min(2, weight.Mean+rate*gradient))
			p.Weights[attribute] = weight
		}
	}
	curvature := make(map[string]float64, len(p.Weights))
	for _, pair := range pairs {
		linear := 0.0
		for attribute, value := range pair.diff {
			linear += p.Weights[attribute].Mean * value
		}
		probability := sigmoid(linear)
		for attribute, value := range pair.diff {
			curvature[attribute] += probability * (1 - probability) * value * value
		}
	}
	for attribute, weight := range p.Weights {
		prior := priors[attribute]
		posteriorVariance := 1 / (1/prior.Variance + curvature[attribute])
		weight.Variance = math.Max(0.05, math.Min(prior.Variance, posteriorVariance))
		p.Weights[attribute] = weight
	}
	return p
}

func featureDiff(a, b Candidate) map[string]float64 {
	out := map[string]float64{}
	for _, f := range a.Features {
		if f.Confidence > 0 {
			out[f.Attribute] += f.Value * f.Confidence
		}
	}
	for _, f := range b.Features {
		if f.Confidence > 0 {
			out[f.Attribute] -= f.Value * f.Confidence
		}
	}
	return out
}

// ScoreCandidate computes a reproducible, evidence-backed score. Unknown
// features contribute neither value nor a fabricated explanation.
func ScoreCandidate(p Profile, c Candidate) Score {
	result := Score{CandidateID: c.ID, Eligible: true}
	if len(c.Features) == 0 {
		result.Eligible = false
	}
	controls := map[string]Control{}
	for _, v := range p.Controls {
		controls[v.Attribute] = v
		if v.Mode == ControlBan {
			for _, f := range c.Features {
				if f.Attribute == v.Attribute && f.Value != 0 {
					result.Eligible = false
				}
			}
		}
	}
	for _, f := range c.Features {
		if f.Attribute == "" || f.Source == "" || f.Confidence <= 0 || f.Confidence > 1 || math.IsNaN(f.Value) || math.IsInf(f.Value, 0) || f.Value < -1 || f.Value > 1 || math.IsNaN(f.Confidence) || math.IsInf(f.Confidence, 0) {
			continue
		}
		w, ok := p.Weights[f.Attribute]
		if !ok {
			w = Weight{Variance: 1}
		}
		control := controls[f.Attribute]
		switch control.Mode {
		case ControlBoost:
			w.Mean += control.Value
		case ControlSuppress:
			w.Mean -= math.Abs(control.Value)
		case ControlPin:
			w.Mean = control.Value
		}
		part := f.Value * f.Confidence * w.Mean
		result.Mean += part
		result.Uncertainty += f.Confidence * f.Confidence * math.Max(0, w.Variance)
		result.Contributions = append(result.Contributions, Contribution{Attribute: f.Attribute, Value: f.Value, Weight: w.Mean, Contribution: part, EvidenceSource: f.Source})
	}
	if len(result.Contributions) == 0 {
		result.Eligible = false
	}
	result.Uncertainty = math.Sqrt(result.Uncertainty)
	result.Explore = result.Uncertainty > 0.5
	sort.Slice(result.Contributions, func(i, j int) bool { return result.Contributions[i].Attribute < result.Contributions[j].Attribute })
	return result
}

func sigmoid(x float64) float64 {
	if x >= 0 {
		z := math.Exp(-x)
		return 1 / (1 + z)
	}
	z := math.Exp(x)
	return z / (1 + z)
}
