//go:build darwin

package resources

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/config"
	"golang.org/x/sys/unix"
)

func openRGBStagePreparationOwner() (openRGBStagePreparation, error) {
	p, err := openRGBReleaseProfileForCurrentHost()
	if err != nil {
		return openRGBStagePreparation{}, err
	}
	return openRGBStagePreparation{payloadRelative: p.Payload, extract: func(ctx context.Context, f *os.File, d string) error { return extractOpenRGBZip(ctx, f, d, p) }, seal: sealOpenRGBStage, verifySeal: verifyOpenRGBAppDirSeal, publish: func(a, b string) error { return unix.RenamexNp(a, b, unix.RENAME_EXCL) }, syncDir: syncOpenRGBStageDirectory}, nil
}
func verifyOpenRGBFileCapabilities(string) error { return nil } // Mach-O has no Linux file-capability xattr.

func openRGBArtifactOwnerUID() (uint32, error) { return 0, nil }

func prepareOpenRGBNativeArtifactParents(parent string) error {
	if os.Geteuid() != 0 {
		return errors.New("root-sealed OpenRGB artifacts require explicit privileged native setup; ordinary enable/start never provisions them")
	}
	root, err := OpenRGBNativeSealedRoot()
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(root, parent)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return errors.New("native artifact parent escapes sealed root")
	}
	for dir := parent; ; dir = filepath.Dir(dir) {
		info, err := os.Lstat(dir)
		if err != nil {
			return err
		}
		st, ok := info.Sys().(*syscall.Stat_t)
		if !ok || st.Uid != 0 || !info.IsDir() {
			return errors.New("native sealed ancestor is not root-owned")
		}
		if err := os.Chmod(dir, 0o755); err != nil {
			return err
		}
		if err := platform.VerifyImmutableNativePath(context.Background(), dir, true); err != nil {
			return err
		}
		if dir == root {
			return nil
		}
	}
}

func openRGBArtifactNativeACLVerifier() (func(string, os.FileInfo, bool) error, error) {
	return func(path string, info os.FileInfo, _ bool) error {
		return platform.VerifyImmutableNativePath(context.Background(), path, info.IsDir())
	}, nil
}

func openRGBArtifactNativeACLSealer() (func(string, os.FileInfo) error, error) {
	if os.Geteuid() != 0 {
		return nil, errors.New("native root-sealed OpenRGB publication requires privileged setup")
	}
	o, err := readOpenRGBNativeOwner()
	if err != nil {
		return nil, err
	}
	p := config.PrivatePathPolicy{OwnerPrincipal: "uid:0", RuntimePrincipal: o.RuntimePrincipal, RuntimeReadOnly: true, NoInherit: true}
	return func(path string, info os.FileInfo) error {
		return config.ConfigureNativeReadOnlyPathAccess(path, info, p)
	}, nil
}
