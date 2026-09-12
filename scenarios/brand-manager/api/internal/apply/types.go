// Package apply is the domain-scoped home for materializing a brand's identity
// into a target scenario's source tree.
//
// Apply is a stateless orchestration domain — it owns no table. It reads a
// brand and its image assets, writes the requested elements as files inside a
// scenario directory, then records the brand↔scenario link:
//
//	Connect handler → Service (validates, plans, optionally writes) → Workspace
//	                  (filesystem)         ↘ BrandStore   (read the brand)
//	                                       ↘ AssetStore   (read logo/favicon bytes)
//	                                       ↘ AssignmentRecorder (record the link)
//	                     ↑                          ↑
//	                     FakeService (handler tests) Fake{BrandStore,AssetStore,
//	                                                 AssignmentRecorder,Workspace}
//	                                                 (service unit tests)
//
// The cross-domain seams are implemented at the composition root
// (handlers/apply/module.go) over the brands, assets, and assignments domains,
// so the internal domains never import each other. types.go owns the domain
// entities, the seams, and the typed sentinels handlers translate at the edge.
package apply

import (
	"context"
	"fmt"
)

// Element names apply knows how to materialize. AllElements is the default set
// used when a request leaves elements empty.
const (
	ElementColors     = "colors"
	ElementTypography = "typography"
	ElementIdentity   = "identity"
	ElementFavicon    = "favicon"
	ElementLogo       = "logo"
	// ElementIcons copies the derived PWA/launcher icon set into ui/public and
	// writes the manifest icon metadata (icons array, theme/background color).
	ElementIcons = "icons"
)

// AllElements is the canonical apply order when no subset is requested. Colors
// writes the managed CSS file; typography appends to it; so colors must precede
// typography for a deterministic combined file. Icons runs after identity so the
// manifest already carries the brand name before the icon metadata is merged.
var AllElements = []string{ElementColors, ElementTypography, ElementIdentity, ElementIcons, ElementFavicon, ElementLogo}

// Action types — the kind of write an ApplyAction records.
const (
	ActionCSS   = "css"
	ActionJSON  = "json"
	ActionAsset = "asset"
)

// BrandView is the read-only projection of a brand the planner needs to render
// CSS / manifest entries. A projection over the brands domain — apply never
// sees the full brand aggregate or its persistence concerns.
type BrandView struct {
	ID          string
	Version     int
	DisplayName string
	Tagline     string
	Colors      Colors
	Typography  Typography
}

// Colors mirrors brands.Colors so the composition-root adapter maps one-to-one.
type Colors struct {
	Primary    string
	Secondary  string
	Accent     string
	Background string
	Surface    string
	Text       string
	Error      string
}

// HasAny reports whether any color slot is set (an all-empty system is skipped).
func (c Colors) HasAny() bool {
	return c.Primary != "" || c.Secondary != "" || c.Accent != "" ||
		c.Background != "" || c.Surface != "" || c.Text != "" || c.Error != ""
}

// Typography mirrors brands.Typography.
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

// AssetContent is a brand image's bytes plus the metadata apply needs to write
// it into the scenario tree.
type AssetContent struct {
	Filename string
	Bytes    []byte
}

// Action records a single file that was (or, in a preview, would be) written.
type Action struct {
	Type    string // css | json | asset
	File    string // path relative to the scenario root
	Element string // the brand element this materializes
}

// Skip records why an element produced no action.
type Skip struct {
	Element string
	Reason  string
}

// Request is the explicit input DTO Service.Preview / Service.Apply accept.
type Request struct {
	BrandID  string
	Scenario string
	Elements []string
}

// Result reports the plan (Preview) or the outcome (Apply).
type Result struct {
	Scenario     string
	BrandID      string
	BrandVersion int
	DryRun       bool
	Applied      []Action
	Skipped      []Skip
}

// BrandStore reads the brand to apply. Implemented at the composition root over
// the brands domain's service.
type BrandStore interface {
	// Get returns the brand projection, or ErrBrandNotFound when no brand
	// matches.
	Get(ctx context.Context, brandID string) (BrandView, error)
}

// AssetStore reads a brand's stored image bytes by kind ("logo" / "favicon").
// Implemented at the composition root over the assets domain's service.
type AssetStore interface {
	// Read returns the asset content for a brand+kind. found=false (nil error)
	// when the brand has no such asset — the planner reports that as a skip, not
	// a failure.
	Read(ctx context.Context, brandID, kind string) (content AssetContent, found bool, err error)
}

// AssignmentRecorder records the brand↔scenario link after a real apply.
// Implemented at the composition root over the assignments domain's service, so
// the recorded version is re-pinned exactly like a normal assign.
type AssignmentRecorder interface {
	Record(ctx context.Context, brandID, scenario string, elements []string) error
}

// Workspace abstracts the filesystem operations apply performs against a target
// scenario's source tree. The real implementation (NewFSWorkspace) is rooted at
// the scenarios directory; tests substitute an in-memory fake.
type Workspace interface {
	// ScenarioExists reports whether the named scenario's directory exists.
	ScenarioExists(ctx context.Context, scenario string) (bool, error)
	// ReadFile reads a file relative to the scenario root. A missing file
	// returns (nil, nil) so callers can merge-or-create.
	ReadFile(ctx context.Context, scenario, rel string) ([]byte, error)
	// WriteFile writes data to scenario/rel, creating parent directories.
	WriteFile(ctx context.Context, scenario, rel string, data []byte) error
}

// ErrBrandNotFound is the typed sentinel returned when the brand does not
// exist. Handlers translate via errors.As into a Connect NotFound response.
type ErrBrandNotFound struct {
	ID string
}

func (e ErrBrandNotFound) Error() string {
	return fmt.Sprintf("brand %q not found", e.ID)
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

// ErrInvalidApply is the typed sentinel returned when the request is malformed
// (missing brand id or scenario name). Handlers translate via errors.As into a
// Connect InvalidArgument response.
type ErrInvalidApply struct {
	Field  string
	Reason string
}

func (e ErrInvalidApply) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Reason)
}
