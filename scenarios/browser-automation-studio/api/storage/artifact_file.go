package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

// ArtifactCopyReceipt describes the bytes copied from one artifact file into
// another. Storage backends and export callers share this filesystem operation.
type ArtifactCopyReceipt struct {
	SizeBytes int64
	SHA256    string
}

// ArtifactCopyError identifies which step failed while preserving the
// underlying filesystem error for errors.Is/errors.As callers.
type ArtifactCopyError struct {
	Operation string
	Err       error
}

func (e *ArtifactCopyError) Error() string { return e.Err.Error() }
func (e *ArtifactCopyError) Unwrap() error { return e.Err }

func artifactCopyError(operation string, err error) error {
	if err == nil {
		return nil
	}
	return &ArtifactCopyError{Operation: operation, Err: err}
}

// CopyArtifactFile replaces destination with source bytes and reports the
// copied byte count and digest from the same stream.
func CopyArtifactFile(source, destination string) (ArtifactCopyReceipt, error) {
	in, err := os.Open(source)
	if err != nil {
		return ArtifactCopyReceipt{}, artifactCopyError("open", err)
	}
	defer in.Close()
	return CopyArtifactReader(in, destination)
}

// CopyArtifactReader is the canonical stream-to-file artifact writer. It owns
// destination creation and returns the byte count and digest from the stream
// actually persisted, so callers do not each implement copy/close semantics.
func CopyArtifactReader(source io.Reader, destination string) (ArtifactCopyReceipt, error) {
	out, err := os.Create(destination)
	if err != nil {
		return ArtifactCopyReceipt{}, artifactCopyError("create", err)
	}
	return CopyArtifactReaderToOpenFile(source, out)
}

// CopyArtifactReaderToOpenFile is the canonical stream-to-open-file writer.
// The file is closed on both success and failure; callers retain ownership of
// any temporary-path removal.
func CopyArtifactReaderToOpenFile(source io.Reader, out *os.File) (ArtifactCopyReceipt, error) {
	if source == nil {
		_, err := CopyReaderToOpenFile(nil, out)
		return ArtifactCopyReceipt{}, err
	}
	hash := sha256.New()
	size, err := CopyReaderToOpenFile(io.TeeReader(source, hash), out)
	if err != nil {
		return ArtifactCopyReceipt{}, err
	}
	return ArtifactCopyReceipt{SizeBytes: size, SHA256: hex.EncodeToString(hash.Sum(nil))}, nil
}

// CopyReaderToOpenFile owns one stream transfer and closes the destination.
// Callers retain ownership of any path lifecycle and may avoid hashing when
// the receiving owner computes integrity as part of persistence.
func CopyReaderToOpenFile(source io.Reader, out *os.File) (int64, error) {
	if source == nil || out == nil {
		if out != nil {
			_ = out.Close()
		}
		return 0, artifactCopyError("copy", os.ErrInvalid)
	}
	size, copyErr := io.Copy(out, source)
	closeErr := out.Close()
	if copyErr != nil {
		return 0, artifactCopyError("copy", copyErr)
	}
	if closeErr != nil {
		return 0, artifactCopyError("close", closeErr)
	}
	return size, nil
}
