package planproto

import (
	planmodel "plan-manager/internal/planmodel"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
)

// LogEntryToProto converts a planmodel.LogEntry into its shared proto message.
func LogEntryToProto(e planmodel.LogEntry) *sharedv1.LogEntry {
	return &sharedv1.LogEntry{
		Id:               e.ID,
		Type:             LogEntryTypeToProto(e.Type),
		PlanId:           e.PlanID,
		ExecutionId:      e.ExecutionID,
		PhaseId:          e.PhaseID,
		Title:            e.Title,
		Detail:           e.Detail,
		Severity:         LogSeverityToProto(e.Severity),
		Triage:           TriageToProto(e.Triage),
		SyncStatus:       LogSyncStatusToProto(e.SyncStatus),
		Downstream:       downstreamRefToProto(e.Downstream),
		SourceCommand:    e.SourceCommand,
		Evidence:         append([]string(nil), e.Evidence...),
		AttributionRunId: e.AttributionRunID,
		IdempotencyKey:   e.IdempotencyKey,
		SupersedesId:     e.SupersedesID,
		PromotedFromId:   e.PromotedFromID,
		CreatedAt:        e.CreatedAt,
		UpdatedAt:        e.UpdatedAt,
	}
}

// LogEntriesToProto converts a slice of entries.
func LogEntriesToProto(entries []planmodel.LogEntry) []*sharedv1.LogEntry {
	out := make([]*sharedv1.LogEntry, 0, len(entries))
	for _, e := range entries {
		out = append(out, LogEntryToProto(e))
	}
	return out
}

func downstreamRefToProto(d planmodel.DownstreamRef) *sharedv1.DownstreamRef {
	if d == (planmodel.DownstreamRef{}) {
		return nil
	}
	return &sharedv1.DownstreamRef{
		System:    d.System,
		Kind:      d.Kind,
		Reference: d.Reference,
		Detail:    d.Detail,
		SyncedAt:  d.SyncedAt,
	}
}

// LogSummaryToProto converts a planmodel.LogSummary into its shared proto.
func LogSummaryToProto(s planmodel.LogSummary) *sharedv1.LogSummary {
	recent := make([]*sharedv1.LogSummaryItem, 0, len(s.Recent))
	for _, it := range s.Recent {
		recent = append(recent, &sharedv1.LogSummaryItem{
			Id:         it.ID,
			Type:       LogEntryTypeToProto(it.Type),
			Title:      it.Title,
			SyncStatus: LogSyncStatusToProto(it.SyncStatus),
			Triage:     TriageToProto(it.Triage),
			PhaseId:    it.PhaseID,
		})
	}
	return &sharedv1.LogSummary{
		Total:             OrderToInt32(s.Total),
		Decisions:         OrderToInt32(s.Decisions),
		Findings:          OrderToInt32(s.Findings),
		BugReports:        OrderToInt32(s.BugReports),
		Records:           OrderToInt32(s.Records),
		Notes:             OrderToInt32(s.Notes),
		CandidateFindings: OrderToInt32(s.CandidateFindings),
		PendingSync:       OrderToInt32(s.PendingSync),
		FailedSync:        OrderToInt32(s.FailedSync),
		Recent:            recent,
	}
}

// --- enum converters ---------------------------------------------------------

func LogEntryTypeToProto(t planmodel.LogEntryType) sharedv1.LogEntryType {
	switch t {
	case planmodel.LogEntryDecision:
		return sharedv1.LogEntryType_LOG_ENTRY_TYPE_DECISION
	case planmodel.LogEntryFinding:
		return sharedv1.LogEntryType_LOG_ENTRY_TYPE_FINDING
	case planmodel.LogEntryBugReport:
		return sharedv1.LogEntryType_LOG_ENTRY_TYPE_BUG_REPORT
	case planmodel.LogEntryRecord:
		return sharedv1.LogEntryType_LOG_ENTRY_TYPE_RECORD
	case planmodel.LogEntryNote:
		return sharedv1.LogEntryType_LOG_ENTRY_TYPE_NOTE
	default:
		return sharedv1.LogEntryType_LOG_ENTRY_TYPE_UNSPECIFIED
	}
}

func LogEntryTypeFromProto(t sharedv1.LogEntryType) planmodel.LogEntryType {
	switch t {
	case sharedv1.LogEntryType_LOG_ENTRY_TYPE_DECISION:
		return planmodel.LogEntryDecision
	case sharedv1.LogEntryType_LOG_ENTRY_TYPE_FINDING:
		return planmodel.LogEntryFinding
	case sharedv1.LogEntryType_LOG_ENTRY_TYPE_BUG_REPORT:
		return planmodel.LogEntryBugReport
	case sharedv1.LogEntryType_LOG_ENTRY_TYPE_RECORD:
		return planmodel.LogEntryRecord
	case sharedv1.LogEntryType_LOG_ENTRY_TYPE_NOTE:
		return planmodel.LogEntryNote
	default:
		return planmodel.LogEntryUnspecified
	}
}

func LogSyncStatusToProto(s planmodel.LogSyncStatus) sharedv1.LogSyncStatus {
	switch s {
	case planmodel.LogSyncLocal:
		return sharedv1.LogSyncStatus_LOG_SYNC_STATUS_LOCAL
	case planmodel.LogSyncPending:
		return sharedv1.LogSyncStatus_LOG_SYNC_STATUS_PENDING
	case planmodel.LogSyncSynced:
		return sharedv1.LogSyncStatus_LOG_SYNC_STATUS_SYNCED
	case planmodel.LogSyncFailed:
		return sharedv1.LogSyncStatus_LOG_SYNC_STATUS_FAILED
	default:
		return sharedv1.LogSyncStatus_LOG_SYNC_STATUS_UNSPECIFIED
	}
}

func LogSyncStatusFromProto(s sharedv1.LogSyncStatus) planmodel.LogSyncStatus {
	switch s {
	case sharedv1.LogSyncStatus_LOG_SYNC_STATUS_LOCAL:
		return planmodel.LogSyncLocal
	case sharedv1.LogSyncStatus_LOG_SYNC_STATUS_PENDING:
		return planmodel.LogSyncPending
	case sharedv1.LogSyncStatus_LOG_SYNC_STATUS_SYNCED:
		return planmodel.LogSyncSynced
	case sharedv1.LogSyncStatus_LOG_SYNC_STATUS_FAILED:
		return planmodel.LogSyncFailed
	default:
		return planmodel.LogSyncUnspecified
	}
}

func LogSeverityToProto(s planmodel.LogSeverity) sharedv1.LogSeverity {
	switch s {
	case planmodel.LogSeverityInfo:
		return sharedv1.LogSeverity_LOG_SEVERITY_INFO
	case planmodel.LogSeverityLow:
		return sharedv1.LogSeverity_LOG_SEVERITY_LOW
	case planmodel.LogSeverityMedium:
		return sharedv1.LogSeverity_LOG_SEVERITY_MEDIUM
	case planmodel.LogSeverityHigh:
		return sharedv1.LogSeverity_LOG_SEVERITY_HIGH
	case planmodel.LogSeverityCritical:
		return sharedv1.LogSeverity_LOG_SEVERITY_CRITICAL
	default:
		return sharedv1.LogSeverity_LOG_SEVERITY_UNSPECIFIED
	}
}

func LogSeverityFromProto(s sharedv1.LogSeverity) planmodel.LogSeverity {
	switch s {
	case sharedv1.LogSeverity_LOG_SEVERITY_INFO:
		return planmodel.LogSeverityInfo
	case sharedv1.LogSeverity_LOG_SEVERITY_LOW:
		return planmodel.LogSeverityLow
	case sharedv1.LogSeverity_LOG_SEVERITY_MEDIUM:
		return planmodel.LogSeverityMedium
	case sharedv1.LogSeverity_LOG_SEVERITY_HIGH:
		return planmodel.LogSeverityHigh
	case sharedv1.LogSeverity_LOG_SEVERITY_CRITICAL:
		return planmodel.LogSeverityCritical
	default:
		return planmodel.LogSeverityUnspecified
	}
}

func TriageToProto(t planmodel.FindingTriage) sharedv1.FindingTriage {
	switch t {
	case planmodel.TriageCandidate:
		return sharedv1.FindingTriage_FINDING_TRIAGE_CANDIDATE
	case planmodel.TriagePromoted:
		return sharedv1.FindingTriage_FINDING_TRIAGE_PROMOTED
	case planmodel.TriageDismissed:
		return sharedv1.FindingTriage_FINDING_TRIAGE_DISMISSED
	default:
		return sharedv1.FindingTriage_FINDING_TRIAGE_UNSPECIFIED
	}
}

func TriageFromProto(t sharedv1.FindingTriage) planmodel.FindingTriage {
	switch t {
	case sharedv1.FindingTriage_FINDING_TRIAGE_CANDIDATE:
		return planmodel.TriageCandidate
	case sharedv1.FindingTriage_FINDING_TRIAGE_PROMOTED:
		return planmodel.TriagePromoted
	case sharedv1.FindingTriage_FINDING_TRIAGE_DISMISSED:
		return planmodel.TriageDismissed
	default:
		return planmodel.TriageUnspecified
	}
}
