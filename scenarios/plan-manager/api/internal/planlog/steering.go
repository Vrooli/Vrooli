package planlog

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

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
			if ref := explicitScenarioQABugReference(e); ref != "" {
				step.Instructions = append(step.Instructions, fmt.Sprintf(
					"This entry names existing Scenario QA bug %s. Do not retry forwarding; reconcile the local entry against that published reference first.", ref))
			} else {
				step.NextActions = append([]NextAction{{
					ID:     "log-sync",
					Kind:   NextActionRecommended,
					Label:  "Retry downstream sync",
					Reason: fmt.Sprintf("Downstream forwarding is %s; the source entry has no explicit Scenario QA reference, and retries retain the same idempotency key.", syncStatusLabel(e.SyncStatus)),
					Argv:   []string{"log", "sync", e.ID},
				}}, step.NextActions...)
			}
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
		step.NextActions[0] = NextAction{
			ID:     "log-get",
			Kind:   NextActionRecommended,
			Label:  "Review degraded downstream entries",
			Reason: fmt.Sprintf("%d bug/record entries are pending or failed; inspect each exact entry for a published Scenario QA reference before syncing. Explicit references are blocked from automatic retry.", summary.PendingSync+summary.FailedSync),
			Argv:   []string{"log", "get", "<entry id>"},
		}
	}
	return step
}

var (
	knowledgeReferencePattern     = regexp.MustCompile(`(?i)^knw-[0-9]+$`)
	scenarioQABugReferencePattern = regexp.MustCompile(`(?i)\bscenario[- ]qa\s+bug\s+(knw-[0-9]+)\b`)
)

// explicitScenarioQABugReference returns a known published knowledge ID only
// when the entry carries the ID in structured downstream state or in source
// text that explicitly identifies it as a Scenario QA bug. A generic knw token
// is not enough to block an otherwise idempotent retry.
func explicitScenarioQABugReference(e Entry) string {
	if e.Type != planmodel.LogEntryBugReport {
		return ""
	}
	if strings.EqualFold(strings.TrimSpace(e.Downstream.System), scenarioQASystem) && strings.EqualFold(strings.TrimSpace(e.Downstream.Kind), "bug_report") {
		if ref := strings.TrimSpace(e.Downstream.Reference); knowledgeReferencePattern.MatchString(ref) {
			return knowledgeReferencePattern.FindString(ref)
		}
	}

	texts := []string{e.Title, e.Detail, e.Bug.Description, e.Bug.Expected, e.Bug.Actual, e.Downstream.Detail}
	texts = append(texts, e.Bug.Repro...)
	texts = append(texts, e.Evidence...)
	contextKeys := make([]string, 0, len(e.Bug.Context))
	for key := range e.Bug.Context {
		contextKeys = append(contextKeys, key)
	}
	sort.Strings(contextKeys)
	for _, key := range contextKeys {
		texts = append(texts, e.Bug.Context[key])
	}

	for _, text := range texts {
		if match := scenarioQABugReferencePattern.FindStringSubmatch(text); len(match) == 2 {
			return match[1]
		}
	}
	return ""
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
