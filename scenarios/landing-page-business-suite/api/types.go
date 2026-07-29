package main

import (
	"encoding/json"
	"time"
)

// VariantSnapshot represents a complete variant configuration loaded from JSON files.
// This is the in-memory representation used by ConfigStore.
type VariantSnapshot struct {
	Variant  VariantSnapshotMeta `json:"variant"`
	Sections []VariantSection    `json:"sections"`
}

// VariantSnapshotMeta holds variant metadata (name, axes, SEO, etc.)
type VariantSnapshotMeta struct {
	Slug         string              `json:"slug"`
	Name         string              `json:"name"`
	Description  string              `json:"description,omitempty"`
	Weight       int                 `json:"weight,omitempty"` // Traffic weight (>0 participates, <=0 disabled)
	Status       string              `json:"status,omitempty"` // active | archived
	Axes         map[string]string   `json:"axes"`
	HeaderConfig LandingHeaderConfig `json:"header_config,omitempty"`
	SEOConfig    json.RawMessage     `json:"seo_config,omitempty"`
}

// VariantSnapshotMetaInput is the JSON file format for variant metadata
type VariantSnapshotMetaInput struct {
	Slug         string               `json:"slug"`
	Name         string               `json:"name"`
	Description  string               `json:"description,omitempty"`
	Weight       int                  `json:"weight,omitempty"` // Traffic weight (0 = disabled, default 50)
	Status       string               `json:"status,omitempty"`
	Axes         map[string]string    `json:"axes"`
	HeaderConfig *LandingHeaderConfig `json:"header_config,omitempty"`
	SEOConfig    json.RawMessage      `json:"seo_config,omitempty"`
}

// VariantSnapshotInput is the JSON file format for variant snapshots
type VariantSnapshotInput struct {
	Variant  VariantSnapshotMetaInput `json:"variant"`
	Sections []VariantSectionInput    `json:"sections"`
}

// VariantSection represents a content section within a variant
type VariantSection struct {
	SectionType string          `json:"section_type"`
	Content     json.RawMessage `json:"content"`
	Order       int             `json:"order"`
	Enabled     bool            `json:"enabled"`
}

// VariantSectionInput is the JSON file format for variant sections
type VariantSectionInput struct {
	SectionType string          `json:"section_type"`
	Content     json.RawMessage `json:"content"`
	Order       int             `json:"order,omitempty"`
	Enabled     *bool           `json:"enabled,omitempty"`
}

// SiteBranding represents the site-wide branding configuration
type SiteBranding struct {
	ID                     int64     `json:"id"`
	SiteName               string    `json:"site_name"`
	Tagline                *string   `json:"tagline,omitempty"`
	LogoURL                *string   `json:"logo_url,omitempty"`
	LogoIconURL            *string   `json:"logo_icon_url,omitempty"`
	FaviconURL             *string   `json:"favicon_url,omitempty"`
	AppleTouchIconURL      *string   `json:"apple_touch_icon_url,omitempty"`
	DefaultTitle           *string   `json:"default_title,omitempty"`
	DefaultDescription     *string   `json:"default_description,omitempty"`
	DefaultOGImageURL      *string   `json:"default_og_image_url,omitempty"`
	ThemePrimaryColor      *string   `json:"theme_primary_color,omitempty"`
	ThemeBackgroundColor   *string   `json:"theme_background_color,omitempty"`
	CanonicalBaseURL       *string   `json:"canonical_base_url,omitempty"`
	GoogleSiteVerification *string   `json:"google_site_verification,omitempty"`
	RobotsTxt              *string   `json:"robots_txt,omitempty"`
	SupportChatURL         *string   `json:"support_chat_url,omitempty"`
	SupportEmail           *string   `json:"support_email,omitempty"`
	SMTPHost               *string   `json:"smtp_host,omitempty"`
	SMTPPort               *int      `json:"smtp_port,omitempty"`
	SMTPUsername           *string   `json:"smtp_username,omitempty"`
	SMTPPassword           *string   `json:"smtp_password,omitempty"`
	SMTPFrom               *string   `json:"smtp_from,omitempty"`
	ComingSoonEnabled      *bool     `json:"coming_soon_enabled,omitempty"`
	ComingSoonMessage      *string   `json:"coming_soon_message,omitempty"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

// BrandingUpdateRequest represents a partial update to branding configuration
type BrandingUpdateRequest struct {
	SiteName               *string `json:"site_name,omitempty"`
	Tagline                *string `json:"tagline,omitempty"`
	LogoURL                *string `json:"logo_url,omitempty"`
	LogoIconURL            *string `json:"logo_icon_url,omitempty"`
	FaviconURL             *string `json:"favicon_url,omitempty"`
	AppleTouchIconURL      *string `json:"apple_touch_icon_url,omitempty"`
	DefaultTitle           *string `json:"default_title,omitempty"`
	DefaultDescription     *string `json:"default_description,omitempty"`
	DefaultOGImageURL      *string `json:"default_og_image_url,omitempty"`
	ThemePrimaryColor      *string `json:"theme_primary_color,omitempty"`
	ThemeBackgroundColor   *string `json:"theme_background_color,omitempty"`
	CanonicalBaseURL       *string `json:"canonical_base_url,omitempty"`
	GoogleSiteVerification *string `json:"google_site_verification,omitempty"`
	RobotsTxt              *string `json:"robots_txt,omitempty"`
	SupportChatURL         *string `json:"support_chat_url,omitempty"`
	SupportEmail           *string `json:"support_email,omitempty"`
	SMTPHost               *string `json:"smtp_host,omitempty"`
	SMTPPort               *int    `json:"smtp_port,omitempty"`
	SMTPUsername           *string `json:"smtp_username,omitempty"`
	SMTPPassword           *string `json:"smtp_password,omitempty"`
	SMTPFrom               *string `json:"smtp_from,omitempty"`
	ComingSoonEnabled      *bool   `json:"coming_soon_enabled,omitempty"`
	ComingSoonMessage      *string `json:"coming_soon_message,omitempty"`
}
