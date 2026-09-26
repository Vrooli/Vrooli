// Package x11 implements native X11 capture and XTest keyboard and pointer input without
// invoking a shell. It is only for an authenticated local X11 user session.
package x11

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"net"
	"strconv"
	"sync"
	"time"

	"device-control/internal/sessions"
	"github.com/jezek/xgb"
	"github.com/jezek/xgb/randr"
	"github.com/jezek/xgb/xproto"
	"github.com/jezek/xgb/xtest"
	desktopv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop"
	"google.golang.org/protobuf/encoding/protojson"
)

var ErrUnavailable = errors.New("X11 desktop unavailable")

// SessionCheck must verify the bound OS login remains active, unlocked and
// authorized. DISPLAY or an X cookie alone is not an authenticated OS session.
type SessionCheck func(context.Context, Peer) error

type Backend struct {
	mu        sync.Mutex
	conn      *xgb.Conn
	wire      net.Conn
	root      xproto.Window
	displayID string
	check     SessionCheck
	peer      Peer
	revision  string
	held      map[byte]bool
	heldKeys  map[string]byte
}

type Snapshot = sessions.DesktopObservation

// newBackend accepts a resolved display number and cookie from trusted session
// bootstrap. It never reads DISPLAY/XAUTHORITY or chooses a remote X server.
func newBackend(ctx context.Context, display uint16, cookieHex string, check SessionCheck) (*Backend, error) {
	if check == nil {
		return nil, ErrUnavailable
	}
	cookie, err := hex.DecodeString(cookieHex)
	if err != nil || len(cookie) != 16 {
		return nil, ErrUnavailable
	}
	wire, err := (&net.Dialer{}).DialContext(ctx, "unix", "/tmp/.X11-unix/X"+strconv.Itoa(int(display)))
	if err != nil {
		return nil, ErrUnavailable
	}
	peer, err := serverPeer(wire)
	if err != nil || check(ctx, peer) != nil {
		wire.Close()
		return nil, ErrUnavailable
	}
	deadline := time.Now().Add(5 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	_ = wire.SetDeadline(deadline)
	stop := context.AfterFunc(ctx, func() { wire.Close() })
	defer stop()
	conn, err := xgb.NewConnNetWithCookieHex(wire, cookieHex)
	if err != nil {
		wire.Close()
		return nil, ErrUnavailable
	}
	fail := func() (*Backend, error) { conn.Close(); wire.Close(); return nil, ErrUnavailable }
	if xtest.Init(conn) != nil || randr.Init(conn) != nil {
		return fail()
	}
	setup := xproto.Setup(conn)
	if len(setup.Roots) != 1 || setup.ImageByteOrder != xproto.ImageOrderLSBFirst {
		return fail()
	}
	root := setup.Roots[0]
	formatOK := false
	for _, f := range setup.PixmapFormats {
		if f.Depth == root.RootDepth && f.BitsPerPixel == 32 && f.ScanlinePad == 32 {
			formatOK = true
		}
	}
	visualOK := false
	for _, depth := range root.AllowedDepths {
		for _, v := range depth.Visuals {
			if v.VisualId == root.RootVisual && v.Class == xproto.VisualClassTrueColor && v.RedMask == 0xff0000 && v.GreenMask == 0xff00 && v.BlueMask == 0xff {
				visualOK = true
			}
		}
	}
	if !formatOK || !visualOK || ctx.Err() != nil {
		return fail()
	}
	_ = wire.SetDeadline(time.Time{})
	return &Backend{conn: conn, wire: wire, root: root.Root, displayID: fmt.Sprintf("x11-root-%d", root.Root), check: check, peer: peer, held: map[byte]bool{}, heldKeys: map[string]byte{}}, nil
}

func (b *Backend) operation(ctx context.Context, check bool, work func() error) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if ctx.Err() != nil || b.conn == nil {
		return ErrUnavailable
	}
	if check && b.check(ctx, b.peer) != nil {
		return ErrUnavailable
	}
	deadline := time.Now().Add(5 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	if b.wire.SetDeadline(deadline) != nil {
		return ErrUnavailable
	}
	stop := context.AfterFunc(ctx, func() { b.wire.Close() })
	defer func() { stop(); _ = b.wire.SetDeadline(time.Time{}) }()
	err := work()
	if ctx.Err() != nil {
		return ErrUnavailable
	}
	return err
}

func (b *Backend) geometry() (uint16, uint16, string, error) {
	g, err := xproto.GetGeometry(b.conn, xproto.Drawable(b.root)).Reply()
	if err != nil || g == nil {
		return 0, 0, "", ErrUnavailable
	}
	if g.Width == 0 || g.Height == 0 || g.Width > 32767 || g.Height > 32767 || uint64(g.Width)*uint64(g.Height) > 16*1024*1024 {
		return 0, 0, "", ErrUnavailable
	}
	resources, err := randr.GetScreenResourcesCurrent(b.conn, b.root).Reply()
	if err != nil || resources == nil {
		return 0, 0, "", ErrUnavailable
	}
	hash := sha256.Sum256([]byte(fmt.Sprintf("%d/%d/%d/%d", b.root, g.Width, g.Height, resources.ConfigTimestamp)))
	return g.Width, g.Height, hex.EncodeToString(hash[:]), nil
}

func (b *Backend) Observe(ctx context.Context) (Snapshot, error) {
	var snapshot Snapshot
	err := b.operation(ctx, true, func() error {
		width, height, revision, err := b.geometry()
		if err != nil {
			return err
		}
		reply, err := xproto.GetImage(b.conn, xproto.ImageFormatZPixmap, xproto.Drawable(b.root), 0, 0, width, height, 0xffffffff).Reply()
		if err != nil || reply == nil || len(reply.Data) != int(width)*int(height)*4 {
			return ErrUnavailable
		}
		pixels := image.NewRGBA(image.Rect(0, 0, int(width), int(height)))
		for i := 0; i < len(reply.Data); i += 4 {
			v := binary.LittleEndian.Uint32(reply.Data[i:])
			pixels.Pix[i] = byte(v >> 16)
			pixels.Pix[i+1] = byte(v >> 8)
			pixels.Pix[i+2] = byte(v)
			pixels.Pix[i+3] = 255
		}
		// Reject observations taken across a display-layout transition.
		_, _, current, err := b.geometry()
		if err != nil || current != revision {
			return ErrUnavailable
		}
		b.revision = revision
		snapshot = Snapshot{Image: pixels, DisplayID: b.displayID, GeometryRevision: revision, CapturedAt: time.Now().UTC()}
		return nil
	})
	return snapshot, err
}

func (b *Backend) pointer(command sessions.DesktopCommand) (*desktopv1.PointerAction, error) {
	var action desktopv1.Action
	if len(command.Payload) > 64*1024 || protojson.Unmarshal(command.Payload, &action) != nil {
		return nil, ErrUnavailable
	}
	p := action.GetPointer()
	if p == nil || p.DisplayId != b.displayID || p.Kind < desktopv1.PointerAction_KIND_MOVE || p.Kind > desktopv1.PointerAction_KIND_CLICK || p.Button < 0 || p.Button > desktopv1.PointerAction_BUTTON_MIDDLE {
		return nil, ErrUnavailable
	}
	width, height, revision, err := b.geometry()
	if err != nil || b.revision == "" || command.GeometryRevision != b.revision || revision != b.revision || !(p.X >= 0 && p.X < float64(width) && p.Y >= 0 && p.Y < float64(height)) {
		return nil, ErrUnavailable
	}
	if p.Kind != desktopv1.PointerAction_KIND_MOVE && p.Button == desktopv1.PointerAction_BUTTON_UNSPECIFIED {
		return nil, ErrUnavailable
	}
	if p.Kind != desktopv1.PointerAction_KIND_MOVE {
		button := pointerButton(p.Button)
		if p.Kind == desktopv1.PointerAction_KIND_UP {
			if !b.held[button] {
				return nil, ErrUnavailable
			}
		} else if b.held[button] {
			if p.Kind == desktopv1.PointerAction_KIND_CLICK {
				return nil, ErrUnavailable
			}
		} else {
			state, err := xproto.QueryPointer(b.conn, b.root).Reply()
			if err != nil || state == nil || state.Mask&(uint16(1)<<(7+button)) != 0 {
				return nil, ErrUnavailable
			}
		}
	}
	return p, nil
}
func pointerButton(button desktopv1.PointerAction_Button) byte {
	return map[desktopv1.PointerAction_Button]byte{desktopv1.PointerAction_BUTTON_PRIMARY: 1, desktopv1.PointerAction_BUTTON_MIDDLE: 2, desktopv1.PointerAction_BUTTON_SECONDARY: 3}[button]
}
func (b *Backend) Validate(ctx context.Context, command sessions.DesktopCommand) error {
	return b.operation(ctx, true, func() error {
		var action desktopv1.Action
		if protojson.Unmarshal(command.Payload, &action) != nil {
			return ErrUnavailable
		}
		if action.GetWheel() != nil {
			_, err := b.wheel(command)
			return err
		}
		if action.GetKey() != nil {
			_, _, err := b.key(command)
			return err
		}
		_, err := b.pointer(command)
		return err
	})
}
func (b *Backend) event(kind, button byte, x, y int16) error {
	return xtest.FakeInputChecked(b.conn, kind, button, 0, b.root, x, y, 0).Check()
}
func (b *Backend) Apply(ctx context.Context, command sessions.DesktopCommand) error {
	return b.operation(ctx, true, func() (result error) {
		// Keep layout validation and input contiguous in the X server request
		// stream. Closing a failed connection also releases its server grab.
		if err := xproto.GrabServerChecked(b.conn).Check(); err != nil {
			return err
		}
		defer func() { result = errors.Join(result, xproto.UngrabServerChecked(b.conn).Check()) }()
		var action desktopv1.Action
		if protojson.Unmarshal(command.Payload, &action) != nil {
			return ErrUnavailable
		}
		if action.GetWheel() != nil {
			return b.applyWheel(command)
		}
		if action.GetKey() != nil {
			return b.applyKey(command)
		}
		p, err := b.pointer(command)
		if err != nil {
			return err
		}
		if err = b.event(xproto.MotionNotify, 0, int16(p.X), int16(p.Y)); err != nil {
			return err
		}
		button := pointerButton(p.Button)
		if p.Kind == desktopv1.PointerAction_KIND_DOWN && b.held[button] {
			return nil
		}
		if p.Kind == desktopv1.PointerAction_KIND_DOWN || p.Kind == desktopv1.PointerAction_KIND_CLICK {
			// Record before sending: a transport failure may follow the press effect.
			b.held[button] = true
			if err = b.event(xproto.ButtonPress, button, 0, 0); err != nil {
				return err
			}
		}
		if p.Kind == desktopv1.PointerAction_KIND_UP || p.Kind == desktopv1.PointerAction_KIND_CLICK {
			if err = b.event(xproto.ButtonRelease, button, 0, 0); err != nil {
				return err
			}
			delete(b.held, button)
		}
		return nil
	})
}
func (b *Backend) ReleaseHeld(ctx context.Context) error {
	return b.operation(ctx, false, func() error {
		var result error
		for name, code := range b.heldKeys {
			if err := b.event(xproto.KeyRelease, code, 0, 0); err != nil {
				result = errors.Join(result, err)
			} else {
				delete(b.heldKeys, name)
			}
		}
		for button := range b.held {
			if err := b.event(xproto.ButtonRelease, button, 0, 0); err != nil {
				result = errors.Join(result, err)
			} else {
				delete(b.held, button)
			}
		}
		return result
	})
}
func (b *Backend) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := b.ReleaseHeld(ctx)
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.conn != nil {
		b.conn.Close()
		b.wire.Close()
		b.conn = nil
	}
	return err
}

// CheckSession is called by destination lease maintenance even when no client
// sends commands, allowing lock/user-session changes to release held input.
func (b *Backend) CheckSession(ctx context.Context) error {
	return b.operation(ctx, true, func() error { return nil })
}
