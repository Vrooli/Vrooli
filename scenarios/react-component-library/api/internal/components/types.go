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
	"context"
	"fmt"
	"strings"
	"time"
)

// Component is the internal domain shape for an indexed component.
// The wire/proto type lives at the transport edge; this struct is the
// only shape internal callers depend on.
type Component struct {
	ID            string
	CatalogID     string
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
	AssetKind     AssetKind
	Dependencies  []AssetDependency
	Expects       []string
	Satisfies     []string
	Metrics       AssetMetrics
}

// AssetKind distinguishes a renderable catalog component from a reusable,
// non-renderable hook. The Components domain retains its stable API name while
// acting as the library-asset projection.
type AssetKind string

const (
	AssetKindComponent AssetKind = "component"
	AssetKindHook      AssetKind = "hook"
)

func (k AssetKind) Valid() bool { return k == AssetKindComponent || k == AssetKindHook }

// AssetDependency pins a consuming asset to one immutable version of another
// library asset. The resolver expands these edges into a deterministic closure.
type AssetDependency struct {
	LibraryID string
	Version   string
	Kind      DependencyKind
}

type DependencyKind string

const (
	DependencyRequires DependencyKind = "requires"
	DependencySuggests DependencyKind = "suggests"
)

func (k DependencyKind) normalized() DependencyKind {
	if k == DependencySuggests {
		return k
	}
	return DependencyRequires
}

// AssetMetrics are server-projected catalog counts. They must be populated in
// batch for list operations so presentation modes never issue per-row counts.
type AssetMetrics struct {
	DirectAdoptionCount    int
	EffectiveAdoptionCount int
	VersionCount           int
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
	Files         []ComponentVersionFile
	// ExperienceContract is the immutable behavior contract promoted beside
	// the version. It is separate from source files so adopters can copy the
	// contract into their own experience registry deliberately.
	ExperienceContract string
	ParityReport       *IngestParityReport
}

// ComponentVersionFile is one immutable member of a versioned component unit.
// Path is relative to that version folder; the entry file remains mirrored on
// ComponentVersion for callers that only need the renderable artifact.
type ComponentVersionFile struct {
	Path          string
	Content       string
	ContentSHA256 string
	IsEntry       bool
	// Slot is the explicit placement slot for this file (e.g. "hook" for a
	// companion). Empty means unspecified — the adoption path resolver derives
	// the slot from an extension heuristic or the component's declared slot.
	// Authored via component.json `fileSlots`; explicit metadata wins.
	Slot string
}

// IngestParityReport is durable evidence of static behavior parity between
// the origin source unit and the harvested library unit.
type IngestParityReport struct {
	OriginFiles    []string        `json:"originFiles"`
	HarvestedFiles []string        `json:"harvestedFiles"`
	Findings       []IngestFinding `json:"findings"`
	Acknowledged   bool            `json:"acknowledged"`
}

// ErrParityWaiverRequired prevents a lossy ingest draft from becoming a
// released artifact until an operator explicitly acknowledges the report.
type ErrParityWaiverRequired struct {
	ComponentID, Version string
	Findings             []IngestFinding
}

func (e ErrParityWaiverRequired) Error() string {
	return "ingest parity report has behavior-loss findings; acknowledge a parity waiver before release"
}

// ErrHarvestBehaviorLoss blocks a harvest whose static behavior inventory
// shows origin behavior signals absent from the harvested unit — the failure
// mode that once let DrawerShell be harvested with its focus-trap hook
// silently dropped. The harvest fails unless the caller passes
// accept_behavior_loss, which does not skip the check but records the named
// losses as an acknowledged parity report on the created version.
type ErrHarvestBehaviorLoss struct {
	Scenario   string
	SourceFile string
	Findings   []IngestFinding
}

func (e ErrHarvestBehaviorLoss) Error() string {
	losses := make([]string, 0, len(e.Findings))
	for _, f := range e.Findings {
		losses = append(losses, f.Message)
	}
	return fmt.Sprintf("harvest of %s:%s drops %d origin behavior signal(s): %s; re-run with --accept-behavior-loss to record and accept these losses",
		e.Scenario, e.SourceFile, len(e.Findings), strings.Join(losses, "; "))
}

// ComponentManifest is the explicit DTO the manifest indexer hands to
// the service. Distinct from Component so the service owns ID
// assignment and IndexedAt/UpdatedAt stamping.
type ComponentManifest struct {
	CatalogID          string
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
	AssetKind          AssetKind
	Dependencies       []AssetDependency
	Expects            []string
	Satisfies          []string
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
	AssetKind     AssetKind
	Dependencies  []AssetDependency
}

// IndexManifestInput is the full registry payload for one component
// manifest and all validated version folders.
type IndexManifestInput struct {
	Manifest ComponentManifest
	Versions []ComponentVersion
	Stories  []ComponentStory
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
	IndexFindingHeaderDisagreement  IndexFindingKind = "header_disagreement"
	IndexFindingStaleDesignStyle    IndexFindingKind = "stale_design_style"
	IndexFindingInvalidStory        IndexFindingKind = "invalid_story"
	IndexFindingMissingStory        IndexFindingKind = "missing_story"
	IndexFindingLegacyStorySource   IndexFindingKind = "legacy_story_source"
	IndexFindingStoryHarnessMissing IndexFindingKind = "story_harness_missing"
	IndexFindingStoryHarnessExport  IndexFindingKind = "story_harness_export"
	IndexFindingStoryHarnessOrphan  IndexFindingKind = "story_harness_orphan"
	// IndexFindingRegistryOrphan is emitted when a component_versions
	// row (or its sibling child rows) has no owning row in the
	// components registry — soft-FK cruft the reindex sweep removes.
	IndexFindingRegistryOrphan IndexFindingKind = "registry_orphan"
	// IndexFindingMissingDesignAffinity is emitted when a component's
	// manifest declares no design-style affinities. Authored components
	// carry 2-3; a promoted harvest with none is catalog-incomplete and
	// renders "No design affinities declared" in the detail view. Soft
	// conformance signal — it never blocks the reindex.
	IndexFindingMissingDesignAffinity IndexFindingKind = "missing_design_affinity"
)

// OrphanVersion is a component_versions row whose component_id has no
// owning row in the components registry — soft-FK cruft left when a
// component is re-slugged or withdrawn without clearing its child
// rows. Returned by Repository.SweepOrphans so the indexer can emit a
// registry-orphan conformance finding for each removed row.
type OrphanVersion struct {
	ComponentID string
	LibraryID   string
	Version     string
	SourcePath  string
}

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
	Match     string
	Tag       string
	Tags      []string
	Category  string
	StyleID   string
	Affinity  string
	AssetKind AssetKind
	Limit     int
}

// ComponentStory is the durable typed projection of one version's story.json.
// The canonical source remains the file; SQLite exists so every transport and
// runtime consumer sees the exact same validated contract.
type ComponentStory struct {
	ID              string
	ComponentID     string
	LibraryID       string
	Version         string
	SchemaVersion   int
	Kind            StoryKind
	Title           string
	ArgsJSON        string
	EnvironmentJSON string
	StoriesJSON     string
	ContractJSON    string
	SourcePath      string
	IndexedAt       time.Time
}

type StoryQuery struct {
	ComponentID string
	Version     string
	Limit       int
}

type InitializeComponentInput struct {
	LibraryID   string
	Slug        string
	DisplayName string
	Description string
	Tags        []string
	Slot        string
	// Category is catalog metadata persisted on the manifest. Ingest supplies
	// it (defaulting when the harvester leaves it blank) so drafts land with the
	// same catalog fields authored components carry.
	Category       string
	InitialVersion string
	FileName       string
	InitialSource  string
	// InitialFiles is the complete version unit. Paths are basenames relative
	// to the version folder; exactly one member is the renderable entry.
	// Empty retains the single-file InitialSource contract.
	InitialFiles []ComponentVersionFile
	// ScaffoldExamples is retained for CLI compatibility and writes a starter story.json.
	// folder when the caller supplies none. Ingest sets it so every harvested
	// draft carries the examples contract authored components ship with.
	ScaffoldExamples bool
}

type InitializeComponentResult struct {
	Component    Component
	ManifestPath string
	SourcePath   string
}

// ScenarioSourceReader reads a path relative to a named scenario's root. The
// production implementation enforces traversal safety; components keeps this
// as a narrow interface so ingest policy remains testable without filesystem
// coupling.
type ScenarioSourceReader interface {
	Read(ctx context.Context, scenario, sourcePath string) ([]byte, error)
}

type IngestFinding struct {
	Code       string
	Message    string
	SourceFile string
}

type IngestComponentInput struct {
	Scenario   string
	SourceFile string
	// SourceFiles optionally supplies companion paths relative to the origin
	// scenario. Relative-import closure discovery supplements this list.
	SourceFiles []string
	// Version is the released semver baseline for a re-harvest. Empty keeps
	// the original 0.1.0 bootstrap behavior.
	Version     string
	Slug        string
	DisplayName string
	Description string
	Tags        []string
	Slot        string
	Category    string
	// AcceptBehaviorLoss records an explicit decision to harvest despite the
	// behavior-inventory diff reporting dropped origin behavior. It never skips
	// the diff — the losses are still named and persisted as an acknowledged
	// parity report on the created version.
	AcceptBehaviorLoss bool
	// ExperienceContractPath is the scenario-relative source contract promoted
	// with the component. Empty derives experience/components/<slug>.json.
	ExperienceContractPath string
}

type IngestComponentResult struct {
	Component     Component
	ManifestPath  string
	SourcePath    string
	DraftVersion  string
	Findings      []IngestFinding
	ParityReport  IngestParityReport
	ChecklistPath string
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
	// Files replaces the single Source body when non-empty. Exactly one member
	// must be marked IsEntry.
	Files                   []ComponentVersionFile
	ParityReport            *IngestParityReport
	AcknowledgeParityWaiver bool
	ChangelogMD             string
	// ScaffoldExamples is retained for CLI compatibility and writes a starter story.json.
	// folder when the caller supplies none. Ingest sets it for harvested drafts.
	ScaffoldExamples bool
	// ExperienceContract is the canonical JSON document written alongside this
	// immutable version's source and examples.
	ExperienceContract string
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
