package execution

import (
	"math"
	"testing"

	internalexecution "plan-manager/internal/execution"
	internalplans "plan-manager/internal/plans"

	"github.com/stretchr/testify/require"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
)

func TestExecutionToProto(t *testing.T) {
	got := executionToProto(internalexecution.Execution{
		ID:             "e1",
		PlanID:         "p1",
		RunID:          "run-1",
		CurrentPhaseID: "ph-2",
		Complete:       true,
		StartedAt:      "t0",
		UpdatedAt:      "t1",
	})
	require.Equal(t, "e1", got.GetId())
	require.Equal(t, "p1", got.GetPlanId())
	require.Equal(t, "run-1", got.GetRunId())
	require.Equal(t, "ph-2", got.GetCurrentPhaseId())
	require.True(t, got.GetComplete())
	require.Equal(t, "t0", got.GetStartedAt())
	require.Equal(t, "t1", got.GetUpdatedAt())
}

func TestPhaseContextToProtoIncludesPresentParts(t *testing.T) {
	pctx := internalexecution.PhaseContext{
		CurrentPhase:    internalplans.Phase{ID: "ph-1", Title: "Cur"},
		NextPhase:       internalplans.Phase{ID: "ph-2", Title: "Next"},
		HasCurrent:      true,
		HasNext:         true,
		HasValidation:   true,
		LastValidation:  internalexecution.ValidationResult{ID: "v1", Verdict: "pass"},
		RequiredReading: []string{"r"},
		Reminders:       []string{"rem"},
		Staleness:       internalplans.StalenessFresh,
		ResumePhaseID:   "ph-1",
		Completeness:    internalexecution.CompletenessPartial,
	}
	got := phaseContextToProto(pctx)
	require.NotNil(t, got.GetCurrentPhase())
	require.Equal(t, "ph-1", got.GetCurrentPhase().GetId())
	require.NotNil(t, got.GetNextPhase())
	require.Equal(t, "ph-2", got.GetNextPhase().GetId())
	require.NotNil(t, got.GetLastValidation())
	require.Equal(t, sharedv1.ValidationVerdict_VALIDATION_VERDICT_PASS, got.GetLastValidation().GetVerdict())
	require.Equal(t, []string{"r"}, got.GetRequiredReading())
	require.Equal(t, sharedv1.StalenessTier_STALENESS_TIER_FRESH, got.GetStaleness())
	require.Equal(t, "ph-1", got.GetResumePhaseId())
	require.Equal(t, sharedv1.Completeness_COMPLETENESS_PARTIAL, got.GetCompleteness())
}

func TestPhaseContextToProtoOmitsAbsentParts(t *testing.T) {
	got := phaseContextToProto(internalexecution.PhaseContext{
		HasCurrent:    false,
		HasNext:       false,
		HasValidation: false,
		Completeness:  internalexecution.CompletenessFull,
	})
	require.Nil(t, got.GetCurrentPhase(), "absent current phase must be nil")
	require.Nil(t, got.GetNextPhase(), "absent next phase must be nil")
	require.Nil(t, got.GetLastValidation(), "absent validation must be nil")
	require.Equal(t, sharedv1.Completeness_COMPLETENESS_FULL, got.GetCompleteness())
}

func TestNudgesToProto(t *testing.T) {
	got := nudgesToProto([]internalexecution.CompletionNudge{
		{Kind: "record_finding", Message: "m1", Satisfied: true},
		{Kind: "file_bugs", Message: "m2", Satisfied: false},
	})
	require.Len(t, got, 2)
	require.Equal(t, "record_finding", got[0].GetKind())
	require.True(t, got[0].GetSatisfied())
	require.Equal(t, "file_bugs", got[1].GetKind())
	require.False(t, got[1].GetSatisfied())
}

func TestDecisionToProto(t *testing.T) {
	got := decisionToProto(internalexecution.Decision{
		ID: "d1", Summary: "s", Detail: "det", PhaseID: "ph-1", RecordedAt: "t",
	})
	require.Equal(t, "d1", got.GetId())
	require.Equal(t, "s", got.GetSummary())
	require.Equal(t, "det", got.GetDetail())
	require.Equal(t, "ph-1", got.GetPhaseId())
	require.Equal(t, "t", got.GetRecordedAt())
}

func TestFindingToProto(t *testing.T) {
	got := findingToProto(internalexecution.Finding{
		ID: "f1", Title: "title", Detail: "det", Triage: internalexecution.TriageCandidate,
		PhaseID: "ph-1", RecordedAt: "t", AttributionRunID: "run-1",
	})
	require.Equal(t, "f1", got.GetId())
	require.Equal(t, "title", got.GetTitle())
	require.Equal(t, sharedv1.FindingTriage_FINDING_TRIAGE_CANDIDATE, got.GetTriage())
	require.Equal(t, "run-1", got.GetAttributionRunId())
}

func TestHandoffToProto(t *testing.T) {
	h := internalexecution.Handoff{
		ID:            "h1",
		ExecutionID:   "e1",
		PlanID:        "p1",
		Completeness:  internalexecution.CompletenessFull,
		ResumePhaseID: "",
		Decisions:     []internalexecution.Decision{{ID: "d1"}},
		CandidateFindings: []internalexecution.Finding{
			{ID: "f1", Triage: internalexecution.TriageCandidate},
		},
		HasValidation:   true,
		LastValidation:  internalexecution.ValidationResult{ID: "v1", Verdict: "fail"},
		Staleness:       internalplans.StalenessDefinitelyStale,
		ProseHandoffRef: "ref",
		AssembledAt:     "t",
	}
	got := handoffToProto(h)
	require.Equal(t, "h1", got.GetId())
	require.Equal(t, sharedv1.Completeness_COMPLETENESS_FULL, got.GetCompleteness())
	require.Len(t, got.GetDecisions(), 1)
	require.Len(t, got.GetCandidateFindings(), 1)
	require.Equal(t, sharedv1.StalenessTier_STALENESS_TIER_DEFINITELY_STALE, got.GetStaleness())
	require.NotNil(t, got.GetLastValidation())
	require.Equal(t, sharedv1.ValidationVerdict_VALIDATION_VERDICT_FAIL, got.GetLastValidation().GetVerdict())
	require.Equal(t, "ref", got.GetProseHandoffRef())
}

func TestHandoffToProtoOmitsValidationWhenAbsent(t *testing.T) {
	got := handoffToProto(internalexecution.Handoff{ID: "h1", HasValidation: false})
	require.Nil(t, got.GetLastValidation(), "a handoff with no validation must not carry one")
}

func TestVelocityToProto(t *testing.T) {
	got := velocityToProto(internalexecution.VelocityPoint{
		ID: "v1", PlanID: "p1", RunID: "run-1",
		WallTimeSeconds: 120, Tokens: 5000, Iterations: 3,
		Completeness: internalexecution.CompletenessPartial, RecordedAt: "t",
	})
	require.Equal(t, "v1", got.GetId())
	require.Equal(t, int64(120), got.GetWallTimeSeconds())
	require.Equal(t, int64(5000), got.GetTokens())
	require.Equal(t, int32(3), got.GetIterations())
	require.Equal(t, sharedv1.Completeness_COMPLETENESS_PARTIAL, got.GetCompleteness())
}

func TestValidationResultToProto(t *testing.T) {
	got := validationResultToProto(internalexecution.ValidationResult{
		ID: "v1", PlanID: "p1", PhaseID: "ph-1", Verdict: "unknown",
		Staleness: internalplans.StalenessLightlyStale, CommandsRun: []string{"git diff"}, Detail: "d", RanAt: "t",
	})
	require.Equal(t, "v1", got.GetId())
	require.Equal(t, sharedv1.ValidationVerdict_VALIDATION_VERDICT_UNKNOWN, got.GetVerdict())
	require.Equal(t, sharedv1.StalenessTier_STALENESS_TIER_LIGHTLY_STALE, got.GetStaleness())
	require.Equal(t, []string{"git diff"}, got.GetCommandsRun())
}

func TestOrderToInt32(t *testing.T) {
	require.Equal(t, int32(0), orderToInt32(-3))
	require.Equal(t, int32(5), orderToInt32(5))
	require.Equal(t, int32(math.MaxInt32), orderToInt32(math.MaxInt32))
	require.Equal(t, int32(math.MaxInt32), orderToInt32(math.MaxInt32+1))
}

func TestPhaseStatusEnumMapping(t *testing.T) {
	cases := []struct {
		domain internalplans.PhaseStatus
		proto  sharedv1.PhaseStatus
	}{
		{internalplans.PhaseStatusTodo, sharedv1.PhaseStatus_PHASE_STATUS_TODO},
		{internalplans.PhaseStatusActive, sharedv1.PhaseStatus_PHASE_STATUS_ACTIVE},
		{internalplans.PhaseStatusDone, sharedv1.PhaseStatus_PHASE_STATUS_DONE},
		{internalplans.PhaseStatusBlocked, sharedv1.PhaseStatus_PHASE_STATUS_BLOCKED},
	}
	for _, tc := range cases {
		require.Equal(t, tc.proto, phaseStatusToProto(tc.domain))
		require.Equal(t, tc.domain, phaseStatusFromProto(tc.proto))
	}
	require.Equal(t, sharedv1.PhaseStatus_PHASE_STATUS_UNSPECIFIED, phaseStatusToProto(internalplans.PhaseStatus("bogus")))
	require.Equal(t, internalplans.PhaseStatus(""), phaseStatusFromProto(sharedv1.PhaseStatus_PHASE_STATUS_UNSPECIFIED))
}

func TestPlanStatusToProto(t *testing.T) {
	planStatus := map[internalplans.PlanStatus]sharedv1.PlanStatus{
		internalplans.PlanStatusDraft:     sharedv1.PlanStatus_PLAN_STATUS_DRAFT,
		internalplans.PlanStatusActive:    sharedv1.PlanStatus_PLAN_STATUS_ACTIVE,
		internalplans.PlanStatusComplete:  sharedv1.PlanStatus_PLAN_STATUS_COMPLETE,
		internalplans.PlanStatusArchived:  sharedv1.PlanStatus_PLAN_STATUS_ARCHIVED,
		internalplans.PlanStatus("bogus"): sharedv1.PlanStatus_PLAN_STATUS_UNSPECIFIED,
	}
	for in, want := range planStatus {
		require.Equal(t, want, planStatusToProto(in), "planStatusToProto(%q)", in)
	}
}

func TestRefEnumConvertersToProto(t *testing.T) {
	refKind := map[internalplans.ReferenceKind]sharedv1.ReferenceKind{
		internalplans.ReferenceCode:          sharedv1.ReferenceKind_REFERENCE_KIND_CODE,
		internalplans.ReferenceReq:           sharedv1.ReferenceKind_REFERENCE_KIND_REQ,
		internalplans.ReferenceDoc:           sharedv1.ReferenceKind_REFERENCE_KIND_DOC,
		internalplans.ReferenceKind("bogus"): sharedv1.ReferenceKind_REFERENCE_KIND_UNSPECIFIED,
	}
	for in, want := range refKind {
		require.Equal(t, want, refKindToProto(in), "refKindToProto(%q)", in)
	}

	refRes := map[internalplans.ReferenceResolution]sharedv1.ReferenceResolution{
		internalplans.ResolutionResolved:           sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_RESOLVED,
		internalplans.ResolutionUnresolved:         sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_UNRESOLVED,
		internalplans.ResolutionFuture:             sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_FUTURE,
		internalplans.ResolutionMissing:            sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_MISSING,
		internalplans.ResolutionUnspecified:        sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_UNSPECIFIED,
		internalplans.ReferenceResolution("bogus"): sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_UNSPECIFIED,
	}
	for in, want := range refRes {
		require.Equal(t, want, refResolutionToProto(in), "refResolutionToProto(%q)", in)
	}

	staleness := map[internalplans.StalenessTier]sharedv1.StalenessTier{
		internalplans.StalenessFresh:           sharedv1.StalenessTier_STALENESS_TIER_FRESH,
		internalplans.StalenessLightlyStale:    sharedv1.StalenessTier_STALENESS_TIER_LIGHTLY_STALE,
		internalplans.StalenessDefinitelyStale: sharedv1.StalenessTier_STALENESS_TIER_DEFINITELY_STALE,
		internalplans.StalenessUnknown:         sharedv1.StalenessTier_STALENESS_TIER_UNSPECIFIED,
	}
	for in, want := range staleness {
		require.Equal(t, want, stalenessToProto(in), "stalenessToProto(%q)", in)
	}
}

func TestCompletenessToProto(t *testing.T) {
	require.Equal(t, sharedv1.Completeness_COMPLETENESS_FULL, completenessToProto(internalexecution.CompletenessFull))
	require.Equal(t, sharedv1.Completeness_COMPLETENESS_PARTIAL, completenessToProto(internalexecution.CompletenessPartial))
	require.Equal(t, sharedv1.Completeness_COMPLETENESS_UNSPECIFIED, completenessToProto(internalexecution.CompletenessUnspecified))
	require.Equal(t, sharedv1.Completeness_COMPLETENESS_UNSPECIFIED, completenessToProto(internalexecution.Completeness("bogus")))
}

func TestTriageEnumMapping(t *testing.T) {
	cases := []struct {
		domain internalexecution.FindingTriage
		proto  sharedv1.FindingTriage
	}{
		{internalexecution.TriageCandidate, sharedv1.FindingTriage_FINDING_TRIAGE_CANDIDATE},
		{internalexecution.TriagePromoted, sharedv1.FindingTriage_FINDING_TRIAGE_PROMOTED},
		{internalexecution.TriageDismissed, sharedv1.FindingTriage_FINDING_TRIAGE_DISMISSED},
	}
	for _, tc := range cases {
		require.Equal(t, tc.proto, triageToProto(tc.domain))
		require.Equal(t, tc.domain, triageFromProto(tc.proto))
	}
	require.Equal(t, sharedv1.FindingTriage_FINDING_TRIAGE_UNSPECIFIED, triageToProto(internalexecution.FindingTriage("bogus")))
	require.Equal(t, internalexecution.TriageUnspecified, triageFromProto(sharedv1.FindingTriage_FINDING_TRIAGE_UNSPECIFIED))
}

func TestVerdictToProto(t *testing.T) {
	require.Equal(t, sharedv1.ValidationVerdict_VALIDATION_VERDICT_PASS, verdictToProto("pass"))
	require.Equal(t, sharedv1.ValidationVerdict_VALIDATION_VERDICT_FAIL, verdictToProto("fail"))
	require.Equal(t, sharedv1.ValidationVerdict_VALIDATION_VERDICT_UNKNOWN, verdictToProto("unknown"))
	require.Equal(t, sharedv1.ValidationVerdict_VALIDATION_VERDICT_UNSPECIFIED, verdictToProto(""))
	require.Equal(t, sharedv1.ValidationVerdict_VALIDATION_VERDICT_UNSPECIFIED, verdictToProto("bogus"))
}
