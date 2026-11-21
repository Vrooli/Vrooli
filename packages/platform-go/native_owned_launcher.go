package platform

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	NativeLauncherProofVersion   = 1
	NativeLauncherProfileOpenRGB = "openrgb-maintained-launcher-v1"
)

// NativeLauncherProof identifies an exact maintained host published by native
// setup. It selects no version: the domain owner must obtain Path and SHA256
// from its exact admitted current role or a fresh verified publication.
// The machine-owned content-addressed publisher seal supplies durable code
// authority independently of a subsequently removed boot definition or role.
type NativeLauncherProof struct {
	SchemaVersion int    `json:"version"`
	Profile       string `json:"profile"`
	SHA256        string `json:"sha256"`
}

// Validate checks only the closed proof vocabulary, without native effects.
func (p NativeLauncherProof) Validate() error {
	if p.SchemaVersion != NativeLauncherProofVersion || p.Profile != NativeLauncherProfileOpenRGB {
		return fmt.Errorf("%w: unsupported native launcher publication profile", ErrUnsupported)
	}
	if !validNativeBrokerDigest(p.SHA256) {
		return fmt.Errorf("%w: malformed native launcher checksum", ErrOperationIdentity)
	}
	return nil
}

// NativeLauncherPublisherRoot returns the canonical collection of maintained
// launcher versions. It observes a path, creates nothing and accepts no override.
func NativeLauncherPublisherRoot(profile string) (string, error) {
	if profile != NativeLauncherProfileOpenRGB {
		return "", ErrUnsupported
	}
	return nativeLauncherPublisherRoot()
}

// OpenPublishedNativeLauncher opens exactly the frozen published image after
// proving canonical namespace, machine ownership, immutable ancestry, checksum,
// native architecture and loader closure. The caller owns the returned FD.
// It does not discover historical versions, infer a current role, or launch it.
func OpenPublishedNativeLauncher(ctx context.Context, path string, proof NativeLauncherProof) (*os.File, error) {
	name := "vrooli"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return openPublishedNativeLauncherWith(ctx, path, proof, name, nativeLauncherPublisherRoot, openPublishedNativeLauncher)
}

// The injected boundary is private and immutable in production. Portable
// fixtures exercise this complete admission ordering with isolated publishers.
func openPublishedNativeLauncherWith(ctx context.Context, path string, proof NativeLauncherProof, name string, root func() (string, error), open func(context.Context, string, string) (*os.File, error)) (*os.File, error) {
	if ctx == nil {
		return nil, ErrOperationIdentity
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := proof.Validate(); err != nil {
		return nil, err
	}
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || len(path) > 4096 || strings.ContainsAny(path, "\x00\r\n") {
		return nil, ErrOperationIdentity
	}
	collection, err := root()
	if err != nil {
		return nil, err
	}
	if !filepath.IsAbs(collection) || filepath.Clean(collection) != collection || path != filepath.Join(collection, proof.SHA256, name) {
		return nil, fmt.Errorf("%w: native launcher is outside its exact publication", ErrOperationIdentity)
	}
	image, err := open(ctx, path, proof.SHA256)
	if err != nil {
		if image != nil {
			image.Close()
		}
		return nil, err
	}
	if image == nil {
		return nil, ErrOperationIdentity
	}
	if err = ctx.Err(); err == nil {
		_, err = image.Seek(0, io.SeekStart)
	}
	if err != nil {
		image.Close()
		return nil, err
	}
	return image, nil
}
