package measures

import (
	"context"
	"fmt"

	"agent-manager/internal/annotationgate"
	"agent-manager/internal/runannotations"

	measurelib "github.com/vrooli/measures-go"
)

// AnnotationGate reports, per annotation label, whether it is proven, for which
// model, and what that proof measured. The rows come from annotationgate, the one
// projection the Usage page's REST read shares, so a rate is never quoted without
// the gate that licenses it.
const AnnotationGate = "usage.annotation_gate"

// SetAnnotations supplies the data-driven annotation service.
func (h *Handler) SetAnnotations(service *runannotations.Service) { h.annotations = service }

// registerAnnotationMeasures declares the registry-only annotation gate measure.
func (h *Handler) registerAnnotationMeasures(registry *measurelib.Registry) error {
	return registry.Register(measurelib.MeasureDeclaration{
		Name: AnnotationGate, Scenario: "agent-manager", Domain: "usage",
		Intent:    "Per-label annotation gate over a window: proven, partial or unproven, the model and role the proof is for, hold-out precision and recall, calibration error and gold size, beside coverage and the rate of proven labels.",
		Questions: []string{"which inferred labels are proven", "which model proves this label", "how often is the operator dissatisfied", "how often does a tool block the work", "what is the annotation coverage"},
		Params:    map[string]measurelib.Param{"window": {Name: "window", Type: measurelib.ParamTypeTimeWindow, Default: string(measurelib.TokenThisWeek)}},
		Result:    measurelib.Result{Kind: measurelib.ResultTable, ValueField: "label", Unit: "share", SummaryTemplate: "annotation gate ({window})"},
		Effect:    measurelib.EffectRead, RunEligible: true,
	}, func(ctx context.Context, request measurelib.MeasureRequest) (measurelib.MeasureResult, error) {
		report, err := h.annotationCoverage(ctx, request)
		if err != nil {
			return measurelib.MeasureResult{}, err
		}
		return measurelib.MeasureResult{Fields: annotationgate.Rows(report), Provenance: measurelib.Provenance{ExecutedQuery: fmt.Sprintf(
			"SELECT label_id, model, resource_role, state, observed_at FROM run_annotations WHERE observed_at >= %q AND observed_at < %q; gate data comes from the loaded pack",
			report.Window.Since, report.Window.Until)}}, nil
	})
}
