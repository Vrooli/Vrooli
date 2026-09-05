package session

import (
	"errors"
	"fmt"
	"strings"

	"web-console/internal/pty"
)

// ErrUnknownKey is stable across the session/handler package boundary.
var ErrUnknownKey = errors.New("unknown key")

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
	// Kind, when KindSet is true, is the explicit transport intent. It is
	// used for synthetic control bytes that do not belong to stdin's reliable
	// text lane.
	Kind    pty.InputKind
	KindSet bool
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

// WithSource returns a copy of in with Source set on its InputMeta.
func (in SessionInput) WithSource(source string) SessionInput {
	in.meta.Source = source
	return in
}

// WithKind returns a copy of in with an explicit PTY transport kind.
func (in SessionInput) WithKind(kind pty.InputKind) SessionInput {
	in.meta.Kind = kind
	in.meta.KindSet = true
	return in
}

// AsPaste returns a copy of in tagged for bracketed-paste delivery.
func (in SessionInput) AsPaste() SessionInput {
	in.meta.IsPaste = true
	in.meta.Kind = pty.KindPaste
	in.meta.KindSet = true
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
			return nil, fmt.Errorf("key map unavailable")
		}
		var buf []byte
		for _, k := range in.keys {
			b, ok := km.Bytes(k)
			if !ok {
				return nil, fmt.Errorf("%w %q", ErrUnknownKey, keyDisplay(k))
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
	if in.meta.KindSet {
		return in.meta.Kind
	}
	if in.meta.IsPaste {
		return pty.KindPaste
	}
	return pty.KindKeystroke
}

// KeyMap resolves a Key to the bytes that should be written to the PTY.
// The Connect RPC owns the default programmatic map; the session package
// keeps this interface so a backend can inject a different map when needed.
type KeyMap interface {
	Bytes(k Key) ([]byte, bool)
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
