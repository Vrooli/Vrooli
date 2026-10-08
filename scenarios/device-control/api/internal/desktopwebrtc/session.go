// Package desktopwebrtc owns the ephemeral browser-session policy around the
// native desktop provider. It deliberately has no capture or OS input code;
// the provider is the only component allowed to touch the user session.
package desktopwebrtc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/pion/webrtc/v4"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	desktopv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const MaxClipboardBytes = 64 * 1024
const MaxDataMessageBytes = 128 * 1024

var (
	ErrNotReady    = errors.New("desktop is not ready")
	ErrNoSession   = errors.New("desktop session is not open")
	ErrNotControl  = errors.New("desktop session is viewer-only")
	ErrStaleEpoch  = errors.New("desktop lease epoch is stale")
	ErrRevoked     = errors.New("desktop session is revoked")
	ErrClipboard   = errors.New("clipboard capability is not granted")
	ErrInvalidCall = errors.New("invalid desktop session request")
)

// Provider is implemented by the native companion or a deterministic test
// fake. It receives already-authorized commands and never receives a browser
// credential or a Bridge transport token.
type Provider interface {
	Readiness(context.Context, string) (*desktopv1.DesktopReadiness, error)
	ApplyInput(context.Context, *desktopv1.InputRequest) error
	ReadClipboard(context.Context) (string, error)
	WriteClipboard(context.Context, string) error
}

// VP8Source is an optional native capture seam. The provider owns capture and
// encoding; Device Control only schedules bounded ephemeral samples into the
// already-authorized peer.
type VP8Source interface {
	CaptureVP8(context.Context, string) ([]byte, time.Duration, error)
}

type closableProvider interface {
	Close(context.Context) error
}

type heldInputReleaser interface {
	ReleaseHeld(context.Context) error
}

// UnavailableProvider is the production fail-closed default until a native
// companion is configured. It makes the public session service safe to mount
// on hosts that cannot provide an interactive desktop.
type UnavailableProvider struct{}

func (UnavailableProvider) Readiness(context.Context, string) (*desktopv1.DesktopReadiness, error) {
	return &desktopv1.DesktopReadiness{State: desktopv1.ReadinessState_READINESS_STATE_OFFLINE, ReasonCode: "companion_unavailable", Recovery: "install_or_start_desktop_companion"}, nil
}
func (UnavailableProvider) ApplyInput(context.Context, *desktopv1.InputRequest) error {
	return ErrNotReady
}
func (UnavailableProvider) ReadClipboard(context.Context) (string, error) { return "", ErrNotReady }
func (UnavailableProvider) WriteClipboard(context.Context, string) error  { return ErrNotReady }

type Manager struct {
	mu       sync.Mutex
	provider Provider
	now      func() time.Time
	sessions map[string]*state
}

type state struct {
	session       *desktopv1.DesktopSession
	clipboard     bool
	revoked       bool
	peer          *PionPeer
	peerCancel    context.CancelFunc
	captureCancel context.CancelFunc
}

// Signal terminates the browser-to-companion negotiation at Device Control.
// Bridge transports the bounded signal but never parses SDP or owns the peer.
func (m *Manager) Signal(ctx context.Context, req *desktopv1.DesktopSignalRequest) (*desktopv1.DesktopSignalResponse, error) {
	if req == nil || req.GetSession() == nil || strings.TrimSpace(req.GetGeneration()) == "" || len(req.GetPayload()) == 0 || len(req.GetPayload()) > 256*1024 {
		return nil, ErrInvalidCall
	}
	if req.GetKind() != desktopv1.DesktopSignalKind_DESKTOP_SIGNAL_KIND_OFFER && req.GetKind() != desktopv1.DesktopSignalKind_DESKTOP_SIGNAL_KIND_ICE {
		return nil, ErrInvalidCall
	}
	var offer webrtc.SessionDescription
	var candidate webrtc.ICECandidateInit
	switch req.GetKind() {
	case desktopv1.DesktopSignalKind_DESKTOP_SIGNAL_KIND_OFFER:
		if err := json.Unmarshal(req.GetPayload(), &offer); err != nil {
			return nil, fmt.Errorf("decode browser offer: %w", err)
		}
	case desktopv1.DesktopSignalKind_DESKTOP_SIGNAL_KIND_ICE:
		if err := json.Unmarshal(req.GetPayload(), &candidate); err != nil {
			return nil, fmt.Errorf("decode browser ICE candidate: %w", err)
		}
	}
	m.mu.Lock()
	s, err := m.lookupLocked(req.GetSession())
	if err == nil && req.GetLeaseId() != s.session.GetLeaseId() {
		err = ErrNotControl
	}
	if err == nil && req.GetLeaseEpoch() != s.session.GetLeaseEpoch() {
		err = ErrStaleEpoch
	}
	if err == nil && s.peer == nil {
		peerCtx, cancel := context.WithCancel(context.Background())
		var peer *PionPeer
		peer, err = NewPionPeerWithDataHandler(peerCtx, iceConfiguration(req.GetIceServers()), func(label string, payload []byte) {
			m.handleDataMessage(req.GetSession().GetSessionId(), peer, label, payload)
		})
		if err == nil {
			peer.SetFailureHandler(func() { m.Revoke(req.GetSession().GetSessionId()) })
		}
		s.peer = peer
		if err == nil {
			s.peerCancel = cancel
		} else {
			cancel()
		}
	}
	var peer *PionPeer
	startCapture := false
	displayID := ""
	if err == nil {
		peer = s.peer
		displayID = s.session.GetSelectedDisplayId()
		startCapture = s.captureCancel == nil
	}
	m.mu.Unlock()
	if err != nil {
		return &desktopv1.DesktopSignalResponse{ReasonCode: reason(err)}, err
	}
	if startCapture {
		m.startCapture(req.GetSession().GetSessionId(), displayID)
	}
	response := &desktopv1.DesktopSignalResponse{Accepted: true, Kind: req.GetKind(), Generation: req.GetGeneration()}
	switch req.GetKind() {
	case desktopv1.DesktopSignalKind_DESKTOP_SIGNAL_KIND_OFFER:
		answer, err := peer.Answer(ctx, offer)
		if err != nil {
			return nil, err
		}
		response.Kind = desktopv1.DesktopSignalKind_DESKTOP_SIGNAL_KIND_ANSWER
		response.Payload, err = json.Marshal(answer)
		if err != nil {
			return nil, err
		}
	case desktopv1.DesktopSignalKind_DESKTOP_SIGNAL_KIND_ICE:
		if err := peer.AddICECandidate(candidate); err != nil {
			return nil, err
		}
	default:
		return nil, ErrInvalidCall
	}
	response.TransportStats = peer.Stats()
	return response, nil
}

func iceConfiguration(servers []*desktopv1.IceServer) webrtc.Configuration {
	configuration := webrtc.Configuration{}
	for _, server := range servers {
		if server == nil || strings.TrimSpace(server.GetUrl()) == "" {
			continue
		}
		configuration.ICEServers = append(configuration.ICEServers, webrtc.ICEServer{URLs: []string{server.GetUrl()}, Username: server.GetUsername(), Credential: server.GetCredential()})
		if len(configuration.ICEServers) == 8 {
			break
		}
	}
	return configuration
}

type dataEnvelope struct {
	Kind    string          `json:"kind"`
	Request json.RawMessage `json:"request"`
}

type dataReceipt struct {
	Kind    string          `json:"kind"`
	Receipt json.RawMessage `json:"receipt,omitempty"`
	Error   string          `json:"error,omitempty"`
}

func (m *Manager) handleDataMessage(sessionID string, peer *PionPeer, label string, payload []byte) {
	if peer == nil || label != "data" || len(payload) == 0 || len(payload) > MaxDataMessageBytes {
		return
	}
	var envelope dataEnvelope
	if json.Unmarshal(payload, &envelope) != nil || len(envelope.Request) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	var response dataReceipt
	switch envelope.Kind {
	case "input":
		var request desktopv1.InputRequest
		if protojson.Unmarshal(envelope.Request, &request) != nil || request.GetSession() == nil || request.GetSession().GetSessionId() != sessionID {
			return
		}
		receipt, err := m.Input(ctx, &request)
		encodedReceipt, marshalErr := protojson.Marshal(receipt)
		if marshalErr != nil {
			return
		}
		response = dataReceipt{Kind: "input", Receipt: encodedReceipt}
		if err != nil {
			response.Error = reason(err)
		}
	case "clipboard":
		var request desktopv1.ClipboardRequest
		if protojson.Unmarshal(envelope.Request, &request) != nil || request.GetSession() == nil || request.GetSession().GetSessionId() != sessionID {
			return
		}
		receipt, err := m.Clipboard(ctx, &request)
		encodedReceipt, marshalErr := protojson.Marshal(receipt)
		if marshalErr != nil {
			return
		}
		response = dataReceipt{Kind: "clipboard", Receipt: encodedReceipt}
		if err != nil {
			response.Error = reason(err)
		}
	default:
		return
	}
	encoded, err := json.Marshal(response)
	if err == nil && len(encoded) <= MaxDataMessageBytes {
		_ = peer.WriteData(encoded)
	}
}

func NewManager(provider Provider) *Manager {
	return &Manager{provider: provider, now: func() time.Time { return time.Now().UTC() }, sessions: make(map[string]*state)}
}

func (m *Manager) Readiness(ctx context.Context, surface *commonv1.SurfaceRef, displayID string) (*desktopv1.DesktopReadiness, error) {
	if m == nil || m.provider == nil || surface == nil {
		return nil, ErrNotReady
	}
	return m.provider.Readiness(ctx, displayID)
}

func (m *Manager) Open(ctx context.Context, req *desktopv1.OpenSessionRequest) (*desktopv1.DesktopSession, error) {
	if req == nil || req.GetSurface() == nil || req.GetSurface().GetTarget() == nil || strings.TrimSpace(req.GetSurface().GetTarget().GetResourceId()) == "" || strings.TrimSpace(req.GetDisplayId()) == "" {
		return nil, ErrInvalidCall
	}
	ready, err := m.Readiness(ctx, req.GetSurface(), req.GetDisplayId())
	if err != nil || ready == nil || ready.GetState() != desktopv1.ReadinessState_READINESS_STATE_READY {
		return nil, fmt.Errorf("%w: %s", ErrNotReady, readinessReason(ready, err))
	}
	if ready.GetSelectedDisplayId() != req.GetDisplayId() {
		return nil, fmt.Errorf("%w: display_not_selected", ErrNotReady)
	}
	ttl := time.Duration(req.GetTtlSeconds()) * time.Second
	if ttl <= 0 || ttl > 30*time.Minute {
		ttl = 10 * time.Minute
	}
	now := m.now()
	id := fmt.Sprintf("desktop-%d", now.UnixNano())
	s := &desktopv1.DesktopSession{
		Ref:       &commonv1.SessionRef{Surface: req.GetSurface(), SessionId: id},
		ChannelId: "channel-" + id, LeaseId: "lease-" + id, LeaseEpoch: 1,
		SelectedDisplayId: req.GetDisplayId(), Codec: "VP8", ConnectionState: "connecting",
		GeometryRevision: ready.GetGeometryRevision(),
		Controller:       req.GetControl(), ExpiresAt: timestamppb.New(now.Add(ttl)),
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reapExpiredLocked("")
	if req.GetControl() {
		for _, existing := range m.sessions {
			if !existing.revoked && existing.session.GetRef().GetSurface().GetTarget().GetResourceId() == req.GetSurface().GetTarget().GetResourceId() && existing.session.GetController() {
				return nil, ErrNotControl
			}
		}
	}
	m.sessions[id] = &state{session: s, clipboard: req.GetClipboard()}
	return cloneSession(s), nil
}

func (m *Manager) startCapture(sessionID, displayID string) {
	source, ok := m.provider.(VP8Source)
	if !ok {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.mu.Lock()
	s, exists := m.sessions[sessionID]
	if !exists || s.revoked || s.peer == nil || s.captureCancel != nil {
		m.mu.Unlock()
		cancel()
		return
	}
	s.captureCancel = cancel
	m.mu.Unlock()
	go func() {
		for {
			payload, duration, err := source.CaptureVP8(ctx, displayID)
			if err != nil {
				// A native capture failure means the session's media authority is
				// gone or its permission state changed. Keep the lease from
				// appearing healthy while the browser waits forever for frames.
				m.Revoke(sessionID)
				return
			}
			if len(payload) > 0 {
				m.mu.Lock()
				current := m.sessions[sessionID]
				peer := (*PionPeer)(nil)
				if current != nil && !current.revoked {
					peer = current.peer
				}
				m.mu.Unlock()
				if peer != nil {
					if err := peer.AddVP8Sample(ctx, payload, duration); err != nil {
						m.Revoke(sessionID)
						return
					}
				}
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(33 * time.Millisecond):
			}
		}
	}()
}

func (m *Manager) AttachViewer(_ context.Context, req *desktopv1.AttachViewerRequest) (*desktopv1.DesktopSession, error) {
	if req == nil {
		return nil, ErrInvalidCall
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	s, err := m.lookupLocked(req.GetSession())
	if err != nil {
		return nil, err
	}
	s.session.ViewerCount++
	return cloneState(s), nil
}

func (m *Manager) TakeControl(_ context.Context, req *desktopv1.TakeControlRequest) (*desktopv1.DesktopSession, error) {
	if req == nil {
		return nil, ErrInvalidCall
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	s, err := m.lookupLocked(req.GetSession())
	if err != nil {
		return nil, err
	}
	if !req.GetTakeover() && s.session.GetController() {
		return nil, ErrNotControl
	}
	targetID := s.session.GetRef().GetSurface().GetTarget().GetResourceId()
	if !req.GetTakeover() {
		for _, existing := range m.sessions {
			if existing == s || existing.revoked || !existing.session.GetController() {
				continue
			}
			if existing.session.GetRef().GetSurface().GetTarget().GetResourceId() == targetID {
				return nil, ErrNotControl
			}
		}
	}
	if req.GetTakeover() {
		hasController := false
		for _, existing := range m.sessions {
			if existing == s || existing.revoked || !existing.session.GetController() {
				continue
			}
			if existing.session.GetRef().GetSurface().GetTarget().GetResourceId() == targetID {
				hasController = true
			}
		}
		if hasController {
			if releaser, ok := m.provider.(heldInputReleaser); ok {
				if err := releaser.ReleaseHeld(context.Background()); err != nil {
					return nil, err
				}
			}
			for _, existing := range m.sessions {
				if existing == s || existing.revoked || !existing.session.GetController() {
					continue
				}
				if existing.session.GetRef().GetSurface().GetTarget().GetResourceId() == targetID {
					existing.session.Controller = false
					existing.session.LeaseEpoch++
				}
			}
		}
	}
	s.session.Controller = true
	s.session.LeaseEpoch++
	return cloneState(s), nil
}

func (m *Manager) Input(ctx context.Context, req *desktopv1.InputRequest) (*desktopv1.InputReceipt, error) {
	if req == nil {
		return nil, ErrInvalidCall
	}
	m.mu.Lock()
	s, err := m.lookupLocked(req.GetSession())
	if err == nil && (!s.session.GetController() || req.GetLeaseId() != s.session.GetLeaseId()) {
		err = ErrNotControl
	}
	if err == nil && req.GetLeaseEpoch() != s.session.GetLeaseEpoch() {
		err = ErrStaleEpoch
	}
	if err == nil && req.GetDisplayId() != s.session.GetSelectedDisplayId() {
		err = ErrInvalidCall
	}
	if err == nil && req.GetGeometryRevision() == "" {
		err = ErrInvalidCall
	}
	if err == nil && req.GetGeometryRevision() != s.session.GetGeometryRevision() {
		err = ErrStaleEpoch
	}
	m.mu.Unlock()
	if err != nil {
		return &desktopv1.InputReceipt{CommandId: req.GetCommandId(), Outcome: desktopv1.InputOutcome_INPUT_OUTCOME_REJECTED, ReasonCode: reason(err)}, err
	}
	if err = m.provider.ApplyInput(ctx, req); err != nil {
		return &desktopv1.InputReceipt{CommandId: req.GetCommandId(), Outcome: desktopv1.InputOutcome_INPUT_OUTCOME_UNKNOWN, ReasonCode: "provider_unconfirmed"}, err
	}
	return &desktopv1.InputReceipt{CommandId: req.GetCommandId(), Outcome: desktopv1.InputOutcome_INPUT_OUTCOME_ACCEPTED, ReasonCode: "applied", ObservedAt: timestamppb.New(m.now())}, nil
}

func (m *Manager) Clipboard(ctx context.Context, req *desktopv1.ClipboardRequest) (*desktopv1.ClipboardReceipt, error) {
	if req == nil || len([]byte(req.GetText())) > MaxClipboardBytes {
		return nil, ErrInvalidCall
	}
	m.mu.Lock()
	s, err := m.lookupLocked(req.GetSession())
	if err == nil && (!s.clipboard || !s.session.GetController() || req.GetLeaseId() != s.session.GetLeaseId()) {
		err = ErrClipboard
	}
	if err == nil && req.GetLeaseEpoch() != s.session.GetLeaseEpoch() {
		err = ErrStaleEpoch
	}
	m.mu.Unlock()
	if err != nil {
		return &desktopv1.ClipboardReceipt{CommandId: req.GetCommandId(), Outcome: desktopv1.InputOutcome_INPUT_OUTCOME_REJECTED, ReasonCode: reason(err)}, err
	}
	var text string
	if req.GetDirection() == desktopv1.ClipboardDirection_CLIPBOARD_DIRECTION_TO_DESKTOP {
		err = m.provider.WriteClipboard(ctx, req.GetText())
	} else if req.GetDirection() == desktopv1.ClipboardDirection_CLIPBOARD_DIRECTION_FROM_DESKTOP {
		text, err = m.provider.ReadClipboard(ctx)
	} else {
		err = ErrInvalidCall
	}
	if err != nil {
		return &desktopv1.ClipboardReceipt{CommandId: req.GetCommandId(), Outcome: desktopv1.InputOutcome_INPUT_OUTCOME_UNKNOWN, ReasonCode: "provider_unconfirmed"}, err
	}
	return &desktopv1.ClipboardReceipt{CommandId: req.GetCommandId(), Outcome: desktopv1.InputOutcome_INPUT_OUTCOME_ACCEPTED, TextLength: uint32(len([]byte(text))), Text: text, ReasonCode: "transferred", ObservedAt: timestamppb.New(m.now())}, nil
}

func (m *Manager) Close(ctx context.Context, req *desktopv1.CloseSessionRequest) (*desktopv1.CloseSessionResponse, error) {
	if req == nil {
		return nil, ErrInvalidCall
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	s, err := m.lookupLocked(req.GetSession())
	if err != nil {
		return nil, err
	}
	s.revoked = true
	if s.captureCancel != nil {
		s.captureCancel()
	}
	if s.peerCancel != nil {
		s.peerCancel()
	}
	if s.peer != nil {
		_ = s.peer.Close()
	}
	if releaser, ok := m.provider.(heldInputReleaser); ok {
		if err := releaser.ReleaseHeld(ctx); err != nil {
			return nil, err
		}
	}
	if err := m.closeProviderIfIdleLocked(); err != nil {
		return nil, err
	}
	return &desktopv1.CloseSessionResponse{Closed: true}, nil
}

func (m *Manager) Revoke(sessionID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reapExpiredLocked("")
	s, ok := m.sessions[sessionID]
	if !ok {
		return false
	}
	s.revoked = true
	if s.captureCancel != nil {
		s.captureCancel()
	}
	if s.peerCancel != nil {
		s.peerCancel()
	}
	if s.peer != nil {
		_ = s.peer.Close()
	}
	if releaser, ok := m.provider.(heldInputReleaser); ok {
		_ = releaser.ReleaseHeld(context.Background())
	}
	_ = m.closeProviderIfIdleLocked()
	return true
}

func (m *Manager) closeProviderIfIdleLocked() error {
	for _, current := range m.sessions {
		if !current.revoked && current.session.GetExpiresAt().AsTime().After(m.now()) {
			return nil
		}
	}
	if closer, ok := m.provider.(closableProvider); ok {
		return closer.Close(context.Background())
	}
	return nil
}

func (m *Manager) lookupLocked(ref *commonv1.SessionRef) (*state, error) {
	if ref == nil || strings.TrimSpace(ref.GetSessionId()) == "" {
		return nil, ErrInvalidCall
	}
	if m.reapExpiredLocked(ref.GetSessionId()) {
		return nil, ErrRevoked
	}
	s, ok := m.sessions[ref.GetSessionId()]
	if !ok {
		return nil, ErrNoSession
	}
	if s.revoked {
		return nil, ErrRevoked
	}
	return s, nil
}

func (m *Manager) reapExpiredLocked(exceptionID string) bool {
	now := m.now()
	expiredException := false
	expiredAny := false
	for id, s := range m.sessions {
		if s == nil || s.revoked || s.session.GetExpiresAt().AsTime().After(now) {
			continue
		}
		expiredAny = true
		s.revoked = true
		if s.captureCancel != nil {
			s.captureCancel()
		}
		if s.peerCancel != nil {
			s.peerCancel()
		}
		if s.peer != nil {
			_ = s.peer.Close()
		}
		if releaser, ok := m.provider.(heldInputReleaser); ok {
			_ = releaser.ReleaseHeld(context.Background())
		}
		if id == exceptionID {
			expiredException = true
		}
		delete(m.sessions, id)
	}
	if expiredAny {
		_ = m.closeProviderIfIdleLocked()
	}
	return expiredException
}

func cloneSession(in *desktopv1.DesktopSession) *desktopv1.DesktopSession {
	return proto.Clone(in).(*desktopv1.DesktopSession)
}

func cloneState(in *state) *desktopv1.DesktopSession {
	if in == nil || in.session == nil {
		return nil
	}
	out := cloneSession(in.session)
	if in.peer != nil {
		out.TransportStats = in.peer.Stats()
	}
	return out
}
func readinessReason(r *desktopv1.DesktopReadiness, err error) string {
	if err != nil {
		return err.Error()
	}
	if r == nil {
		return "missing"
	}
	return r.GetReasonCode()
}
func reason(err error) string {
	switch {
	case errors.Is(err, ErrStaleEpoch):
		return "stale_epoch"
	case errors.Is(err, ErrClipboard):
		return "clipboard_not_granted"
	case errors.Is(err, ErrNotControl):
		return "controller_required"
	case errors.Is(err, ErrRevoked):
		return "revoked"
	default:
		return "rejected"
	}
}
