package desktopwebrtc

import (
	"context"
	"time"

	desktopv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// JPEGSource supplies bounded snapshots for the same-origin fallback.
// A frame still requires a live session and its current epoch.
type JPEGSource interface {
	CaptureJPEG(context.Context, string) ([]byte, error)
}

// 90 KiB expands to 120 KiB in Connect JSON, within the web proxy limit.
const MaxJPEGBytes = 90 * 1024
const minFrameInterval = 150 * time.Millisecond

func (m *Manager) ReadFrame(ctx context.Context, req *desktopv1.ReadFrameRequest) (*desktopv1.ReadFrameResponse, error) {
	source, ok := m.provider.(JPEGSource)
	if !ok {
		return nil, ErrNotReady
	}
	if req == nil {
		return nil, ErrInvalidCall
	}

	m.mu.Lock()
	s, err := m.lookupLocked(req.GetSession())
	if err != nil {
		m.mu.Unlock()
		return nil, err
	}
	if err := frameLease(s, req); err != nil {
		m.mu.Unlock()
		return nil, err
	}
	now := m.now()
	if s.peer != nil || m.frameBusy || now.Before(s.nextFrame) {
		m.mu.Unlock()
		return nil, ErrFrameBusy
	}
	s.nextFrame = now.Add(minFrameInterval)
	m.frameBusy = true
	s.inFlight++
	display := s.session.GetSelectedDisplayId()
	m.mu.Unlock()

	jpeg, captureErr := source.CaptureJPEG(ctx, display)

	m.mu.Lock()
	defer func() {
		retryCleanup := s.revoked && s.cleanupPending && s.inFlight == 0
		m.mu.Unlock()
		if retryCleanup {
			m.Revoke(req.GetSession().GetSessionId())
		}
	}()
	m.frameBusy = false
	s.inFlight--
	current, err := m.lookupLocked(req.GetSession())
	if err != nil || current != s {
		return nil, ErrRevoked
	}
	if err := frameLease(s, req); err != nil {
		return nil, err
	}
	if captureErr != nil || len(jpeg) == 0 || len(jpeg) > MaxJPEGBytes {
		return nil, ErrNotReady
	}
	return &desktopv1.ReadFrameResponse{Jpeg: jpeg, CapturedAt: timestamppb.New(now)}, nil
}

func frameLease(s *state, req *desktopv1.ReadFrameRequest) error {
	if !proto.Equal(req.GetSession(), s.session.GetRef()) || s.releasing || req.GetLeaseId() == "" || req.GetLeaseId() != s.session.GetLeaseId() {
		return ErrNotControl
	}
	if req.GetLeaseEpoch() != s.session.GetLeaseEpoch() {
		return ErrStaleEpoch
	}
	return nil
}
