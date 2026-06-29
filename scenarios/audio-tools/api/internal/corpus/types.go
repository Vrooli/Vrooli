// Package corpus is audio-tools' speech-eval corpus store: per-clip
// metadata (reference transcript, tags, duration, blob key) in a SQLite
// `corpus` domain, with the audio BYTES held separately in the blob store
// under the git-ignored runtime data dir. It is the substrate the eval
// harness (internal/eval) replays against. Operator-facing and local-only
// in v1 — not exposed cross-scenario.
package corpus

import (
	"context"
	"fmt"
	"time"
)

// Source records how a clip was captured: free-form dictation, or a
// scripted prompt read aloud. It is a soft enum stored as text.
type Source string

const (
	SourceFreeForm Source = "free_form"
	SourceScripted Source = "scripted"
)

// Valid reports whether s is a known source. The empty string is treated
// as free_form by Normalize.
func (s Source) Valid() bool { return s == SourceFreeForm || s == SourceScripted }

// Normalize coerces an empty/unknown source to the free_form default.
func (s Source) Normalize() Source {
	if s.Valid() {
		return s
	}
	return SourceFreeForm
}

// Clip is one corpus item's metadata. The audio bytes are NOT here — they
// live in the blob store under BlobKey.
type Clip struct {
	ID            string
	ReferenceText string
	Tags          []string
	DurationMs    int64
	SampleRateHz  int
	Format        string
	BlobKey       string
	Source        Source
	CreatedAt     time.Time
}

// ListFilter narrows a List query. TagContains matches clips whose tags
// JSON contains the substring (a cheap LIKE; tags are short).
type ListFilter struct {
	TagContains string
	Limit       int
	Offset      int
}

// ErrClipNotFound is returned when a clip id has no row.
type ErrClipNotFound struct{ ID string }

func (e ErrClipNotFound) Error() string { return fmt.Sprintf("corpus: clip %q not found", e.ID) }

// Repository is the corpus metadata persistence seam. Production is the
// SQLite impl in sqlite.go; tests substitute fakes.
type Repository interface {
	Create(ctx context.Context, c Clip) (Clip, error)
	Get(ctx context.Context, id string) (Clip, error)
	List(ctx context.Context, filter ListFilter) ([]Clip, error)
	Delete(ctx context.Context, id string) error
}
