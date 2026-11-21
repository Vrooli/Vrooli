//go:build !windows

package resources

import (
	"context"
	"errors"
	"fmt"
	"os"
	"syscall"
)

// The private owner directory and socket admit only the actual operator. Every
// frame additionally requires the exact managed peer PID and incarnation.
func configureOpenRGBNativeClaimSocket(ctx context.Context, c openRGBNativeCapsule) (os.FileInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	expected, err := openRGBNativeCapsuleFrozenEndpoint(c)
	if err != nil || expected != c.Endpoint {
		return nil, errors.New("native claim socket path differs from admitted capsule")
	}

	return configureOpenRGBNativeSocketAccess(ctx, c.Endpoint, c.OwnerPrincipal, "")
}
func configureOpenRGBNativeSocketAccess(ctx context.Context, path, ownerPrincipal, runtimePrincipal string) (os.FileInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	native, ok := info.Sys().(*syscall.Stat_t)
	if !ok || native.Uid != uint32(os.Geteuid()) || ownerPrincipal != fmt.Sprintf("unix:%d", os.Getuid()) {
		return nil, errors.New("native socket belongs to another kernel principal")
	}
	if info.Mode()&os.ModeSocket == 0 || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("native claim endpoint is not a real socket")
	}
	if err = os.Chmod(path, 0600); err != nil {
		return info, err
	}
	after, err := os.Lstat(path)
	if err != nil || !os.SameFile(info, after) || after.Mode().Perm() != 0600 {
		return info, errors.New("native claim socket changed during access setup")
	}
	return after, nil
}
