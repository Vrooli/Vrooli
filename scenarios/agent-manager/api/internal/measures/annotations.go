package measures

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"agent-manager/internal/runannotations"

	measurelib "github.com/vrooli/measures-go"
	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
)

// AnnotationCoverage reports per-label annotation coverage and the gate state
// that decides whether a label may produce a rate.
const AnnotationCoverage = "usage.annotation_coverage"

// AnnotationRate reports the rate of each proven annotation label over the
// window. An unproven label is absent, never zero.
const AnnotationRate = "usage.annotation_rate"

// SetAnnotations supplies the data-driven annotation service.
func (h *Handler) SetAnnotations(service *runannotations.Service) { h.annotations = service }

var annotationCoverageDefinition = Definition{
	ID: AnnotationCoverage,
	Counts: "runs with at least one annotation observation, per label, over the window; " +
		"the gate state says whether the label may be reported as a rate",
	Numerator:   "runs where the label was observed",
	Denominator: "runs in the eligible annotated population (or the configured candidate census)",
	SourceTable: "run_annotations",
	Limitation: "A label whose gate is unproven, insufficient_gold or unmeasured is evidence, not a rate. " +
		"An empty window is unknown coverage, never zero friction. Rates over a sampled population are " +
		"reported with the weighting the pack declares.",
}

// registerAnnotationMeasures declares the annotation coverage and rate
// measures. They are registry-only: no typed RPC exists.
func (h *Handler) registerAnnotationMeasures(registry *measurelib.Registry) error {
	params := map[string]measurelib.Param{"window": {Name: "window", Type: measurelib.ParamTypeTimeWindow, Default: string(measurelib.TokenThisWeek)}}
	if err := registry.Register(measurelib.MeasureDeclaration{
		Name: AnnotationCoverage, Scenario: "agent-manager", Domain: "usage",
		Intent:    "Per-label annotation coverage and gate state over a window: what was observed, what was eligible, and whether the label is proven.",
		Questions: []string{"which inferred labels are proven", "how many runs are annotated", "what is the annotation coverage"},
		Params:    params,
		Result:    measurelib.Result{Kind: measurelib.ResultTable, ValueField: "label", Unit: "share", SummaryTemplate: "annotation coverage ({window})"},
		Effect:    measurelib.EffectRead, RunEligible: true,
	}, func(ctx context.Context, request measurelib.MeasureRequest) (measurelib.MeasureResult, error) {
		report, err := h.annotationCoverage(ctx, request)
		if err != nil {
			return measurelib.MeasureResult{}, err
		}
		fields := make([]map[string]string, 0, len(report.Labels))
		for _, label := range report.Labels {
			fields = append(fields, annotationCoverageRow(label))
		}
		return measurelib.MeasureResult{Fields: fields, Provenance: measurelib.Provenance{ExecutedQuery: fmt.Sprintf(
			"SELECT label_id, state, observed_at FROM run_annotations WHERE observed_at >= %q AND observed_at < %q; gate data comes from the loaded pack",
			report.Window.Since, report.Window.Until)}}, nil
	}); err != nil {
		return err
	}
	return registry.Register(measurelib.MeasureDeclaration{
		Name: AnnotationRate, Scenario: "agent-manager", Domain: "usage",
		Intent:    "The rate of each proven annotation label over the window; unproven labels are omitted, never reported as zero.",
		Questions: []string{"how often is the agent spinning", "how often is the operator frustrated", "which inferred friction labels are proven"},
		Params:    params,
		Result:    measurelib.Result{Kind: measurelib.ResultTable, ValueField: "label", Unit: "share", SummaryTemplate: "annotation rate ({window})"},
		Effect:    measurelib.EffectRead, RunEligible: true,
	}, func(ctx context.Context, request measurelib.MeasureRequest) (measurelib.MeasureResult, error) {
		report, err := h.annotationCoverage(ctx, request)
		if err != nil {
			return measurelib.MeasureResult{}, err
		}
		fields := make([]map[string]string, 0, len(report.Labels))
		for _, label := range report.Labels {
			if label.Rate == nil {
				continue
			}
			row := annotationCoverageRow(label)
			row["rate"] = formatAnnotationFloat(*label.Rate)
			fields = append(fields, row)
		}
		if len(fields) == 0 {
			fields = append(fields, map[string]string{
				"label": "none", "state": "unproven",
				"note": "no annotation label passed its gate; this is not evidence of no friction",
			})
		}
		return measurelib.MeasureResult{Fields: fields, Provenance: measurelib.Provenance{ExecutedQuery: fmt.Sprintf(
			"SELECT label_id, state, observed_at FROM run_annotations WHERE observed_at >= %q AND observed_at < %q; only labels with a proven gate expose a rate",
			report.Window.Since, report.Window.Until)}}, nil
	})
}

func (h *Handler) annotationCoverage(ctx context.Context, request measurelib.MeasureRequest) (runannotations.CoverageResponse, error) {
	if h.annotations == nil {
		return runannotations.CoverageResponse{}, fmt.Errorf("annotation service is unavailable")
	}
	window, err := h.resolveWindow(request, measurelib.TokenThisWeek)
	if err != nil {
		return runannotations.CoverageResponse{}, err
	}
	return h.annotations.Coverage(ctx, window.From.UTC().Format(time.RFC3339))
}

// protoInferredAnnotations maps the annotation coverage contract onto the
// friction response. A rate survives only for a proven gate; unproven labels
// keep their measured counts and gate targets so a caller can show evidence
// without inventing a rate.
func protoInferredAnnotations(report runannotations.CoverageResponse) *measurepb.UsageInferredAnnotations {
	section := &measurepb.UsageInferredAnnotations{
		PackName: report.Pack.Name, PackVersion: report.Pack.Version,
		EligibleRuns: int64(report.EligibleRuns), ObservedRuns: int64(report.ObservedRuns),
		Coverage: report.Coverage,
	}
	for _, label := range report.Labels {
		row := &measurepb.UsageInferredAnnotation{
			Id: label.ID, Group: label.Group, State: label.State,
			Observed: int64(label.Observed), Eligible: int64(label.Eligible),
			GateState: label.Gate.State, Gold: int64(label.Gate.Gold),
			Precision: label.Gate.Precision, Recall: label.Gate.Recall,
			TargetPrecision: label.Gate.TargetPrecision, TargetRecall: label.Gate.TargetRecall,
			Friction: label.Friction.Enabled, SearchFacet: label.SearchFacet,
		}
		if label.Eligible > 0 {
			row.Coverage = float64(label.Observed) / float64(label.Eligible)
		}
		if label.Rate != nil {
			rate := *label.Rate
			row.Rate = &rate
		}
		if label.MinProbability != nil {
			row.MinProbability = *label.MinProbability
		}
		section.Labels = append(section.Labels, row)
	}
	for _, note := range report.Notes {
		if section.Note == "" {
			section.Note = note
			break
		}
	}
	if report.Coverage == 0 && report.EligibleRuns == 0 && section.Note == "" {
		section.Note = "no run was eligible in this window: coverage is unknown, not zero friction"
	}
	return section
}

func annotationCoverageRow(label runannotations.CoverageLabel) map[string]string {
	row := map[string]string{
		"label": label.ID, "group": label.Group, "state": label.State,
		"observed": strconv.Itoa(label.Observed), "eligible": strconv.Itoa(label.Eligible), "positives": strconv.Itoa(label.Positives),
		"gate_state": label.Gate.State, "gold": strconv.Itoa(label.Gate.Gold), "min_gold": strconv.Itoa(label.Gate.MinGold),
		"target_precision": formatAnnotationFloat(label.Gate.TargetPrecision),
		"target_recall":    formatAnnotationFloat(label.Gate.TargetRecall),
		"precision":        formatAnnotationFloat(label.Gate.Precision),
		"recall":           formatAnnotationFloat(label.Gate.Recall),
	}
	if label.Rate != nil {
		row["rate"] = formatAnnotationFloat(*label.Rate)
	}
	if label.Gate.Reason != "" {
		row["gate_reason"] = label.Gate.Reason
	}
	return row
}

func formatAnnotationFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
