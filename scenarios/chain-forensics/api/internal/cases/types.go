package cases

import (
	"fmt"
	"time"

	chainforensics "github.com/vrooli/chain-forensics"
)

type Record struct {
	Case      chainforensics.Case
	CreatedAt time.Time
	UpdatedAt time.Time
}

type AnalystLabel struct {
	ID                 string
	CaseID             string
	Subject            string
	Label              string
	SourceURL          string
	SourceCitation     string
	Grade              chainforensics.Grade
	CounterExplanation string
	UnresolvedGap      string
	CreatedAt          time.Time
}

type PacketRecord struct {
	CaseID        string
	SchemaVersion string
	SHA256        string
	JSON          []byte
	CreatedAt     time.Time
}

// RevisionAdditions appends evidence and supports explicit summary/gap
// maintenance. Existing sources, claims, and observations cannot be replaced
// or removed. Edges change only through explicit, history-preserving revisions.
type RevisionAdditions struct {
	Summary         string                      `json:"summary,omitempty"`
	Sources         []chainforensics.Source     `json:"sources,omitempty"`
	Claims          []chainforensics.Claim      `json:"claims,omitempty"`
	Occurrences     []chainforensics.Occurrence `json:"occurrences,omitempty"`
	Edges           []chainforensics.Edge       `json:"edges,omitempty"`
	EdgeRevisions   []EdgeRevision              `json:"edge_revisions,omitempty"`
	Gaps            []string                    `json:"gaps,omitempty"`
	Counterexamples []string                    `json:"counterexamples,omitempty"`
	Disagreements   []string                    `json:"source_disagreements,omitempty"`
	ResolvedGaps    []string                    `json:"resolved_gaps,omitempty"`
}

type EdgeRevision struct {
	SupersedesEdgeID string              `json:"supersedes_edge_id"`
	Replacement      chainforensics.Edge `json:"replacement"`
}

type CaseRevision struct {
	ID                string
	CaseID            string
	PreviousUpdatedAt time.Time
	AppliedAt         time.Time
	Additions         RevisionAdditions
	PreviousCase      chainforensics.Case
	PreviousCreatedAt time.Time
}

type ErrNotFound struct{ ID string }

func (e ErrNotFound) Error() string { return fmt.Sprintf("case %q not found", e.ID) }

type ErrConflict struct{ ID string }

func (e ErrConflict) Error() string {
	return fmt.Sprintf("case %q already exists with different evidence", e.ID)
}

type ErrStaleRevision struct{ ID string }

func (e ErrStaleRevision) Error() string {
	return fmt.Sprintf("case %q changed since it was read; reload before revising", e.ID)
}

type ErrInvalid struct {
	Field  string
	Reason string
}

func (e ErrInvalid) Error() string { return fmt.Sprintf("%s: %s", e.Field, e.Reason) }
