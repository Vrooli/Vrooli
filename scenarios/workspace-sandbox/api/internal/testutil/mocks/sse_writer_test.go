package mocks

import (
	"errors"
	"testing"

	"workspace-sandbox/internal/sse"
)

func TestFakeSSEWriter_RecordsFrames(t *testing.T) {
	f := NewFakeSSEWriter()
	if err := f.WriteData([]byte("hello")); err != nil {
		t.Fatalf("WriteData: %v", err)
	}
	if err := f.WriteEvent("exit", []byte("ok")); err != nil {
		t.Fatalf("WriteEvent: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	frames := f.Frames()
	if len(frames) != 3 {
		t.Fatalf("got %d frames, want 3", len(frames))
	}
	if frames[0].Event != "" || string(frames[0].Data) != "hello" {
		t.Errorf("frame[0] = %+v", frames[0])
	}
	if frames[1].Event != "exit" || string(frames[1].Data) != "ok" {
		t.Errorf("frame[1] = %+v", frames[1])
	}
	if frames[2].Event != "end" {
		t.Errorf("frame[2].Event = %q, want end", frames[2].Event)
	}
	for i, fr := range frames {
		if !fr.Flushed {
			t.Errorf("frame[%d] not marked flushed", i)
		}
	}
}

func TestFakeSSEWriter_CloseIdempotent(t *testing.T) {
	f := NewFakeSSEWriter()
	if err := f.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("Close again: %v", err)
	}
	if got := len(f.Frames()); got != 1 {
		t.Errorf("got %d frames, want 1", got)
	}
}

func TestFakeSSEWriter_WriteAfterCloseFails(t *testing.T) {
	f := NewFakeSSEWriter()
	if err := f.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := f.WriteData([]byte("x")); !errors.Is(err, sse.ErrAlreadyClosed) {
		t.Errorf("WriteData = %v, want ErrAlreadyClosed", err)
	}
	if err := f.WriteEvent("e", []byte("x")); !errors.Is(err, sse.ErrAlreadyClosed) {
		t.Errorf("WriteEvent = %v, want ErrAlreadyClosed", err)
	}
}

func TestFakeSSEWriter_ErrorOverrides(t *testing.T) {
	want := errors.New("network down")
	f := NewFakeSSEWriter()
	f.WriteDataErr = want
	if err := f.WriteData([]byte("x")); !errors.Is(err, want) {
		t.Errorf("WriteData = %v, want %v", err, want)
	}
	f.WriteDataErr = nil
	f.WriteEventErr = want
	if err := f.WriteEvent("e", []byte("x")); !errors.Is(err, want) {
		t.Errorf("WriteEvent = %v, want %v", err, want)
	}
	f.WriteEventErr = nil
	// Record a successful write so Frames() can prove the error path
	// did not erase prior history.
	if err := f.WriteData([]byte("ok")); err != nil {
		t.Fatalf("WriteData: %v", err)
	}
	f.FlushErr = want
	if err := f.Flush(); !errors.Is(err, want) {
		t.Errorf("Flush = %v, want %v", err, want)
	}
	if got := len(f.Frames()); got == 0 {
		t.Errorf("expected frames recorded before error injection, got 0")
	}
}

func TestFakeSSEWriter_DataIsCopied(t *testing.T) {
	f := NewFakeSSEWriter()
	src := []byte("mutable")
	if err := f.WriteData(src); err != nil {
		t.Fatalf("WriteData: %v", err)
	}
	src[0] = 'X'
	frames := f.Frames()
	if string(frames[0].Data) == "Xutable" {
		t.Errorf("FakeSSEWriter held caller's slice; mutation leaked")
	}
}
