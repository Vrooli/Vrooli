package main

import (
	"errors"
	"strings"
	"testing"
)

type admissionTestEngine struct {
	streams int
	fail    bool
}

func (e *admissionTestEngine) NewStream() (STTStream, error) {
	if e.fail {
		return nil, errors.New("decoder unavailable")
	}
	e.streams++
	return &admissionTestStream{}, nil
}
func (*admissionTestEngine) Close() {}

type admissionTestStream struct{ closes int }

func (*admissionTestStream) AcceptPCM([]byte) ([]STTEvent, error) { return nil, nil }
func (*admissionTestStream) Finish() ([]STTEvent, error)          { return nil, nil }
func (s *admissionTestStream) Close()                             { s.closes++ }

func TestLimitedStreamingRejectsExcessAndReleasesExactlyOnce(t *testing.T) {
	inner := &admissionTestEngine{}
	engine := limitStreaming(inner, 1)
	first, err := engine.NewStream()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := engine.NewStream(); err == nil || !strings.Contains(err.Error(), "capacity reached") {
		t.Fatalf("second stream error = %v, want capacity rejection", err)
	}
	first.Close()
	first.Close()
	second, err := engine.NewStream()
	if err != nil {
		t.Fatalf("capacity was not released after close: %v", err)
	}
	second.Close()
	if inner.streams != 2 {
		t.Fatalf("underlying streams = %d, want 2", inner.streams)
	}
}

func TestLimitedStreamingReleasesCapacityWhenCreationFails(t *testing.T) {
	inner := &admissionTestEngine{fail: true}
	engine := limitStreaming(inner, 1)
	if _, err := engine.NewStream(); err == nil {
		t.Fatal("expected underlying creation failure")
	}
	inner.fail = false
	stream, err := engine.NewStream()
	if err != nil {
		t.Fatalf("capacity leaked after creation failure: %v", err)
	}
	stream.Close()
}
