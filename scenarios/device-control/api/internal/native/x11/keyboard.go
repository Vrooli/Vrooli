package x11

import (
	"device-control/internal/sessions"
	"errors"
	"github.com/jezek/xgb/xproto"
	desktopv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop"
	"google.golang.org/protobuf/encoding/protojson"
)

// Names denote keysyms, not physical US keyboard positions. Printable text is
// a separate action: it must not silently acquire layout-dependent modifiers.
var namedKeys = map[string]xproto.Keysym{
	"Backspace": 0xff08, "Tab": 0xff09, "Enter": 0xff0d, "Escape": 0xff1b,
	"Home": 0xff50, "ArrowLeft": 0xff51, "ArrowUp": 0xff52, "ArrowRight": 0xff53,
	"ArrowDown": 0xff54, "PageUp": 0xff55, "PageDown": 0xff56, "End": 0xff57,
	"Insert": 0xff63, "Delete": 0xffff, "Space": 0x20,
	"ShiftLeft": 0xffe1, "ShiftRight": 0xffe2, "ControlLeft": 0xffe3,
	"ControlRight": 0xffe4, "AltLeft": 0xffe9, "AltRight": 0xffea,
	"MetaLeft": 0xffeb, "MetaRight": 0xffec,
	"F1": 0xffbe, "F2": 0xffbf, "F3": 0xffc0, "F4": 0xffc1,
	"F5": 0xffc2, "F6": 0xffc3, "F7": 0xffc4, "F8": 0xffc5,
	"F9": 0xffc6, "F10": 0xffc7, "F11": 0xffc8, "F12": 0xffc9,
}

// key runs under the backend lock, and again under a server grab before input.
func (b *Backend) key(command sessions.DesktopCommand) (*desktopv1.KeyAction, byte, error) {
	var action desktopv1.Action
	if len(command.Payload) > 64*1024 || protojson.Unmarshal(command.Payload, &action) != nil {
		return nil, 0, ErrUnavailable
	}
	k := action.GetKey()
	if k == nil || k.Kind < desktopv1.KeyAction_KIND_DOWN || k.Kind > desktopv1.KeyAction_KIND_PRESS {
		return nil, 0, ErrUnavailable
	}
	symbol, ok := namedKeys[k.Key]
	// Lowercase Latin letters and digits identify their unshifted keysyms,
	// allowing explicit Control/Alt shortcuts without assuming key positions.
	if len(k.Key) == 1 && ((k.Key[0] >= 'a' && k.Key[0] <= 'z') || (k.Key[0] >= '0' && k.Key[0] <= '9')) {
		symbol, ok = xproto.Keysym(k.Key[0]), true
	}
	if !ok {
		return nil, 0, ErrUnavailable
	}
	_, _, revision, err := b.geometry()
	if err != nil || b.revision == "" || command.GeometryRevision != b.revision || revision != b.revision {
		return nil, 0, ErrUnavailable
	}
	// Release the original code even if the map changes while a key is held.
	if code, held := b.heldKeys[k.Key]; held {
		if k.Kind == desktopv1.KeyAction_KIND_PRESS {
			return nil, 0, ErrUnavailable
		}
		return k, code, nil
	}
	if k.Kind == desktopv1.KeyAction_KIND_UP {
		return nil, 0, ErrUnavailable
	}
	setup := xproto.Setup(b.conn)
	mapping, err := xproto.GetKeyboardMapping(b.conn, setup.MinKeycode, byte(int(setup.MaxKeycode)-int(setup.MinKeycode)+1)).Reply()
	if err != nil || mapping == nil || mapping.KeysymsPerKeycode == 0 {
		return nil, 0, ErrUnavailable
	}
	for i := 0; i < len(mapping.Keysyms); i += int(mapping.KeysymsPerKeycode) {
		if mapping.Keysyms[i] != symbol {
			continue
		}
		code := byte(int(setup.MinKeycode) + i/int(mapping.KeysymsPerKeycode))
		state, err := xproto.QueryKeymap(b.conn).Reply()
		if err != nil || state == nil || state.Keys[code/8]&(1<<(code%8)) != 0 {
			return nil, 0, ErrUnavailable
		}
		return k, code, nil
	}
	return nil, 0, ErrUnavailable
}

func (b *Backend) applyKey(command sessions.DesktopCommand) error {
	k, code, err := b.key(command)
	if err != nil {
		return err
	}
	if k.Kind != desktopv1.KeyAction_KIND_UP {
		if _, held := b.heldKeys[k.Key]; held {
			return nil
		}
		// Record before sending; a failed acknowledgement may follow an effect.
		b.heldKeys[k.Key] = code
		if err := b.event(xproto.KeyPress, code, 0, 0); err != nil {
			return err
		}
	}
	if k.Kind != desktopv1.KeyAction_KIND_DOWN {
		if err := b.event(xproto.KeyRelease, code, 0, 0); err != nil {
			return err
		}
		delete(b.heldKeys, k.Key)
	}
	return nil
}

// The HTTPS console uses a password-style soft keyboard field for printable
// characters. Use the server's actual keymap and its first two shift levels
// instead of assuming US keycodes or rewriting the X11 keymap.
func isExtendedPrintablePress(key *desktopv1.KeyAction) bool {
	if key.Kind != desktopv1.KeyAction_KIND_PRESS || len(key.Key) != 1 {
		return false
	}
	c := key.Key[0]
	return c >= ' ' && c <= '~' && !(c >= 'a' && c <= 'z') && !(c >= '0' && c <= '9')
}

func (b *Backend) printableCodes(command sessions.DesktopCommand, key string) (code, shift byte, err error) {
	_, _, revision, err := b.geometry()
	if err != nil || b.revision == "" || command.GeometryRevision != b.revision || revision != b.revision {
		return 0, 0, ErrUnavailable
	}
	pointer, err := xproto.QueryPointer(b.conn, b.root).Reply()
	if err != nil || pointer == nil || pointer.Mask&(xproto.ModMaskShift|xproto.ModMaskLock|xproto.ModMaskControl|xproto.ModMask1|xproto.ModMask3|xproto.ModMask4|xproto.ModMask5) != 0 {
		return 0, 0, ErrUnavailable
	}
	setup := xproto.Setup(b.conn)
	mapping, err := xproto.GetKeyboardMapping(b.conn, setup.MinKeycode, byte(int(setup.MaxKeycode)-int(setup.MinKeycode)+1)).Reply()
	if err != nil || mapping == nil || mapping.KeysymsPerKeycode < 2 {
		return 0, 0, ErrUnavailable
	}
	state, err := xproto.QueryKeymap(b.conn).Reply()
	if err != nil || state == nil {
		return 0, 0, ErrUnavailable
	}
	available := func(code byte) bool {
		return state.Keys[code/8]&(1<<(code%8)) == 0
	}
	var shifted byte
	for i := 0; i < len(mapping.Keysyms); i += int(mapping.KeysymsPerKeycode) {
		candidate := byte(int(setup.MinKeycode) + i/int(mapping.KeysymsPerKeycode))
		if !available(candidate) {
			continue
		}
		if mapping.Keysyms[i] == xproto.Keysym(key[0]) {
			return candidate, 0, nil
		}
		if mapping.Keysyms[i] == namedKeys["ShiftLeft"] {
			shift = candidate
		}
		if shifted == 0 && mapping.Keysyms[i+1] == xproto.Keysym(key[0]) {
			shifted = candidate
		}
	}
	if shifted == 0 || shift == 0 {
		return 0, 0, ErrUnavailable
	}
	return shifted, shift, nil
}

func (b *Backend) applyPrintable(command sessions.DesktopCommand, key string) (result error) {
	code, shift, err := b.printableCodes(command, key)
	if err != nil {
		return err
	}
	send := func(name string, code byte) error {
		b.heldKeys[name] = code
		return b.event(xproto.KeyPress, code, 0, 0)
	}
	release := func(name string, code byte) {
		err := b.event(xproto.KeyRelease, code, 0, 0)
		if err == nil {
			delete(b.heldKeys, name)
		}
		result = errors.Join(result, err)
	}
	if shift != 0 {
		defer release("soft-keyboard-shift", shift)
		if err := send("soft-keyboard-shift", shift); err != nil {
			return err
		}
	}
	defer release("soft-keyboard-character", code)
	return send("soft-keyboard-character", code)
}
