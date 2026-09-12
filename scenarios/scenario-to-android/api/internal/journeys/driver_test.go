package journeys

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

type fakeBAS struct{}

func (fakeBAS) Execute(context.Context, BASRequest) (BASResult, error) {
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

func TestDriverCompletesBASStepWithReferenceOnlyVideo(t *testing.T) {
	result, err := (Driver{Devices: &fakeDevice{}, BAS: fakeBAS{}}).Execute(context.Background(), testRequest())
	if err != nil {
		t.Fatal(err)
	}
	if result.Disposition != deliveryramp.DispositionPass || result.Steps[0].VideoStartOffsetMs == nil || result.Steps[0].VideoEndOffsetMs == nil {
		t.Fatalf("unexpected result: %#v", result)
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
