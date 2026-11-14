package hyperframes

import (
	"context"
	"fmt"
	"os"
)

// openRegularContext prevents special-file replacement from blocking before
// the opened descriptor can be checked. Unix supplies nonblocking/no-follow
// open; unsupported platforms retain pre/open inode checks without that claim.
func openRegularContext(ctx context.Context, path string, max int64) (*os.File, error) {
	return openCheckedRegular(ctx, path, max, openRegularFile)
}

func openCheckedRegular(ctx context.Context, path string, max int64, open func(string) (*os.File, error)) (*os.File, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	before, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !before.Mode().IsRegular() || before.Size() > max {
		return nil, fmt.Errorf("expected regular file of at most %d bytes: %s", max, path)
	}
	f, err := open(path)
	if err != nil {
		return nil, err
	}
	after, err := f.Stat()
	if err == nil && (!after.Mode().IsRegular() || !os.SameFile(before, after) || after.Size() > max) {
		err = fmt.Errorf("file type or identity changed before open: %s", path)
	}
	if err == nil {
		err = ctx.Err()
	}
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	return f, nil
}
