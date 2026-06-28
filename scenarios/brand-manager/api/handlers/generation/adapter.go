package generation

import (
	"brand-manager/internal/generation"

	generationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/generation"
)

// statusesToProto converts the domain provider statuses into the wire shape.
func statusesToProto(statuses []generation.ProviderStatus) []*generationv1.ProviderStatus {
	out := make([]*generationv1.ProviderStatus, 0, len(statuses))
	for _, s := range statuses {
		out = append(out, &generationv1.ProviderStatus{Name: s.Name, Available: s.Available})
	}
	return out
}

// elementsResultToProto converts the domain ElementsResult into the wire shape.
func elementsResultToProto(r generation.ElementsResult) *generationv1.GenerateBrandElementsResponse {
	results := make([]*generationv1.ElementResult, 0, len(r.Results))
	for _, e := range r.Results {
		results = append(results, &generationv1.ElementResult{
			Element: e.Element,
			Status:  e.Status,
			Detail:  e.Detail,
		})
	}
	return &generationv1.GenerateBrandElementsResponse{
		Results:      results,
		Applied:      append([]string(nil), r.Applied...),
		Provider:     r.Provider,
		Model:        r.Model,
		BrandVersion: int32(r.Version),
	}
}

// imageResultToProto converts the domain ImageResult into the wire shape.
func imageResultToProto(r generation.ImageResult) *generationv1.GenerateBrandImageResponse {
	return &generationv1.GenerateBrandImageResponse{
		BrandId:  r.BrandID,
		AssetId:  r.AssetID,
		Type:     r.Type,
		Filename: r.Filename,
		MimeType: r.MimeType,
		Size:     r.Size,
		Provider: r.Provider,
		Model:    r.Model,
	}
}
