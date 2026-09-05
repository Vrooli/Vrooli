package x11

import (
	"context"
	"encoding/binary"
	"time"

	"device-control/internal/sessions"
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
		attributes, err := xproto.GetWindowAttributes(b.conn, xproto.Window(active)).Reply()
		if err != nil || attributes == nil || attributes.MapState != xproto.MapStateViewable {
			return ErrUnavailable
		}
		pid, err := b.windowProperty(xproto.Window(active), "_NET_WM_PID", xproto.AtomCardinal)
		if err != nil || pid == 0 {
			return ErrUnavailable
		}
		pointer, err := xproto.QueryPointer(b.conn, b.root).Reply()
		if err != nil || pointer == nil || !pointer.SameScreen || pointer.Root != b.root {
			return ErrUnavailable
		}
		current, err := b.windowProperty(b.root, "_NET_ACTIVE_WINDOW", xproto.AtomWindow)
		if err != nil || current != active {
			return ErrUnavailable
		}
		_, _, after, err := b.geometry()
		if err != nil || after != revision || b.check(ctx, b.peer) != nil {
			return ErrUnavailable
		}
		result = ActivationContext{ActiveWindow: uint64(active), PointerWindow: uint64(pointer.Child), ProcessID: pid,
			PointerX: int32(pointer.RootX), PointerY: int32(pointer.RootY), DisplayID: b.displayID, GeometryRevision: revision, CapturedAt: time.Now()}
		return nil
	})
	if err != nil {
		return ActivationContext{}, err
	}
	return result, nil
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
