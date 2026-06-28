// Package planlog is the execution-log ledger domain. It is Plan Manager's
// single durable home for the typed work products an agent produces while
// executing a plan: design decisions, candidate findings, filed bug reports,
// reusable records, and lightweight notes. Findings, bug reports, and records
// are DISTINCT concepts (a finding is unvalidated; a bug_report is filed to the
// issue tracker; a record is reusable learning) and are never conflated.
//
// Layering mirrors the canonical domain pattern:
//
//	handler → Service → {Repository, Resolver, BugReporter, RecordWriter}
//	            ↑            ↑ home store  ↑ plans/exec  ↑ scenario-qa  ↑ swarm-manager
//	        (proto edge)   (all faked in tests; downstream forwarding degrades
//	                        gracefully — a failed sync leaves the entry durable
//	                        and retryable, never blocks plan execution)
//
// The structured LogEntry/LogSummary Go types live in the neutral planmodel
// kernel so the execution domain can read compact summaries through a seam
// without importing this package. The proto wire types live one floor up
// (handlers/planlog) and never import this package.
package planlog

import planmodel "plan-manager/internal/planmodel"

// Domain type aliases onto the shared planmodel kernel so the service signatures
// read in the log domain's vocabulary.
type (
	Entry         = planmodel.LogEntry
	EntryType     = planmodel.LogEntryType
	SyncStatus    = planmodel.LogSyncStatus
	Severity      = planmodel.LogSeverity
	Triage        = planmodel.FindingTriage
	DownstreamRef = planmodel.DownstreamRef
	Summary       = planmodel.LogSummary
	SummaryItem   = planmodel.LogSummaryItem
	Filter        = planmodel.LogFilter
)

// AddInputs are the common fields for creating any typed ledger entry. Type and
// type-specific fields (severity) are supplied by the typed Add* service methods.
type AddInputs struct {
	PlanOrExecution string
	PhaseID         string
	Title           string
	Detail          string
	Severity        Severity
	Evidence        []string
	SourceCommand   string
	IdempotencyKey  string
	RunID           string
}

// UpdateInputs are the mutable fields UpdateEntry can change. Empty string /
// unspecified enum leaves the field unchanged; AddEvidence appends.
type UpdateInputs struct {
	Title       string
	Detail      string
	Severity    Severity
	Triage      Triage
	AddEvidence []string
}

// NextActionKind classifies how strongly the wizard recommends an action.
type NextActionKind string

const (
	NextActionRecommended NextActionKind = "recommended"
	NextActionAlternative NextActionKind = "alternative"
	NextActionOptional    NextActionKind = "optional"
	NextActionRecovery    NextActionKind = "recovery"
)

// NextAction is one API-owned concrete action for the current guided step.
type NextAction struct {
	ID                 string
	Kind               NextActionKind
	Label              string
	Reason             string
	Argv               []string
	ContentPlaceholder string
	BlockedBy          []string
}

// GuidedStep is deterministic just-in-time steering for the current log action.
type GuidedStep struct {
	StepKind       string
	Title          string
	Summary        string
	Instructions   []string
	RequiredInputs []string
	Examples       []string
	CommonMistakes []string
	NextActions    []NextAction
}
