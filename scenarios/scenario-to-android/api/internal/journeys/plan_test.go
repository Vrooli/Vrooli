package journeys

import (
	"testing"

	"github.com/stretchr/testify/require"
	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
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

func TestInstalledOfflineTransitionPlanKeepsOnlyTheNetworkChapter(t *testing.T) {
	plan := InstalledAppOfflineTransitionPlan()
	require.Equal(t, OfflineTransitionPlanID, plan.ID)
	require.Len(t, plan.Steps, 7)
	for _, spec := range plan.Steps {
		require.Equal(t, "offline_transition", spec.Arguments["chapter_id"])
		require.NotContains(t, []string{"install", "install-artifact", "uninstall", "revoke-permission", "rotate"}, spec.Action)
	}
	require.Equal(t, "launch", plan.Steps[2].Action)
	full := AndroidPlan().JourneyPlan()
	var published []deliveryramp.JourneyStepSpec
	for _, spec := range full.Steps {
		if spec.Arguments["chapter_id"] == "offline_transition" {
			published = append(published, spec)
		}
	}
	for index, spec := range published {
		focused := plan.Steps[index+3]
		require.Equal(t, spec.ID, focused.ID)
		require.Equal(t, spec.Action, focused.Action)
		require.Equal(t, spec.Arguments["target"], focused.Arguments["target"])
		if spec.Action == "bas-flow" {
			require.Equal(t, DefaultScenarioBASFlowPath, focused.Arguments["flow_path"])
			require.Equal(t, "hello-mobile-smoke", focused.Arguments["flow_reference"])
			require.NotNil(t, focused.Assertion, "both offline and online BAS must be asserted")
		}
	}
}

func TestAndroidPlanDelegatesWebContentToARegisteredBASFlow(t *testing.T) {
	plan := AndroidPlan()
	for _, chapter := range plan.Chapters {
		for _, step := range chapter.Steps {
			if step.Kind == "bas" {
				require.Empty(t, step.RequiredCapabilities, "the app's WebView is a runtime capability checked after launch")
			}
		}
	}
	require.Equal(t, "hello-mobile-smoke", plan.Chapters[0].Steps[len(plan.Chapters[0].Steps)-1].Target)
}

func TestUpdateMigrationUsesAStatePreservationAssertionBeforeAnyRewrite(t *testing.T) {
	plan := AndroidPlan()
	update := chapterByID(t, plan, "update_migration")
	require.Len(t, update.Steps, 3)
	require.Equal(t, "install", update.Steps[0].Reference)
	require.Equal(t, "scenario-app-update-preservation", update.Steps[2].Target)
	require.Equal(t, "bas/cases/mobile/update-preservation-flow.json", update.Steps[2].Arguments["flow_path"])
	journey := plan.JourneyPlan()
	var updateFlow *deliveryramp.JourneyStepSpec
	for index := range journey.Steps {
		if journey.Steps[index].ID == "update_migration/web-flow" {
			updateFlow = &journey.Steps[index]
		}
	}
	require.NotNil(t, updateFlow)
	require.Equal(t, "bas-flow", updateFlow.Action)
	require.Equal(t, "bas/cases/mobile/update-preservation-flow.json", updateFlow.Arguments["flow_path"])
	require.NotNil(t, updateFlow.Assertion)
	require.Equal(t, "app-owned local storage survives the update", updateFlow.Assertion.Expected)
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
	if journey.Steps[0].ID != "install_cold_start/package-preflight" || journey.Steps[1].ID != "install_cold_start/wake" || journey.Steps[6].Action != "bas-flow" {
		t.Fatalf("chapter identity or BAS mapping was lost: %#v", journey.Steps[:3])
	}
	if journey.Steps[0].Assertion != nil || journey.Steps[6].Assertion == nil || journey.Steps[0].Readiness.Timeout <= 0 {
		t.Fatal("chapter assertion was not attached only to its final step")
	}
	if journey.Steps[0].Arguments["target"] != "absent" || journey.Steps[0].Arguments["required_capabilities"] != "app-lifecycle" {
		t.Fatalf("conformance install did not require verified package absence: %#v", journey.Steps[0].Arguments)
	}
	if journey.Steps[1].Arguments["timeout_ms"] != "60000" {
		t.Fatalf("wake step did not retain bounded device timeout: %#v", journey.Steps[1].Arguments)
	}
	if journey.Steps[3].Arguments["timeout_ms"] != "120000" {
		t.Fatalf("install step did not retain wireless-install timeout: %#v", journey.Steps[3].Arguments)
	}
}

func TestAndroidPlanAdaptsNotificationPermissionChapterToOSVersion(t *testing.T) {
	android10, err := AndroidPlanForOSVersion("10")
	require.NoError(t, err)
	require.NoError(t, android10.Validate())
	permission := chapterByID(t, android10, "permission_deny_grace")
	require.Len(t, permission.Steps, 1)
	require.Equal(t, "skip", permission.Steps[0].Kind)
	require.Contains(t, permission.Steps[0].Target, "requires Android 13 or newer")
	journey := android10.JourneyPlan()
	var skipped *deliveryramp.JourneyStepSpec
	for index := range journey.Steps {
		if journey.Steps[index].ID == "permission_deny_grace/not-applicable" {
			skipped = &journey.Steps[index]
		}
	}
	require.NotNil(t, skipped)
	require.Equal(t, "not-applicable", skipped.Action)
	require.Nil(t, skipped.Assertion)

	android13, err := AndroidPlanForOSVersion("13")
	require.NoError(t, err)
	permission = chapterByID(t, android13, "permission_deny_grace")
	require.Equal(t, "revoke-permission", permission.Steps[0].Reference)
	require.Equal(t, "android.permission.POST_NOTIFICATIONS", permission.Steps[0].Target)
	journey = android13.JourneyPlan()
	require.Nil(t, journey.Steps[7].Assertion)
	require.NotNil(t, journey.Steps[9].Assertion)

	_, err = AndroidPlanForOSVersion("unknown")
	require.ErrorContains(t, err, "numeric major version")
}

func chapterByID(t *testing.T, plan Plan, id string) Chapter {
	t.Helper()
	for _, chapter := range plan.Chapters {
		if chapter.ID == id {
			return chapter
		}
	}
	t.Fatalf("chapter %q was not found", id)
	return Chapter{}
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

func TestScenarioAppPlansSeparateTemporaryCleanupFromDurableInstall(t *testing.T) {
	temporary, err := ScenarioAppPlan("temporary")
	require.NoError(t, err)
	require.Equal(t, TemporaryAppTestPlanID, temporary.ID)
	require.Len(t, temporary.Steps, 7)
	require.Equal(t, "screen", temporary.Steps[0].Action)
	require.Equal(t, "wake", temporary.Steps[0].Arguments["target"])
	require.Equal(t, "screen", temporary.Steps[1].Action)
	require.Equal(t, "unlock", temporary.Steps[1].Arguments["target"])
	require.Equal(t, "bas-flow", temporary.Steps[3].Action)
	require.Empty(t, temporary.Steps[3].Arguments["required_capabilities"], "the launched app's WebView is a runtime capability")
	require.Equal(t, "bas-flow", temporary.Steps[6].Action)
	require.Empty(t, temporary.Steps[6].Arguments["required_capabilities"], "the resumed app's WebView is a runtime capability")

	durable, err := ScenarioAppPlan("durable")
	require.NoError(t, err)
	require.Equal(t, DurableAppPlanID, durable.ID)
	require.Len(t, durable.Steps, 8)
	require.Equal(t, "package-state", durable.Steps[7].Action)
	require.Equal(t, "installed", durable.Steps[7].Arguments["target"])

	update, err := ScenarioAppPlan("update")
	require.NoError(t, err)
	require.Equal(t, UpdateAppPlanID, update.ID)
	require.Len(t, update.Steps, 6)
	require.Equal(t, "bas-flow", update.Steps[1].Action)
	require.Equal(t, "scenario-app-update-preservation", update.Steps[1].Arguments["flow_reference"])
	require.Equal(t, "bas/cases/mobile/update-preservation-flow.json", update.Steps[1].Arguments["flow_path"])
	require.Equal(t, "package-state", update.Steps[5].Action)
	require.Equal(t, "installed", update.Steps[5].Arguments["target"])

	verify, err := ScenarioAppPlan("verify")
	require.NoError(t, err)
	require.Equal(t, VerifyAppPlanID, verify.ID)
	require.Equal(t, "stop", verify.Steps[2].Action)
	require.Equal(t, "scenario-app-update-preservation", verify.Steps[4].Arguments["flow_reference"])
	require.Equal(t, "bas/cases/mobile/update-preservation-flow.json", verify.Steps[4].Arguments["flow_path"])
	require.Equal(t, "package-state", verify.Steps[9].Action)
	require.Equal(t, "installed", verify.Steps[9].Arguments["target"])
	for _, step := range verify.Steps {
		require.NotEqual(t, "install-artifact", step.Action, "installed-app verification must not replace the package")
		require.NotEqual(t, "uninstall", step.Action, "installed-app verification must not remove the package")
	}

	if _, err := ScenarioAppPlan("other"); err == nil {
		t.Fatal("unsupported scenario app mode must be refused")
	}
}
