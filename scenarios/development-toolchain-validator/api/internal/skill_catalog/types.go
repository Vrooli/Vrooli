// Package skill_catalog owns the local mirror of prompt-manager's skill
// catalog (OT-P0-002). Other DTV domains (manifest, validation_run,
// staleness, report) consult this package — never prompt-manager —
// when they need to reference a skill.
//
// Layering mirrors the canonical Vrooli pattern:
//
//	HTTP → handler → Service (sync + read) → Repository (persists)
//	                     ↑                      ↑
//	                     FakeService            FakeRepository
//
// The outbound seam that talks to prompt-manager is SkillCatalogSource;
// production wires the REST adapter from api/integrations/prompt_manager/.
package skill_catalog

import (
	"fmt"
	"time"
)

// Skill is the internal domain shape for a mirrored skill. Distinct
// from the proto wire type at packages/proto/gen/go/.../skill_catalog —
// handlers translate at the boundary so the domain layer never imports
// proto.
type Skill struct {
	ID          string
	Version     string
	ContentHash string
	SyncedAt    time.Time
}

// ErrSkillNotFound is the typed sentinel returned when no row matches.
// Handlers translate via errors.As into Connect's CodeNotFound.
type ErrSkillNotFound struct {
	ID string
}

func (e ErrSkillNotFound) Error() string {
	return fmt.Sprintf("skill %q not found", e.ID)
}

// ErrInvalidSkill is the typed sentinel returned by the service when
// input validation fails. Handlers translate via errors.As into Connect's
// CodeInvalidArgument.
type ErrInvalidSkill struct {
	Field  string
	Reason string
}

func (e ErrInvalidSkill) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Reason)
}

// ErrSyncFailed wraps upstream (prompt-manager) failures so the service
// layer can surface a structured error without leaking transport
// details. Handlers translate via errors.As into Connect's CodeInternal
// for transport faults and CodeUnavailable when discovery says the
// dependency is not running.
type ErrSyncFailed struct {
	Reason   string
	Wrapped  error
	NotReady bool // true when prompt-manager isn't running per discovery
}

func (e ErrSyncFailed) Error() string {
	if e.Wrapped != nil {
		return fmt.Sprintf("skill catalog sync: %s: %v", e.Reason, e.Wrapped)
	}
	return fmt.Sprintf("skill catalog sync: %s", e.Reason)
}

func (e ErrSyncFailed) Unwrap() error { return e.Wrapped }
