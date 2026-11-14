package platform

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// VerifyImmutableNativePath observes machine-owned execution authority. It
// rejects foreign write authority through modes, ACLs, aliases or ancestors;
// directory mutation by the privileged publishing owner remains its governed
// responsibility. It performs no provisioning and requires no elevation.
func VerifyImmutableNativePath(ctx context.Context, path string, directory bool) error {
	if ctx == nil {
		return fmt.Errorf("platform: immutable path context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || len(path) > 4096 || strings.ContainsAny(path, "\x00\r\n") {
		return ErrOperationIdentity
	}
	return verifyImmutableNativePath(ctx, path, directory)
}

// FrozenOwnedServiceEnvironment returns a copy of the complete digest-bound
// nonsecret environment for an exact running native service receipt. Native
// manager additions must never be guessed or whitelisted by a capsule.
func FrozenOwnedServiceEnvironment(ctx context.Context, ref ServiceRef) ([]string, error) {
	if ctx == nil {
		return nil, fmt.Errorf("platform: frozen service environment context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := ref.ValidateIdentity(); err != nil {
		return nil, err
	}
	return frozenOwnedServiceEnvironment(ctx, ref)
}

// PrepareImmutableNativeDirectory atomically creates privileged-owned machine
// directories and pins the verified ancestor chain until release. Domain
// publishers must bind their own closed destination namespace first.
func PrepareImmutableNativeDirectory(ctx context.Context, path string) (func() error, error) {
	if ctx == nil {
		return nil, fmt.Errorf("platform: immutable directory context required")
	}
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || len(path) > 4096 || strings.ContainsAny(path, "\x00\r\n") {
		return nil, ErrOperationIdentity
	}
	return prepareImmutableNativeDirectory(ctx, path)
}

// WriteImmutableNativeFile publishes new machine-owned bytes without replacing
// an existing inode. The domain callback must validate the complete expected
// bytes before returning nil; failures never publish a target.
func WriteImmutableNativeFile(ctx context.Context, path string, write func(io.Writer) error) error {
	if ctx == nil || write == nil {
		return fmt.Errorf("platform: immutable file context and writer required")
	}
	if e := ctx.Err(); e != nil {
		return e
	}
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || len(path) > 4096 || strings.ContainsAny(path, "\x00\r\n") {
		return ErrOperationIdentity
	}
	return writeImmutableNativeFile(ctx, path, write)
}

// GrantImmutableNativeCodeReaders permits selected principals to inspect
// metadata/traverse the fixed machine code store, never private state or HOME.
// It grants no directory listing/data/write or ownership authority.
func GrantImmutableNativeCodeReaders(ctx context.Context, path string, readers []string) error {
	if ctx == nil || len(readers) == 0 || len(readers) > 16 {
		return fmt.Errorf("platform: exact code metadata readers required")
	}
	if e := ctx.Err(); e != nil {
		return e
	}
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || len(path) > 4096 || strings.ContainsAny(path, "\x00\r\n") {
		return ErrOperationIdentity
	}
	return grantImmutableNativeCodeReaders(ctx, path, readers)
}
