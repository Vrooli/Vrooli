package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/automation/engine"
	"github.com/vrooli/browser-automation-studio/automation/events"
	executionwriter "github.com/vrooli/browser-automation-studio/automation/execution-writer"
	"github.com/vrooli/browser-automation-studio/storage"

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
			if got := resolveStepScreenshotPolicy(config.ArtifactCollectionSettings{ScreenshotPolicy: tc.execution, CaptureValidationCheckpoints: true}, tc.action); got != tc.want {
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

func TestCheckpointProfileDisablesAutomaticFramesButRetainsExplicitImages(t *testing.T) {
	settings := config.ResolveArtifactSettings(&basexecution.ArtifactCollectionConfig{Profile: proto.String(config.ProfileCheckpoints)})
	if !settings.CollectScreenshots || !settings.CollectAssertions || !settings.CollectExtractedData {
		t.Fatal("explicit checkpoint evidence would be discarded")
	}
	for _, kind := range []basactions.ActionType{basactions.ActionType_ACTION_TYPE_NAVIGATE, basactions.ActionType_ACTION_TYPE_CLICK, basactions.ActionType_ACTION_TYPE_ASSERT, basactions.ActionType_ACTION_TYPE_SCREENSHOT} {
		got := applyTelemetryDirective(contracts.CompiledInstruction{Action: action(kind)}, &settings)
		if got.Telemetry.GetScreenshot() != basexecution.ScreenshotCapturePolicy_SCREENSHOT_CAPTURE_POLICY_NEVER {
			t.Fatalf("checkpoint action %v enabled automatic screenshot", kind)
		}
	}
	none := config.ResolveArtifactSettings(&basexecution.ArtifactCollectionConfig{Profile: proto.String(config.ProfileNone)})
	if none.CollectScreenshots {
		t.Fatal("none profile must still discard all image artifacts")
	}
}

type screenshotOutcomeSession struct {
	stubEngineSession
	calls          int
	failures       int
	transportError bool
	png            []byte
}

func (s *screenshotOutcomeSession) Run(_ context.Context, instruction contracts.CompiledInstruction) (contracts.StepOutcome, error) {
	if instruction.Action.Type != basactions.ActionType_ACTION_TYPE_SCREENSHOT {
		return contracts.StepOutcome{Success: true}, nil
	}
	s.calls++
	if s.calls <= s.failures {
		if s.transportError {
			return contracts.StepOutcome{}, errors.New("capture transport failed")
		}
		return contracts.StepOutcome{Failure: &contracts.StepFailure{Code: "SCREENSHOT_FAILED", Message: "capture failed", Retryable: true}}, nil
	}
	return contracts.StepOutcome{Success: true, Screenshot: &contracts.Screenshot{Data: s.png, MediaType: "image/png", Width: 1, Height: 1}}, nil
}

// [REQ:BAS-RH-J08] Explicit screenshot failures obey retry/continuation policy
// and remain failed in the persisted outcome, even when continuation is allowed.
func TestExecuteExplicitScreenshotOutcome(t *testing.T) {
	cases := []struct {
		name                         string
		context                      map[string]any
		workflowContinue             *bool
		failures, calls              int
		workflowSuccess, stepSuccess bool
	}{
		{name: "required", failures: 1, calls: 1},
		{name: "workflow-continue", workflowContinue: proto.Bool(true), failures: 1, calls: 1, workflowSuccess: true},
		{name: "node-continue", context: map[string]any{"continueOnError": true}, failures: 1, calls: 1, workflowSuccess: true},
		{name: "node-stop", context: map[string]any{"continueOnError": false}, workflowContinue: proto.Bool(true), failures: 1, calls: 1},
		{name: "success", calls: 1, workflowSuccess: true, stepSuccess: true},
		{name: "retry", context: map[string]any{"resilience": map[string]any{"maxAttempts": 2, "delayMs": 0}}, failures: 1, calls: 2, workflowSuccess: true, stepSuccess: true},
	}
	for _, graph := range []bool{false, true} {
		for _, transportError := range []bool{false, true} {
			for _, tc := range cases {
				t.Run(fmt.Sprintf("graph=%t/transport=%t/%s", graph, transportError, tc.name), func(t *testing.T) {
					var pngData bytes.Buffer
					require.NoError(t, png.Encode(&pngData, image.NewRGBA(image.Rect(0, 0, 1, 1))))
					sess := &screenshotOutcomeSession{failures: tc.failures, transportError: transportError, png: pngData.Bytes()}
					eng := &finalizationEngine{session: sess}

					instructions := []contracts.CompiledInstruction{
						{Index: 0, NodeID: "navigate", Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_NAVIGATE, Params: &basactions.ActionDefinition_Navigate{Navigate: &basactions.NavigateParams{Url: "https://fixture.invalid"}}}},
						{Index: 1, NodeID: "screenshot", Context: tc.context, Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_SCREENSHOT, Params: &basactions.ActionDefinition_Screenshot{Screenshot: &basactions.ScreenshotParams{}}}},
					}
					plan := contracts.ExecutionPlan{ExecutionID: uuid.New(), WorkflowID: uuid.New(), Instructions: instructions}
					if graph {
						plan.Instructions = nil
						plan.Graph = &contracts.PlanGraph{Steps: []contracts.PlanStep{
							{Index: 0, NodeID: "navigate", Action: instructions[0].Action, Outgoing: []contracts.PlanEdge{{Target: "screenshot"}}},
							{Index: 1, NodeID: "screenshot", Action: instructions[1].Action, Context: tc.context},
						}}
					}
					dir := t.TempDir()
					store := storage.NewMemoryStorage()
					writer := executionwriter.NewFileWriter(nil, store, nil, executionwriter.NewStaticRoot(dir))
					err := NewSimpleExecutor(nil).Execute(context.Background(), Request{Plan: plan, EngineName: eng.Name(), EngineFactory: engine.NewStaticFactory(eng), Recorder: writer, EventSink: events.NewMemorySink(contracts.DefaultEventBufferLimits), ContinueOnError: tc.workflowContinue})
					if tc.workflowSuccess {
						require.NoError(t, err)
					} else {
						require.ErrorContains(t, err, "capture")
					}
					require.Equal(t, tc.calls, sess.calls, "declared retry must remain available")
					data, readErr := os.ReadFile(filepath.Join(dir, plan.ExecutionID.String(), "result.json"))
					require.NoError(t, readErr)
					var result struct {
						Entries []struct {
							Context struct {
								Success bool   `json:"success"`
								Error   string `json:"error"`
							} `json:"context"`
						} `json:"entries"`
					}
					require.NoError(t, json.Unmarshal(data, &result))
					require.Len(t, result.Entries, 2)
					require.Equal(t, tc.stepSuccess, result.Entries[1].Context.Success)
					if !tc.stepSuccess {
						require.Contains(t, result.Entries[1].Context.Error, "capture")
						require.Zero(t, store.ObjectCount())
					} else {
						require.Greater(t, store.ObjectCount(), 0)
					}
				})
			}
		}
	}
}

type failedScreenshotStorage struct {
	*storage.MemoryStorage
	err error
}

func (s *failedScreenshotStorage) StoreScreenshot(context.Context, uuid.UUID, string, []byte, string) (*storage.ScreenshotInfo, error) {
	return nil, s.err
}

func TestExecuteReportsScreenshotStorageFailure(t *testing.T) {
	for _, storeErr := range []error{errors.New("image store unavailable"), nil} {
		t.Run(fmt.Sprint(storeErr), func(t *testing.T) {
			var data bytes.Buffer
			require.NoError(t, png.Encode(&data, image.NewRGBA(image.Rect(0, 0, 1, 1))))
			sess := &screenshotOutcomeSession{png: data.Bytes()}
			eng := &finalizationEngine{session: sess}
			dir := t.TempDir()
			writer := executionwriter.NewFileWriter(nil, &failedScreenshotStorage{MemoryStorage: storage.NewMemoryStorage(), err: storeErr}, nil, executionwriter.NewStaticRoot(dir))
			plan := contracts.ExecutionPlan{ExecutionID: uuid.New(), WorkflowID: uuid.New(), Instructions: []contracts.CompiledInstruction{
				{Index: 0, NodeID: "navigate", Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_NAVIGATE, Params: &basactions.ActionDefinition_Navigate{Navigate: &basactions.NavigateParams{Url: "https://fixture.invalid"}}}},
				{Index: 1, NodeID: "capture", Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_SCREENSHOT, Params: &basactions.ActionDefinition_Screenshot{Screenshot: &basactions.ScreenshotParams{}}}},
			}}
			err := NewSimpleExecutor(nil).Execute(context.Background(), Request{Plan: plan, EngineName: eng.Name(), EngineFactory: engine.NewStaticFactory(eng), Recorder: writer, EventSink: events.NewMemorySink(contracts.DefaultEventBufferLimits)})
			require.ErrorContains(t, err, "screenshot")
			if storeErr != nil {
				require.ErrorIs(t, err, storeErr)
			}
			manifest, readErr := os.ReadFile(filepath.Join(dir, plan.ExecutionID.String(), "result.json"))
			require.NoError(t, readErr)
			require.Contains(t, string(manifest), "SCREENSHOT_PERSISTENCE_FAILED")
			require.Contains(t, string(manifest), "EXECUTION_STATUS_FAILED")
		})
	}
}

// Capture has a requested final image; successful setup steps add no images.
// ON_FAILURE still tells the driver to preserve diagnostics for every failure.
func TestCaptureProfileKeepsFailureDiagnosticsWithoutPassiveFrames(t *testing.T) {
	settings := config.ResolveArtifactSettings(&basexecution.ArtifactCollectionConfig{Profile: proto.String("capture")})
	require.True(t, settings.CollectScreenshots)
	require.True(t, settings.CollectConsoleLogs)
	require.True(t, settings.CollectNetworkEvents)
	require.True(t, settings.CollectExtractedData)
	for _, kind := range []basactions.ActionType{basactions.ActionType_ACTION_TYPE_NAVIGATE, basactions.ActionType_ACTION_TYPE_WAIT, basactions.ActionType_ACTION_TYPE_EVALUATE, basactions.ActionType_ACTION_TYPE_ASSERT, basactions.ActionType_ACTION_TYPE_SCREENSHOT} {
		instruction := contracts.CompiledInstruction{Action: action(kind)}
		got := applyTelemetryDirective(instruction, &settings)
		require.Equal(t, basexecution.ScreenshotCapturePolicy_SCREENSHOT_CAPTURE_POLICY_ON_FAILURE, got.Telemetry.GetScreenshot(), kind.String())
	}
}
