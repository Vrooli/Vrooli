package desktopwebrtc

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	desktopv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type fakeProvider struct {
	ready           desktopv1.ReadinessState
	selectedDisplay string
	clipboard       string
	inputs          int
	releases        int
	closes          int
	releaseSignal   chan struct{}
}

type captureProvider struct {
	fakeProvider
	captures   chan struct{}
	captureErr error
}

func TestIceConfigurationPreservesEphemeralTURNMetadata(t *testing.T) {
	configuration := iceConfiguration([]*desktopv1.IceServer{
		{Url: "turn:relay.example", Username: "user", Credential: "credential"},
	})
	if len(configuration.ICEServers) != 1 || configuration.ICEServers[0].URLs[0] != "turn:relay.example" || configuration.ICEServers[0].Credential != "credential" {
		t.Fatalf("configuration = %+v", configuration)
	}
}

func (p *captureProvider) CaptureVP8(context.Context, string) ([]byte, time.Duration, error) {
	select {
	case p.captures <- struct{}{}:
	default:
	}
	if p.captureErr != nil {
		return nil, 0, p.captureErr
	}
	return []byte{0x10, 0x00, 0x00}, 33 * time.Millisecond, nil
}

func (f *fakeProvider) Readiness(context.Context, string) (*desktopv1.DesktopReadiness, error) {
	display := f.selectedDisplay
	if display == "" {
		display = "main"
	}
	return &desktopv1.DesktopReadiness{State: f.ready, ReasonCode: "fixture", SelectedDisplayId: display, GeometryRevision: "geometry-1"}, nil
}
func (f *fakeProvider) ApplyInput(context.Context, *desktopv1.InputRequest) error {
	f.inputs++
	return nil
}
func (f *fakeProvider) ReadClipboard(context.Context) (string, error) { return f.clipboard, nil }
func (f *fakeProvider) WriteClipboard(_ context.Context, text string) error {
	f.clipboard = text
	return nil
}
func (f *fakeProvider) ReleaseHeld(context.Context) error {
	f.releases++
	if f.releaseSignal != nil {
		select {
		case <-f.releaseSignal:
		default:
			close(f.releaseSignal)
		}
	}
	return nil
}
func (f *fakeProvider) Close(context.Context) error {
	f.closes++
	return nil
}

func surface() *commonv1.SurfaceRef {
	return &commonv1.SurfaceRef{SurfaceId: "desktop", OwnerScenario: "device-control", Target: &commonv1.TargetRef{OwnerScenario: "device-control", ResourceId: "minimouse"}}
}

func TestManagerRefusesNotReadyAndRequiresExplicitDisplay(t *testing.T) {
	p := &fakeProvider{ready: desktopv1.ReadinessState_READINESS_STATE_NO_SESSION}
	m := NewManager(p)
	if _, err := m.Open(context.Background(), &desktopv1.OpenSessionRequest{Surface: surface(), DisplayId: "main", Control: true}); err == nil {
		t.Fatal("expected no-session readiness refusal")
	}
	p.ready = desktopv1.ReadinessState_READINESS_STATE_READY
	if _, err := m.Open(context.Background(), &desktopv1.OpenSessionRequest{Surface: surface(), Control: true}); err != ErrInvalidCall {
		t.Fatalf("missing display error = %v", err)
	}
}

func TestManagerRejectsMalformedSessionCalls(t *testing.T) {
	m := NewManager(&fakeProvider{ready: desktopv1.ReadinessState_READINESS_STATE_READY})
	if _, err := m.Open(context.Background(), &desktopv1.OpenSessionRequest{Surface: &commonv1.SurfaceRef{SurfaceId: "desktop"}, DisplayId: "main"}); err != ErrInvalidCall {
		t.Fatalf("missing target error=%v", err)
	}
	if _, err := m.AttachViewer(context.Background(), nil); err != ErrInvalidCall {
		t.Fatalf("nil viewer error=%v", err)
	}
	if _, err := m.TakeControl(context.Background(), nil); err != ErrInvalidCall {
		t.Fatalf("nil takeover error=%v", err)
	}
	if _, err := m.Close(context.Background(), nil); err != ErrInvalidCall {
		t.Fatalf("nil close error=%v", err)
	}
}

func TestManagerFencesRequestedDisplayAndControllerExclusivity(t *testing.T) {
	p := &fakeProvider{ready: desktopv1.ReadinessState_READINESS_STATE_READY, selectedDisplay: "other"}
	m := NewManager(p)
	if _, err := m.Open(context.Background(), &desktopv1.OpenSessionRequest{Surface: surface(), DisplayId: "main", Control: true}); err == nil {
		t.Fatal("expected selected-display mismatch refusal")
	}
	p.selectedDisplay = "main"
	first, err := m.Open(context.Background(), &desktopv1.OpenSessionRequest{Surface: surface(), DisplayId: "main", Control: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Open(context.Background(), &desktopv1.OpenSessionRequest{Surface: surface(), DisplayId: "main", Control: true}); err != ErrNotControl {
		t.Fatalf("second controller error=%v, want=%v", err, ErrNotControl)
	}
	if !m.Revoke(first.GetRef().GetSessionId()) {
		t.Fatal("expected first controller revoke")
	}
}

func TestManagerEnforcesControllerEpochAndExplicitClipboard(t *testing.T) {
	p := &fakeProvider{ready: desktopv1.ReadinessState_READINESS_STATE_READY}
	m := NewManager(p)
	s, err := m.Open(context.Background(), &desktopv1.OpenSessionRequest{Surface: surface(), DisplayId: "main", Control: true, Clipboard: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.AttachViewer(context.Background(), &desktopv1.AttachViewerRequest{Session: s.Ref}); err != nil {
		t.Fatal(err)
	}
	stale := &desktopv1.InputRequest{Session: s.Ref, LeaseId: s.LeaseId, LeaseEpoch: s.LeaseEpoch + 1, DisplayId: "main", GeometryRevision: s.GeometryRevision, CommandId: "stale"}
	if receipt, err := m.Input(context.Background(), stale); err != ErrStaleEpoch || receipt.GetOutcome() != desktopv1.InputOutcome_INPUT_OUTCOME_REJECTED {
		t.Fatalf("stale input receipt=%v err=%v", receipt, err)
	}
	valid := &desktopv1.InputRequest{Session: s.Ref, LeaseId: s.LeaseId, LeaseEpoch: s.LeaseEpoch, DisplayId: "main", GeometryRevision: s.GeometryRevision, CommandId: "click", Action: &desktopv1.Action{Action: &desktopv1.Action_Pointer{Pointer: &desktopv1.PointerAction{Kind: desktopv1.PointerAction_KIND_CLICK}}}}
	if receipt, err := m.Input(context.Background(), valid); err != nil || receipt.GetOutcome() != desktopv1.InputOutcome_INPUT_OUTCOME_ACCEPTED || p.inputs != 1 {
		t.Fatalf("valid input receipt=%v err=%v inputs=%d", receipt, err, p.inputs)
	}
	wrongGeometry := proto.Clone(valid).(*desktopv1.InputRequest)
	wrongGeometry.GeometryRevision = "stale-geometry"
	if receipt, err := m.Input(context.Background(), wrongGeometry); err == nil || receipt.GetOutcome() != desktopv1.InputOutcome_INPUT_OUTCOME_REJECTED {
		t.Fatalf("stale geometry receipt=%v err=%v", receipt, err)
	}
	clip := &desktopv1.ClipboardRequest{Session: s.Ref, LeaseId: s.LeaseId, LeaseEpoch: s.LeaseEpoch, Direction: desktopv1.ClipboardDirection_CLIPBOARD_DIRECTION_TO_DESKTOP, Text: "héllo", CommandId: "clip"}
	if receipt, err := m.Clipboard(context.Background(), clip); err != nil || receipt.GetOutcome() != desktopv1.InputOutcome_INPUT_OUTCOME_ACCEPTED {
		t.Fatalf("clipboard receipt=%v err=%v", receipt, err)
	}
	if p.clipboard != "héllo" {
		t.Fatalf("clipboard = %q", p.clipboard)
	}
	if !m.Revoke(s.Ref.SessionId) {
		t.Fatal("expected revoke")
	}
	if _, err := m.Input(context.Background(), valid); err != ErrRevoked {
		t.Fatalf("post-revoke error = %v", err)
	}
}

func TestManagerRejectsSignalWithoutGeneration(t *testing.T) {
	p := &fakeProvider{ready: desktopv1.ReadinessState_READINESS_STATE_READY}
	m := NewManager(p)
	s, err := m.Open(context.Background(), &desktopv1.OpenSessionRequest{Surface: surface(), DisplayId: "main", Control: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Signal(context.Background(), &desktopv1.DesktopSignalRequest{Session: s.Ref, LeaseId: s.LeaseId, LeaseEpoch: s.LeaseEpoch, Kind: desktopv1.DesktopSignalKind_DESKTOP_SIGNAL_KIND_OFFER, Payload: []byte(`{"type":"offer"}`)}); err != ErrInvalidCall {
		t.Fatalf("missing generation error=%v", err)
	}
}

func TestManagerRejectsMalformedSignalBeforeCreatingPeer(t *testing.T) {
	p := &fakeProvider{ready: desktopv1.ReadinessState_READINESS_STATE_READY}
	m := NewManager(p)
	s, err := m.Open(context.Background(), &desktopv1.OpenSessionRequest{Surface: surface(), DisplayId: "main", Control: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Signal(context.Background(), &desktopv1.DesktopSignalRequest{Session: s.Ref, LeaseId: s.LeaseId, LeaseEpoch: s.LeaseEpoch, Kind: desktopv1.DesktopSignalKind_DESKTOP_SIGNAL_KIND_OFFER, Generation: "generation-1", Payload: []byte("not-json")}); err == nil {
		t.Fatal("expected malformed offer refusal")
	}
	m.mu.Lock()
	peer := m.sessions[s.Ref.GetSessionId()].peer
	m.mu.Unlock()
	if peer != nil {
		t.Fatal("malformed offer created a peer")
	}
}

func TestManagerTakeoverDemotesExistingController(t *testing.T) {
	p := &fakeProvider{ready: desktopv1.ReadinessState_READINESS_STATE_READY}
	m := NewManager(p)
	controller, err := m.Open(context.Background(), &desktopv1.OpenSessionRequest{Surface: surface(), DisplayId: "main", Control: true})
	if err != nil {
		t.Fatal(err)
	}
	viewer, err := m.Open(context.Background(), &desktopv1.OpenSessionRequest{Surface: surface(), DisplayId: "main", Control: false})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.TakeControl(context.Background(), &desktopv1.TakeControlRequest{Session: viewer.Ref, Takeover: false}); err != ErrNotControl {
		t.Fatalf("non-takeover result=%v, want=%v", err, ErrNotControl)
	}
	claimed, err := m.TakeControl(context.Background(), &desktopv1.TakeControlRequest{Session: viewer.Ref, Takeover: true})
	if err != nil || !claimed.GetController() {
		t.Fatalf("takeover session=%v err=%v", claimed, err)
	}
	if _, err := m.Input(context.Background(), &desktopv1.InputRequest{Session: controller.Ref, LeaseId: controller.LeaseId, LeaseEpoch: controller.LeaseEpoch, DisplayId: "main", GeometryRevision: "geometry-1", CommandId: "old-controller", Action: &desktopv1.Action{Action: &desktopv1.Action_Pointer{Pointer: &desktopv1.PointerAction{Kind: desktopv1.PointerAction_KIND_CLICK, DisplayId: "main"}}}}); err != ErrNotControl {
		t.Fatalf("demoted controller input error=%v, want=%v", err, ErrNotControl)
	}
	if p.releases != 1 {
		t.Fatalf("takeover releases=%d, want=1", p.releases)
	}
}

func TestManagerReapsExpiredSessionResources(t *testing.T) {
	p := &fakeProvider{ready: desktopv1.ReadinessState_READINESS_STATE_READY}
	m := NewManager(p)
	now := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	m.now = func() time.Time { return now }
	s, err := m.Open(context.Background(), &desktopv1.OpenSessionRequest{Surface: surface(), DisplayId: "main", Control: true, TtlSeconds: 1})
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(2 * time.Second)
	if _, err := m.Input(context.Background(), &desktopv1.InputRequest{Session: s.Ref}); err != ErrRevoked {
		t.Fatalf("expired input error=%v, want=%v", err, ErrRevoked)
	}
	if _, ok := m.sessions[s.Ref.GetSessionId()]; ok {
		t.Fatal("expired session remained in manager state")
	}
	if p.releases != 1 || p.closes != 1 {
		t.Fatalf("expired cleanup releases=%d closes=%d, want 1/1", p.releases, p.closes)
	}
}

func TestManagerRevokesSessionWhenNativeCaptureFails(t *testing.T) {
	released := make(chan struct{})
	p := &captureProvider{fakeProvider: fakeProvider{ready: desktopv1.ReadinessState_READINESS_STATE_READY, releaseSignal: released}, captures: make(chan struct{}, 1), captureErr: errors.New("screen capture permission revoked")}
	m := NewManager(p)
	s, err := m.Open(context.Background(), &desktopv1.OpenSessionRequest{Surface: surface(), DisplayId: "main", Control: true})
	if err != nil {
		t.Fatal(err)
	}
	peer := &PionPeer{videoQueue: NewBoundedQueue[media.Sample](1), controlQueue: NewBoundedQueue[[]byte](1), dataQueue: NewBoundedQueue[[]byte](1)}
	m.mu.Lock()
	m.sessions[s.GetRef().GetSessionId()].peer = peer
	m.mu.Unlock()
	m.startCapture(s.GetRef().GetSessionId(), "main")
	select {
	case <-released:
	case <-time.After(time.Second):
		t.Fatal("capture failure did not revoke and release the session")
	}
	m.mu.Lock()
	state := m.sessions[s.GetRef().GetSessionId()]
	m.mu.Unlock()
	if state == nil || !state.revoked {
		t.Fatalf("session state after capture failure = %+v", state)
	}
}

func TestManagerReturnsClipboardTextOnlyInAuthorizedReceipt(t *testing.T) {
	p := &fakeProvider{ready: desktopv1.ReadinessState_READINESS_STATE_READY, clipboard: "héllo"}
	m := NewManager(p)
	s, err := m.Open(context.Background(), &desktopv1.OpenSessionRequest{Surface: surface(), DisplayId: "main", Control: true, Clipboard: true})
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := m.Clipboard(context.Background(), &desktopv1.ClipboardRequest{Session: s.Ref, LeaseId: s.LeaseId, LeaseEpoch: s.LeaseEpoch, Direction: desktopv1.ClipboardDirection_CLIPBOARD_DIRECTION_FROM_DESKTOP, CommandId: "read"})
	if err != nil || receipt.GetText() != "héllo" || receipt.GetTextLength() != uint32(len([]byte("héllo"))) {
		t.Fatalf("clipboard receipt=%v err=%v", receipt, err)
	}
}

func TestManagerReleasesHeldInputWhenControllerClosesWithViewerAttached(t *testing.T) {
	p := &fakeProvider{ready: desktopv1.ReadinessState_READINESS_STATE_READY}
	m := NewManager(p)
	s, err := m.Open(context.Background(), &desktopv1.OpenSessionRequest{Surface: surface(), DisplayId: "main", Control: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.AttachViewer(context.Background(), &desktopv1.AttachViewerRequest{Session: s.Ref}); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Close(context.Background(), &desktopv1.CloseSessionRequest{Session: s.Ref}); err != nil {
		t.Fatal(err)
	}
	if p.releases != 1 {
		t.Fatalf("release count = %d", p.releases)
	}
}

func TestManagerDoesNotPersistClipboardContent(t *testing.T) {
	p := &fakeProvider{ready: desktopv1.ReadinessState_READINESS_STATE_READY}
	m := NewManager(p)
	s, err := m.Open(context.Background(), &desktopv1.OpenSessionRequest{Surface: surface(), DisplayId: "main", Control: true, Clipboard: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Clipboard(context.Background(), &desktopv1.ClipboardRequest{Session: s.Ref, LeaseId: s.LeaseId, LeaseEpoch: s.LeaseEpoch, Direction: desktopv1.ClipboardDirection_CLIPBOARD_DIRECTION_TO_DESKTOP, Text: string(make([]byte, MaxClipboardBytes+1))}); err == nil {
		t.Fatal("expected clipboard size refusal")
	}
}

func TestManagerAnswersBrowserOfferThroughTypedSignal(t *testing.T) {
	p := &captureProvider{fakeProvider: fakeProvider{ready: desktopv1.ReadinessState_READINESS_STATE_READY}, captures: make(chan struct{}, 1)}
	m := NewManager(p)
	s, err := m.Open(context.Background(), &desktopv1.OpenSessionRequest{Surface: surface(), DisplayId: "main", Control: true})
	if err != nil {
		t.Fatal(err)
	}
	mediaEngine := &webrtc.MediaEngine{}
	if err := mediaEngine.RegisterCodec(webrtc.RTPCodecParameters{RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8, ClockRate: 90000}, PayloadType: 96}, webrtc.RTPCodecTypeVideo); err != nil {
		t.Fatal(err)
	}
	browser, err := webrtc.NewAPI(webrtc.WithMediaEngine(mediaEngine)).NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		t.Fatal(err)
	}
	defer browser.Close()
	if _, err := browser.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly}); err != nil {
		t.Fatal(err)
	}
	offer, err := browser.CreateOffer(nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := browser.SetLocalDescription(offer); err != nil {
		t.Fatal(err)
	}
	select {
	case <-webrtc.GatheringCompletePromise(browser):
	case <-time.After(5 * time.Second):
		t.Fatal("browser ICE gathering timed out")
	}
	offerBytes, err := json.Marshal(*browser.LocalDescription())
	if err != nil {
		t.Fatal(err)
	}
	answer, err := m.Signal(context.Background(), &desktopv1.DesktopSignalRequest{Session: s.Ref, LeaseId: s.LeaseId, LeaseEpoch: s.LeaseEpoch, Kind: desktopv1.DesktopSignalKind_DESKTOP_SIGNAL_KIND_OFFER, Generation: "generation-1", Payload: offerBytes})
	if err != nil || !answer.GetAccepted() || answer.GetKind() != desktopv1.DesktopSignalKind_DESKTOP_SIGNAL_KIND_ANSWER {
		t.Fatalf("answer=%v err=%v", answer, err)
	}
	select {
	case <-p.captures:
	case <-time.After(time.Second):
		t.Fatal("native VP8 capture pump did not run")
	}
	var description webrtc.SessionDescription
	if err := json.Unmarshal(answer.GetPayload(), &description); err != nil || description.Type != webrtc.SDPTypeAnswer {
		t.Fatalf("answer description=%+v err=%v", description, err)
	}
	if _, err := m.Signal(context.Background(), &desktopv1.DesktopSignalRequest{Session: s.Ref, LeaseId: s.LeaseId + "-stale", LeaseEpoch: s.LeaseEpoch, Kind: desktopv1.DesktopSignalKind_DESKTOP_SIGNAL_KIND_ICE, Generation: "generation-1", Payload: []byte(`{"candidate":"candidate"}`)}); err != ErrNotControl {
		t.Fatalf("stale signal error=%v", err)
	}
	if !m.Revoke(s.Ref.SessionId) {
		t.Fatal("expected revoke")
	}
}

func TestManagerHandlesBrowserDataInputWithReceipt(t *testing.T) {
	p := &captureProvider{fakeProvider: fakeProvider{ready: desktopv1.ReadinessState_READINESS_STATE_READY}, captures: make(chan struct{}, 1)}
	m := NewManager(p)
	s, err := m.Open(context.Background(), &desktopv1.OpenSessionRequest{Surface: surface(), DisplayId: "main", Control: true})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	browser, err := webrtc.NewAPI().NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		t.Fatal(err)
	}
	defer browser.Close()
	responsePayload := make(chan []byte, 1)
	browser.OnDataChannel(func(channel *webrtc.DataChannel) {
		if channel.Label() != "data" {
			return
		}
		channel.OnMessage(func(message webrtc.DataChannelMessage) {
			responsePayload <- append([]byte(nil), message.Data...)
		})
		channel.OnOpen(func() {
			request, marshalErr := protojson.Marshal(&desktopv1.InputRequest{
				Session: s.Ref, LeaseId: s.LeaseId, LeaseEpoch: s.LeaseEpoch,
				CommandId: "data-input", DisplayId: "main", GeometryRevision: "geometry-1",
				Action: &desktopv1.Action{Action: &desktopv1.Action_Pointer{Pointer: &desktopv1.PointerAction{Kind: desktopv1.PointerAction_KIND_MOVE, DisplayId: "main", X: 1, Y: 2}}},
			})
			if marshalErr != nil {
				t.Errorf("marshal input: %v", marshalErr)
				return
			}
			payload, marshalErr := json.Marshal(map[string]any{"kind": "input", "request": json.RawMessage(request)})
			if marshalErr != nil {
				t.Errorf("marshal data envelope: %v", marshalErr)
				return
			}
			if err := channel.SendText(string(payload)); err != nil {
				t.Errorf("send data input: %v", err)
			}
		})
	})
	if _, err := browser.CreateDataChannel("browser-init", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := browser.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo, webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly}); err != nil {
		t.Fatal(err)
	}
	offer, err := browser.CreateOffer(nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := browser.SetLocalDescription(offer); err != nil {
		t.Fatal(err)
	}
	select {
	case <-webrtc.GatheringCompletePromise(browser):
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	offerBytes, err := json.Marshal(*browser.LocalDescription())
	if err != nil {
		t.Fatal(err)
	}
	answer, err := m.Signal(ctx, &desktopv1.DesktopSignalRequest{Session: s.Ref, LeaseId: s.LeaseId, LeaseEpoch: s.LeaseEpoch, Kind: desktopv1.DesktopSignalKind_DESKTOP_SIGNAL_KIND_OFFER, Generation: "generation-data", Payload: offerBytes})
	if err != nil {
		t.Fatal(err)
	}
	var description webrtc.SessionDescription
	if err := json.Unmarshal(answer.GetPayload(), &description); err != nil {
		t.Fatal(err)
	}
	if err := browser.SetRemoteDescription(description); err != nil {
		t.Fatal(err)
	}
	select {
	case payload := <-responsePayload:
		var response dataReceipt
		if err := json.Unmarshal(payload, &response); err != nil {
			t.Fatal(err)
		}
		if response.Kind != "input" || response.Error != "" {
			t.Fatalf("response=%+v", response)
		}
		var receipt desktopv1.InputReceipt
		if err := protojson.Unmarshal(response.Receipt, &receipt); err != nil {
			t.Fatal(err)
		}
		if receipt.GetOutcome() != desktopv1.InputOutcome_INPUT_OUTCOME_ACCEPTED || receipt.GetCommandId() != "data-input" {
			t.Fatalf("receipt=%v", &receipt)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for browser data receipt")
	}
	if p.inputs != 1 {
		t.Fatalf("input count = %d", p.inputs)
	}
	if !m.Revoke(s.Ref.SessionId) {
		t.Fatal("expected revoke")
	}
}

func TestManagerRevokesSessionWhenPeerTransportFails(t *testing.T) {
	p := &captureProvider{
		fakeProvider: fakeProvider{
			ready:         desktopv1.ReadinessState_READINESS_STATE_READY,
			releaseSignal: make(chan struct{}),
		},
		captures: make(chan struct{}, 1),
	}
	m := NewManager(p)
	s, err := m.Open(context.Background(), &desktopv1.OpenSessionRequest{Surface: surface(), DisplayId: "main", Control: true})
	if err != nil {
		t.Fatal(err)
	}
	peer, err := NewPionPeer(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer peer.Close()
	m.mu.Lock()
	m.sessions[s.Ref.SessionId].peer = peer
	m.mu.Unlock()
	peer.videoQueue.Close()
	m.startCapture(s.Ref.SessionId, "main")

	select {
	case <-p.releaseSignal:
	case <-time.After(time.Second):
		t.Fatal("peer transport failure did not revoke the session")
	}
	m.mu.Lock()
	state := m.sessions[s.Ref.SessionId]
	revoked := state != nil && state.revoked
	m.mu.Unlock()
	if !revoked {
		t.Fatal("session remained active after peer transport failure")
	}
	if p.releases != 1 {
		t.Fatalf("release count = %d", p.releases)
	}
}
