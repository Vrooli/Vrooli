package detection

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
)

// Source scans are bounded independently of repository size. Oversized or
// unreadable source fails the analysis instead of publishing incomplete rows.
const maxDetectionSourceBytes = 16 << 20

func readDetectionSource(ctx context.Context, path string, entry fs.DirEntry) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !entry.Type().IsRegular() {
		return nil, fmt.Errorf("dependency source is not a regular file: %s", path)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > maxDetectionSourceBytes {
		return nil, fmt.Errorf("dependency source is nonregular or exceeds %d bytes: %s", maxDetectionSourceBytes, path)
	}
	data, err := io.ReadAll(io.LimitReader(file, maxDetectionSourceBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxDetectionSourceBytes {
		return nil, fmt.Errorf("dependency source grew beyond %d bytes: %s", maxDetectionSourceBytes, path)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return data, nil
}
