package design

import (
	"brand-manager/internal/design"

	designv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/design"
)

// designToProto converts the rendered domain Design into the wire response.
func designToProto(d design.Design) *designv1.GenerateDesignLanguageResponse {
	return &designv1.GenerateDesignLanguageResponse{
		BrandId:  d.BrandID,
		Markdown: d.Markdown,
	}
}
