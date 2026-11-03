package journeys

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

type fakeDevice struct {
	leaseLost bool
	noOffsets bool
	actionErr error
	actions   []map[string]string
	evidence  []deliveryramp.EvidenceReference
	unlocked  []string
}

func (f *fakeDevice) Unlock(_ context.Context, _ Lease, profileID string) error {
	f.unlocked = append(f.unlocked, profileID)
	return nil
}

type chapterFakeDevice struct {
	fakeDevice
	started []string
	stopped []string
	stopErr error
}

type evidenceFakeDevice struct {
	fakeDevice
	clockCalls int
}

func (f *evidenceFakeDevice) StartLogCapture(context.Context, Lease) error { return nil }

func (f *evidenceFakeDevice) StopLogCapture(context.Context, Lease) (LogCaptureArtifact, error) {
	return LogCaptureArtifact{Reference: deliveryramp.EvidenceReference{ID: "logcat-1", Kind: "log", Checksum: "sha256:logcat", Redacted: true}}, nil
}

func (f *evidenceFakeDevice) SampleClock(context.Context, Lease) (ClockSample, error) {
	f.clockCalls++
	deviceTime := time.Unix(1_755_259_200, 0).UTC()
	hostTime := deviceTime.Add(10 * time.Second)
	return ClockSample{HostTime: hostTime, DeviceTime: deviceTime, OffsetMs: 10000, UncertaintyMs: 2, Evidence: deliveryramp.EvidenceReference{ID: "clock-" + string(rune('0'+f.clockCalls)), Kind: "log", Checksum: "sha256:clock", Redacted: true}}, nil
}

func (f *chapterFakeDevice) StartChapterRecording(_ context.Context, _ Lease, chapter string) error {
	f.started = append(f.started, chapter)
	return nil
}

func (f *chapterFakeDevice) StopChapterRecording(_ context.Context, _ Lease, chapter string) (RecordingArtifact, error) {
	f.stopped = append(f.stopped, chapter)
	if f.stopErr != nil {
		return RecordingArtifact{}, f.stopErr
	}
	return RecordingArtifact{Reference: deliveryramp.EvidenceReference{ID: "video-" + chapter, Checksum: "sha256:" + chapter, Redacted: true, Kind: "video"}, StartMs: 10, EndMs: 100, HasOffsets: true}, nil
}

func (f *chapterFakeDevice) FinalizeReviewRecording(_ context.Context, _ Lease, chapters []deliveryramp.EvidenceReference) (ReviewRecording, error) {
	if len(chapters) == 0 {
		return ReviewRecording{}, errors.New("review received no chapter list")
	}
	return ReviewRecording{Reference: deliveryramp.EvidenceReference{ID: "review-1", Kind: "video", Checksum: "sha256:review", Redacted: true}, Path: "/tmp/review.mp4"}, nil
}

func TestDriverKeepsBehavioralJourneyWhenChapterVideoFails(t *testing.T) {
	device := &chapterFakeDevice{stopErr: errors.New("native recorder unavailable")}
	request := testRequest()
	request.Plan.Steps = []deliveryramp.JourneyStepSpec{
		{ID: "observe", Action: "observe", Arguments: map[string]string{"chapter_id": "cold-start"}},
	}
	result, err := (Driver{Devices: device}).Execute(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if result.Disposition != deliveryramp.DispositionPass || result.Steps[0].VideoDisposition != deliveryramp.StepUnavailable {
		t.Fatalf("video failure degraded behavioral journey: %#v", result)
	}
	if result.Steps[0].VideoError == "" {
		t.Fatal("video failure did not retain a reason")
	}
}

func (f *fakeDevice) Acquire(context.Context, string, string, time.Duration) (Lease, error) {
	return Lease{ID: "lease-1", Token: "token-1"}, nil
}

func (f *fakeDevice) ValidateLease(context.Context, Lease) error {
	if f.leaseLost {
		return errors.New("lease expired")
	}
	return nil
}

func (f *fakeDevice) Execute(_ context.Context, _ Lease, _ string, arguments map[string]string) (ActionResult, error) {
	f.actions = append(f.actions, arguments)
	if f.actionErr != nil {
		return ActionResult{}, f.actionErr
	}
	return ActionResult{Evidence: f.evidence}, nil
}
func (f *fakeDevice) StartRecording(context.Context, Lease) error { return nil }
func (f *fakeDevice) StopRecording(context.Context, Lease) (RecordingArtifact, error) {
	return RecordingArtifact{Reference: deliveryramp.EvidenceReference{ID: "video-1", Checksum: "sha256:video", Redacted: true, Kind: "video"}, StartMs: 10, EndMs: 100, HasOffsets: !f.noOffsets}, nil
}
func (f *fakeDevice) Release(context.Context, Lease) error { return nil }

type fakeBAS struct{ executeErr error }

func (fakeBAS) ValidateFlow(context.Context, BASRequest) error { return nil }

func (f fakeBAS) Execute(context.Context, BASRequest) (BASResult, error) {
	if f.executeErr != nil {
		return BASResult{}, f.executeErr
	}
	return BASResult{Completed: true}, nil
}

func testRequest() deliveryramp.DriverRequest {
	target := deliveryramp.Target{ID: "android:emulator:local", Label: "hello-mobile", Platform: "android", Available: true, Capabilities: []string{"android-webview"}, Transport: deliveryramp.Transport{Kind: deliveryramp.TransportLocal, ID: "local", Available: true}}
	return deliveryramp.DriverRequest{RunID: "run-1", Cell: deliveryramp.Cell{ID: "cell-1", Target: target}, Artifact: deliveryramp.Artifact{ImmutableRef: "android-debug:abc"}, Plan: deliveryramp.JourneyPlan{ID: "android-smoke-v1", Capability: "android-webview", Profile: "release", Steps: []deliveryramp.JourneyStepSpec{{ID: "bas", Action: "bas-flow", Purpose: "run BAS flow"}}}}
}

func TestDriverRequiresRecordingOffsets(t *testing.T) {
	device := &fakeDevice{noOffsets: true}
	_, err := (Driver{Devices: device, BAS: fakeBAS{}}).Execute(context.Background(), testRequest())
	if err == nil {
		t.Fatal("expected missing offset failure")
	}
}

func TestDriverFailsWhenLeaseIsLost(t *testing.T) {
	device := &fakeDevice{leaseLost: true}
	_, err := (Driver{Devices: device, BAS: fakeBAS{}}).Execute(context.Background(), testRequest())
	if err == nil {
		t.Fatal("expected lease loss failure")
	}
}

func TestDriverRestoresNetworkWhenOfflineChapterFails(t *testing.T) {
	device := &fakeDevice{}
	originalFailure := errors.New("offline WebView attach failed")
	request := testRequest()
	request.Cell.Target.Capabilities = append(request.Cell.Target.Capabilities, "network-control")
	request.Artifact.Metadata = map[string]string{"package_name": "com.example.app"}
	request.Plan.Steps = []deliveryramp.JourneyStepSpec{
		{ID: "offline", Action: "network", Arguments: map[string]string{"chapter_id": "offline_transition", "target": "offline"}},
		{ID: "web-flow-offline", Action: "bas-flow", Arguments: map[string]string{"chapter_id": "offline_transition"}},
		{ID: "online", Action: "network", Arguments: map[string]string{"chapter_id": "offline_transition", "target": "online"}},
	}

	result, err := (Driver{Devices: device, BAS: fakeBAS{executeErr: originalFailure}}).Execute(context.Background(), request)
	if !errors.Is(err, originalFailure) {
		t.Fatalf("original offline flow failure was not preserved: %v", err)
	}
	if len(device.actions) != 2 || device.actions[0]["target"] != "offline" || device.actions[1]["target"] != "online" {
		t.Fatalf("network was not restored through Device Control: %#v", device.actions)
	}
	if len(result.Steps) != 2 || result.Steps[1].ID != "device-cleanup/network-online" || result.Steps[1].Disposition != deliveryramp.StepPassed {
		t.Fatalf("successful network recovery was not retained: %#v", result.Steps)
	}
}

type networkRestoreFailureDevice struct {
	fakeDevice
	restoreErr error
}

func (f *networkRestoreFailureDevice) Execute(ctx context.Context, lease Lease, action string, arguments map[string]string) (ActionResult, error) {
	if arguments["target"] == "online" {
		f.actions = append(f.actions, arguments)
		return ActionResult{}, f.restoreErr
	}
	return f.fakeDevice.Execute(ctx, lease, action, arguments)
}

func TestDriverPreservesOfflineAndNetworkRecoveryFailures(t *testing.T) {
	originalFailure := errors.New("offline WebView attach failed")
	restoreFailure := errors.New("Device Control network recovery failed")
	device := &networkRestoreFailureDevice{restoreErr: restoreFailure}
	request := testRequest()
	request.Cell.Target.Capabilities = append(request.Cell.Target.Capabilities, "network-control")
	request.Artifact.Metadata = map[string]string{"package_name": "com.example.app"}
	request.Plan.Steps = []deliveryramp.JourneyStepSpec{
		{ID: "offline", Action: "network", Arguments: map[string]string{"chapter_id": "offline_transition", "target": "offline"}},
		{ID: "web-flow-offline", Action: "bas-flow", Arguments: map[string]string{"chapter_id": "offline_transition"}},
		{ID: "online", Action: "network", Arguments: map[string]string{"chapter_id": "offline_transition", "target": "online"}},
	}

	result, err := (Driver{Devices: device, BAS: fakeBAS{executeErr: originalFailure}}).Execute(context.Background(), request)
	if !errors.Is(err, originalFailure) || !errors.Is(err, restoreFailure) {
		t.Fatalf("journey did not preserve both the chapter and recovery failures: %v", err)
	}
	if len(device.actions) != 2 || device.actions[0]["target"] != "offline" || device.actions[1]["target"] != "online" {
		t.Fatalf("network recovery did not run through Device Control: %#v", device.actions)
	}
	if len(result.Steps) != 2 {
		t.Fatalf("expected interrupted chapter and retained recovery step, got %d steps: %#v", len(result.Steps), result.Steps)
	}
	recovery := result.Steps[1]
	if recovery.ID != "device-cleanup/network-online" || recovery.Disposition != deliveryramp.StepFailed || !strings.Contains(recovery.Error, restoreFailure.Error()) {
		t.Fatalf("failed network recovery was not retained: %#v", recovery)
	}
}

func TestDriverCompletesBASStepWithReferenceOnlyVideo(t *testing.T) {
	result, err := (Driver{Devices: &fakeDevice{}, BAS: fakeBAS{}}).Execute(context.Background(), testRequest())
	if err != nil {
		t.Fatal(err)
	}
	if result.Disposition != deliveryramp.DispositionPass || result.Steps[0].VideoStartOffsetMs == nil || result.Steps[0].VideoEndOffsetMs == nil {
		t.Fatalf("unexpected result: %#v", result)
	}
}

type scenarioInstallFakeDevice struct {
	preflightErr     error
	packageStateErr  error
	packageState     InstalledPackageState
	postInstallState *InstalledPackageState
	installErr       error
	packageReads     int
	calls            []map[string]string
	actions          []string
	startedRecording int
	state            DeviceStateSnapshot
	sleepReads       int
	sleepReadsLeft   int
}

func (f *scenarioInstallFakeDevice) ReadPackageState(context.Context, Lease, string) (InstalledPackageState, error) {
	f.packageReads++
	if f.packageStateErr != nil {
		return InstalledPackageState{}, f.packageStateErr
	}
	return f.packageState, nil
}

func (f *scenarioInstallFakeDevice) ReadState(context.Context, string) (DeviceStateSnapshot, error) {
	if f.state.ScreenState == "" {
		f.state = DeviceStateSnapshot{ForegroundPackage: "com.android.launcher", ScreenState: "off", LockState: "locked"}
	}
	if f.sleepReadsLeft > 0 {
		f.sleepReadsLeft--
		if f.sleepReadsLeft == 0 {
			f.state.ScreenState = "off"
			f.state.LockState = "locked"
		}
	}
	return f.state, nil
}

func (f *scenarioInstallFakeDevice) Unlock(context.Context, Lease, string) error {
	f.state.LockState = "unlocked"
	return nil
}

func (f *scenarioInstallFakeDevice) Acquire(context.Context, string, string, time.Duration) (Lease, error) {
	return Lease{ID: "lease-app-test", DeviceID: "android-phone", Token: "lease-token"}, nil
}

func (f *scenarioInstallFakeDevice) ValidateLease(context.Context, Lease) error { return nil }

func (f *scenarioInstallFakeDevice) Execute(_ context.Context, _ Lease, action string, arguments map[string]string) (ActionResult, error) {
	copied := make(map[string]string, len(arguments))
	for key, value := range arguments {
		copied[key] = value
	}
	f.calls = append(f.calls, copied)
	f.actions = append(f.actions, action)
	if action == "install-artifact" {
		if f.postInstallState != nil {
			f.packageState = *f.postInstallState
		}
		if f.installErr != nil {
			return ActionResult{}, f.installErr
		}
	}
	if arguments["step_id"] == "install-preflight/package-state" && f.preflightErr != nil {
		return ActionResult{}, f.preflightErr
	}
	if arguments["target"] == "wake" && arguments["step_id"] != "" {
		f.state.ScreenState = "on"
	}
	if arguments["target"] == "sleep" && arguments["step_id"] != "" {
		if f.sleepReads > 0 {
			f.sleepReadsLeft = f.sleepReads
		} else {
			f.state.ScreenState = "off"
			f.state.LockState = "locked"
		}
	}
	return ActionResult{Evidence: []deliveryramp.EvidenceReference{{ID: "evidence-" + arguments["step_id"], Kind: "device-control", Checksum: "sha256:evidence"}}}, nil
}

func TestConformanceInstallUsesImmutableArtifactOwnerPath(t *testing.T) {
	device := &scenarioInstallFakeDevice{}
	request := deliveryramp.DriverRequest{
		RunID:    "run-install-route",
		Cell:     deliveryramp.Cell{ID: "cell-install-route", Target: deliveryramp.Target{ID: "android-phone", Label: "hello-mobile", Platform: "android", Available: true}},
		Artifact: deliveryramp.Artifact{ImmutableRef: "android-debug:" + strings.Repeat("a", 64), Checksum: "sha256:" + strings.Repeat("a", 64), Metadata: map[string]string{"package_name": "com.example.hello", "version_name": "1.2.3", "apk_path": "/private/build/app.apk"}},
		Plan:     deliveryramp.JourneyPlan{ID: PlanID, Steps: []deliveryramp.JourneyStepSpec{{ID: "install_cold_start/install", Action: "install", Arguments: map[string]string{"target": ""}}}},
	}
	result, err := (Driver{Devices: device}).Execute(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, deliveryramp.DispositionPass, result.Disposition)
	require.Equal(t, []string{"install-artifact"}, device.actions)
	require.Equal(t, request.Artifact.ImmutableRef, device.calls[0]["artifact_ref"])
	require.Equal(t, "1.2.3", device.calls[0]["version_name"])
	require.Equal(t, "com.example.hello", device.calls[0]["package"])
	require.NotContains(t, device.calls[0], "value", "the client must not pass a private APK path to the generic installer")
}

func (f *scenarioInstallFakeDevice) StartRecording(context.Context, Lease) error {
	f.startedRecording++
	return nil
}

func (f *scenarioInstallFakeDevice) StopRecording(context.Context, Lease) (RecordingArtifact, error) {
	return RecordingArtifact{Reference: deliveryramp.EvidenceReference{ID: "recording", Checksum: "sha256:recording", Redacted: true, Kind: "video"}, StartMs: 10, EndMs: 100, HasOffsets: true}, nil
}

func (f *scenarioInstallFakeDevice) Release(context.Context, Lease) error { return nil }

type countingBAS struct {
	calls          int
	requests       []BASRequest
	preflightCalls []BASRequest
	preflightErr   error
}

func (f *countingBAS) ValidateFlow(_ context.Context, request BASRequest) error {
	f.preflightCalls = append(f.preflightCalls, request)
	return f.preflightErr
}

func (f *countingBAS) Execute(_ context.Context, request BASRequest) (BASResult, error) {
	f.calls++
	f.requests = append(f.requests, request)
	return BASResult{Completed: true, ExecutionID: fmt.Sprintf("bas-execution-%d", f.calls)}, nil
}

type failingBAS struct{ err error }

func (failingBAS) ValidateFlow(context.Context, BASRequest) error { return nil }

func (f failingBAS) Execute(context.Context, BASRequest) (BASResult, error) {
	return BASResult{}, f.err
}

type transientThenPassingBAS struct{ calls int }

func (*transientThenPassingBAS) ValidateFlow(context.Context, BASRequest) error { return nil }

func (b *transientThenPassingBAS) Execute(context.Context, BASRequest) (BASResult, error) {
	b.calls++
	if b.calls == 1 {
		return BASResult{ExecutionID: "bas-closed-context"}, &transientWebViewClosureError{ExecutionID: "bas-closed-context", NodeIDs: []string{"open"}}
	}
	return BASResult{Completed: true, ExecutionID: fmt.Sprintf("bas-pass-%d", b.calls)}, nil
}

func appInstallRequest(t *testing.T, mode string) deliveryramp.DriverRequest {
	t.Helper()
	plan, err := ScenarioAppPlan(mode)
	if err != nil {
		t.Fatal(err)
	}
	request := testRequest()
	request.Plan = plan
	request.Cell.Target.Capabilities = []string{"app-lifecycle", "webview-attach", "input", "screen-control"}
	request.Artifact = deliveryramp.Artifact{
		ImmutableRef: "android-debug:" + strings.Repeat("a", 64),
		Checksum:     "sha256:" + strings.Repeat("a", 64),
		Metadata: map[string]string{
			"package_name": "com.example.generated", "version_name": "1.2.3", "version_code": "12", "signing": "debug-only",
			"scenario_name": "hello-mobile", "auth_profile_id": "profile-1",
		},
	}
	return request
}

func TestTemporaryScenarioAppRefusesPreexistingPackageBeforeCaptureOrInstall(t *testing.T) {
	device := &scenarioInstallFakeDevice{preflightErr: errors.New("package already installed")}
	bas := &countingBAS{}
	result, err := (Driver{Devices: device, BAS: bas}).Execute(context.Background(), appInstallRequest(t, "temporary"))
	if err != nil {
		t.Fatalf("prestate refusal should be a structured journey result, got %v", err)
	}
	if result.Disposition != deliveryramp.DispositionFailed || len(device.calls) != 1 || device.calls[0]["expected"] != "absent" {
		t.Fatalf("existing package was not refused before mutation: result=%#v calls=%#v", result, device.calls)
	}
	if device.startedRecording != 0 || bas.calls != 0 {
		t.Fatalf("prestate refusal started capture or BAS: recording=%d bas=%d", device.startedRecording, bas.calls)
	}
}

func TestTemporaryScenarioAppUsesImmutableInstallAndVerifiesCleanup(t *testing.T) {
	device := &scenarioInstallFakeDevice{}
	bas := &countingBAS{}
	result, err := (Driver{Devices: device, BAS: bas}).Execute(context.Background(), appInstallRequest(t, "temporary"))
	if err != nil || result.Disposition != deliveryramp.DispositionPass {
		t.Fatalf("temporary scenario app journey failed: result=%#v err=%v", result, err)
	}
	if bas.calls != 2 {
		t.Fatalf("cold and warm BAS flows were not both executed: %d", bas.calls)
	}
	var installSeen, uninstallSeen, absentVerified bool
	for _, call := range device.calls {
		switch call["step_id"] {
		case "artifact-install/install":
			installSeen = call["artifact_ref"] == "android-debug:"+strings.Repeat("a", 64) && call["package"] == "com.example.generated" && call["version_name"] == "1.2.3" && call["timeout_ms"] == "120000"
		case "temporary-cleanup/uninstall":
			uninstallSeen = call["package"] == "com.example.generated"
		case "temporary-cleanup/verify-absent":
			absentVerified = call["expected"] == "absent"
		}
	}
	if !installSeen || !uninstallSeen || !absentVerified {
		t.Fatalf("immutable install or cleanup contract missing: install=%v uninstall=%v absent=%v calls=%#v", installSeen, uninstallSeen, absentVerified, device.calls)
	}
	if device.state.ScreenState != "off" || device.state.LockState != "locked" {
		t.Fatalf("temporary app test did not restore the original screen state: %#v", device.state)
	}
	if result.Steps[len(result.Steps)-1].ID != "device-cleanup/restore-screen-state" || result.Steps[len(result.Steps)-1].AssertionStatus != "observed" {
		t.Fatalf("verified screen restoration was not retained: %#v", result.Steps[len(result.Steps)-1])
	}
	if result.Events[len(result.Events)-1].Observed != "BAS flow completed; execution_id=bas-execution-2" {
		t.Fatalf("BAS execution identity was not retained: %#v", result.Events)
	}
}

func TestTemporaryScenarioAppCleansUpAfterBASFailure(t *testing.T) {
	device := &scenarioInstallFakeDevice{}
	basErr := errors.New("selector wait timed out")
	result, err := (Driver{Devices: device, BAS: failingBAS{err: basErr}}).Execute(context.Background(), appInstallRequest(t, "temporary"))
	if !errors.Is(err, basErr) {
		t.Fatalf("BAS failure was not retained: result=%#v err=%v", result, err)
	}
	var uninstallSeen, absentVerified bool
	for _, call := range device.calls {
		switch call["step_id"] {
		case "temporary-cleanup/uninstall":
			uninstallSeen = call["package"] == "com.example.generated"
		case "temporary-cleanup/verify-absent":
			absentVerified = call["expected"] == "absent"
		}
	}
	if !uninstallSeen || !absentVerified {
		t.Fatalf("temporary app failure did not verify cleanup: uninstall=%v absent=%v calls=%#v", uninstallSeen, absentVerified, device.calls)
	}
	if device.state.ScreenState != "off" || device.state.LockState != "locked" {
		t.Fatalf("temporary app failure did not restore the initial screen state: %#v", device.state)
	}
}

type temporaryInstallTimeoutDevice struct {
	*scenarioInstallFakeDevice
	installErr error
}

func (f *temporaryInstallTimeoutDevice) Execute(ctx context.Context, lease Lease, action string, arguments map[string]string) (ActionResult, error) {
	switch arguments["step_id"] {
	case "artifact-install/install":
		f.postInstallState = &InstalledPackageState{Installed: true, Version: "1.2.3", VersionCode: 12}
		f.scenarioInstallFakeDevice.installErr = f.installErr
		return f.scenarioInstallFakeDevice.Execute(ctx, lease, action, arguments)
	case "temporary-cleanup/uninstall":
		result, err := f.scenarioInstallFakeDevice.Execute(ctx, lease, action, arguments)
		f.packageState = InstalledPackageState{}
		return result, err
	case "temporary-cleanup/verify-absent":
		if f.packageState.Installed {
			return ActionResult{}, fmt.Errorf("package %s remains installed after temporary cleanup", arguments["package"])
		}
	}
	return f.scenarioInstallFakeDevice.Execute(ctx, lease, action, arguments)
}

func TestTemporaryScenarioAppCleansUpAfterUncertainInstallFailure(t *testing.T) {
	installErr := errors.New("install command timed out after package side effect")
	device := &temporaryInstallTimeoutDevice{scenarioInstallFakeDevice: &scenarioInstallFakeDevice{}, installErr: installErr}
	result, err := (Driver{Devices: device, BAS: &countingBAS{}}).Execute(context.Background(), appInstallRequest(t, "temporary"))
	require.NoError(t, err, "the matrix result should retain the owner failure without losing cleanup evidence")
	require.Equal(t, deliveryramp.DispositionFailed, result.Disposition)
	require.Contains(t, result.DegradedReason, installErr.Error())

	callIndex := func(id string) int {
		for index := range device.calls {
			if device.calls[index]["step_id"] == id {
				return index
			}
		}
		return -1
	}
	installIndex := callIndex("artifact-install/install")
	uninstallIndex := callIndex("temporary-cleanup/uninstall")
	absenceIndex := callIndex("temporary-cleanup/verify-absent")
	require.GreaterOrEqual(t, installIndex, 0)
	require.Greater(t, uninstallIndex, installIndex, "uncertain install must be followed by uninstall")
	require.Greater(t, absenceIndex, uninstallIndex, "package absence must be verified after uninstall")
	cleanupIndex := -1
	for index := range result.Steps {
		if result.Steps[index].ID == "temporary-cleanup/verify-absent" {
			cleanupIndex = index
			break
		}
	}
	require.NotEqual(t, -1, cleanupIndex)
	require.Equal(t, deliveryramp.StepPassed, result.Steps[cleanupIndex].Disposition)
	require.Equal(t, "observed", result.Steps[cleanupIndex].AssertionStatus)
	require.Equal(t, "off", device.state.ScreenState)
	require.Equal(t, "locked", device.state.LockState)
}

func TestTemporaryScenarioAppRecordsAndRetriesClosedWebViewExecutionOnce(t *testing.T) {
	device := &scenarioInstallFakeDevice{}
	bas := &transientThenPassingBAS{}
	result, err := (Driver{Devices: device, BAS: bas}).Execute(context.Background(), appInstallRequest(t, "temporary"))
	if err != nil {
		t.Fatalf("transient BAS context closure did not recover: %v; result=%#v", err, result)
	}
	if bas.calls != 3 { // one failed cold-start attempt, then cold and warm pass
		t.Fatalf("BAS calls = %d, want 3", bas.calls)
	}
	var retryRecorded bool
	for _, event := range result.Events {
		if event.Type == "bas-retry" && strings.Contains(event.Observed, "execution_id=bas-closed-context") {
			retryRecorded = true
		}
	}
	if !retryRecorded {
		t.Fatalf("retry attempt and BAS execution identity were not retained: %#v", result.Events)
	}
}

func TestDurableScenarioAppKeepsPackageInstalled(t *testing.T) {
	device := &scenarioInstallFakeDevice{}
	bas := &countingBAS{}
	result, err := (Driver{Devices: device, BAS: bas}).Execute(context.Background(), appInstallRequest(t, "durable"))
	if err != nil || result.Disposition != deliveryramp.DispositionPass {
		t.Fatalf("durable scenario app journey failed: result=%#v err=%v", result, err)
	}
	if bas.calls != 2 {
		t.Fatalf("durable install did not run cold and warm BAS assertions: %d", bas.calls)
	}
	for _, call := range device.calls {
		if call["step_id"] == "temporary-cleanup/uninstall" {
			t.Fatal("durable scenario app was unexpectedly uninstalled")
		}
	}
	foundInstalledReadback := false
	for _, call := range device.calls {
		if call["step_id"] == "verify-installed" && call["expected"] == "installed" {
			foundInstalledReadback = true
		}
	}
	if !foundInstalledReadback {
		t.Fatalf("durable app state was not read back: %#v", device.calls)
	}
	if device.state.ScreenState != "off" || device.state.LockState != "locked" {
		t.Fatalf("durable install test did not restore the original screen state: %#v", device.state)
	}
}

func TestInstalledScenarioAppVerificationPreservesPackageAndChecksSavedStateBeforeInput(t *testing.T) {
	device := &scenarioInstallFakeDevice{packageState: InstalledPackageState{Installed: true, Version: "1.2.3"}}
	bas := &countingBAS{}
	result, err := (Driver{Devices: device, BAS: bas}).Execute(context.Background(), appInstallRequest(t, "verify"))
	require.NoError(t, err)
	require.Equal(t, deliveryramp.DispositionPass, result.Disposition)
	require.Equal(t, 1, device.packageReads)
	require.Equal(t, 3, bas.calls, "saved-state, cold-start, and warm-resume BAS flows must all pass")
	require.Len(t, bas.preflightCalls, 3)
	require.Equal(t, UpdatePreservationFlowID, bas.requests[0].Arguments["flow_reference"])
	require.Equal(t, UpdatePreservationFlowPath, bas.requests[0].FlowPath)
	stepIndex := func(id string) int {
		for index, step := range result.Steps {
			if step.ID == id {
				return index
			}
		}
		return -1
	}
	savedStateIndex := stepIndex("saved-state")
	coldIndex := stepIndex("cold-bas")
	warmIndex := stepIndex("warm-bas")
	installedIndex := stepIndex("verify-installed")
	require.GreaterOrEqual(t, savedStateIndex, 0)
	require.Greater(t, coldIndex, savedStateIndex, "saved-state must be asserted before the mutating conformance flow")
	require.Greater(t, warmIndex, coldIndex)
	require.Greater(t, installedIndex, warmIndex)
	require.Equal(t, "observed", result.Steps[installedIndex].AssertionStatus, "package-state assertion must be finalized when recording is disabled")
	require.Equal(t, "device-cleanup/restore-screen-state", result.Steps[len(result.Steps)-1].ID)
	require.Equal(t, "off", device.state.ScreenState)
	require.Equal(t, "locked", device.state.LockState)
	for _, call := range device.calls {
		require.NotEqual(t, "install-artifact", call["artifact_ref"], "verification must not reinstall or overwrite the app")
		require.NotEqual(t, "uninstall", call["step_id"], "verification must not delete the app or its data")
	}
	require.Zero(t, device.startedRecording, "verification must not create another external recording handle")
}

func TestInstalledScenarioAppVerificationRefusesKnownVersionCodeMismatchBeforeBAS(t *testing.T) {
	device := &scenarioInstallFakeDevice{packageState: InstalledPackageState{Installed: true, Version: "1.2.3", VersionCode: 11}}
	bas := &countingBAS{}
	result, err := (Driver{Devices: device, BAS: bas}).Execute(context.Background(), appInstallRequest(t, "verify"))
	require.NoError(t, err)
	require.Equal(t, deliveryramp.DispositionFailed, result.Disposition)
	require.Contains(t, result.DegradedReason, "version code 11 does not match expected 12")
	require.Zero(t, bas.calls)
	require.Empty(t, device.calls, "a mismatched package must be refused before device mutation")
}

func TestScenarioAppUpdateRequiresOlderInstalledVersionAndPreservesStateBeforeBASInput(t *testing.T) {
	device := &scenarioInstallFakeDevice{packageState: InstalledPackageState{Installed: true, Version: "1.1.0", VersionCode: 11}}
	bas := &countingBAS{}
	request := appInstallRequest(t, "update")
	result, err := (Driver{Devices: device, BAS: bas}).Execute(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, deliveryramp.DispositionPass, result.Disposition)
	require.Equal(t, 1, device.packageReads)
	require.Equal(t, 3, bas.calls)
	require.Len(t, bas.preflightCalls, 2)
	require.Equal(t, UpdatePreservationFlowID, bas.preflightCalls[0].Arguments["flow_reference"])
	require.Equal(t, UpdatePreservationFlowPath, bas.preflightCalls[0].FlowPath)
	require.Equal(t, DefaultScenarioBASFlowPath, bas.preflightCalls[1].FlowPath)
	require.Equal(t, UpdatePreservationFlowID, bas.requests[0].Arguments["flow_reference"])
	require.Equal(t, UpdatePreservationFlowPath, bas.requests[0].FlowPath)
	require.Equal(t, request.RunID, bas.requests[0].RunID, "pre-update observer flow must receive a run-scoped BAS app context")
	require.Equal(t, "scenario-app-update-preservation", bas.requests[1].Arguments["flow_reference"])
	require.Equal(t, UpdatePreservationFlowPath, bas.requests[1].FlowPath)
	require.Empty(t, bas.requests[2].FlowPath)
	var preflight *deliveryramp.JourneyStep
	for index := range result.Steps {
		if result.Steps[index].ID == "install-preflight/package-state" {
			preflight = &result.Steps[index]
			break
		}
	}
	require.NotNil(t, preflight)
	require.Equal(t, "install-preflight/package-state", preflight.ID)
	require.Equal(t, "installed versionCode 11 is less than candidate versionCode 12", preflight.ExpectedState)
	require.Equal(t, "installed=true; versionName=1.1.0; versionCode=11", preflight.ObservedState)
	require.True(t, preflight.AssertionSurfaceObserved)
	var installSeen, uninstallSeen bool
	for _, call := range device.calls {
		if call["step_id"] == "artifact-install/install" {
			installSeen = call["artifact_ref"] == request.Artifact.ImmutableRef && call["package"] == "com.example.generated"
		}
		if call["step_id"] == "temporary-cleanup/uninstall" {
			uninstallSeen = true
		}
	}
	require.True(t, installSeen)
	require.False(t, uninstallSeen, "durable update must keep the updated package installed")
	stepIndex := func(id string) int {
		for index, step := range result.Steps {
			if step.ID == id {
				return index
			}
		}
		return -1
	}
	require.GreaterOrEqual(t, stepIndex("update-preflight/saved-state"), 0)
	require.Less(t, stepIndex("update-preflight/saved-state"), stepIndex("artifact-install/install"), "existing app state must be observed before replacement")
}

func TestScenarioAppUpdateRefusesReplacementWhenPreUpdateSavedStateAssertionFails(t *testing.T) {
	device := &scenarioInstallFakeDevice{packageState: InstalledPackageState{Installed: true, Version: "1.1.0", VersionCode: 11}}
	result, err := (Driver{Devices: device, BAS: failingBAS{err: errors.New("saved-state sentinel missing")}}).Execute(context.Background(), appInstallRequest(t, "update"))
	require.NoError(t, err)
	require.Equal(t, deliveryramp.DispositionFailed, result.Disposition)
	require.Contains(t, result.DegradedReason, "pre-update saved-state assertion failed")
	for _, call := range device.calls {
		require.NotEqual(t, "artifact-install/install", call["step_id"], "failed baseline proof must prevent replacement")
	}
	require.Equal(t, "off", device.state.ScreenState)
	require.Equal(t, "locked", device.state.LockState)
}

func TestScenarioAppUpdateReadsBackPriorPackageAfterInstallErrorWithoutRetry(t *testing.T) {
	device := &scenarioInstallFakeDevice{
		packageState: InstalledPackageState{Installed: true, Version: "1.1.0", VersionCode: 11},
		installErr:   errors.New("install command timed out"),
	}
	result, err := (Driver{Devices: device, BAS: &countingBAS{}}).Execute(context.Background(), appInstallRequest(t, "update"))
	require.NoError(t, err)
	require.Equal(t, deliveryramp.DispositionFailed, result.Disposition)
	require.Equal(t, 2, device.packageReads, "preflight and post-error package readback must both run")
	var recovery *deliveryramp.JourneyStep
	for index := range result.Steps {
		if result.Steps[index].ID == "update-recovery/installed-package-state" {
			recovery = &result.Steps[index]
		}
	}
	require.NotNil(t, recovery)
	require.Equal(t, deliveryramp.StepPassed, recovery.Disposition)
	require.Equal(t, "installed=true; versionName=1.1.0; versionCode=11", recovery.ObservedState)
	require.Contains(t, recovery.DegradedReason, "prior durable version remains installed")
	installCalls := 0
	for _, call := range device.calls {
		if call["step_id"] == "artifact-install/install" {
			installCalls++
		}
	}
	require.Equal(t, 1, installCalls, "an uncertain install must not be retried automatically")
	require.Equal(t, "off", device.state.ScreenState)
	require.Equal(t, "locked", device.state.LockState)
}

func TestScenarioAppUpdateIdentifiesCandidateInstalledDespiteInstallError(t *testing.T) {
	candidate := InstalledPackageState{Installed: true, Version: "1.2.3", VersionCode: 12}
	device := &scenarioInstallFakeDevice{
		packageState:     InstalledPackageState{Installed: true, Version: "1.1.0", VersionCode: 11},
		postInstallState: &candidate,
		installErr:       errors.New("install command timed out"),
	}
	bas := &countingBAS{}
	result, err := (Driver{Devices: device, BAS: bas}).Execute(context.Background(), appInstallRequest(t, "update"))
	require.NoError(t, err)
	require.Equal(t, deliveryramp.DispositionFailed, result.Disposition, "a candidate package readback does not replace post-update BAS verification")
	require.Equal(t, 2, device.packageReads)
	require.Equal(t, 1, bas.calls, "only the pre-update assertion may run after install reports an error")
	var recovery *deliveryramp.JourneyStep
	for index := range result.Steps {
		if result.Steps[index].ID == "update-recovery/installed-package-state" {
			recovery = &result.Steps[index]
		}
	}
	require.NotNil(t, recovery)
	require.Equal(t, "installed=true; versionName=1.2.3; versionCode=12", recovery.ObservedState)
	require.Contains(t, recovery.DegradedReason, "post-update BAS verification is still required")
	installCalls := 0
	for _, call := range device.calls {
		if call["step_id"] == "artifact-install/install" {
			installCalls++
		}
	}
	require.Equal(t, 1, installCalls, "the candidate-installed observation must not trigger a second installation")
}

func TestScenarioAppUpdateKeepsInstalledVersionUnknownAfterErrorUnverified(t *testing.T) {
	unknown := InstalledPackageState{Installed: true, Version: "1.2.3"}
	device := &scenarioInstallFakeDevice{
		packageState:     InstalledPackageState{Installed: true, Version: "1.1.0", VersionCode: 11},
		postInstallState: &unknown,
		installErr:       errors.New("install command timed out"),
	}
	result, err := (Driver{Devices: device, BAS: &countingBAS{}}).Execute(context.Background(), appInstallRequest(t, "update"))
	require.NoError(t, err)
	require.Equal(t, deliveryramp.DispositionFailed, result.Disposition)
	var recovery *deliveryramp.JourneyStep
	for index := range result.Steps {
		if result.Steps[index].ID == "update-recovery/installed-package-state" {
			recovery = &result.Steps[index]
		}
	}
	require.NotNil(t, recovery)
	require.Equal(t, deliveryramp.StepUnavailable, recovery.Disposition)
	require.Equal(t, "unknown", recovery.AssertionStatus)
	require.Contains(t, recovery.Error, "installed version code is unknown")
}

func TestScenarioAppUpdateRefusesBeforeDeviceLeaseWhenBASFlowIsUnavailable(t *testing.T) {
	device := &scenarioInstallFakeDevice{packageState: InstalledPackageState{Installed: true, Version: "1.1.0", VersionCode: 11}}
	bas := &countingBAS{preflightErr: errors.New("preservation flow is missing from the scenario registry")}
	result, err := (Driver{Devices: device, BAS: bas}).Execute(context.Background(), appInstallRequest(t, "update"))
	require.NoError(t, err)
	require.Equal(t, deliveryramp.DispositionFailed, result.Disposition)
	require.Contains(t, result.DegradedReason, "preservation flow is missing")
	require.Len(t, bas.preflightCalls, 1)
	require.Zero(t, bas.calls)
	require.Zero(t, device.packageReads)
	require.Empty(t, device.calls, "missing preservation proof must fail before any device operation")
	require.Equal(t, "install-preflight/bas-flow/update-preservation", result.Steps[0].ID)
}

func TestScenarioAppUpdateRefusesAbsentUnknownOrNonIncreasingPackageBeforeInstall(t *testing.T) {
	cases := []struct {
		name       string
		installed  InstalledPackageState
		wantReason string
	}{
		{name: "absent", installed: InstalledPackageState{}, wantReason: "package is not installed"},
		{name: "unknown version code", installed: InstalledPackageState{Installed: true, Version: "1.1.0"}, wantReason: "installed version code is unknown"},
		{name: "same version code", installed: InstalledPackageState{Installed: true, Version: "1.2.3", VersionCode: 12}, wantReason: "must be greater than installed version code 12"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			device := &scenarioInstallFakeDevice{packageState: tc.installed}
			bas := &countingBAS{}
			result, err := (Driver{Devices: device, BAS: bas}).Execute(context.Background(), appInstallRequest(t, "update"))
			require.NoError(t, err)
			require.Equal(t, deliveryramp.DispositionFailed, result.Disposition)
			require.Contains(t, result.DegradedReason, tc.wantReason)
			require.Empty(t, device.calls, "the install and app actions must not run after failed version preflight")
			require.Zero(t, bas.calls)
		})
	}
}

func TestScenarioAppRefusesScreenOffUnlockedStateBeforeInstall(t *testing.T) {
	device := &scenarioInstallFakeDevice{state: DeviceStateSnapshot{ForegroundPackage: "com.android.launcher", ScreenState: "off", LockState: "unlocked"}}
	result, err := (Driver{Devices: device, BAS: &countingBAS{}}).Execute(context.Background(), appInstallRequest(t, "temporary"))
	if err != nil {
		t.Fatal(err)
	}
	if result.Disposition != deliveryramp.DispositionFailed {
		t.Fatalf("unrestorable pre-test state was not refused: %#v", result)
	}
	for _, call := range device.calls {
		if call["step_id"] == "artifact-install/install" {
			t.Fatalf("install occurred despite unrestorable pre-state: %#v", device.calls)
		}
	}
}

func TestScreenRestoreWaitsForDelayedDeviceTransition(t *testing.T) {
	device := &scenarioInstallFakeDevice{
		state:      DeviceStateSnapshot{ForegroundPackage: "com.example.generated", ScreenState: "on", LockState: "unlocked"},
		sleepReads: 3,
	}
	initial := DeviceStateSnapshot{ForegroundPackage: "com.android.launcher", ScreenState: "off", LockState: "locked"}
	err := restoreAndroidScreenState(context.Background(), device, Lease{DeviceID: "android-phone"}, initial, "profile-1")
	if err != nil {
		t.Fatalf("delayed screen sleep was not verified: %v", err)
	}
	if device.state.ScreenState != "off" || device.state.LockState != "locked" {
		t.Fatalf("screen restore ended in the wrong state: %#v", device.state)
	}
}

func TestDriverCorrelatesCompletedBASFlowAsObservation(t *testing.T) {
	device := &evidenceFakeDevice{}
	request := testRequest()
	request.Plan.Steps[0].Assertion = &deliveryramp.AssertionSpec{ID: "bas/expected", Expected: "WebView usable"}
	result, err := (Driver{Devices: device, BAS: fakeBAS{}}).Execute(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if result.Disposition != deliveryramp.DispositionPass || result.Steps[0].AssertionStatus != "observed" {
		t.Fatalf("completed BAS flow did not satisfy its assertion: %#v", result)
	}
	if len(result.Events) == 0 || result.Events[0].Type != "bas-observation" || result.Events[0].Source != "browser-automation-studio" {
		t.Fatalf("BAS observation event was not retained: %#v", result.Events)
	}
}

func TestDriverRetainsNotApplicableChapterWithoutDeviceMutation(t *testing.T) {
	device := &fakeDevice{}
	request := testRequest()
	request.Plan.Steps = []deliveryramp.JourneyStepSpec{{
		ID: "permission_deny_grace/not-applicable", Action: "not-applicable",
		Arguments: map[string]string{"chapter_id": "permission_deny_grace", "reason": "requires Android 13 or newer; target reports Android 10"},
	}}
	result, err := (Driver{Devices: device}).Execute(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if result.Disposition != deliveryramp.DispositionPass || len(result.Steps) != 1 || result.Steps[0].Disposition != deliveryramp.StepNotRun {
		t.Fatalf("not-applicable chapter was not retained as an explicit skip: %#v", result)
	}
	if result.Steps[0].ObservedState != "requires Android 13 or newer; target reports Android 10" {
		t.Fatalf("skip omitted applicability evidence: %#v", result.Steps[0])
	}
	if len(device.actions) != 0 {
		t.Fatalf("skipped chapter actuated the device: %#v", device.actions)
	}
	if len(result.Events) != 1 || result.Events[0].Type != "chapter-not-applicable" {
		t.Fatalf("skip event was not retained: %#v", result.Events)
	}
}

func TestCorrelateLogcatEventsKeepsAdjacentBASObservationsOnTheirOwnSteps(t *testing.T) {
	started := time.Now().UTC()
	steps := []deliveryramp.JourneyStep{
		{ID: "offline", Action: "bas-flow", Disposition: deliveryramp.StepPassed, StartedAt: started, CompletedAt: started.Add(time.Second), AssertionStatus: "pending"},
		{ID: "online", Action: "bas-flow", Disposition: deliveryramp.StepPassed, StartedAt: started.Add(1100 * time.Millisecond), CompletedAt: started.Add(2 * time.Second), AssertionStatus: "pending"},
	}
	events := []deliveryramp.JourneyEvent{
		{Type: "bas-observation", StepID: "offline", StartedAt: started.Add(900 * time.Millisecond)},
		{Type: "bas-observation", StepID: "online", StartedAt: started.Add(1500 * time.Millisecond)},
	}
	correlated := correlateLogcatEvents(steps, events)
	for _, step := range correlated {
		if step.AssertionStatus != "observed" {
			t.Fatalf("BAS observation was assigned to the wrong adjacent step: %#v", correlated)
		}
	}
}

func TestDriverForwardsPackageStateExpectation(t *testing.T) {
	device := &fakeDevice{}
	request := testRequest()
	request.Artifact.Metadata = map[string]string{"package_name": "com.example.generated", "apk_path": "/tmp/app.apk"}
	request.Plan.Steps = []deliveryramp.JourneyStepSpec{{ID: "assert-absent", Action: "package-state", Arguments: map[string]string{"target": "absent"}}}
	if _, err := (Driver{Devices: device}).Execute(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if len(device.actions) != 1 || device.actions[0]["expected"] != "absent" {
		t.Fatalf("package-state expectation was not forwarded: %#v", device.actions)
	}
}

func TestDriverUsesConfiguredAuthProfileForUnlock(t *testing.T) {
	device := &fakeDevice{}
	request := testRequest()
	request.Artifact.Metadata = map[string]string{"auth_profile_id": "profile-1"}
	request.Plan.Steps = []deliveryramp.JourneyStepSpec{{ID: "unlock", Action: "screen", Arguments: map[string]string{"target": "unlock", "chapter_id": "cold-start"}}}
	result, err := (Driver{Devices: device}).Execute(context.Background(), request)
	if err != nil || result.Disposition != deliveryramp.DispositionPass {
		t.Fatalf("profile-backed unlock failed: result=%#v err=%v", result, err)
	}
	if len(device.unlocked) != 1 || device.unlocked[0] != "profile-1" || len(device.actions) != 0 {
		t.Fatalf("unlock profile was not routed through the authenticator: unlocked=%v actions=%#v", device.unlocked, device.actions)
	}
}

func TestDriverRetainsNativeDeviceEvidence(t *testing.T) {
	device := &fakeDevice{evidence: []deliveryramp.EvidenceReference{{ID: "frame-1", Kind: "screenshot", Checksum: "sha256:frame", Redacted: true}}}
	request := testRequest()
	request.Artifact.Metadata = map[string]string{"package_name": "com.example.generated", "apk_path": "/tmp/app.apk"}
	request.Plan.Steps = []deliveryramp.JourneyStepSpec{{ID: "observe", Action: "observe", Arguments: map[string]string{"target": ""}}}
	result, err := (Driver{Devices: device}).Execute(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Steps) != 1 || len(result.Steps[0].Evidence) != 2 || result.Steps[0].Evidence[0].ID != "frame-1" {
		t.Fatalf("native evidence was not retained: %#v", result.Steps)
	}
}

func TestDriverUsesOneRecordingPerChapterWhenClientSupportsIt(t *testing.T) {
	device := &chapterFakeDevice{}
	request := testRequest()
	request.Plan.Steps = []deliveryramp.JourneyStepSpec{
		{ID: "launch", Action: "launch", Arguments: map[string]string{"chapter_id": "cold-start"}},
		{ID: "observe", Action: "observe", Arguments: map[string]string{"chapter_id": "cold-start"}},
		{ID: "resume", Action: "launch", Arguments: map[string]string{"chapter_id": "resume"}},
	}
	request.Artifact.Metadata = map[string]string{"package_name": "com.example.generated", "apk_path": "/tmp/app.apk"}
	result, err := (Driver{Devices: device}).Execute(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if len(device.started) != 2 || len(device.stopped) != 2 || result.Steps[0].ChapterID != "cold-start" || result.Steps[2].ChapterID != "resume" {
		t.Fatalf("chapter recording boundaries were not preserved: started=%v stopped=%v steps=%#v", device.started, device.stopped, result.Steps)
	}
	if result.ReviewRecording == nil || result.ReviewRecording.ID != "review-1" {
		t.Fatalf("review recording was not finalized: %#v", result.ReviewRecording)
	}
}

func TestDriverRecordsAssertionMetadataAndClockSamples(t *testing.T) {
	device := &evidenceFakeDevice{}
	request := testRequest()
	request.Plan.Steps = []deliveryramp.JourneyStepSpec{{
		ID: "install", Action: "install", Arguments: map[string]string{"chapter_id": "cold-start"},
		Assertion: &deliveryramp.AssertionSpec{ID: "cold-start/expected", Expected: "package installed"},
	}}
	result, err := (Driver{Devices: device}).Execute(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if result.Disposition != deliveryramp.DispositionPass || result.ClockOffsetStart == nil || result.ClockOffsetEnd == nil {
		t.Fatalf("clock samples were not retained: %#v", result)
	}
	step := result.Steps[0]
	if step.AssertionID != "cold-start/expected" || step.ExpectedState != "package installed" || step.AssertionStatus != "observed" {
		t.Fatalf("assertion metadata was not correlated: %#v", step)
	}
	if len(result.Events) != 1 || result.Events[0].Source != "device-control" || result.Events[0].StepID != step.ID {
		t.Fatalf("device action observation was not retained: %#v", result.Events)
	}
}

func TestDriverFailsLaunchAssertionWithoutLogcatEvent(t *testing.T) {
	device := &evidenceFakeDevice{}
	request := testRequest()
	request.Plan.Steps = []deliveryramp.JourneyStepSpec{{
		ID: "launch", Action: "launch", Arguments: map[string]string{"chapter_id": "cold-start"},
		Assertion: &deliveryramp.AssertionSpec{ID: "cold-start/expected", Expected: "activity displayed"},
	}}
	result, err := (Driver{Devices: device}).Execute(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if result.Disposition != deliveryramp.DispositionFailed || result.Steps[0].AssertionStatus != "failed" {
		t.Fatalf("missing launch logcat event did not fail closed: %#v", result)
	}
}

func TestDriverPreservesFailedActionAndDoesNotPromoteUnavailable(t *testing.T) {
	device := &fakeDevice{actionErr: errors.New("visible surface unavailable")}
	request := testRequest()
	request.Plan.Steps = []deliveryramp.JourneyStepSpec{{ID: "observe", Action: "observe", Arguments: map[string]string{"chapter_id": "cold-start"}}}
	result, err := (Driver{Devices: device}).Execute(context.Background(), request)
	if err != nil || result.Disposition != deliveryramp.DispositionFailed || result.Steps[0].Disposition != deliveryramp.StepFailed {
		t.Fatalf("failed action was not retained fail-closed: result=%#v err=%v", result, err)
	}

	device = &fakeDevice{}
	request = testRequest()
	request.Cell.Target.Capabilities = []string{"android-webview"}
	request.Plan.Steps = []deliveryramp.JourneyStepSpec{{ID: "offline", Action: "network", Arguments: map[string]string{"chapter_id": "offline", "required_capabilities": "network-control"}}}
	result, err = (Driver{Devices: device}).Execute(context.Background(), request)
	if err != nil || result.Disposition != deliveryramp.DispositionUnavailable {
		t.Fatalf("unavailable chapter was promoted to pass: result=%#v err=%v", result, err)
	}
}

func TestDriverForwardsBoundedDeviceTimeout(t *testing.T) {
	device := &fakeDevice{}
	request := testRequest()
	request.Plan.Steps = []deliveryramp.JourneyStepSpec{{
		ID: "install", Action: "install", Arguments: map[string]string{
			"chapter_id": "cold-start", "target": "", "reference": "install", "timeout_ms": "120000",
		},
	}}
	if _, err := (Driver{Devices: device}).Execute(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if len(device.actions) != 1 || device.actions[0]["timeout_ms"] != "120000" {
		t.Fatalf("bounded device timeout was dropped: %#v", device.actions)
	}
}

func TestDriverMapsNativeActionTargetsToDeviceArguments(t *testing.T) {
	device := &fakeDevice{}
	request := testRequest()
	request.Plan.Steps = []deliveryramp.JourneyStepSpec{{
		ID: "deny", Action: "revoke-permission", Arguments: map[string]string{
			"chapter_id": "permissions", "target": "android.permission.POST_NOTIFICATIONS", "reference": "revoke-permission",
		},
	}}
	request.Artifact.Metadata = map[string]string{"package_name": "com.vrooli.hellomobile", "apk_path": "/tmp/app.apk"}
	if _, err := (Driver{Devices: device}).Execute(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if len(device.actions) != 1 || device.actions[0]["permission"] != "android.permission.POST_NOTIFICATIONS" || device.actions[0]["value"] != "android.permission.POST_NOTIFICATIONS" || device.actions[0]["package"] != "com.vrooli.hellomobile" {
		t.Fatalf("native action target was not mapped explicitly: %#v", device.actions)
	}
}

func TestTransientWebViewClosureIsRetryableButOtherBASFailuresAreNot(t *testing.T) {
	if !isTransientWebViewClosure(errors.New("step 0 failed: Target page, context or browser has been closed")) {
		t.Fatal("known renderer closure was not classified as retryable")
	}
	if isTransientWebViewClosure(errors.New("selector did not become visible")) {
		t.Fatal("ordinary BAS failure was incorrectly classified as retryable")
	}
}

func TestHTTPDeviceClientSamplesDeviceClockThroughRedactedEvidence(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/flows/run":
			_ = json.NewEncoder(w).Encode(map[string]any{"disposition": "passed", "evidence": []map[string]any{{"id": "clock-ref", "kind": "log", "checksum": "abc", "redaction_verified": true}}})
		case "/api/v1/evidence/clock-ref":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("1755259200.125000000\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := &HTTPDeviceClient{BaseURL: server.URL, Client: server.Client()}
	sample, err := client.SampleClock(context.Background(), Lease{ID: "lease-1", DeviceID: "device-1", Token: "token-1"})
	if err != nil {
		t.Fatal(err)
	}
	if !sample.DeviceTime.Equal(time.Unix(1755259200, 125000000).UTC()) || sample.Evidence.ID != "clock-ref" || sample.UncertaintyMs < 0 {
		t.Fatalf("unexpected clock sample: %#v", sample)
	}
}

func TestHTTPDeviceClientUnlocksThroughDeviceControlAuthEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/unlock" {
			http.NotFound(w, r)
			return
		}
		var request map[string]string
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request["profile_id"] != "profile-1" || request["device_id"] != "device-1" || request["lease_token"] != "token-1" {
			http.Error(w, "bad unlock request", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"outcome": "unlocked"})
	}))
	defer server.Close()

	client := &HTTPDeviceClient{BaseURL: server.URL, Actor: "scenario-to-android", Client: server.Client()}
	if err := client.Unlock(context.Background(), Lease{DeviceID: "device-1", Token: "token-1"}, "profile-1"); err != nil {
		t.Fatal(err)
	}
}

func TestHTTPDeviceClientReadsPreTestStateFromDeviceControl(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/devices/android-phone/state" {
			t.Fatalf("unexpected state request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"foreground_package":"com.android.launcher","screen_state":"off","lock_state":"locked"}`))
	}))
	defer server.Close()

	client := &HTTPDeviceClient{BaseURL: server.URL, Client: server.Client()}
	state, err := client.ReadState(context.Background(), "android-phone")
	if err != nil {
		t.Fatal(err)
	}
	if state.ForegroundPackage != "com.android.launcher" || state.ScreenState != "off" || state.LockState != "locked" {
		t.Fatalf("unexpected pre-test state: %#v", state)
	}
}

func TestHTTPDeviceClientPreservesFailedChapterReason(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/flows/run" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"disposition": "failed",
			"chapters": []map[string]string{{
				"disposition": "failed",
				"message":     `visible surface unavailable: lock state is "locked" and screen state is "on"`,
			}},
		})
	}))
	defer server.Close()

	client := &HTTPDeviceClient{BaseURL: server.URL, Client: server.Client()}
	_, err := client.Execute(context.Background(), Lease{ID: "lease-1", DeviceID: "device-1", Token: "token-1"}, "observe", map[string]string{"step_id": "observe"})
	if err == nil || !strings.Contains(err.Error(), "visible surface unavailable: lock state is") {
		t.Fatalf("failed chapter reason was discarded: %v", err)
	}
}

func TestHTTPDeviceClientUsesUSBForNetworkControlWhenSelectedRouteIsWireless(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/flows/run" {
			http.NotFound(w, r)
			return
		}
		var request struct {
			Flow struct {
				Transport string `json:"transport"`
			} `json:"flow"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode flow request: %v", err)
		}
		if request.Flow.Transport != "usb" {
			t.Fatalf("network-control flow used transport %q, want usb", request.Flow.Transport)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"disposition":"passed","evidence":[]}`))
	}))
	defer server.Close()

	client := &HTTPDeviceClient{BaseURL: server.URL, DeviceTransport: "wireless", Client: server.Client()}
	_, err := client.Execute(context.Background(), Lease{ID: "lease-1", DeviceID: "device-1", Token: "token-1"}, "network", map[string]string{
		"step_id": "offline",
		"target":  "offline",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestHTTPDeviceClientUsesUSBForOfflineTransitionWebViewAndRecording(t *testing.T) {
	var recordingTransport, webViewTransport string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/recordings/start":
			var request map[string]string
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatalf("decode recording request: %v", err)
			}
			recordingTransport = request["transport"]
			_, _ = w.Write([]byte(`{"handle":{"id":"recording-1"}}`))
		case "/api/v1/devices/device-1/webview/attach":
			var request map[string]string
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatalf("decode WebView request: %v", err)
			}
			webViewTransport = request["transport"]
			_, _ = w.Write([]byte(`{"endpoint":{"cdp_endpoint":"http://127.0.0.1:9222","renderer_id":"renderer-1","renderer_url":"http://localhost"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := &HTTPDeviceClient{BaseURL: server.URL, DeviceTransport: "wireless", Client: server.Client()}
	lease := Lease{ID: "lease-1", DeviceID: "device-1", Token: "token-1"}
	if err := client.StartChapterRecording(context.Background(), lease, "offline_transition"); err != nil {
		t.Fatal(err)
	}
	if _, err := client.AttachWebViewWithTransport(context.Background(), lease, "com.example.hello", "usb"); err != nil {
		t.Fatal(err)
	}
	if recordingTransport != "usb" {
		t.Fatalf("offline-transition recording used transport %q, want usb", recordingTransport)
	}
	if webViewTransport != "usb" {
		t.Fatalf("offline-transition WebView attach used transport %q, want usb", webViewTransport)
	}
}

func TestHTTPDeviceClientSendsDeviceTimeoutOutsideActionArguments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/flows/run" {
			http.NotFound(w, r)
			return
		}
		var request struct {
			Flow struct {
				Steps []struct {
					TimeoutMS int64          `json:"timeout_ms"`
					Arguments map[string]any `json:"arguments"`
				} `json:"steps"`
			} `json:"flow"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode flow request: %v", err)
		}
		if len(request.Flow.Steps) != 1 || request.Flow.Steps[0].TimeoutMS != 120000 {
			t.Fatalf("timeout was not promoted to the flow step: %#v", request)
		}
		if _, present := request.Flow.Steps[0].Arguments["timeout_ms"]; present {
			t.Fatal("timeout leaked into device action arguments")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"disposition":"passed","evidence":[]}`))
	}))
	defer server.Close()

	client := &HTTPDeviceClient{BaseURL: server.URL, Client: server.Client()}
	_, err := client.Execute(context.Background(), Lease{ID: "lease-1", DeviceID: "device-1", Token: "token-1"}, "install", map[string]string{
		"step_id":    "install",
		"target":     "",
		"value":      "/tmp/app.apk",
		"timeout_ms": "120000",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestHTTPDeviceClientConfirmsOnlyImmutableArtifactInstall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Confirmed bool `json:"confirmed"`
			Flow      struct {
				Steps []struct {
					Kind      string         `json:"kind"`
					Arguments map[string]any `json:"arguments"`
				} `json:"steps"`
			} `json:"flow"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode flow request: %v", err)
		}
		if !request.Confirmed || len(request.Flow.Steps) != 1 || request.Flow.Steps[0].Kind != "install-artifact" {
			t.Fatalf("verified install was not explicitly confirmed: %#v", request)
		}
		if request.Flow.Steps[0].Arguments["artifact_ref"] != "android-debug:immutable" || request.Flow.Steps[0].Arguments["version_name"] != "1.2.3" {
			t.Fatalf("immutable identity was not sent to Device Control: %#v", request.Flow.Steps[0].Arguments)
		}
		if len(request.Flow.Steps[0].Arguments) != 3 {
			t.Fatalf("journey metadata was sent as install-artifact arguments: %#v", request.Flow.Steps[0].Arguments)
		}
		for _, key := range []string{"chapter_id", "required_capabilities", "reference"} {
			if _, present := request.Flow.Steps[0].Arguments[key]; present {
				t.Fatalf("journey metadata %q leaked into Device Control action arguments: %#v", key, request.Flow.Steps[0].Arguments)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"disposition":"passed","evidence":[]}`))
	}))
	defer server.Close()

	client := &HTTPDeviceClient{BaseURL: server.URL, Client: server.Client()}
	_, err := client.Execute(context.Background(), Lease{ID: "lease-1", DeviceID: "device-1", Token: "token-1"}, "install-artifact", map[string]string{
		"step_id": "install_cold_start/install", "chapter_id": "install_cold_start", "required_capabilities": "app-lifecycle",
		"target": "", "reference": "install", "timeout_ms": "120000",
		"artifact_ref": "android-debug:immutable", "package": "com.example.generated", "version_name": "1.2.3",
	})
	if err != nil {
		t.Fatalf("confirmed verified install failed: %v", err)
	}
}

func TestHTTPDeviceClientReadsInstalledPackageVersionThroughLeaseOwnedLifecycle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/devices/device-1/apps", r.URL.Path)
		var request map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
		require.Equal(t, "operator", request["actor"])
		require.Equal(t, "lease-token", request["lease_token"])
		require.Equal(t, false, request["confirmed"], "package version readback must remain read-only")
		require.Equal(t, "package-state", request["operation"])
		require.Equal(t, "com.example.generated", request["package"])
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":{"operation":"package-state","package":"com.example.generated","status":"available","installed":true,"version":"1.2.0","version_code":11}}`))
	}))
	defer server.Close()

	client := &HTTPDeviceClient{BaseURL: server.URL, Actor: "operator", Client: server.Client()}
	state, err := client.ReadPackageState(context.Background(), Lease{DeviceID: "device-1", Token: "lease-token"}, "com.example.generated")
	require.NoError(t, err)
	require.Equal(t, InstalledPackageState{Installed: true, Version: "1.2.0", VersionCode: 11}, state)
}
