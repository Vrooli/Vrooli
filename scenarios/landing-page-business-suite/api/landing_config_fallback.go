package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// fallbackResponse builds a renderable response when the live configuration
// cannot be used. It deliberately shares the response contract with the
// normal configuration path.
func (s *LandingConfigService) fallbackResponse(mark bool) *LandingConfigResponse {
	payload := s.fallbackPayload()
	response := &LandingConfigResponse{
		Variant:   payload.Variant,
		Sections:  payload.Sections,
		Pricing:   payload.Pricing,
		Downloads: payload.Downloads,
		Header:    payload.Header,
		Fallback:  mark,
	}
	if mark {
		trimmedSlug := strings.TrimSpace(response.Variant.Slug)
		if trimmedSlug == "" || strings.EqualFold(trimmedSlug, "control") {
			response.Variant.Slug = "fallback"
		}
	}
	if s.configStore != nil {
		if siteBranding := s.configStore.GetBranding(); siteBranding != nil {
			response.Branding = &LandingBranding{
				SiteName:             siteBranding.SiteName,
				Tagline:              siteBranding.Tagline,
				LogoURL:              siteBranding.LogoURL,
				LogoIconURL:          siteBranding.LogoIconURL,
				FaviconURL:           siteBranding.FaviconURL,
				ThemePrimaryColor:    siteBranding.ThemePrimaryColor,
				ThemeBackgroundColor: siteBranding.ThemeBackgroundColor,
				SupportChatURL:       siteBranding.SupportChatURL,
				SupportEmail:         siteBranding.SupportEmail,
				ComingSoonEnabled:    siteBranding.ComingSoonEnabled,
				ComingSoonMessage:    siteBranding.ComingSoonMessage,
			}
		}
	}
	return response
}

func (s *LandingConfigService) fallbackWithReason(reason string, err error, meta map[string]interface{}) (*LandingConfigResponse, error) {
	fields := map[string]interface{}{"reason": reason}
	if err != nil {
		fields["error"] = err.Error()
	}
	for key, value := range meta {
		fields[key] = value
	}
	logStructured("landing_config_fallback", fields)
	return s.fallbackResponse(true), nil
}

func (s *LandingConfigService) fallbackPayload() *LandingConfigPayload {
	provider := s.fallbackProvider
	if provider == nil {
		provider = defaultFallbackProvider
	}
	return cloneLandingPayload(provider())
}

func defaultFallbackProvider() *LandingConfigPayload { return fallbackLanding }

func loadFallbackLandingFromFile(path string) (*LandingConfigPayload, error) {
	// #nosec G304 -- init supplies only scenario-owned fallback paths.
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseFallbackLandingConfig(data)
}

type fallbackLandingPayload struct {
	Variant   LandingVariantSummary `json:"variant"`
	Sections  []fallbackSection     `json:"sections"`
	Pricing   json.RawMessage       `json:"pricing"`
	Downloads json.RawMessage       `json:"downloads"`
	Axes      map[string]string     `json:"axes"`
	Header    LandingHeaderConfig   `json:"header"`
}

type fallbackSection struct {
	SectionType string                 `json:"section_type"`
	Content     map[string]interface{} `json:"content"`
	Order       *int                   `json:"order"`
	Enabled     *bool                  `json:"enabled"`
}

func parseFallbackLandingConfig(data []byte) (*LandingConfigPayload, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("fallback config payload is empty")
	}
	var raw fallbackLandingPayload
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse fallback config: %w", err)
	}
	variantSlug := strings.TrimSpace(raw.Variant.Slug)
	if variantSlug == "" {
		return nil, fmt.Errorf("fallback config missing variant slug")
	}
	if len(raw.Pricing) == 0 {
		return nil, fmt.Errorf("fallback config missing pricing")
	}
	var pricing PricingOverview
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(raw.Pricing, &pricing); err != nil {
		return nil, fmt.Errorf("parse fallback pricing: %w", err)
	}
	sections := normalizeFallbackSections(raw.Sections)
	if len(sections) == 0 {
		return nil, fmt.Errorf("fallback config has no usable sections")
	}
	downloadApps, err := parseFallbackDownloads(raw.Downloads)
	if err != nil {
		return nil, fmt.Errorf("parse fallback downloads: %w", err)
	}
	payload := &LandingConfigPayload{
		Variant: raw.Variant, Sections: sections, Pricing: &pricing,
		Downloads: normalizeDownloads(downloadApps),
		Header:    normalizeLandingHeaderConfig(&raw.Header, raw.Variant.Name),
	}
	payload.Variant.Slug = variantSlug
	if len(payload.Variant.Axes) == 0 && len(raw.Axes) > 0 {
		payload.Variant.Axes = raw.Axes
	}
	return payload, nil
}

func normalizeFallbackSections(sections []fallbackSection) []LandingSection {
	normalized := make([]LandingSection, 0, len(sections))
	for idx, section := range sections {
		sectionType := strings.TrimSpace(section.SectionType)
		if sectionType == "" {
			continue
		}
		order := idx + 1
		if section.Order != nil && *section.Order > 0 {
			order = *section.Order
		}
		enabled := true
		if section.Enabled != nil {
			enabled = *section.Enabled
		}
		content := section.Content
		if content == nil {
			content = map[string]interface{}{}
		}
		normalized = append(normalized, LandingSection{SectionType: sectionType, Content: content, Order: order, Enabled: enabled})
	}
	sort.SliceStable(normalized, func(i, j int) bool {
		if normalized[i].Order == normalized[j].Order {
			return i < j
		}
		return normalized[i].Order < normalized[j].Order
	})
	return normalized
}

func parseFallbackDownloads(raw json.RawMessage) ([]DownloadApp, error) {
	if len(raw) == 0 {
		return []DownloadApp{}, nil
	}
	var appsGuess []DownloadApp
	if err := json.Unmarshal(raw, &appsGuess); err == nil {
		hasStructuredApps := false
		for _, app := range appsGuess {
			if strings.TrimSpace(app.AppKey) != "" || len(app.Platforms) > 0 {
				hasStructuredApps = true
				break
			}
		}
		if hasStructuredApps {
			for i := range appsGuess {
				if appsGuess[i].Platforms == nil {
					appsGuess[i].Platforms = []DownloadAsset{}
				}
			}
			return appsGuess, nil
		}
	}
	var flatAssets []DownloadAsset
	if err := json.Unmarshal(raw, &flatAssets); err != nil {
		return nil, err
	}
	if len(flatAssets) == 0 {
		return []DownloadApp{}, nil
	}
	bundleKey, appKey := flatAssets[0].BundleKey, flatAssets[0].AppKey
	if strings.TrimSpace(appKey) == "" {
		appKey = "bundle_downloads"
	}
	return []DownloadApp{{BundleKey: bundleKey, AppKey: appKey, Name: "Bundle downloads", Tagline: "Installer payload generated from fallback config", Platforms: flatAssets}}, nil
}

func normalizeDownloads(downloads []DownloadApp) []DownloadApp {
	if downloads == nil {
		return []DownloadApp{}
	}
	copied := make([]DownloadApp, len(downloads))
	for i, download := range downloads {
		copied[i] = download
		copied[i].Platforms = cloneDownloadAssets(download.Platforms)
		copied[i].InstallSteps = append([]string{}, download.InstallSteps...)
		copied[i].Storefronts = cloneStorefronts(download.Storefronts)
	}
	return copied
}

func ensureRenderableSections(sections []LandingSection) error {
	if len(sections) == 0 {
		return fmt.Errorf("no enabled sections configured")
	}
	for _, section := range sections {
		if strings.EqualFold(section.SectionType, "hero") {
			return nil
		}
	}
	return fmt.Errorf("hero section missing")
}

func cloneLandingPayload(payload *LandingConfigPayload) *LandingConfigPayload {
	if payload == nil {
		return &LandingConfigPayload{}
	}
	cloned := &LandingConfigPayload{
		Variant:  LandingVariantSummary{ID: payload.Variant.ID, Slug: payload.Variant.Slug, Name: payload.Variant.Name, Description: payload.Variant.Description, Axes: cloneStringMap(payload.Variant.Axes)},
		Sections: cloneLandingSections(payload.Sections), Downloads: cloneDownloads(payload.Downloads), Header: cloneHeaderConfig(payload.Header, payload.Variant.Name),
	}
	cloned.Pricing = clonePricing(payload.Pricing)
	return cloned
}

func cloneLandingSections(sections []LandingSection) []LandingSection {
	if len(sections) == 0 {
		return []LandingSection{}
	}
	cloned := make([]LandingSection, len(sections))
	for i, section := range sections {
		cloned[i] = LandingSection{SectionType: section.SectionType, Content: cloneContentMap(section.Content), Order: section.Order, Enabled: section.Enabled}
	}
	return cloned
}

func cloneContentMap(content map[string]interface{}) map[string]interface{} {
	if content == nil {
		return map[string]interface{}{}
	}
	data, err := json.Marshal(content)
	if err != nil {
		copy := make(map[string]interface{}, len(content))
		for k, v := range content {
			copy[k] = v
		}
		return copy
	}
	var copy map[string]interface{}
	if err := json.Unmarshal(data, &copy); err != nil || copy == nil {
		return map[string]interface{}{}
	}
	return copy
}

func cloneDownloads(downloads []DownloadApp) []DownloadApp {
	if len(downloads) == 0 {
		return []DownloadApp{}
	}
	cloned := make([]DownloadApp, 0, len(downloads))
	for _, app := range downloads {
		cloned = append(cloned, DownloadApp{ID: app.ID, BundleKey: app.BundleKey, AppKey: app.AppKey, Name: app.Name, Tagline: app.Tagline, Description: app.Description, InstallOverview: app.InstallOverview, InstallSteps: append([]string{}, app.InstallSteps...), Storefronts: cloneStorefronts(app.Storefronts), Metadata: cloneContentMap(app.Metadata), DisplayOrder: app.DisplayOrder, Platforms: cloneDownloadAssets(app.Platforms)})
	}
	return cloned
}

func cloneStorefronts(storefronts []DownloadStorefront) []DownloadStorefront {
	if len(storefronts) == 0 {
		return []DownloadStorefront{}
	}
	copied := make([]DownloadStorefront, len(storefronts))
	copy(copied, storefronts)
	return copied
}

func cloneDownloadAssets(assets []DownloadAsset) []DownloadAsset {
	if len(assets) == 0 {
		return []DownloadAsset{}
	}
	copied := make([]DownloadAsset, len(assets))
	for i, asset := range assets {
		copied[i] = DownloadAsset{ID: asset.ID, BundleKey: asset.BundleKey, AppKey: asset.AppKey, Platform: asset.Platform, ArtifactURL: asset.ArtifactURL, ReleaseVersion: asset.ReleaseVersion, ReleaseNotes: asset.ReleaseNotes, Checksum: asset.Checksum, RequiresEntitlement: asset.RequiresEntitlement, Metadata: cloneContentMap(asset.Metadata)}
	}
	return copied
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}
	copy := make(map[string]string, len(input))
	for k, v := range input {
		copy[k] = v
	}
	return copy
}

func cloneHeaderConfig(cfg LandingHeaderConfig, variantName string) LandingHeaderConfig {
	copy := cfg
	copy.Nav.Links = append([]HeaderNavLink{}, cfg.Nav.Links...)
	return normalizeLandingHeaderConfig(&copy, variantName)
}

func clonePricing(pricing *PricingOverview) *PricingOverview {
	if pricing == nil {
		return nil
	}
	cloned := proto.Clone(pricing)
	if cloned == nil {
		return nil
	}
	return cloned.(*PricingOverview)
}
