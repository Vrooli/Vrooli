package terminal

import (
	"bytes"
	"testing"

	"web-console/session"
)

func TestDefaultKeyMap_NamedKeys(t *testing.T) {
	km := DefaultKeyMap{}
	cases := []struct {
		key  session.Key
		want []byte
	}{
		{session.Key{Name: "Enter"}, []byte{0x0d}},
		{session.Key{Name: "tab"}, []byte{0x09}},
		{session.Key{Name: "BACKSPACE"}, []byte{0x7f}},
		{session.Key{Name: "Up"}, []byte("\x1b[A")},
		{session.Key{Name: "F5"}, []byte("\x1b[15~")},
	}
	for _, c := range cases {
		got, ok := km.Bytes(c.key)
		if !ok || !bytes.Equal(got, c.want) {
			t.Errorf("Bytes(%q): got %v ok=%v, want %v", c.key.Name, got, ok, c.want)
		}
	}
}

func TestDefaultKeyMap_Modifiers(t *testing.T) {
	km := DefaultKeyMap{}
	for letter, want := range map[string]byte{"c": 0x03, "z": 0x1a, "a": 0x01, "d": 0x04} {
		got, ok := km.Bytes(session.Key{Name: letter, Ctrl: true})
		if !ok || len(got) != 1 || got[0] != want {
			t.Errorf("Ctrl+%s: got %v ok=%v, want %v", letter, got, ok, []byte{want})
		}
	}
	got, ok := km.Bytes(session.Key{Name: "Enter", Alt: true})
	if !ok || !bytes.Equal(got, []byte{0x1b, 0x0d}) {
		t.Errorf("Alt+Enter: got %v ok=%v", got, ok)
	}
}
