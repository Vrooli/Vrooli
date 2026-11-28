package source

import (
	"context"
	"fmt"
	"os"
)

// openRegularRecording refuses special files before opening and checks the actual
// opened inode. Unix uses nonblocking/no-follow open so replacement by a FIFO or
// symlink between stat and open cannot strand an export worker.
func openRegularRecording(ctx context.Context, path string) (*os.File, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	before, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !before.Mode().IsRegular() {
		return nil, fmt.Errorf("recording must be a regular file without symlinks")
	}
	file, err := openRecordingFile(path)
	if err != nil {
		return nil, err
	}
	after, err := file.Stat()
	if err == nil && (!after.Mode().IsRegular() || !os.SameFile(before, after)) {
		err = fmt.Errorf("recording type or identity changed before open")
	}
	if err == nil {
		err = ctx.Err()
	}
	if err != nil {
		file.Close()
		return nil, err
	}
	return file, nil
}
