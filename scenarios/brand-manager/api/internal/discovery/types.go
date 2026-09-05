// Package discovery is the domain-scoped home for scanning an existing scenario's
// source tree for branding state and turning it into a draft brand.
//
// Discovery is a stateless orchestration domain — it owns no table. It reads a
// scenario's files, infers a draft brand from whatever branding it finds, then
// (on import) persists that draft through the brands domain:
//
//	Connect handler → Service (scans, infers, optionally imports) → Scanner
//	                  (filesystem)                  ↘ BrandStore (create the brand)
//	                     ↑                                  ↑
//	                     FakeService (handler tests)        Fake{Scanner,BrandStore}
//	                                                        (service unit tests)
//
// The cross-domain seam is implemented at the composition root
// (handlers/discovery/module.go) over the brands domain's service, so the
// internal domains never import each other. types.go owns the domain entities,
// the seams, and the typed sentinels handlers translate at the edge.
package discovery

import (
	"context"
	"fmt"
)

// Source types — the kind of file a DiscoverySource records.
const (
	SourceServiceJSON  = "service_json"
	SourceBrandingJSON = "branding_json"
	SourceManifest     = "manifest"
	SourceThemeCSS     = "theme_css"
	SourceAsset        = "asset"
)

// Identity holds the visual-identity facets discovered for a draft brand.
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

// Colors holds the color system discovered for a draft brand. Values are CSS
// colors.
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

// DraftBrand is the brand the scanner inferred from the discovered sources. Only
// the facets discovery can extract are populated; the brands domain fills in the
// rest of the aggregate at import time.
type DraftBrand struct {
	Name        string
	Description string
	Identity    Identity
	Colors      Colors
}

// Source records one place branding data was found and how confident the scanner
// is in it.
type Source struct {
	File       string
	Type       string
	Confidence float64
	Fields     int
}

// Result reports the outcome of a scan: the sources found, the draft brand
// inferred from them, an overall confidence, and suggestions for missing data.
type Result struct {
	Scenario    string
	Sources     []Source
	Draft       DraftBrand
	Confidence  float64
	Suggestions []string
}

// HasSources reports whether the scan matched at least one source (and so
// produced a non-empty draft).
func (r Result) HasSources() bool {
	return len(r.Sources) > 0
}

// Created is the summary of a brand persisted by Import — the fields the import
// response carries back, without exposing the full brands aggregate.
type Created struct {
	ID      string
	Name    string
	Version int
}

// ImportResult reports the brand created from a scan plus the sources and
// confidence that backed it.
type ImportResult struct {
	Brand      Created
	Sources    []Source
	Confidence float64
}

// Scanner abstracts the read-only filesystem access discovery performs against a
// target scenario's source tree. The real implementation (NewFSScanner) is
// rooted at the scenarios directory; tests substitute an in-memory fake.
type Scanner interface {
	// ScenarioExists reports whether the named scenario's directory exists.
	ScenarioExists(ctx context.Context, scenario string) (bool, error)
	// ReadFile reads a file relative to the scenario root. A missing file
	// returns (nil, nil) so callers can probe-or-skip.
	ReadFile(ctx context.Context, scenario, rel string) ([]byte, error)
	// ListDir lists the entry names directly under scenario/rel. A missing
	// directory returns (nil, nil).
	ListDir(ctx context.Context, scenario, rel string) ([]string, error)
}

// BrandStore persists a discovered draft as a new brand. Implemented at the
// composition root over the brands domain's service, so an import goes through
// the normal create + version-snapshot path.
type BrandStore interface {
	// Create persists the draft and returns the created brand summary.
	Create(ctx context.Context, draft DraftBrand) (Created, error)
}

// ErrScenarioNotFound is the typed sentinel returned when the target scenario's
// directory does not exist. Handlers translate via errors.As into a Connect
// NotFound response.
type ErrScenarioNotFound struct {
	Scenario string
}

func (e ErrScenarioNotFound) Error() string {
	return fmt.Sprintf("scenario %q not found", e.Scenario)
}

// ErrNoBrandingFound is the typed sentinel returned by Import when a scan found
// no branding state to import. Handlers translate via errors.As into a Connect
// FailedPrecondition response.
type ErrNoBrandingFound struct {
	Scenario string
}

func (e ErrNoBrandingFound) Error() string {
	return fmt.Sprintf("no branding state found to import in scenario %q", e.Scenario)
}

// ErrInvalidDiscovery is the typed sentinel returned when the request is
// malformed (missing scenario name). Handlers translate via errors.As into a
// Connect InvalidArgument response.
type ErrInvalidDiscovery struct {
	Field  string
	Reason string
}

func (e ErrInvalidDiscovery) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Reason)
}
