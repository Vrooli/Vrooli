package generation

import (
	"brand-manager/internal/generation"

	generationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/generation"
)

// statusesToProto converts the domain text-provider statuses into the wire shape.
func statusesToProto(statuses []generation.ProviderStatus) []*generationv1.ProviderStatus {
	out := make([]*generationv1.ProviderStatus, 0, len(statuses))
	for _, s := range statuses {
		out = append(out, &generationv1.ProviderStatus{Name: s.Name, Available: s.Available})
	}
	return out
}

// imageBackendStatusToProto converts the domain image-backend status into the
// wire shape.
func imageBackendStatusToProto(s generation.ImageBackendStatus) *generationv1.GetImageBackendStatusResponse {
	ops := make([]*generationv1.ImageOperationStatus, 0, len(s.Operations))
	for _, op := range s.Operations {
		ops = append(ops, &generationv1.ImageOperationStatus{
			Operation: op.Operation,
			Ready:     op.Ready,
			ModelId:   op.ModelID,
			Tier:      op.Tier,
			Hint:      op.Hint,
			Warnings:  append([]string(nil), op.Warnings...),
		})
	}
	return &generationv1.GetImageBackendStatusResponse{
		Available:  s.Available,
		Detail:     s.Detail,
		Operations: ops,
	}
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

// imageResultToProto converts a domain ImageResult into the shared
// BrandImageAsset wire shape.
func imageResultToProto(r generation.ImageResult) *generationv1.BrandImageAsset {
	return &generationv1.BrandImageAsset{
		BrandId:   r.BrandID,
		AssetId:   r.AssetID,
		Kind:      r.Kind,
		Filename:  r.Filename,
		MimeType:  r.MimeType,
		Size:      r.Size,
		ModelId:   r.ModelID,
		Tier:      r.Tier,
		Canonical: r.Canonical,
		Warnings:  append([]string(nil), r.Warnings...),
	}
}

// deriveIconsResultToProto converts the derived icon set into the wire shape.
func deriveIconsResultToProto(icons []generation.ImageResult, warnings []string) *generationv1.DeriveBrandIconsResponse {
	out := make([]*generationv1.BrandImageAsset, 0, len(icons))
	for _, ic := range icons {
		out = append(out, imageResultToProto(ic))
	}
	return &generationv1.DeriveBrandIconsResponse{
		Icons:    out,
		Warnings: append([]string(nil), warnings...),
	}
}
