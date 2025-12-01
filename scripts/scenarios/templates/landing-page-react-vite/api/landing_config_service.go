package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var (
	fallbackLanding            LandingConfigPayload
	defaultFallbackLandingJSON = []byte(`{
		"variant": {
			"id": 0,
			"slug": "fallback",
			"name": "Browser Automation Studio",
			"description": "Offline-safe fallback rendered when live APIs fail."
		},
		"axes": {
			"persona": "ops_leader",
			"jtbd": "launch_bundle",
			"conversionStyle": "demo_led"
		},
		"sections": [
			{
				"section_type": "hero",
				"order": 1,
				"enabled": true,
				"content": {
					"title": "Launch Browser Automation Studio Instantly",
					"subtitle": "Ship automation workflows with Stripe-ready billing and admin analytics.",
					"cta_text": "Get Started",
					"cta_url": "/checkout?plan=pro"
				}
			}
		],
		"pricing": {
			"bundle": {
				"id": 0,
				"bundle_key": "business_suite",
				"name": "Vrooli Business Suite",
				"stripe_product_id": "prod_business_suite",
				"credits_per_usd": 1000000,
				"display_credits_multiplier": 0.001,
				"display_credits_label": "credits",
				"environment": "production"
			},
			"monthly": [
				{
					"plan_name": "Pro Monthly",
					"plan_tier": "pro",
					"billing_interval": "month",
					"amount_cents": 14900,
					"currency": "usd",
					"intro_enabled": true,
					"intro_type": "flat_amount",
					"intro_amount_cents": 100,
					"intro_periods": 1,
					"intro_price_lookup_key": "pro_monthly_intro",
					"stripe_price_id": "price_pro_monthly",
					"monthly_included_credits": 25000000,
					"one_time_bonus_credits": 0,
					"plan_rank": 1,
					"bonus_type": "none",
					"display_weight": 10,
					"metadata": {
						"features": [
							"Team workflows",
							"Priority support",
							"Desktop bundle downloads"
						]
					}
				}
			],
			"yearly": [],
			"updated_at": "2025-01-01T00:00:00Z"
		},
		"downloads": [
			{
				"id": 0,
				"bundle_key": "business_suite",
				"platform": "windows",
				"artifact_url": "https://downloads.vrooli.local/business-suite/win/VrooliBusinessSuiteSetup.exe",
				"release_version": "1.0.0",
				"requires_entitlement": true
			}
		]
	}`)
)

func init() {
	path := filepath.Join("..", ".vrooli", "variants", "fallback.json")
	payload, err := loadFallbackLandingFromFile(path)
	if err != nil {
		log.Printf("failed to read fallback config at %s: %v; using baked defaults", path, err)
		payload, err = parseFallbackLandingConfig(defaultFallbackLandingJSON)
		if err != nil {
			panic(fmt.Sprintf("default fallback config invalid: %v", err))
		}
	}
	fallbackLanding = payload
}

// LandingConfigService aggregates variant, section, pricing, and download data.
type LandingConfigService struct {
	variantService  *VariantService
	contentService  *ContentService
	planService     *PlanService
	downloadService *DownloadService
}

// LandingConfigResponse is returned by GET /landing-config.
type LandingConfigResponse struct {
	Variant   LandingVariantSummary `json:"variant"`
	Sections  []LandingSection      `json:"sections"`
	Pricing   *PricingOverview      `json:"pricing"`
	Downloads []DownloadAsset       `json:"downloads"`
	Fallback  bool                  `json:"fallback"`
}

type LandingVariantSummary struct {
	ID          int               `json:"id,omitempty"`
	Slug        string            `json:"slug"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Axes        map[string]string `json:"axes,omitempty"`
}

type LandingSection struct {
	SectionType string                 `json:"section_type"`
	Content     map[string]interface{} `json:"content"`
	Order       int                    `json:"order"`
	Enabled     bool                   `json:"enabled"`
}

type LandingConfigPayload struct {
	Variant   LandingVariantSummary `json:"variant"`
	Sections  []LandingSection      `json:"sections"`
	Pricing   PricingOverview       `json:"pricing"`
	Downloads []DownloadAsset       `json:"downloads"`
}

func NewLandingConfigService(
	variantService *VariantService,
	contentService *ContentService,
	planService *PlanService,
	downloadService *DownloadService,
) *LandingConfigService {
	return &LandingConfigService{
		variantService:  variantService,
		contentService:  contentService,
		planService:     planService,
		downloadService: downloadService,
	}
}

func (s *LandingConfigService) GetLandingConfig(ctx context.Context, variantSlug string) (*LandingConfigResponse, error) {
	pricing, err := s.planService.GetPricingOverview()
	if err != nil {
		return s.fallbackWithReason("pricing_fetch_failed", err, nil)
	}

	downloads, err := s.downloadService.ListAssets(s.planService.BundleKey())
	if err != nil {
		return s.fallbackWithReason("download_list_failed", err, nil)
	}

	var variant *Variant
	if variantSlug != "" {
		variant, err = s.variantService.GetVariantBySlug(variantSlug)
	} else {
		variant, err = s.variantService.SelectVariant()
	}
	if err != nil || variant == nil {
		reason := "weighted_selection_failed"
		meta := map[string]interface{}{}
		if variantSlug != "" {
			reason = "variant_lookup_failed"
			meta["variant_slug"] = variantSlug
		}
		return s.fallbackWithReason(reason, err, meta)
	}

	sections, err := s.contentService.GetPublicSections(int64(variant.ID))
	if err != nil {
		return s.fallbackWithReason("section_fetch_failed", err, map[string]interface{}{
			"variant_id":   variant.ID,
			"variant_slug": variant.Slug,
		})
	}

	response := &LandingConfigResponse{
		Variant: LandingVariantSummary{
			ID:          variant.ID,
			Slug:        variant.Slug,
			Name:        variant.Name,
			Description: variant.Description,
			Axes:        variant.Axes,
		},
		Pricing:   pricing,
		Downloads: downloads,
		Fallback:  false,
	}

	landingSections := make([]LandingSection, 0, len(sections))
	for _, section := range sections {
		landingSections = append(landingSections, LandingSection{
			SectionType: section.SectionType,
			Content:     section.Content,
			Order:       section.Order,
			Enabled:     section.Enabled,
		})
	}
	sort.SliceStable(landingSections, func(i, j int) bool {
		return landingSections[i].Order < landingSections[j].Order
	})

	// ASSUMPTION: Every active variant must render at least one section and expose a hero.
	// If admins disable the hero or all sections we treat it as a misconfiguration and fail closed.
	if err := ensureRenderableSections(landingSections); err != nil {
		return s.fallbackWithReason("section_renderability_failed", err, map[string]interface{}{
			"variant_slug": variant.Slug,
		})
	}

	response.Sections = landingSections

	return response, nil
}

func (s *LandingConfigService) fallbackResponse(mark bool) *LandingConfigResponse {
	// fallbackLanding already contains pricing/download placeholders.
	return &LandingConfigResponse{
		Variant:   fallbackLanding.Variant,
		Sections:  fallbackLanding.Sections,
		Pricing:   &fallbackLanding.Pricing,
		Downloads: fallbackLanding.Downloads,
		Fallback:  mark,
	}
}

func (s *LandingConfigService) fallbackWithReason(reason string, err error, meta map[string]interface{}) (*LandingConfigResponse, error) {
	fields := map[string]interface{}{
		"reason": reason,
	}
	if err != nil {
		fields["error"] = err.Error()
	}
	for key, value := range meta {
		fields[key] = value
	}
	logStructured("landing_config_fallback", fields)
	return s.fallbackResponse(true), nil
}

func loadFallbackLandingFromFile(path string) (LandingConfigPayload, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return LandingConfigPayload{}, err
	}
	return parseFallbackLandingConfig(data)
}

type fallbackLandingPayload struct {
	Variant   LandingVariantSummary `json:"variant"`
	Sections  []fallbackSection     `json:"sections"`
	Pricing   *PricingOverview      `json:"pricing"`
	Downloads []DownloadAsset       `json:"downloads"`
	Axes      map[string]string     `json:"axes"`
}

type fallbackSection struct {
	SectionType string                 `json:"section_type"`
	Content     map[string]interface{} `json:"content"`
	Order       *int                   `json:"order"`
	Enabled     *bool                  `json:"enabled"`
}

func parseFallbackLandingConfig(data []byte) (LandingConfigPayload, error) {
	if len(data) == 0 {
		return LandingConfigPayload{}, fmt.Errorf("fallback config payload is empty")
	}

	var raw fallbackLandingPayload
	if err := json.Unmarshal(data, &raw); err != nil {
		return LandingConfigPayload{}, fmt.Errorf("parse fallback config: %w", err)
	}

	variantSlug := strings.TrimSpace(raw.Variant.Slug)
	if variantSlug == "" {
		return LandingConfigPayload{}, fmt.Errorf("fallback config missing variant slug")
	}

	if raw.Pricing == nil {
		return LandingConfigPayload{}, fmt.Errorf("fallback config missing pricing")
	}

	sections := normalizeFallbackSections(raw.Sections)
	if len(sections) == 0 {
		return LandingConfigPayload{}, fmt.Errorf("fallback config has no usable sections")
	}

	payload := LandingConfigPayload{
		Variant:   raw.Variant,
		Sections:  sections,
		Pricing:   *raw.Pricing,
		Downloads: normalizeDownloads(raw.Downloads),
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

		normalized = append(normalized, LandingSection{
			SectionType: sectionType,
			Content:     content,
			Order:       order,
			Enabled:     enabled,
		})
	}

	sort.SliceStable(normalized, func(i, j int) bool {
		if normalized[i].Order == normalized[j].Order {
			return i < j
		}
		return normalized[i].Order < normalized[j].Order
	})

	return normalized
}

func normalizeDownloads(downloads []DownloadAsset) []DownloadAsset {
	if downloads == nil {
		return []DownloadAsset{}
	}
	copied := make([]DownloadAsset, len(downloads))
	copy(copied, downloads)
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
