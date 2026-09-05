// Package handoffrules owns the capture-rule domain: named patterns that
// decide when the console OFFERS a handoff.
//
// A rule never sends anything. A match produces a suggestion the operator can
// dismiss; pressing it opens the same composer every other entry point opens.
// That is the safety property that makes operator-authored patterns
// shippable — a wrong rule costs a dismissed chip, never a message delivered
// to an agent — and it is why nothing in this package can reach the send path.
//
// DOC: docs/internal/STORAGE_AUDIT.md
// DOC: docs/internal/ROLES-AND-HANDOFFS-UX.md
package handoffrules

import (
	"errors"
	"time"
)

// Source names what a rule's pattern is matched against.
const (
	// SourceFilePath matches a glob against paths a session mentioned.
	SourceFilePath = "file_path"
	// SourceMessageText matches a regular expression against message text.
	// The first capture group becomes the payload, or the whole match when
	// the pattern has no group.
	SourceMessageText = "message_text"
)

// ErrInvalidRule is returned for caller-visible validation failures: a blank
// name, a blank pattern, or an unknown source. Handlers map it to
// CodeInvalidArgument.
var ErrInvalidRule = errors.New("handoffrules: invalid rule")

// Rule decides when a handoff suggestion appears.
//
// Nothing here names a workflow. A rule knows a pattern and where a match may
// render; it does not know that a path is a plan, and it has no field that
// could say so.
type Rule struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	Source  string `json:"source"`
	Pattern string `json:"pattern"`
	// Surfaces names where a match may render. An empty list means every
	// surface that knows how to render a suggestion.
	Surfaces  []string `json:"surfaces"`
	SortOrder int      `json:"sort_order"`
	CreatedAt string   `json:"created_at,omitempty"`
	UpdatedAt string   `json:"updated_at,omitempty"`
}

// UpsertRequest is the create-or-update payload. A blank ID is assigned by
// the store.
type UpsertRequest struct {
	ID        string
	Name      string
	Enabled   bool
	Source    string
	Pattern   string
	Surfaces  []string
	SortOrder int
}

// Validate rejects the caller mistakes the store must not persist. A blank
// pattern is rejected because a rule that matches everything would put a
// suggestion under every message, which reads as a broken console rather than
// as a rule the operator should edit.
func (r UpsertRequest) Validate() error {
	if r.Name == "" || r.Pattern == "" {
		return ErrInvalidRule
	}
	switch r.Source {
	case SourceFilePath, SourceMessageText:
		return nil
	default:
		return ErrInvalidRule
	}
}

// FormatTime returns a UTC RFC3339Nano string for the timestamp columns.
func FormatTime(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }
