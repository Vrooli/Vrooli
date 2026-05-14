package session

import (
	"fmt"
	"strings"

	"web-console/internal/pty"
)

// inputVariant tags a SessionInput's payload kind.
type inputVariant uint8

const (
	variantUnset inputVariant = iota
	variantText
	variantKeys
	variantRaw
)

// Key is one named keypress with optional modifiers. The session's
// active KeyMap resolves Key → bytes; bytes are then routed through the
// same applyInput path as text and raw input.
type Key struct {
	Name  string
	Ctrl  bool
	Alt   bool
	Shift bool
}

// InputMeta carries non-payload metadata about a SessionInput.
type InputMeta struct {
	// Source tags the origin for diagnostics (e.g. "ws", "cli",
	// "agent:claude-driver", "ansi-responder"). Free-form; never
	// affects routing.
	Source string
	// IsPaste hints that the bytes should be delivered via the PTY's
	// bracketed-paste path when the underlying backend supports it.
	// Maps to pty.KindPaste; otherwise pty.KindKeystroke.
	IsPaste bool
}

// SessionInput is the typed envelope for client-origin input. Construct
// via InputText / InputKeys / InputRaw. Once built, the value is opaque
// to callers — the session owns the conversion to bytes.
type SessionInput struct {
	kind inputVariant
	text string
	keys []Key
	raw  []byte
	meta InputMeta
}

// InputText builds a text-payload SessionInput. The string is delivered
// as UTF-8 bytes; embedded LF characters are forwarded as-is.
func InputText(s string) SessionInput {
	return SessionInput{kind: variantText, text: s}
}

// InputKeys builds a keys-payload SessionInput. The session's active
// KeyMap resolves each Key to bytes at delivery time.
func InputKeys(keys ...Key) SessionInput {
	cp := make([]Key, len(keys))
	copy(cp, keys)
	return SessionInput{kind: variantKeys, keys: cp}
}

// InputRaw builds a raw-bytes SessionInput. The bytes are delivered
// verbatim. Reserved for tooling that needs byte-level control.
func InputRaw(b []byte) SessionInput {
	cp := make([]byte, len(b))
	copy(cp, b)
	return SessionInput{kind: variantRaw, raw: cp}
}

// WithMeta returns a copy of in with InputMeta replaced.
func (in SessionInput) WithMeta(meta InputMeta) SessionInput {
	in.meta = meta
	return in
}

// WithSource returns a copy of in with Source set on its InputMeta.
func (in SessionInput) WithSource(source string) SessionInput {
	in.meta.Source = source
	return in
}

// AsPaste returns a copy of in tagged for bracketed-paste delivery.
func (in SessionInput) AsPaste() SessionInput {
	in.meta.IsPaste = true
	return in
}

// resolveBytes converts a SessionInput to its on-wire byte form using
// the supplied KeyMap (only consulted for variantKeys). Returns an
// error for unrecognized key names.
func (in SessionInput) resolveBytes(km KeyMap) ([]byte, error) {
	switch in.kind {
	case variantText:
		return []byte(in.text), nil
	case variantRaw:
		return in.raw, nil
	case variantKeys:
		if km == nil {
			km = DefaultKeyMap{}
		}
		var buf []byte
		for _, k := range in.keys {
			b, ok := km.Bytes(k)
			if !ok {
				return nil, fmt.Errorf("unknown key %q", keyDisplay(k))
			}
			buf = append(buf, b...)
		}
		return buf, nil
	default:
		return nil, fmt.Errorf("input has no payload")
	}
}

// ptyKind returns the pty.InputKind appropriate for this input.
func (in SessionInput) ptyKind() pty.InputKind {
	if in.meta.IsPaste {
		return pty.KindPaste
	}
	return pty.KindKeystroke
}

// KeyMap resolves a Key to the bytes that should be written to the PTY.
// The default map (DefaultKeyMap) handles the standard xterm-ish set;
// backends may register their own via BackendDescriptor.KeyMap (added
// in Phase 4) to override or extend.
type KeyMap interface {
	Bytes(k Key) ([]byte, bool)
}

// DefaultKeyMap is the baseline mapping: standard ASCII control keys,
// xterm cursor/function keys, and Ctrl+<letter> shorthand. Sufficient
// for most shells and TUIs; backends with non-standard expectations
// (e.g. application-mode arrow keys) can layer on top.
type DefaultKeyMap struct{}

// Bytes implements KeyMap. Lookup is case-insensitive on Name. Returns
// (nil, false) for unknown keys so callers can surface a clear error.
func (DefaultKeyMap) Bytes(k Key) ([]byte, bool) {
	name := strings.ToLower(strings.TrimSpace(k.Name))
	// Ctrl+<letter> is a special case: any unmodified one-character
	// name combined with Ctrl maps to the corresponding control byte.
	if k.Ctrl && !k.Alt && !k.Shift && len(name) == 1 {
		c := name[0]
		switch {
		case c >= 'a' && c <= 'z':
			return []byte{c - 'a' + 1}, true
		case c == '@':
			return []byte{0x00}, true
		case c == '[':
			return []byte{0x1b}, true
		case c == '\\':
			return []byte{0x1c}, true
		case c == ']':
			return []byte{0x1d}, true
		case c == '^':
			return []byte{0x1e}, true
		case c == '_':
			return []byte{0x1f}, true
		case c == ' ':
			return []byte{0x00}, true
		}
	}
	b, ok := defaultKeyBytes[name]
	if !ok {
		return nil, false
	}
	if k.Alt {
		out := make([]byte, 0, 1+len(b))
		out = append(out, 0x1b)
		out = append(out, b...)
		return out, true
	}
	return b, true
}

// defaultKeyBytes is the canonical name → bytes table for the default
// KeyMap. Names are lower-cased for lookup; callers may send any case.
//
// Function-key encodings use the xterm "VT220" set (CSI 11~ … CSI 24~)
// because that matches what most modern terminal emulators emit and
// what GNU readline / ncurses are wired to recognize.
var defaultKeyBytes = map[string][]byte{
	"enter":     {0x0d},
	"return":    {0x0d},
	"tab":       {0x09},
	"backspace": {0x7f},
	"delete":    []byte("\x1b[3~"),
	"escape":    {0x1b},
	"esc":       {0x1b},
	"space":     {0x20},
	"up":        []byte("\x1b[A"),
	"down":      []byte("\x1b[B"),
	"right":     []byte("\x1b[C"),
	"left":      []byte("\x1b[D"),
	"home":      []byte("\x1b[H"),
	"end":       []byte("\x1b[F"),
	"pageup":    []byte("\x1b[5~"),
	"pagedown":  []byte("\x1b[6~"),
	"insert":    []byte("\x1b[2~"),
	"f1":        []byte("\x1bOP"),
	"f2":        []byte("\x1bOQ"),
	"f3":        []byte("\x1bOR"),
	"f4":        []byte("\x1bOS"),
	"f5":        []byte("\x1b[15~"),
	"f6":        []byte("\x1b[17~"),
	"f7":        []byte("\x1b[18~"),
	"f8":        []byte("\x1b[19~"),
	"f9":        []byte("\x1b[20~"),
	"f10":       []byte("\x1b[21~"),
	"f11":       []byte("\x1b[23~"),
	"f12":       []byte("\x1b[24~"),
}

func keyDisplay(k Key) string {
	var parts []string
	if k.Ctrl {
		parts = append(parts, "Ctrl")
	}
	if k.Alt {
		parts = append(parts, "Alt")
	}
	if k.Shift {
		parts = append(parts, "Shift")
	}
	parts = append(parts, k.Name)
	return strings.Join(parts, "+")
}
