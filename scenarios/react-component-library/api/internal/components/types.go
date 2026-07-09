// Package components is the domain-scoped home for the component
// registry — the indexed view of Git-tracked component manifests and
// their explicit version folders. Cross-cutting concerns (versions,
// adoptions, deps, themes) live in sibling internal/<dom>/ packages and
// reference components by ID with no hard FK.
//
// Layering:
//
//	HTTP → handler → Service (validation, defaults) → Repository (sqlite)
//	                      ↑                                ↑
//	                      FakeService (handler tests)       FakeRepository (service tests)
//	                                                        Real sqlite (repository tests)
//
// types.go owns the domain entity and typed sentinels. repository.go
// owns the persistence seam. service.go owns application policy.
// indexer.go walks library/components/*/component.json and validates
// the source-local version headers.
package components

import (
	"fmt"
	"time"
)

// Component is the internal domain shape for an indexed component.
// The wire/proto type lives at the transport edge; this struct is the
// only shape internal callers depend on.
type Component struct {
	ID            string
	LibraryID     string
	Slug          string
	DisplayName   string
	Description   string
	Slot          string
	Category      string
	SourcePath    string
	Version       string
	LatestVersion string
	DraftVersion  string
	ManifestPath  string
	Tags          []string
	IndexedAt     time.Time
	UpdatedAt     time.Time
	Headers       map[string]string
	DesignStyles  []ComponentDesignAffinity
}

// ComponentVersionStatus classifies a version folder.
type ComponentVersionStatus string

const (
	VersionStatusDraft      ComponentVersionStatus = "draft"
	VersionStatusReleased   ComponentVersionStatus = "released"
	VersionStatusDeprecated ComponentVersionStatus = "deprecated"
	VersionStatusArchived   ComponentVersionStatus = "archived"
)

// ComponentVersion is the immutable/draft source artifact indexed for
// a component version folder.
type ComponentVersion struct {
	ID            string
	ComponentID   string
	LibraryID     string
	Version       string
	Status        ComponentVersionStatus
	SourcePath    string
	Content       string
	ContentSHA256 string
	ChangelogMD   string
	IndexedAt     time.Time
	ReleasedAt    time.Time
	Headers       map[string]string
}

// ComponentManifest is the explicit DTO the manifest indexer hands to
// the service. Distinct from Component so the service owns ID
// assignment and IndexedAt/UpdatedAt stamping.
type ComponentManifest struct {
	LibraryID          string
	Slug               string
	DisplayName        string
	Description        string
	Slot               string
	Category           string
	ManifestPath       string
	LatestVersion      string
	DraftVersion       string
	DeprecatedVersions []string
	Tags               []string
	DesignStyles       []ComponentDesignAffinity
}

// UpsertInput is retained as a convenience alias for older tests and
// service callers inside this greenfield scenario. New index code uses
// ComponentManifest directly.
type UpsertInput struct {
	LibraryID     string
	Slug          string
	DisplayName   string
	Description   string
	Slot          string
	Category      string
	ManifestPath  string
	SourcePath    string
	Version       string
	LatestVersion string
	DraftVersion  string
	Tags          []string
	Headers       map[string]string
	DesignStyles  []ComponentDesignAffinity
}

// IndexManifestInput is the full registry payload for one component
// manifest and all validated version folders.
type IndexManifestInput struct {
	Manifest ComponentManifest
	Versions []ComponentVersion
	Examples []ComponentExample
	Headers  map[string]string
	Findings []IndexFinding
}

type DesignAffinity string

const (
	DesignAffinityNative      DesignAffinity = "native"
	DesignAffinityCompatible  DesignAffinity = "compatible"
	DesignAffinityDiscouraged DesignAffinity = "discouraged"
)

type ComponentDesignAffinity struct {
	StyleID  string
	Affinity DesignAffinity
	Reason   string
}

type StyleFitVerdictKind string

const (
	StyleFitVerdictOK   StyleFitVerdictKind = "ok"
	StyleFitVerdictInfo StyleFitVerdictKind = "info"
	StyleFitVerdictWarn StyleFitVerdictKind = "warn"
)

type StyleFitVerdict struct {
	Kind          StyleFitVerdictKind
	ComponentID   string
	Version       string
	Scenario      string
	ScenarioStyle string
	Affinity      DesignAffinity
	Detail        string
}

type ComponentSlot string

const (
	ComponentSlotUIPrimitive ComponentSlot = "ui-primitive"
	ComponentSlotUIPattern   ComponentSlot = "ui-pattern"
	ComponentSlotLayoutNav   ComponentSlot = "layout-nav"
)

type IndexFindingKind string

const (
	IndexFindingHeaderDisagreement IndexFindingKind = "header_disagreement"
	IndexFindingStaleDesignStyle   IndexFindingKind = "stale_design_style"
	IndexFindingInvalidExample     IndexFindingKind = "invalid_example"
)

type IndexFinding struct {
	Kind       IndexFindingKind
	SourcePath string
	Field      string
	Expected   string
	Actual     string
	Detail     string
}

// SearchQuery filters a List call. All fields optional.
//
// Match is a case-insensitive substring tested against library_id,
// display_name, description, and source_path. When Match is set the
// result is ordered by display_name (case-insensitive); otherwise by
// indexed_at DESC.
//
// Tag (singular) matches a single exact-token hit. Tags (plural)
// matches ANY of the supplied tokens (OR semantics, req SF-002). Both
// AND with Match and with Category. Category is an AND filter that
// looks up the canonical `@category` header field; only components
// declaring that header value match.
type SearchQuery struct {
	Match    string
	Tag      string
	Tags     []string
	Category string
	StyleID  string
	Affinity string
	Limit    int
}

type ExampleQuery struct {
	ComponentID string
	Version     string
	Limit       int
}

type ComponentExample struct {
	ID          string
	ComponentID string
	LibraryID   string
	Version     string
	Name        string
	DisplayName string
	PropsJSON   string
	SetupJSON   string
	ExpectJSON  string
	SourcePath  string
	IndexedAt   time.Time
}

type InitializeComponentInput struct {
	LibraryID      string
	Slug           string
	DisplayName    string
	Description    string
	Tags           []string
	InitialVersion string
	FileName       string
	InitialSource  string
}

type InitializeComponentResult struct {
	Component    Component
	ManifestPath string
	SourcePath   string
}

type VersionIntent string

const (
	VersionIntentDraft   VersionIntent = "draft"
	VersionIntentRelease VersionIntent = "release"
)

type CreateComponentVersionInput struct {
	ComponentID string
	Version     string
	FromVersion string
	Intent      VersionIntent
	FileName    string
	Source      string
	ChangelogMD string
}

type CreateComponentVersionResult struct {
	Component  Component
	Version    ComponentVersion
	SourcePath string
}

type UpdateComponentManifestInput struct {
	ComponentID        string
	DisplayName        string
	Description        string
	Tags               []string
	LatestVersion      string
	DraftVersion       string
	DeprecatedVersions []string
}

// ErrComponentNotFound is the typed sentinel handlers translate to a
// 404 via errors.As.
type ErrComponentNotFound struct {
	IDOrLibraryID string
}

func (e ErrComponentNotFound) Error() string {
	return fmt.Sprintf("component %q not found", e.IDOrLibraryID)
}

// ErrInvalidHeader is returned by the indexer/service when a manifest
// or source-local header fails validation. Field names the offending
// field; Reason is a human-safe explanation.
type ErrInvalidHeader struct {
	SourcePath string
	Field      string
	Reason     string
}

func (e ErrInvalidHeader) Error() string {
	return fmt.Sprintf("%s: header field %s: %s", e.SourcePath, e.Field, e.Reason)
}

// ErrComponentAlreadyExists is returned before creating source folders
// when either the manifest libraryId or slug is already in use.
type ErrComponentAlreadyExists struct {
	LibraryID string
	Slug      string
}

func (e ErrComponentAlreadyExists) Error() string {
	if e.LibraryID != "" {
		return fmt.Sprintf("component libraryId %q already exists", e.LibraryID)
	}
	return fmt.Sprintf("component slug %q already exists", e.Slug)
}
