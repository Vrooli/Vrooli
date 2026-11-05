package measures

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"agent-manager/internal/interventions"

	measurelib "github.com/vrooli/measures-go"
)

// UsageOperatorSignals is the rate of operator messages that report a failure
// or a correction (and, when proven, frustration and satisfaction), per usage
// mode period and model, against the operator's own prior period.
const UsageOperatorSignals = "usage.operator_signals"

// SignalSource reads the operator messages of a window with their signals.
type SignalSource interface {
	SignalObservations(ctx context.Context, version string, from, to time.Time) ([]interventions.SignalObservation, error)
}

// SetSignals supplies the operator-signal classifications.
func (h *Handler) SetSignals(source SignalSource) { h.signals = source }

var operatorSignalsDefinition = Definition{
	ID: UsageOperatorSignals, Counts: "verified human operator messages that followed agent work, labelled by the local decision model with the agent's preceding text",
	Numerator:   "messages the proven gate admits as the label",
	Denominator: "classified operator messages in the same mode period and model",
	SourceTable: "operator_signals, operator_interventions, run_ledger_facts",
	Limitation:  "a label is counted only once its model met precision 0.8 and recall 0.7 on the labelled hold-out; an unproven label reports its measured precision and recall and no rate. Rates are the classifier's counts, not corrected for its errors. The baseline is this operator's own preceding period of equal length, outside any usage mode, same model. A cell under 20 classified messages reports no rate. Single operator, aggregates only",
}

// registerOperatorSignals declares the measure. It is registry-only: no typed
// RPC exists, so Service and Method are empty.
func (h *Handler) registerOperatorSignals(registry *measurelib.Registry) error {
	return registry.Register(measurelib.MeasureDeclaration{
		Name: UsageOperatorSignals, Scenario: "agent-manager", Domain: "usage",
		Intent:    "How often operators report a failure or correct the agent, per usage mode period and model, against their own previous period, with classification coverage and the classifier's measured accuracy",
		Questions: []string{"did quality drop under the mode", "how often do I report failures", "operator corrections per message by model", "is the operator signal proven"},
		Params:    map[string]measurelib.Param{"window": {Name: "window", Type: measurelib.ParamTypeTimeWindow, Default: string(measurelib.TokenThisWeek)}},
		Result:    measurelib.Result{Kind: measurelib.ResultTable, ValueField: "rate", Unit: "share of classified operator messages", SummaryTemplate: "operator signals ({window})"},
		Effect:    measurelib.EffectRead, RunEligible: true,
	}, func(ctx context.Context, request measurelib.MeasureRequest) (measurelib.MeasureResult, error) {
		if !interventions.SignalsEnabled() {
			return measurelib.MeasureResult{Fields: []map[string]string{{"signal": "all", "state": "off", "note": "operator signals are switched off (" + interventions.SignalsSwitchEnv + "); nothing is classified or reported"}},
				Provenance: measurelib.Provenance{ExecutedQuery: "none: switched off"}}, nil
		}
		if h.signals == nil {
			return measurelib.MeasureResult{}, fmt.Errorf("operator signal classifications are unavailable")
		}
		window, err := h.resolveWindow(request, measurelib.TokenThisWeek)
		if err != nil {
			return measurelib.MeasureResult{}, err
		}
		length := window.To.Sub(window.From)
		current, err := h.signals.SignalObservations(ctx, interventions.SignalVersion, window.From, window.To)
		if err != nil {
			return measurelib.MeasureResult{}, err
		}
		prior, err := h.signals.SignalObservations(ctx, interventions.SignalVersion, window.From.Add(-length), window.From)
		if err != nil {
			return measurelib.MeasureResult{}, err
		}
		cells := interventions.SummarizeSignals(current, prior, interventions.ProvenSignalGate, interventions.SignalHoldout)
		return measurelib.MeasureResult{Fields: operatorSignalRows(cells), Provenance: measurelib.Provenance{ExecutedQuery: fmt.Sprintf(
			"SELECT signals FROM operator_signals JOIN operator_interventions JOIN run_ledger_facts WHERE signal_version = %q AND occurred_at >= %q AND occurred_at < %q; baseline window %q to %q",
			interventions.SignalVersion, window.From.UTC().Format(time.RFC3339Nano), window.To.UTC().Format(time.RFC3339Nano),
			window.From.Add(-length).UTC().Format(time.RFC3339Nano), window.From.UTC().Format(time.RFC3339Nano))}}, nil
	})
}

// operatorSignalRows is one row per mode period, model and label (the pooled
// explicit_negative bucket included). Empty cells mean unknown, never zero.
func operatorSignalRows(cells []interventions.SignalCell) []map[string]string {
	num := func(v *float64) string {
		if v == nil {
			return ""
		}
		return strconv.FormatFloat(*v, 'f', 4, 64)
	}
	var rows []map[string]string
	for _, cell := range cells {
		for _, rate := range cell.Rates {
			row := map[string]string{
				"mode": cell.ModeID, "model": cell.Model, "signal": string(rate.Signal), "state": rate.State,
				"messages": strconv.FormatInt(cell.Messages, 10), "classified": strconv.FormatInt(cell.Classified, 10), "coverage": num(cell.Coverage),
				"count": "", "rate": "", "low": "", "high": "",
				"baseline_rate": "", "baseline_low": "", "baseline_high": "", "baseline_classified": "",
				"difference": "", "difference_low": "", "difference_high": "", "comparison": rate.Comparison,
				"measured_precision": num(rate.Precision), "measured_recall": num(rate.Recall), "classifier_version": interventions.SignalVersion,
			}
			if rate.State == interventions.StateProven {
				row["count"], row["rate"], row["low"], row["high"] = strconv.FormatInt(rate.Count, 10), num(rate.Rate), num(rate.Low), num(rate.High)
				row["baseline_rate"], row["baseline_low"], row["baseline_high"] = num(rate.Baseline), num(rate.BaselineLow), num(rate.BaselineHigh)
				row["baseline_classified"] = strconv.FormatInt(rate.BaselineN, 10)
				row["difference"], row["difference_low"], row["difference_high"] = num(rate.Difference), num(rate.DifferenceLow), num(rate.DifferenceHigh)
			} else {
				row["comparison"] = ""
				row["note"] = "unproven: the classifier did not meet the accuracy gate for this label; no rate is reported"
			}
			rows = append(rows, row)
		}
	}
	if len(rows) == 0 {
		rows = append(rows, map[string]string{"signal": "all", "state": "no_data", "note": "no operator messages after agent work in the window"})
	}
	return rows
}
