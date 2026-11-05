package interventions

import (
	"math"
	"sort"
)

// MinSignalCell is the fewest classified messages a cell or its baseline needs
// before a rate is shown: below it the range spans most of 0..1 and a
// comparison would be noise.
const MinSignalCell = 20

// SignalAccuracy is a label's measured hold-out accuracy for one model.
type SignalAccuracy struct {
	TP, Pred, Gold int
}

func (a SignalAccuracy) Precision() *float64 { return ratio(a.TP, a.Pred) }
func (a SignalAccuracy) Recall() *float64    { return ratio(a.TP, a.Gold) }

func ratio(k, n int) *float64 {
	if n == 0 {
		return nil
	}
	v := float64(k) / float64(n)
	return &v
}

// Comparison states against the operator's own prior period.
const (
	ComparisonHigher     = "higher"
	ComparisonLower      = "lower"
	ComparisonSame       = "no_clear_difference"
	ComparisonNotEnough  = "not_enough_data"
	StateProven          = "proven"
	StateUnproven        = "unproven"
	NoMode               = "no_mode"
	UnknownModel         = "unknown"
	modeApplied          = "applied"
	modeRanDuringOnTaget = "ran_during_on_target"
)

// SignalRate is one label in one cell.
type SignalRate struct {
	Signal Signal
	// State is proven or unproven. An unproven label reports no count or rate:
	// only its measured hold-out precision and recall.
	State string
	// Count is the messages the gate admits as the label; Rate its share of the
	// cell's classified messages with a 95% Wilson range.
	Count             int64
	Rate, Low, High   *float64
	Baseline          *float64 // the operator's rate in the prior period, same model, outside any mode
	BaselineLow       *float64
	BaselineHigh      *float64
	BaselineN         int64
	Difference        *float64 // Rate minus Baseline, 95% Newcombe range
	DifferenceLow     *float64
	DifferenceHigh    *float64
	Comparison        string
	Precision, Recall *float64 // measured on the hold-out, whatever the state
}

// SignalCell is one (mode period, model) group of a window.
type SignalCell struct {
	ModeID     string // NoMode when the run was not under a mode period
	Model      string // the model that did the work; UnknownModel when no run fact says
	Messages   int64  // operator messages after agent work
	Classified int64
	Coverage   *float64
	Rates      []SignalRate
}

// SummarizeSignals groups a window's observations by mode period and model and
// compares each cell with baseline, the operator's prior period of the same
// length. Only labels the gate proves for the answering model are counted.
// measured is the hold-out accuracy per classifier model, shown beside every
// label. The result is a single operator's own data: no per-person split.
func SummarizeSignals(current, baseline []SignalObservation, gate SignalGate, measured map[string]map[Signal]SignalAccuracy) []SignalCell {
	type key struct{ mode, model string }
	groups := map[key][]SignalObservation{}
	for _, o := range current {
		groups[key{cellMode(o), cellModel(o)}] = append(groups[key{cellMode(o), cellModel(o)}], o)
	}
	base := map[string][]SignalObservation{}
	for _, o := range baseline {
		if cellMode(o) == NoMode {
			base[cellModel(o)] = append(base[cellModel(o)], o)
		}
	}
	keys := make([]key, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].mode != keys[j].mode {
			return keys[i].mode < keys[j].mode
		}
		return keys[i].model < keys[j].model
	})
	labels := append(Signals(), ExplicitNegative)
	var cells []SignalCell
	for _, k := range keys {
		observations := groups[k]
		cell := SignalCell{ModeID: k.mode, Model: k.model, Messages: int64(len(observations))}
		for _, o := range observations {
			if o.Status == StatusClassified {
				cell.Classified++
			}
		}
		if cell.Messages > 0 {
			v := float64(cell.Classified) / float64(cell.Messages)
			cell.Coverage = &v
		}
		baseCount := int64(0)
		for _, o := range base[k.model] {
			if o.Status == StatusClassified {
				baseCount++
			}
		}
		for _, label := range labels {
			rate := SignalRate{Signal: label, State: StateUnproven, Comparison: ComparisonNotEnough}
			rate.Precision, rate.Recall = bestAccuracy(measured, label)
			if gate.Proven(label) {
				rate.State = StateProven
				rate.Count = countShown(observations, gate, label)
				if cell.Classified >= MinSignalCell {
					rate.Rate, rate.Low, rate.High = proportion(rate.Count, cell.Classified)
				}
				rate.BaselineN = baseCount
				if baseCount >= MinSignalCell {
					baseShown := countShown(base[k.model], gate, label)
					rate.Baseline, rate.BaselineLow, rate.BaselineHigh = proportion(baseShown, baseCount)
					if rate.Rate != nil {
						compare(&rate, rate.Count, cell.Classified, baseShown, baseCount)
					}
				}
			}
			cell.Rates = append(cell.Rates, rate)
		}
		cells = append(cells, cell)
	}
	return cells
}

func cellMode(o SignalObservation) string {
	if o.ModeID != "" && (o.ModeRelation == modeApplied || o.ModeRelation == modeRanDuringOnTaget) {
		return o.ModeID
	}
	return NoMode
}

func cellModel(o SignalObservation) string {
	if o.ActualModel == "" {
		return UnknownModel
	}
	return o.ActualModel
}

func countShown(observations []SignalObservation, gate SignalGate, label Signal) int64 {
	var n int64
	for _, o := range observations {
		if o.Status != StatusClassified {
			continue
		}
		var p *float64
		switch {
		case label == ExplicitNegative:
			p = distributionMass(o.Distribution, ReportedFailure, Correction)
		case len(o.Distribution) > 0:
			v := o.Distribution[label]
			p = &v
		case o.Signal != label:
			continue // no distribution: only the answer's own label can count
		}
		if gate.Shown(o.SignalModel, label, p) {
			n++
		}
	}
	return n
}

func distributionMass(distribution map[Signal]float64, labels ...Signal) *float64 {
	if len(distribution) == 0 {
		return nil
	}
	mass := 0.0
	for _, l := range labels {
		mass += distribution[l]
	}
	return &mass
}

// bestAccuracy returns the precision and recall of the model with the most
// hold-out gold rows for the label.
func bestAccuracy(measured map[string]map[Signal]SignalAccuracy, label Signal) (precision, recall *float64) {
	best, found := SignalAccuracy{}, false
	for _, byLabel := range measured {
		if a, ok := byLabel[label]; ok && (!found || a.Gold > best.Gold) {
			best, found = a, true
		}
	}
	if !found {
		return nil, nil
	}
	return best.Precision(), best.Recall()
}

func proportion(k, n int64) (rate, low, high *float64) {
	if n <= 0 {
		return nil, nil, nil
	}
	p := float64(k) / float64(n)
	lo, hi := proportionInterval(k, n)
	return &p, &lo, &hi
}

// proportionInterval is the 95% Wilson score interval.
func proportionInterval(k, n int64) (float64, float64) {
	const z = 1.96
	p, nf := float64(k)/float64(n), float64(n)
	d := 1 + z*z/nf
	c := (p + z*z/(2*nf)) / d
	h := z * math.Sqrt(p*(1-p)/nf+z*z/(4*nf*nf)) / d
	return math.Max(0, c-h), math.Min(1, c+h)
}

// compare fills the difference of two proportions with Newcombe's hybrid score
// interval and names it: a range that excludes zero is a difference.
func compare(rate *SignalRate, k1, n1, k2, n2 int64) {
	p1, p2 := float64(k1)/float64(n1), float64(k2)/float64(n2)
	l1, u1 := proportionInterval(k1, n1)
	l2, u2 := proportionInterval(k2, n2)
	diff := p1 - p2
	low := diff - math.Sqrt((p1-l1)*(p1-l1)+(u2-p2)*(u2-p2))
	high := diff + math.Sqrt((u1-p1)*(u1-p1)+(p2-l2)*(p2-l2))
	rate.Difference, rate.DifferenceLow, rate.DifferenceHigh = &diff, &low, &high
	switch {
	case low > 0:
		rate.Comparison = ComparisonHigher
	case high < 0:
		rate.Comparison = ComparisonLower
	default:
		rate.Comparison = ComparisonSame
	}
}

// SignalMeasurements is the hold-out accuracy per answering model and label.
type SignalMeasurements map[string]map[Signal]SignalAccuracy

// Precision is the model's measured hold-out precision for the label, nil when
// it was never measured or predicted the label zero times.
func (m SignalMeasurements) Precision(model string, label Signal) *float64 {
	return m[model][label].Precision()
}
