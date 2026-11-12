package platform

import (
	"context"
	"os"
)

// This reader proves a root-published immutable launcher directly. It does not
// consult a boot-unit definition, current manager, journal, newest pointer or
// mutable registration. Native construction supplies the existing root/ACL path
// verifier and the actual Mach-O architecture/system-loader closure parser.
type darwinPublishedLauncherReader struct {
	path   func(context.Context, string, bool) error
	format func(*os.File) error
}

func (r darwinPublishedLauncherReader) open(ctx context.Context, path, digest string) (*os.File, error) {
	if ctx == nil || !validNativeBrokerDigest(digest) {
		return nil, ErrOperationIdentity
	}
	if err := r.path(ctx, path, false); err != nil {
		return nil, err
	}
	before, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	immutable := func(st os.FileInfo) bool {
		return st.Mode().IsRegular() && st.Mode().Perm()&0222 == 0 && st.Mode().Perm()&0111 != 0
	}
	if !immutable(before) {
		return nil, ErrOperationIdentity
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	fail := func(err error) (*os.File, error) { file.Close(); return nil, err }
	opened, err := file.Stat()
	if err != nil {
		return fail(err)
	}
	if !immutable(opened) || !os.SameFile(before, opened) {
		return fail(ErrOperationIdentity)
	}
	if err = verifyNativeSystemBrokerImage(file, digest); err != nil {
		return fail(err)
	}
	if err = r.format(file); err != nil {
		return fail(err)
	}
	if err = r.path(ctx, path, false); err != nil {
		return fail(err)
	}
	final, err := os.Lstat(path)
	if err != nil {
		return fail(err)
	}
	if !immutable(final) || !os.SameFile(opened, final) {
		return fail(ErrOperationIdentity)
	}
	if err = ctx.Err(); err != nil {
		return fail(err)
	}
	return file, nil
}
