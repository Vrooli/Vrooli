package reporting

import (
	"context"
	"encoding/json"
	"io"

	"test-genie/internal/requirements/enrichment"
	"test-genie/internal/requirements/parsing"
)

// JSONRenderer renders reports as JSON.
type JSONRenderer struct{}

// NewJSONRenderer creates a new JSONRenderer.
func NewJSONRenderer() *JSONRenderer {
	return &JSONRenderer{}
}

// Render writes a JSON report.
func (r *JSONRenderer) Render(ctx context.Context, index *parsing.ModuleIndex, summary enrichment.Summary, opts Options, w io.Writer) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	data := BuildReportData(index, summary, opts)

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// RenderSummary writes only the summary portion as JSON.
func (r *JSONRenderer) RenderSummary(ctx context.Context, summary enrichment.Summary, w io.Writer) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	data := SummaryData{
		Total:       summary.Total,
		CriticalGap: summary.CriticalityGap,
	}

	// Convert status counts
	for status, count := range summary.ByDeclaredStatus {
		switch status {
		case "complete":
			data.Complete = count
		case "in_progress":
			data.InProgress = count
		case "pending":
			data.Pending = count
		}
	}

	// Convert live status counts
	for status, count := range summary.ByLiveStatus {
		switch status {
		case "passed":
			data.LivePassed = count
		case "failed":
			data.LiveFailed = count
		}
	}

	// Calculate rates
	if data.Total > 0 {
		data.CompletionRate = float64(data.Complete) / float64(data.Total) * 100
	}
	tested := data.LivePassed + data.LiveFailed
	if tested > 0 {
		data.PassRate = float64(data.LivePassed) / float64(tested) * 100
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// RenderRequirements writes only the requirements as JSON.
func (r *JSONRenderer) RenderRequirements(ctx context.Context, index *parsing.ModuleIndex, opts Options, w io.Writer) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	var requirements []RequirementData

	for _, module := range index.Modules {
		for _, req := range module.Requirements {
			reqData := RequirementData{
				ID:          req.ID,
				Title:       req.Title,
				Status:      string(req.Status),
				LiveStatus:  string(req.LiveStatus),
				Criticality: string(req.Criticality),
			}

			if opts.IncludeValidations {
				for _, val := range req.Validations {
					reqData.Validations = append(reqData.Validations, ValidationData{
						Type:       string(val.Type),
						Ref:        val.Ref,
						Phase:      val.Phase,
						Status:     string(val.Status),
						LiveStatus: string(val.LiveStatus),
					})
				}
			}

			requirements = append(requirements, reqData)
		}
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(requirements)
}

// RenderValidations writes only the validations as JSON.
func (r *JSONRenderer) RenderValidations(ctx context.Context, index *parsing.ModuleIndex, phase string, w io.Writer) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	type ValidationWithReq struct {
		RequirementID string `json:"requirement_id"`
		ValidationData
	}

	var validations []ValidationWithReq

	for _, module := range index.Modules {
		for _, req := range module.Requirements {
			for _, val := range req.Validations {
				if phase != "" && val.Phase != phase {
					continue
				}
				validations = append(validations, ValidationWithReq{
					RequirementID: req.ID,
					ValidationData: ValidationData{
						Type:       string(val.Type),
						Ref:        val.Ref,
						Phase:      val.Phase,
						Status:     string(val.Status),
						LiveStatus: string(val.LiveStatus),
					},
				})
			}
		}
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(validations)
}
