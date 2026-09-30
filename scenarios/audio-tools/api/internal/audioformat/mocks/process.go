// Package mocks holds the hoisted test fakes for the audioformat
// substrate: a fake decode subprocess (FakeProcessRunner / FakeProcess)
// and a fake StreamDecoder. They let stream/segmenter tests exercise the
// streaming-decode path without spawning a real ffmpeg child.
package mocks

import (
	"context"
	"io"
	"sync"

	"audio-tools/internal/audioformat"
)

// StartCall records one ProcessRunner.Start invocation for argv assertion.
type StartCall struct {
	Name string
	Args []string
}

// FakeProcessRunner is the canonical audioformat.ProcessRunner fake. It
// records Start calls and returns a FakeProcess whose stdout is the
// stdin bytes passed through Transform (identity by default — encoded
// bytes echo back as "PCM", enough to exercise frame alignment).
type FakeProcessRunner struct {
	Calls []StartCall
	// Transform maps one written chunk to the bytes the decoder reads.
	// Nil means identity. Applied per Write so tests can produce odd-length
	// chunks to exercise partial-frame buffering.
	Transform func([]byte) []byte
	// StartErr, when set, makes Start fail.
	StartErr error
}

// Start records the call and returns a FakeProcess (or StartErr).
func (f *FakeProcessRunner) Start(ctx context.Context, name string, args []string) (audioformat.Process, error) {
	f.Calls = append(f.Calls, StartCall{Name: name, Args: append([]string(nil), args...)})
	if f.StartErr != nil {
		return nil, f.StartErr
	}
	p := newFakeProcess(f.Transform)
	// Honor ctx cancellation: closing the pipes with an error makes the
	// decoder's read loop observe a non-EOF failure, mirroring a killed child.
	go func() {
		<-ctx.Done()
		p.failClose(ctx.Err())
	}()
	return p, nil
}

var _ audioformat.ProcessRunner = (*FakeProcessRunner)(nil)

// FakeProcess is an in-memory Process: bytes written to stdin are passed
// through Transform and become readable on stdout.
type FakeProcess struct {
	inR  *io.PipeReader
	inW  *io.PipeWriter
	outR *io.PipeReader
	outW *io.PipeWriter

	transform func([]byte) []byte
	closeOnce sync.Once
}

func newFakeProcess(transform func([]byte) []byte) *FakeProcess {
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	p := &FakeProcess{inR: inR, inW: inW, outR: outR, outW: outW, transform: transform}
	go p.pump()
	return p
}

// pump copies each stdin chunk through transform onto stdout. When stdin
// closes (CloseInput), it closes stdout so the reader sees EOF.
func (p *FakeProcess) pump() {
	buf := make([]byte, 4096)
	for {
		n, err := p.inR.Read(buf)
		if n > 0 {
			chunk := append([]byte(nil), buf[:n]...)
			if p.transform != nil {
				chunk = p.transform(chunk)
			}
			if _, werr := p.outW.Write(chunk); werr != nil {
				return
			}
		}
		if err != nil {
			_ = p.outW.CloseWithError(err) // io.EOF on clean CloseInput
			return
		}
	}
}

func (p *FakeProcess) Write(b []byte) (int, error) { return p.inW.Write(b) }
func (p *FakeProcess) Read(b []byte) (int, error)  { return p.outR.Read(b) }
func (p *FakeProcess) CloseInput() error           { return p.inW.Close() }

func (p *FakeProcess) Close() error {
	p.closeOnce.Do(func() {
		_ = p.inW.Close()
		_ = p.outW.Close()
	})
	return nil
}

// failClose tears down the pipes with err so an in-flight Read returns it
// (used to simulate a ctx-cancelled / killed process).
func (p *FakeProcess) failClose(err error) {
	_ = p.inW.CloseWithError(err)
	_ = p.outW.CloseWithError(err)
}

var _ audioformat.Process = (*FakeProcess)(nil)
