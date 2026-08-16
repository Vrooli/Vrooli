package conformance

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAndroidPlanDeclaresTheTwelveGeneratedAppChapters(t *testing.T) {
	plan := AndroidPlan()
	require.NoError(t, plan.Validate())
	require.Equal(t, PlanID, plan.ID)
	require.Equal(t, Fixture, plan.Fixture)
	require.Len(t, plan.Chapters, 12)
	require.Equal(t, "install_cold_start", plan.Chapters[0].ID)
	require.Equal(t, "clean_uninstall", plan.Chapters[len(plan.Chapters)-1].ID)
	require.NotEmpty(t, plan.Chapters[0].Readiness.TimeoutMS)
	require.NotEmpty(t, plan.Chapters[0].Settle.MaximumMS)
}

func TestAndroidPlanDelegatesWebContentToARegisteredBASFlow(t *testing.T) {
	plan := AndroidPlan()
	var found bool
	for _, step := range plan.Chapters[0].Steps {
		if step.Kind == "bas" {
			found = true
			require.Equal(t, "hello-mobile-smoke", step.Target)
			require.Equal(t, []string{"webview-attach"}, step.RequiredCapabilities)
		}
	}
	require.True(t, found)
}

func TestAndroidPlanRejectsUnboundedReadiness(t *testing.T) {
	plan := AndroidPlan()
	plan.Chapters[0].Readiness.TimeoutMS = 0
	require.ErrorContains(t, plan.Validate(), "unbounded readiness")
}

func TestAndroidPlanFlattensEveryChapterStepForMatrixExecution(t *testing.T) {
	plan := AndroidPlan()
	journey := plan.JourneyPlan()
	count := 0
	for _, chapter := range plan.Chapters {
		count += len(chapter.Steps)
	}
	if len(journey.Steps) != count {
		t.Fatalf("flattened %d steps, want %d", len(journey.Steps), count)
	}
	if journey.Steps[0].ID != "install_cold_start/wake" || journey.Steps[5].Action != "bas-flow" {
		t.Fatalf("chapter identity or BAS mapping was lost: %#v", journey.Steps[:3])
	}
	if journey.Steps[0].Assertion == nil || journey.Steps[0].Readiness.Timeout <= 0 {
		t.Fatal("flattened step lost assertion/readiness policy")
	}
	if journey.Steps[0].Arguments["timeout_ms"] != "60000" {
		t.Fatalf("wake step did not retain bounded device timeout: %#v", journey.Steps[0].Arguments)
	}
	if journey.Steps[2].Arguments["timeout_ms"] != "120000" {
		t.Fatalf("install step did not retain wireless-install timeout: %#v", journey.Steps[2].Arguments)
	}
}

func TestAndroidPlanRoutesWebContentThroughBAS(t *testing.T) {
	for _, chapter := range AndroidPlan().Chapters {
		for _, step := range chapter.Steps {
			if step.Reference == "semantic-assert" || step.Reference == "semantic-target" {
				t.Fatalf("chapter %q routes WebView content through device-control %q", chapter.ID, step.Reference)
			}
		}
	}
}
