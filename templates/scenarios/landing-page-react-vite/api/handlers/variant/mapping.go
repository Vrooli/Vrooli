package variant

import (
	"encoding/json"

	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	landingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"

	internalcontent "landing-page-react-vite-api/internal/content"
	internalvariant "landing-page-react-vite-api/internal/variant"
)

// --- variant <-> proto ---------------------------------------------------

// variantToProto maps a domain variant to its proto form. includeSEO controls
// whether the optional seo_config field is populated (admin single reads only).
func variantToProto(v *internalvariant.Variant, includeSEO bool) *landingv1.Variant {
	p := &landingv1.Variant{
		Id:           int64(v.ID),
		Slug:         v.Slug,
		Name:         v.Name,
		Description:  v.Description,
		Weight:       int32(v.Weight),
		Status:       v.Status,
		CreatedAt:    timestamppb.New(v.CreatedAt),
		UpdatedAt:    timestamppb.New(v.UpdatedAt),
		Axes:         v.Axes,
		HeaderConfig: headerToProto(v.HeaderConfig),
	}
	if v.ArchivedAt != nil {
		p.ArchivedAt = timestamppb.New(*v.ArchivedAt)
	}
	if includeSEO && v.SEOConfig != nil {
		p.SeoConfig = seoRawToProto(*v.SEOConfig)
	}
	return p
}

// --- header config <-> proto --------------------------------------------

func headerToProto(h internalvariant.LandingHeaderConfig) *landingv1.LandingHeaderConfig {
	return &landingv1.LandingHeaderConfig{
		Branding: &landingv1.HeaderBrandingConfig{
			Mode:             h.Branding.Mode,
			Label:            h.Branding.Label,
			Subtitle:         h.Branding.Subtitle,
			MobilePreference: h.Branding.MobilePreference,
		},
		Nav:      navToProto(h.Nav),
		Ctas:     ctaGroupToProto(h.Ctas),
		Behavior: &landingv1.HeaderBehaviorConfig{Sticky: h.Behavior.Sticky, HideOnScroll: h.Behavior.HideOnScroll},
	}
}

func navToProto(n internalvariant.HeaderNavConfig) *landingv1.HeaderNavConfig {
	links := make([]*landingv1.HeaderNavLink, 0, len(n.Links))
	for i := range n.Links {
		links = append(links, navLinkToProto(n.Links[i]))
	}
	return &landingv1.HeaderNavConfig{Links: links}
}

func navLinkToProto(l internalvariant.HeaderNavLink) *landingv1.HeaderNavLink {
	pl := &landingv1.HeaderNavLink{
		Id:          l.ID,
		Type:        l.Type,
		Label:       l.Label,
		SectionType: l.SectionType,
		Anchor:      l.Anchor,
		Href:        l.Href,
		VisibleOn:   &landingv1.HeaderVisibilityConfig{Desktop: l.VisibleOn.Desktop, Mobile: l.VisibleOn.Mobile},
	}
	if l.SectionID != nil {
		id := int32(*l.SectionID)
		pl.SectionId = &id
	}
	for i := range l.Children {
		pl.Children = append(pl.Children, navLinkToProto(l.Children[i]))
	}
	return pl
}

func ctaGroupToProto(g internalvariant.HeaderCTAGroup) *landingv1.HeaderCTAGroup {
	return &landingv1.HeaderCTAGroup{Primary: ctaToProto(g.Primary), Secondary: ctaToProto(g.Secondary)}
}

func ctaToProto(c internalvariant.HeaderCTAConfig) *landingv1.HeaderCTAConfig {
	return &landingv1.HeaderCTAConfig{Mode: c.Mode, Label: c.Label, Href: c.Href, Variant: c.Variant}
}

// headerFromProto maps an incoming proto header config to the domain type. A
// nil proto returns nil so callers can distinguish "leave unchanged".
func headerFromProto(p *landingv1.LandingHeaderConfig) *internalvariant.LandingHeaderConfig {
	if p == nil {
		return nil
	}
	cfg := internalvariant.LandingHeaderConfig{}
	if p.Branding != nil {
		cfg.Branding = internalvariant.HeaderBrandingConfig{
			Mode:             p.Branding.Mode,
			Label:            p.Branding.Label,
			Subtitle:         p.Branding.Subtitle,
			MobilePreference: p.Branding.MobilePreference,
		}
	}
	if p.Nav != nil {
		cfg.Nav.Links = make([]internalvariant.HeaderNavLink, 0, len(p.Nav.Links))
		for _, l := range p.Nav.Links {
			cfg.Nav.Links = append(cfg.Nav.Links, navLinkFromProto(l))
		}
	}
	if p.Ctas != nil {
		if p.Ctas.Primary != nil {
			cfg.Ctas.Primary = ctaFromProto(p.Ctas.Primary)
		}
		if p.Ctas.Secondary != nil {
			cfg.Ctas.Secondary = ctaFromProto(p.Ctas.Secondary)
		}
	}
	if p.Behavior != nil {
		cfg.Behavior = internalvariant.HeaderBehaviorConfig{Sticky: p.Behavior.Sticky, HideOnScroll: p.Behavior.HideOnScroll}
	}
	return &cfg
}

func navLinkFromProto(l *landingv1.HeaderNavLink) internalvariant.HeaderNavLink {
	link := internalvariant.HeaderNavLink{
		ID:          l.Id,
		Type:        l.Type,
		Label:       l.Label,
		SectionType: l.SectionType,
		Anchor:      l.Anchor,
		Href:        l.Href,
	}
	if l.VisibleOn != nil {
		link.VisibleOn = internalvariant.HeaderVisibilityConfig{Desktop: l.VisibleOn.Desktop, Mobile: l.VisibleOn.Mobile}
	}
	if l.SectionId != nil {
		id := int(*l.SectionId)
		link.SectionID = &id
	}
	for _, c := range l.Children {
		link.Children = append(link.Children, navLinkFromProto(c))
	}
	return link
}

func ctaFromProto(c *landingv1.HeaderCTAConfig) internalvariant.HeaderCTAConfig {
	return internalvariant.HeaderCTAConfig{Mode: c.Mode, Label: c.Label, Href: c.Href, Variant: c.Variant}
}

// --- SEO config <-> proto -----------------------------------------------

func seoRawToProto(raw json.RawMessage) *landingv1.VariantSEOConfig {
	if len(raw) == 0 {
		return nil
	}
	var cfg internalvariant.SEOConfigJSON
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil
	}
	return seoJSONToProto(cfg)
}

func seoJSONToProto(cfg internalvariant.SEOConfigJSON) *landingv1.VariantSEOConfig {
	p := &landingv1.VariantSEOConfig{
		Title:         cfg.Title,
		Description:   cfg.Description,
		OgTitle:       cfg.OGTitle,
		OgDescription: cfg.OGDescription,
		OgImageUrl:    cfg.OGImageURL,
		TwitterCard:   cfg.TwitterCard,
		CanonicalPath: cfg.CanonicalPath,
		Noindex:       cfg.NoIndex,
	}
	if len(cfg.StructuredData) > 0 {
		if s, err := structpb.NewStruct(cfg.StructuredData); err == nil {
			p.StructuredData = s
		}
	}
	return p
}

func seoProtoToJSON(p *landingv1.VariantSEOConfig) internalvariant.SEOConfigJSON {
	cfg := internalvariant.SEOConfigJSON{
		Title:         p.Title,
		Description:   p.Description,
		OGTitle:       p.OgTitle,
		OGDescription: p.OgDescription,
		OGImageURL:    p.OgImageUrl,
		TwitterCard:   p.TwitterCard,
		CanonicalPath: p.CanonicalPath,
		NoIndex:       p.Noindex,
	}
	if p.StructuredData != nil {
		cfg.StructuredData = p.StructuredData.AsMap()
	}
	return cfg
}

// seoProtoToRaw serializes an incoming proto SEO config to its stored JSON form.
func seoProtoToRaw(p *landingv1.VariantSEOConfig) (json.RawMessage, error) {
	if p == nil {
		return nil, nil
	}
	return json.Marshal(seoProtoToJSON(p))
}

// --- content section <-> proto (snapshot) -------------------------------

func sectionToProto(s internalcontent.Section) *landingv1.ContentSection {
	return &landingv1.ContentSection{
		Id:          s.ID,
		VariantId:   s.VariantID,
		SectionType: s.SectionType,
		Content:     mapToStruct(s.Content),
		Order:       int32(s.Order),
		Enabled:     s.Enabled,
		CreatedAt:   timestamppb.New(s.CreatedAt),
		UpdatedAt:   timestamppb.New(s.UpdatedAt),
	}
}

func sectionInputFromProto(p *landingv1.ContentSection) internalcontent.SectionInput {
	enabled := p.Enabled
	return internalcontent.SectionInput{
		SectionType: p.SectionType,
		Content:     structToMap(p.Content),
		Order:       int(p.Order),
		Enabled:     &enabled,
	}
}

func mapToStruct(m map[string]interface{}) *structpb.Struct {
	if m == nil {
		return nil
	}
	s, err := structpb.NewStruct(m)
	if err != nil {
		return nil
	}
	return s
}

func structToMap(s *structpb.Struct) map[string]interface{} {
	if s == nil {
		return map[string]interface{}{}
	}
	return s.AsMap()
}
