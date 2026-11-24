package storage

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

type boundedReadProbe struct {
	max  int
	data []byte
}

func (r *boundedReadProbe) Read(p []byte) (int, error) {
	if len(p) > r.max {
		return 0, io.ErrShortBuffer
	}
	n := copy(p, r.data)
	r.data = r.data[n:]
	if n == 0 {
		return 0, io.EOF
	}
	return n, nil
}

func TestSniffContentTypeReadsAtMostDetectionPrefix(t *testing.T) {
	probe := &boundedReadProbe{max: 512, data: append([]byte("{\"artifact\":true}"), bytes.Repeat([]byte("x"), 1<<20)...)}
	require.Equal(t, "text/plain; charset=utf-8", SniffContentType(probe))
	require.Len(t, probe.data, (1<<20)+len("{\"artifact\":true}")-512)
}

func TestDetectContentTypeFromFilePreservesFallbackExtensionAndSniffOrder(t *testing.T) {
	root := t.TempDir()
	data := []byte("plain text content")
	unknown := filepath.Join(root, "opaque.unknown-ext")
	withType := filepath.Join(root, "capture.txt")
	require.NoError(t, os.WriteFile(unknown, data, 0o600))
	require.NoError(t, os.WriteFile(withType, data, 0o600))
	require.Equal(t, "text/plain; charset=utf-8", DetectContentTypeFromFile(unknown, ""))
	require.Equal(t, "text/plain; charset=utf-8", DetectContentTypeFromFile(withType, ""))
	require.Equal(t, "application/custom", DetectContentTypeFromFile(withType, "application/custom"))
}

func TestCopyArtifactFileReplacesDestinationAndReportsCopiedIdentity(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.bin")
	destination := filepath.Join(dir, "destination.bin")
	data := []byte("artifact bytes")
	require.NoError(t, os.WriteFile(source, data, 0o600))
	require.NoError(t, os.WriteFile(destination, []byte("long previous contents"), 0o600))

	receipt, err := CopyArtifactFile(source, destination)
	require.NoError(t, err)
	require.Equal(t, int64(len(data)), receipt.SizeBytes)
	sum := sha256.Sum256(data)
	require.Equal(t, hex.EncodeToString(sum[:]), receipt.SHA256)
	got, err := os.ReadFile(destination)
	require.NoError(t, err)
	require.Equal(t, data, got)
}

func TestCopyArtifactReaderMatchesFileCopyReceiptAndBytes(t *testing.T) {
	dir := t.TempDir()
	destination := filepath.Join(dir, "destination.bin")
	data := []byte("same source stream")
	receipt, err := CopyArtifactReader(bytes.NewReader(data), destination)
	require.NoError(t, err)
	require.Equal(t, int64(len(data)), receipt.SizeBytes)
	sum := sha256.Sum256(data)
	require.Equal(t, hex.EncodeToString(sum[:]), receipt.SHA256)
	got, err := os.ReadFile(destination)
	require.NoError(t, err)
	require.Equal(t, data, got)
}

func TestCopyReaderToOpenFileOwnsCloseAndPreservesCreatedFileMode(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "private-artifact-*")
	require.NoError(t, err)
	path := file.Name()
	createdInfo, err := file.Stat()
	require.NoError(t, err)
	data := []byte("private downloaded artifact")

	written, err := CopyReaderToOpenFile(bytes.NewReader(data), file)
	require.NoError(t, err)
	require.Equal(t, int64(len(data)), written)
	_, err = file.Write([]byte("closed"))
	require.Error(t, err, "the shared writer owns closing the destination")
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, createdInfo.Mode().Perm(), info.Mode().Perm(), "shared copy must preserve the temp path's creation mode")
	if runtime.GOOS != "windows" {
		require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	}
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, data, got)
}

func TestCopyArtifactFileReturnsSourceError(t *testing.T) {
	_, err := CopyArtifactFile(filepath.Join(t.TempDir(), "missing"), filepath.Join(t.TempDir(), "destination"))
	require.ErrorIs(t, err, os.ErrNotExist)
}
