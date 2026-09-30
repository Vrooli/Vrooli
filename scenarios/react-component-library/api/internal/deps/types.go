// Package deps owns parsed `@deps` declarations from component headers
// and the semver-intersection logic that turns them into adoption
// verdicts (ok / warn / block) against a target scenario's
// package.json.
//
// Layering:
//
//	HTTP → handler → Service.ValidateAdoption → Repository (sqlite)
//	                                             ↑
//	                       FakeRepository (service tests) / mocks/.
package deps

import "fmt"

// Declaration is one parsed @deps entry. version_range is the raw
// semver range string the header carried (e.g., "^18.0.0"); we keep
// it textual so the verdict can echo it back to the operator.
type Declaration struct {
	ComponentID  string
	LibraryID    string
	Version      string
	DepName      string
	VersionRange string
	Kind         DepKind
}

// SyncInput is the per-component DTO the indexer hands to the
// repository's Sync call. Replaces the stored declarations for the
// component atomically (delete-then-insert).
type SyncInput struct {
	ComponentID  string
	LibraryID    string
	Version      string
	Declarations []DeclarationFields
}

// DeclarationFields is the slim payload of one declaration during a
// Sync (component_id + library_id are taken from SyncInput).
type DeclarationFields struct {
	DepName      string
	VersionRange string
	Version      string
	Kind         DepKind
}

type DepKind string

const (
	DepKindRuntime DepKind = "runtime"
	DepKindPeer    DepKind = "peer"
	DepKindDev     DepKind = "dev"
)

// VerdictKind is the high-level adoption outcome. Stored as a wire
// string so the proto enum at the edge maps trivially.
type VerdictKind string

const (
	VerdictOK    VerdictKind = "ok"
	VerdictWarn  VerdictKind = "warn"
	VerdictBlock VerdictKind = "block"
)

// IssueKind describes one dep mismatch surfaced in a Verdict.
type IssueKind string

const (
	IssueMissingDep        IssueKind = "missing_dep"
	IssueRangeDoesNotMatch IssueKind = "range_does_not_match"
	IssueIncompatibleMajor IssueKind = "incompatible_major"
	IssueUnparseableRange  IssueKind = "unparseable_range"
	IssueUnparseableTarget IssueKind = "unparseable_target"
)

// Issue is one row in the verdict's findings — the UI renders these
// in a table alongside the high-level Kind.
type Issue struct {
	DepName         string
	DeclaredRange   string
	ScenarioVersion string
	Version         string
	DepKind         DepKind
	Kind            IssueKind
	Detail          string
}

// Severity returns the per-issue severity so the service can fold
// multiple issues into a single VerdictKind (worst-of wins).
func (i Issue) Severity() VerdictKind {
	if i.Kind == IssueMissingDep && i.DepKind == DepKindPeer {
		return VerdictBlock
	}
	switch i.Kind {
	case IssueIncompatibleMajor, IssueUnparseableTarget:
		return VerdictBlock
	default:
		return VerdictWarn
	}
}

// Verdict is the service-layer result of ValidateAdoption.
type Verdict struct {
	Kind   VerdictKind
	Issues []Issue
}

// ErrComponentNotIndexed is the typed sentinel ValidateAdoption returns
// when the requested component has no declaration rows. Callers can
// translate to a clearer UI message ("re-run components index").
type ErrComponentNotIndexed struct {
	ComponentID string
}

func (e ErrComponentNotIndexed) Error() string {
	return fmt.Sprintf("no dep declarations for component %q (run components index)", e.ComponentID)
}

// ErrScenarioPackageJSONMissing is returned by ValidateAdoption when
// the target scenario's package.json cannot be read.
type ErrScenarioPackageJSONMissing struct {
	Scenario string
	Cause    error
}

func (e ErrScenarioPackageJSONMissing) Error() string {
	return fmt.Sprintf("scenario %q package.json missing: %v", e.Scenario, e.Cause)
}

func (e ErrScenarioPackageJSONMissing) Unwrap() error { return e.Cause }
