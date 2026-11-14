package resources

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/vrooli/binaryfetch"
	platform "github.com/vrooli/platform-go"
)

// PrepareOpenRGBNativeLauncher belongs to explicit native privileged setup.
// The protected installed native boot definition selects the source image.
// The caller must bind its trusted installed service host to this same image
// digest; it cannot nominate a source pathname or a replacement executable.
// Existing exact sealed copies are reusable without publication authority.
func (c *Controller) PrepareOpenRGBNativeLauncher(ctx context.Context) (string, error) {
	if ctx == nil || c == nil || runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		return "", errors.New("native OpenRGB launcher preparation requires its explicit native setup context")
	}
	if _, err := openRGBReleaseProfileForCurrentHost(); err != nil {
		return "", err
	}
	image, err := platform.OpenInstalledNativeSupervisorImage(ctx)
	if err != nil {
		return "", err
	}
	defer image.Close()
	digest, err := managedUpdateImageDigest(ctx, image)
	if err != nil {
		return "", err
	}
	return c.stageOpenRGBNativeLauncher(ctx, image, digest)
}

// stageOpenRGBNativeLauncher publishes the exact already-open control-plane
// image into machine-owned execution authority. It never executes a candidate,
// acquires software, grants an account, or rewrites an existing sealed version.
func (c *Controller) stageOpenRGBNativeLauncher(ctx context.Context, image *os.File, digest string) (result string, returnErr error) {
	if ctx == nil || image == nil || c == nil || !updateDigest(digest) || (runtime.GOOS != "darwin" && runtime.GOOS != "windows") {
		return "", errors.New("native launcher staging requires its admitted image and exact checksum")
	}
	if _, err := openRGBReleaseProfileForCurrentHost(); err != nil {
		return "", err
	}
	digest = strings.ToLower(digest)
	root, err := OpenRGBNativeSealedRoot()
	if err != nil {
		return "", err
	}
	name := "vrooli"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	final := filepath.Join(root, "launchers", digest)
	target := filepath.Join(final, name)
	verify := func(path string) error {
		if err := platform.VerifyImmutableNativePath(ctx, path, false); err != nil {
			return err
		}
		if err := binaryfetch.VerifyArtifactContext(ctx, path, "file", digest); err != nil {
			return err
		}
		p, err := openRGBReleaseProfileForCurrentHost()
		if err != nil {
			return err
		}
		p.MinimumDarwinVersion = ^uint32(0)
		return p.verifyImageABI(path, true)
	}
	if _, err := os.Lstat(target); err == nil {
		return target, verify(target)
	} else if !os.IsNotExist(err) {
		return "", err
	}
	if err := requireOpenRGBNativeStagePublisher(); err != nil {
		return "", err
	}
	releaseParent, err := openRGBStageParentLease(ctx, filepath.Dir(final))
	if err != nil {
		return "", err
	}
	defer func() { returnErr = errors.Join(returnErr, releaseParent()) }()
	release, err := acquireOpenRGBStageLock(ctx, final)
	if err != nil {
		return "", err
	}
	defer release()
	if _, err := os.Lstat(target); err == nil {
		return target, verify(target)
	} else if !os.IsNotExist(err) {
		return "", err
	}
	pending := final + ".pending"
	if err := removeOpenRGBPendingStage(pending); err != nil {
		return "", err
	}
	if err := openRGBMakeStageScratch(ctx, pending); err != nil {
		return "", err
	}
	defer func() { returnErr = errors.Join(returnErr, removeOpenRGBPendingStage(pending)) }()
	if _, err := image.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	pendingImage := filepath.Join(pending, name)
	if err := openRGBWriteStageFile(ctx, pendingImage, 0o700, func(w io.Writer) error {
		const limit = int64(256 << 20)
		hash := sha256.New()
		size, err := io.Copy(io.MultiWriter(w, hash), io.LimitReader(&managedUpdateContextReader{ctx: ctx, reader: image}, limit+1))
		if err != nil {
			return err
		}
		if size <= 0 || size > limit || fmt.Sprintf("%x", hash.Sum(nil)) != digest {
			return errors.New("native launcher source changed or exceeds its admitted size bound")
		}
		return nil
	}); err != nil {
		return "", err
	}
	owner, err := openRGBStagePreparationOwner()
	if err != nil {
		return "", err
	}
	if err := owner.seal(ctx, pending); err != nil {
		return "", err
	}
	if err := verify(pendingImage); err != nil {
		return "", err
	}
	if err := owner.publish(pending, final); err != nil {
		return "", err
	}
	if err := owner.syncDir(filepath.Dir(final)); err != nil {
		return "", errors.Join(ErrOpenRGBStageCommitUnknown, err)
	}
	return target, verify(target)
}
