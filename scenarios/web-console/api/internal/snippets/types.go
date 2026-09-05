// Package snippets owns reusable message text the sender owns. Nothing in
// this package knows about groups, roles, receivers, or payload semantics.
//
// DOC: docs/internal/STORAGE_AUDIT.md
// DOC: docs/internal/SNIPPETS-AND-MESSAGE-ACTIONS-UX.md
package snippets

import (
	"errors"
	"strings"
	"time"
)

var (
	// ErrInvalidSnippet identifies caller-visible validation failures.
	ErrInvalidSnippet = errors.New("snippets: invalid snippet")
	// ErrSnippetNotFound identifies a use of an id that does not exist.
	ErrSnippetNotFound = errors.New("snippets: snippet not found")
)

type Snippet struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Body       string `json:"body"`
	Color      string `json:"color"`
	Pinned     bool   `json:"pinned"`
	UseCount   int    `json:"use_count"`
	LastUsedAt string `json:"last_used_at,omitempty"`
	SortOrder  int    `json:"sort_order"`
	CreatedAt  string `json:"created_at,omitempty"`
	UpdatedAt  string `json:"updated_at,omitempty"`
}

type UpsertRequest struct {
	ID        string
	Name      string
	Body      string
	Color     string
	Pinned    bool
	HasPinned bool
	SortOrder int
}

func (r UpsertRequest) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return ErrInvalidSnippet
	}
	return nil
}

func FormatTime(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }
