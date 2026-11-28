package discovery

import (
	"brand-manager/internal/discovery"

	discoveryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/discovery"
)

// resultToProto converts the domain scan Result into the wire DiscoveryResult.
func resultToProto(r discovery.Result) *discoveryv1.DiscoveryResult {
	out := &discoveryv1.DiscoveryResult{
		Scenario:    r.Scenario,
		Sources:     sourcesToProto(r.Sources),
		Confidence:  r.Confidence,
		Suggestions: append([]string(nil), r.Suggestions...),
	}
	// Only attach a draft when the scan matched something — an empty scan reports
	// a null draft so callers can distinguish "found nothing" from "found a blank
	// brand".
	if r.HasSources() {
		out.DraftBrand = draftToProto(r.Draft)
	}
	return out
}

// importResultToProto converts the domain ImportResult into the wire response.
func importResultToProto(r discovery.ImportResult) *discoveryv1.ImportBrandResponse {
	return &discoveryv1.ImportBrandResponse{
		BrandId:      r.Brand.ID,
		BrandName:    r.Brand.Name,
		BrandVersion: int32(r.Brand.Version),
		Sources:      sourcesToProto(r.Sources),
		Confidence:   r.Confidence,
	}
}

func sourcesToProto(in []discovery.Source) []*discoveryv1.DiscoverySource {
	out := make([]*discoveryv1.DiscoverySource, 0, len(in))
	for _, s := range in {
		out = append(out, &discoveryv1.DiscoverySource{
			File:       s.File,
			Type:       s.Type,
			Confidence: s.Confidence,
			Fields:     int32(s.Fields),
		})
	}
	return out
}

func draftToProto(d discovery.DraftBrand) *discoveryv1.DraftBrand {
	return &discoveryv1.DraftBrand{
		Name:        d.Name,
		Description: d.Description,
		Identity: &discoveryv1.DraftIdentity{
			DisplayName: d.Identity.DisplayName,
			Tagline:     d.Identity.Tagline,
			LogoPath:    d.Identity.LogoPath,
			FaviconPath: d.Identity.FaviconPath,
			IconPath:    d.Identity.IconPath,
		},
		Colors: &discoveryv1.DraftColors{
			Primary:    d.Colors.Primary,
			Secondary:  d.Colors.Secondary,
			Accent:     d.Colors.Accent,
			Background: d.Colors.Background,
			Surface:    d.Colors.Surface,
			Text:       d.Colors.Text,
			Error:      d.Colors.Error,
		},
	}
}
