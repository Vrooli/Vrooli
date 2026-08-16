// DOC: docs/reference/api/landing.md - Public landing page configuration API
// DOC: docs/concepts/CONCEPTS.md#data-flow-architecture - Data flow overview
// DOC: PRD.md#OT-P0-031 - API-driven landing config + fallback requirement
package landing

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"
	"sort"

	"landing-page-business-suite-api/internal/commerce"
	"landing-page-business-suite-api/internal/delivery"
	"landing-page-business-suite-api/internal/experimentation"
	"landing-page-business-suite-api/internal/logx"
)

var (
	fallbackLanding            *LandingConfigPayload
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
							"description": "Next up: swarm-manager + PRD control tower so agents can improve flows and enforce requirements.",
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
							"answer": "UX metrics layer (friction, duration, spatial paths) and agent loops that fix flows via swarm-manager + PRD control tower."
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
				"app_key": "browser-automation-studio",
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
						"app_key": "browser-automation-studio",
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
						"app_key": "browser-automation-studio",
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
						"app_key": "browser-automation-studio",
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

type fallbackProvider func() *LandingConfigPayload

func init() {
	_, sourcePath, _, ok := runtime.Caller(0)
	if !ok {
		panic("resolve landing config source path")
	}
	scenarioRoot := filepath.Clean(filepath.Join(filepath.Dir(sourcePath), "..", "..", ".."))
	primaryPath := filepath.Join(scenarioRoot, ".vrooli", "fallback", "fallback.json")
	legacyPath := filepath.Join(scenarioRoot, ".vrooli", "variants", "fallback.json")
	path := primaryPath
	payload, err := loadFallbackLandingFromFile(path)
	if err != nil {
		path = legacyPath
		payload, err = loadFallbackLandingFromFile(path)
	}
	if err != nil {
		logx.Printf("failed to read fallback config at %s: %v; using baked defaults", path, err)
		payload, err = parseFallbackLandingConfig(defaultFallbackLandingJSON)
		if err != nil {
			panic(fmt.Sprintf("default fallback config invalid: %v", err))
		}
	}
	fallbackLanding = payload
}

// LandingConfigService aggregates variant, section, pricing, and download data.
// NOTE: Variants and branding are now stored in JSON files and accessed via ConfigStore.
// The deprecated DB-backed services have been removed.
type LandingConfigService struct {
	planService      *commerce.PlanService
	downloadService  *delivery.CatalogService
	configStore      *experimentation.ConfigStore
	introOfferLookup IntroOfferLookup
	fallbackProvider fallbackProvider
	eventLogger      EventLogger
}

// LandingConfigResponse is returned by LandingConfigService.GetLandingConfig.
type LandingConfigResponse struct {
	Variant        LandingVariantSummary               `json:"variant"`
	Sections       []LandingSection                    `json:"sections"`
	Pricing        *commerce.PricingOverview           `json:"pricing"`
	Downloads      []delivery.App                      `json:"downloads"`
	Header         experimentation.LandingHeaderConfig `json:"header"`
	Branding       *LandingBranding                    `json:"branding,omitempty"`
	CouponMappings map[string]string                   `json:"coupon_mappings,omitempty"`
	IntroOffers    []IntroOffer                        `json:"intro_offers,omitempty"`
	Fallback       bool                                `json:"fallback"`
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
	SupportChatURL       *string `json:"support_chat_url,omitempty"`
	SupportEmail         *string `json:"support_email,omitempty"`
	ComingSoonEnabled    *bool   `json:"coming_soon_enabled,omitempty"`
	ComingSoonMessage    *string `json:"coming_soon_message,omitempty"`
}

type LandingVariantSummary struct {
	ID          int               `json:"id,omitempty"`
	Slug        string            `json:"slug"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Axes        map[string]string `json:"axes,omitempty"`
}

type LandingSection struct {
	Key         string                 `json:"key,omitempty"`
	SectionType string                 `json:"section_type"`
	Content     map[string]interface{} `json:"content"`
	Order       int                    `json:"order"`
	Enabled     bool                   `json:"enabled"`
}

type LandingConfigPayload struct {
	Variant   LandingVariantSummary               `json:"variant"`
	Sections  []LandingSection                    `json:"sections"`
	Pricing   *commerce.PricingOverview           `json:"pricing"`
	Downloads []delivery.App                      `json:"downloads"`
	Header    experimentation.LandingHeaderConfig `json:"header"`
}

// IntroOffer is the public, display-safe coupon projection used by the landing
// configuration domain. Payment-provider details remain in commerce wiring.
type IntroOffer struct {
	ID               string
	Name             *string
	AmountOff        int64
	PercentOff       float64
	Currency         *string
	Duration         string
	DurationInMonths *int
	MaxRedemptions   *int
	RedeemBy         int64
	TimesRedeemed    int
	Valid            bool
	Created          int64
	IsIntroCoupon    bool
	IntroTier        *string
}

type IntroOfferLookup func(context.Context, string) (*IntroOffer, error)

// EventLogger records domain events without coupling content policy to the
// application's logging implementation.
type EventLogger func(message string, fields map[string]interface{})

// NewLandingConfigServiceWithConfigStore creates a LandingConfigService using ConfigStore (JSON files as source of truth)
func NewLandingConfigServiceWithConfigStore(
	configStore *experimentation.ConfigStore,
	planService *commerce.PlanService,
	downloadService *delivery.CatalogService,
	introOfferLookup IntroOfferLookup,
) *LandingConfigService {
	return &LandingConfigService{
		configStore:      configStore,
		planService:      planService,
		downloadService:  downloadService,
		introOfferLookup: introOfferLookup,
		fallbackProvider: defaultFallbackProvider,
		eventLogger: func(message string, fields map[string]interface{}) {
			logx.Printf("%s: %+v", message, fields)
		},
	}
}

// UseFallbackProvider overrides the source of fallback content (primarily for tests).
func (s *LandingConfigService) UseFallbackProvider(provider fallbackProvider) {
	s.fallbackProvider = provider
}

// UseEventLogger overrides fallback-event reporting. Application composition
// supplies structured logging; tests may inject a deterministic observer.
func (s *LandingConfigService) UseEventLogger(logger EventLogger) {
	s.eventLogger = logger
}

func (s *LandingConfigService) GetLandingConfig(ctx context.Context, variantSlug string) (*LandingConfigResponse, error) {
	// Use ConfigStore (JSON files as source of truth)
	return s.getLandingConfigFromConfigStore(ctx, variantSlug)
}

// getLandingConfigFromConfigStore uses ConfigStore to fetch landing config
func (s *LandingConfigService) getLandingConfigFromConfigStore(ctx context.Context, variantSlug string) (*LandingConfigResponse, error) {
	pricing, err := s.planService.GetPricingOverview()
	if err != nil {
		return s.fallbackWithReason("pricing_fetch_failed", err, nil)
	}

	downloads, err := s.downloadService.ListAppsContext(ctx, s.planService.BundleKey())
	if err != nil {
		return s.fallbackWithReason("download_list_failed", err, nil)
	}

	var variantSnapshot *experimentation.VariantSnapshot
	if variantSlug != "" {
		variantSnapshot, err = s.configStore.GetVariant(variantSlug)
	} else {
		// Use weighted random selection for A/B testing
		variants := s.configStore.ListVariants()
		if len(variants) > 0 {
			variantSnapshot = experimentation.SelectWeightedRandomVariant(variants)
		} else {
			err = fmt.Errorf("no variants available")
		}
	}
	if err != nil || variantSnapshot == nil {
		reason := "variant_selection_failed"
		meta := map[string]interface{}{}
		if variantSlug != "" {
			reason = "variant_lookup_failed"
			meta["variant_slug"] = variantSlug
		}
		return s.fallbackWithReason(reason, err, meta)
	}

	// Fetch branding from ConfigStore
	var branding *LandingBranding
	siteBranding := s.configStore.GetBranding()
	if siteBranding != nil {
		branding = &LandingBranding{
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

	// Fetch coupon mappings and resolve intro offers for public display
	couponMappings := s.planService.GetCouponMappings()
	var introOffers []IntroOffer
	if len(couponMappings) > 0 && s.introOfferLookup != nil {
		seen := make(map[string]bool)
		for _, couponID := range couponMappings {
			if !seen[couponID] {
				seen[couponID] = true
				if coupon, err := s.introOfferLookup(ctx, couponID); err == nil && coupon != nil {
					introOffers = append(introOffers, *coupon)
				}
			}
		}
	}

	response := &LandingConfigResponse{
		Variant: LandingVariantSummary{
			Slug:        variantSnapshot.Variant.Slug,
			Name:        variantSnapshot.Variant.Name,
			Description: variantSnapshot.Variant.Description,
			Axes:        variantSnapshot.Variant.Axes,
		},
		Header:         variantSnapshot.Variant.HeaderConfig,
		Pricing:        pricing,
		Downloads:      downloads,
		Branding:       branding,
		CouponMappings: couponMappings,
		IntroOffers:    introOffers,
		Fallback:       false,
	}

	// Convert sections from VariantSnapshot format to LandingSection
	landingSections := make([]LandingSection, 0, len(variantSnapshot.Sections))
	for _, section := range variantSnapshot.Sections {
		if section.Enabled {
			var content map[string]interface{}
			if len(section.Content) > 0 {
				if err := json.Unmarshal(section.Content, &content); err != nil {
					logx.Printf("landing config section content unmarshal failed: variant=%s section=%s error=%v", variantSnapshot.Variant.Slug, section.SectionType, err)
					content = make(map[string]interface{})
				}
			} else {
				content = make(map[string]interface{})
			}
			landingSections = append(landingSections, LandingSection{
				Key:         section.Key,
				SectionType: section.SectionType,
				Content:     content,
				Order:       section.Order,
				Enabled:     section.Enabled,
			})
		}
	}
	sort.SliceStable(landingSections, func(i, j int) bool {
		return landingSections[i].Order < landingSections[j].Order
	})

	// ASSUMPTION: Every active variant must render at least one section and expose a hero.
	if err := ensureRenderableSections(landingSections); err != nil {
		return s.fallbackWithReason("section_renderability_failed", err, map[string]interface{}{
			"variant_slug": variantSnapshot.Variant.Slug,
		})
	}

	response.Sections = landingSections

	return response, nil
}
