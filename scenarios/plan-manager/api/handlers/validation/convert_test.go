package validation

import (
	"testing"

	internalplans "plan-manager/internal/plans"
	internalvalidation "plan-manager/internal/validation"

	"github.com/stretchr/testify/require"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
)

func TestResultToProto(t *testing.T) {
	got := resultToProto(internalvalidation.Result{
		ID:          "v1",
		PlanID:      "p1",
		PhaseID:     "ph-1",
		Verdict:     internalvalidation.VerdictFail,
		Staleness:   internalplans.StalenessDefinitelyStale,
		CommandsRun: []string{"git-control-tower baseline diff --scenario foo --name impl"},
		Detail:      "regression",
		RanAt:       "t",
		CommandFindings: []internalvalidation.CommandFinding{{
			CommandText: "vrooli scenario tost cli-health",
			Verdict:     "invalid",
			Level:       "owner_identified",
			Message:     "unknown_command: command path was not found",
			Location:    "phase.ph-1.intent",
			IssueCodes:  []string{"unknown_command"},
			Suggestions: []string{"vrooli scenario test cli-health"},
			Guidance:    []string{"Fix the command to a current catalog command."},
		}},
	})
	require.Equal(t, "v1", got.GetId())
	require.Equal(t, "p1", got.GetPlanId())
	require.Equal(t, "ph-1", got.GetPhaseId())
	require.Equal(t, sharedv1.ValidationVerdict_VALIDATION_VERDICT_FAIL, got.GetVerdict())
	require.Equal(t, sharedv1.StalenessTier_STALENESS_TIER_DEFINITELY_STALE, got.GetStaleness())
	require.Equal(t, []string{"git-control-tower baseline diff --scenario foo --name impl"}, got.GetCommandsRun())
	require.Equal(t, "regression", got.GetDetail())
	require.Equal(t, "t", got.GetRanAt())
	require.Len(t, got.GetCommandFindings(), 1)
	require.Equal(t, "vrooli scenario tost cli-health", got.GetCommandFindings()[0].GetCommandText())
	require.Equal(t, "invalid", got.GetCommandFindings()[0].GetVerdict())
	require.Equal(t, "owner_identified", got.GetCommandFindings()[0].GetValidationLevel())
	require.Equal(t, "unknown_command: command path was not found", got.GetCommandFindings()[0].GetMessage())
	require.Equal(t, "phase.ph-1.intent", got.GetCommandFindings()[0].GetLocation())
	require.Equal(t, []string{"unknown_command"}, got.GetCommandFindings()[0].GetIssueCodes())
	require.Equal(t, []string{"vrooli scenario test cli-health"}, got.GetCommandFindings()[0].GetSuggestions())
	require.Equal(t, []string{"Fix the command to a current catalog command."}, got.GetCommandFindings()[0].GetGuidance())
}

func TestReferencesToProto(t *testing.T) {
	got := referencesToProto([]internalplans.Reference{
		{ID: "r1", Kind: internalplans.ReferenceCode, Target: "scenarios/foo/x.go", Resolution: internalplans.ResolutionResolved, Staleness: internalplans.StalenessFresh, ChangeFactor: 0.5, Note: "n"},
		{ID: "r2", Kind: internalplans.ReferenceReq, Target: "OT-1", Future: true, Resolution: internalplans.ResolutionFuture},
	})
	require.Len(t, got, 2)
	require.Equal(t, "r1", got[0].GetId())
	require.Equal(t, sharedv1.ReferenceKind_REFERENCE_KIND_CODE, got[0].GetKind())
	require.Equal(t, sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_RESOLVED, got[0].GetResolution())
	require.Equal(t, sharedv1.StalenessTier_STALENESS_TIER_FRESH, got[0].GetStaleness())
	require.InDelta(t, 0.5, got[0].GetChangeFactor(), 1e-9)
	require.True(t, got[1].GetFuture())
	require.Equal(t, sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_FUTURE, got[1].GetResolution())
}

func TestReferencesToProtoEmpty(t *testing.T) {
	require.Empty(t, referencesToProto(nil))
}

func TestVerdictToProto(t *testing.T) {
	cases := []struct {
		domain internalvalidation.Verdict
		proto  sharedv1.ValidationVerdict
	}{
		{internalvalidation.VerdictPass, sharedv1.ValidationVerdict_VALIDATION_VERDICT_PASS},
		{internalvalidation.VerdictFail, sharedv1.ValidationVerdict_VALIDATION_VERDICT_FAIL},
		{internalvalidation.VerdictUnknown, sharedv1.ValidationVerdict_VALIDATION_VERDICT_UNKNOWN},
	}
	for _, tc := range cases {
		require.Equal(t, tc.proto, verdictToProto(tc.domain), "verdict %s", tc.domain)
	}
	require.Equal(t, sharedv1.ValidationVerdict_VALIDATION_VERDICT_UNSPECIFIED, verdictToProto(internalvalidation.VerdictUnspecified))
	require.Equal(t, sharedv1.ValidationVerdict_VALIDATION_VERDICT_UNSPECIFIED, verdictToProto(internalvalidation.Verdict("bogus")))
}

func TestRefKindToProto(t *testing.T) {
	require.Equal(t, sharedv1.ReferenceKind_REFERENCE_KIND_CODE, refKindToProto(internalplans.ReferenceCode))
	require.Equal(t, sharedv1.ReferenceKind_REFERENCE_KIND_REQ, refKindToProto(internalplans.ReferenceReq))
	require.Equal(t, sharedv1.ReferenceKind_REFERENCE_KIND_DOC, refKindToProto(internalplans.ReferenceDoc))
	require.Equal(t, sharedv1.ReferenceKind_REFERENCE_KIND_UNSPECIFIED, refKindToProto(internalplans.ReferenceKind("bogus")))
}

func TestRefResolutionToProto(t *testing.T) {
	require.Equal(t, sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_RESOLVED, refResolutionToProto(internalplans.ResolutionResolved))
	require.Equal(t, sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_UNRESOLVED, refResolutionToProto(internalplans.ResolutionUnresolved))
	require.Equal(t, sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_FUTURE, refResolutionToProto(internalplans.ResolutionFuture))
	require.Equal(t, sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_MISSING, refResolutionToProto(internalplans.ResolutionMissing))
	require.Equal(t, sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_UNSPECIFIED, refResolutionToProto(internalplans.ResolutionUnspecified))
}

func TestStalenessToProto(t *testing.T) {
	require.Equal(t, sharedv1.StalenessTier_STALENESS_TIER_FRESH, stalenessToProto(internalplans.StalenessFresh))
	require.Equal(t, sharedv1.StalenessTier_STALENESS_TIER_LIGHTLY_STALE, stalenessToProto(internalplans.StalenessLightlyStale))
	require.Equal(t, sharedv1.StalenessTier_STALENESS_TIER_DEFINITELY_STALE, stalenessToProto(internalplans.StalenessDefinitelyStale))
	require.Equal(t, sharedv1.StalenessTier_STALENESS_TIER_UNSPECIFIED, stalenessToProto(internalplans.StalenessUnknown))
}
