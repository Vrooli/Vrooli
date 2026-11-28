package platform

import (
	"context"
	"os"
	"path/filepath"
)

// A private immutable native boundary: production supplies Windows path/ACL
// pinning and opening; portable fixtures retain the REAL image SHA/PE parser.
// This is never a launcher registry or a caller-selected execution hook.
type windowsPublishedLauncherOpenEffects struct {
	PinParents func(string) (func(), error)
	OpenImage  func(string) (*os.File, error)
}

func openWindowsPublishedLauncherWith(ctx context.Context, path, digest, architecture string, e windowsPublishedLauncherOpenEffects) (*os.File, error) {
	if ctx == nil {
		return nil, ErrOperationIdentity
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if e.PinParents == nil || e.OpenImage == nil {
		return nil, ErrOperationIdentity
	}
	release, err := e.PinParents(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	if release == nil {
		return nil, ErrOperationIdentity
	}
	defer release()
	image, err := e.OpenImage(path)
	if err != nil {
		return nil, err
	}
	if image == nil {
		return nil, ErrOperationIdentity
	}
	fail := func(err error) (*os.File, error) { image.Close(); return nil, err }
	if err = verifyNativeSystemBrokerImage(image, digest); err != nil {
		return fail(err)
	}
	if err = verifyWindowsMaintainedLauncherPE(image, architecture); err != nil {
		return fail(err)
	}
	if err = ctx.Err(); err != nil {
		return fail(err)
	}
	if _, err = image.Seek(0, 0); err != nil {
		return fail(err)
	}
	return image, nil
}
