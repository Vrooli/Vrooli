package session

import (
	"bytes"
	"errors"
	"testing"

	"web-console/internal/pty"
)

type testKeyMap struct{}

func (testKeyMap) Bytes(k Key) ([]byte, bool) {
	if k.Name == "Enter" {
		return []byte("\r"), true
	}
	return nil, false
}

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

func TestSessionInput_MetadataAndKeys(t *testing.T) {
	in := InputKeys(Key{Name: "Enter"}).WithSource("test").AsPaste().WithKind(pty.KindControl)
	got, err := in.resolveBytes(testKeyMap{})
	if err != nil || !bytes.Equal(got, []byte("\r")) {
		t.Fatalf("keys: got %q, err %v", got, err)
	}
	if in.ptyKind() != pty.KindControl {
		t.Fatalf("explicit kind was not preserved: %v", in.ptyKind())
	}
	if _, err := InputKeys(Key{Name: "NoSuchKey"}).resolveBytes(testKeyMap{}); err == nil {
		t.Fatal("unrecognized key unexpectedly resolved")
	} else if !errors.Is(err, ErrUnknownKey) {
		t.Fatalf("unrecognized key error = %v, want ErrUnknownKey", err)
	}
	if _, err := (SessionInput{}).resolveBytes(testKeyMap{}); err == nil {
		t.Fatal("empty input unexpectedly resolved")
	}
}
