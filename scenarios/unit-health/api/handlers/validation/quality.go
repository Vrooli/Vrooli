package validation

import (
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/unit-health/v1/validation"
	"strings"
	"unit-health/internal/testquality"
)

func qualityStatusToProto(in testquality.Status) validationv1.QualityCheckStatus {
	switch testquality.NormalizeStatus(in) {
	case testquality.Violation:
		return validationv1.QualityCheckStatus_QUALITY_CHECK_STATUS_VIOLATION
	case testquality.CheckedClean:
		return validationv1.QualityCheckStatus_QUALITY_CHECK_STATUS_CHECKED_CLEAN
	case testquality.NotApplicable:
		return validationv1.QualityCheckStatus_QUALITY_CHECK_STATUS_NOT_APPLICABLE
	default:
		return validationv1.QualityCheckStatus_QUALITY_CHECK_STATUS_UNKNOWN
	}
}

func traceabilityToProto(in *testquality.TraceabilityReport) *validationv1.RequirementTraceabilityReport {
	if in == nil {
		return nil
	}
	r := in.Normalized()
	out := &validationv1.RequirementTraceabilityReport{SchemaVersion: r.SchemaVersion, Limitations: r.Limitations}
	if r.UnavailableReason != "" {
		out.UnavailableReason = qualityReasonToProto(r.UnavailableReason)
	}
	if r.EvidenceUnavailableReason != "" {
		out.EvidenceUnavailableReason = qualityReasonToProto(r.EvidenceUnavailableReason)
	}
	for _, scope := range r.Requirements {
		value := validationv1.RequirementApplicability_value["REQUIREMENT_APPLICABILITY_"+strings.ToUpper(scope.Applicability)]
		out.Requirements = append(out.Requirements, &validationv1.RequirementTraceScope{RequirementId: scope.ID, Applicability: validationv1.RequirementApplicability(value)})
	}
	for _, link := range r.Links {
		registration := validationv1.RequirementRegistration_value["REQUIREMENT_REGISTRATION_"+strings.ToUpper(link.Registration)]
		execution := validationv1.RequirementExecutionState_value["REQUIREMENT_EXECUTION_STATE_"+strings.ToUpper(string(link.Execution))]
		out.Links = append(out.Links, &validationv1.RequirementTraceLink{RequirementId: link.ID,
			Target:       &validationv1.QualityTestTarget{Workspace: link.Target.Workspace, File: link.Target.File, TestId: link.Target.TestID, Scope: link.Target.Scope},
			Registration: validationv1.RequirementRegistration(registration), Execution: validationv1.RequirementExecutionState(execution), Reason: qualityReasonToProto(link.Reason), RunId: link.RunID})
	}
	return out
}

func qualityReasonToProto(in testquality.Reason) validationv1.QualityReason {
	name := "QUALITY_REASON_" + strings.ToUpper(strings.ReplaceAll(string(testquality.NormalizeReason(in)), "-", "_"))
	if value, ok := validationv1.QualityReason_value[name]; ok {
		return validationv1.QualityReason(value)
	}
	return validationv1.QualityReason_QUALITY_REASON_UNRECOGNIZED_VALUE
}

func qualityReportToProto(in *testquality.Report) *validationv1.TestQualityReport {
	if in == nil {
		return nil
	}
	normalized := in.Normalized()
	in = &normalized
	out := &validationv1.TestQualityReport{SchemaVersion: in.SchemaVersion, CatalogVersion: in.CatalogVersion,
		Truncated: in.Truncated, DetailsRef: in.DetailsRef, ReasonGuidance: testquality.ReasonGuidance(in.UnavailableReason)}
	if in.TotalResults >= 0 {
		out.TotalResults = uint64(in.TotalResults)
	}
	if in.UnavailableReason != "" {
		out.UnavailableReason = qualityReasonToProto(in.UnavailableReason)
	}
	for _, reason := range in.IncompleteReasons {
		out.CollectionLimitations = append(out.CollectionLimitations, &validationv1.QualityCollectionLimitation{Reason: qualityReasonToProto(reason), Guidance: testquality.ReasonGuidance(reason)})
	}
	for _, raw := range in.Results {
		r := raw.Normalized()
		out.Results = append(out.Results, &validationv1.QualityCheckResult{
			RuleId: r.RuleID, RuleVersion: r.RuleVersion, TestKind: r.TestKind, SupportProfile: r.SupportProfile,
			Target:   &validationv1.QualityTestTarget{Workspace: r.Target.Workspace, File: r.Target.File, TestId: r.Target.TestID, Scope: r.Target.Scope},
			Location: &validationv1.QualitySourceLocation{Line: int32(r.Location.Line), Column: int32(r.Location.Column), EndLine: int32(r.Location.EndLine), EndColumn: int32(r.Location.EndColumn)},
			Status:   qualityStatusToProto(r.Status), Reason: qualityReasonToProto(r.Reason),
			ReasonGuidance: testquality.ReasonGuidance(r.Reason),
			EvidenceKind:   validationv1.QualityEvidenceKind(validationv1.QualityEvidenceKind_value["QUALITY_EVIDENCE_KIND_"+strings.ToUpper(string(r.EvidenceKind))]),
			Severity:       validationv1.QualitySeverity(validationv1.QualitySeverity_value["QUALITY_SEVERITY_"+strings.ToUpper(string(r.Severity))]),
			Enforcement:    validationv1.QualityEnforcement(validationv1.QualityEnforcement_value["QUALITY_ENFORCEMENT_"+strings.ToUpper(string(r.Enforcement))]),
			EvidenceRefs:   append([]string(nil), r.EvidenceRefs...), Limitations: append([]string(nil), r.Limitations...),
		})
		for _, d := range r.Diagnostics {
			last := out.Results[len(out.Results)-1]
			last.Diagnostics = append(last.Diagnostics, &validationv1.QualityNativeDiagnostic{NativeRuleId: d.RuleID, MessageId: d.MessageID, Message: d.Message, NativeSeverity: int32(d.Severity), Location: &validationv1.QualitySourceLocation{Line: int32(d.Location.Line), Column: int32(d.Location.Column), EndLine: int32(d.Location.EndLine), EndColumn: int32(d.Location.EndColumn)}})
		}
		if native := r.RuntimeObservation; native != nil {
			observation := &validationv1.QualityRuntimeObservation{RunId: native.RunID, State: native.State, Seed: native.Seed}
			if native.RetryCount != nil {
				count := int32(*native.RetryCount)
				observation.RetryCount = &count
			}
			if native.RetryOrdinal != nil {
				ordinal := int32(*native.RetryOrdinal)
				observation.RetryOrdinal = &ordinal
			}
			out.Results[len(out.Results)-1].RuntimeObservation = observation
		}
	}
	for _, c := range in.Coverage {
		out.Coverage = append(out.Coverage, &validationv1.QualityAssessmentCoverage{RuleId: c.RuleID, SupportProfile: c.SupportProfile, Scope: c.Scope,
			Discovered: uint64(c.Discovered), Assessed: uint64(c.Assessed), Unknown: uint64(c.Unknown), NotApplicable: uint64(c.NotApplicable)})
	}
	return out
}
