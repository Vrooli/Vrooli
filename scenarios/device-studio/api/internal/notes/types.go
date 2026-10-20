// Package notes is the domain-scoped home for the notes resource.
//
// Layering inside this package mirrors the canonical Vrooli pattern
// (agent-manager, swarm-manager, etc. all use domain-scoped packages
// rather than a generic services/ directory):
//
//	HTTP → handler → Service (validates, applies defaults) → Repository (persists)
//	                     ↑                                       ↑
//	                     FakeService (handler tests)              FakeRepository (service tests)
//	                                                              Real sqlite (repository tests)
//
// types.go owns the domain entity and the typed sentinels handlers
// translate at the transport edge. repository.go owns the persistence
// seam. service.go owns the application surface (validation, defaults,
// cross-handler policy). The proto wire types live one floor up
// (packages/proto/...) and never import this package; the handler is
// the only translation point.
package notes

import (
	"fmt"
	"time"
)

// Note is the internal domain shape for a note. Distinct from the
// proto wire type at packages/proto/gen/go/.../v1/notes.Note — handlers
// translate at the boundary so the domain layer never imports proto
// (api-steer §7).
type Note struct {
	ID             string
	Title          string
	Body           string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	AttachmentKeys []string
}

// Attachment is the domain metadata record for a file uploaded through
// the notes multipart REST sub-resource. Bytes live in BlobStore; this
// type records the note-scoped handle needed by UI, CLI, and future APIs.
type Attachment struct {
	Key        string
	MIMEType   string
	SizeBytes  int64
	NoteID     string
	UploadedAt time.Time
}

// CreateInput is the explicit input DTO Service.Create accepts.
// Distinct from Note so callers cannot accidentally pass an ID or
// timestamp the service has no way to honour — those fields belong to
// the persistence layer (Repository.Create generates them).
//
// Establishing this convention here matters more than the boilerplate:
// future scenarios that copy this shape inherit the discipline of
// "service inputs are explicit DTOs, never partially-zeroed domain
// objects."
type CreateInput struct {
	Title string
	Body  string
}

// ErrNoteNotFound is the typed sentinel returned by Repository.Get
// (and propagated by Service.Get) when no row matches. Handlers
// translate via errors.As into a 404 response carrying
// httpx.CodeNotFound.
type ErrNoteNotFound struct {
	ID string
}

func (e ErrNoteNotFound) Error() string {
	return fmt.Sprintf("note %q not found", e.ID)
}

// ErrInvalidNote is the typed sentinel returned by Service.Create when
// validation fails. Field names the offending field; Reason is a
// human-safe explanation. Handlers translate via errors.As into a 400
// response carrying httpx.CodeInvalidRequest with envelope message
// "<field>: <reason>".
type ErrInvalidNote struct {
	Field  string
	Reason string
}

func (e ErrInvalidNote) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Reason)
}
