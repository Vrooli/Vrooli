package segmenter

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/audioformat"
)

// TestCaptureTimelineRejectedRangeIsObservable is the regression for the
// truncated-tail-as-success failure: AudioTimeline rejects a non-contiguous
// chunk, but captureTimeline used to swallow the error and just close the
// chunk channel. A downstream strategy then saw a clean EOF and emitted a
// normal Done, so the caller could not tell a complete turn from a truncated
// one. The failure channel makes that outcome explicit.
func TestCaptureTimelineRejectedRangeIsObservable(t *testing.T) {
	timeline := NewAudioTimeline(0)
	in := make(chan sttchain.AudioChunk, 2)
	in <- sttchain.AudioChunk{StartSample: 0, EndSample: 2, Audio: []byte{1, 0, 2, 0}}
	// Starts at 3 instead of the previous end (2): a rejected range.
	in <- sttchain.AudioChunk{StartSample: 3, EndSample: 5, Audio: []byte{3, 0, 4, 0}}
	close(in)

	out, failures := captureTimeline(context.Background(), timeline, in)
	var forwarded []sttchain.AudioChunk
	for chunk := range out {
		forwarded = append(forwarded, chunk)
	}

	require.Len(t, forwarded, 1, "the chunk that broke contiguity must not be forwarded")
	err := <-failures
	require.Error(t, err)
	require.Contains(t, err.Error(), "non-contiguous")
}

// TestRunEgressSurfacesCaptureFailureBeforeDone proves the terminal ordering:
// the strategy still emits its ordinary Done after its input closes, so the
// capture error must be inserted immediately before that Done. Otherwise a
// transport would see a clean terminal event and never learn the tail was
// truncated.
func TestRunEgressSurfacesCaptureFailureBeforeDone(t *testing.T) {
	captureErr := errors.New("audio-tools/stt/segmenter: capture canonical audio timeline: non-contiguous chunk range")
	failures := make(chan error, 1)
	failures <- captureErr
	close(failures)

	inner := make(chan sttchain.StreamEvent, 1)
	inner <- sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{FinalText: "truncated"}}
	close(inner)

	out := make(chan sttchain.StreamEvent, 4)
	done := make(chan error, 1)
	runEgress(context.Background(), nil, nil, inner, out, failures, done)
	close(out)

	got := drain(out)
	require.Len(t, got, 2)
	require.Equal(t, sttchain.StreamEventError, got[0].Kind)
	require.ErrorIs(t, got[0].Error, captureErr)
	require.Equal(t, sttchain.StreamEventDone, got[1].Kind)
	require.Equal(t, "truncated", got[1].Done.FinalText)
	require.ErrorIs(t, <-done, captureErr)
}

// deterministicProcess is a decode process whose CloseInput can be held open.
// It lets the teardown test observe whether cleanup joined the feeder goroutine
// instead of relying on a timing-sensitive goroutine count.
type deterministicProcess struct {
	entered     chan struct{}
	release     chan struct{}
	closed      chan struct{}
	enteredOnce sync.Once
	closedOnce  sync.Once
}

func newDeterministicProcess() *deterministicProcess {
	return &deterministicProcess{
		entered: make(chan struct{}),
		release: make(chan struct{}),
		closed:  make(chan struct{}),
	}
}

func (p *deterministicProcess) Write(b []byte) (int, error) { return len(b), nil }

// Read blocks until teardown closes the process, mirroring a decoder waiting on
// a live ffmpeg stdout.
func (p *deterministicProcess) Read([]byte) (int, error) {
	<-p.closed
	return 0, io.EOF
}

// CloseInput signals that the feeder reached the cancellation flush, then
// blocks until the test releases it.
func (p *deterministicProcess) CloseInput() error {
	p.enteredOnce.Do(func() { close(p.entered) })
	<-p.release
	return nil
}

func (p *deterministicProcess) Close() error {
	p.closedOnce.Do(func() { close(p.closed) })
	return nil
}

type deterministicRunner struct{ process audioformat.Process }

func (r deterministicRunner) Start(context.Context, string, []string) (audioformat.Process, error) {
	return r.process, nil
}

// TestNormalizeChunksCleanupWaitsForWorkers pins deterministic teardown.
// cleanup used to cancel and close the decoder and return immediately, so the
// feeder could still be inside the decoder's flush (or the adapter could still
// be forwarding) after teardown reported success. The test parks CloseInput,
// asserts cleanup stays blocked, then releases it.
func TestNormalizeChunksCleanupWaitsForWorkers(t *testing.T) {
	proc := newDeterministicProcess()
	engine := audioformat.New(
		audioformat.WithProcessRunner(deterministicRunner{process: proc}),
		audioformat.WithFfmpegProbe(func() bool { return true }),
	)
	seg := New(Deps{Engine: engine})

	chunks := make(chan sttchain.AudioChunk) // open but idle: the feeder waits for input
	events := make(chan sttchain.StreamEvent, 1)
	_, _, cleanup, err := seg.normalizeChunks(context.Background(), sttchain.StreamStart{InputFormat: "webm"}, chunks, events)
	require.NoError(t, err)

	cleaned := make(chan struct{})
	go func() {
		cleanup()
		close(cleaned)
	}()

	select {
	case <-proc.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("feeder never reached CloseInput during cancellation")
	}

	// The feeder is provably still running: cleanup must not have returned.
	select {
	case <-cleaned:
		close(proc.release)
		t.Fatal("cleanup returned before the feeder goroutine exited")
	case <-time.After(50 * time.Millisecond):
	}

	close(proc.release)
	select {
	case <-cleaned:
	case <-time.After(2 * time.Second):
		t.Fatal("cleanup did not return after the feeder exited")
	}
}
