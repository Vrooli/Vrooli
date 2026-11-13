package sidecar

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFrameWriterAppendsNewline(t *testing.T) {
	var buf bytes.Buffer
	w := newFrameWriter(&buf)
	require.NoError(t, w.Write(context.Background(), map[string]string{"hello": "world"}))
	require.Equal(t, `{"hello":"world"}`+"\n", buf.String())
}

func TestFrameWriterIsConcurrentSafe(t *testing.T) {
	var buf bytes.Buffer
	w := newFrameWriter(&buf)
	const n = 50
	done := make(chan struct{}, n)
	for i := 0; i < n; i++ {
		i := i
		go func() {
			defer func() { done <- struct{}{} }()
			require.NoError(t, w.Write(context.Background(), map[string]int{"i": i}))
		}()
	}
	for i := 0; i < n; i++ {
		<-done
	}
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	require.Len(t, lines, n)
	// Each line must be a parseable JSON object — proves the writes did
	// not interleave bytes.
	for _, l := range lines {
		var m map[string]int
		require.NoError(t, json.Unmarshal([]byte(l), &m))
	}
}

// TestFrameWriterHonoursDeadlineWithoutTearingFrames models a child that
// stopped reading stdin: the write that is in progress and the writer queued
// behind it both return at their deadlines, and once the reader resumes it
// sees whole frames only.
func TestFrameWriterHonoursDeadlineWithoutTearingFrames(t *testing.T) {
	r, pw := io.Pipe()
	w := newFrameWriter(pw)

	for _, name := range []string{"in-progress", "queued"} {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		start := time.Now()
		err := w.Write(ctx, map[string]string{"frame": name})
		cancel()
		require.True(t, errors.Is(err, context.DeadlineExceeded), "%s write: got %v", name, err)
		require.Less(t, time.Since(start), time.Second, "%s write ignored its deadline", name)
	}

	// The reader resumes: the frame that had started writing arrives whole,
	// and a later write follows it on its own line.
	frames := make(chan string, 2)
	go func() {
		scanner := newFrameScanner(r)
		for scanner.Scan() {
			frames <- scanner.Text()
		}
	}()
	require.NoError(t, w.Write(context.Background(), map[string]string{"frame": "next"}))
	require.Equal(t, `{"frame":"in-progress"}`, <-frames)
	require.Equal(t, `{"frame":"next"}`, <-frames)
}

func TestFrameScannerHandlesLargeLine(t *testing.T) {
	// 2 MiB string inside a JSON object → comfortably above the default
	// 64 KiB bufio.Scanner buffer.
	const sz = 2 * 1024 * 1024
	big := strings.Repeat("a", sz)
	payload, err := json.Marshal(map[string]string{"big": big})
	require.NoError(t, err)
	buf := append(payload, '\n')

	scanner := newFrameScanner(bytes.NewReader(buf))
	require.True(t, scanner.Scan(), "scanner should accept oversize line; err=%v", scanner.Err())
	require.NoError(t, scanner.Err())

	var got map[string]string
	require.NoError(t, json.Unmarshal(scanner.Bytes(), &got))
	require.Equal(t, sz, len(got["big"]))
}
