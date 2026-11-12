package resources

import (
	"context"
	"errors"
	"io"
	"os"
)

// Offline decoding and privileged publication share format validation, while
// keeping their creation authority explicit. Research/test extraction uses
// only invoking-account private files; production Windows uses atomic native
// machine-owned constructors from its sealed-store publisher.
type openRGBTreeIO struct {
	mkdir func(context.Context, string) error
	write func(context.Context, string, os.FileMode, func(io.Writer) error) error
}

func openRGBPrivateTreeIO() openRGBTreeIO {
	return openRGBTreeIO{mkdir: func(ctx context.Context, path string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		return os.MkdirAll(path, 0o700)
	}, write: func(ctx context.Context, path string, mode os.FileMode, write func(io.Writer) error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
		if err != nil {
			return err
		}
		return errors.Join(write(f), f.Close())
	}}
}
func openRGBSealedTreeIO() openRGBTreeIO {
	return openRGBTreeIO{mkdir: openRGBMakeStageDirectory, write: openRGBWriteStageFile}
}
