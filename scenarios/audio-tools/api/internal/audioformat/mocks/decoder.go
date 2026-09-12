package mocks

import (
	"sync"

	"audio-tools/internal/audioformat"
)

// FakeStreamDecoder is a canned audioformat.StreamDecoder. Bytes written
// are forwarded verbatim to Frames (identity), letting segmenter-level
// tests drive the normalization wrap without a real decode process.
type FakeStreamDecoder struct {
	frames   chan []byte
	once     sync.Once
	WriteErr error // returned by Write when set
	CloseErr error // returned by Close when set
	terminal error
}

// NewFakeStreamDecoder constructs a decoder with a buffered frame channel.
func NewFakeStreamDecoder() *FakeStreamDecoder {
	return &FakeStreamDecoder{frames: make(chan []byte, 16)}
}

func (d *FakeStreamDecoder) Write(p []byte) error {
	if d.WriteErr != nil {
		return d.WriteErr
	}
	if len(p) == 0 {
		return nil
	}
	d.frames <- append([]byte(nil), p...)
	return nil
}

func (d *FakeStreamDecoder) Frames() <-chan []byte { return d.frames }

func (d *FakeStreamDecoder) CloseInput() error {
	d.once.Do(func() { close(d.frames) })
	return nil
}

func (d *FakeStreamDecoder) Close() error {
	d.once.Do(func() { close(d.frames) })
	return d.CloseErr
}

func (d *FakeStreamDecoder) Err() error { return d.terminal }

// SetErr seeds the terminal error reported by Err.
func (d *FakeStreamDecoder) SetErr(err error) { d.terminal = err }

var _ audioformat.StreamDecoder = (*FakeStreamDecoder)(nil)
