package conformance

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAndroidCapabilityPlanContainsOnlyDeviceOwnedChapters(t *testing.T) {
	plan := AndroidCapabilityPlan()
	require.NoError(t, plan.Validate())
	require.Equal(t, PlanID, plan.ID)
	require.Len(t, plan.Chapters, 5)
	require.Equal(t, "device_observation", plan.Chapters[0].ID)
	for _, chapter := range plan.Chapters {
		require.NotContains(t, chapter.Purpose, "fixture")
		for _, step := range chapter.Steps {
			require.NotContains(t, step.Target, "hello-mobile")
		}
	}
}

func TestCapabilityChapterFlowContainsNoGeneratedAppIdentity(t *testing.T) {
	flow := AndroidCapabilityPlan().Chapters[0].Flow()
	require.Equal(t, PlanID+":device_observation", flow.ID)
	require.Len(t, flow.Steps, 1)
	require.Equal(t, "observe", flow.Steps[0].Kind)
	require.NotContains(t, flow.Name, "apk")
}

func TestAndroidCapabilityPlanRejectsUnboundedSteps(t *testing.T) {
	plan := AndroidCapabilityPlan()
	plan.Chapters[0].Steps[0].TimeoutMS = 11 * 60 * 1000
	require.Error(t, plan.Validate())
}
