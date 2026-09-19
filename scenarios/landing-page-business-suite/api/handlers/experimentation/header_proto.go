package variant

import (
	"fmt"
	"math"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
	"landing-page-business-suite-api/internal/experimentation"
)

// HeaderProto translates persisted landing-header configuration at the
// transport boundary. Domain configuration remains protobuf-free.
func HeaderProto(header experimentation.LandingHeaderConfig) (*sharedv1.LandingHeaderConfig, error) {
	links, err := headerLinksProto(header.Nav.Links)
	if err != nil {
		return nil, err
	}
	return &sharedv1.LandingHeaderConfig{
		Branding: &sharedv1.HeaderBrandingConfig{Mode: header.Branding.Mode, Label: header.Branding.Label, Subtitle: header.Branding.Subtitle, MobilePreference: header.Branding.MobilePreference},
		Nav:      &sharedv1.HeaderNavConfig{Links: links},
		Ctas: &sharedv1.HeaderCTAGroup{
			Primary:   &sharedv1.HeaderCTAConfig{Mode: header.Ctas.Primary.Mode, Label: header.Ctas.Primary.Label, Href: header.Ctas.Primary.Href, Variant: header.Ctas.Primary.Variant},
			Secondary: &sharedv1.HeaderCTAConfig{Mode: header.Ctas.Secondary.Mode, Label: header.Ctas.Secondary.Label, Href: header.Ctas.Secondary.Href, Variant: header.Ctas.Secondary.Variant},
		},
		Behavior: &sharedv1.HeaderBehaviorConfig{Sticky: header.Behavior.Sticky, HideOnScroll: header.Behavior.HideOnScroll},
	}, nil
}

func headerLinksProto(links []experimentation.HeaderNavLink) ([]*sharedv1.HeaderNavLink, error) {
	result := make([]*sharedv1.HeaderNavLink, 0, len(links))
	for _, link := range links {
		var sectionID *int32
		if link.SectionID != nil {
			if *link.SectionID < math.MinInt32 || *link.SectionID > math.MaxInt32 {
				return nil, fmt.Errorf("header link %q section ID %d is outside the protobuf int32 range", link.ID, *link.SectionID)
			}
			value := int32(*link.SectionID)
			sectionID = &value
		}
		children, err := headerLinksProto(link.Children)
		if err != nil {
			return nil, err
		}
		result = append(result, &sharedv1.HeaderNavLink{Id: link.ID, Type: link.Type, Label: link.Label, SectionType: link.SectionType, SectionId: sectionID, Anchor: link.Anchor, Href: link.Href, VisibleOn: &sharedv1.HeaderVisibilityConfig{Desktop: link.VisibleOn.Desktop, Mobile: link.VisibleOn.Mobile}, Children: children})
	}
	return result, nil
}
