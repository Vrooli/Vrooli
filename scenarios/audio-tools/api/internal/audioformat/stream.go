package audioformat

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
)

// streamReadChunk is the stdout read buffer for the streaming decoder.
// ffmpeg flushes packets promptly (-flush_packets 1), so read latency is
// driven by ffmpeg's flush cadence, not by filling this buffer.
const streamReadChunk = 4096

// Process is one running decode subprocess: write encoded bytes to its
// stdin, read canonical PCM from its stdout, signal end-of-input, and
// tear down.
//
// seam: Process/ProcessRunner is the audioformat streaming-decode seam
// (SEAMS.md row "audioformat.ProcessRunner"). Production wires
// execProcessRunner (a real ffmpeg child); tests wire
// internal/audioformat/mocks.FakeProcessRunner.
type Process interface {
	io.Writer          // feed encoded bytes to stdin
	io.Reader          // read decoded PCM from stdout
	CloseInput() error // close stdin → ffmpeg flushes its tail and exits
	Close() error      // kill + reap the process
}

// ProcessRunner starts a decode Process bound to ctx. The production impl
// uses exec.CommandContext so a ctx cancel kills the child.
type ProcessRunner interface {
	Start(ctx context.Context, name string, args []string) (Process, error)
}

// StreamDecoder converts a live stream of encoded chunks into canonical
// PCM frames. Feed encoded bytes with Write; consume PCM from Frames;
// call CloseInput at end-of-stream to flush; Close tears down. Exactly
// one decoder is created per session (NewStreamDecoder) so concurrent
// sessions never share a process.
type StreamDecoder interface {
	// Write feeds encoded bytes (or canonical PCM, in the fast-path).
	Write(p []byte) error
	// Frames yields canonical PCM, even-byte aligned (whole int16 samples).
	// It is closed after CloseInput drains the tail or the process dies.
	Frames() <-chan []byte
	// CloseInput signals end-of-input; the decoder flushes and closes Frames.
	CloseInput() error
	// Close tears down immediately (used on ctx cancel / abnormal end).
	Close() error
	// Err returns the terminal decode error, if any, after Frames closes.
	Err() error
}

// NewStreamDecoder builds a per-session decoder for the declared codec.
//   - CodecPCMS16LE → identity decoder: no ffmpeg, bytes flow straight
//     through (even-byte aligned). This is the browser PCM fast-path.
//   - any other codec → ffmpeg-backed decoder (one long-lived process).
//   - ffmpeg absent for a non-PCM codec → ErrFfmpegRequired (the selector
//     turns this into a BufferedFallback decision upstream).
func (e *Engine) NewStreamDecoder(ctx context.Context, codec Codec) (StreamDecoder, error) {
	if codec.IsCanonicalPCM() {
		return newIdentityDecoder(), nil
	}
	if codec == CodecUnknown {
		return nil, ErrUnknownFormat
	}
	if !e.hasFfmpeg() {
		return nil, ErrFfmpegRequired
	}
	args := []string{
		"-loglevel", "error",
		"-i", "pipe:0",
		"-f", "s16le",
		"-ar", fmt.Sprint(CanonicalSampleRate),
		"-ac", fmt.Sprint(CanonicalChannels),
		"-flush_packets", "1",
		"pipe:1",
	}
	proc, err := e.process.Start(ctx, "ffmpeg", args)
	if err != nil {
		return nil, fmt.Errorf("audio-tools/audioformat: start decode process: %w", err)
	}
	return newProcessDecoder(proc), nil
}

// NewStreamFilter builds a per-session PCM→PCM ffmpeg filter process: it reads
// canonical PCM (s16le / 16 kHz / mono, headerless) from stdin, applies the
// given ffmpeg -af filter chain, and emits canonical PCM. It reuses the same
// streaming wrapper as NewStreamDecoder (Write / Frames / CloseInput / Close),
// so callers drive it identically. Unlike NewStreamDecoder there is no PCM
// fast-path — the whole point is to transform the PCM.
//
// This is the substrate for the pre-recognition ingress enhancement stage
// (internal/stt/ingress): e.g. afftdn FFT denoise. ffmpeg absent →
// ErrFfmpegRequired so the caller can gate the stage off.
func (e *Engine) NewStreamFilter(ctx context.Context, filter string) (StreamDecoder, error) {
	if strings.TrimSpace(filter) == "" {
		return nil, fmt.Errorf("audio-tools/audioformat: NewStreamFilter requires a non-empty -af filter")
	}
	if !e.hasFfmpeg() {
		return nil, ErrFfmpegRequired
	}
	args := []string{
		"-loglevel", "error",
		"-f", "s16le",
		"-ar", fmt.Sprint(CanonicalSampleRate),
		"-ac", fmt.Sprint(CanonicalChannels),
		"-i", "pipe:0",
		"-af", filter,
		"-f", "s16le",
		"-ar", fmt.Sprint(CanonicalSampleRate),
		"-ac", fmt.Sprint(CanonicalChannels),
		"-flush_packets", "1",
		"pipe:1",
	}
	proc, err := e.process.Start(ctx, "ffmpeg", args)
	if err != nil {
		return nil, fmt.Errorf("audio-tools/audioformat: start filter process: %w", err)
	}
	return newProcessDecoder(proc), nil
}

// ErrDecoderClosed is returned by Write after the decoder was torn down
// via Close (abnormal teardown / ctx cancel).
var ErrDecoderClosed = errors.New("audio-tools/audioformat: decoder closed")

// ----- identity (PCM fast-path) decoder -----
//
// The identity decoder forwards bytes verbatim (the bytes are already
// canonical PCM). It is the browser PCM fast-path: no ffmpeg, no process.
//
// Lifecycle: CloseInput closes Frames (clean EOF); Close signals `done`
// so a Write blocked on backpressure unblocks with ErrDecoderClosed
// instead of deadlocking. Frames is only ever closed by CloseInput, and
// the single feeder calls Write then CloseInput sequentially, so there is
// no send-on-closed-channel race.
type identityDecoder struct {
	frames chan []byte
	done   chan struct{}
	inOnce sync.Once
	clOnce sync.Once
}

func newIdentityDecoder() *identityDecoder {
	return &identityDecoder{frames: make(chan []byte, 8), done: make(chan struct{})}
}

func (d *identityDecoder) Write(p []byte) error {
	if len(p) == 0 {
		return nil
	}
	buf := make([]byte, len(p))
	copy(buf, p)
	select {
	case d.frames <- buf:
		return nil
	case <-d.done:
		return ErrDecoderClosed
	}
}

func (d *identityDecoder) Frames() <-chan []byte { return d.frames }

func (d *identityDecoder) CloseInput() error {
	d.inOnce.Do(func() { close(d.frames) })
	return nil
}

func (d *identityDecoder) Close() error {
	d.clOnce.Do(func() { close(d.done) })
	return nil
}

func (d *identityDecoder) Err() error { return nil }

// ----- ffmpeg-backed decoder -----

type processDecoder struct {
	proc   Process
	frames chan []byte

	mu  sync.Mutex
	err error

	closeOnce sync.Once
}

func newProcessDecoder(proc Process) *processDecoder {
	d := &processDecoder{
		proc:   proc,
		frames: make(chan []byte, 8),
	}
	go d.readLoop()
	return d
}

// readLoop reads decoded PCM from the process stdout, aligns to whole
// int16 samples (carrying an odd trailing byte across reads), and forwards
// frames. It closes Frames when stdout reaches EOF or errors.
func (d *processDecoder) readLoop() {
	defer close(d.frames)
	var carry []byte
	buf := make([]byte, streamReadChunk)
	for {
		n, err := d.proc.Read(buf)
		if n > 0 {
			chunk := append(carry, buf[:n]...)
			if len(chunk)%CanonicalBytesPerSample != 0 {
				// Hold the trailing odd byte until the next read completes it.
				carry = []byte{chunk[len(chunk)-1]}
				chunk = chunk[:len(chunk)-1]
			} else {
				carry = nil
			}
			if len(chunk) > 0 {
				out := make([]byte, len(chunk))
				copy(out, chunk)
				d.frames <- out
			}
		}
		if err != nil {
			if err != io.EOF {
				d.setErr(err)
			}
			return
		}
	}
}

func (d *processDecoder) Write(p []byte) error {
	if len(p) == 0 {
		return nil
	}
	if _, err := d.proc.Write(p); err != nil {
		d.setErr(err)
		return err
	}
	return nil
}

func (d *processDecoder) Frames() <-chan []byte { return d.frames }

func (d *processDecoder) CloseInput() error {
	// Closing stdin makes ffmpeg flush its remaining output and exit; the
	// readLoop then sees EOF and closes Frames.
	return d.proc.CloseInput()
}

func (d *processDecoder) Close() error {
	var err error
	d.closeOnce.Do(func() {
		err = d.proc.Close()
	})
	return err
}

func (d *processDecoder) Err() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.err
}

func (d *processDecoder) setErr(err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.err == nil {
		d.err = err
	}
}

// ----- production exec process -----

type execProcessRunner struct{}

func (execProcessRunner) Start(ctx context.Context, name string, args []string) (Process, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &execProcess{cmd: cmd, stdin: stdin, stdout: stdout}, nil
}

type execProcess struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
}

func (p *execProcess) Write(b []byte) (int, error) { return p.stdin.Write(b) }
func (p *execProcess) Read(b []byte) (int, error)  { return p.stdout.Read(b) }
func (p *execProcess) CloseInput() error           { return p.stdin.Close() }

func (p *execProcess) Close() error {
	_ = p.stdin.Close()
	if p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
	}
	return p.cmd.Wait()
}
