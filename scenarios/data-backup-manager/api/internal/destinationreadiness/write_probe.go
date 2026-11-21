package destinationreadiness

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"time"
)

// writeProbeBytes is deliberately tiny: this proves the destination accepts
// and returns bytes, not that the filesystem is consistent.
const writeProbeBytes = 4096

// FileWriteProber is the production write probe. It creates one bounded file
// under the destination, fsyncs it, reads it back, verifies the bytes, removes
// it, and fsyncs the directory. Every artifact is cleaned up on every path.
type FileWriteProber struct{}

// NewFileWriteProber constructs the production probe.
func NewFileWriteProber() WriteProber { return FileWriteProber{} }

// Probe implements WriteProber.
func (FileWriteProber) Probe(ctx context.Context, location string) (result WriteProbeResult, err error) {
	result.Ran = true
	started := time.Now()
	defer func() { result.Latency = time.Since(started) }()

	if err := ctx.Err(); err != nil {
		result.Error = err.Error()
		return result, nil
	}

	payload := make([]byte, writeProbeBytes)
	if _, err := rand.Read(payload); err != nil {
		result.Error = fmt.Sprintf("generate probe bytes: %v", err)
		return result, nil
	}
	result.Bytes = int64(len(payload))

	file, err := os.CreateTemp(location, ".dbm-write-probe-*")
	if err != nil {
		result.Error = fmt.Sprintf("create probe file: %v", err)
		return result, nil
	}
	name := file.Name()
	cleanup := func() {
		_ = file.Close()
		_ = os.Remove(name)
	}

	if _, err := file.Write(payload); err != nil {
		cleanup()
		result.Error = fmt.Sprintf("write probe file: %v", err)
		return result, nil
	}
	if err := file.Sync(); err != nil {
		cleanup()
		result.Error = fmt.Sprintf("fsync probe file: %v", err)
		return result, nil
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(name)
		result.Error = fmt.Sprintf("close probe file: %v", err)
		return result, nil
	}

	got, err := os.ReadFile(name)
	if err != nil {
		_ = os.Remove(name)
		result.Error = fmt.Sprintf("read probe file: %v", err)
		return result, nil
	}
	if !bytes.Equal(got, payload) {
		_ = os.Remove(name)
		result.Error = "probe read-back did not match the written bytes"
		return result, nil
	}
	if err := os.Remove(name); err != nil {
		result.Error = fmt.Sprintf("remove probe file: %v", err)
		return result, nil
	}
	if dir, err := os.Open(location); err == nil {
		syncErr := dir.Sync()
		_ = dir.Close()
		if syncErr != nil {
			result.Error = fmt.Sprintf("fsync destination directory: %v", syncErr)
			return result, nil
		}
	}

	result.Passed = true
	return result, nil
}
