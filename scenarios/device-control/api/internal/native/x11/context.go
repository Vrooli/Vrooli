package x11

import (
	"context"
	"encoding/binary"
	"image"
	"time"

	"device-control/internal/sessions"
	"github.com/jezek/xgb/composite"
	"github.com/jezek/xgb/res"
	"github.com/jezek/xgb/xproto"
)

// ActivationContext is short-lived native evidence. Window IDs and PIDs are
// observations, never renderer-facing capabilities or permission to actuate.
// PointerWindow is the root's immediate child and may be a decoration: consumers
// must not assume it uniquely identifies an application or selected element.
type ActivationContext = sessions.DesktopActivationContext

func (b *Backend) windowProperty(window xproto.Window, name string, kind xproto.Atom) (uint32, error) {
	atom, err := xproto.InternAtom(b.conn, true, uint16(len(name)), name).Reply()
	if err != nil || atom == nil || atom.Atom == 0 {
		return 0, ErrUnavailable
	}
	value, err := xproto.GetProperty(b.conn, false, window, atom.Atom, kind, 0, 1).Reply()
	if err != nil || value == nil || value.Type != kind || value.Format != 32 || value.ValueLen != 1 || value.BytesAfter != 0 || len(value.Value) != 4 {
		return 0, ErrUnavailable
	}
	return binary.LittleEndian.Uint32(value.Value), nil
}

// CaptureActivation samples before the companion takes focus. It neither takes
// focus nor captures pixels/text. A changing desktop yields no usable snapshot.
func (b *Backend) CaptureActivation(ctx context.Context) (ActivationContext, error) {
	metadata, _, err := b.captureActivation(ctx, false)
	return metadata, err
}

// CaptureActivationImage reads an already-redirected client pixmap. It never
// changes compositing policy or falls back to pixels from overlapping windows.
func (b *Backend) CaptureActivationImage(ctx context.Context) (sessions.DesktopActivationImage, error) {
	metadata, pixels, err := b.captureActivation(ctx, true)
	if err != nil {
		return sessions.DesktopActivationImage{}, err
	}
	return sessions.DesktopActivationImage{Context: metadata, Image: pixels}, nil
}

func (b *Backend) captureActivation(ctx context.Context, includeImage bool) (ActivationContext, *image.RGBA, error) {
	var pixels *image.RGBA
	var result ActivationContext
	err := b.operation(ctx, true, func() error {
		_, _, revision, err := b.geometry()
		if err != nil {
			return err
		}
		active, err := b.windowProperty(b.root, "_NET_ACTIVE_WINDOW", xproto.AtomWindow)
		if err != nil || active == 0 || xproto.Window(active) == b.root {
			return ErrUnavailable
		}
		bounds, err := b.activationBounds(xproto.Window(active))
		if err != nil {
			return err
		}
		pid, err := b.windowProperty(xproto.Window(active), "_NET_WM_PID", xproto.AtomCardinal)
		if err != nil || pid == 0 {
			return ErrUnavailable
		}
		if includeImage {
			pixels, err = b.activationPixels(xproto.Window(active), bounds)
			if err != nil {
				return err
			}
		}
		pointer, err := xproto.QueryPointer(b.conn, b.root).Reply()
		if err != nil || pointer == nil || !pointer.SameScreen || pointer.Root != b.root {
			return ErrUnavailable
		}
		current, err := b.windowProperty(b.root, "_NET_ACTIVE_WINDOW", xproto.AtomWindow)
		if err != nil || current != active {
			return ErrUnavailable
		}
		afterBounds, err := b.activationBounds(xproto.Window(active))
		if err != nil || afterBounds != bounds {
			return ErrUnavailable
		}
		_, _, after, err := b.geometry()
		if err != nil || after != revision || b.check(ctx, b.peer) != nil {
			return ErrUnavailable
		}
		result = ActivationContext{
			SourceBounds: bounds, ActiveWindow: uint64(active), PointerWindow: uint64(pointer.Child), ProcessID: pid,
			PointerX: int32(pointer.RootX), PointerY: int32(pointer.RootY), DisplayID: b.displayID, GeometryRevision: revision, CapturedAt: time.Now(),
		}
		return nil
	})
	if err != nil {
		return ActivationContext{}, nil, err
	}
	return result, pixels, nil
}

// VerifyWindowProcess binds a companion-owned X resource to its kernel-authenticated
// local IPC PID. Unlike _NET_WM_PID, XRes identity comes from the X server. The
// caller must obtain expectedPID from its authenticated transport, never JSON.
func (b *Backend) VerifyWindowProcess(ctx context.Context, window uint64, expectedPID uint32) error {
	if window == 0 || window > uint64(^uint32(0)) || expectedPID == 0 || xproto.Window(window) == b.root {
		return ErrUnavailable
	}
	return b.operation(ctx, true, func() error {
		geometry, err := xproto.GetGeometry(b.conn, xproto.Drawable(window)).Reply()
		if err != nil || geometry == nil || geometry.Root != b.root {
			return ErrUnavailable
		}
		if res.Init(b.conn) != nil {
			return ErrUnavailable
		}
		version, err := res.QueryVersion(b.conn, 1, 2).Reply()
		if err != nil || version == nil || version.ServerMajor < 1 || (version.ServerMajor == 1 && version.ServerMinor < 2) {
			return ErrUnavailable
		}
		clients, err := res.QueryClients(b.conn).Reply()
		if err != nil || clients == nil || clients.NumClients > 4096 || len(clients.Clients) != int(clients.NumClients) {
			return ErrUnavailable
		}
		var base uint32
		matches := 0
		for _, client := range clients.Clients {
			if uint32(window) & ^client.ResourceMask == client.ResourceBase {
				base = client.ResourceBase
				matches++
			}
		}
		if matches != 1 || base == 0 {
			return ErrUnavailable
		}
		ids, err := res.QueryClientIds(b.conn, 1, []res.ClientIdSpec{{Client: base, Mask: res.ClientIdMaskLocalClientPID}}).Reply()
		if err != nil || ids == nil || ids.NumIds != 1 || len(ids.Ids) != 1 {
			return ErrUnavailable
		}
		id := ids.Ids[0]
		if id.Spec.Client != base || id.Spec.Mask != res.ClientIdMaskLocalClientPID || id.Length != 4 || len(id.Value) != 1 || id.Value[0] != expectedPID {
			return ErrUnavailable
		}
		return b.check(ctx, b.peer)
	})
}

func (b *Backend) activationBounds(window xproto.Window) (sessions.DesktopBounds, error) {
	attributes, err := xproto.GetWindowAttributes(b.conn, window).Reply()
	if err != nil || attributes == nil || attributes.MapState != xproto.MapStateViewable {
		return sessions.DesktopBounds{}, ErrUnavailable
	}
	geometry, err := xproto.GetGeometry(b.conn, xproto.Drawable(window)).Reply()
	if err != nil || geometry == nil || geometry.Root != b.root {
		return sessions.DesktopBounds{}, ErrUnavailable
	}
	position, err := xproto.TranslateCoordinates(b.conn, window, b.root, 0, 0).Reply()
	if err != nil || position == nil || !position.SameScreen {
		return sessions.DesktopBounds{}, ErrUnavailable
	}
	bounds := sessions.DesktopBounds{X: int32(position.DstX), Y: int32(position.DstY), Width: uint32(geometry.Width), Height: uint32(geometry.Height)}
	if !bounds.Valid() {
		return sessions.DesktopBounds{}, ErrUnavailable
	}
	return bounds, nil
}

// Composite off-screen storage includes borders. Crop them using current window
// geometry, and release our pixmap reference on every successful naming path.
func (b *Backend) activationPixels(window xproto.Window, bounds sessions.DesktopBounds) (*image.RGBA, error) {
	if !bounds.Valid() || uint64(bounds.Width)*uint64(bounds.Height) > 16*1024*1024 {
		return nil, ErrUnavailable
	}
	if composite.Init(b.conn) != nil {
		return nil, ErrUnavailable
	}
	version, err := composite.QueryVersion(b.conn, 0, 4).Reply()
	if err != nil || version == nil || version.MajorVersion != 0 || version.MinorVersion < 2 {
		return nil, ErrUnavailable
	}
	geometry, err := xproto.GetGeometry(b.conn, xproto.Drawable(window)).Reply()
	if err != nil || geometry == nil || geometry.Root != b.root || uint32(geometry.Width) != bounds.Width || uint32(geometry.Height) != bounds.Height || geometry.BorderWidth > 32767 {
		return nil, ErrUnavailable
	}
	attributes, err := xproto.GetWindowAttributes(b.conn, window).Reply()
	root := xproto.Setup(b.conn).Roots[0]
	// The backend's validated root visual has known byte order and RGB masks.
	// Other visuals need their own decoder; guessing can misrepresent content.
	if err != nil || attributes == nil || attributes.Visual != root.RootVisual || geometry.Depth != root.RootDepth {
		return nil, ErrUnavailable
	}
	pixmap, err := xproto.NewPixmapId(b.conn)
	if err != nil {
		return nil, err
	}
	if composite.NameWindowPixmapChecked(b.conn, window, pixmap).Check() != nil {
		return nil, ErrUnavailable
	}
	defer xproto.FreePixmap(b.conn, pixmap)
	backing, err := xproto.GetGeometry(b.conn, xproto.Drawable(pixmap)).Reply()
	if err != nil || backing == nil || backing.Root != b.root || backing.Depth != geometry.Depth || uint32(backing.Width) != bounds.Width+2*uint32(geometry.BorderWidth) || uint32(backing.Height) != bounds.Height+2*uint32(geometry.BorderWidth) {
		return nil, ErrUnavailable
	}
	reply, err := xproto.GetImage(b.conn, xproto.ImageFormatZPixmap, xproto.Drawable(pixmap), int16(geometry.BorderWidth), int16(geometry.BorderWidth), geometry.Width, geometry.Height, 0xffffffff).Reply()
	if err != nil || reply == nil || reply.Depth != geometry.Depth || len(reply.Data) != int(bounds.Width)*int(bounds.Height)*4 {
		return nil, ErrUnavailable
	}
	pixels := image.NewRGBA(image.Rect(0, 0, int(bounds.Width), int(bounds.Height)))
	for i := 0; i < len(reply.Data); i += 4 {
		v := binary.LittleEndian.Uint32(reply.Data[i:])
		pixels.Pix[i] = byte(v >> 16)
		pixels.Pix[i+1] = byte(v >> 8)
		pixels.Pix[i+2] = byte(v)
		pixels.Pix[i+3] = 255
	}
	return pixels, nil
}
