package measures

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"agent-manager/internal/annotationgate"
	"agent-manager/internal/runannotations"

	measurelib "github.com/vrooli/measures-go"
	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
)

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
			ProvenObserved: int64(label.ProvenObserved),
		}
		// The measurement's model and calibration are named only when one exists,
		// so an unmeasured label never appears measured for a model.
		if label.Gate.Split != "" {
			row.GateModel, row.GateRole = label.Gate.Model, label.Gate.ResourceRole
			row.Ece, row.Calibration, row.Temperature = label.Gate.ECE, annotationgate.Calibration(label.Gate), label.Gate.Temperature
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

func formatAnnotationFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
