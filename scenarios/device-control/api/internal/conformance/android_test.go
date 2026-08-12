package conformance

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAndroidPlanContainsAllBoundedPhysicalChapters(t *testing.T) {
	plan := AndroidPlan()
	require.NoError(t, plan.Validate())
	require.Equal(t, PlanID, plan.ID)
	require.Equal(t, FixtureID, plan.Fixture)
	require.Len(t, plan.Chapters, 12)
	require.Equal(t, "install_cold_start", plan.Chapters[0].ID)
	require.Equal(t, "clean_uninstall", plan.Chapters[len(plan.Chapters)-1].ID)
	last := plan.Chapters[len(plan.Chapters)-1]
	require.Equal(t, "package-state", last.Steps[1].Kind)
	require.Equal(t, "absent", last.Steps[1].Target)
}

func TestChapterFlowBindsFixtureIdentityWithoutEmbeddingBytes(t *testing.T) {
	flow := AndroidPlan().Chapters[0].Flow(Fixture{ID: FixtureID, PackageName: "com.example.hello", APKPath: "/tmp/hello.apk", DeepLink: "hello://home"})
	require.Equal(t, "/tmp/hello.apk", flow.Steps[0].Target)
	require.Equal(t, "com.example.hello", flow.Steps[1].Target)
	require.NotContains(t, flow.Name, "hello.apk")
	for _, step := range flow.Steps {
		require.NotContains(t, step.Target, "bytes")
	}
}

func TestAndroidPlanAssertsFixtureStateRatherThanOnlyCapturingScreenshots(t *testing.T) {
	plan := AndroidPlan()
	byID := make(map[string]Chapter, len(plan.Chapters))
	for _, chapter := range plan.Chapters {
		byID[chapter.ID] = chapter
	}
	assertion := func(chapterID, stepID, expected string) {
		for _, step := range byID[chapterID].Steps {
			if step.ID == stepID {
				require.Equal(t, "semantic-assert", step.Kind)
				require.Equal(t, expected, step.Arguments["expected"])
				return
			}
		}
		t.Fatalf("chapter %s missing assertion %s", chapterID, stepID)
	}
	assertion("background_resume", "assert-state", "hello-mobile-state")
	assertion("process_death_restore", "assert-restored-state", "hello-mobile-state")
	assertion("offline_transition", "assert-offline", "Connectivity: offline")
	assertion("offline_transition", "assert-online", "Connectivity: online")
	assertion("deep_link", "assert-route", "Route: home")
}

func TestFixtureValidationFailsClosed(t *testing.T) {
	require.Error(t, (Fixture{ID: FixtureID, PackageName: "com.example.hello"}).Validate())
	require.NoError(t, (Fixture{ID: FixtureID, PackageName: "com.example.hello", APKPath: "/tmp/hello.apk", DeepLink: "hello://home"}).Validate())
}
