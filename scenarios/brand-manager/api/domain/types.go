// Package domain defines the core brand-manager entities and value types.
//
// These types represent the domain mental model:
//   - Brand: the root entity containing all branding facets
//   - BrandVersion: an immutable snapshot of a brand's state
//   - Assignment: links a brand to a scenario
//
// DOC: docs/concepts/ARCHITECTURE.md#domain-model
package domain

import "time"

// Brand is the root aggregate for a complete brand identity.
type Brand struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Identity    *Identity   `json:"identity,omitempty"`
	Colors      *Colors     `json:"colors,omitempty"`
	Typography  *Typography `json:"typography,omitempty"`
	Voice       *Voice      `json:"voice,omitempty"`
	Notes       string      `json:"notes,omitempty"`
	Version     int         `json:"version"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// Identity holds the visual identity facets of a brand.
type Identity struct {
	DisplayName string `json:"display_name,omitempty"`
	Tagline     string `json:"tagline,omitempty"`
	LogoPath    string `json:"logo_path,omitempty"`
	FaviconPath string `json:"favicon_path,omitempty"`
	IconPath    string `json:"icon_path,omitempty"`
}

// Colors holds the color system for a brand.
type Colors struct {
	Primary    string `json:"primary,omitempty"`
	Secondary  string `json:"secondary,omitempty"`
	Accent     string `json:"accent,omitempty"`
	Background string `json:"background,omitempty"`
	Surface    string `json:"surface,omitempty"`
	Text       string `json:"text,omitempty"`
	Error      string `json:"error,omitempty"`
}

// Typography holds font and text style definitions.
type Typography struct {
	HeadingFont  string `json:"heading_font,omitempty"`
	BodyFont     string `json:"body_font,omitempty"`
	MonoFont     string `json:"mono_font,omitempty"`
	BaseFontSize string `json:"base_font_size,omitempty"`
}

// Voice describes the brand's communication tone and style.
type Voice struct {
	Tone     string   `json:"tone,omitempty"`
	Style    string   `json:"style,omitempty"`
	Keywords []string `json:"keywords,omitempty"`
}

// BrandVersion is an immutable snapshot of a brand at a point in time.
type BrandVersion struct {
	ID        string    `json:"id"`
	BrandID   string    `json:"brand_id"`
	Version   int       `json:"version"`
	Snapshot  string    `json:"snapshot"` // JSON-encoded full brand state
	CreatedAt time.Time `json:"created_at"`
}

// Assignment links a brand to a scenario.
type Assignment struct {
	ID           string    `json:"id"`
	BrandID      string    `json:"brand_id"`
	ScenarioName string    `json:"scenario_name"`
	BrandVersion int       `json:"brand_version"`
	Elements     []string  `json:"elements,omitempty"` // which elements were applied
	AppliedAt    time.Time `json:"applied_at"`
}

// ApplyPartialUpdate merges non-zero fields from other into b.
// This implements "partial update" semantics: only fields the caller explicitly
// provides (non-empty string, non-nil pointer) overwrite the existing values.
// Zero-valued fields in other are treated as "not provided" and left unchanged.
func (b *Brand) ApplyPartialUpdate(other Brand) {
	if other.Name != "" {
		b.Name = other.Name
	}
	if other.Description != "" {
		b.Description = other.Description
	}
	if other.Identity != nil {
		b.Identity = other.Identity
	}
	if other.Colors != nil {
		b.Colors = other.Colors
	}
	if other.Typography != nil {
		b.Typography = other.Typography
	}
	if other.Voice != nil {
		b.Voice = other.Voice
	}
	if other.Notes != "" {
		b.Notes = other.Notes
	}
}

// ScenarioStatus reports whether a scenario has a brand assigned and, if so,
// which brand version and elements were applied. This is the response shape
// for GET /api/v1/scenarios/{name}/status.
type ScenarioStatus struct {
	Scenario     string     `json:"scenario"`
	HasBrand     bool       `json:"has_brand"`
	BrandID      *string    `json:"brand_id"`
	BrandVersion *int       `json:"brand_version"`
	Elements     []string   `json:"elements,omitempty"`
	AppliedAt    *time.Time `json:"applied_at,omitempty"`
}

// ScenarioStatusUnassigned returns a status indicating no brand is assigned.
func ScenarioStatusUnassigned(scenario string) ScenarioStatus {
	return ScenarioStatus{Scenario: scenario, HasBrand: false}
}

// ScenarioStatusFromAssignment builds a status from an existing assignment.
func ScenarioStatusFromAssignment(scenario string, a *Assignment) ScenarioStatus {
	return ScenarioStatus{
		Scenario:     scenario,
		HasBrand:     true,
		BrandID:      &a.BrandID,
		BrandVersion: &a.BrandVersion,
		Elements:     a.Elements,
		AppliedAt:    &a.AppliedAt,
	}
}

// Asset represents a brand asset file (logo, favicon, icon, etc.).
type Asset struct {
	ID        string    `json:"id"`
	BrandID   string    `json:"brand_id"`
	Filename  string    `json:"filename"`
	MimeType  string    `json:"mime_type"`
	FilePath  string    `json:"file_path"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
}

// ScanResult represents a single inline branding marker found in a file.
type ScanResult struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Type    string `json:"type"`    // "css" or "json"
	Marker  string `json:"marker"`  // the matched marker text
	Element string `json:"element"` // extracted element name (e.g. "primary", "logo")
}

// ScanReport summarizes inline validation scan results for a scenario.
type ScanReport struct {
	Scenario   string       `json:"scenario"`
	CSSMarkers int          `json:"css_markers"`
	JSONKeys   int          `json:"json_keys"`
	Total      int          `json:"total"`
	Results    []ScanResult `json:"results"`
}

// BrandFilter specifies optional filters for listing brands.
type BrandFilter struct {
	NameContains string
	Limit        int
	Offset       int
}
