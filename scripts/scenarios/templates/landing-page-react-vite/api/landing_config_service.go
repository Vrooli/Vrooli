package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sort"
)

var fallbackLanding LandingConfigPayload

func init() {
	data, err := os.ReadFile(filepath.Join("..", ".vrooli", "variants", "fallback.json"))
	if err != nil {
		log.Printf("failed to read fallback config: %v", err)
		return
	}

	if err := json.Unmarshal(data, &fallbackLanding); err != nil {
		log.Printf("failed to parse fallback landing config: %v", err)
	}
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
	ID          int    `json:"id,omitempty"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
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
		return s.fallbackResponse(true), nil
	}

	downloads, err := s.downloadService.ListAssets(s.planService.BundleKey())
	if err != nil {
		return s.fallbackResponse(true), nil
	}

	var variant *Variant
	if variantSlug != "" {
		variant, err = s.variantService.GetVariantBySlug(variantSlug)
	} else {
		variant, err = s.variantService.SelectVariant()
	}
	if err != nil || variant == nil {
		return s.fallbackResponse(true), nil
	}

	sections, err := s.contentService.GetPublicSections(int64(variant.ID))
	if err != nil {
		return s.fallbackResponse(true), nil
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
