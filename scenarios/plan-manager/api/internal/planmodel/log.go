package planmodel

// =============================================================================
// EXECUTION-LOG LEDGER MODEL — the typed plan-log vocabulary
// =============================================================================
//
// The log domain is Plan Manager's single durable ledger for the work products
// an agent produces while executing a plan: design decisions, candidate
// findings, filed bug reports, reusable records, and lightweight notes. These
// types live in the neutral planmodel kernel (no transport/persistence) so the
// log domain owns them, the execution domain can read compact summaries through
// a seam, and every surface speaks one vocabulary.
//
// Findings, bug reports, and records are DISTINCT concepts and must not be
// conflated (see docs/concepts/PLAN-MODEL.md):
//   - a finding is an unvalidated candidate observation (a possible bug),
//   - a bug_report is a defect deliberately filed to the issue tracker,
//   - a record is reusable learning/work captured for the learning loop.
//
// Bug reports and records are forwarded to downstream systems through internal
// seams; the local entry stays durable with a sync status even when the
// downstream write fails (failure is never fatal to plan execution).
// =============================================================================

// LogEntryType is the typed kind of an execution-log ledger entry.
type LogEntryType string

const (
	LogEntryUnspecified LogEntryType = ""
	LogEntryDecision    LogEntryType = "decision"
	LogEntryFinding     LogEntryType = "finding"
	LogEntryBugReport   LogEntryType = "bug_report"
	LogEntryRecord      LogEntryType = "record"
	LogEntryNote        LogEntryType = "note"
)

// LogSyncStatus is the downstream-integration state of a log entry. LOCAL means
// the entry has no downstream target (decisions/findings/notes are local-only).
// PENDING/SYNCED/FAILED apply to bug_report and record entries Plan Manager
// forwards through an internal seam.
type LogSyncStatus string

const (
	LogSyncUnspecified LogSyncStatus = ""
	LogSyncLocal       LogSyncStatus = "local"
	LogSyncPending     LogSyncStatus = "pending"
	LogSyncSynced      LogSyncStatus = "synced"
	LogSyncFailed      LogSyncStatus = "sync_failed"
)

// LogSeverity is the optional severity of a finding or bug-report entry.
type LogSeverity string

const (
	LogSeverityUnspecified LogSeverity = ""
	LogSeverityInfo        LogSeverity = "info"
	LogSeverityLow         LogSeverity = "low"
	LogSeverityMedium      LogSeverity = "medium"
	LogSeverityHigh        LogSeverity = "high"
	LogSeverityCritical    LogSeverity = "critical"
)

// FindingTriage is the operator-triage state of a finding entry. Findings are
// filed as CANDIDATE (unvalidated) and never auto-promoted; an operator triages
// them. Reused by execution/handoff for the candidate-findings roll-up.
type FindingTriage string

const (
	TriageUnspecified FindingTriage = ""
	TriageCandidate   FindingTriage = "candidate"
	TriagePromoted    FindingTriage = "promoted"
	TriageDismissed   FindingTriage = "dismissed"
)

// DownstreamRef records the result of forwarding a log entry to a downstream
// system (the scenario-qa bug inbox, swarm-manager records).
type DownstreamRef struct {
	System    string // "scenario-qa" | "swarm-manager"
	Kind      string // "bug_report" | "record"
	Reference string // downstream id/url once synced
	Detail    string // last sync detail or error
	SyncedAt  string // RFC3339; empty until first successful sync
	Capture   CaptureDisposition
}

// CaptureDisposition is the downstream owning writer's outcome. A draft is a
// durable but private artifact that needs repair; it is not a failed sync and
// must never be represented as a published downstream reference.
type CaptureDisposition struct {
	State      string
	DraftID    string
	Needs      []string
	Invalid    []CaptureDiagnostic
	Warnings   []string
	NextAction []string
}

// CaptureDiagnostic names one invalid supplied field and its recovery hint.
type CaptureDiagnostic struct {
	Field   string
	Value   string
	Message string
}

// BugReportPayload is the complete Scenario QA reporter taxonomy carried by a
// Plan Manager log entry. Plan Manager transports it losslessly; it does not
// select signal types or invent expected/repro/actual facts.
type BugReportPayload struct {
	SignalType   string
	Severity     string
	Repro        []string
	Expected     string
	Actual       string
	Description  string
	Context      map[string]string
	HonestyFlags []string
}

// RecordPayload is the complete Swarm Manager capture classification. Keeping
// it separate from the ledger title/detail prevents the adapter from silently
// hard-coding scenario, kind, or outcome.
type RecordPayload struct {
	Kind      string
	Scenario  string
	Trigger   string
	Approach  string
	Evidence  string
	Outcome   string
	CreatedBy string
}

// LogEntry is one typed execution-log ledger entry — the unified record for
// decisions, candidate findings, bug reports, reusable records, and notes.
type LogEntry struct {
	ID          string
	Type        LogEntryType
	PlanID      string
	ExecutionID string
	PhaseID     string
	Title       string
	Detail      string
	Severity    LogSeverity
	// Triage is meaningful for finding entries (candidate -> promoted/dismissed);
	// other types stay unspecified.
	Triage     FindingTriage
	SyncStatus LogSyncStatus
	Downstream DownstreamRef
	// Bug and Record are meaningful only for their matching entry type.
	Bug     BugReportPayload
	Record  RecordPayload
	Capture CaptureDisposition
	// SourceCommand is the plan-manager CLI command path that produced the entry
	// (audit trail of which guided step created it).
	SourceCommand string
	// Evidence holds optional supporting locators (paths, command output, urls).
	Evidence []string
	// AttributionRunID is an explicitly supplied or verified run id; it powers
	// attribution-keyed dedup.
	AttributionRunID string
	// VerificationStatus records the api-core provenance outcome for the
	// write. Only verified entries carry AttributionRunID.
	VerificationStatus string
	// Harness observations identify the client channel without asserting agent
	// authority or supplying a run id.
	HarnessSessionID string
	HarnessKind      string
	// IdempotencyKey deduplicates retries; a retry with the same key returns the
	// existing entry instead of creating a duplicate.
	IdempotencyKey string
	// SupersedesID links a correction to the entry it corrects/supersedes.
	SupersedesID string
	// PromotedFromID links a bug_report/record back to the finding it came from.
	PromotedFromID string
	CreatedAt      string
	UpdatedAt      string
}

// LogSummaryItem is one compact line in a LogSummary.
type LogSummaryItem struct {
	ID         string
	Type       LogEntryType
	Title      string
	SyncStatus LogSyncStatus
	Triage     FindingTriage
	PhaseID    string
}

// LogSummary is a compact roll-up of an execution's (or plan's) log ledger,
// surfaced into resume/status/handoff so a resumed agent reorients without
// reading every entry.
type LogSummary struct {
	Total             int
	Decisions         int
	Findings          int
	BugReports        int
	Records           int
	Notes             int
	CandidateFindings int
	PendingSync       int
	FailedSync        int
	// Recent is a short, most-recent-first list for quick reorientation.
	Recent []LogSummaryItem
}

// LogFilter narrows a ledger listing. A zero value matches every entry.
type LogFilter struct {
	PlanID      string
	ExecutionID string
	PhaseID     string
	Type        LogEntryType
	Triage      FindingTriage
	SyncStatus  LogSyncStatus
}

// DefaultSyncStatusForType returns the sync status a freshly-created entry of the
// given type starts in: bug reports and records need downstream forwarding
// (PENDING); everything else is local-only.
func DefaultSyncStatusForType(t LogEntryType) LogSyncStatus {
	switch t {
	case LogEntryBugReport, LogEntryRecord:
		return LogSyncPending
	default:
		return LogSyncLocal
	}
}

// SummarizeLog rolls a slice of entries (oldest-first) into a LogSummary. The
// Recent list is most-recent-first and capped at recentLimit.
func SummarizeLog(entries []LogEntry, recentLimit int) LogSummary {
	var s LogSummary
	s.Total = len(entries)
	for _, e := range entries {
		switch e.Type {
		case LogEntryDecision:
			s.Decisions++
		case LogEntryFinding:
			s.Findings++
			if e.Triage == TriageCandidate || e.Triage == TriageUnspecified {
				s.CandidateFindings++
			}
		case LogEntryBugReport:
			s.BugReports++
		case LogEntryRecord:
			s.Records++
		case LogEntryNote:
			s.Notes++
		}
		switch e.SyncStatus {
		case LogSyncPending:
			s.PendingSync++
		case LogSyncFailed:
			s.FailedSync++
		}
	}
	if recentLimit <= 0 {
		recentLimit = 5
	}
	for i := len(entries) - 1; i >= 0 && len(s.Recent) < recentLimit; i-- {
		e := entries[i]
		s.Recent = append(s.Recent, LogSummaryItem{
			ID:         e.ID,
			Type:       e.Type,
			Title:      e.Title,
			SyncStatus: e.SyncStatus,
			Triage:     e.Triage,
			PhaseID:    e.PhaseID,
		})
	}
	return s
}
