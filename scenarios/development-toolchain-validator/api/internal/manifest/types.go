// Package manifest owns the per-(skill_id, golden_slug) expected-diff
// manifest (OT-P0-003). A manifest is the input the diff evaluator
// consults at validation time to decide whether a skill's run was a
// pass, a mutation, or a failure.
//
// Layering mirrors the canonical Vrooli pattern:
//
//	HTTP → handler → Service (validates) → Repository (persists)
//	                     ↑                      ↑
//	                     FakeService            FakeRepository
//
// evaluator.go is a pure policy function — no I/O — so the decision
// boundary is exhaustively table-testable.
package manifest

import (
	"fmt"
	"time"
)

// ConvergenceTarget describes what re-running a converged (skill, golden)
// tuple must produce.
type ConvergenceTarget int

const (
	ConvergenceTargetUnspecified ConvergenceTarget = 0
	ConvergenceTargetNone        ConvergenceTarget = 1
	ConvergenceTargetEmptyDiff   ConvergenceTarget = 2
)

// ContentRule constrains the content of any file whose path matches
// PathGlob. MustContain and MustNotContain are AND-combined.
type ContentRule struct {
	PathGlob       string
	MustContain    []string
	MustNotContain []string
}

// Manifest is the domain shape for a stored expectation. Distinct from
// the proto wire type at packages/proto/gen/go/.../manifest.Manifest —
// handlers translate at the boundary so the domain layer never imports
// proto.
type Manifest struct {
	SkillID               string
	GoldenSlug            string
	AllowedPaths          []string
	ContentRules          []ContentRule
	WildcardAllowed       bool
	ConvergenceTarget     ConvergenceTarget
	TemplateVersionPinned string
	SkillVersionPinned    string
	UpdatedAt             time.Time
}

// UpsertInput is the explicit DTO Service.Upsert accepts. Distinct from
// Manifest so callers cannot accidentally smuggle an UpdatedAt the
// service has no way to honor.
type UpsertInput struct {
	SkillID               string
	GoldenSlug            string
	AllowedPaths          []string
	ContentRules          []ContentRule
	WildcardAllowed       bool
	ConvergenceTarget     ConvergenceTarget
	TemplateVersionPinned string
	SkillVersionPinned    string
}

// DiffFile is one entry in the diff that the evaluator consumes.
// Content is optional — only required when a ContentRule matches the
// path; resolved by the validation_run worker before invoking Evaluate.
type DiffFile struct {
	Path    string
	Content string
}

// VerdictKind classifies the evaluator's decision over a diff.
type VerdictKind int

const (
	VerdictUnspecified VerdictKind = 0

	// VerdictPass means every diff entry matches an allowed path
	// (or wildcard) and every applicable content rule was satisfied.
	VerdictPass VerdictKind = 1

	// VerdictUnexpectedMutation means at least one diff entry fell
	// outside the manifest's allowed surface (path or content).
	VerdictUnexpectedMutation VerdictKind = 2
)

// Verdict is the evaluator's structured decision. Violations enumerates
// every reason a non-PASS verdict was returned so callers (CLI report,
// validation_record) can render them.
type Verdict struct {
	Kind       VerdictKind
	Violations []Violation
}

// Violation is one reason a diff entry failed manifest evaluation.
type Violation struct {
	Path   string
	Reason string
}

// ErrManifestNotFound is the typed sentinel returned when no row matches
// the (skill_id, golden_slug) lookup.
type ErrManifestNotFound struct {
	SkillID    string
	GoldenSlug string
}

func (e ErrManifestNotFound) Error() string {
	return fmt.Sprintf("manifest for skill=%q golden=%q not found", e.SkillID, e.GoldenSlug)
}

// ErrInvalidManifest is the typed sentinel returned when input
// validation fails. Handlers translate via errors.As into Connect's
// CodeInvalidArgument.
type ErrInvalidManifest struct {
	Field  string
	Reason string
}

func (e ErrInvalidManifest) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Reason)
}
