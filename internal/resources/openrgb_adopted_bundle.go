package resources

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/binaryfetch"
)

func openRGBAdoptedBundleRoot(payload string, p openRGBReleaseProfile) (string, error) {
	resolved, err := filepath.EvalSymlinks(payload)
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(resolved) {
		return "", errors.New("adopted OpenRGB payload requires an absolute path")
	}
	suffix := string(filepath.Separator) + filepath.FromSlash(p.Payload)
	if !strings.HasSuffix(resolved, suffix) {
		return "", errors.New("adopted OpenRGB payload is outside the exact native bundle layout")
	}
	root := strings.TrimSuffix(resolved, suffix)
	if root == "" || root == string(filepath.Separator) {
		return "", errors.New("adopted OpenRGB bundle root is invalid")
	}
	return root, nil
}

func copyOpenRGBAdoptedBundle(ctx context.Context, source, destination string, p openRGBReleaseProfile) error {
	return copyOpenRGBAdoptedBundleWithIO(ctx, source, destination, p, openRGBPrivateTreeIO())
}

func copyOpenRGBAdoptedBundleWithIO(ctx context.Context, source, destination string, p openRGBReleaseProfile, owner openRGBTreeIO) error {
	if err := binaryfetch.VerifyArtifactContext(ctx, source, "dir", p.TreeSHA256); err != nil {
		return err
	}
	if err := p.verifyPayloadABI(filepath.Join(source, filepath.FromSlash(p.Payload))); err != nil {
		return err
	}
	type link struct{ path, target string }
	var links []link
	entries := 0
	var size int64
	err := filepath.WalkDir(source, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		entries++
		if entries > 512 {
			return errors.New("adopted OpenRGB bundle exceeds entry limit")
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.IsDir() {
			return owner.mkdir(ctx, target)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			value, err := os.Readlink(path)
			if err != nil {
				return err
			}
			links = append(links, link{target, value})
			return nil
		}
		if !info.Mode().IsRegular() || info.Size() > (128<<20)-size {
			return errors.New("adopted OpenRGB bundle contains unsafe files")
		}
		size += info.Size()
		in, err := openOpenRGBSourceFile(path, info)
		if err != nil {
			return err
		}
		defer in.Close()
		mode := os.FileMode(0o600)
		if info.Mode().Perm()&0o111 != 0 {
			mode = 0o700
		}
		return owner.write(ctx, target, mode, func(out io.Writer) error {
			n, err := io.Copy(out, io.LimitReader(&managedUpdateContextReader{ctx: ctx, reader: in}, info.Size()+1))
			if err != nil {
				return err
			}
			if n != info.Size() {
				return errors.New("adopted source file changed while copying")
			}
			return nil
		})
	})
	if err != nil {
		return err
	}
	for _, l := range links {
		if err := verifyOpenRGBExtractionLinkParent(destination, l.path, false); err != nil {
			return err
		}
		if err := os.Symlink(l.target, l.path); err != nil {
			return err
		}
	}
	return binaryfetch.VerifyArtifactContext(ctx, destination, "dir", p.TreeSHA256)
}

// Explicit adoption performs offline content/ABI proof before any program can
// run, so upstream constructors cannot inspect ambient preferences or devices.
func inspectOpenRGBAdoptedSource(ctx context.Context, payload string) (string, error) {
	p, err := openRGBReleaseProfileForCurrentHost()
	if err != nil {
		return "", err
	}
	return inspectOpenRGBAdoptedSourceForProfile(ctx, payload, p)
}

// Explicit profiles support offline release qualification only; production
// adoption obtains its current native host profile above.
func inspectOpenRGBAdoptedSourceForProfile(ctx context.Context, payload string, p openRGBReleaseProfile) (string, error) {
	if err := p.validate(); err != nil {
		return "", err
	}
	digest, err := binaryfetch.FileDigestContext(ctx, payload)
	if err != nil {
		return "", err
	}
	if digest == p.EnvelopeSHA256 {
		return digest, nil
	}
	if digest != p.PayloadSHA256 {
		return "", errors.New("adopted OpenRGB is not the exact qualified native release payload or envelope")
	}
	root, err := openRGBAdoptedBundleRoot(payload, p)
	if err != nil {
		return "", err
	}
	if err := binaryfetch.VerifyArtifactContext(ctx, root, "dir", p.TreeSHA256); err != nil {
		return "", err
	}
	if err := p.verifyPayloadABI(payload); err != nil {
		return "", err
	}
	return digest, nil
}
