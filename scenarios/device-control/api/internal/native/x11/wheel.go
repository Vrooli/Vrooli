package x11

import (
	"device-control/internal/sessions"
	"github.com/jezek/xgb/xproto"
	desktopv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop"
	"google.golang.org/protobuf/encoding/protojson"
)

func (b *Backend) wheel(command sessions.DesktopCommand) (*desktopv1.WheelAction, error) {
	var action desktopv1.Action
	if len(command.Payload) > 64*1024 || protojson.Unmarshal(command.Payload, &action) != nil {
		return nil, ErrUnavailable
	}
	w := action.GetWheel()
	if w == nil || w.DisplayId != b.displayID {
		return nil, ErrUnavailable
	}
	// Widen before taking an absolute value so min-int32 cannot overflow.
	h, v := int64(w.HorizontalTicks), int64(w.VerticalTicks)
	if h < 0 {
		h = -h
	}
	if v < 0 {
		v = -v
	}
	if h+v == 0 || h+v > 20 {
		return nil, ErrUnavailable
	}
	width, height, revision, err := b.geometry()
	if err != nil || b.revision == "" || command.GeometryRevision != b.revision || revision != b.revision || !(w.X >= 0 && w.X < float64(width) && w.Y >= 0 && w.Y < float64(height)) {
		return nil, ErrUnavailable
	}
	return w, nil
}

// applyWheel runs under the same server grab as geometry validation. Every
// detent is a press/release pair; uncertain releases retain cleanup ownership.
func (b *Backend) applyWheel(command sessions.DesktopCommand) error {
	w, err := b.wheel(command)
	if err != nil {
		return err
	}
	if err := b.event(xproto.MotionNotify, 0, int16(w.X), int16(w.Y)); err != nil {
		return err
	}
	for _, axis := range []struct {
		ticks              int32
		negative, positive byte
	}{{w.VerticalTicks, 4, 5}, {w.HorizontalTicks, 6, 7}} {
		button, count := axis.positive, axis.ticks
		if count < 0 {
			button, count = axis.negative, -count
		}
		for i := int32(0); i < count; i++ {
			b.held[button] = true
			if err := b.event(xproto.ButtonPress, button, 0, 0); err != nil {
				return err
			}
			if err := b.event(xproto.ButtonRelease, button, 0, 0); err != nil {
				return err
			}
			delete(b.held, button)
		}
	}
	return nil
}
