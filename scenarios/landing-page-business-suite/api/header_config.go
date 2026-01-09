package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// LandingHeaderConfig captures per-variant presentation controls for the public landing header.
type LandingHeaderConfig struct {
	Branding HeaderBrandingConfig `json:"branding"`
	Nav      HeaderNavConfig      `json:"nav"`
	Ctas     HeaderCTAGroup       `json:"ctas"`
	Behavior HeaderBehaviorConfig `json:"behavior"`
}

type HeaderBrandingConfig struct {
	Mode             string `json:"mode"` // none | logo | name | logo_and_name
	Label            string `json:"label,omitempty"`
	Subtitle         string `json:"subtitle,omitempty"`
	MobilePreference string `json:"mobile_preference,omitempty"` // auto | logo | name | stacked
}

type HeaderNavConfig struct {
	Links []HeaderNavLink `json:"links"`
}

type HeaderNavLink struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"` // section | downloads | custom | menu
	Label       string                 `json:"label"`
	SectionType string                 `json:"section_type,omitempty"`
	SectionID   *int                   `json:"section_id,omitempty"`
	Anchor      string                 `json:"anchor,omitempty"`
	Href        string                 `json:"href,omitempty"`
	VisibleOn   HeaderVisibilityConfig `json:"visible_on"`
	Children    []HeaderNavLink        `json:"children,omitempty"`
}

type HeaderVisibilityConfig struct {
	Desktop bool `json:"desktop"`
	Mobile  bool `json:"mobile"`
}

type HeaderCTAGroup struct {
	Primary   HeaderCTAConfig `json:"primary"`
	Secondary HeaderCTAConfig `json:"secondary"`
}

type HeaderCTAConfig struct {
	Mode    string `json:"mode"` // inherit_hero | downloads | custom | hidden
	Label   string `json:"label,omitempty"`
	Href    string `json:"href,omitempty"`
	Variant string `json:"variant,omitempty"` // solid | ghost
}

type HeaderBehaviorConfig struct {
	Sticky       bool `json:"sticky"`
	HideOnScroll bool `json:"hide_on_scroll"`
}

func defaultLandingHeaderConfig(variantName string) LandingHeaderConfig {
	return LandingHeaderConfig{
		Branding: HeaderBrandingConfig{
			Mode:             "logo_and_name",
			Label:            valueOrDefault(variantName, "Landing"),
			MobilePreference: "auto",
		},
		Nav: HeaderNavConfig{
			Links: []HeaderNavLink{},
		},
		Ctas: HeaderCTAGroup{
			Primary: HeaderCTAConfig{
				Mode:    "inherit_hero",
				Variant: "solid",
			},
			Secondary: HeaderCTAConfig{
				Mode:    "downloads",
				Variant: "ghost",
			},
		},
		Behavior: HeaderBehaviorConfig{
			Sticky:       true,
			HideOnScroll: false,
		},
	}
}

func normalizeLandingHeaderConfig(input *LandingHeaderConfig, variantName string) LandingHeaderConfig {
	def := defaultLandingHeaderConfig(variantName)
	if input == nil {
		return def
	}

	cfg := LandingHeaderConfig{
		Branding: def.Branding,
		Nav: HeaderNavConfig{
			Links: make([]HeaderNavLink, len(input.Nav.Links)),
		},
		Ctas: HeaderCTAGroup{
			Primary:   def.Ctas.Primary,
			Secondary: def.Ctas.Secondary,
		},
		Behavior: def.Behavior,
	}

	if strings.TrimSpace(input.Branding.Mode) != "" {
		cfg.Branding.Mode = strings.TrimSpace(input.Branding.Mode)
	}
	if label := strings.TrimSpace(input.Branding.Label); label != "" {
		cfg.Branding.Label = label
	} else {
		cfg.Branding.Label = def.Branding.Label
	}
	if subtitle := strings.TrimSpace(input.Branding.Subtitle); subtitle != "" {
		cfg.Branding.Subtitle = subtitle
	}
	if pref := strings.TrimSpace(input.Branding.MobilePreference); pref != "" {
		cfg.Branding.MobilePreference = pref
	}

	for idx, link := range input.Nav.Links {
		cfg.Nav.Links[idx] = normalizeHeaderNavLink(link, idx)
	}

	cfg.Ctas.Primary = normalizeHeaderCTA(input.Ctas.Primary, def.Ctas.Primary)
	cfg.Ctas.Secondary = normalizeHeaderCTA(input.Ctas.Secondary, def.Ctas.Secondary)

	if (input.Behavior != HeaderBehaviorConfig{}) {
		cfg.Behavior.Sticky = input.Behavior.Sticky
		if cfg.Behavior.Sticky {
			cfg.Behavior.HideOnScroll = input.Behavior.HideOnScroll
		} else {
			cfg.Behavior.HideOnScroll = false
		}
	}

	return cfg
}

func normalizeHeaderNavLink(link HeaderNavLink, index int) HeaderNavLink {
	normalized := HeaderNavLink{
		Type:        valueOrDefault(link.Type, "section"),
		Label:       strings.TrimSpace(link.Label),
		SectionType: strings.TrimSpace(link.SectionType),
		Anchor:      strings.TrimSpace(link.Anchor),
		Href:        strings.TrimSpace(link.Href),
		VisibleOn: HeaderVisibilityConfig{
			Desktop: true,
			Mobile:  true,
		},
	}
	if len(link.Children) > 0 {
		normalized.Children = make([]HeaderNavLink, len(link.Children))
		for idx, child := range link.Children {
			normalized.Children[idx] = normalizeHeaderNavLink(child, idx)
		}
	}
	if normalized.Label == "" {
		normalized.Label = "Section"
	}
	if link.SectionID != nil {
		id := *link.SectionID
		normalized.SectionID = &id
	}
	desktop := link.VisibleOn.Desktop
	mobile := link.VisibleOn.Mobile
	if !desktop && !mobile {
		desktop = true
		mobile = true
	}
	normalized.VisibleOn.Desktop = desktop
	normalized.VisibleOn.Mobile = mobile
	if strings.TrimSpace(link.ID) != "" {
		normalized.ID = link.ID
	} else {
		normalized.ID = fmt.Sprintf("%s-%d", normalized.Type, index)
	}
	return normalized
}

func normalizeHeaderCTA(input HeaderCTAConfig, fallback HeaderCTAConfig) HeaderCTAConfig {
	cfg := fallback
	if strings.TrimSpace(input.Mode) != "" {
		cfg.Mode = strings.TrimSpace(input.Mode)
	}
	if label := strings.TrimSpace(input.Label); label != "" {
		cfg.Label = label
	}
	if href := strings.TrimSpace(input.Href); href != "" {
		cfg.Href = href
	}
	if variant := strings.TrimSpace(input.Variant); variant != "" {
		cfg.Variant = variant
	}
	return cfg
}

func marshalHeaderConfig(cfg LandingHeaderConfig) ([]byte, error) {
	return json.Marshal(cfg)
}
