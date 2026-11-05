// Package wayland implements the compositor-mediated Linux desktop boundary.
// It deliberately uses the XDG desktop portals instead of XWayland or shell
// helpers. A portal grant is established in the logged-in user session and
// every operation remains bound to that session.
package wayland

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"device-control/internal/sessions"
	"github.com/godbus/dbus/v5"
	"github.com/google/uuid"
	desktopv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	portalName              = "org.freedesktop.portal.Desktop"
	portalPath              = dbus.ObjectPath("/org/freedesktop/portal/desktop")
	requestIface            = "org.freedesktop.portal.Request"
	maxCaptureBytes         = 32 << 20
	maxCapturePixels        = 64 << 20
	portalTimeout           = 30 * time.Second
	deviceKeyboard   uint32 = 1
	devicePointer    uint32 = 2
)

var (
	ErrUnavailable      = errors.New("Wayland portal desktop unavailable")
	ErrPermissionDenied = errors.New("Wayland portal permission denied")
)

// Backend is safe for concurrent controller calls. Portal requests are
// serialized because a single helper owns one short-lived remote-desktop
// grant. The portal never receives a caller-supplied object path.
type Backend struct {
	mu             sync.Mutex
	conn           *dbus.Conn
	busID          string
	compositor     string
	session        dbus.ObjectPath
	screenSession  dbus.ObjectPath
	stream         uint32
	started        bool
	heldKeys       map[int32]bool
	heldButtons    map[uint32]bool
	displayID      string
	revision       string
	width          int
	height         int
	closeOnce      sync.Once
	closeErr       error
	requestFn      func(context.Context, string, string, ...any) (map[string]dbus.Variant, error)
	callFn         func(context.Context, string, ...any) error
	sessionCheckFn func(context.Context) error
}

func New(ctx context.Context, compositor string) (*Backend, error) {
	compositor = strings.ToLower(strings.TrimSpace(compositor))
	if compositor != "gnome" && compositor != "kde" {
		return nil, ErrUnavailable
	}
	conn, err := dbus.SessionBusPrivate()
	if err != nil {
		return nil, ErrUnavailable
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if ctx.Err() != nil {
		conn.Close()
		return nil, ctx.Err()
	}
	var busID string
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	err = conn.BusObject().CallWithContext(checkCtx, "org.freedesktop.DBus.GetId", dbus.FlagNoAutoStart).Store(&busID)
	cancel()
	if err != nil || strings.TrimSpace(busID) == "" {
		conn.Close()
		return nil, ErrUnavailable
	}
	return &Backend{conn: conn, busID: busID, compositor: compositor, displayID: "wayland-" + compositor + "-portal", heldKeys: make(map[int32]bool), heldButtons: make(map[uint32]bool)}, nil
}

func (b *Backend) CheckSession(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.checkSessionLocked(ctx)
}

// checkSessionLocked verifies that the helper is still attached to the same
// user session that established the portal connection. Callers already holding
// b.mu use this form so every operation can enforce the boundary without a
// lock re-entry.
func (b *Backend) checkSessionLocked(ctx context.Context) error {
	if b.sessionCheckFn != nil {
		return b.sessionCheckFn(ctx)
	}
	if b.conn == nil && b.requestFn == nil {
		return ErrUnavailable
	}
	if b.conn != nil && b.busID != "" {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		var current string
		if err := b.conn.BusObject().CallWithContext(ctx, "org.freedesktop.DBus.GetId", dbus.FlagNoAutoStart).Store(&current); err != nil || current != b.busID {
			return ErrUnavailable
		}
	}
	return nil
}

func (b *Backend) Observe(ctx context.Context) (sessions.DesktopObservation, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.checkSessionLocked(ctx); err != nil {
		return sessions.DesktopObservation{}, err
	}
	if b.conn == nil && b.requestFn == nil {
		return sessions.DesktopObservation{}, ErrUnavailable
	}
	data, err := b.screenshot(ctx)
	if err != nil {
		return sessions.DesktopObservation{}, err
	}
	img, err := decodePNG(data)
	if err != nil {
		return sessions.DesktopObservation{}, err
	}
	b.width, b.height = img.Bounds().Dx(), img.Bounds().Dy()
	b.revision = fmt.Sprintf("%s/%dx%d/%d", b.displayID, b.width, b.height, time.Now().UnixNano())
	return sessions.DesktopObservation{Image: img, DisplayID: b.displayID, GeometryRevision: b.revision, CapturedAt: time.Now().UTC()}, nil
}

func (b *Backend) Validate(ctx context.Context, command sessions.DesktopCommand) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.checkSessionLocked(ctx); err != nil {
		return err
	}
	if b.conn == nil && b.requestFn == nil || command.GeometryRevision == "" || command.GeometryRevision != b.revision {
		return ErrUnavailable
	}
	return b.validateAction(ctx, command)
}

func (b *Backend) Apply(ctx context.Context, command sessions.DesktopCommand) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.checkSessionLocked(ctx); err != nil {
		return err
	}
	if b.conn == nil && b.requestFn == nil || command.GeometryRevision == "" || command.GeometryRevision != b.revision {
		return ErrUnavailable
	}
	if err := b.validateAction(ctx, command); err != nil {
		return err
	}
	var action desktopv1.Action
	if err := protojson.Unmarshal(command.Payload, &action); err != nil {
		return ErrUnavailable
	}
	if p := action.GetPointer(); p != nil {
		return b.pointer(ctx, p)
	}
	if k := action.GetKey(); k != nil {
		return b.key(ctx, k)
	}
	if w := action.GetWheel(); w != nil {
		return b.wheel(ctx, w)
	}
	return ErrUnavailable
}

func (b *Backend) ReleaseHeld(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.heldKeys) == 0 && len(b.heldButtons) == 0 {
		return nil
	}
	if !b.started || b.session == "" {
		return ErrUnavailable
	}
	var result error
	for keysym := range b.heldKeys {
		if err := b.call(ctx, "NotifyKeyboardKeysym", b.session, map[string]dbus.Variant{}, keysym, uint32(0)); err != nil {
			result = errors.Join(result, err)
		} else {
			delete(b.heldKeys, keysym)
		}
	}
	for button := range b.heldButtons {
		if err := b.call(ctx, "NotifyPointerButton", b.session, map[string]dbus.Variant{}, int32(button), uint32(0)); err != nil {
			result = errors.Join(result, err)
		} else {
			delete(b.heldButtons, button)
		}
	}
	return result
}

func (b *Backend) Close() error {
	b.closeOnce.Do(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), portalTimeout)
		b.closeErr = b.ReleaseHeld(cleanupCtx)
		cancel()
		b.mu.Lock()
		defer b.mu.Unlock()
		if b.started && b.session != "" && (b.conn != nil || b.callFn != nil) {
			ctx, cancel := context.WithTimeout(context.Background(), portalTimeout)
			b.closeErr = errors.Join(b.closeErr, b.closeSession(ctx, b.session))
			cancel()
		}
		if b.conn != nil {
			b.conn.Close()
			b.conn = nil
		}
	})
	return b.closeErr
}

func (b *Backend) validateAction(ctx context.Context, command sessions.DesktopCommand) error {
	var action desktopv1.Action
	if len(command.Payload) == 0 || len(command.Payload) > 64*1024 || protojson.Unmarshal(command.Payload, &action) != nil {
		return ErrUnavailable
	}
	if p := action.GetPointer(); p != nil {
		if p.DisplayId != b.displayID || p.Kind < desktopv1.PointerAction_KIND_MOVE || p.Kind > desktopv1.PointerAction_KIND_CLICK || p.X < 0 || p.Y < 0 || p.X >= float64(b.width) || p.Y >= float64(b.height) {
			return ErrUnavailable
		}
		if p.Kind != desktopv1.PointerAction_KIND_MOVE && p.Button == desktopv1.PointerAction_BUTTON_UNSPECIFIED {
			return ErrUnavailable
		}
		if err := b.ensureRemoteDesktop(ctx); err != nil {
			return err
		}
		if b.stream == 0 {
			return ErrUnavailable
		}
		return nil
	}
	if k := action.GetKey(); k != nil {
		if k.Key == "" || k.Kind < desktopv1.KeyAction_KIND_DOWN || k.Kind > desktopv1.KeyAction_KIND_PRESS {
			return ErrUnavailable
		}
		return b.ensureRemoteDesktop(ctx)
	}
	if w := action.GetWheel(); w != nil {
		if w.DisplayId != b.displayID || w.X < 0 || w.Y < 0 || w.X >= float64(b.width) || w.Y >= float64(b.height) || w.VerticalTicks < -20 || w.VerticalTicks > 20 || w.HorizontalTicks < -20 || w.HorizontalTicks > 20 || (w.VerticalTicks == 0 && w.HorizontalTicks == 0) {
			return ErrUnavailable
		}
		return b.ensureRemoteDesktop(ctx)
	}
	return ErrUnavailable
}

func (b *Backend) screenshot(ctx context.Context) ([]byte, error) {
	options := map[string]dbus.Variant{"interactive": dbus.MakeVariant(false)}
	results, err := b.portalRequest(ctx, "org.freedesktop.portal.Screenshot", "Screenshot", "", options)
	if err != nil {
		return nil, err
	}
	value, ok := results["uri"]
	uri, ok2 := value.Value().(string)
	if !ok || !ok2 {
		return nil, ErrUnavailable
	}
	u, err := url.Parse(uri)
	if err != nil || u.Scheme != "file" || u.Host != "" || !filepath.IsAbs(u.Path) {
		return nil, ErrPermissionDenied
	}
	f, err := os.Open(u.Path)
	if err != nil {
		return nil, ErrPermissionDenied
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxCaptureBytes+1))
	if err != nil || len(data) == 0 || len(data) > maxCaptureBytes {
		return nil, ErrUnavailable
	}
	return data, nil
}

func (b *Backend) ensureRemoteDesktop(ctx context.Context) error {
	if b.started {
		return nil
	}
	created, err := b.portalRequest(ctx, "org.freedesktop.portal.RemoteDesktop", "CreateSession", sessionOptions())
	if err != nil {
		return err
	}
	variant, ok := created["session_handle"]
	if !ok {
		return ErrPermissionDenied
	}
	path, ok := sessionPath(variant)
	if !ok || path == "" {
		return ErrPermissionDenied
	}
	if _, err = b.portalRequest(ctx, "org.freedesktop.portal.RemoteDesktop", "SelectDevices", path, map[string]dbus.Variant{"types": dbus.MakeVariant(deviceKeyboard | devicePointer)}); err != nil {
		return err
	}
	// Select monitor sources on the same RemoteDesktop session before Start.
	// RemoteDesktop.Start then returns the stream list, which is the only safe
	// coordinate space for absolute pointer notifications.
	if _, selectErr := b.portalRequest(ctx, "org.freedesktop.portal.ScreenCast", "SelectSources", path, map[string]dbus.Variant{"types": dbus.MakeVariant(uint32(1))}); selectErr == nil {
		b.screenSession = path
	}
	started, err := b.portalRequest(ctx, "org.freedesktop.portal.RemoteDesktop", "Start", path, "", map[string]dbus.Variant{})
	if err != nil {
		return err
	}
	// An absolute pointer requires a ScreenCast stream. The portal may omit it
	// for keyboard-only grants; refusing in that case prevents a wrong-target
	// relative motion from being mistaken for a coordinate action.
	if stream, ok := extractStreamID(started["streams"]); ok {
		b.stream = stream
	}
	b.session, b.started = path, true
	return nil
}

func sessionOptions() map[string]dbus.Variant {
	// Portal session tokens become object-path elements. Keep them random and
	// path-safe so concurrent helpers cannot collide on a shared user bus.
	return map[string]dbus.Variant{"session_handle_token": dbus.MakeVariant("vrooli" + strings.ReplaceAll(uuid.NewString(), "-", ""))}
}

func sessionPath(value dbus.Variant) (dbus.ObjectPath, bool) {
	switch v := value.Value().(type) {
	case dbus.ObjectPath:
		if v != "" && v.IsValid() {
			return v, true
		}
	case string:
		path := dbus.ObjectPath(v)
		if strings.HasPrefix(v, "/") && path.IsValid() {
			return path, true
		}
	}
	return "", false
}

func extractStreamID(v dbus.Variant) (uint32, bool) {
	streams, ok := v.Value().([]struct {
		ID         uint32
		Properties map[string]dbus.Variant
	})
	if !ok || len(streams) == 0 {
		return 0, false
	}
	return streams[0].ID, true
}

func (b *Backend) pointer(ctx context.Context, p *desktopv1.PointerAction) error {
	if b.stream == 0 {
		return ErrUnavailable
	}
	if err := b.call(ctx, "NotifyPointerMotionAbsolute", b.session, map[string]dbus.Variant{}, b.stream, p.X/float64(b.width), p.Y/float64(b.height)); err != nil {
		return err
	}
	if p.Kind == desktopv1.PointerAction_KIND_MOVE {
		return nil
	}
	button := uint32(1)
	if p.Button == desktopv1.PointerAction_BUTTON_SECONDARY {
		button = 3
	} else if p.Button == desktopv1.PointerAction_BUTTON_MIDDLE {
		button = 2
	}
	if p.Kind == desktopv1.PointerAction_KIND_DOWN && b.heldButtons[button] {
		return nil
	}
	if p.Kind == desktopv1.PointerAction_KIND_UP && !b.heldButtons[button] {
		return ErrUnavailable
	}
	if p.Kind == desktopv1.PointerAction_KIND_CLICK && b.heldButtons[button] {
		return ErrUnavailable
	}
	if p.Kind == desktopv1.PointerAction_KIND_DOWN || p.Kind == desktopv1.PointerAction_KIND_CLICK {
		if b.heldButtons == nil {
			b.heldButtons = make(map[uint32]bool)
		}
		// Record before sending; a transport failure may follow the press effect.
		b.heldButtons[button] = true
		if err := b.call(ctx, "NotifyPointerButton", b.session, map[string]dbus.Variant{}, int32(button), uint32(1)); err != nil {
			return err
		}
	}
	if p.Kind == desktopv1.PointerAction_KIND_UP || p.Kind == desktopv1.PointerAction_KIND_CLICK {
		if err := b.call(ctx, "NotifyPointerButton", b.session, map[string]dbus.Variant{}, int32(button), uint32(0)); err != nil {
			return err
		}
		delete(b.heldButtons, button)
	}
	return nil
}

func (b *Backend) wheel(ctx context.Context, w *desktopv1.WheelAction) error {
	if b.stream == 0 {
		return ErrUnavailable
	}
	if err := b.call(ctx, "NotifyPointerMotionAbsolute", b.session, map[string]dbus.Variant{}, b.stream, w.X/float64(b.width), w.Y/float64(b.height)); err != nil {
		return err
	}
	return b.call(ctx, "NotifyPointerAxis", b.session, map[string]dbus.Variant{}, float64(w.HorizontalTicks*15), float64(w.VerticalTicks*15))
}

func (b *Backend) key(ctx context.Context, k *desktopv1.KeyAction) error {
	keysym, ok := keySym(k.Key)
	if !ok {
		return ErrUnavailable
	}
	send := func(state uint32) error {
		return b.call(ctx, "NotifyKeyboardKeysym", b.session, map[string]dbus.Variant{}, keysym, state)
	}
	switch k.Kind {
	case desktopv1.KeyAction_KIND_DOWN:
		if b.heldKeys == nil {
			b.heldKeys = make(map[int32]bool)
		}
		if b.heldKeys[keysym] {
			return nil
		}
		// Record before sending; a transport failure may follow the press effect.
		b.heldKeys[keysym] = true
		return send(1)
	case desktopv1.KeyAction_KIND_UP:
		if !b.heldKeys[keysym] {
			return ErrUnavailable
		}
		if err := send(0); err != nil {
			return err
		}
		delete(b.heldKeys, keysym)
		return nil
	case desktopv1.KeyAction_KIND_PRESS:
		if b.heldKeys != nil && b.heldKeys[keysym] {
			return ErrUnavailable
		}
		if b.heldKeys == nil {
			b.heldKeys = make(map[int32]bool)
		}
		b.heldKeys[keysym] = true
		if err := send(1); err != nil {
			return err
		}
		if err := send(0); err != nil {
			return err
		}
		delete(b.heldKeys, keysym)
		return nil
	}
	return ErrUnavailable
}

func keySym(key string) (int32, bool) {
	key = strings.TrimSpace(key)
	if len([]rune(key)) == 1 {
		return int32([]rune(key)[0]), true
	}
	m := map[string]int32{
		"ENTER": 0xff0d, "RETURN": 0xff0d, "ESC": 0xff1b, "ESCAPE": 0xff1b,
		"TAB": 0xff09, "BACKSPACE": 0xff08, "SPACE": 0x20,
		"HOME": 0xff50, "LEFT": 0xff51, "UP": 0xff52, "RIGHT": 0xff53, "DOWN": 0xff54,
		"PAGEUP": 0xff55, "PAGEDOWN": 0xff56, "END": 0xff57, "INSERT": 0xff63, "DELETE": 0xffff,
		"SHIFT": 0xffe1, "SHIFTLEFT": 0xffe1, "SHIFTRIGHT": 0xffe2,
		"CTRL": 0xffe3, "CONTROL": 0xffe3, "CTRLLEFT": 0xffe3, "CONTROLLEFT": 0xffe3,
		"CTRLRIGHT": 0xffe4, "CONTROLRIGHT": 0xffe4,
		"ALT": 0xffe9, "ALTLEFT": 0xffe9, "ALTRIGHT": 0xffea,
		"META": 0xffeb, "METALEFT": 0xffeb, "METARIGHT": 0xffec, "SUPER": 0xffeb, "COMMAND": 0xffeb,
		"F1": 0xffbe, "F2": 0xffbf, "F3": 0xffc0, "F4": 0xffc1, "F5": 0xffc2, "F6": 0xffc3,
		"F7": 0xffc4, "F8": 0xffc5, "F9": 0xffc6, "F10": 0xffc7, "F11": 0xffc8, "F12": 0xffc9,
	}
	v, ok := m[strings.ToUpper(key)]
	return v, ok
}

func (b *Backend) call(ctx context.Context, method string, args ...any) error {
	if b.callFn != nil {
		return b.callFn(ctx, method, args...)
	}
	ctx, cancel := context.WithTimeout(ctx, portalTimeout)
	defer cancel()
	return b.conn.Object(portalName, portalPath).CallWithContext(ctx, "org.freedesktop.portal.RemoteDesktop."+method, 0, args...).Err
}

func (b *Backend) closeSession(ctx context.Context, path dbus.ObjectPath) error {
	if b.callFn != nil {
		return b.callFn(ctx, "Session.Close", path)
	}
	ctx, cancel := context.WithTimeout(ctx, portalTimeout)
	defer cancel()
	return b.conn.Object(portalName, path).CallWithContext(ctx, "org.freedesktop.portal.Session.Close", 0).Err
}

func (b *Backend) portalRequest(ctx context.Context, iface, method string, args ...any) (map[string]dbus.Variant, error) {
	if b.requestFn != nil {
		return b.requestFn(ctx, iface, method, args...)
	}
	ctx, cancel := context.WithTimeout(ctx, portalTimeout)
	defer cancel()
	ch := make(chan *dbus.Signal, 8)
	b.conn.Signal(ch)
	defer b.conn.RemoveSignal(ch)
	matchIface := dbus.WithMatchInterface(requestIface)
	matchMember := dbus.WithMatchMember("Response")
	if err := b.conn.AddMatchSignal(matchIface, matchMember); err != nil {
		return nil, ErrUnavailable
	}
	defer b.conn.RemoveMatchSignal(matchIface, matchMember)
	var request dbus.ObjectPath
	if err := b.conn.Object(portalName, portalPath).CallWithContext(ctx, iface+"."+method, 0, args...).Store(&request); err != nil || request == "" {
		return nil, ErrPermissionDenied
	}
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case signal := <-ch:
			if signal == nil || signal.Path != request || len(signal.Body) != 2 {
				continue
			}
			code, ok := signal.Body[0].(uint32)
			if !ok {
				return nil, ErrUnavailable
			}
			if code != 0 {
				return nil, ErrPermissionDenied
			}
			results, ok := signal.Body[1].(map[string]dbus.Variant)
			if !ok {
				return nil, ErrUnavailable
			}
			return results, nil
		}
	}
}

func decodePNG(data []byte) (*image.RGBA, error) {
	if len(data) == 0 || len(data) > maxCaptureBytes {
		return nil, ErrUnavailable
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > maxCapturePixels {
		return nil, ErrUnavailable
	}
	decoded, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, ErrUnavailable
	}
	rgba := image.NewRGBA(decoded.Bounds())
	for y := decoded.Bounds().Min.Y; y < decoded.Bounds().Max.Y; y++ {
		for x := decoded.Bounds().Min.X; x < decoded.Bounds().Max.X; x++ {
			rgba.Set(x, y, decoded.At(x, y))
		}
	}
	return rgba, nil
}

var _ sessions.DesktopNative = (*Backend)(nil)
