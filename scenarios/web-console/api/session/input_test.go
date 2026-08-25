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
