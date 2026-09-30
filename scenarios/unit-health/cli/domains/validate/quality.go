package validate

import (
	"fmt"

	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/unit-health/v1/validation"
)

func qualityStatusLabel(status validationv1.QualityCheckStatus) string {
	switch status {
	case validationv1.QualityCheckStatus_QUALITY_CHECK_STATUS_VIOLATION:
		return "violation"
	case validationv1.QualityCheckStatus_QUALITY_CHECK_STATUS_CHECKED_CLEAN:
		return "checked_clean"
	case validationv1.QualityCheckStatus_QUALITY_CHECK_STATUS_NOT_APPLICABLE:
		return "not_applicable"
	default:
		return "unknown"
	}
}

func evidenceStageLines(stages *validationv1.EvidenceStages) []string {
	if stages == nil {
		return []string{"Evidence stages: unknown (historical or missing stage report); validation status is not execution or review evidence"}
	}
	known := func(value string, allowed ...string) string {
		for _, candidate := range allowed {
			if value == candidate {
				return value
			}
		}
		return "unknown"
	}
	return []string{fmt.Sprintf("Evidence stages: configured=%s analyzed=%s executed=%s reviewed=%s; source_run=%s",
		known(stages.GetConfigured(), "observed", "cached"), known(stages.GetAnalyzed(), "observed", "partial", "cached"),
		known(stages.GetExecuted(), "not_requested", "not_executed", "passed", "failed", "refused", "cached"), known(stages.GetReviewed(), "not_supplied"), stages.GetSourceRunId())}
}

func qualityLines(report *validationv1.TestQualityReport) []string {
	if report == nil {
		return []string{"Test quality: unknown (no scoped assessment supplied)"}
	}
	unsupported := report.GetSchemaVersion() != "test-quality/v1" || report.GetCatalogVersion() == ""
	unavailable := unsupported || (report.GetUnavailableReason() != validationv1.QualityReason_QUALITY_REASON_UNSPECIFIED && report.GetUnavailableReason() != validationv1.QualityReason_QUALITY_REASON_NONE)
	total := "unknown"
	if !unavailable && report.GetTotalResults() >= uint64(len(report.GetResults())) {
		total = fmt.Sprint(report.GetTotalResults())
	}
	lines := []string{fmt.Sprintf("Test quality: %s scoped result(s); checked_clean is not behavioral certification", total)}
	if unsupported {
		lines = append(lines, "  Test quality: unknown (missing or unsupported assessment schema)")
	}
	if unavailable {
		lines = append(lines, fmt.Sprintf("  Assessment unavailable: %s; retained locations are not clean evidence", report.GetUnavailableReason()))
	}
	if report.GetReasonGuidance() != "" {
		lines = append(lines, "  Next: "+report.GetReasonGuidance())
	}
	for _, limitation := range report.GetCollectionLimitations() {
		lines = append(lines, fmt.Sprintf("  Collection incomplete (%s): counts cover reported rows only, not complete discovery. %s", limitation.GetReason(), limitation.GetGuidance()))
	}
	if len(report.GetCoverage()) == 0 || unavailable {
		lines = append(lines, "  Assessment coverage: unknown (no trustworthy denominator supplied)")
	} else {
		for _, c := range report.GetCoverage() {
			if c.GetAssessed() > c.GetDiscovered() || c.GetUnknown() > c.GetDiscovered()-c.GetAssessed() || c.GetNotApplicable() != c.GetDiscovered()-c.GetAssessed()-c.GetUnknown() {
				lines = append(lines, fmt.Sprintf("  %s/%s assessment coverage: unknown (inconsistent denominator)", c.GetRuleId(), c.GetSupportProfile()))
				continue
			}
			lines = append(lines, fmt.Sprintf("  %s/%s scope=%s assessed=%d/discovered=%d unknown=%d not_applicable=%d; not behavioral coverage", c.GetRuleId(), c.GetSupportProfile(), c.GetScope(), c.GetAssessed(), c.GetDiscovered(), c.GetUnknown(), c.GetNotApplicable()))
		}
	}
	for _, row := range report.GetResults() {
		target := row.GetTarget().GetTestId()
		if row.GetTarget().GetScope() == "file" {
			target = "[file scope]"
		}
		status := qualityStatusLabel(row.GetStatus())
		if unavailable {
			status = "unknown"
		}
		lines = append(lines, fmt.Sprintf("  %s/%s [%s] %s:%d %s — %s; %s; severity=%s; enforcement=%s",
			row.GetRuleId(), row.GetRuleVersion(), status, row.GetTarget().GetFile(), row.GetLocation().GetLine(), target, row.GetReason(), row.GetEvidenceKind(), row.GetSeverity(), row.GetEnforcement()))
		for _, d := range row.GetDiagnostics() {
			lines = append(lines, fmt.Sprintf("    %s:%d:%d %s (%s): %s", row.GetTarget().GetFile(), d.GetLocation().GetLine(), d.GetLocation().GetColumn(), d.GetNativeRuleId(), d.GetMessageId(), d.GetMessage()))
		}
		if row.GetReasonGuidance() != "" {
			lines = append(lines, "    Next: "+row.GetReasonGuidance())
		}
		if native := row.GetRuntimeObservation(); native != nil {
			line := fmt.Sprintf("    native final state=%s run=%s; not a complete attempt history", native.GetState(), native.GetRunId())
			if native.Seed != nil {
				line += fmt.Sprintf(" seed=%q", native.GetSeed())
			}
			if native.RetryCount != nil {
				line += fmt.Sprintf(" retry_count=%d", native.GetRetryCount())
			}
			if native.RetryOrdinal != nil {
				line += fmt.Sprintf(" retry_ordinal=%d", native.GetRetryOrdinal())
			}
			lines = append(lines, line)
		}
	}
	if report.GetTruncated() {
		lines = append(lines, fmt.Sprintf("  Showing %d of %d; details: %s", len(report.GetResults()), report.GetTotalResults(), report.GetDetailsRef()))
	}
	return lines
}

func traceabilityLines(report *validationv1.RequirementTraceabilityReport) []string {
	if report == nil || report.GetSchemaVersion() != "requirement-traceability/v1" {
		return []string{"Requirement traceability: unknown (missing or unsupported report)"}
	}
	lines := []string{"Requirement traceability: declared links and passing observations are not behavioral proof"}
	unavailable := func(reason validationv1.QualityReason) bool {
		return reason != validationv1.QualityReason_QUALITY_REASON_UNSPECIFIED && reason != validationv1.QualityReason_QUALITY_REASON_NONE
	}
	if unavailable(report.GetUnavailableReason()) {
		lines = append(lines, fmt.Sprintf("  Registry: unknown (%s)", report.GetUnavailableReason()))
	}
	if unavailable(report.GetEvidenceUnavailableReason()) {
		lines = append(lines, fmt.Sprintf("  Evidence: incomplete (%s)", report.GetEvidenceUnavailableReason()))
	}
	if !unavailable(report.GetUnavailableReason()) {
		for _, scope := range report.GetRequirements() {
			label := "unknown"
			switch scope.GetApplicability() {
			case validationv1.RequirementApplicability_REQUIREMENT_APPLICABILITY_APPLICABLE:
				label = "applicable"
			case validationv1.RequirementApplicability_REQUIREMENT_APPLICABILITY_NOT_APPLICABLE:
				label = "not_applicable"
			}
			lines = append(lines, fmt.Sprintf("  %s unit responsibility: %s", scope.GetRequirementId(), label))
		}
	}
	for _, link := range report.GetLinks() {
		registration, execution := "unknown", "unknown"
		if !unavailable(report.GetUnavailableReason()) {
			switch link.GetRegistration() {
			case validationv1.RequirementRegistration_REQUIREMENT_REGISTRATION_REGISTERED:
				registration = "registered"
			case validationv1.RequirementRegistration_REQUIREMENT_REGISTRATION_STALE:
				registration = "stale"
			}
		}
		switch link.GetExecution() {
		case validationv1.RequirementExecutionState_REQUIREMENT_EXECUTION_STATE_PASSED:
			execution = "passed"
		case validationv1.RequirementExecutionState_REQUIREMENT_EXECUTION_STATE_FAILED:
			execution = "failed"
		case validationv1.RequirementExecutionState_REQUIREMENT_EXECUTION_STATE_SKIPPED:
			execution = "skipped"
		case validationv1.RequirementExecutionState_REQUIREMENT_EXECUTION_STATE_NOT_RUN:
			execution = "not_run"
		}
		lines = append(lines, fmt.Sprintf("  %s [%s] %s:%s execution=%s run=%s reason=%s", link.GetRequirementId(), registration, link.GetTarget().GetFile(), link.GetTarget().GetTestId(), execution, link.GetRunId(), link.GetReason()))
	}
	return lines
}
