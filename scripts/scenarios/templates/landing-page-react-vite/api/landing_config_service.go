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

	"google.golang.org/protobuf/encoding/protojson"
)

var (
	fallbackLanding            LandingConfigPayload
	defaultFallbackLandingJSON = []byte(`{
		"variant": {
			"id": 0,
			"slug": "fallback",
			"name": "Vrooli Ascension",
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
					"title": "Launch Vrooli Ascension Instantly",
					"subtitle": "Ship automation workflows with Stripe-ready billing and admin analytics.",
					"cta_text": "Get Started",
					"cta_url": "/checkout?plan=pro",
					"image_url": "/assets/fallback/hero.png"
				}
			},
			{
				"section_type": "video",
				"order": 2,
				"enabled": true,
				"content": {
					"title": "Watch the landing/runtime handoff",
					"videoUrl": "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
					"thumbnailUrl": "/assets/fallback/video-thumb.png",
					"caption": "Tour the variant selector, analytics lane, and gated downloads in four minutes."
				}
			},
			{
				"section_type": "features",
				"order": 3,
				"enabled": true,
				"content": {
					"title": "Everything you need to sell automation bundles",
					"subtitle": "Variants, analytics, download gating, and entitlement checks all live in one repo.",
					"features": [
						{
							"title": "A/B-tested landing pages",
							"description": "Variants, analytics, and agent customization in one template.",
							"icon": "zap"
						},
						{
							"title": "Subscription awareness",
							"description": "Pricing auto-syncs from Stripe metadata and intro offers.",
							"icon": "shield"
						},
						{
							"title": "Bundle downloads",
							"description": "Gate Windows, macOS, and Linux installers behind entitlements.",
							"icon": "sparkles"
						}
					]
				}
			},
			{
				"section_type": "testimonials",
				"order": 4,
				"enabled": true,
				"content": {
					"title": "Proof from operators",
					"subtitle": "How teams describe Vrooli Ascension hand-offs during reviews.",
					"testimonials": [
						{
							"name": "Marina Patel",
							"role": "VP Operations",
							"company": "Clause",
							"content": "We sold enterprise seats the same week we scaffolded this template. Downloads + billing landed perfectly.",
							"rating": 5
						},
						{
							"name": "Devon Brooks",
							"role": "Founder",
							"company": "Atlas Automations",
							"content": "Entitlement-aware installers made our desktop launch credible with zero extra wiring.",
							"rating": 5
						},
						{
							"name": "Lina Gomez",
							"role": "Product Lead",
							"company": "Brightlab",
							"content": "Landing + admin portal shipped with metrics, FAQ, and CTA sequencing ready for paid media.",
							"rating": 5
						}
					]
				}
			},
			{
				"section_type": "pricing",
				"order": 5,
				"enabled": true,
				"content": {
					"title": "Simple, transparent pricing",
					"subtitle": "Bundled credit packs for Vrooli Ascension.",
					"tiers": [
						{
							"name": "Solo",
							"price": "$49",
							"description": "1 workspace, 5M included credits, email support",
							"features": [
								"Solo workspace",
								"5M included credits",
								"Email support"
							],
							"cta_text": "Start for $1",
							"cta_url": "/checkout?plan=solo"
						},
						{
							"name": "Pro",
							"price": "$149",
							"description": "3 workspaces, priority support, desktop bundle downloads",
							"features": [
								"Team workflows",
								"Priority support",
								"Desktop bundle downloads"
							],
							"cta_text": "Upgrade",
							"cta_url": "/checkout?plan=pro",
							"highlighted": true
						},
						{
							"name": "Studio",
							"price": "$349",
							"description": "Unlimited automations, dedicated architect, enterprise compliance",
							"features": [
								"Unlimited automations",
								"Dedicated architect",
								"Enterprise compliance"
							],
							"cta_text": "Talk to sales",
							"cta_url": "/contact"
						}
					]
				}
			},
			{
				"section_type": "downloads",
				"order": 6,
				"enabled": true,
				"content": {
					"title": "Download Vrooli Ascension",
					"subtitle": "macOS, Windows, Linux, and store links inherit entitlement gating by default."
				}
			},
			{
				"section_type": "faq",
				"order": 7,
				"enabled": true,
				"content": {
					"title": "Guardrails teams ask about before launch",
					"subtitle": "From staging folders to download gating, here are the answers ops leaders reference most often.",
					"faqs": [
						{
							"question": "How do downloads stay in sync with entitlements?",
							"answer": "Gated installers call the downloads API, which verifies the subscription before handing back the asset."
						},
						{
							"question": "Can we restyle every section?",
							"answer": "Yes. Each section maps to styling.json tokens and can be updated live through the admin portal."
						},
						{
							"question": "Do variants inherit pricing and analytics?",
							"answer": "All variants read from the same pricing overview and emit download + CTA events with their slug."
						}
					]
				}
			},
			{
				"section_type": "cta",
				"order": 8,
				"enabled": true,
				"content": {
					"title": "Deploy Vrooli Ascension in under an hour",
					"subtitle": "Vrooli landing system manages subscriptions, download gating, and future upgrades.",
					"cta_text": "Book a live demo",
					"cta_url": "/contact"
				}
			},
			{
				"section_type": "footer",
				"order": 9,
				"enabled": true,
				"content": {
					"company_name": "Vrooli Business Suite",
					"tagline": "Clause-inspired landings with analytics, download gating, and styling guardrails baked in.",
					"columns": [
						{
							"title": "Product",
							"links": [
								{ "label": "Features", "url": "#features" },
								{ "label": "Pricing", "url": "#pricing" },
								{ "label": "Downloads", "url": "#downloads-section" }
							]
						},
						{
							"title": "Company",
							"links": [
								{ "label": "Docs", "url": "/docs" },
								{ "label": "PRD", "url": "/prd" },
								{ "label": "Careers", "url": "/careers" }
							]
						},
						{
							"title": "Legal",
							"links": [
								{ "label": "Privacy", "url": "/privacy" },
								{ "label": "Terms", "url": "/terms" },
								{ "label": "Security", "url": "/security" }
							]
						}
					],
					"social_links": {
						"github": "https://github.com/vrooli",
						"twitter": "https://twitter.com/vrooli",
						"linkedin": "https://www.linkedin.com/company/vrooli",
						"email": "hello@vrooli.com"
					},
					"copyright": "© 2025 Vrooli. All rights reserved."
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
				"bundle_key": "business_suite",
				"app_key": "automation_studio",
				"name": "Vrooli Ascension",
				"tagline": "Clause-grade desktop automations",
				"description": "Desktop suite that orchestrates automations, download rail governance, and analytics traced per variant.",
				"install_overview": "Pick your OS, download the installer, and sign in with the email tied to your plan to unlock entitlement-gated downloads.",
				"install_steps": [
					"Download the installer for your OS",
					"Launch the setup wizard and finish the install",
					"Sign in with your subscription email to unlock the workspace"
				],
				"storefronts": [
					{
						"store": "app_store",
						"label": "macOS App Store",
						"url": "https://apps.apple.com/app/id000000",
						"badge": "Download on the App Store"
					}
				],
				"platforms": [
					{
						"bundle_key": "business_suite",
						"app_key": "automation_studio",
						"platform": "windows",
						"artifact_url": "https://downloads.vrooli.local/business-suite/win/VrooliBusinessSuiteSetup.exe",
						"release_version": "1.0.0",
						"release_notes": "Initial GA release with Vrooli Ascension.",
						"requires_entitlement": true,
						"metadata": {
							"size_mb": 210
						}
					},
					{
						"bundle_key": "business_suite",
						"app_key": "automation_studio",
						"platform": "mac",
						"artifact_url": "https://downloads.vrooli.local/business-suite/mac/VrooliBusinessSuite.dmg",
						"release_version": "1.0.0",
						"release_notes": "Universal build for Apple Silicon and Intel.",
						"requires_entitlement": true,
						"metadata": {
							"size_mb": 190
						}
					},
					{
						"bundle_key": "business_suite",
						"app_key": "automation_studio",
						"platform": "linux",
						"artifact_url": "https://downloads.vrooli.local/business-suite/linux/vrooli-business-suite.tar.gz",
						"release_version": "1.0.0",
						"release_notes": "AppImage bundle tested on Ubuntu/Debian.",
						"requires_entitlement": true,
						"metadata": {
							"size_mb": 205
						}
					}
				]
			},
			{
				"bundle_key": "business_suite",
				"app_key": "command_center",
				"name": "Vrooli Command Center",
				"tagline": "Mobile companion + approvals",
				"description": "Approve downloads, monitor install attempts, and push updates from iOS/Android or lightweight desktop utilities.",
				"install_overview": "Install from your preferred store for mobile or download the desktop notifier to watch bundle health.",
				"install_steps": [
					"Install from the App Store or Google Play",
					"Enable notifications for entitlement changes",
					"Link the workspace to sync download telemetry"
				],
				"storefronts": [
					{
						"store": "app_store",
						"label": "Apple App Store",
						"url": "https://apps.apple.com/app/id000111",
						"badge": "Download on the App Store"
					},
					{
						"store": "play_store",
						"label": "Google Play",
						"url": "https://play.google.com/store/apps/details?id=vrooli.command",
						"badge": "Get it on Google Play"
					}
				],
				"platforms": [
					{
						"bundle_key": "business_suite",
						"app_key": "command_center",
						"platform": "windows",
						"artifact_url": "https://downloads.vrooli.local/command-center/win/VrooliCommandCenter.exe",
						"release_version": "0.9.5",
						"release_notes": "Preview build with entitlement notifications.",
						"requires_entitlement": false,
						"metadata": {
							"size_mb": 120,
							"channel": "beta"
						}
					},
					{
						"bundle_key": "business_suite",
						"app_key": "command_center",
						"platform": "mac",
						"artifact_url": "https://downloads.vrooli.local/command-center/mac/VrooliCommandCenter.dmg",
						"release_version": "0.9.5",
						"release_notes": "Signed build with push notification helpers.",
						"requires_entitlement": false,
						"metadata": {
							"size_mb": 115,
							"channel": "beta"
						}
					},
					{
						"bundle_key": "business_suite",
						"app_key": "command_center",
						"platform": "linux",
						"artifact_url": "https://downloads.vrooli.local/command-center/linux/VrooliCommandCenter.tar.gz",
						"release_version": "0.9.5",
						"release_notes": "Preview release for Debian/Ubuntu.",
						"requires_entitlement": false,
						"metadata": {
							"size_mb": 118,
							"channel": "beta"
						}
					}
				]
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
	brandingService *BrandingService
}

// LandingConfigResponse is returned by GET /landing-config.
type LandingConfigResponse struct {
	Variant   LandingVariantSummary `json:"variant"`
	Sections  []LandingSection      `json:"sections"`
	Pricing   *PricingOverview      `json:"pricing"`
	Downloads []DownloadApp         `json:"downloads"`
	Header    LandingHeaderConfig   `json:"header"`
	Branding  *LandingBranding      `json:"branding,omitempty"`
	Fallback  bool                  `json:"fallback"`
}

// LandingBranding contains public branding fields for the frontend.
type LandingBranding struct {
	SiteName             string  `json:"site_name"`
	Tagline              *string `json:"tagline,omitempty"`
	LogoURL              *string `json:"logo_url,omitempty"`
	LogoIconURL          *string `json:"logo_icon_url,omitempty"`
	FaviconURL           *string `json:"favicon_url,omitempty"`
	ThemePrimaryColor    *string `json:"theme_primary_color,omitempty"`
	ThemeBackgroundColor *string `json:"theme_background_color,omitempty"`
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
	Downloads []DownloadApp         `json:"downloads"`
	Header    LandingHeaderConfig   `json:"header"`
}

func NewLandingConfigService(
	variantService *VariantService,
	contentService *ContentService,
	planService *PlanService,
	downloadService *DownloadService,
	brandingService *BrandingService,
) *LandingConfigService {
	return &LandingConfigService{
		variantService:  variantService,
		contentService:  contentService,
		planService:     planService,
		downloadService: downloadService,
		brandingService: brandingService,
	}
}

func (s *LandingConfigService) GetLandingConfig(ctx context.Context, variantSlug string) (*LandingConfigResponse, error) {
	pricing, err := s.planService.GetPricingOverview()
	if err != nil {
		return s.fallbackWithReason("pricing_fetch_failed", err, nil)
	}

	downloads, err := s.downloadService.ListApps(s.planService.BundleKey())
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

	// Fetch branding (non-critical - don't fail on error)
	var branding *LandingBranding
	if s.brandingService != nil {
		if siteBranding, err := s.brandingService.Get(); err == nil && siteBranding != nil {
			branding = &LandingBranding{
				SiteName:             siteBranding.SiteName,
				Tagline:              siteBranding.Tagline,
				LogoURL:              siteBranding.LogoURL,
				LogoIconURL:          siteBranding.LogoIconURL,
				FaviconURL:           siteBranding.FaviconURL,
				ThemePrimaryColor:    siteBranding.ThemePrimaryColor,
				ThemeBackgroundColor: siteBranding.ThemeBackgroundColor,
			}
		}
	}

	response := &LandingConfigResponse{
		Variant: LandingVariantSummary{
			ID:          variant.ID,
			Slug:        variant.Slug,
			Name:        variant.Name,
			Description: variant.Description,
			Axes:        variant.Axes,
		},
		Header:    variant.HeaderConfig,
		Pricing:   pricing,
		Downloads: downloads,
		Branding:  branding,
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
	response := &LandingConfigResponse{
		Variant:   fallbackLanding.Variant,
		Sections:  fallbackLanding.Sections,
		Pricing:   &fallbackLanding.Pricing,
		Downloads: fallbackLanding.Downloads,
		Header:    fallbackLanding.Header,
		Fallback:  mark,
	}

	// Try to include branding even in fallback
	if s.brandingService != nil {
		if siteBranding, err := s.brandingService.Get(); err == nil && siteBranding != nil {
			response.Branding = &LandingBranding{
				SiteName:             siteBranding.SiteName,
				Tagline:              siteBranding.Tagline,
				LogoURL:              siteBranding.LogoURL,
				LogoIconURL:          siteBranding.LogoIconURL,
				FaviconURL:           siteBranding.FaviconURL,
				ThemePrimaryColor:    siteBranding.ThemePrimaryColor,
				ThemeBackgroundColor: siteBranding.ThemeBackgroundColor,
			}
		}
	}

	return response
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

	if len(raw.Pricing) == 0 {
		return LandingConfigPayload{}, fmt.Errorf("fallback config missing pricing")
	}
	var pricing PricingOverview
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(raw.Pricing, &pricing); err != nil {
		return LandingConfigPayload{}, fmt.Errorf("parse fallback pricing: %w", err)
	}

	sections := normalizeFallbackSections(raw.Sections)
	if len(sections) == 0 {
		return LandingConfigPayload{}, fmt.Errorf("fallback config has no usable sections")
	}

	downloadApps, err := parseFallbackDownloads(raw.Downloads)
	if err != nil {
		return LandingConfigPayload{}, fmt.Errorf("parse fallback downloads: %w", err)
	}

	payload := LandingConfigPayload{
		Variant:   raw.Variant,
		Sections:  sections,
		Pricing:   pricing,
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

	bundleKey := flatAssets[0].BundleKey
	appKey := flatAssets[0].AppKey
	if strings.TrimSpace(appKey) == "" {
		appKey = "bundle_downloads"
	}

	app := DownloadApp{
		BundleKey: bundleKey,
		AppKey:    appKey,
		Name:      "Bundle downloads",
		Tagline:   "Installer payload generated from fallback config",
		Platforms: flatAssets,
	}

	return []DownloadApp{app}, nil
}

func normalizeDownloads(downloads []DownloadApp) []DownloadApp {
	if downloads == nil {
		return []DownloadApp{}
	}
	copied := make([]DownloadApp, len(downloads))
	for i, download := range downloads {
		copied[i] = download
		if download.Platforms == nil {
			copied[i].Platforms = []DownloadAsset{}
		} else {
			platforms := make([]DownloadAsset, len(download.Platforms))
			copy(platforms, download.Platforms)
			copied[i].Platforms = platforms
		}
		if download.InstallSteps == nil {
			copied[i].InstallSteps = []string{}
		} else {
			steps := make([]string, len(download.InstallSteps))
			copy(steps, download.InstallSteps)
			copied[i].InstallSteps = steps
		}
		if download.Storefronts == nil {
			copied[i].Storefronts = []DownloadStorefront{}
		} else {
			storefronts := make([]DownloadStorefront, len(download.Storefronts))
			copy(storefronts, download.Storefronts)
			copied[i].Storefronts = storefronts
		}
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
