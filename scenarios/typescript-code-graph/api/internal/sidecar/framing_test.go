package sidecar

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFrameWriterAppendsNewline(t *testing.T) {
	var buf bytes.Buffer
	w := newFrameWriter(&buf)
	require.NoError(t, w.Write(map[string]string{"hello": "world"}))
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
			require.NoError(t, w.Write(map[string]int{"i": i}))
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
