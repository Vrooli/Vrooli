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
	LibraryVersionStatusEmpty         LibraryVersionStatus = ""
	LibraryVersionStatusCurrent       LibraryVersionStatus = "current"
	LibraryVersionStatusBehind        LibraryVersionStatus = "behind"
	LibraryVersionStatusDeprecated    LibraryVersionStatus = "deprecated"
	LibraryVersionStatusMissing       LibraryVersionStatus = "missing"
	LibraryVersionStatusUnknown       LibraryVersionStatus = "unknown"
	LibraryVersionStatusSourceDrifted LibraryVersionStatus = "source_drifted"
)

func (s LibraryVersionStatus) Valid() bool {
	switch s {
	case LibraryVersionStatusEmpty, LibraryVersionStatusCurrent, LibraryVersionStatusBehind, LibraryVersionStatusDeprecated, LibraryVersionStatusMissing, LibraryVersionStatusUnknown, LibraryVersionStatusSourceDrifted:
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

// ForkStatus records the operator's interpretation of local divergence. An
// explicitly declared fork is a durable product choice; reconverge must never
// overwrite it or silently reinterpret it as a clean copy.
type ForkStatus string

const (
	ForkStatusNone                  ForkStatus = ""
	ForkStatusDeclared              ForkStatus = "declared-fork"
	ForkStatusUnintendedDrift       ForkStatus = "unintended-drift"
	ForkStatusMechanicalTranslation ForkStatus = "mechanical-translation"
	ForkStatusContractPreserved     ForkStatus = "contract-preserved"
	ForkStatusLocalAddition         ForkStatus = "local-addition"
	ForkStatusLocalFork             ForkStatus = "local-fork"
)

func (s ForkStatus) Valid() bool {
	switch s {
	case ForkStatusNone, ForkStatusDeclared, ForkStatusUnintendedDrift, ForkStatusMechanicalTranslation, ForkStatusContractPreserved, ForkStatusLocalAddition, ForkStatusLocalFork:
		return true
	}
	return false
}

// AdoptionMode records how an adopter consumes the library asset.
type AdoptionMode string

const (
	AdoptionModeCopied  AdoptionMode = "copied"
	AdoptionModeLinked  AdoptionMode = "linked"
	AdoptionModeEjected AdoptionMode = "ejected"
)

func (m AdoptionMode) Valid() bool {
	return m == AdoptionModeCopied || m == AdoptionModeLinked || m == AdoptionModeEjected
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
	ForkStatus            ForkStatus
	ForkReason            string
	ExtensionPoints       []string
	Mode                  AdoptionMode
	// DriftBacklogRef is the swarm-manager backlog item ("<kind>/<name>")
	// filed by Refresh when this adoption first transitioned to
	// behind/modified. Cleared back to "" when status returns to current
	// so a future drift files a fresh item.
	DriftBacklogRef string
	// IncludeSuggestions records optional dependencies the operator accepted;
	// reapply carries the same choices forward.
	IncludeSuggestions []string
	Files              []AdoptionFile
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
	IncludeSuggestions    []string
	ForkReason            string
	ExtensionPoints       []string
	Mode                  AdoptionMode
	Files                 []AdoptionFile
}

type LinkInput struct {
	ComponentID     string
	Scenario        string
	Version         string
	ImportSubpath   string
	ConfirmExisting bool
}

type LinkResult struct {
	Adoption      Adoption
	PackagePath   string
	ImportSubpath string
	UpdatedFiles  []string
}

type EjectInput struct {
	ApplyInput
	Reason string
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
	ForkReason         string
	ExtensionPoints    []string
}

type BatchApplyItem struct {
	ComponentID        string
	Scenario           string
	AdoptedPath        string
	Version            string
	ConfirmOverwrite   bool
	OverrideValidation bool
	ReplaceExisting    bool
	IncludeSuggestions []string
	ForkReason         string
	ExtensionPoints    []string
}

type BatchApplyInput struct{ Items []BatchApplyItem }

type BatchApplyResult struct {
	Results            []ApplyResult
	SharedDependencies []string
}

// ErrBatchDependencyConflict identifies two roots that require the same
// dependency at different pinned versions. A batch cannot safely materialize
// both versions into one target tree, so the operator must split the batch.
type ErrBatchDependencyConflict struct {
	Dependency    string
	FirstRoot     string
	FirstVersion  string
	SecondRoot    string
	SecondVersion string
}

func (e ErrBatchDependencyConflict) Error() string {
	return fmt.Sprintf("batch dependency %s is pinned by %s at %s and by %s at %s", e.Dependency, e.FirstRoot, e.FirstVersion, e.SecondRoot, e.SecondVersion)
}

type PreflightInput struct {
	ComponentID string
	Scenario    string
	Version     string
}

type PreflightResult struct {
	ComponentID    string
	Scenario       string
	Version        string
	Verdict        AdoptionVerdict
	Tokens         TokenVerdict
	Dependency     string
	StyleFit       string
	StyleFitDetail string
	I18n           string
	Selectors      string
	Blocking       bool
}

// MaturityVerdict is the evidence-backed readiness projection consumed by
// adoption policy. The adoptions domain does not compute catalog maturity; it
// only consumes the catalogcoverage result through MaturityReader.
type MaturityVerdict struct {
	Achieved string
	Floor    string
}

// AdoptionVerdict is the single adoptability decision for one asset/target.
// The individual fields remain visible so a blocked operator can remediate
// the actual failing dimension instead of receiving a generic refusal.
type AdoptionVerdict struct {
	Dependency     string
	StyleFit       string
	StyleFitDetail string
	Tokens         TokenVerdict
	Version        components.ComponentVersionStatus
	Maturity       MaturityVerdict
	I18n           string
	Selectors      string
	Warnings       []string
}

func (v AdoptionVerdict) Blocking() bool {
	if !v.Tokens.Satisfied() || v.Version == components.VersionStatusArchived {
		return true
	}
	if v.Version == components.VersionStatusDraft {
		return true
	}
	if v.Maturity.Floor != "" && maturityRank(v.Maturity.Achieved) < maturityRank(v.Maturity.Floor) {
		return true
	}
	if v.I18n == "fail" || v.Selectors == "fail" {
		return true
	}
	return v.Dependency == "block" || v.StyleFit == string(components.DesignAffinityDiscouraged)
}

func maturityRank(value string) int {
	switch value {
	case "missing":
		return 0
	case "scaffolded":
		return 1
	case "implemented":
		return 2
	case "verified":
		return 3
	case "production-ready":
		return 4
	default:
		return -1
	}
}

// ErrAdoptionReadinessBlocked names a non-token adoptability failure.
type ErrAdoptionReadinessBlocked struct {
	ComponentID string
	Scenario    string
	Verdict     AdoptionVerdict
}

func readinessBlockedError(componentID, scenario string, verdict AdoptionVerdict) error {
	if verdict.Dependency == "block" || verdict.StyleFit == string(components.DesignAffinityDiscouraged) {
		return ErrAdoptionValidationBlocked{ComponentID: componentID, Version: "", Scenario: scenario}
	}
	return ErrAdoptionReadinessBlocked{ComponentID: componentID, Scenario: scenario, Verdict: verdict}
}

func (e ErrAdoptionReadinessBlocked) Error() string {
	return fmt.Sprintf("adoption %s into %s is not adoptable: version=%s maturity=%s (floor=%s) dependency=%s style=%s",
		e.ComponentID, e.Scenario, e.Verdict.Version, e.Verdict.Maturity.Achieved, e.Verdict.Maturity.Floor, e.Verdict.Dependency, e.Verdict.StyleFit)
}

type DeleteResult struct {
	AdoptionID           string
	RemovableFiles       []string
	RemovedFiles         []string
	RequiresConfirmation bool
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
	DryRun                bool
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

// ReconvergeDisposition explains why a BEHIND + MODIFIED copy cannot be
// auto-upgraded. Ambiguous changes are conservatively treated as forks.
type ReconvergeDisposition string

const (
	ReconvergeDispositionTranslationOnly   ReconvergeDisposition = "translation_only"
	ReconvergeDispositionLocalAddition     ReconvergeDisposition = "local_addition"
	ReconvergeDispositionLocalFork         ReconvergeDisposition = "local_fork"
	ReconvergeDispositionTokenBlocked      ReconvergeDisposition = "token_blocked"
	ReconvergeDispositionContractPreserved ReconvergeDisposition = "contract_preserved"
)

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
	// ReconvergeActionBlockedTokens — an apply-mode re-apply was stopped by
	// the target scenario's unsatisfied styling contract.
	ReconvergeActionBlockedTokens ReconvergeAction = "blocked_tokens"
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
	Disposition          ReconvergeDisposition
	ForkStatus           ForkStatus
	Detail               string
	Files                []ReconvergeFileOutcome
}

// ReconvergeResult is the outcome rollup of a reconverge pass.
type ReconvergeResult struct {
	Scanned         int
	Behind          int
	Reapplied       int
	Flagged         int
	TranslationOnly int
	LocalAddition   int
	LocalFork       int
	TokenBlocked    int
	Skipped         int
	Errored         int
	Outcomes        []ReconvergeOutcome
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
	ForkStatus           ForkStatus
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
