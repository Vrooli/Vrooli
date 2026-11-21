package terminal

import (
	"strings"

	"web-console/session"
)

//go:generate node ../../../shared/generate-terminal-keymap.mjs

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
