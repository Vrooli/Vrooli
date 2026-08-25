package terminal

import (
	"strings"

	"web-console/session"
)

// DefaultKeyMap serves the Connect RPC's KeySequence variant for
// programmatic callers. The WebSocket path never reaches this map: the
// browser deliberately encodes keys because the server emulator does not
// model application-cursor mode.
type DefaultKeyMap struct{}

// Bytes implements session.KeyMap. Lookup is case-insensitive on Name.
func (DefaultKeyMap) Bytes(k session.Key) ([]byte, bool) {
	name := strings.ToLower(strings.TrimSpace(k.Name))
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

var defaultKeyBytes = map[string][]byte{
	"enter": {0x0d}, "return": {0x0d}, "tab": {0x09}, "backspace": {0x7f},
	"delete": []byte("\x1b[3~"), "escape": {0x1b}, "esc": {0x1b}, "space": {0x20},
	"up": []byte("\x1b[A"), "down": []byte("\x1b[B"), "right": []byte("\x1b[C"), "left": []byte("\x1b[D"),
	"home": []byte("\x1b[H"), "end": []byte("\x1b[F"), "pageup": []byte("\x1b[5~"), "pagedown": []byte("\x1b[6~"),
	"insert": []byte("\x1b[2~"), "f1": []byte("\x1bOP"), "f2": []byte("\x1bOQ"), "f3": []byte("\x1bOR"), "f4": []byte("\x1bOS"),
	"f5": []byte("\x1b[15~"), "f6": []byte("\x1b[17~"), "f7": []byte("\x1b[18~"), "f8": []byte("\x1b[19~"),
	"f9": []byte("\x1b[20~"), "f10": []byte("\x1b[21~"), "f11": []byte("\x1b[23~"), "f12": []byte("\x1b[24~"),
}
