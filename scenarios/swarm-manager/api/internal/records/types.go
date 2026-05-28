// Package records is the write-side of the recursive-learning loop:
// narrative artifacts of completed work (bug fixes, features, refactors,
// investigations). Records sit alongside backlog and captures; they are
// immutable once their narrative is filled.
//
// DOC: docs/concepts/ARCHITECTURE.md
// DOC: docs/internal/SEAMS.md
package records

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// RecordKind mirrors backlog.BacklogKind by string identity. An anti-drift
// test asserts the two enums stay equal; see types_test.go.
type RecordKind string

const (
	KindIdea     RecordKind = "idea"
	KindResearch RecordKind = "research"
	KindFix      RecordKind = "fix"
	KindExecute  RecordKind = "execute"
	KindChore    RecordKind = "chore"
)

// AllKinds is the canonical enumeration.
var AllKinds = []RecordKind{KindIdea, KindResearch, KindFix, KindExecute, KindChore}

// ParseKind validates a raw kind string against AllKinds (case-insensitive).
func ParseKind(raw string) (RecordKind, error) {
	candidate := RecordKind(strings.ToLower(strings.TrimSpace(raw)))
	for _, k := range AllKinds {
		if candidate == k {
			return k, nil
		}
	}
	return "", fmt.Errorf("invalid record kind %q (expected one of %v)", raw, AllKinds)
}

// Outcome classifies how the work concluded.
type Outcome string

const (
	OutcomeShipped    Outcome = "shipped"
	OutcomePartial    Outcome = "partial"
	OutcomeAbandoned  Outcome = "abandoned"
	OutcomeDuplicate  Outcome = "duplicate"
)

// AllOutcomes is the canonical enumeration.
var AllOutcomes = []Outcome{OutcomeShipped, OutcomePartial, OutcomeAbandoned, OutcomeDuplicate}

// ParseOutcome validates a raw outcome string (case-insensitive).
func ParseOutcome(raw string) (Outcome, error) {
	candidate := Outcome(strings.ToLower(strings.TrimSpace(raw)))
	for _, o := range AllOutcomes {
		if candidate == o {
			return o, nil
		}
	}
	return "", fmt.Errorf("invalid outcome %q (expected one of %v)", raw, AllOutcomes)
}

// Record is a narrative artifact of completed work. Once Stub flips false,
// every field except SupersededBy is immutable; amendments require a new
// record with Supersedes set to this record's ID.
type Record struct {
	ID           string     `json:"id"`
	Kind         RecordKind `json:"kind"`
	Scenario     string     `json:"scenario"`
	BacklogRef   string     `json:"backlog_ref,omitempty"`
	Supersedes   string     `json:"supersedes,omitempty"`
	SupersededBy string     `json:"superseded_by,omitempty"`
	Trigger      string     `json:"trigger"`
	Approach     string     `json:"approach"`
	RuledOut     []string   `json:"ruled_out,omitempty"`
	Commit       string     `json:"commit,omitempty"`
	FilesChanged []string   `json:"files_changed,omitempty"`
	Outcome      Outcome    `json:"outcome"`
	Stub         bool       `json:"stub"`
	CreatedAt    time.Time  `json:"created_at"`
	CreatedBy    string     `json:"created_by,omitempty"`
	NarrativeAt  time.Time  `json:"narrative_at,omitempty"`
}

// ErrStubLocked is returned when an attempt is made to fill or edit a
// record's narrative after the stub has already been filled.
var ErrStubLocked = errors.New("record narrative is already filled; create a superseding record to amend")

// ErrSupersedeCycle is returned when a Supersede operation would create a
// cycle in the supersede chain.
var ErrSupersedeCycle = errors.New("supersede would create a cycle")

// ErrAlreadySuperseded is returned when a record already has SupersededBy set.
var ErrAlreadySuperseded = errors.New("record is already superseded")

// hasNarrative reports whether the record has non-empty narrative content.
// A record with empty Trigger AND empty Approach AND empty RuledOut is a
// stub; once any of these are non-empty it is considered filled.
func (r *Record) hasNarrative() bool {
	return strings.TrimSpace(r.Trigger) != "" ||
		strings.TrimSpace(r.Approach) != "" ||
		len(r.RuledOut) > 0
}

// EmbeddingText composes the text used for semantic indexing.
func (r *Record) EmbeddingText() string {
	parts := []string{r.Trigger, r.Approach}
	parts = append(parts, r.RuledOut...)
	return strings.TrimSpace(strings.Join(parts, "\n"))
}
