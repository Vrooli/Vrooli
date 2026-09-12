// Package branding is the application layer for the singleton site-branding
// record: read, partial (COALESCE) update, and single-field clear. The Connect
// handler in handlers/branding is a thin proto<->domain adapter over this
// Service; all SQL and business rules live here.
package branding

import (
	"context"
	"database/sql"
	"time"
)

// SiteBranding is the singleton branding record. Nullable columns are modeled
// as *string so "not configured" is distinct from empty string.
type SiteBranding struct {
	ID                     int
	SiteName               string
	Tagline                *string
	LogoURL                *string
	LogoIconURL            *string
	FaviconURL             *string
	AppleTouchIconURL      *string
	DefaultTitle           *string
	DefaultDescription     *string
	DefaultOGImageURL      *string
	ThemePrimaryColor      *string
	ThemeBackgroundColor   *string
	CanonicalBaseURL       *string
	GoogleSiteVerification *string
	RobotsTxt              *string
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

// UpdateRequest carries a partial branding update. A nil field is preserved
// (COALESCE); a non-nil field overwrites.
type UpdateRequest struct {
	SiteName               *string
	Tagline                *string
	LogoURL                *string
	LogoIconURL            *string
	FaviconURL             *string
	AppleTouchIconURL      *string
	DefaultTitle           *string
	DefaultDescription     *string
	DefaultOGImageURL      *string
	ThemePrimaryColor      *string
	ThemeBackgroundColor   *string
	CanonicalBaseURL       *string
	GoogleSiteVerification *string
	RobotsTxt              *string
}

// clearableFields is the whitelist of columns ClearField may null. site_name
// is intentionally excluded (NOT NULL); unknown fields are a silent no-op.
// Interpolating the field name into SQL is safe only because of this whitelist.
var clearableFields = map[string]bool{
	"tagline": true, "logo_url": true, "logo_icon_url": true,
	"favicon_url": true, "apple_touch_icon_url": true,
	"default_title": true, "default_description": true,
	"default_og_image_url": true, "theme_primary_color": true,
	"theme_background_color": true, "canonical_base_url": true,
	"google_site_verification": true, "robots_txt": true,
}

const brandingColumns = `id, site_name, tagline, logo_url, logo_icon_url, favicon_url,
	apple_touch_icon_url, default_title, default_description,
	default_og_image_url, theme_primary_color, theme_background_color,
	canonical_base_url, google_site_verification, robots_txt,
	created_at, updated_at`

// Service owns branding reads and writes.
type Service struct {
	db *sql.DB
}

// NewService constructs the branding Service.
func NewService(db *sql.DB) *Service { return &Service{db: db} }

func scanBranding(row interface{ Scan(...any) error }) (*SiteBranding, error) {
	var b SiteBranding
	err := row.Scan(
		&b.ID, &b.SiteName, &b.Tagline, &b.LogoURL, &b.LogoIconURL, &b.FaviconURL,
		&b.AppleTouchIconURL, &b.DefaultTitle, &b.DefaultDescription,
		&b.DefaultOGImageURL, &b.ThemePrimaryColor, &b.ThemeBackgroundColor,
		&b.CanonicalBaseURL, &b.GoogleSiteVerification, &b.RobotsTxt,
		&b.CreatedAt, &b.UpdatedAt,
	)
	return &b, err
}

// Get returns the singleton branding row, or a hardcoded default when the row
// does not yet exist (never an error for the missing-row case).
func (s *Service) Get(ctx context.Context) (*SiteBranding, error) {
	b, err := scanBranding(s.db.QueryRowContext(ctx, `SELECT `+brandingColumns+` FROM site_branding WHERE id = 1`))
	if err == sql.ErrNoRows {
		return &SiteBranding{ID: 1, SiteName: "My Landing"}, nil
	}
	if err != nil {
		return nil, err
	}
	return b, nil
}

// Update applies a partial COALESCE update and returns the resulting row.
func (s *Service) Update(ctx context.Context, req UpdateRequest) (*SiteBranding, error) {
	const query = `
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
		RETURNING ` + brandingColumns
	return scanBranding(s.db.QueryRowContext(ctx, query,
		req.SiteName, req.Tagline, req.LogoURL, req.LogoIconURL,
		req.FaviconURL, req.AppleTouchIconURL, req.DefaultTitle,
		req.DefaultDescription, req.DefaultOGImageURL, req.ThemePrimaryColor,
		req.ThemeBackgroundColor, req.CanonicalBaseURL,
		req.GoogleSiteVerification, req.RobotsTxt,
	))
}

// ClearField nulls a single whitelisted field. Unknown fields are a no-op.
func (s *Service) ClearField(ctx context.Context, field string) error {
	if !clearableFields[field] {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `UPDATE site_branding SET `+field+` = NULL, updated_at = NOW() WHERE id = 1`)
	return err
}
