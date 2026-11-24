package agenttranscript

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

// SHA256Bytes returns the lowercase SHA-256 digest of data.
func SHA256Bytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// SHA256BytesPrefix hashes at most maxBytes from data. A nonpositive bound
// hashes the empty prefix.
func SHA256BytesPrefix(data []byte, maxBytes int) string {
	if maxBytes < 0 {
		maxBytes = 0
	}
	if maxBytes < len(data) {
		data = data[:maxBytes]
	}
	return SHA256Bytes(data)
}

// SHA256File streams the complete contents of path into SHA-256.
func SHA256File(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// SHA256FilePrefixExact hashes exactly size bytes from path. It returns an
// error if the file is shorter than the requested prefix.
func SHA256FilePrefixExact(path string, size int64) (string, error) {
	if size < 0 {
		return "", fmt.Errorf("negative fingerprint size")
	}
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if size > 0 {
		written, err := io.CopyN(hash, file, size)
		if err != nil && err != io.EOF {
			return "", err
		}
		if written != size {
			return "", fmt.Errorf("fingerprint source changed: read %d bytes, expected %d", written, size)
		}
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
