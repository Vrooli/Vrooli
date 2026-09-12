// Package brands is the domain-scoped home for the brand-identity resource.
//
// Layering mirrors the canonical Vrooli pattern (see internal/notes for the
// reference example domain):
//
//	Connect handler → Service (validates, applies defaults, owns version
//	                  lifecycle) → Repository / VersionRepository (persist)
//	                     ↑                          ↑
//	                     FakeService (handler tests) FakeRepository (service tests)
//	                                                 Real sqlite (repository tests)
//
// types.go owns the domain entities and the typed sentinels handlers translate
// at the transport edge. The proto wire types live one floor up
// (packages/proto/...) and never import this package; the handler is the only
// translation point (api-steer §7).
package brands

import (
	"fmt"
	"time"
)

// Brand is the internal domain shape for a complete brand identity. Distinct
// from the proto wire type at packages/proto/gen/go/.../v1/brands.Brand —
// handlers translate at the boundary so the domain layer never imports proto.
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
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Identity holds the visual-identity facets of a brand.
type Identity struct {
	DisplayName string
	Tagline     string
	LogoPath    string
	FaviconPath string
	IconPath    string
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

// Typography holds font and text-style definitions.
type Typography struct {
	HeadingFont  string
	BodyFont     string
	MonoFont     string
	BaseFontSize string
}

// Voice describes the brand's communication tone and style.
type Voice struct {
	Tone     string
	Style    string
	Keywords []string
}

// BrandVersion is an immutable snapshot of a brand at a point in time.
type BrandVersion struct {
	ID        string
	BrandID   string
	Version   int
	Snapshot  string // JSON-encoded full brand state at this version
	CreatedAt time.Time
}

// CreateInput is the explicit input DTO Service.Create accepts. Distinct from
// Brand so callers cannot pass an ID, version, or timestamp the service has no
// way to honour — those belong to the persistence layer.
type CreateInput struct {
	Name        string
	Description string
	Notes       string
	Identity    Identity
	Colors      Colors
	Typography  Typography
	Voice       Voice
}

// UpdateInput is the partial-update DTO Service.Update accepts. Only non-empty
// scalar fields and non-empty facet sub-fields overwrite the stored value. A
// zero-valued facet field means "leave unchanged" (merge semantics).
//
// ExpectedVersion, when > 0, makes the update optimistic-locked: it is rejected
// with ErrVersionConflict unless it equals the brand's current version.
type UpdateInput struct {
	ID              string
	Name            string
	Description     string
	Notes           string
	Identity        Identity
	Colors          Colors
	Typography      Typography
	Voice           Voice
	ExpectedVersion int
}

// ListFilter specifies optional filters for listing brands.
type ListFilter struct {
	NameContains string
	Limit        int
	Offset       int
}

// ErrBrandNotFound is the typed sentinel returned when no row matches.
// Handlers translate via errors.As into a Connect NotFound response.
type ErrBrandNotFound struct {
	ID string
}

func (e ErrBrandNotFound) Error() string {
	return fmt.Sprintf("brand %q not found", e.ID)
}

// ErrInvalidBrand is the typed sentinel returned when validation fails. Field
// names the offending field; Reason is a human-safe explanation. Handlers
// translate via errors.As into a Connect InvalidArgument response.
type ErrInvalidBrand struct {
	Field  string
	Reason string
}

func (e ErrInvalidBrand) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Reason)
}

// ErrVersionConflict is the typed sentinel returned when an UpdateBrand
// expected_version does not match the brand's current version. Handlers
// translate via errors.As into a Connect FailedPrecondition response.
type ErrVersionConflict struct {
	ID       string
	Expected int
	Actual   int
}

func (e ErrVersionConflict) Error() string {
	return fmt.Sprintf("brand %q version conflict: expected %d, actual %d", e.ID, e.Expected, e.Actual)
}
