// Package targets is the domain-scoped home for backup targets: sources owned
// by other scenarios, self-registered idempotently and keyed by (owner, name).
//
// Layering mirrors the canonical Vrooli per-domain pattern:
//
//	Connect handler → Service (validates, decides upsert) → Repository (persists)
//	                      ↑                                      ↑
//	                      FakeService (handler tests)            FakeRepository (service tests)
//	                                                             real sqlite (repository tests)
//
// The proto wire types live one floor up (packages/proto/...) and never import
// this package; the handler is the only translation point (including proto
// SourceKind enum ↔ sources.SourceKind).
package targets

import (
	"fmt"
	"time"

	"data-backup-manager/internal/sources"
)

// Target is the internal domain shape for a registered backup source. Distinct
// from the proto wire type; handlers translate at the boundary so the domain
// layer never imports proto.
type Target struct {
	ID         string
	Owner      string
	Name       string
	SourceKind sources.SourceKind
	Locator    string
	Critical   bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// sameSpec reports whether two targets carry the same registerable spec
// (everything an owner controls on re-registration). Identity fields and
// timestamps are excluded — this is what makes a re-register a no-op.
func (t Target) sameSpec(other Target) bool {
	return t.SourceKind == other.SourceKind && t.Locator == other.Locator && t.Critical == other.Critical
}

// RegisterInput is the explicit DTO Service.Register accepts. Distinct from
// Target so callers cannot pass an ID or timestamp the service has no way to
// honour — those belong to the persistence layer.
type RegisterInput struct {
	Owner      string
	Name       string
	SourceKind sources.SourceKind
	Locator    string
	Critical   bool
}

// ErrTargetNotFound is the typed sentinel returned by Repository.GetByID /
// GetByOwnerName when no row matches. Handlers translate it into a 404 /
// connect.CodeNotFound.
type ErrTargetNotFound struct {
	// One of Owner+Name or ID is populated depending on the lookup.
	Owner string
	Name  string
	ID    string
}

func (e ErrTargetNotFound) Error() string {
	if e.ID != "" {
		return fmt.Sprintf("target %q not found", e.ID)
	}
	return fmt.Sprintf("target %s/%s not found", e.Owner, e.Name)
}

// ErrInvalidTarget is the typed sentinel returned by Service validation.
// Handlers translate it into a 400 / connect.CodeInvalidArgument carrying
// "<field>: <reason>".
type ErrInvalidTarget struct {
	Field  string
	Reason string
}

func (e ErrInvalidTarget) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Reason)
}
