// Package annotationgate projects the annotation coverage contract into the
// rows every consumer of a label's gate reads: the usage.annotation_gate measure
// (CLI and agents), the Usage page's REST read and, through the same fields, the
// friction digest. The engine owns every number; this package only flattens it,
// so a rate is never quoted without the gate and the model that license it.
package annotationgate

import (
	"strconv"
	"time"

	"agent-manager/internal/runannotations"
)

// Row kinds. A consumer groups label and model rows by their `label` cell.
const (
	gateRowCoverage = "coverage"
	gateRowLabel    = "label"
	gateRowModel    = "model"
)

func formatFloat(value float64) string { return strconv.FormatFloat(value, 'f', -1, 64) }

// Rows is the whole projection: one coverage row, then a label
// row for every label with a model row for each (model, resource role) it was
// observed with. A rate is present only when ProvenObserved > 0, and counts only
// those observations.
func Rows(report runannotations.CoverageResponse) []map[string]string {
	rows := []map[string]string{coverageRow(report)}
	for _, label := range report.Labels {
		rows = append(rows, labelGateRow(label))
		for _, model := range label.Models {
			rows = append(rows, modelGateRow(label.ID, model))
		}
	}
	return rows
}

func coverageRow(report runannotations.CoverageResponse) map[string]string {
	row := map[string]string{
		"kind": gateRowCoverage, "pack": report.Pack.Name, "pack_version": report.Pack.Version,
		"window_since": report.Window.Since, "window_until": report.Window.Until,
		"eligible_runs": strconv.Itoa(report.EligibleRuns), "observed_runs": strconv.Itoa(report.ObservedRuns),
		"coverage": formatFloat(report.Coverage),
	}
	if provenance := report.Provenance; provenance != nil {
		row["human_text_runs"] = strconv.Itoa(provenance.RunsHumanText)
		row["brief_only_runs"] = strconv.Itoa(provenance.RunsBriefOnly)
		row["no_operator_text_runs"] = strconv.Itoa(provenance.RunsNoOperatorText)
		row["human_messages"] = strconv.Itoa(provenance.HumanMessages)
		if provenance.RunsStale > 0 {
			row["stale_runs"] = strconv.Itoa(provenance.RunsStale)
		}
	}
	if len(report.Notes) > 0 {
		row["note"] = report.Notes[0]
	}
	return row
}

func labelGateRow(label runannotations.CoverageLabel) map[string]string {
	row := map[string]string{
		"kind": gateRowLabel, "label": label.ID, "group": label.Group, "scope": string(label.Scope),
		"state": label.State, "friction": strconv.FormatBool(label.Friction.Enabled),
		"observed": strconv.Itoa(label.Observed), "eligible": strconv.Itoa(label.Eligible),
		"positives": strconv.Itoa(label.Positives), "proven_observed": strconv.Itoa(label.ProvenObserved),
		"proven_positives": strconv.Itoa(label.ProvenPositives),
	}
	if label.Population != nil {
		row["population"] = label.Population.Basis
		row["population_unit"] = label.Population.Unit
	}
	if label.Rate != nil {
		row["rate"] = formatFloat(*label.Rate)
	}
	if label.SearchFacet != "" {
		row["search_facet"] = label.SearchFacet
	}
	for key, value := range gateCells(label.Gate) {
		row[key] = value
	}
	return row
}

func modelGateRow(labelID string, model runannotations.ModelCoverage) map[string]string {
	row := map[string]string{
		"kind": gateRowModel, "label": labelID, "model": model.Model, "resource_role": model.ResourceRole,
		"observed": strconv.Itoa(model.Observed), "positives": strconv.Itoa(model.Positives),
		"proven": strconv.FormatBool(model.Proven),
	}
	for key, value := range gateCells(model.Gate) {
		row[key] = value
	}
	return row
}

// gateCells flattens one gate status. The measurement cells, including the model
// the measurement is for, are present only when a measurement exists, so an empty
// cell never reads as zero and an unmeasured label never names a model it was not
// measured for.
func gateCells(gate runannotations.GateStatus) map[string]string {
	cells := map[string]string{
		"gate_state": gate.State, "target_precision": formatFloat(gate.TargetPrecision),
		"target_recall": formatFloat(gate.TargetRecall), "min_gold": strconv.Itoa(gate.MinGold),
	}
	if gate.Reason != "" {
		cells["gate_reason"] = gate.Reason
	}
	if gate.Split == "" {
		// Nothing was measured: the status names a model only as the one asked about.
		return cells
	}
	cells["gate_model"], cells["gate_role"] = gate.Model, gate.ResourceRole
	cells["split"] = gate.Split
	cells["precision"], cells["recall"] = formatFloat(gate.Precision), formatFloat(gate.Recall)
	cells["gold"], cells["gold_positives"] = strconv.Itoa(gate.Gold), strconv.Itoa(gate.Positives)
	cells["ece"], cells["brier"] = formatFloat(gate.ECE), formatFloat(gate.Brier)
	cells["threshold"], cells["temperature"] = formatFloat(gate.Threshold), formatFloat(gate.Temperature)
	cells["calibration"] = Calibration(gate)
	if gate.MeasuredAt != "" {
		if at, err := time.Parse(time.RFC3339, gate.MeasuredAt); err == nil {
			cells["measured_at"] = at.UTC().Format(time.RFC3339)
		}
	}
	return cells
}

// Calibration says how a measurement's probabilities are read: a rule has
// none, a fitted temperature other than 1 is calibrated, anything else is raw.
// A gate with no measurement has no calibration to name.
func Calibration(gate runannotations.GateStatus) string {
	switch {
	case gate.Split == "":
		return ""
	case gate.Model == runannotations.RuleModel:
		return "rule"
	case gate.Temperature > 0 && gate.Temperature != 1:
		return "calibrated"
	default:
		return "raw"
	}
}
