package resourcecli

import (
	"github.com/vrooli/vrooli/internal/resources"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// --- blueprint converters -----------------------------------------------------

func blueprintReferenceMessages(refs []resources.BlueprintReference) []*cliv1.ResourceBlueprintReference {
	out := make([]*cliv1.ResourceBlueprintReference, 0, len(refs))
	for _, ref := range refs {
		out = append(out, &cliv1.ResourceBlueprintReference{
			Kind:  ref.Kind,
			Value: ref.Value,
		})
	}
	return out
}

func blueprintMessage(bp resources.Blueprint) *cliv1.ResourceBlueprint {
	return &cliv1.ResourceBlueprint{
		Schema:           bp.Schema,
		Name:             bp.Name,
		DisplayName:      bp.DisplayName,
		Category:         bp.Category,
		Summary:          bp.Summary,
		WhyItMatters:     bp.WhyItMatters,
		WhenToUse:        bp.WhenToUse,
		ExampleScenarios: bp.ExampleScenarios,
		IntegrationKind:  bp.IntegrationKind,
		PlatformSupport: &cliv1.ResourceBlueprintPlatformSupport{
			PortabilityTier: bp.PlatformSupport.PortabilityTier,
			Notes:           bp.PlatformSupport.Notes,
			Linux:           bp.PlatformSupport.Linux,
			Macos:           bp.PlatformSupport.MacOS,
			Windows:         bp.PlatformSupport.Windows,
		},
		Prerequisites:       bp.Prerequisites,
		Dependencies:        bp.Dependencies,
		SuggestedTemplate:   bp.SuggestedTemplate,
		ImplementationNotes: bp.ImplementationNotes,
		OperationalNotes:    bp.OperationalNotes,
		Risks:               bp.Risks,
		Status:              bp.Status,
		ReplacementFor:      bp.ReplacementFor,
		References:          blueprintReferenceMessages(bp.References),
		LastReviewed:        bp.LastReviewed,
	}
}

func blueprintSummaryMessage(s resources.BlueprintSummary) *cliv1.ResourceBlueprintSummary {
	return &cliv1.ResourceBlueprintSummary{
		Name:              s.Name,
		DisplayName:       s.DisplayName,
		Category:          s.Category,
		Status:            s.Status,
		IntegrationKind:   s.IntegrationKind,
		SuggestedTemplate: s.SuggestedTemplate,
		LastReviewed:      s.LastReviewed,
		Summary:           s.Summary,
	}
}

// --- blueprint response builders ----------------------------------------------

// ResourceBlueprintListResponse maps `resource blueprint list --json`.
func ResourceBlueprintListResponse(items []resources.Blueprint) *cliv1.ResourceBlueprintListResponse {
	resp := &cliv1.ResourceBlueprintListResponse{Success: true}
	for _, item := range items {
		resp.Blueprints = append(resp.Blueprints, blueprintMessage(item))
	}
	return resp
}

// ResourceBlueprintInfoResponse maps `resource blueprint info --json`.
func ResourceBlueprintInfoResponse(item resources.Blueprint) *cliv1.ResourceBlueprintInfoResponse {
	return &cliv1.ResourceBlueprintInfoResponse{
		Success:   true,
		Blueprint: blueprintMessage(item),
	}
}

// ResourceBlueprintSearchResponse maps `resource blueprint search --json`.
func ResourceBlueprintSearchResponse(query string, items []resources.Blueprint) *cliv1.ResourceBlueprintSearchResponse {
	resp := &cliv1.ResourceBlueprintSearchResponse{Success: true, Query: query}
	for _, item := range items {
		resp.Blueprints = append(resp.Blueprints, blueprintMessage(item))
	}
	return resp
}

// ResourceBlueprintValidationResponse maps `resource blueprint validate --json`.
func ResourceBlueprintValidationResponse(report resources.BlueprintValidationReport) *cliv1.ResourceBlueprintValidationReport {
	body := &cliv1.ResourceBlueprintValidationReportBody{Count: int32(report.Count)}
	for _, item := range report.Blueprints {
		body.Blueprints = append(body.Blueprints, blueprintSummaryMessage(item))
	}
	return &cliv1.ResourceBlueprintValidationReport{
		Success: true,
		Report:  body,
	}
}
