package desktopcompanion

import (
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"device-control/internal/desktophelper"
	"device-control/internal/desktopvideo"
	"device-control/internal/desktopwebrtc"
	"device-control/internal/native/x11"
	"device-control/internal/sessions"

	desktopv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// X11Config identifies an exact operator login. Its values come from trusted
// local provisioning, never from a remote session open or input request.
type X11Config struct {
	Display       uint16
	AuthorityFile string
	SessionID     string
}

type X11Provider struct {
	mu      sync.Mutex
	backend *x11.Backend
	config  X11Config
	version string
	closed  bool
}

func NewX11(config X11Config, version string) (*X11Provider, error) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return nil, desktopvideo.ErrEncode
	}
	return &X11Provider{config: config, version: version}, nil
}

func (p *X11Provider) native(ctx context.Context) (*x11.Backend, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil, x11.ErrUnavailable
	}
	if p.backend != nil {
		return p.backend, nil
	}
	sessionID := p.config.SessionID
	if sessionID == "active-user" {
		bounded, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		answer, err := exec.CommandContext(bounded, "loginctl", "show-user", strconv.Itoa(os.Getuid()), "--property=Display", "--value").Output()
		sessionID = strings.TrimSpace(string(answer))
		if err != nil || sessionID == "" || len(sessionID) > 64 {
			return nil, x11.ErrUnavailable
		}
	}
	backend, err := desktophelper.OpenX11Session(ctx, p.config.Display, p.config.AuthorityFile, sessionID)
	if err == nil {
		p.backend = backend
	}
	return backend, err
}

func (p *X11Provider) Readiness(ctx context.Context, displayID string) (*desktopv1.DesktopReadiness, error) {
	now := time.Now()
	r := &desktopv1.DesktopReadiness{
		State: desktopv1.ReadinessState_READINESS_STATE_NO_SESSION, ReasonCode: "x11_session_inactive",
		Recovery: "restore_the_bound_unlocked_x11_session", SessionType: "x11",
		CompanionVersion: p.version, ObservedAt: timestamppb.New(now), ExpiresAt: timestamppb.New(now.Add(10 * time.Second)),
	}
	backend, err := p.native(ctx)
	if err != nil {
		return r, nil
	}
	view, err := backend.Observe(ctx)
	if err != nil {
		return r, nil
	}
	width, height := view.Image.Bounds().Dx(), view.Image.Bounds().Dy()
	r.Displays = []*desktopv1.Display{{Id: view.DisplayID, Name: "X11 desktop", Width: uint32(width), Height: uint32(height), Scale: 1}}
	if displayID != "" && displayID != view.DisplayID {
		r.State = desktopv1.ReadinessState_READINESS_STATE_NO_DISPLAY
		r.ReasonCode, r.Recovery = "display_not_selected", "select_one_available_display"
		return r, nil
	}
	r.Displays[0].Selected = true
	r.State = desktopv1.ReadinessState_READINESS_STATE_READY
	r.ReasonCode, r.Recovery = "ready", ""
	r.SelectedDisplayId, r.GeometryRevision = view.DisplayID, view.GeometryRevision
	r.CaptureAllowed, r.InputAllowed, r.HttpsFramesAllowed = true, true, true
	return r, nil
}

func (p *X11Provider) CaptureVP8(ctx context.Context, displayID string) ([]byte, time.Duration, error) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 4*time.Second)
	defer cancel()
	backend, err := p.native(ctx)
	if err != nil {
		return nil, 0, err
	}
	view, err := backend.Observe(ctx)
	if err != nil || view.DisplayID != displayID {
		return nil, 0, x11.ErrUnavailable
	}
	width, height := view.Image.Bounds().Dx(), view.Image.Bounds().Dy()
	frame, err := desktopvideo.EncodeRGBA(ctx, width, height, view.Image.Pix)
	if err != nil {
		return nil, 0, err
	}
	if err := backend.CheckSession(ctx); err != nil {
		return nil, 0, err
	}
	return frame, 33 * time.Millisecond, nil
}

func (p *X11Provider) ApplyInput(ctx context.Context, request *desktopv1.InputRequest) error {
	if request == nil || request.GetAction() == nil {
		return desktopwebrtc.ErrInvalidCall
	}
	payload, err := protojson.Marshal(request.GetAction())
	if err != nil {
		return err
	}
	backend, err := p.native(ctx)
	if err != nil {
		return err
	}
	bounded, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return backend.Apply(bounded, sessions.DesktopCommand{
		ID: request.GetCommandId(), GeometryRevision: request.GetGeometryRevision(), Payload: payload,
	})
}

func (*X11Provider) ReadClipboard(context.Context) (string, error) {
	return "", desktopwebrtc.ErrClipboard
}

func (*X11Provider) WriteClipboard(context.Context, string) error {
	return desktopwebrtc.ErrClipboard
}

func (p *X11Provider) ReleaseHeld(ctx context.Context) error {
	p.mu.Lock()
	backend := p.backend
	p.mu.Unlock()
	if backend == nil {
		return nil
	}
	return backend.ReleaseHeld(ctx)
}

func (p *X11Provider) Shutdown() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
	if p.backend == nil {
		return nil
	}
	return p.backend.Close()
}

var _ desktopwebrtc.Provider = (*X11Provider)(nil)
var _ desktopwebrtc.VP8Source = (*X11Provider)(nil)

func (p *X11Provider) CaptureJPEG(ctx context.Context, displayID string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 4*time.Second)
	defer cancel()
	backend, err := p.native(ctx)
	if err != nil {
		return nil, err
	}
	view, err := backend.Observe(ctx)
	if err != nil || view.DisplayID != displayID {
		return nil, x11.ErrUnavailable
	}
	frame, err := desktopvideo.EncodeJPEG(view.Image, desktopwebrtc.MaxJPEGBytes)
	if err != nil {
		return nil, err
	}
	if err := backend.CheckSession(ctx); err != nil {
		return nil, err
	}
	return frame, nil
}
