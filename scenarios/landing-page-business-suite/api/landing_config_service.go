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
	"google.golang.org/protobuf/proto"
)

var (
	fallbackLanding            LandingConfigPayload
	defaultFallbackLandingJSON = []byte(`{
		"variant": {
			"id": 0,
			"slug": "control",
			"name": "Silent Founder OS",
			"description": "Offline-safe fallback for the Silent Founder OS landing page."
		},
		"axes": {
			"persona": "silentFounder",
			"jtbd": "entrepreneurship",
			"conversionStyle": "emotional"
		},
		"sections": [
			{
				"section_type": "hero",
				"order": 1,
				"enabled": true,
				"content": {
					"title": "Record once. Automate forever",
					"subtitle": "And turn every recording into a polished, professional demo video",
					"cta_text": "Start free",
					"cta_url": "/checkout?plan=pro",
					"secondary_cta_text": "Watch video",
					"secondary_cta_url": "#video-2",
					"image_url": "/assets/fallback/hero.png"
				}
			},
			{
				"section_type": "video",
				"order": 2,
				"enabled": true,
				"content": {
					"title": "Watch Vrooli Ascension build and replay a flow",
					"videoUrl": "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
					"thumbnailUrl": "/assets/fallback/video-thumb.png",
					"caption": "Visual workflow builder \u2192 e2e test \u2192 replay-as-movie export. Available today; Silent Founder OS keeps adding tools."
				}
			},
			{
				"section_type": "features",
				"order": 3,
				"enabled": true,
				"content": {
					"title": "Vrooli Ascension is live now. The suite keeps growing.",
					"subtitle": "Automate browsers, ship tests, and generate proof-ready replays today. Your subscription includes future Vrooli Business Suite apps at no extra cost.",
					"features": [
						{
							"title": "Visual workflow builder",
							"description": "Point-and-record or assemble actions to automate admin panels, dashboards, and back-office flows.",
							"icon": "zap"
						},
						{
							"title": "Workflows = e2e tests",
							"description": "Run flows in CI, capture failures, and share evidence without rewriting tests.",
							"icon": "shield"
						},
						{
							"title": "Replay-as-movie exports",
							"description": "Fake browser frames, smooth cursor animation, zoom/pan highlights, MP4 export with watermark rules by plan.",
							"icon": "sparkles"
						},
						{
							"title": "Future UX metrics layer",
							"description": "Coming soon: friction scores, duration, and spatial patterns generated from your workflows.",
							"icon": "layers"
						},
						{
							"title": "Agent loops on deck",
							"description": "Next up: ecosystem-manager + PRD control tower so agents can improve flows and enforce requirements.",
							"icon": "target"
						}
					]
				}
			},
			{
				"section_type": "pricing",
				"order": 4,
				"enabled": true,
				"content": {
					"title": "Simple, transparent pricing",
					"subtitle": "Vrooli Ascension today. More Silent Founder OS tools added over time.",
					"tiers": [
						{
							"name": "Free",
							"price": "$0",
							"description": "50 runs/month, builder, replay viewer (watermarked MP4)",
							"features": [
								"Visual workflow builder",
								"Replay viewer with watermark",
								"50 runs/month",
								"No agents or UX metrics"
							],
							"cta_text": "Start free",
							"cta_url": "/checkout?plan=free",
							"badge": "Try it now"
						},
						{
							"name": "Solo",
							"price": "$29",
							"description": "200 runs/month, MP4 export with watermark",
							"features": [
								"200 runs/month",
								"MP4 export (watermark)",
								"Workflow builder + replays",
								"Email support"
							],
							"cta_text": "Upgrade to Solo",
							"cta_url": "/checkout?plan=solo"
						},
						{
							"name": "Pro",
							"price": "$79",
							"description": "Unlimited runs, MP4 without watermark, CI hooks",
							"features": [
								"Unlimited runs (fair use)",
								"MP4 exports without watermark",
								"CI integrations + advanced workflow tooling",
								"Early UX metrics access",
								"Limited agent loops"
							],
							"cta_text": "Default for Silent Founders",
							"cta_url": "/checkout?plan=pro",
							"highlighted": true,
							"badge": "Recommended"
						},
						{
							"name": "Studio",
							"price": "$199",
							"description": "Agency-ready replays + branding, more agent loops",
							"features": [
								"Multi-seat studio",
								"Custom branding in replays",
								"More agent loop concurrency",
								"Priority support"
							],
							"cta_text": "Choose Studio",
							"cta_url": "/checkout?plan=studio"
						},
						{
							"name": "Business",
							"price": "$499",
							"description": "For small teams with heavy automation + API needs",
							"features": [
								"Unlimited agent loops",
								"API + webhooks",
								"Reliability & SSO mode prep",
								"Best for teams/clients"
							],
							"cta_text": "Talk async",
							"cta_url": "/contact"
						}
					]
				}
			},
			{
				"section_type": "faq",
				"order": 5,
				"enabled": true,
				"content": {
					"title": "Answers for quiet founders",
					"subtitle": "What ships today, what is coming, and how we price it.",
					"faqs": [
						{
							"question": "What do I get today?",
							"answer": "Vrooli Ascension: visual workflow builder, CI-friendly tests, replay viewer, and MP4 exports (watermark rules by plan)."
						},
						{
							"question": "What is coming next?",
							"answer": "UX metrics layer (friction, duration, spatial paths) and agent loops that fix flows via ecosystem-manager + PRD control tower."
						},
						{
							"question": "Do I have to talk to sales?",
							"answer": "No. No sales calls, no per-seat pricing. Subscribe, download, and grow quietly. Support is async."
						},
						{
							"question": "Can I cancel or switch?",
							"answer": "Yes. Plans are flat, cancellable, and your price is honored as the suite expands."
						}
					]
				}
			},
			{
				"section_type": "cta",
				"order": 6,
				"enabled": true,
				"content": {
					"title": "See Vrooli Ascension in action",
					"subtitle": "Start free, export a replay, and know more tools are coming to the same subscription.",
					"cta_text": "Get started quietly",
					"cta_url": "/checkout?plan=pro"
				}
			},
			{
				"section_type": "downloads",
				"order": 7,
				"enabled": true,
				"content": {
					"title": "Download Vrooli Ascension",
					"subtitle": "Install now and start automating today."
				}
			},
			{
				"section_type": "footer",
				"order": 8,
				"enabled": true,
				"content": {
					"company_name": "Vrooli Business Suite · Silent Founder OS",
					"tagline": "Vrooli Ascension today. Agents and new tools tomorrow. No meetings required.",
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
				"name": "Vrooli Business Suite (Silent Founder OS)",
				"stripe_product_id": "prod_business_suite",
				"credits_per_usd": 1000000,
				"display_credits_multiplier": 0.001,
				"display_credits_label": "credits",
				"environment": "production"
			},
			"monthly": [
				{
					"plan_name": "Free Monthly",
					"plan_tier": "free",
					"billing_interval": "month",
					"amount_cents": 0,
					"currency": "usd",
					"intro_enabled": false,
					"stripe_price_id": "price_free_monthly",
					"monthly_included_credits": 50,
					"one_time_bonus_credits": 0,
					"plan_rank": 0,
					"bonus_type": "none",
					"display_weight": 5,
					"metadata": {
						"features": [
							"50 runs/month",
							"Replay viewer (watermark)",
							"Builder access"
						],
						"badge": "Start free"
					}
				},
				{
					"plan_name": "Solo Monthly",
					"plan_tier": "solo",
					"billing_interval": "month",
					"amount_cents": 2900,
					"currency": "usd",
					"intro_enabled": false,
					"stripe_price_id": "price_solo_monthly",
					"monthly_included_credits": 200,
					"one_time_bonus_credits": 0,
					"plan_rank": 1,
					"bonus_type": "none",
					"display_weight": 20,
					"metadata": {
						"features": [
							"200 runs/month",
							"MP4 export (watermark)",
							"Async support"
						],
						"cta_label": "Upgrade to Solo"
					}
				},
				{
					"plan_name": "Pro Monthly",
					"plan_tier": "pro",
					"billing_interval": "month",
					"amount_cents": 7900,
					"currency": "usd",
					"intro_enabled": false,
					"stripe_price_id": "price_pro_monthly",
					"monthly_included_credits": 1000000,
					"one_time_bonus_credits": 0,
					"plan_rank": 2,
					"bonus_type": "none",
					"display_weight": 40,
					"metadata": {
						"features": [
							"Unlimited runs (fair use)",
							"MP4 without watermark",
							"CI hooks + advanced workflows",
							"Limited agent loops",
							"Early UX metrics access"
						],
						"badge": "Recommended",
						"highlight": true,
						"cta_label": "Choose Pro"
					}
				},
				{
					"plan_name": "Studio Monthly",
					"plan_tier": "studio",
					"billing_interval": "month",
					"amount_cents": 19900,
					"currency": "usd",
					"intro_enabled": false,
					"stripe_price_id": "price_studio_monthly",
					"monthly_included_credits": 2000000,
					"one_time_bonus_credits": 0,
					"plan_rank": 3,
					"bonus_type": "none",
					"display_weight": 25,
					"metadata": {
						"features": [
							"Custom branding in replays",
							"More agent loop concurrency",
							"Multi-seat studio",
							"Priority support"
						],
						"cta_label": "Choose Studio"
					}
				},
				{
					"plan_name": "Business Monthly",
					"plan_tier": "business",
					"billing_interval": "month",
					"amount_cents": 49900,
					"currency": "usd",
					"intro_enabled": false,
					"stripe_price_id": "price_business_monthly",
					"monthly_included_credits": 4000000,
					"one_time_bonus_credits": 0,
					"plan_rank": 4,
					"bonus_type": "none",
					"display_weight": 10,
					"metadata": {
						"features": [
							"Unlimited agent loops",
							"API + webhooks",
							"Reliability options"
						],
						"cta_label": "Talk async"
					}
				}
			],
			"yearly": [
				{
					"plan_name": "Solo Yearly",
					"plan_tier": "solo",
					"billing_interval": "year",
					"amount_cents": 29000,
					"currency": "usd",
					"intro_enabled": false,
					"stripe_price_id": "price_solo_yearly",
					"monthly_included_credits": 200,
					"one_time_bonus_credits": 0,
					"plan_rank": 1,
					"bonus_type": "yearly_bonus",
					"display_weight": 10,
					"metadata": {
						"features": [
							"2 months free equivalent",
							"MP4 export (watermark)"
						]
					}
				},
				{
					"plan_name": "Pro Yearly",
					"plan_tier": "pro",
					"billing_interval": "year",
					"amount_cents": 79000,
					"currency": "usd",
					"intro_enabled": false,
					"stripe_price_id": "price_pro_yearly",
					"monthly_included_credits": 1000000,
					"one_time_bonus_credits": 0,
					"plan_rank": 2,
					"bonus_type": "yearly_bonus",
					"display_weight": 20,
					"metadata": {
						"features": [
							"MP4 without watermark",
							"CI hooks + advanced workflows",
							"Limited agent loops"
						]
					}
				},
				{
					"plan_name": "Studio Yearly",
					"plan_tier": "studio",
					"billing_interval": "year",
					"amount_cents": 199000,
					"currency": "usd",
					"intro_enabled": false,
					"stripe_price_id": "price_studio_yearly",
					"monthly_included_credits": 2000000,
					"one_time_bonus_credits": 0,
					"plan_rank": 3,
					"bonus_type": "yearly_bonus",
					"display_weight": 30,
					"metadata": {
						"features": [
							"Custom branding in replays",
							"More agent loop concurrency",
							"Multi-seat studio"
						]
					}
				},
				{
					"plan_name": "Business Yearly",
					"plan_tier": "business",
					"billing_interval": "year",
					"amount_cents": 499000,
					"currency": "usd",
					"intro_enabled": false,
					"stripe_price_id": "price_business_yearly",
					"monthly_included_credits": 4000000,
					"one_time_bonus_credits": 0,
					"plan_rank": 4,
					"bonus_type": "yearly_bonus",
					"display_weight": 5,
					"metadata": {
						"features": [
							"Unlimited agent loops",
							"API + webhooks",
							"Reliability + SSO prep"
						]
					}
				}
			],
			"updated_at": "2025-01-01T00:00:00Z"
		},
		"downloads": [
			{
				"bundle_key": "business_suite",
				"app_key": "automation_studio",
				"name": "Vrooli Ascension",
				"tagline": "Silent Founder OS \u00b7 Day-one value",
				"description": "Desktop suite for visual browser automation, tests, and cinematic replays.",
				"install_overview": "Pick your OS, download the installer, sign in with the email tied to your plan to unlock entitlement-gated downloads.",
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
						"release_notes": "Vrooli Ascension GA with replay exports.",
						"requires_entitlement": false,
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
						"requires_entitlement": false,
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
						"requires_entitlement": false,
						"metadata": {
							"size_mb": 205
						}
					}
				]
			}
		]
	}`)
)

type fallbackProvider func() LandingConfigPayload

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
	variantService   *VariantService
	contentService   *ContentService
	planService      *PlanService
	downloadService  *DownloadService
	brandingService  *BrandingService
	fallbackProvider fallbackProvider
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
		variantService:   variantService,
		contentService:   contentService,
		planService:      planService,
		downloadService:  downloadService,
		brandingService:  brandingService,
		fallbackProvider: defaultFallbackProvider,
	}
}

// UseFallbackProvider overrides the source of fallback content (primarily for tests).
func (s *LandingConfigService) UseFallbackProvider(provider fallbackProvider) {
	s.fallbackProvider = provider
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
	payload := s.fallbackPayload()

	// fallbackLanding already contains pricing/download placeholders.
	response := &LandingConfigResponse{
		Variant:   payload.Variant,
		Sections:  payload.Sections,
		Pricing:   &payload.Pricing,
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

func (s *LandingConfigService) fallbackPayload() LandingConfigPayload {
	provider := s.fallbackProvider
	if provider == nil {
		provider = defaultFallbackProvider
	}
	return cloneLandingPayload(provider())
}

func defaultFallbackProvider() LandingConfigPayload {
	return fallbackLanding
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

func cloneLandingPayload(payload LandingConfigPayload) LandingConfigPayload {
	cloned := LandingConfigPayload{
		Variant: LandingVariantSummary{
			ID:          payload.Variant.ID,
			Slug:        payload.Variant.Slug,
			Name:        payload.Variant.Name,
			Description: payload.Variant.Description,
			Axes:        cloneStringMap(payload.Variant.Axes),
		},
		Sections:  cloneLandingSections(payload.Sections),
		Downloads: cloneDownloads(payload.Downloads),
		Header:    cloneHeaderConfig(payload.Header, payload.Variant.Name),
	}

	if pricing := clonePricing(payload.Pricing); pricing != nil {
		cloned.Pricing = *pricing
	}

	return cloned
}

func cloneLandingSections(sections []LandingSection) []LandingSection {
	if len(sections) == 0 {
		return []LandingSection{}
	}

	cloned := make([]LandingSection, len(sections))
	for i, section := range sections {
		cloned[i] = LandingSection{
			SectionType: section.SectionType,
			Content:     cloneContentMap(section.Content),
			Order:       section.Order,
			Enabled:     section.Enabled,
		}
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
		appCopy := DownloadApp{
			ID:              app.ID,
			BundleKey:       app.BundleKey,
			AppKey:          app.AppKey,
			Name:            app.Name,
			Tagline:         app.Tagline,
			Description:     app.Description,
			InstallOverview: app.InstallOverview,
			InstallSteps:    append([]string{}, app.InstallSteps...),
			Storefronts:     cloneStorefronts(app.Storefronts),
			Metadata:        cloneContentMap(app.Metadata),
			DisplayOrder:    app.DisplayOrder,
			Platforms:       cloneDownloadAssets(app.Platforms),
		}
		cloned = append(cloned, appCopy)
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
		copied[i] = DownloadAsset{
			ID:                  asset.ID,
			BundleKey:           asset.BundleKey,
			AppKey:              asset.AppKey,
			Platform:            asset.Platform,
			ArtifactURL:         asset.ArtifactURL,
			ReleaseVersion:      asset.ReleaseVersion,
			ReleaseNotes:        asset.ReleaseNotes,
			Checksum:            asset.Checksum,
			RequiresEntitlement: asset.RequiresEntitlement,
			Metadata:            cloneContentMap(asset.Metadata),
		}
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

func clonePricing(pricing PricingOverview) *PricingOverview {
	cloned := proto.Clone(&pricing)
	if cloned == nil {
		return nil
	}
	return cloned.(*PricingOverview)
}
