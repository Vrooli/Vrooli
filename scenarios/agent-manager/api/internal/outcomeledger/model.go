// Package outcomeledger gives every run the same facts however it started,
// links runs into work units and estimates each unit's outcome. Everything but
// the job is a pure function of its inputs, so measures and later analysis
// reuse it and tests state the answer.
package outcomeledger

import (
	"sort"
	"strings"
	"time"
)

// Start paths.
const (
	PathImported           = "imported"
	PathImportedWebConsole = "imported_web_console"
	PathInteractive        = "am_interactive"
	PathHeadless           = "am_headless"
	PathAttached           = "attached"
)

// PathFor names a run's start path from its execution mode and transcript root.
func PathFor(executionMode, rootKind string) string {
	switch executionMode {
	case "imported":
		if strings.HasPrefix(rootKind, "web-console") {
			return PathImportedWebConsole
		}
		return PathImported
	case "interactive":
		return PathInteractive
	case "attached":
		return PathAttached
	}
	return PathHeadless
}

// Classes of attached rows.
const (
	AttachedLinked     = "linked_imported"
	AttachedHelper     = "helper"
	AttachedNoReader   = "no_transcript_reader"
	AttachedUnlinked   = "unlinked_session"
	attachedSessionMin = 3 * time.Minute
)

// Approval states the ledger adds to a run's own. A run approved by the sandbox
// policy at run end (approved_by AutoApprover) is ApprovalAutomatic: the change
// was applied, not judged, so it is no evidence of acceptance.
const (
	ApprovalAutomatic = "auto_approved"
	AutoApprover      = "applyAtRunEnd"
)

// Outcome labels.
const (
	// OutcomeAccepted needs evidence independent of the run's own lifecycle: an
	// operator approval, a passing validation or an explicit positive verdict.
	OutcomeAccepted = "accepted"
	// OutcomeCompleted is lifecycle only: the run finished and nothing says
	// whether the result was accepted. It never counts as accepted.
	OutcomeCompleted = "completed"
	OutcomeCorrected = "corrected"
	OutcomeFailed    = "failed"
	OutcomeAbandoned = "abandoned"
	OutcomeUnknown   = "unknown"
)

// Outcome estimate sources.
const (
	SourceInProgress         = "in_progress"
	SourceOperatorApproval   = "operator_approval"
	SourceOperatorRejection  = "operator_rejection"
	SourceOperatorCorrection = "operator_correction"
	// SourceOperatorReportedFailure: the operator said the work was broken or
	// failing (operator-signal label reported_failure).
	SourceOperatorReportedFailure = "operator_reported_failure"
	// SourceOperatorSignalCorrection: the operator corrected the agent (operator-signal label correction).
	SourceOperatorSignalCorrection = "operator_signal_correction"
	SourceRunStatus                = "run_status"
	SourceRetry                    = "retry"
	SourceAgentVerdict             = "agent_verdict"
	SourceValidation               = "validation"
	SourceHeadlessStatus           = "headless_status"
	SourceNoSignal                 = "no_signal"
)

// Mode relations.
const (
	ModeApplied        = "applied"
	ModeSkipped        = "skipped"
	ModeOnTarget       = "ran_during_on_target"
	ModeNotChanged     = "ran_during_not_changed"
	ModePredates       = "predates_mode"
	ModeNone           = "no_mode"
	BasisSelection     = "selection_reason"
	BasisTimeAndModel  = "time_and_model"
	BasisTime          = "time"
	inferredConfidence = 0.7
)

// RunFacts is one run's ledger row.
type RunFacts struct {
	RunID             string
	SourceStamp       string
	Path              string
	Harness           string
	Status            string
	Live              bool
	TerminalClass     string
	StopReason        string
	ApprovalState     string
	ParentRunID       string
	TaskID            string
	GoalID            string
	SessionKey        string
	WebConsoleSession string
	WorkloadKey       string
	SourceRunIDs      []string
	Role              string
	CreatedAt         time.Time
	SessionStart      time.Time
	FirstActivity     time.Time
	LastActivity      time.Time

	RequestedModel  string
	ActualModel     string
	Effort          string
	SelectionReason string
	ModeID          string
	ModeRelation    string
	ModeBasis       string
	ModeConfidence  float64

	// CanaryArm is the arm the run's immutable policy snapshot recorded
	// (runs.canary_arm): incumbent, challenger or empty.
	CanaryArm string
	// Arm is the arm of a Learn comparison the run belongs to, by assignment,
	// with the mode, how it is known and any recorded exception.
	Arm          string
	ArmMode      string
	ArmBasis     string
	ArmException string

	Requests            int64
	TranscriptToolCalls int64
	FactRows            int64
	Commands            int64
	KnownCommands       int64
	OlderParser         bool
	RecallEvicted       bool
	FileStatus          string
	FileFullyRead       bool
	Edits               int64
	ValidationsPassed   int64
	ValidationsFailed   int64
	LastValidation      string
	LastValidationAt    time.Time
	OperatorMessages    int64
	WallMS              int64
	ActiveMS            int64

	AttachedClass  string
	LinkedRunID    string
	WorkUnitID     string
	LinkKind       string
	LinkConfidence float64
	UnitOutcome    string
	UnitSource     string
}

// Child reports a sub-agent or child run; its parent's unit owns it.
func (r RunFacts) Child() bool { return r.ParentRunID != "" }

// HasRequests reports a run with at least one model request on record.
func (r RunFacts) HasRequests() bool { return r.Requests > 0 }

// UsageExpected reports a run that should have usage: it has requests, or its
// transcript was not read to the end, so "no requests" cannot be told apart
// from "not read yet".
func (r RunFacts) UsageExpected() bool {
	return r.HasRequests() || (r.FileStatus != "" && !r.FileFullyRead)
}

// ModePeriod is one usage mode as observed.
type ModePeriod struct {
	ID      string
	Runner  string
	Role    string
	Profile string
	// Targets are the models the mode lowers work to, per harness family.
	Targets []string
	// Kind is "learn" for a controlled comparison. Learn periods only assign
	// arms; they never make a run "ran during" a saving mode.
	Kind string
	// Label is the operator's label, kept so an ended period can still be named.
	Label string
	// Split, SessionOptIn and BlockDays describe a learn period's randomization.
	Split        float64
	SessionOptIn bool
	BlockDays    int
	StartedAt    time.Time
	ExpiresAt    time.Time
	EndedAt      time.Time
}

// Contains reports whether at falls in the period; a zero end is still open.
func (m ModePeriod) Contains(at time.Time) bool {
	if at.Before(m.StartedAt) {
		return false
	}
	return !at.After(m.end())
}

func (m ModePeriod) end() time.Time {
	switch {
	case !m.EndedAt.IsZero():
		return m.EndedAt
	case !m.ExpiresAt.IsZero():
		return m.ExpiresAt
	}
	return time.Unix(1<<40, 0)
}

// ModeRelation states how a usage mode relates to a run.
type ModeRelation struct {
	ID, Relation, Basis string
	Confidence          float64
}

// ResolveMode is exact from the selection reason of a managed run, otherwise
// inferred from the session's start, its actual model and the mode periods.
func ResolveMode(run RunFacts, periods []ModePeriod) ModeRelation {
	for _, part := range strings.Split(run.SelectionReason, ";") {
		switch {
		// usage_mode_learn:<id> is a comparison assignment, read by ResolveArm.
		case strings.HasPrefix(part, "usage_mode:"):
			return ModeRelation{ID: strings.TrimPrefix(part, "usage_mode:"), Relation: ModeApplied, Basis: BasisSelection, Confidence: 1}
		case strings.HasPrefix(part, "usage_mode_skipped:"):
			return ModeRelation{Relation: ModeSkipped, Basis: BasisSelection, Confidence: 1}
		}
	}
	start := run.FirstActivity
	if !run.SessionStart.IsZero() && (start.IsZero() || run.SessionStart.Before(start)) {
		start = run.SessionStart
	}
	predates := ModeRelation{}
	for _, period := range periods {
		if period.Kind == "learn" || !period.matchesRunner(run.Harness) {
			continue
		}
		if period.Contains(start) {
			for _, target := range period.Targets {
				if strings.EqualFold(target, run.ActualModel) {
					return ModeRelation{ID: period.ID, Relation: ModeOnTarget, Basis: BasisTimeAndModel, Confidence: inferredConfidence}
				}
			}
			return ModeRelation{ID: period.ID, Relation: ModeNotChanged, Basis: BasisTimeAndModel, Confidence: inferredConfidence}
		}
		if start.Before(period.StartedAt) && !run.LastActivity.Before(period.StartedAt) {
			predates = ModeRelation{ID: period.ID, Relation: ModePredates, Basis: BasisTime, Confidence: 1}
		}
	}
	if predates.Relation != "" {
		return predates
	}
	return ModeRelation{Relation: ModeNone, Basis: BasisTime, Confidence: 1}
}

func (m ModePeriod) matchesRunner(harness string) bool {
	if m.Runner == "" {
		return true
	}
	return strings.Contains(strings.ToLower(harness), strings.ToLower(strings.TrimSuffix(m.Runner, "-code")))
}

// activeGap is the longest silence between usage reports still counted as work.
const activeGap = 5 * time.Minute

// ActiveTime splits the span of a run's usage reports into wall time and active
// time. Silence over activeGap is idle: parked, waiting on the operator, or in
// a very long tool call.
func ActiveTime(reports []time.Time) (wall, active time.Duration) {
	if len(reports) < 2 {
		return 0, 0
	}
	first, last := reports[0], reports[0]
	for _, at := range reports {
		if at.Before(first) {
			first = at
		}
		if at.After(last) {
			last = at
		}
	}
	sorted := append([]time.Time(nil), reports...)
	sortTimes(sorted)
	for i := 1; i < len(sorted); i++ {
		if gap := sorted[i].Sub(sorted[i-1]); gap <= activeGap {
			active += gap
		}
	}
	return last.Sub(first), active
}

func sortTimes(values []time.Time) {
	sort.Slice(values, func(i, j int) bool { return values[i].Before(values[j]) })
}

// CallFact is one projected tool call segment.
type CallFact struct {
	CallID  string
	At      time.Time
	Tool    string
	Command string
	Help    bool
	Outcome string
	// Basis is how the call was paired with its result.
	Basis string
}

var commandTools = map[string]bool{"exec": true, "exec_command": true, "shell": true, "shell_command": true, "local_shell": true, "Bash": true, "bash": true}

// validationPrefixes are commands whose outcome is a validation result.
var validationPrefixes = []string{"vrooli scenario test", "test-genie runs wait", "go test", "make test", "pnpm test", "pnpm run test", "npm test", "vitest", "pytest", "scenario-dependency-analyzer verify"}

// IsValidation reports a command whose outcome validates work; a help request
// is not one.
func IsValidation(command string) bool {
	command = strings.TrimSpace(command)
	if strings.Contains(command, "--help") {
		return false
	}
	for _, prefix := range validationPrefixes {
		if command == prefix || strings.HasPrefix(command, prefix+" ") {
			return true
		}
	}
	return false
}

// CallSummary is what one run's tool call facts say.
type CallSummary struct {
	Rows, Commands, Known                int64
	ValidationsPassed, ValidationsFailed int64
	LastValidation                       string
	LastValidationAt                     time.Time
}

// SummarizeCalls counts each command once, known when any of its segments has
// an outcome, and reads validation results per segment. With nestedResults the
// transcript's parser reports the commands inside a Codex `exec` wrapper as their
// own calls, so the wrapper itself, paired only by call id, is not a command.
func SummarizeCalls(calls []CallFact, nestedResults bool) CallSummary {
	var out CallSummary
	type command struct{ known bool }
	commands := map[string]*command{}
	for _, call := range calls {
		out.Rows++
		wrapper := nestedResults && call.Tool == "exec" && call.Basis == "call-id"
		if commandTools[call.Tool] && !wrapper {
			c := commands[call.CallID]
			if c == nil {
				c = &command{}
				commands[call.CallID] = c
			}
			if call.Outcome != "" && call.Outcome != "unknown" {
				c.known = true
			}
		}
		if call.Help || !IsValidation(call.Command) {
			continue
		}
		switch call.Outcome {
		case "success":
			out.ValidationsPassed++
		case "failure":
			out.ValidationsFailed++
		default:
			continue
		}
		if !call.At.Before(out.LastValidationAt) {
			out.LastValidation, out.LastValidationAt = call.Outcome, call.At
		}
	}
	out.Commands = int64(len(commands))
	for _, c := range commands {
		if c.known {
			out.Known++
		}
	}
	return out
}
