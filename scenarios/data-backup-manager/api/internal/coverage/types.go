// Package coverage is the first-real-backup readiness domain. It composes the
// existing discovery, targets, plans, runs and restores seams into one
// decision-oriented coverage report and one bulk default-acceptance action. It
// owns NO scanner or catalog logic of its own — every fact it reports is read
// through a seam, so coverage stays a thin policy layer over the five-noun
// product plus discovery.
//
// Layering mirrors the canonical per-domain pattern, but there is no repository:
// coverage derives everything live and persists nothing.
//
//	Connect handler → Service (compose, classify) → seams (suggestions, catalog, plans, runs, restores)
//
// The proto wire types live one floor up (packages/proto/...) and never import
// this package; the handler is the only translation point.
package coverage

import (
	"context"
	"time"

	"data-backup-manager/internal/sources"
)

// Suggestion is a discovered durable source not yet registered (discovery has
// already filtered out registered and dismissed ones). Non-sensitive
// suggestions are the default coverage set; sensitive ones are review-only.
type Suggestion struct {
	ID          string
	Owner       string
	Name        string
	SourceKind  sources.SourceKind
	Locator     string
	Rationale   string
	ApproxBytes int64
	Sensitive   bool
	Warning     string
}

// CatalogTarget is the minimal projection of a registered target the coverage
// service reads from (and writes to) the targets catalog.
type CatalogTarget struct {
	ID         string
	Owner      string
	Name       string
	SourceKind sources.SourceKind
	Locator    string
}

// RegisterInput is the DTO the coverage service passes to the targets catalog
// when bulk-accepting a suggestion. It carries only the locators discovery
// surfaced — coverage never reads file contents.
type RegisterInput struct {
	Owner      string
	Name       string
	SourceKind sources.SourceKind
	Locator    string
}

// RegisteredTarget annotates a registered target with the coverage states an
// operator needs to judge whether it is actually protected.
type RegisteredTarget struct {
	CatalogTarget
	Planned        bool
	LastSuccessAt  time.Time
	LastVerifiedAt time.Time
}

// Summary is the at-a-glance scorecard the report leads with.
type Summary struct {
	RegisteredCount               int
	RecommendedCount              int
	SensitiveCount                int
	PlannedCount                  int
	BackedUpCount                 int
	VerifiedCount                 int
	DefaultCoverageComplete       bool
	HasSensitiveUnreviewed        bool
	HasUnplannedRegisteredTargets bool
	HasUnverifiedTargets          bool
}

// Report is the full first-backup-readiness picture, derived live.
type Report struct {
	Summary     Summary
	Registered  []RegisteredTarget
	Recommended []Suggestion // non-sensitive
	Sensitive   []Suggestion // sensitive, review-only
}

// AcceptOptions configures bulk default acceptance.
type AcceptOptions struct {
	IncludeSensitive bool
	DryRun           bool
}

// AcceptedTarget is one suggestion the bulk accept registered (or, under
// DryRun, would register). TargetID is empty for a dry run.
type AcceptedTarget struct {
	TargetID     string
	SuggestionID string
	Owner        string
	Name         string
	SourceKind   sources.SourceKind
	Locator      string
	Sensitive    bool
}

// AcceptFailure reports a per-item registration failure so partial failures are
// never swallowed.
type AcceptFailure struct {
	SuggestionID string
	Owner        string
	Name         string
	Message      string
}

// AcceptResult is the outcome of a bulk default acceptance.
type AcceptResult struct {
	Accepted         []AcceptedTarget
	SkippedSensitive []Suggestion
	Failed           []AcceptFailure
	DryRun           bool
}

// SuggestionSource lists discovered target suggestions (registered/dismissed
// already filtered out by discovery). Backed by discovery.Service in the
// composition root.
type SuggestionSource interface {
	ListTargetSuggestions(ctx context.Context) ([]Suggestion, error)
}

// TargetCatalog reads and registers targets. Backed by targets.Service.
type TargetCatalog interface {
	List(ctx context.Context) ([]CatalogTarget, error)
	Register(ctx context.Context, in RegisterInput) (CatalogTarget, error)
}

// PlanCatalog reports which target ids are bound to at least one plan. Backed by
// plans.Service.
type PlanCatalog interface {
	PlannedTargetIDs(ctx context.Context) (map[string]struct{}, error)
}

// RunStatusSource reports the last successful backup time per target. Backed by
// runs.Service.
type RunStatusSource interface {
	LastSuccessByTarget(ctx context.Context, targetIDs []string) (map[string]time.Time, error)
}

// VerifiedSource reports the last verified-restore time per target. Backed by
// restores.Service.
type VerifiedSource interface {
	LastVerifiedByTarget(ctx context.Context, targetIDs []string) (map[string]time.Time, error)
}
