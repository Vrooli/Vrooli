package smoketest_test

import (
	"context"
	"scenario-to-desktop-api/smoketest"
	"scenario-to-desktop-api/smoketest/mocks"
	"testing"
	"time"

	"connectrpc.com/connect"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/shared"
)

type connectServiceSpy struct {
	platform string
	calls    chan smokeTestCall
}

type smokeTestCall struct {
	id, scenario, artifact, platform string
	ctx                              context.Context
}

func (s *connectServiceSpy) CurrentPlatform() string { return s.platform }

func (s *connectServiceSpy) PerformSmokeTest(ctx context.Context, id, scenario, artifact, platform string) {
	s.calls <- smokeTestCall{id: id, scenario: scenario, artifact: artifact, platform: platform, ctx: ctx}
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
	start, err := handler.StartSmokeTest(context.Background(), connect.NewRequest(&domainv1.SmokeTestStartRequest{ScenarioName: "hello-desktop", ArtifactPath: "/tmp/hello-desktop.AppImage", Platform: &platform}))
	if err != nil {
		t.Fatalf("StartSmokeTest() error = %v", err)
	}
	if start.Msg.GetSmokeTestId() == "" || start.Msg.GetStatus() != sharedv1.SmokeTestStatus_SMOKE_TEST_STATUS_RUNNING {
		t.Fatalf("StartSmokeTest() = %+v, want queued running test with id", start.Msg)
	}
	if got := store.Statuses[start.Msg.GetSmokeTestId()]; got == nil || got.Status != "running" {
		t.Fatalf("stored status = %+v, want running status", got)
	}
	return start
}

func assertSmokeCall(t *testing.T, calls <-chan smokeTestCall, id string) {
	t.Helper()
	select {
	case call := <-calls:
		if call.id != id || call.scenario != "hello-desktop" || call.artifact != "/tmp/hello-desktop.AppImage" || call.platform != "linux" {
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
		ScreenRecording: &smoketest.ScreenRecordingResult{
			Recorded:      true,
			VideoPath:     "/private/recordings/smoke-evidence.mp4",
			DurationMs:    1250,
			FileSizeBytes: 4096,
		},
	}

	got := smoketest.StatusToProto(status).GetScreenRecording()
	if got == nil {
		t.Fatal("StatusToProto() screen recording = nil, want evidence summary")
	}
	if !got.GetRecorded() || got.GetDurationMs() != 1250 || got.GetFileSizeBytes() != 4096 {
		t.Fatalf("StatusToProto() screen recording = %+v, want recorded evidence metadata", got)
	}
}
