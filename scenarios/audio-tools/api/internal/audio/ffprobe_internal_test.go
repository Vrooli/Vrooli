package audio

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestContentTypeFor locks the internal content-type map to the handler's
// (handlers/audio/mappers.go). Split reports per-chunk content types from
// this map, so drift here would make Split disagree with every other op.
func TestContentTypeFor(t *testing.T) {
	cases := map[string]string{
		"":     "audio/wav",
		"wav":  "audio/wav",
		"mp3":  "audio/mpeg",
		"flac": "audio/flac",
		"aac":  "audio/aac",
		"ogg":  "audio/ogg",
		"webm": "audio/webm",
		"opus": "audio/opus",
		"???":  "application/octet-stream",
	}
	for format, want := range cases {
		require.Equal(t, want, ContentTypeFor(format), "format=%q", format)
	}
}

// TestWriteTempInputsCleansUpOnCreateFailure ensures a partial batch is
// removed when a later source cannot be materialized. Before the fix the
// in-flight file was only tracked after Close, so it leaked.
func TestWriteTempInputsCleansUpOnCreateFailure(t *testing.T) {
	dir := t.TempDir()
	orig := createTempFile
	t.Cleanup(func() { createTempFile = orig })

	calls := 0
	createTempFile = func(pattern string) (*os.File, error) {
		calls++
		if calls == 2 {
			return nil, errors.New("synthetic create failure")
		}
		return os.CreateTemp(dir, pattern)
	}

	_, err := writeTempInputs([][]byte{[]byte("a"), []byte("b")})
	require.Error(t, err)
	requireNoFiles(t, dir)
}

// TestWriteTempInputsCleansUpOnWriteFailure covers the in-flight file:
// the write fails after creation, so the file is not yet closed. Cleanup
// must still remove it.
func TestWriteTempInputsCleansUpOnWriteFailure(t *testing.T) {
	dir := t.TempDir()
	orig := createTempFile
	t.Cleanup(func() { createTempFile = orig })

	createTempFile = func(pattern string) (*os.File, error) {
		f, err := os.CreateTemp(dir, pattern)
		if err != nil {
			return nil, err
		}
		name := f.Name()
		_ = f.Close()
		// Re-open read-only so any Write fails while f.Name() still
		// points at the file cleanup must delete.
		return os.OpenFile(name, os.O_RDONLY, 0)
	}

	_, err := writeTempInputs([][]byte{[]byte("a")})
	require.Error(t, err)
	requireNoFiles(t, dir)
}

func requireNoFiles(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	require.Empty(t, names, "leaked temp files")
}
