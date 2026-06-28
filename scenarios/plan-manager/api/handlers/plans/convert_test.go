package plans

import (
	"errors"
	"math"
	"testing"

	internalplans "plan-manager/internal/plans"

	"github.com/stretchr/testify/require"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
)

// fullDomainPlan returns a plan with every field (including every slice) populated
// with valid enum values so a domain→proto→domain round-trip can be asserted with a
// single require.Equal — no nil-vs-empty-slice ambiguity.
func fullDomainPlan() internalplans.Plan {
	ref := internalplans.Reference{
		ID:           "ref-1",
		Kind:         internalplans.ReferenceCode,
		Target:       "scenarios/foo/api/x.go",
		Future:       false,
		Resolution:   internalplans.ResolutionResolved,
		Staleness:    internalplans.StalenessFresh,
		ChangeFactor: 0.25,
		Note:         "a note",
	}
	phase := internalplans.Phase{
		ID:              "ph-1",
		Order:           2,
		Title:           "Phase Title",
		Intent:          "do the thing",
		RequiredReading: []string{"docs/a.md"},
		Reminders:       []string{"remember"},
		BaselineScope:   []string{"scenarios/foo"},
		Acceptance:      "it works",
		Status:          internalplans.PhaseStatusActive,
		References:      []internalplans.Reference{ref},
	}
	return internalplans.Plan{
		ID:          "plan-1",
		Slug:        "the-slug",
		Title:       "The Title",
		Status:      internalplans.PlanStatusActive,
		ContentHash: "deadbeef",
		CreatedAt:   "2026-01-01T00:00:00Z",
		UpdatedAt:   "2026-01-02T00:00:00Z",
		Purpose:     "purpose",
		Scope:       "scope",
		Constraints: "constraints",
		NonGoals:    "non-goals",
		References:  []internalplans.Reference{ref},
		RegressionAnchor: internalplans.RegressionAnchor{
			Strategy:       "captured",
			Scenario:       "foo",
			BaselineName:   "baseline",
			HeadSha:        "abc123",
			AllowlistPaths: []string{"scenarios/foo"},
			Commands:       []string{"git-control-tower baseline diff --scenario foo --name impl"},
			CapturedAt:     "2026-01-01T00:00:00Z",
			Unavailable:    false,
		},
		DefinitionOfDone: "done when green",
		Phases:           []internalplans.Phase{phase},
		Supersedes:       []string{"plan-0"},
		SupersededBy:     []string{"plan-2"},
	}
}

func TestPlanRoundTrip(t *testing.T) {
	p := fullDomainPlan()
	got := planFromProto(planToProto(p))
	require.Equal(t, p, got, "domain→proto→domain must preserve every field")
}

func TestPlanToProtoMapsFields(t *testing.T) {
	p := fullDomainPlan()
	got := planToProto(p)
	require.Equal(t, "plan-1", got.GetId())
	require.Equal(t, "the-slug", got.GetSlug())
	require.Equal(t, "The Title", got.GetTitle())
	require.Equal(t, sharedv1.PlanStatus_PLAN_STATUS_ACTIVE, got.GetStatus())
	require.Equal(t, "deadbeef", got.GetContentHash())
	require.Equal(t, []string{"plan-0"}, got.GetSupersedes())
	require.Equal(t, []string{"plan-2"}, got.GetSupersededBy())
	require.Len(t, got.GetPhases(), 1)
	require.Equal(t, int32(2), got.GetPhases()[0].GetOrder())
	require.Equal(t, sharedv1.PhaseStatus_PHASE_STATUS_ACTIVE, got.GetPhases()[0].GetStatus())
	require.Len(t, got.GetReferences(), 1)
	require.Equal(t, sharedv1.ReferenceKind_REFERENCE_KIND_CODE, got.GetReferences()[0].GetKind())
	require.NotNil(t, got.GetRegressionAnchor())
	require.Equal(t, "abc123", got.GetRegressionAnchor().GetHeadSha())
}

func TestPlanFromProtoNil(t *testing.T) {
	require.Equal(t, internalplans.Plan{}, planFromProto(nil), "nil proto plan must map to a zero domain Plan")
}

func TestPhaseFromProtoNil(t *testing.T) {
	require.Equal(t, internalplans.Phase{}, phaseFromProto(nil), "nil proto phase must map to a zero domain Phase")
}

func TestPhaseFromProtoCheckedRejectsJoinedFields(t *testing.T) {
	_, err := phaseFromProtoChecked(&sharedv1.Phase{
		Title:          "Invalid write",
		LastValidation: &sharedv1.ValidationResult{Id: "validation-1"},
	})
	require.Error(t, err)
	var invalid internalplans.ErrInvalidPlan
	require.True(t, errors.As(err, &invalid))
	require.Contains(t, invalid.Reason, "last_validation")
}

func TestAnchorFromProtoNil(t *testing.T) {
	require.Equal(t, internalplans.RegressionAnchor{}, anchorFromProto(nil), "nil proto anchor must map to a zero domain RegressionAnchor")
}

func TestReferencesFromProtoSkipsNil(t *testing.T) {
	valid := &sharedv1.Reference{Id: "r-1", Kind: sharedv1.ReferenceKind_REFERENCE_KIND_DOC, Target: "t"}
	got := referencesFromProto([]*sharedv1.Reference{nil, valid, nil})
	require.Len(t, got, 1, "nil reference entries must be skipped")
	require.Equal(t, "r-1", got[0].ID)
	require.Equal(t, internalplans.ReferenceDoc, got[0].Kind)
}

func TestEdgeToProto(t *testing.T) {
	got := edgeToProto(internalplans.PlanEdge{FromPlanID: "a", ToPlanID: "b", Kind: "supersedes"})
	require.Equal(t, "a", got.GetFromPlanId())
	require.Equal(t, "b", got.GetToPlanId())
	require.Equal(t, "supersedes", got.GetKind())
}

func TestOrderToInt32(t *testing.T) {
	cases := []struct {
		name string
		in   int
		want int32
	}{
		{"negative_clamps_to_zero", -5, 0},
		{"zero", 0, 0},
		{"small", 7, 7},
		{"max_int32", math.MaxInt32, math.MaxInt32},
		{"overflow_clamps_to_max", math.MaxInt32 + 1, math.MaxInt32},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, orderToInt32(tc.in))
		})
	}
}

func TestPlanStatusEnumMapping(t *testing.T) {
	cases := []struct {
		domain internalplans.PlanStatus
		proto  sharedv1.PlanStatus
	}{
		{internalplans.PlanStatusDraft, sharedv1.PlanStatus_PLAN_STATUS_DRAFT},
		{internalplans.PlanStatusActive, sharedv1.PlanStatus_PLAN_STATUS_ACTIVE},
		{internalplans.PlanStatusComplete, sharedv1.PlanStatus_PLAN_STATUS_COMPLETE},
		{internalplans.PlanStatusArchived, sharedv1.PlanStatus_PLAN_STATUS_ARCHIVED},
	}
	for _, tc := range cases {
		require.Equal(t, tc.proto, planStatusToProto(tc.domain), "to proto: %s", tc.domain)
		require.Equal(t, tc.domain, planStatusFromProto(tc.proto), "from proto: %s", tc.proto)
	}
	// Unknown/default fallbacks.
	require.Equal(t, sharedv1.PlanStatus_PLAN_STATUS_UNSPECIFIED, planStatusToProto(internalplans.PlanStatus("bogus")))
	require.Equal(t, internalplans.PlanStatus(""), planStatusFromProto(sharedv1.PlanStatus_PLAN_STATUS_UNSPECIFIED))
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
		require.Equal(t, tc.proto, phaseStatusToProto(tc.domain), "to proto: %s", tc.domain)
		require.Equal(t, tc.domain, phaseStatusFromProto(tc.proto), "from proto: %s", tc.proto)
	}
	require.Equal(t, sharedv1.PhaseStatus_PHASE_STATUS_UNSPECIFIED, phaseStatusToProto(internalplans.PhaseStatus("bogus")))
	require.Equal(t, internalplans.PhaseStatus(""), phaseStatusFromProto(sharedv1.PhaseStatus_PHASE_STATUS_UNSPECIFIED))
}

func TestRefKindEnumMapping(t *testing.T) {
	cases := []struct {
		domain internalplans.ReferenceKind
		proto  sharedv1.ReferenceKind
	}{
		{internalplans.ReferenceCode, sharedv1.ReferenceKind_REFERENCE_KIND_CODE},
		{internalplans.ReferenceReq, sharedv1.ReferenceKind_REFERENCE_KIND_REQ},
		{internalplans.ReferenceDoc, sharedv1.ReferenceKind_REFERENCE_KIND_DOC},
	}
	for _, tc := range cases {
		require.Equal(t, tc.proto, refKindToProto(tc.domain), "to proto: %s", tc.domain)
		require.Equal(t, tc.domain, refKindFromProto(tc.proto), "from proto: %s", tc.proto)
	}
	require.Equal(t, sharedv1.ReferenceKind_REFERENCE_KIND_UNSPECIFIED, refKindToProto(internalplans.ReferenceKind("bogus")))
	// NOTE: refKindFromProto's default is CODE, not unspecified — an unknown wire
	// kind is read back as code (the most common locator).
	require.Equal(t, internalplans.ReferenceCode, refKindFromProto(sharedv1.ReferenceKind_REFERENCE_KIND_UNSPECIFIED))
}

func TestRefResolutionEnumMapping(t *testing.T) {
	cases := []struct {
		domain internalplans.ReferenceResolution
		proto  sharedv1.ReferenceResolution
	}{
		{internalplans.ResolutionResolved, sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_RESOLVED},
		{internalplans.ResolutionUnresolved, sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_UNRESOLVED},
		{internalplans.ResolutionFuture, sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_FUTURE},
		{internalplans.ResolutionMissing, sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_MISSING},
	}
	for _, tc := range cases {
		require.Equal(t, tc.proto, refResolutionToProto(tc.domain), "to proto: %s", tc.domain)
		require.Equal(t, tc.domain, refResolutionFromProto(tc.proto), "from proto: %s", tc.proto)
	}
	require.Equal(t, sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_UNSPECIFIED, refResolutionToProto(internalplans.ReferenceResolution("bogus")))
	require.Equal(t, internalplans.ResolutionUnspecified, refResolutionFromProto(sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_UNSPECIFIED))
}

func TestStalenessEnumMapping(t *testing.T) {
	cases := []struct {
		domain internalplans.StalenessTier
		proto  sharedv1.StalenessTier
	}{
		{internalplans.StalenessFresh, sharedv1.StalenessTier_STALENESS_TIER_FRESH},
		{internalplans.StalenessLightlyStale, sharedv1.StalenessTier_STALENESS_TIER_LIGHTLY_STALE},
		{internalplans.StalenessDefinitelyStale, sharedv1.StalenessTier_STALENESS_TIER_DEFINITELY_STALE},
	}
	for _, tc := range cases {
		require.Equal(t, tc.proto, stalenessToProto(tc.domain), "to proto: %s", tc.domain)
		require.Equal(t, tc.domain, stalenessFromProto(tc.proto), "from proto: %s", tc.proto)
	}
	require.Equal(t, sharedv1.StalenessTier_STALENESS_TIER_UNSPECIFIED, stalenessToProto(internalplans.StalenessTier("bogus")))
	require.Equal(t, internalplans.StalenessUnknown, stalenessFromProto(sharedv1.StalenessTier_STALENESS_TIER_UNSPECIFIED))
}
