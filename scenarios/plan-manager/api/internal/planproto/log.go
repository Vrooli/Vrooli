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

var (
	logEntryTypePairs = []enumPair[planmodel.LogEntryType, sharedv1.LogEntryType]{
		{planmodel.LogEntryDecision, sharedv1.LogEntryType_LOG_ENTRY_TYPE_DECISION},
		{planmodel.LogEntryFinding, sharedv1.LogEntryType_LOG_ENTRY_TYPE_FINDING},
		{planmodel.LogEntryBugReport, sharedv1.LogEntryType_LOG_ENTRY_TYPE_BUG_REPORT},
		{planmodel.LogEntryRecord, sharedv1.LogEntryType_LOG_ENTRY_TYPE_RECORD},
		{planmodel.LogEntryNote, sharedv1.LogEntryType_LOG_ENTRY_TYPE_NOTE},
	}
	logSyncStatusPairs = []enumPair[planmodel.LogSyncStatus, sharedv1.LogSyncStatus]{
		{planmodel.LogSyncLocal, sharedv1.LogSyncStatus_LOG_SYNC_STATUS_LOCAL},
		{planmodel.LogSyncPending, sharedv1.LogSyncStatus_LOG_SYNC_STATUS_PENDING},
		{planmodel.LogSyncSynced, sharedv1.LogSyncStatus_LOG_SYNC_STATUS_SYNCED},
		{planmodel.LogSyncFailed, sharedv1.LogSyncStatus_LOG_SYNC_STATUS_FAILED},
	}
	logSeverityPairs = []enumPair[planmodel.LogSeverity, sharedv1.LogSeverity]{
		{planmodel.LogSeverityInfo, sharedv1.LogSeverity_LOG_SEVERITY_INFO},
		{planmodel.LogSeverityLow, sharedv1.LogSeverity_LOG_SEVERITY_LOW},
		{planmodel.LogSeverityMedium, sharedv1.LogSeverity_LOG_SEVERITY_MEDIUM},
		{planmodel.LogSeverityHigh, sharedv1.LogSeverity_LOG_SEVERITY_HIGH},
		{planmodel.LogSeverityCritical, sharedv1.LogSeverity_LOG_SEVERITY_CRITICAL},
	}
	triagePairs = []enumPair[planmodel.FindingTriage, sharedv1.FindingTriage]{
		{planmodel.TriageCandidate, sharedv1.FindingTriage_FINDING_TRIAGE_CANDIDATE},
		{planmodel.TriagePromoted, sharedv1.FindingTriage_FINDING_TRIAGE_PROMOTED},
		{planmodel.TriageDismissed, sharedv1.FindingTriage_FINDING_TRIAGE_DISMISSED},
	}
)

func LogEntryTypeToProto(t planmodel.LogEntryType) sharedv1.LogEntryType {
	return enumToProto(t, logEntryTypePairs, sharedv1.LogEntryType_LOG_ENTRY_TYPE_UNSPECIFIED)
}

func LogEntryTypeFromProto(t sharedv1.LogEntryType) planmodel.LogEntryType {
	return enumFromProto(t, logEntryTypePairs, planmodel.LogEntryUnspecified)
}

func LogSyncStatusToProto(s planmodel.LogSyncStatus) sharedv1.LogSyncStatus {
	return enumToProto(s, logSyncStatusPairs, sharedv1.LogSyncStatus_LOG_SYNC_STATUS_UNSPECIFIED)
}

func LogSyncStatusFromProto(s sharedv1.LogSyncStatus) planmodel.LogSyncStatus {
	return enumFromProto(s, logSyncStatusPairs, planmodel.LogSyncUnspecified)
}

func LogSeverityToProto(s planmodel.LogSeverity) sharedv1.LogSeverity {
	return enumToProto(s, logSeverityPairs, sharedv1.LogSeverity_LOG_SEVERITY_UNSPECIFIED)
}

func LogSeverityFromProto(s sharedv1.LogSeverity) planmodel.LogSeverity {
	return enumFromProto(s, logSeverityPairs, planmodel.LogSeverityUnspecified)
}

func TriageToProto(t planmodel.FindingTriage) sharedv1.FindingTriage {
	return enumToProto(t, triagePairs, sharedv1.FindingTriage_FINDING_TRIAGE_UNSPECIFIED)
}

func TriageFromProto(t sharedv1.FindingTriage) planmodel.FindingTriage {
	return enumFromProto(t, triagePairs, planmodel.TriageUnspecified)
}
