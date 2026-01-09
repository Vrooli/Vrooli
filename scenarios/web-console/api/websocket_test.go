package main

import (
	"bufio"
	"encoding/binary"
	"io"
	"os"
	"testing"
)

func TestHandleBinaryMessageWritesToPTY(t *testing.T) {
	readEnd, writeEnd, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe creation failed: %v", err)
	}
	t.Cleanup(func() {
		_ = readEnd.Close()
		_ = writeEnd.Close()
	})

	sess := &session{
		ptyFile:      writeEnd,
		lastInputSeq: make(map[string]uint64),
		metrics:      newMetricsRegistry(),
		transcript: &transcriptManager{
			writer: bufio.NewWriter(io.Discard),
		},
	}

	client := newWSClient(sess, nil)

	payload := []byte("hello world")
	frame := buildBinaryInputFrame(42, "tab-123", payload)

	client.handleBinaryMessage(frame)

	received := make([]byte, len(payload))
	if _, err := io.ReadFull(readEnd, received); err != nil {
		t.Fatalf("failed to read PTY payload: %v", err)
	}
	if string(received) != string(payload) {
		t.Fatalf("unexpected payload: got %q want %q", string(received), string(payload))
	}

	if got := sess.lastInputSeq["tab-123"]; got != 42 {
		t.Fatalf("sequence not recorded: got %d want %d", got, 42)
	}
}

func TestHandleBinaryMessageClientFrameCompatibility(t *testing.T) {
	readEnd, writeEnd, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe creation failed: %v", err)
	}
	t.Cleanup(func() {
		_ = readEnd.Close()
		_ = writeEnd.Close()
	})

	sess := &session{
		ptyFile:      writeEnd,
		lastInputSeq: make(map[string]uint64),
		metrics:      newMetricsRegistry(),
		transcript: &transcriptManager{
			writer: bufio.NewWriter(io.Discard),
		},
	}

	client := &wsClient{session: sess}

	payload := []byte("printf 'ok'\n")
	frame := make([]byte, 1+8+2+len(payload))
	frame[0] = 0x01
	binary.BigEndian.PutUint64(frame[1:9], 1)
	binary.BigEndian.PutUint16(frame[9:11], 0)
	copy(frame[11:], payload)
	if got := binary.BigEndian.Uint64(frame[1:9]); got != 1 {
		t.Fatalf("unexpected encoded seq: got %d want %d", got, 1)
	}

	client.handleBinaryMessage(frame)

	received := make([]byte, len(payload))
	if _, err := io.ReadFull(readEnd, received); err != nil {
		t.Fatalf("failed to read PTY payload: %v", err)
	}
	if string(received) != string(payload) {
		t.Fatalf("unexpected payload: got %q want %q", string(received), string(payload))
	}

}

func TestHandleBinaryMessageIgnoresInvalidFrames(t *testing.T) {
	sess := &session{
		lastInputSeq: make(map[string]uint64),
	}
	client := &wsClient{session: sess}

	client.handleBinaryMessage(nil)
	client.handleBinaryMessage([]byte{0x01})
	client.handleBinaryMessage([]byte{0x02, 0x00})

	if len(sess.lastInputSeq) != 0 {
		t.Fatalf("expected no sequence tracking entries, got %d", len(sess.lastInputSeq))
	}
}

func buildBinaryInputFrame(seq uint64, source string, data []byte) []byte {
	sourceBytes := []byte(source)
	if len(sourceBytes) > 0xffff {
		sourceBytes = sourceBytes[:0xffff]
	}

	frame := make([]byte, 1+8+2+len(sourceBytes)+len(data))
	frame[0] = 0x01
	binary.BigEndian.PutUint64(frame[1:9], seq)
	binary.BigEndian.PutUint16(frame[9:11], uint16(len(sourceBytes)))
	copy(frame[11:], sourceBytes)
	copy(frame[11+len(sourceBytes):], data)
	return frame
}
