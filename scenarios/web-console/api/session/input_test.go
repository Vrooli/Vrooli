package session

import (
	"bytes"
	"testing"
)

func TestSessionInput_TextResolveBytes(t *testing.T) {
	in := InputText("hello\n")
	got, err := in.resolveBytes(nil)
	if err != nil {
		t.Fatalf("resolveBytes: %v", err)
	}
	if !bytes.Equal(got, []byte("hello\n")) {
		t.Errorf("text: got %q, want %q", got, "hello\n")
	}
}

func TestSessionInput_RawResolveBytes(t *testing.T) {
	in := InputRaw([]byte{0x1b, 0x5b, 0x41})
	got, err := in.resolveBytes(nil)
	if err != nil {
		t.Fatalf("resolveBytes: %v", err)
	}
	if !bytes.Equal(got, []byte{0x1b, 0x5b, 0x41}) {
		t.Errorf("raw: got %v, want ESC [ A", got)
	}
}

func TestDefaultKeyMap_NamedKeys(t *testing.T) {
	km := DefaultKeyMap{}
	cases := []struct {
		key  Key
		want []byte
	}{
		{Key{Name: "Enter"}, []byte{0x0d}},
		{Key{Name: "tab"}, []byte{0x09}},
		{Key{Name: "BACKSPACE"}, []byte{0x7f}},
		{Key{Name: "Up"}, []byte("\x1b[A")},
		{Key{Name: "F5"}, []byte("\x1b[15~")},
	}
	for _, c := range cases {
		got, ok := km.Bytes(c.key)
		if !ok {
			t.Errorf("Bytes(%q): not found", c.key.Name)
			continue
		}
		if !bytes.Equal(got, c.want) {
			t.Errorf("Bytes(%q): got %v, want %v", c.key.Name, got, c.want)
		}
	}
}

func TestDefaultKeyMap_CtrlLetters(t *testing.T) {
	km := DefaultKeyMap{}
	for letter, want := range map[string]byte{"c": 0x03, "z": 0x1a, "a": 0x01, "d": 0x04} {
		got, ok := km.Bytes(Key{Name: letter, Ctrl: true})
		if !ok || len(got) != 1 || got[0] != want {
			t.Errorf("Ctrl+%s: got %v ok=%v, want %v", letter, got, ok, []byte{want})
		}
	}
}

func TestDefaultKeyMap_AltPrefix(t *testing.T) {
	km := DefaultKeyMap{}
	got, ok := km.Bytes(Key{Name: "Enter", Alt: true})
	if !ok {
		t.Fatal("Alt+Enter: not found")
	}
	if !bytes.Equal(got, []byte{0x1b, 0x0d}) {
		t.Errorf("Alt+Enter: got %v, want ESC + CR", got)
	}
}

func TestSessionInput_KeysResolveBytes(t *testing.T) {
	in := InputKeys(Key{Name: "Up"}, Key{Name: "Enter"})
	got, err := in.resolveBytes(nil)
	if err != nil {
		t.Fatalf("resolveBytes: %v", err)
	}
	want := append([]byte("\x1b[A"), 0x0d)
	if !bytes.Equal(got, want) {
		t.Errorf("keys: got %v, want %v", got, want)
	}
}

func TestSessionInput_UnknownKey(t *testing.T) {
	in := InputKeys(Key{Name: "DoesNotExist"})
	_, err := in.resolveBytes(nil)
	if err == nil {
		t.Fatal("expected error for unknown key")
	}
}
