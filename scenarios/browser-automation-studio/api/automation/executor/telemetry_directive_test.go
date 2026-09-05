package executor

import (
	"testing"

	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/config"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
)

func action(t basactions.ActionType) *basactions.ActionDefinition {
	return &basactions.ActionDefinition{Type: t}
}

// The execution-wide policy decides intent; the action type decides which steps
// are exempt because their screenshot is the evidence a reader actually wants.
func TestResolveStepScreenshotPolicy(t *testing.T) {
	always := basexecution.ScreenshotCapturePolicy_SCREENSHOT_CAPTURE_POLICY_ALWAYS
	onFailure := basexecution.ScreenshotCapturePolicy_SCREENSHOT_CAPTURE_POLICY_ON_FAILURE
	never := basexecution.ScreenshotCapturePolicy_SCREENSHOT_CAPTURE_POLICY_NEVER
	unspecified := basexecution.ScreenshotCapturePolicy_SCREENSHOT_CAPTURE_POLICY_UNSPECIFIED

	cases := []struct {
		name      string
		execution basexecution.ScreenshotCapturePolicy
		action    *basactions.ActionDefinition
		want      basexecution.ScreenshotCapturePolicy
	}{
		// An unset execution policy must behave exactly as before the directive
		// existed, or every product replay silently loses frames.
		{"unspecified defaults to always", unspecified, action(basactions.ActionType_ACTION_TYPE_CLICK), always},
		{"always stays always", always, action(basactions.ActionType_ACTION_TYPE_EVALUATE), always},
		{"never stays never", never, action(basactions.ActionType_ACTION_TYPE_ASSERT), never},

		// Under on-failure, evidence-bearing steps are promoted back to always.
		{"on-failure promotes assert", onFailure, action(basactions.ActionType_ACTION_TYPE_ASSERT), always},
		{"on-failure promotes navigate", onFailure, action(basactions.ActionType_ACTION_TYPE_NAVIGATE), always},
		{"on-failure promotes screenshot", onFailure, action(basactions.ActionType_ACTION_TYPE_SCREENSHOT), always},

		// ...and leaves the steps that produce an image nobody reads.
		{"on-failure keeps evaluate deferred", onFailure, action(basactions.ActionType_ACTION_TYPE_EVALUATE), onFailure},
		{"on-failure keeps wait deferred", onFailure, action(basactions.ActionType_ACTION_TYPE_WAIT), onFailure},
		{"on-failure keeps click deferred", onFailure, action(basactions.ActionType_ACTION_TYPE_CLICK), onFailure},

		// A nil action must not panic and must not be promoted.
		{"on-failure tolerates nil action", onFailure, nil, onFailure},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveStepScreenshotPolicy(tc.execution, tc.action); got != tc.want {
				t.Fatalf("resolveStepScreenshotPolicy(%v, %v) = %v, want %v", tc.execution, tc.action, got, tc.want)
			}
		})
	}
}

// Without artifact settings the executor must leave the instruction untouched,
// so a caller that never resolved a profile keeps driver defaults.
func TestApplyTelemetryDirectiveNilSettingsLeavesInstructionUnchanged(t *testing.T) {
	instruction := contracts.CompiledInstruction{
		NodeID: "step-1",
		Action: action(basactions.ActionType_ACTION_TYPE_CLICK),
	}

	got := applyTelemetryDirective(instruction, nil)

	if got.Telemetry != nil {
		t.Fatalf("expected no telemetry directive, got %v", got.Telemetry)
	}
}

func TestApplyTelemetryDirectiveStampsResolvedPolicy(t *testing.T) {
	settings := config.DefaultArtifactSettingsForProfile(config.ProfileValidation)
	instruction := contracts.CompiledInstruction{
		NodeID: "store-project-name",
		Action: action(basactions.ActionType_ACTION_TYPE_EVALUATE),
	}

	got := applyTelemetryDirective(instruction, &settings)

	if got.Telemetry == nil {
		t.Fatal("expected a telemetry directive to be stamped")
	}
	want := basexecution.ScreenshotCapturePolicy_SCREENSHOT_CAPTURE_POLICY_ON_FAILURE
	if got.Telemetry.GetScreenshot() != want {
		t.Fatalf("screenshot policy = %v, want %v", got.Telemetry.GetScreenshot(), want)
	}
}

// The ship gate for this whole feature: a product or replay execution must keep
// every frame. The Replay tab and render-video build a storyboard from per-step
// screenshots, so a validation profile leaking into a non-validation caller
// would quietly degrade a user-facing feature.
func TestProductProfilesStillCaptureEveryStep(t *testing.T) {
	profiles := []string{config.ProfileStandard, config.ProfileFull, config.ProfileDebug, config.ProfileMinimal}
	actions := []basactions.ActionType{
		basactions.ActionType_ACTION_TYPE_EVALUATE,
		basactions.ActionType_ACTION_TYPE_CLICK,
		basactions.ActionType_ACTION_TYPE_WAIT,
	}

	for _, profile := range profiles {
		t.Run(profile, func(t *testing.T) {
			settings := config.DefaultArtifactSettingsForProfile(profile)
			for _, actionType := range actions {
				instruction := contracts.CompiledInstruction{Action: action(actionType)}
				got := applyTelemetryDirective(instruction, &settings).Telemetry.GetScreenshot()
				want := basexecution.ScreenshotCapturePolicy_SCREENSHOT_CAPTURE_POLICY_ALWAYS
				if got != want {
					t.Fatalf("profile %s action %v resolved to %v, want %v", profile, actionType, got, want)
				}
			}
		})
	}
}

// The validation profile must keep the artifacts that ARE the test result.
// Trading those away for speed would make the suite fast and useless.
func TestValidationProfileRetainsResultBearingArtifacts(t *testing.T) {
	settings := config.DefaultArtifactSettingsForProfile(config.ProfileValidation)

	if !settings.CollectAssertions {
		t.Error("validation profile must collect assertions")
	}
	if !settings.CollectExtractedData {
		t.Error("validation profile must collect extracted data")
	}
	// Persisting stays on so the frames that DO get captured (failures,
	// asserts, navigations) still reach disk for debugging.
	if !settings.CollectScreenshots {
		t.Error("validation profile must still persist captured screenshots")
	}
}
