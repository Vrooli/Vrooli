// Package versions is the domain-scoped home for per-component version
// history (req 11). Cross-domain reference (component_id) is stored as
// a soft FK — no SQL constraint.
//
// Layering:
//
//	HTTP → handler → Service (Record / List / Get / Diff) → Repository (sqlite)
//
// types.go owns the domain entity + typed sentinels. repository.go
// owns persistence. service.go owns the record-on-change policy + the
// LCS line-diff used by DiffVersions.
package versions

import (
	"fmt"
	"time"
)

// Version is the internal domain shape for a recorded version row.
type Version struct {
	ID                    string
	ComponentID           string
	LibraryID             string
	Version               string
	Status                string
	SourcePath            string
	Content               string
	ContentSHA256         string
	ChangelogMD           string
	RecordedAt            time.Time
	CreatedAt             time.Time
	ReleasedAt            time.Time
	RequiredTokens        []string
	RequiredTokenPatterns []string
}

// RecordInput is the DTO the recorder hands to the service. The
// service computes ContentSHA256, derives Version from the `@version`
// header in Content, and decides whether a new row is inserted.
type RecordInput struct {
	ComponentID string
	// Content is the post-save TSX body. The recorder parses
	// `@version` out of it to populate Version.
	Content string
	// ChangelogMD is an optional human note. Defaults to "auto-recorded
	// on save" when empty.
	ChangelogMD string
}

// ListQuery filters and bounds a List call.
type ListQuery struct {
	ComponentID string
	Limit       int
}

// ErrVersionNotFound is returned when no row matches the requested
// (component_id, version) pair. Maps to NotFound at the transport.
type ErrVersionNotFound struct {
	ComponentID string
	Version     string
}

// ErrVersionExists means a content save attempted to record a version label
// that already belongs to a different content hash. The recorder must surface
// this collision instead of treating the failed INSERT as a swallowed no-op.
type ErrVersionExists struct {
	ComponentID string
	Version     string
}

func (e ErrVersionExists) Error() string {
	return fmt.Sprintf("version %q for component %q already exists with different content", e.Version, e.ComponentID)
}

func (e ErrVersionNotFound) Error() string {
	return fmt.Sprintf("version %q for component %q not found", e.Version, e.ComponentID)
}

// ErrInvalidVersion signals a malformed RecordInput — empty
// component_id or empty content. Maps to InvalidArgument.
type ErrInvalidVersion struct {
	Field  string
	Reason string
}

func (e ErrInvalidVersion) Error() string {
	return fmt.Sprintf("invalid version: %s — %s", e.Field, e.Reason)
}
