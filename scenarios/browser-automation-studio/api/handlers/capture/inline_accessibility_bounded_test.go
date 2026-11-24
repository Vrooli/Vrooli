package capture

import (
	"bytes"
	"testing"
)

type countedReader struct {
	reader *bytes.Reader
	read   int
}

func (r *countedReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.read += n
	return n, err
}

func TestInlineAccessibilityReadStopsAtResponseLimitSentinel(t *testing.T) {
	const limit = 8
	input := []byte("0123456789abcdef")
	reader := &countedReader{reader: bytes.NewReader(input)}

	got, err := readInlineAccessibilityBounded(reader, limit)
	if err != nil {
		t.Fatal(err)
	}
	if got != string(input[:limit]) {
		t.Fatalf("snapshot = %q, want %q", got, input[:limit])
	}
	if reader.read != limit+1 {
		t.Fatalf("bytes consumed = %d, want limit sentinel %d", reader.read, limit+1)
	}
}
