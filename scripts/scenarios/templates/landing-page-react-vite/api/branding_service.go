package main

import (
	"database/sql"
	"time"
)

// SiteBranding represents site-wide branding configuration
type SiteBranding struct {
	ID                     int        `json:"id"`
	SiteName               string     `json:"site_name"`
	Tagline                *string    `json:"tagline,omitempty"`
	LogoURL                *string    `json:"logo_url,omitempty"`
	LogoIconURL            *string    `json:"logo_icon_url,omitempty"`
	FaviconURL             *string    `json:"favicon_url,omitempty"`
	AppleTouchIconURL      *string    `json:"apple_touch_icon_url,omitempty"`
	DefaultTitle           *string    `json:"default_title,omitempty"`
	DefaultDescription     *string    `json:"default_description,omitempty"`
	DefaultOGImageURL      *string    `json:"default_og_image_url,omitempty"`
	ThemePrimaryColor      *string    `json:"theme_primary_color,omitempty"`
	ThemeBackgroundColor   *string    `json:"theme_background_color,omitempty"`
	CanonicalBaseURL       *string    `json:"canonical_base_url,omitempty"`
	GoogleSiteVerification *string    `json:"google_site_verification,omitempty"`
	RobotsTxt              *string    `json:"robots_txt,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

// BrandingService handles site-wide branding operations
type BrandingService struct {
	db *sql.DB
}

// NewBrandingService creates a new branding service
func NewBrandingService(db *sql.DB) *BrandingService {
	return &BrandingService{db: db}
}

// Get retrieves the site branding (singleton row)
func (s *BrandingService) Get() (*SiteBranding, error) {
	query := `
		SELECT id, site_name, tagline, logo_url, logo_icon_url, favicon_url,
		       apple_touch_icon_url, default_title, default_description,
		       default_og_image_url, theme_primary_color, theme_background_color,
		       canonical_base_url, google_site_verification, robots_txt,
		       created_at, updated_at
		FROM site_branding
		WHERE id = 1
	`

	var b SiteBranding
	err := s.db.QueryRow(query).Scan(
		&b.ID, &b.SiteName, &b.Tagline, &b.LogoURL, &b.LogoIconURL, &b.FaviconURL,
		&b.AppleTouchIconURL, &b.DefaultTitle, &b.DefaultDescription,
		&b.DefaultOGImageURL, &b.ThemePrimaryColor, &b.ThemeBackgroundColor,
		&b.CanonicalBaseURL, &b.GoogleSiteVerification, &b.RobotsTxt,
		&b.CreatedAt, &b.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		// Return default branding if row doesn't exist
		return &SiteBranding{
			ID:       1,
			SiteName: "My Landing",
		}, nil
	}
	if err != nil {
		return nil, err
	}

	return &b, nil
}

// BrandingUpdateRequest represents fields that can be updated
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
}

// Update updates the site branding
func (s *BrandingService) Update(req *BrandingUpdateRequest) (*SiteBranding, error) {
	query := `
		UPDATE site_branding SET
			site_name = COALESCE($1, site_name),
			tagline = COALESCE($2, tagline),
			logo_url = COALESCE($3, logo_url),
			logo_icon_url = COALESCE($4, logo_icon_url),
			favicon_url = COALESCE($5, favicon_url),
			apple_touch_icon_url = COALESCE($6, apple_touch_icon_url),
			default_title = COALESCE($7, default_title),
			default_description = COALESCE($8, default_description),
			default_og_image_url = COALESCE($9, default_og_image_url),
			theme_primary_color = COALESCE($10, theme_primary_color),
			theme_background_color = COALESCE($11, theme_background_color),
			canonical_base_url = COALESCE($12, canonical_base_url),
			google_site_verification = COALESCE($13, google_site_verification),
			robots_txt = COALESCE($14, robots_txt),
			updated_at = NOW()
		WHERE id = 1
		RETURNING id, site_name, tagline, logo_url, logo_icon_url, favicon_url,
		          apple_touch_icon_url, default_title, default_description,
		          default_og_image_url, theme_primary_color, theme_background_color,
		          canonical_base_url, google_site_verification, robots_txt,
		          created_at, updated_at
	`

	var b SiteBranding
	err := s.db.QueryRow(query,
		req.SiteName, req.Tagline, req.LogoURL, req.LogoIconURL,
		req.FaviconURL, req.AppleTouchIconURL, req.DefaultTitle,
		req.DefaultDescription, req.DefaultOGImageURL, req.ThemePrimaryColor,
		req.ThemeBackgroundColor, req.CanonicalBaseURL,
		req.GoogleSiteVerification, req.RobotsTxt,
	).Scan(
		&b.ID, &b.SiteName, &b.Tagline, &b.LogoURL, &b.LogoIconURL, &b.FaviconURL,
		&b.AppleTouchIconURL, &b.DefaultTitle, &b.DefaultDescription,
		&b.DefaultOGImageURL, &b.ThemePrimaryColor, &b.ThemeBackgroundColor,
		&b.CanonicalBaseURL, &b.GoogleSiteVerification, &b.RobotsTxt,
		&b.CreatedAt, &b.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &b, nil
}

// ClearField sets a specific branding field to NULL
func (s *BrandingService) ClearField(field string) error {
	allowedFields := map[string]bool{
		"tagline": true, "logo_url": true, "logo_icon_url": true,
		"favicon_url": true, "apple_touch_icon_url": true,
		"default_title": true, "default_description": true,
		"default_og_image_url": true, "theme_primary_color": true,
		"theme_background_color": true, "canonical_base_url": true,
		"google_site_verification": true, "robots_txt": true,
	}

	if !allowedFields[field] {
		return nil
	}

	query := "UPDATE site_branding SET " + field + " = NULL, updated_at = NOW() WHERE id = 1"
	_, err := s.db.Exec(query)
	return err
}
