package evidence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"scenario-to-desktop-api/captures"
	"scenario-to-desktop-api/livedesktop"

	"connectrpc.com/connect"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/shared"
	"google.golang.org/protobuf/types/known/structpb"
)

type fakeCaptureStore struct {
	items map[string][]captures.Capture
}

func (s *fakeCaptureStore) List(scenario string) ([]captures.Capture, error) {
	return append([]captures.Capture(nil), s.items[scenario]...), nil
}

func (s *fakeCaptureStore) Add(capture captures.Capture) error {
	s.items[capture.ScenarioName] = append(s.items[capture.ScenarioName], capture)
	return nil
}
func (s *fakeCaptureStore) Delete(scenario, captureID string) error { return nil }
func (s *fakeCaptureStore) DeleteAll(scenario string) ([]captures.Capture, error) {
	deleted := s.items[scenario]
	delete(s.items, scenario)
	return deleted, nil
}

func (s *fakeCaptureStore) Summary(scenario string) (captures.CapturesSummary, error) {
	var total int64
	for _, capture := range s.items[scenario] {
		total += capture.FileSizeBytes
	}
	return captures.CapturesSummary{Count: len(s.items[scenario]), TotalBytes: total}, nil
}

type fakeCaptureService struct{ store captures.Store }

func (s fakeCaptureService) Store() captures.Store              { return s.store }
func (s fakeCaptureService) DeleteCapture(string, string) error { return nil }
func (s fakeCaptureService) CleanAll(string) error              { return nil }

type fakeDesktopService struct {
	session          *livedesktop.Session
	sessions         []*livedesktop.Session
	startConfig      livedesktop.SessionConfig
	startErr         error
	getErr           error
	launchErr        error
	heartbeatErr     error
	findPath         string
	findErr          error
	action           *livedesktop.ActionResult
	actionErr        error
	stopErr          error
	lastAction       string
	lastActionParams string
}

func (s *fakeDesktopService) StartSession(_ context.Context, config livedesktop.SessionConfig) (*livedesktop.Session, error) {
	s.startConfig = config
	return s.session, s.startErr
}

func (s *fakeDesktopService) GetSession(string) (*livedesktop.Session, error) {
	return s.session, s.getErr
}
func (s *fakeDesktopService) ListSessions() []*livedesktop.Session { return s.sessions }
func (s *fakeDesktopService) LaunchApp(string, string) error       { return s.launchErr }
func (s *fakeDesktopService) Heartbeat(string) error               { return s.heartbeatErr }
func (s *fakeDesktopService) FindArtifact(string) (string, error)  { return s.findPath, s.findErr }
func (s *fakeDesktopService) ExecuteAction(_ context.Context, _ string, action string, params json.RawMessage) (*livedesktop.ActionResult, error) {
	s.lastAction, s.lastActionParams = action, string(params)
	return s.action, s.actionErr
}
func (s *fakeDesktopService) StopSession(string) error { return s.stopErr }

type deletingCaptureService struct {
	store     captures.Store
	deleteErr error
	cleanErr  error
	deleted   string
	cleaned   string
}

func (s *deletingCaptureService) Store() captures.Store { return s.store }
func (s *deletingCaptureService) DeleteCapture(scenario, captureID string) error {
	s.deleted = scenario + "/" + captureID
	return s.deleteErr
}

func (s *deletingCaptureService) CleanAll(scenario string) error {
	s.cleaned = scenario
	return s.cleanErr
}

func evidenceTestSession() *livedesktop.Session {
	now := time.Now().UTC()
	return &livedesktop.Session{ID: "session-1", ScenarioName: "hello-desktop", State: livedesktop.StateRunning, Platform: "linux", Width: 1280, Height: 720, CreatedAt: now, LastHeartbeat: now}
}

func evidenceString(value string) *string { return &value }
func evidenceInt32(value int32) *int32    { return &value }

func TestConnectServiceRejectsBridgeTargetWithoutDispatch(t *testing.T) {
	handler := NewConnectService(nil, nil)
	_, err := handler.StartDesktopSession(context.Background(), connect.NewRequest(&domainv1.DesktopSessionRequest{
		ScenarioName: "hello-desktop",
		Target:       &domainv1.EvidenceTarget{Kind: domainv1.EvidenceTarget_KIND_BRIDGE_NODE},
	}))
	if connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Fatalf("StartDesktopSession() error code = %v, want unimplemented (err %v)", connect.CodeOf(err), err)
	}
	connectErr := new(connect.Error)
	if !errors.As(err, &connectErr) {
		t.Fatalf("error is not a Connect error: %v", err)
	}
	for _, detail := range connectErr.Details() {
		value, valueErr := detail.Value()
		if valueErr != nil {
			t.Fatalf("decode error detail: %v", valueErr)
		}
		if envelope, ok := value.(*sharedv1.ErrorEnvelope); ok {
			if envelope.GetCode() == "" || envelope.GetCategory() == "" || envelope.GetRecovery() == "" || envelope.GetRecoveryHint() == "" {
				t.Fatalf("incomplete remediation envelope: %#v", envelope)
			}
			return
		}
	}
	t.Fatal("Connect error did not carry a remediation envelope")
}

func TestConnectServiceListsEvidenceCaptureSummary(t *testing.T) {
	store := &fakeCaptureStore{items: map[string][]captures.Capture{
		"hello-desktop": {{ID: "capture-1", ScenarioName: "hello-desktop", FileSizeBytes: 42, CreatedAt: time.Now()}},
	}}
	handler := NewConnectService(nil, fakeCaptureService{store: store})

	response, err := handler.GetEvidenceCapturesSummary(context.Background(), connect.NewRequest(&domainv1.ListEvidenceCapturesRequest{ScenarioName: "hello-desktop"}))
	if err != nil {
		t.Fatalf("GetEvidenceCapturesSummary() error = %v", err)
	}
	if response.Msg.GetCount() != 1 || response.Msg.GetTotalBytes() != 42 {
		t.Fatalf("summary = (%d, %d), want (1, 42)", response.Msg.GetCount(), response.Msg.GetTotalBytes())
	}
}

func TestSessionToProtoPreservesTypedStateAndMetrics(t *testing.T) {
	splashDuration, readyDuration := int64(800), int64(1500)
	currentCPU, currentRSS, peakRSS := 25.5, 150.0, 192.0
	result := sessionToProto(livedesktop.SessionView{
		ID:           "session-1",
		ScenarioName: "hello-desktop",
		State:        livedesktop.StateRunning,
		NetworkMode:  "slow",
		Metrics: &livedesktop.MetricsView{
			SplashDurationMs: &splashDuration,
			SplashDetected:   true,
			ReadyDurationMs:  &readyDuration,
			ReadyDetected:    true,
			CurrentCPU:       &currentCPU,
			CurrentRSSMB:     &currentRSS,
			PeakRSSMB:        &peakRSS,
			SampleCount:      3,
		},
	}, nil)

	if result.GetState() != domainv1.DesktopSessionState_DESKTOP_SESSION_STATE_RUNNING {
		t.Fatalf("state = %v, want running enum", result.GetState())
	}
	if result.GetNetworkMode() != domainv1.DesktopNetworkMode_DESKTOP_NETWORK_MODE_SLOW {
		t.Fatalf("network mode = %v, want slow enum", result.GetNetworkMode())
	}
	if got := networkModeProto(""); got != domainv1.DesktopNetworkMode_DESKTOP_NETWORK_MODE_NORMAL {
		t.Fatalf("empty network mode = %v, want normal enum", got)
	}
	if result.GetMetrics().GetSampleCount() != 3 || result.GetMetrics().GetCurrentCpuPercent() != currentCPU {
		t.Fatalf("metrics = %#v, want sample count 3 and CPU %v", result.GetMetrics(), currentCPU)
	}
}

func TestConnectServiceDesktopLifecycleAndArtifactLookup(t *testing.T) {
	session := evidenceTestSession()
	desktops := &fakeDesktopService{session: session, sessions: []*livedesktop.Session{session}, findPath: "/tmp/hello-desktop.AppImage"}
	handler := NewConnectService(desktops, &deletingCaptureService{store: &fakeCaptureStore{items: map[string][]captures.Capture{}}})

	started, err := handler.StartDesktopSession(context.Background(), connect.NewRequest(&domainv1.DesktopSessionRequest{ScenarioName: "hello-desktop", ArtifactPath: evidenceString("/tmp/hello"), Platform: sharedv1.Platform_PLATFORM_LINUX, Width: evidenceInt32(1280), Height: evidenceInt32(720)}))
	if err != nil || started.Msg.GetSessionId() != session.ID {
		t.Fatalf("StartDesktopSession() = %#v, %v", started, err)
	}
	if desktops.startConfig.Platform != "linux" || desktops.startConfig.Width != 1280 {
		t.Fatalf("start config = %#v", desktops.startConfig)
	}
	if _, err := handler.StartDesktopSession(context.Background(), connect.NewRequest(&domainv1.DesktopSessionRequest{})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("empty scenario error code = %v, want invalid argument", connect.CodeOf(err))
	}
	if got, err := handler.GetDesktopSession(context.Background(), connect.NewRequest(&domainv1.DesktopSessionRef{SessionId: session.ID})); err != nil || got.Msg.GetPlatform() != sharedv1.Platform_PLATFORM_LINUX {
		t.Fatalf("GetDesktopSession() = %#v, %v", got, err)
	}
	if listed, err := handler.ListDesktopSessions(context.Background(), connect.NewRequest(&domainv1.ListDesktopSessionsRequest{ScenarioName: evidenceString("other")})); err != nil || len(listed.Msg.GetSessions()) != 0 {
		t.Fatalf("filtered sessions = %#v, %v", listed, err)
	}
	if _, err := handler.LaunchDesktopArtifact(context.Background(), connect.NewRequest(&domainv1.LaunchDesktopArtifactRequest{SessionId: session.ID, ArtifactPath: evidenceString("/tmp/hello")})); err != nil {
		t.Fatalf("LaunchDesktopArtifact(): %v", err)
	}
	if _, err := handler.HeartbeatDesktopSession(context.Background(), connect.NewRequest(&domainv1.DesktopSessionRef{SessionId: session.ID})); err != nil {
		t.Fatalf("HeartbeatDesktopSession(): %v", err)
	}
	artifact, err := handler.FindDesktopArtifact(context.Background(), connect.NewRequest(&domainv1.FindDesktopArtifactRequest{ScenarioName: "hello-desktop"}))
	if err != nil || artifact.Msg.GetArtifactPath() != desktops.findPath {
		t.Fatalf("FindDesktopArtifact() = %#v, %v", artifact, err)
	}
	if _, err := handler.StopDesktopSession(context.Background(), connect.NewRequest(&domainv1.DesktopSessionRef{SessionId: session.ID})); err != nil {
		t.Fatalf("StopDesktopSession(): %v", err)
	}
}

func TestConnectServiceCaptureControlAndDeletionContracts(t *testing.T) {
	session := evidenceTestSession()
	capture := captures.Capture{ID: "capture-1", ScenarioName: session.ScenarioName, Type: captures.CaptureScreenshot, Filename: "capture.png", FileSizeBytes: 42, Width: 1280, Height: 720, SourceSession: session.ID, CreatedAt: time.Now()}
	store := &fakeCaptureStore{items: map[string][]captures.Capture{session.ScenarioName: {capture}}}
	captureService := &deletingCaptureService{store: store}
	desktops := &fakeDesktopService{session: session, action: &livedesktop.ActionResult{Status: "ok", Message: "done", Data: map[string]any{"capture_id": capture.ID}}}
	handler := NewConnectService(desktops, captureService)

	screenshot, err := handler.CaptureScreenshot(context.Background(), connect.NewRequest(&domainv1.CaptureScreenshotRequest{SessionId: session.ID}))
	if err != nil || screenshot.Msg.GetCapture().GetCaptureId() != capture.ID {
		t.Fatalf("CaptureScreenshot() = %#v, %v", screenshot, err)
	}
	params, err := structpb.NewStruct(map[string]any{"x": 4})
	if err != nil {
		t.Fatal(err)
	}
	control, err := handler.ControlDesktop(context.Background(), connect.NewRequest(&domainv1.DesktopControlRequest{SessionId: session.ID, Action: "click", Params: params}))
	if err != nil || control.Msg.GetResult().GetFields()["status"].GetStringValue() != "ok" || desktops.lastAction != "click" || desktops.lastActionParams == "" {
		t.Fatalf("ControlDesktop() = %#v, %v; action=%q params=%q", control, err, desktops.lastAction, desktops.lastActionParams)
	}
	listed, err := handler.ListEvidenceCaptures(context.Background(), connect.NewRequest(&domainv1.ListEvidenceCapturesRequest{ScenarioName: session.ScenarioName}))
	if err != nil || len(listed.Msg.GetCaptures()) != 1 || listed.Msg.GetCaptures()[0].GetWidth() != 1280 {
		t.Fatalf("ListEvidenceCaptures() = %#v, %v", listed, err)
	}
	if _, err := handler.DeleteEvidenceCapture(context.Background(), connect.NewRequest(&domainv1.EvidenceCaptureRef{ScenarioName: session.ScenarioName, CaptureId: capture.ID})); err != nil || captureService.deleted != "hello-desktop/capture-1" {
		t.Fatalf("DeleteEvidenceCapture() = %v, deleted=%q", err, captureService.deleted)
	}
	if _, err := handler.DeleteAllEvidenceCaptures(context.Background(), connect.NewRequest(&domainv1.ListEvidenceCapturesRequest{ScenarioName: session.ScenarioName})); err != nil || captureService.cleaned != session.ScenarioName {
		t.Fatalf("DeleteAllEvidenceCaptures() = %v, cleaned=%q", err, captureService.cleaned)
	}
}

func TestConnectServiceMapsDesktopAndDurabilityFailures(t *testing.T) {
	session := evidenceTestSession()
	store := &fakeCaptureStore{items: map[string][]captures.Capture{session.ScenarioName: {}}}
	desktops := &fakeDesktopService{session: session, launchErr: errors.New("not ready"), heartbeatErr: errors.New("gone"), findErr: errors.New("missing"), action: &livedesktop.ActionResult{Data: map[string]any{}}, stopErr: errors.New("cannot stop")}
	captures := &deletingCaptureService{store: store, deleteErr: fmt.Errorf("missing"), cleanErr: fmt.Errorf("disk failure")}
	handler := NewConnectService(desktops, captures)

	checks := []struct {
		name string
		err  error
		want connect.Code
	}{
		{"launch", func() error {
			_, err := handler.LaunchDesktopArtifact(context.Background(), connect.NewRequest(&domainv1.LaunchDesktopArtifactRequest{}))
			return err
		}(), connect.CodeFailedPrecondition},
		{"heartbeat", func() error {
			_, err := handler.HeartbeatDesktopSession(context.Background(), connect.NewRequest(&domainv1.DesktopSessionRef{}))
			return err
		}(), connect.CodeNotFound},
		{"find", func() error {
			_, err := handler.FindDesktopArtifact(context.Background(), connect.NewRequest(&domainv1.FindDesktopArtifactRequest{}))
			return err
		}(), connect.CodeNotFound},
		{"capture without durable id", func() error {
			_, err := handler.CaptureScreenshot(context.Background(), connect.NewRequest(&domainv1.CaptureScreenshotRequest{}))
			return err
		}(), connect.CodeInternal},
		{"stop", func() error {
			_, err := handler.StopDesktopSession(context.Background(), connect.NewRequest(&domainv1.DesktopSessionRef{}))
			return err
		}(), connect.CodeFailedPrecondition},
		{"delete", func() error {
			_, err := handler.DeleteEvidenceCapture(context.Background(), connect.NewRequest(&domainv1.EvidenceCaptureRef{}))
			return err
		}(), connect.CodeNotFound},
		{"clean", func() error {
			_, err := handler.DeleteAllEvidenceCaptures(context.Background(), connect.NewRequest(&domainv1.ListEvidenceCapturesRequest{}))
			return err
		}(), connect.CodeInternal},
	}
	for _, check := range checks {
		if got := connect.CodeOf(check.err); got != check.want {
			t.Errorf("%s error code = %v, want %v (err %v)", check.name, got, check.want, check.err)
		}
	}
}
