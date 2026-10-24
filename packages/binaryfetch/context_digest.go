package binaryfetch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

// FileDigestContext streams regular file content without allocating the whole
// artifact. Final symlinks retain legacy file-verification semantics; the open
// file must match the statted regular target. Kernel filesystem calls themselves
// are not preemptible; cancellation is checked between bounded reads.
func FileDigestContext(ctx context.Context, path string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	before, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !before.Mode().IsRegular() {
		return "", fmt.Errorf("binaryfetch: digest input %q is not a regular file", path)
	}
	file, err := openDigestFile(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return "", err
	}
	if !opened.Mode().IsRegular() || !os.SameFile(before, opened) {
		return "", fmt.Errorf("binaryfetch: digest input %q changed or is not regular", path)
	}
	return digestReaderContext(ctx, file)
}

func digestReaderContext(ctx context.Context, reader io.Reader) (string, error) {
	hash := sha256.New()
	buffer := make([]byte, 64*1024)
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		n, err := reader.Read(buffer)
		if canceled := ctx.Err(); canceled != nil {
			return "", canceled
		}
		if n > 0 {
			_, _ = hash.Write(buffer[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if n == 0 {
			return "", io.ErrNoProgress
		}
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
