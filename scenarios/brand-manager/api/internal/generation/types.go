// Package generation is the domain-scoped home for AI-assisted brand element
// and image generation.
//
// Generation is a stateless orchestration domain — it owns no table. It reads a
// brand, asks an AI provider chain to produce facets/images, then writes the
// results back through two cross-domain seams:
//
//	Connect handler → Service (validates, fans out to the provider chain, applies
//	                  results) → Providers (AI chain)
//	                              ↘ BrandStore (read brand, apply text facets)
//	                              ↘ AssetStore (store generated images)
//	                     ↑                 ↑
//	                     FakeService        Fake{Providers,BrandStore,AssetStore}
//	                     (handler tests)    (service unit tests)
//
// The two cross-domain seams are implemented at the composition root
// (handlers/generation/module.go) over the brands and assets domains, so the
// internal domains never import each other. types.go owns the domain entities,
// the seams, and the typed sentinels handlers translate at the transport edge.
package generation

import "fmt"

// Colors holds a generated color system. Field names mirror brands.Colors so
// the composition-root adapter maps them one-to-one.
type Colors struct {
	Primary    string
	Secondary  string
	Accent     string
	Background string
	Surface    string
	Text       string
	Error      string
}

// Typography holds generated font choices. Mirrors brands.Typography.
type Typography struct {
	HeadingFont  string
	BodyFont     string
	MonoFont     string
	BaseFontSize string
}

// Voice holds a generated communication tone/style. Mirrors brands.Voice.
type Voice struct {
	Tone     string
	Style    string
	Keywords []string
}

// BrandView is the read-only slice of a brand the generator needs to build its
// prompts. A projection over the brands domain — generation never sees the full
// brand aggregate.
type BrandView struct {
	ID           string
	Name         string
	Description  string
	Notes        string
	PrimaryColor string
	Version      int
}

// ApplyElementsInput carries the generated text facets to merge onto a brand. A
// nil facet pointer means "not generated, leave unchanged" — the brands domain
// applies partial-merge semantics so siblings survive.
type ApplyElementsInput struct {
	BrandID    string
	Colors     *Colors
	Typography *Typography
	Voice      *Voice
}

// AssetUpload is the byte payload + metadata the generator hands the assets
// domain to persist a generated image.
type AssetUpload struct {
	BrandID  string
	Filename string
	MimeType string
	Content  []byte
}

// StoredAsset is the catalog metadata the assets domain returns after storing a
// generated image.
type StoredAsset struct {
	ID       string
	Filename string
	MimeType string
	Size     int64
}

// ElementsResult is the outcome of a GenerateBrandElements call.
type ElementsResult struct {
	Results  []ElementOutcome
	Applied  []string
	Provider string
	Model    string
	Version  int
}

// ElementOutcome is the per-element status the response reports.
type ElementOutcome struct {
	Element string
	Status  string // applied | failed | unsupported
	Detail  string
}

// Per-element status constants (the wire contract's `status` strings).
const (
	StatusApplied     = "applied"
	StatusFailed      = "failed"
	StatusUnsupported = "unsupported"
)

// ImageResult is the outcome of an image RPC (generate / edit / remove
// background / one derived icon). It is the stored brand asset plus the
// image-tools model/tier that produced it.
type ImageResult struct {
	BrandID   string
	AssetID   string
	Kind      string // logo | favicon | logo-transparent | favicon-32 | apple-touch-icon | ...
	Filename  string
	MimeType  string
	Size      int64
	ModelID   string // image-tools model; empty for deterministic derivations
	Tier      string // local-gpu | local-cpu | byok-cloud | deterministic
	Canonical bool   // true when also written as the canonical asset for its kind
	Warnings  []string
}

// AssetBytes is a brand asset's bytes + metadata, returned when the generator
// loads a source image (for edits / background removal / icon derivation).
type AssetBytes struct {
	ID       string
	Filename string
	MimeType string
	Content  []byte
}

// ErrBrandNotFound is the typed sentinel returned when the target brand does
// not exist. Handlers translate via errors.As into a Connect NotFound response.
type ErrBrandNotFound struct {
	ID string
}

func (e ErrBrandNotFound) Error() string {
	return fmt.Sprintf("brand %q not found", e.ID)
}

// ErrInvalidGeneration is the typed sentinel returned when the request is
// malformed (missing brand id, empty elements, unknown image type). Handlers
// translate via errors.As into a Connect InvalidArgument response.
type ErrInvalidGeneration struct {
	Field  string
	Reason string
}

func (e ErrInvalidGeneration) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Reason)
}

// ErrProvidersUnavailable is the typed sentinel returned when no text AI
// provider in the chain is currently reachable. Handlers translate via errors.As
// into a Connect Unavailable response, so callers can retry later.
type ErrProvidersUnavailable struct{}

func (ErrProvidersUnavailable) Error() string {
	return "no text AI providers are currently available"
}

// ErrSourceAssetNotFound is the typed sentinel returned when an image edit /
// background removal / icon derivation references a source asset that does not
// exist. Handlers translate via errors.As into a Connect NotFound response.
type ErrSourceAssetNotFound struct {
	ID string
}

func (e ErrSourceAssetNotFound) Error() string {
	return fmt.Sprintf("source asset %q not found", e.ID)
}
