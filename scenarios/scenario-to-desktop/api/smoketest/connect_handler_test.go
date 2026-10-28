package smoketest_test

import (
	"context"
	"testing"
	"time"

	"scenario-to-desktop-api/smoketest"
	"scenario-to-desktop-api/smoketest/mocks"

	"connectrpc.com/connect"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/shared"
)

type connectServiceSpy struct {
	platform string
	calls    chan smokeTestCall
}

type smokeTestCall struct {
	id, scenario, artifact, platform    string
	deploymentMode, journeyID, proxyURL string
	ctx                                 context.Context
}

func (s *connectServiceSpy) CurrentPlatform() string { return s.platform }

func (s *connectServiceSpy) PerformSmokeTest(ctx context.Context, id, scenario, artifact, platform string) {
	s.calls <- smokeTestCall{id: id, scenario: scenario, artifact: artifact, platform: platform, ctx: ctx}
}

func (s *connectServiceSpy) PerformSmokeTestRequest(ctx context.Context, request smoketest.SmokeTestRequest) {
	s.calls <- smokeTestCall{id: request.SmokeTestID, scenario: request.ScenarioName, artifact: request.ArtifactPath, platform: request.Platform, ctx: ctx, deploymentMode: request.DeploymentMode, journeyID: request.JourneyID, proxyURL: request.ProxyURL}
}

func TestConnectServiceStartGetAndCancelSmokeTest(t *testing.T) {
	store := mocks.NewMockStore()
	cancels := mocks.NewMockCancelManager()
	service := &connectServiceSpy{platform: "linux", calls: make(chan smokeTestCall, 1)}
	handler := smoketest.NewConnectService(service, store, cancels)
	start := startSmokeTest(t, handler, store)
	assertSmokeCall(t, service.calls, start.Msg.GetSmokeTestId())
	assertStoredSmokeTest(t, handler, start.Msg.GetSmokeTestId())
	assertCancelledSmokeTest(t, handler, start.Msg.GetSmokeTestId())
}

func startSmokeTest(t *testing.T, handler *smoketest.ConnectService, store *mocks.MockStore) *connect.Response[domainv1.SmokeTestStartResponse] {
	t.Helper()
	platform := sharedv1.Platform_PLATFORM_LINUX
	mode, journey := "bundled", "hello-desktop"
	start, err := handler.StartSmokeTest(context.Background(), connect.NewRequest(&domainv1.SmokeTestStartRequest{ScenarioName: "hello-desktop", ArtifactPath: "/tmp/hello-desktop.AppImage", Platform: &platform, DeploymentMode: &mode, JourneyId: &journey}))
	if err != nil {
		t.Fatalf("StartSmokeTest() error = %v", err)
	}
	if start.Msg.GetSmokeTestId() == "" || start.Msg.GetStatus() != sharedv1.SmokeTestStatus_SMOKE_TEST_STATUS_RUNNING {
		t.Fatalf("StartSmokeTest() = %+v, want queued running test with id", start.Msg)
	}
	if got := store.Statuses[start.Msg.GetSmokeTestId()]; got == nil || got.Status != "running" || got.DeploymentMode != "bundled" {
		t.Fatalf("stored status = %+v, want running status", got)
	}
	return start
}

func assertSmokeCall(t *testing.T, calls <-chan smokeTestCall, id string) {
	t.Helper()
	select {
	case call := <-calls:
		if call.id != id || call.scenario != "hello-desktop" || call.artifact != "/tmp/hello-desktop.AppImage" || call.platform != "linux" || call.deploymentMode != "bundled" || call.journeyID != "hello-desktop" {
			t.Fatalf("PerformSmokeTest() = %+v, want request values", call)
		}
	case <-time.After(time.Second):
		t.Fatal("PerformSmokeTest() was not started")
	}
}

func assertStoredSmokeTest(t *testing.T, handler *smoketest.ConnectService, id string) {
	t.Helper()
	got, err := handler.GetSmokeTest(context.Background(), connect.NewRequest(&domainv1.SmokeTestStatusRequest{SmokeTestId: id}))
	if err != nil {
		t.Fatalf("GetSmokeTest() error = %v", err)
	}
	if got.Msg.GetArtifactPath() != "/tmp/hello-desktop.AppImage" || got.Msg.GetPlatform() != sharedv1.Platform_PLATFORM_LINUX {
		t.Fatalf("GetSmokeTest() = %+v, want stored smoke-test details", got.Msg)
	}
}

func assertCancelledSmokeTest(t *testing.T, handler *smoketest.ConnectService, id string) {
	t.Helper()
	cancelled, err := handler.CancelSmokeTest(context.Background(), connect.NewRequest(&domainv1.SmokeTestCancelRequest{SmokeTestId: id}))
	if err != nil {
		t.Fatalf("CancelSmokeTest() error = %v", err)
	}
	if cancelled.Msg.GetStatus() != "cancelling" {
		t.Fatalf("CancelSmokeTest() status = %q, want cancelling", cancelled.Msg.GetStatus())
	}
}

func TestConnectServiceRejectsIncompleteSmokeTestRequest(t *testing.T) {
	handler := smoketest.NewConnectService(&connectServiceSpy{platform: "linux", calls: make(chan smokeTestCall)}, mocks.NewMockStore(), mocks.NewMockCancelManager())
	_, err := handler.StartSmokeTest(context.Background(), connect.NewRequest(&domainv1.SmokeTestStartRequest{ScenarioName: "hello-desktop"}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("StartSmokeTest() error code = %v, want invalid_argument (err %v)", connect.CodeOf(err), err)
	}
}

func TestConnectServiceReturnsNotFoundForUnknownSmokeTest(t *testing.T) {
	handler := smoketest.NewConnectService(&connectServiceSpy{platform: "linux", calls: make(chan smokeTestCall)}, mocks.NewMockStore(), mocks.NewMockCancelManager())
	_, err := handler.GetSmokeTest(context.Background(), connect.NewRequest(&domainv1.SmokeTestStatusRequest{SmokeTestId: "missing"}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("GetSmokeTest() error code = %v, want not_found (err %v)", connect.CodeOf(err), err)
	}
}

func TestStatusToProtoPreservesScreenRecordingEvidenceWithoutVideoPath(t *testing.T) {
	status := &smoketest.Status{
		SmokeTestID:  "smoke-evidence",
		ScenarioName: "hello-desktop",
		StartedAt:    time.Now(),
		ScreenRecording: &smoketest.RecordingStatus{
			Recorded:  true,
			CaptureID: "capture-smoke-evidence",
		},
	}

	got := smoketest.StatusToProto(status).GetScreenRecording()
	if got == nil {
		t.Fatal("StatusToProto() screen recording = nil, want evidence summary")
	}
	if !got.GetRecorded() || got.GetCaptureId() != "capture-smoke-evidence" {
		t.Fatalf("StatusToProto() screen recording = %+v, want recorded capture reference", got)
	}
}

func TestStatusToProtoProjectsVersionedEvidenceReview(t *testing.T) {
	status := &smoketest.Status{
		SmokeTestID:  "smoke-review",
		ScenarioName: "hello-desktop",
		StartedAt:    time.Now(),
		EvidenceReview: &smoketest.JourneyReview{
			SchemaVersion: "journey-evidence.v2",
			Capability:    "hello-desktop",
			PlanID:        "hello-desktop.baseline.v2",
			Profile:       "normal-review",
			Disposition:   "degraded",
			Reason:        "recording unavailable",
			EventCount:    2,
			Chapters:      []smoketest.JourneyChapter{{ID: "greet", Purpose: "Prove greeting", Action: "semantic_greet", Disposition: "passed", Expected: "Hello", Observed: "Hello", EvidenceIDs: []string{"capture-1"}}},
		},
	}

	review := smoketest.StatusToProto(status).GetEvidenceReview()
	if review == nil || review.GetCapability() != "hello-desktop" || review.GetDisposition() != "degraded" || len(review.GetChapters()) != 1 {
		t.Fatalf("evidence review = %+v", review)
	}
	if review.GetChapters()[0].GetPurpose() != "Prove greeting" || review.GetChapters()[0].GetEvidenceIds()[0] != "capture-1" {
		t.Fatalf("chapter projection = %+v", review.GetChapters()[0])
	}
}

func TestConnectServiceRejectsInvalidDeploymentContext(t *testing.T) {
	for _, mode := range []string{"unknown", "proxy"} {
		t.Run(mode, func(t *testing.T) {
			handler := smoketest.NewConnectService(&connectServiceSpy{platform: "linux", calls: make(chan smokeTestCall, 1)}, mocks.NewMockStore(), mocks.NewMockCancelManager())
			_, err := handler.StartSmokeTest(context.Background(), connect.NewRequest(&domainv1.SmokeTestStartRequest{ScenarioName: "hello-desktop", ArtifactPath: "/tmp/hello.AppImage", DeploymentMode: &mode}))
			if connect.CodeOf(err) != connect.CodeInvalidArgument {
				t.Fatalf("invalid deployment context error = %v", err)
			}
		})
	}
}

func TestConnectServicePreservesProxyTargetBeforeExecution(t *testing.T) {
	store := mocks.NewMockStore()
	service := &connectServiceSpy{platform: "linux", calls: make(chan smokeTestCall, 1)}
	handler := smoketest.NewConnectService(service, store, mocks.NewMockCancelManager())
	mode, target := "proxy", "http://127.0.0.1:12345"
	started, err := handler.StartSmokeTest(context.Background(), connect.NewRequest(&domainv1.SmokeTestStartRequest{ScenarioName: "hello-desktop", ArtifactPath: "/tmp/hello.AppImage", DeploymentMode: &mode, ProxyUrl: &target}))
	if err != nil {
		t.Fatal(err)
	}
	status, _ := store.Get(started.Msg.GetSmokeTestId())
	if status.DeploymentMode != mode || status.ProxyURL != target {
		t.Fatalf("queued target = %q %q", status.DeploymentMode, status.ProxyURL)
	}
	select {
	case call := <-service.calls:
		if call.deploymentMode != mode || call.proxyURL != target {
			t.Fatalf("execution lost proxy target: %+v", call)
		}
	case <-time.After(time.Second):
		t.Fatal("smoke execution was not started")
	}
}

func TestStatusToProtoPreservesTargetAndCleanupEvidence(t *testing.T) {
	status := &smoketest.Status{SmokeTestID: "closed-run", ScenarioName: "hello-desktop", Status: "failed", StartedAt: time.Now(), DeploymentMode: "bundled", ScreenContentSource: "bundled_private", ProcessLifetimeChecks: []smoketest.ProcessLifetimeCheck{
		{Stage: "protocol", SessionID: "protocol-session", Passed: true},
		{Stage: "demo", SessionID: "demo-session", Remaining: []int{42}, Assertion: "descendant remains"},
	}}
	got := smoketest.StatusToProto(status)
	if got.GetDeploymentMode() != "bundled" || got.GetScreenContentSource() != "bundled_private" || got.GetStatus() != sharedv1.SmokeTestStatus_SMOKE_TEST_STATUS_FAILED {
		t.Fatalf("target/application outcome lost: %v", got)
	}
	checks := got.GetProcessLifetimeChecks()
	if len(checks) != 2 {
		t.Fatalf("cleanup checks = %v", checks)
	}
	if checks[0].GetStage() != "protocol" || checks[0].GetSessionId() != "protocol-session" || !checks[0].GetPassed() || len(checks[0].GetRemainingPids()) != 0 {
		t.Fatalf("passing cleanup evidence = %v", checks[0])
	}
	if checks[1].GetStage() != "demo" || checks[1].GetSessionId() != "demo-session" || checks[1].GetPassed() || len(checks[1].GetRemainingPids()) != 1 || checks[1].GetRemainingPids()[0] != 42 || checks[1].GetAssertion() != "descendant remains" {
		t.Fatalf("failed cleanup evidence = %v", checks[1])
	}
}
