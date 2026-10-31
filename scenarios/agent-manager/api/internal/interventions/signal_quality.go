package interventions

import (
	"context"
	"time"

	"agent-manager/internal/usagemodes"

	"github.com/vrooli/cli-core/usagemode"
)

// ModeQuality finds a proven rise in the operator's own failure and correction
// reports (the explicit_negative bucket) under a usage mode. It implements
// usagemodes.QualitySource and reports nothing unless the bucket passed the
// accuracy gate for the answering model and the switch is on.
type ModeQuality struct {
	Store    *Store
	Gate     SignalGate
	Measured SignalMeasurements
}

// QualityDrop compares the mode's period with the operator's preceding period
// of equal length, outside any mode, on the same model, and returns the model
// with the clearest rise. Cells and baselines under MinSignalCell classified
// messages never qualify.
func (q ModeQuality) QualityDrop(ctx context.Context, mode usagemode.Mode, now time.Time) (usagemodes.QualityFinding, bool, error) {
	if q.Store == nil || !SignalsEnabled() || !q.Gate.Proven(ExplicitNegative) {
		return usagemodes.QualityFinding{}, false, nil
	}
	length := now.Sub(mode.StartedAt)
	current, err := q.Store.SignalObservations(ctx, SignalVersion, mode.StartedAt, now)
	if err != nil {
		return usagemodes.QualityFinding{}, false, err
	}
	prior, err := q.Store.SignalObservations(ctx, SignalVersion, mode.StartedAt.Add(-length), mode.StartedAt)
	if err != nil {
		return usagemodes.QualityFinding{}, false, err
	}
	return qualityFinding(SummarizeSignals(current, prior, q.Gate, q.Measured), mode.ID)
}

func qualityFinding(cells []SignalCell, modeID string) (usagemodes.QualityFinding, bool, error) {
	var best usagemodes.QualityFinding
	found := false
	for _, cell := range cells {
		if cell.ModeID != modeID {
			continue
		}
		for _, rate := range cell.Rates {
			if rate.Signal != ExplicitNegative || rate.State != StateProven || rate.Comparison != ComparisonHigher || rate.Rate == nil || rate.Baseline == nil || rate.DifferenceLow == nil || rate.DifferenceHigh == nil {
				continue
			}
			if found && *rate.DifferenceLow <= best.DifferenceLow {
				continue
			}
			best = usagemodes.QualityFinding{Rate: *rate.Rate, Baseline: *rate.Baseline, DifferenceLow: *rate.DifferenceLow, DifferenceHigh: *rate.DifferenceHigh,
				Messages: cell.Classified, BaselineMessages: rate.BaselineN, Model: cell.Model}
			if rate.Precision != nil {
				best.ClassifierPrecision = *rate.Precision
			}
			found = true
		}
	}
	return best, found, nil
}
