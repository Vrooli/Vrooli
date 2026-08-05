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

	"react-component-library/internal/components"
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
	Files           []AdoptionFile
}

// AdoptionFile is the per-file provenance and snapshot for a vendored unit.
// The parent fields remain the entry-file mirror for compatibility.
type AdoptionFile struct {
	LibraryPath           string
	AdoptedPath           string
	SourceSHA256          string
	AdoptedSnapshotSHA256 string
	// SourceAsset* is immutable per-file attribution. A root adoption can
	// materialize dependency assets, so the parent Adoption.ComponentID is not
	// sufficient to establish which catalog asset supplied this file.
	SourceAssetID   string
	SourceLibraryID string
	SourceVersion   string
}

// EffectiveAdoption is a provenance-backed asset use. ParentAdoption remains
// the direct operator action; Mediated distinguishes a dependency file from a
// root file supplied by that same parent.
type EffectiveAdoption struct {
	SourceAssetID   string
	SourceLibraryID string
	SourceVersion   string
	Mediated        bool
	ParentAdoption  Adoption
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
	Files                 []AdoptionFile
}

type ApplyInput struct {
	ComponentID        string
	Scenario           string
	AdoptedPath        string
	Version            string
	ConfirmOverwrite   bool
	OverrideValidation bool
	ReplaceExisting    bool
	IncludeSuggestions []string
}

// ApplyResult keeps an adoption write and its immediate consumer evidence
// together. ImportSites is intentionally computed after the write so callers
// receive the locations that now consume the adopted target.
type ApplyResult struct {
	Adoption             Adoption
	WrittenPath          string
	ExperiencePath       string
	ImportSites          []string
	StyleFitAffinity     components.DesignAffinity
	StyleFitDetail       string
	CopiedAssets         []string
	SatisfiedPorts       []string
	AvailableSuggestions []string
}

type ReapplyInput struct {
	ID                    string
	Version               string
	ConfirmLocalOverwrite bool
	OverrideValidation    bool
}

// ReconcileInput intentionally defaults to dry-run. Apply is the only mode
// that writes adoption records; scenario source trees are always read-only.
type ReconcileInput struct{ Apply bool }

type ReconcileFinding struct {
	Scenario    string
	AdoptedPath string
	LibraryID   string
	Version     string
	Detail      string
}

type ReconcileResult struct {
	Scanned         int
	AlreadyRecorded int
	Created         int
	// Healed counts already-recorded rows whose snapshot was captured from a
	// non-pristine copy (read CLEAN but the bytes are not the library's) and was
	// re-derived from the library so the row now reads MODIFIED honestly. Only
	// apply mode heals; dry-run leaves records untouched.
	Healed   int
	Findings []ReconcileFinding
}

// RebaselineInput corrects the recorded pristine snapshot of an existing
// adoption without re-applying library bytes to disk and without forcing the row
// to current/clean (unlike AppliedSnapshotUpdate). It is the heal seam for
// records whose snapshot was captured from a locally-modified copy — making a
// modified file masquerade as CLEAN. The caller supplies honest snapshots and
// recomputes drift status afterward.
type RebaselineInput struct {
	ID                    string
	AdoptedSnapshotSHA256 string
	Files                 []AdoptionFile
}

// ReconvergeInput drives the batch drift-reconverge flow. It defaults to
// dry-run; Apply is the only mode that writes scenario source files, and it
// does so through the same Reapply primitive so server-side Apply validation
// is never bypassed.
type ReconvergeInput struct {
	// Scenario optionally restricts reconverge to a single adopter scenario.
	// Empty processes every BEHIND adoption in the registry.
	Scenario string
	Apply    bool
}

// ReconvergeAction is the per-adoption disposition of a reconverge pass.
type ReconvergeAction string

const (
	// ReconvergeActionReapplied — the copy was BEHIND and CLEAN and was
	// re-applied to the current library version.
	ReconvergeActionReapplied ReconvergeAction = "reapplied"
	// ReconvergeActionWouldReapply — dry-run disposition for a BEHIND + CLEAN
	// copy that would be re-applied under --apply.
	ReconvergeActionWouldReapply ReconvergeAction = "would_reapply"
	// ReconvergeActionFlaggedModified — BEHIND but the copy carries local edits;
	// reconverge flags it for human review and never overwrites it.
	ReconvergeActionFlaggedModified ReconvergeAction = "flagged_modified"
	// ReconvergeActionSkippedUnresolved — BEHIND but the copy is missing or its
	// status is otherwise unresolved; left untouched for human review.
	ReconvergeActionSkippedUnresolved ReconvergeAction = "skipped_unresolved"
	// ReconvergeActionError — an apply-mode re-apply attempt failed (e.g. a
	// blocking dependency verdict, or an unresolved library version).
	ReconvergeActionError ReconvergeAction = "error"
)

// ReconvergeFileOutcome is the per-file drift disposition inside one adoption.
type ReconvergeFileOutcome struct {
	LibraryPath string
	AdoptedPath string
	LocalStatus LocalStatus
}

// ReconvergeOutcome is the per-adoption result of a reconverge pass.
type ReconvergeOutcome struct {
	AdoptionID           string
	Scenario             string
	ComponentID          string
	LibraryID            string
	AdoptedVersion       string
	TargetVersion        string
	LibraryVersionStatus LibraryVersionStatus
	LocalStatus          LocalStatus
	Action               ReconvergeAction
	Detail               string
	Files                []ReconvergeFileOutcome
}

// ReconvergeResult is the outcome rollup of a reconverge pass.
type ReconvergeResult struct {
	Scanned   int
	Behind    int
	Reapplied int
	Flagged   int
	Skipped   int
	Errored   int
	Outcomes  []ReconvergeOutcome
}

// DiscoverInput drives the content-similarity discovery pass. Discovery is
// always read-only; ConfirmDiscovery performs the only write.
type DiscoverInput struct {
	// Scenario optionally restricts the scan to one scenario UI tree. Empty
	// scans every scenario.
	Scenario string
	// MinSimilarity is the Sørensen–Dice threshold in [0,1] a candidate must
	// meet to surface. Zero uses DefaultDiscoverThreshold.
	MinSimilarity float64
	// Limit caps returned candidates. Zero uses defaultListLimit.
	Limit int
}

// DiscoveryCandidate is one header-less scenario file matched to a library
// version, with the similarity evidence an operator reviews before confirming.
type DiscoveryCandidate struct {
	Scenario       string
	AdoptedPath    string
	ComponentID    string
	LibraryID      string
	Version        string
	DisplayName    string
	Similarity     float64
	SharedLines    int
	CandidateLines int
	SourceLines    int
	BasenameMatch  bool
	Evidence       []string
}

// DiscoverResult is the outcome of a discovery pass.
type DiscoverResult struct {
	Scanned       int
	MinSimilarity float64
	Candidates    []DiscoveryCandidate
}

// ConfirmDiscoveryInput names the exact component + version to attribute to a
// header-less file (echoed from a DiscoveryCandidate). The service re-reads the
// file, verifies it is still header-less, injects the provenance header, and
// creates the adoption record.
type ConfirmDiscoveryInput struct {
	Scenario    string
	AdoptedPath string
	ComponentID string
	Version     string
}

// ConfirmDiscoveryResult carries the created record, the written path, and the
// similarity that was recomputed at confirm time (durable evidence).
type ConfirmDiscoveryResult struct {
	Adoption    Adoption
	WrittenPath string
	Similarity  float64
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

type AppliedUnitUpdate struct {
	AppliedSnapshotUpdate
	Files []AdoptionFile
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
