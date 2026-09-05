package planlog

import (
	"fmt"

	planmodel "plan-manager/internal/planmodel"
)

// stepForEntry returns the guided next action after creating/updating one entry.
// It steers toward the entry's natural follow-up: a finding can be promoted to a
// bug or record; a pending/failed bug or record can be retried via log sync.
func stepForEntry(e Entry) GuidedStep {
	step := GuidedStep{
		StepKind:     "log_entry_recorded",
		Title:        "Log Entry Recorded",
		Summary:      fmt.Sprintf("Recorded %s %q in the plan log ledger.", entryTypeLabel(e.Type), e.Title),
		Instructions: []string{"Continue execution; the entry is durable and surfaces in status/resume/handoff summaries."},
		NextActions: []NextAction{{
			ID:     "log-get",
			Kind:   NextActionOptional,
			Label:  "Inspect this entry",
			Reason: "Read the full entry, including any downstream reference.",
			Argv:   []string{"log", "get", e.ID},
		}, {
			ID:                 "log-reassign",
			Kind:               NextActionRecovery,
			Label:              "Move this entry to another phase",
			Reason:             "Use this if the computed phase scope is not the phase you intended.",
			Argv:               []string{"log", "reassign", e.ID, "--phase", "<phase-id-or-ordinal>"},
			ContentPlaceholder: "--phase <phase-id-or-ordinal>",
		}},
	}
	switch e.Type {
	case planmodel.LogEntryFinding:
		if e.Triage == planmodel.TriageCandidate || e.Triage == planmodel.TriageUnspecified {
			step.NextActions = append([]NextAction{{
				ID:                 "log-promote",
				Kind:               NextActionRecommended,
				Label:              "Promote finding to a bug or record",
				Reason:             "Promote this candidate finding to a filed bug report or a reusable record when it is validated.",
				Argv:               []string{"log", "promote", e.ID, "--to", "bug"},
				ContentPlaceholder: "--to bug|record",
			}}, step.NextActions...)
		}
	case planmodel.LogEntryBugReport, planmodel.LogEntryRecord:
		if e.SyncStatus == planmodel.LogSyncPending || e.SyncStatus == planmodel.LogSyncFailed {
			step.NextActions = append([]NextAction{{
				ID:     "log-sync",
				Kind:   NextActionRecommended,
				Label:  "Retry downstream sync",
				Reason: fmt.Sprintf("Downstream forwarding is %s; retry once the downstream is reachable.", syncStatusLabel(e.SyncStatus)),
				Argv:   []string{"log", "sync", e.ID},
			}}, step.NextActions...)
		}
	}
	return step
}

// stepForList returns the guided step for a ledger listing with a summary.
func stepForList(summary Summary) GuidedStep {
	instructions := []string{
		fmt.Sprintf("%d entries: %d decisions, %d findings, %d bug reports, %d records, %d notes.",
			summary.Total, summary.Decisions, summary.Findings, summary.BugReports, summary.Records, summary.Notes),
	}
	step := GuidedStep{
		StepKind:     "log_list",
		Title:        "Plan Log Ledger",
		Summary:      "The plan log ledger holds decisions, findings, bug reports, and records for this plan/execution.",
		Instructions: instructions,
		NextActions: []NextAction{{
			ID:     "log-get",
			Kind:   NextActionOptional,
			Label:  "Inspect an entry",
			Reason: "Read a full entry by id.",
			Argv:   []string{"log", "get", "<entry id>"},
		}},
	}
	if summary.PendingSync+summary.FailedSync > 0 {
		step.NextActions = append([]NextAction{{
			ID:     "log-sync",
			Kind:   NextActionRecommended,
			Label:  "Retry degraded downstream syncs",
			Reason: fmt.Sprintf("%d bug/record entries are pending or failed downstream; retry them.", summary.PendingSync+summary.FailedSync),
			Argv:   []string{"log", "sync", "<entry id>"},
		}}, step.NextActions...)
	}
	return step
}

func entryTypeLabel(t EntryType) string {
	switch t {
	case planmodel.LogEntryDecision:
		return "decision"
	case planmodel.LogEntryFinding:
		return "candidate finding"
	case planmodel.LogEntryBugReport:
		return "bug report"
	case planmodel.LogEntryRecord:
		return "record"
	case planmodel.LogEntryNote:
		return "note"
	default:
		return "entry"
	}
}

func syncStatusLabel(s SyncStatus) string {
	switch s {
	case planmodel.LogSyncLocal:
		return "local"
	case planmodel.LogSyncPending:
		return "pending"
	case planmodel.LogSyncSynced:
		return "synced"
	case planmodel.LogSyncFailed:
		return "sync_failed"
	default:
		return "unspecified"
	}
}
