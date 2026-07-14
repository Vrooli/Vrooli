// Package adoptions is the domain-scoped home for the adoption registry
// — soft links from a library component (by id) to a concrete copy of
// its source under a target scenario's tree. Cross-domain references
// (to components.id) are stored as IDs only, no SQL constraint.
//
// Layering:
//
//	HTTP → handler → Service (drift compute) → Repository (sqlite)
//	                       ↑                          ↑
//	                       FakeService (handler tests) FakeRepository (service tests)
//	                                                    Real sqlite (repository tests)
//
// types.go owns the domain entity + typed sentinels. repository.go
// owns persistence. service.go owns the Refresh policy (current /
// behind / modified / unknown). The scenarios-tree resolver and the
// components-content reader are seams the service depends on; tests
// inject fakes.
package adoptions

import (
	"fmt"
	"time"
)

// LibraryVersionStatus tracks whether the adopted version is still the
// component's current library release.
type LibraryVersionStatus string

const (
	LibraryVersionStatusEmpty      LibraryVersionStatus = ""
	LibraryVersionStatusCurrent    LibraryVersionStatus = "current"
	LibraryVersionStatusBehind     LibraryVersionStatus = "behind"
	LibraryVersionStatusDeprecated LibraryVersionStatus = "deprecated"
	LibraryVersionStatusMissing    LibraryVersionStatus = "missing"
	LibraryVersionStatusUnknown    LibraryVersionStatus = "unknown"
)

func (s LibraryVersionStatus) Valid() bool {
	switch s {
	case LibraryVersionStatusEmpty, LibraryVersionStatusCurrent, LibraryVersionStatusBehind, LibraryVersionStatusDeprecated, LibraryVersionStatusMissing, LibraryVersionStatusUnknown:
		return true
	}
	return false
}

// LocalStatus tracks whether the adopted file still matches its source
// version or has local scenario edits.
type LocalStatus string

const (
	LocalStatusEmpty    LocalStatus = ""
	LocalStatusClean    LocalStatus = "clean"
	LocalStatusModified LocalStatus = "modified"
	LocalStatusMissing  LocalStatus = "missing"
	LocalStatusUnknown  LocalStatus = "unknown"
)

func (s LocalStatus) Valid() bool {
	switch s {
	case LocalStatusEmpty, LocalStatusClean, LocalStatusModified, LocalStatusMissing, LocalStatusUnknown:
		return true
	}
	return false
}

// Adoption is the internal domain shape for an adoption record. The
// wire/proto type lives at the transport edge; this struct is the only
// shape internal callers depend on.
type Adoption struct {
	ID                    string
	ComponentID           string
	LibraryID             string
	Scenario              string
	AdoptedPath           string
	AdoptedVersion        string
	SourceSHA256          string
	AdoptedSnapshotSHA256 string
	LibraryVersionStatus  LibraryVersionStatus
	LocalStatus           LocalStatus
	StatusDetail          string
	CreatedAt             time.Time
	RefreshedAt           time.Time
	AppliedAt             time.Time
	// DriftBacklogRef is the swarm-manager backlog item ("<kind>/<name>")
	// filed by Refresh when this adoption first transitioned to
	// behind/modified. Cleared back to "" when status returns to current
	// so a future drift files a fresh item.
	DriftBacklogRef string
}

// CreateInput is the explicit DTO the service hands to the repository
// on create. ID is optional; callers that need to stamp the same ID
// into generated provenance comments may reserve one up front.
type CreateInput struct {
	ID                    string
	ComponentID           string
	LibraryID             string
	Scenario              string
	AdoptedPath           string
	AdoptedVersion        string
	SourceSHA256          string
	AdoptedSnapshotSHA256 string
}

type ApplyInput struct {
	ComponentID        string
	Scenario           string
	AdoptedPath        string
	Version            string
	ConfirmOverwrite   bool
	OverrideValidation bool
}

type ReapplyInput struct {
	ID                    string
	Version               string
	ConfirmLocalOverwrite bool
	OverrideValidation    bool
}

// AppliedSnapshotUpdate records the exact library version and bytes
// last written into the scenario file during apply/reapply.
type AppliedSnapshotUpdate struct {
	ID                    string
	AdoptedVersion        string
	SourceSHA256          string
	AdoptedSnapshotSHA256 string
	AppliedAt             time.Time
}

// RefreshUpdate carries the per-row update Refresh writes back to the
// repository after computing drift. Repository merges these fields
// onto the existing row without touching CreatedAt / ComponentID /
// Scenario / AdoptedPath / AdoptedVersion.
type RefreshUpdate struct {
	ID                   string
	LibraryVersionStatus LibraryVersionStatus
	LocalStatus          LocalStatus
	StatusDetail         string
	RefreshedAt          time.Time
	// DriftBacklogRef is set when Refresh successfully filed a backlog
	// item via swarm-manager. Empty string means "leave the existing
	// value untouched"; the explicit empty-clear path uses
	// ClearDriftBacklogRef below so the repository can tell the
	// difference between "no opinion" and "clear it".
	DriftBacklogRef      string
	ClearDriftBacklogRef bool
}

// ListQuery filters a List call. All fields optional.
type ListQuery struct {
	ComponentID string
	Scenario    string
	Limit       int
}

// ErrAdoptionNotFound is the typed sentinel handlers translate to a
// 404 via errors.As.
type ErrAdoptionNotFound struct {
	ID string
}

func (e ErrAdoptionNotFound) Error() string {
	return fmt.Sprintf("adoption %q not found", e.ID)
}

// ErrInvalidAdoption is returned by the service when the caller's
// input does not satisfy the create contract. Field names the
// offending field; Reason is a human-safe explanation.
type ErrInvalidAdoption struct {
	Field  string
	Reason string
}

func (e ErrInvalidAdoption) Error() string {
	return fmt.Sprintf("invalid adoption: %s: %s", e.Field, e.Reason)
}

// ErrAdoptionValidationBlocked means the server-side dependency gate returned
// a blocking verdict. It is distinct from malformed input so transports can
// make the operator explicitly acknowledge the required override.
type ErrAdoptionValidationBlocked struct {
	ComponentID string
	Scenario    string
	Version     string
}

func (e ErrAdoptionValidationBlocked) Error() string {
	return fmt.Sprintf("adoption validation blocked %q@%q for scenario %q; set override_validation to continue", e.ComponentID, e.Version, e.Scenario)
}
