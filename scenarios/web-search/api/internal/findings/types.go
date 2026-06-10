package findings

import (
	"fmt"
	"time"
)

// Lifecycle states. Stored verbatim in findings.status.
const (
	StatusActive     = "active"
	StatusDisputed   = "disputed"
	StatusSuperseded = "superseded"
)

// Provenance of a finding. Stored verbatim in findings.source.
const (
	SourceManual = "manual"
	SourceL2     = "l2"
	SourceL3     = "l3"
)

// Mutation types recorded in finding_audit.mutation_type.
const (
	MutationCreate    = "create"
	MutationEdit      = "edit"
	MutationSupersede = "supersede"
	MutationFlag      = "flag"
	MutationPrune     = "prune"
	MutationResolve   = "resolve"
)

// Dispute resolutions accepted by Service.ResolveDispute.
const (
	// ResolutionKeep returns a DISPUTED finding to ACTIVE and clears its note.
	ResolutionKeep = "keep"
	// ResolutionSupersede retires a DISPUTED finding in favor of a replacement.
	ResolutionSupersede = "supersede"
)

// Finding is one citation-backed claim in the knowledge store.
type Finding struct {
	ID            string
	Claim         string
	BriefID       string
	Confidence    float64
	Status        string
	RetrievalDate time.Time
	Query         string
	SupersededBy  string
	DisputeNote   string
	Source        string
	Citations     []Citation
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Citation is one cited source backing a finding.
type Citation struct {
	ID          string
	URL         string
	Title       string
	RetrievedAt time.Time
}

// Usage is the usage-telemetry counter for one finding (OT-P2-001): how often
// it was surfaced (returned by a search), how often it was explicitly marked
// used, and when it was last surfaced. A finding with no usage row is treated
// as the zero Usage (never surfaced). Kept separate from Finding so surfacing
// events never mutate the provenance-bearing finding row.
type Usage struct {
	FindingID      string
	SurfacedCount  int
	UsedCount      int
	LastSurfacedAt time.Time
}

// Brief is the research artifact (L2/L3) a finding was distilled from.
type Brief struct {
	ID           string
	Query        string
	Level        string
	Summary      string
	AgentRunID   string
	RunTimestamp time.Time
	Metadata     string
}

// NewCitation is a citation supplied when adding a finding.
type NewCitation struct {
	URL   string
	Title string
}

// NewFinding is the input DTO Service.Add accepts. Distinct from Finding so
// callers cannot pass ID, status, or timestamps.
type NewFinding struct {
	Claim      string
	Confidence float64
	Query      string
	Source     string
	BriefID    string
	Citations  []NewCitation
}

// EditInput is the input DTO Service.Edit accepts. Both fields overwrite.
type EditInput struct {
	Claim      string
	Confidence float64
}

// ListFilter narrows ListFindings. Status "" means "all visible" (active +
// disputed, superseded excluded unless IncludeArchived).
type ListFilter struct {
	Status          string
	IncludeArchived bool
	Limit           int
}

// ErrFindingNotFound is returned when no row matches. Handlers translate via
// errors.As into a 404.
type ErrFindingNotFound struct {
	ID string
}

func (e ErrFindingNotFound) Error() string {
	return fmt.Sprintf("finding %q not found", e.ID)
}

// ErrInvalidFinding is returned on validation failure. Handlers translate via
// errors.As into a 400.
type ErrInvalidFinding struct {
	Field  string
	Reason string
}

func (e ErrInvalidFinding) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Reason)
}

// ValidStatus reports whether s is a recognized lifecycle state.
func ValidStatus(s string) bool {
	switch s {
	case StatusActive, StatusDisputed, StatusSuperseded:
		return true
	default:
		return false
	}
}

// normalizeSource maps an input source to a stored value, defaulting to manual.
func normalizeSource(s string) string {
	switch s {
	case SourceManual, SourceL2, SourceL3:
		return s
	default:
		return SourceManual
	}
}
