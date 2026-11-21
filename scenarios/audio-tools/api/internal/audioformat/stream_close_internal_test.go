package audioformat

import (
	"errors"
	"io"
	"sync"
	"testing"
	"time"
)

// blockingProcess returns whole int16 frames forever and signals when it has
// produced enough reads to fill the decoder's frame buffer. It never closes
// stdout on its own, so the only way the decoder can stop is the close signal.
type blockingProcess struct {
	reads     int
	saturated chan struct{}
	closed    chan struct{}
	closeOnce sync.Once
}

func newBlockingProcess() *blockingProcess {
	return &blockingProcess{
		saturated: make(chan struct{}),
		closed:    make(chan struct{}),
	}
}

func (p *blockingProcess) Read(b []byte) (int, error) {
	p.reads++
	if p.reads == 9 {
		// Eight frames fit the buffer; the ninth send is the one that parks.
		close(p.saturated)
	}
	select {
	case <-p.closed:
		return 0, io.EOF
	default:
	}
	n := len(b) - len(b)%CanonicalBytesPerSample
	return n, nil
}

func (p *blockingProcess) Write(b []byte) (int, error) { return len(b), nil }
func (p *blockingProcess) CloseInput() error           { return nil }
func (p *blockingProcess) Close() error {
	p.closeOnce.Do(func() { close(p.closed) })
	return nil
}

// TestProcessDecoderCloseUnblocksFullFrameBuffer pins the teardown contract:
// a consumer that stops reading (cancel, error, early return) must not leave
// the reader goroutine parked on a full channel. Before the done channel,
// Close killed the child but Frames never closed.
func TestProcessDecoderCloseUnblocksFullFrameBuffer(t *testing.T) {
	proc := newBlockingProcess()
	d := newProcessDecoder(proc)

	select {
	case <-proc.saturated:
	case <-time.After(2 * time.Second):
		t.Fatal("decoder never filled its frame buffer")
	}

	closeDone := make(chan error, 1)
	go func() { closeDone <- d.Close() }()

	select {
	case err := <-closeDone:
		if err != nil {
			t.Fatalf("Close: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Close blocked")
	}

	select {
	case _, open := <-d.Frames():
		// Buffered frames may still be drained; keep draining until closed.
		if open {
			for range d.Frames() {
			}
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Frames did not close after Close; reader goroutine is still parked")
	}

	select {
	case _, open := <-d.Frames():
		if open {
			t.Fatal("Frames produced a frame after it closed")
		}
	case <-time.After(time.Second):
		t.Fatal("Frames never closed")
	}
	if err := d.Err(); err != nil && !errors.Is(err, io.EOF) {
		t.Fatalf("unexpected terminal error: %v", err)
	}
}
