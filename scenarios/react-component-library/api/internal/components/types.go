// Package components is the domain-scoped home for the component
// registry — the indexed view of TSX files on disk that declare a
// `@libraryId` header. Cross-cutting concerns (versions, adoptions,
// deps, themes) live in sibling internal/<dom>/ packages and reference
// components by ID with no hard FK.
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
// indexer.go walks the filesystem (via api-core/storage) and parses
// the canonical header comment block.
package components

import (
	"fmt"
	"time"
)

// Component is the internal domain shape for an indexed component.
// The wire/proto type lives at the transport edge; this struct is the
// only shape internal callers depend on.
type Component struct {
	ID          string
	LibraryID   string
	DisplayName string
	Description string
	SourcePath  string
	Version     string
	Tags        []string
	IndexedAt   time.Time
	UpdatedAt   time.Time
	Headers     map[string]string
}

// UpsertInput is the explicit DTO the indexer hands to the service.
// Distinct from Component so the service owns ID assignment and
// IndexedAt/UpdatedAt stamping (Repository.Upsert generates them).
type UpsertInput struct {
	LibraryID   string
	DisplayName string
	Description string
	SourcePath  string
	Version     string
	Tags        []string
	Headers     map[string]string
}

// SearchQuery filters a List call. All fields optional.
// Match is a case-insensitive substring tested against library_id,
// display_name, description, and source_path. Tag matches require an
// exact (case-insensitive) hit in the comma-separated tags column.
type SearchQuery struct {
	Match string
	Tag   string
	Limit int
}

// ErrComponentNotFound is the typed sentinel handlers translate to a
// 404 via errors.As.
type ErrComponentNotFound struct {
	IDOrLibraryID string
}

func (e ErrComponentNotFound) Error() string {
	return fmt.Sprintf("component %q not found", e.IDOrLibraryID)
}

// ErrInvalidHeader is returned by the indexer/service when a header
// block fails to satisfy the @libraryId contract documented in the
// scenario PRD. Field names the offending header field; Reason is a
// human-safe explanation.
type ErrInvalidHeader struct {
	SourcePath string
	Field      string
	Reason     string
}

func (e ErrInvalidHeader) Error() string {
	return fmt.Sprintf("%s: header field %s: %s", e.SourcePath, e.Field, e.Reason)
}
