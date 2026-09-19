package statusconv

import (
	"testing"

	"github.com/stretchr/testify/require"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
)

func TestPlanStatusFlag(t *testing.T) {
	cases := map[string]sharedv1.PlanStatus{
		"draft":    sharedv1.PlanStatus_PLAN_STATUS_DRAFT,
		"active":   sharedv1.PlanStatus_PLAN_STATUS_ACTIVE,
		"complete": sharedv1.PlanStatus_PLAN_STATUS_COMPLETE,
		"archived": sharedv1.PlanStatus_PLAN_STATUS_ARCHIVED,
		"  Active": sharedv1.PlanStatus_PLAN_STATUS_ACTIVE,
		"DRAFT":    sharedv1.PlanStatus_PLAN_STATUS_DRAFT,
		"":         sharedv1.PlanStatus_PLAN_STATUS_UNSPECIFIED,
		"bogus":    sharedv1.PlanStatus_PLAN_STATUS_UNSPECIFIED,
	}
	for in, want := range cases {
		require.Equalf(t, want, PlanStatusFlag(in), "PlanStatusFlag(%q)", in)
	}
}

func TestPhaseStatusFlag(t *testing.T) {
	cases := map[string]sharedv1.PhaseStatus{
		"todo":    sharedv1.PhaseStatus_PHASE_STATUS_TODO,
		"active":  sharedv1.PhaseStatus_PHASE_STATUS_ACTIVE,
		"done":    sharedv1.PhaseStatus_PHASE_STATUS_DONE,
		"blocked": sharedv1.PhaseStatus_PHASE_STATUS_BLOCKED,
		" Done ":  sharedv1.PhaseStatus_PHASE_STATUS_DONE,
		"":        sharedv1.PhaseStatus_PHASE_STATUS_UNSPECIFIED,
		"unknown": sharedv1.PhaseStatus_PHASE_STATUS_UNSPECIFIED,
	}
	for in, want := range cases {
		require.Equalf(t, want, PhaseStatusFlag(in), "PhaseStatusFlag(%q)", in)
	}
}

func TestPlanStatusLabel(t *testing.T) {
	require.Equal(t, "draft", PlanStatusLabel(sharedv1.PlanStatus_PLAN_STATUS_DRAFT))
	require.Equal(t, "active", PlanStatusLabel(sharedv1.PlanStatus_PLAN_STATUS_ACTIVE))
	require.Equal(t, "complete", PlanStatusLabel(sharedv1.PlanStatus_PLAN_STATUS_COMPLETE))
	require.Equal(t, "archived", PlanStatusLabel(sharedv1.PlanStatus_PLAN_STATUS_ARCHIVED))
	require.Equal(t, "unspecified", PlanStatusLabel(sharedv1.PlanStatus_PLAN_STATUS_UNSPECIFIED))
}

func TestPhaseStatusLabels(t *testing.T) {
	require.Equal(t, "todo", PhaseStatusLabel(sharedv1.PhaseStatus_PHASE_STATUS_TODO))
	require.Equal(t, "active", PhaseStatusLabel(sharedv1.PhaseStatus_PHASE_STATUS_ACTIVE))
	require.Equal(t, "done", PhaseStatusLabel(sharedv1.PhaseStatus_PHASE_STATUS_DONE))
	require.Equal(t, "blocked", PhaseStatusLabel(sharedv1.PhaseStatus_PHASE_STATUS_BLOCKED))
	require.Equal(t, "unspecified", PhaseStatusLabel(sharedv1.PhaseStatus_PHASE_STATUS_UNSPECIFIED))
	require.Equal(t, "todo", PlanPhaseStatusLabel(sharedv1.PhaseStatus_PHASE_STATUS_UNSPECIFIED))
}
