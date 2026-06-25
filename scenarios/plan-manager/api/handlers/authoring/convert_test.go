package authoring

import (
	"testing"

	internalauthoring "plan-manager/internal/authoring"
	internalplans "plan-manager/internal/plans"

	"github.com/stretchr/testify/require"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
)

func TestSessionToProtoMapsFields(t *testing.T) {
	sess := internalauthoring.Session{
		ID:                "s1",
		Title:             "My Plan",
		Slug:              "my-plan",
		CurrentSectionKey: internalauthoring.SectionPurpose,
		Finalized:         true,
		PlanID:            "plan-9",
		Sections: []internalauthoring.Section{
			{Key: internalauthoring.SectionPurpose, Label: "Purpose", Content: "why", Mandatory: true, Filled: true, Autofilled: false},
			{Key: internalauthoring.SectionScope, Label: "Scope", Content: "", Mandatory: true, Filled: false, Autofilled: false},
		},
	}
	got := sessionToProto(sess)
	require.Equal(t, "s1", got.GetId())
	require.Equal(t, "My Plan", got.GetTitle())
	require.Equal(t, "my-plan", got.GetPlanSlug())
	require.Equal(t, "purpose", got.GetCurrentSectionKey())
	require.True(t, got.GetFinalized())
	require.Equal(t, "plan-9", got.GetPlanId())
	require.Len(t, got.GetSections(), 2)
	require.Equal(t, "purpose", got.GetSections()[0].GetKey())
	require.Equal(t, "Purpose", got.GetSections()[0].GetLabel())
	require.Equal(t, "why", got.GetSections()[0].GetContent())
	require.True(t, got.GetSections()[0].GetMandatory())
	require.True(t, got.GetSections()[0].GetFilled())
}

func TestSectionToProto(t *testing.T) {
	got := sectionToProto(internalauthoring.Section{
		Key:        internalauthoring.SectionPhases,
		Label:      "Phases",
		Content:    "### Phase 1 — x",
		Mandatory:  true,
		Filled:     true,
		Autofilled: true,
	})
	require.Equal(t, "phases", got.GetKey())
	require.Equal(t, "Phases", got.GetLabel())
	require.Equal(t, "### Phase 1 — x", got.GetContent())
	require.True(t, got.GetMandatory())
	require.True(t, got.GetFilled())
	require.True(t, got.GetAutofilled())
}

func TestViolationsToProto(t *testing.T) {
	got := violationsToProto([]internalauthoring.StructureViolation{
		{SectionKey: internalauthoring.SectionPurpose, Message: "empty"},
		{SectionKey: internalauthoring.SectionRegressionAnchor, Message: "no anchor"},
	})
	require.Len(t, got, 2)
	require.Equal(t, "purpose", got[0].GetSectionKey())
	require.Equal(t, "empty", got[0].GetMessage())
	require.Equal(t, "regression_anchor", got[1].GetSectionKey())
}

func TestAutofillResultsToProto(t *testing.T) {
	got := autofillResultsToProto([]internalauthoring.AutofillResult{
		{Source: internalauthoring.AutofillRegressionAnchor, SectionKey: internalauthoring.SectionRegressionAnchor, Filled: true, Degraded: false, Detail: "autofilled"},
		{Source: internalauthoring.AutofillReferences, SectionKey: internalauthoring.SectionReferences, Filled: false, Degraded: true, Detail: "code-facts unavailable"},
	})
	require.Len(t, got, 2)
	require.Equal(t, "regression_anchor", got[0].GetSource())
	require.Equal(t, "regression_anchor", got[0].GetSectionKey())
	require.True(t, got[0].GetFilled())
	require.False(t, got[0].GetDegraded())
	require.Equal(t, "references", got[1].GetSource())
	require.True(t, got[1].GetDegraded())
	require.Equal(t, "code-facts unavailable", got[1].GetDetail())
}

func TestAutofillSourcesFromProto(t *testing.T) {
	require.Nil(t, autofillSourcesFromProto(nil), "no sources must map to nil (service then defaults to all)")
	require.Nil(t, autofillSourcesFromProto([]string{}), "empty sources must map to nil")
	got := autofillSourcesFromProto([]string{"regression_anchor", "references"})
	require.Equal(t, []internalauthoring.AutofillSource{
		internalauthoring.AutofillRegressionAnchor,
		internalauthoring.AutofillReferences,
	}, got)
}

func TestPlanToProtoMapsFields(t *testing.T) {
	p := internalplans.Plan{
		ID:     "plan-1",
		Slug:   "slug",
		Title:  "Title",
		Status: internalplans.PlanStatusComplete,
		Phases: []internalplans.Phase{
			{ID: "ph-1", Order: 3, Title: "P", Status: internalplans.PhaseStatusDone},
		},
		References: []internalplans.Reference{
			{ID: "r-1", Kind: internalplans.ReferenceReq, Target: "OT-1", Resolution: internalplans.ResolutionResolved, Staleness: internalplans.StalenessFresh},
		},
		RegressionAnchor: internalplans.RegressionAnchor{Strategy: "captured", BaselineName: "bn"},
	}
	got := planToProto(p)
	require.Equal(t, "plan-1", got.GetId())
	require.Equal(t, sharedv1.PlanStatus_PLAN_STATUS_COMPLETE, got.GetStatus())
	require.Len(t, got.GetPhases(), 1)
	require.Equal(t, int32(3), got.GetPhases()[0].GetOrder())
	require.Equal(t, sharedv1.PhaseStatus_PHASE_STATUS_DONE, got.GetPhases()[0].GetStatus())
	require.Len(t, got.GetReferences(), 1)
	require.Equal(t, sharedv1.ReferenceKind_REFERENCE_KIND_REQ, got.GetReferences()[0].GetKind())
	require.Equal(t, "bn", got.GetRegressionAnchor().GetBaselineName())
}

func TestInt32Of(t *testing.T) {
	cases := []struct {
		name string
		in   int
		want int32
	}{
		{"negative_clamps_to_zero", -1, 0},
		{"zero", 0, 0},
		{"normal", 42, 42},
		// NOTE: unlike handlers/plans + handlers/execution (which clamp to MaxInt32),
		// authoring's int32Of clamps an out-of-range value to 0.
		{"overflow_clamps_to_zero", 1<<31 + 1, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, int32Of(tc.in))
		})
	}
}

func TestEnumConvertersToProto(t *testing.T) {
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

	phaseStatus := map[internalplans.PhaseStatus]sharedv1.PhaseStatus{
		internalplans.PhaseStatusTodo:      sharedv1.PhaseStatus_PHASE_STATUS_TODO,
		internalplans.PhaseStatusActive:    sharedv1.PhaseStatus_PHASE_STATUS_ACTIVE,
		internalplans.PhaseStatusDone:      sharedv1.PhaseStatus_PHASE_STATUS_DONE,
		internalplans.PhaseStatusBlocked:   sharedv1.PhaseStatus_PHASE_STATUS_BLOCKED,
		internalplans.PhaseStatus("bogus"): sharedv1.PhaseStatus_PHASE_STATUS_UNSPECIFIED,
	}
	for in, want := range phaseStatus {
		require.Equal(t, want, phaseStatusToProto(in), "phaseStatusToProto(%q)", in)
	}

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
