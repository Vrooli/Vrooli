// Package design is the domain-scoped home for rendering a brand's identity as a
// canonical DESIGN.md document.
//
// Design is a stateless orchestration domain — it owns no table. It reads a
// brand through a cross-domain seam and renders a markdown document from it:
//
//	Connect handler → Service (loads brand, renders markdown) → BrandStore
//	                     ↑                                        (read brand)
//	                     FakeService (handler tests)                  ↑
//	                                                            FakeBrandStore
//	                                                            (service tests)
//
// The cross-domain seam is implemented at the composition root
// (handlers/design/module.go) over the brands domain's service, so the internal
// domains never import each other. types.go owns the domain entities, the seam,
// and the typed sentinels handlers translate at the edge.
package design

import (
	"context"
	"fmt"
)

// Brand is the read-only view of a brand the design renderer needs. It mirrors
// the brands aggregate but is owned by this package so design never imports the
// brands domain — the composition root adapts brands.Brand onto this shape.
type Brand struct {
	ID          string
	Name        string
	Description string
	Identity    Identity
	Colors      Colors
	Typography  Typography
	Voice       Voice
	Notes       string
	Version     int
}

// Identity holds the visual-identity facets of a brand.
type Identity struct {
	DisplayName string
	Tagline     string
	LogoPath    string
	FaviconPath string
	IconPath    string
}

// HasAny reports whether any identity slot is set.
func (i Identity) HasAny() bool {
	return i.DisplayName != "" || i.Tagline != "" || i.LogoPath != "" ||
		i.FaviconPath != "" || i.IconPath != ""
}

// Colors holds the color system for a brand. Values are CSS colors.
type Colors struct {
	Primary    string
	Secondary  string
	Accent     string
	Background string
	Surface    string
	Text       string
	Error      string
}

// HasAny reports whether any color slot is set.
func (c Colors) HasAny() bool {
	return c.Primary != "" || c.Secondary != "" || c.Accent != "" ||
		c.Background != "" || c.Surface != "" || c.Text != "" || c.Error != ""
}

// Typography holds font and text-style definitions.
type Typography struct {
	HeadingFont  string
	BodyFont     string
	MonoFont     string
	BaseFontSize string
}

// HasAny reports whether any typography slot is set.
func (t Typography) HasAny() bool {
	return t.HeadingFont != "" || t.BodyFont != "" || t.MonoFont != "" || t.BaseFontSize != ""
}

// Voice describes the brand's communication tone and style.
type Voice struct {
	Tone     string
	Style    string
	Keywords []string
}

// HasAny reports whether any voice slot is set.
func (v Voice) HasAny() bool {
	return v.Tone != "" || v.Style != "" || len(v.Keywords) > 0
}

// Design is the rendered DESIGN.md document for a brand — the fields the design
// response carries back.
type Design struct {
	BrandID  string
	Markdown string
}

// BrandStore reads a brand for rendering. Implemented at the composition root
// over the brands domain's service, so design loads brands through the normal
// read path without importing the brands package.
type BrandStore interface {
	// Get returns the brand to render. Returns ErrBrandNotFound when no brand
	// matches the id; any other error is a genuine lookup failure (never a
	// not-found in disguise).
	Get(ctx context.Context, brandID string) (Brand, error)
}

// ErrInvalidDesign is the typed sentinel returned when the request is malformed
// (missing brand id). Handlers translate via errors.As into a Connect
// InvalidArgument response.
type ErrInvalidDesign struct {
	Field  string
	Reason string
}

func (e ErrInvalidDesign) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Reason)
}

// ErrBrandNotFound is the typed sentinel returned when the requested brand does
// not exist. Handlers translate via errors.As into a Connect NotFound response.
type ErrBrandNotFound struct {
	ID string
}

func (e ErrBrandNotFound) Error() string {
	return fmt.Sprintf("brand %q not found", e.ID)
}
