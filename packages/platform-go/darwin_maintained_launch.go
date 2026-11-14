package platform

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

// Native construction supplies the immutable installed-host image owner and
// native path/ACL verifier. Fixtures reuse the complete shape/hash guard with
// isolated files; no mutable launcher registration or arbitrary callback state
// is persisted or exposed to callers.
type darwinMaintainedLaunchGuard struct {
	approved   func(context.Context) (*os.File, error)
	verifyPath func(context.Context, string, bool) error
	published  func(context.Context, string, NativeLauncherProof) (*os.File, error)
}

func (g darwinMaintainedLaunchGuard) validate(ctx context.Context, i OperationIntent, s OperationSpec) error {
	if e := validateDarwinMaintainedLaunchShape(i, s); e != nil {
		return e
	}
	if s.NativeLauncher != nil {
		if g.published == nil {
			return ErrUnsupported
		}
		file, err := g.published(ctx, s.Path, *s.NativeLauncher)
		if err != nil {
			return err
		}
		if file == nil {
			return ErrOperationIdentity
		}
		if err = file.Close(); err != nil {
			return err
		}
		return ctx.Err()
	}
	if e := g.verifyPath(ctx, s.Path, false); e != nil {
		return e
	}
	approved, e := g.approved(ctx)
	if e != nil {
		return e
	}
	defer approved.Close()
	h := sha256.New()
	n, e := io.Copy(h, io.LimitReader(approved, maxNativeSystemBrokerImageBytes+1))
	if e != nil {
		return e
	}
	if n <= 0 || n > maxNativeSystemBrokerImageBytes {
		return ErrOperationIdentity
	}
	candidate, e := os.Open(s.Path)
	if e != nil {
		return e
	}
	defer candidate.Close()
	if e = verifyNativeSystemBrokerImage(candidate, hex.EncodeToString(h.Sum(nil))); e != nil {
		return e
	}
	return ctx.Err()
}
