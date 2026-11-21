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

	platform "github.com/vrooli/platform-go"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/config"
)

const (
	openRGBSealedStagesDir = "openrgb-sealed"
	openRGBSealedAppDir    = "AppDir"
	openRGBSealedEnvelope  = "OpenRGB.AppImage"
)

var ErrOpenRGBStageCommitUnknown = errors.New("OpenRGB stage is visible but durable publication is unconfirmed")

// OpenRGBAppImageStagePath selects the exact owner-managed stage without
// creating, acquiring, extracting or inspecting a device. Retained rollback
// envelopes with the same release identity select the same stage.
func (c *Controller) OpenRGBAppImageStagePath(envelope ManagedServiceArtifact) (string, error) {
	profile, profileErr := openRGBReleaseProfileForCurrentHost()
	if profileErr != nil || profile.validate() != nil {
		return "", errOpenRGBReleasePayloadUnqualified
	}
	if c == nil || !filepath.IsAbs(c.Home) || filepath.Clean(c.Home) != c.Home || envelope.Resource != openRGBResourceName || envelope.Version != openRGBAppImageVersion || !profile.acceptsArtifactSHA256(envelope.SHA256) {
		return "", errOpenRGBReleasePayloadUnqualified
	}
	artifacts, err := repocontract.RuntimeHomeEntryPath(c.Home, repocontract.HomeKeyArtifacts)
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		artifacts, err = OpenRGBNativeSealedRoot()
	}
	if err != nil {
		return "", err
	}

	stage := filepath.Join(artifacts, openRGBSealedStagesDir, envelope.Version, envelope.SHA256)
	if profile.ServingRecipeID != "" {
		stage = filepath.Join(stage, "serving-"+profile.ServingRecipeSHA256)
	}
	return filepath.Join(stage, profile.bundleDirectory()), nil
}

// PrepareOpenRGBAppImageStage belongs to explicit resource setup after governed
// acquisition. It checks the current artifact owner and never acquires bytes,
// installs a tool, enables lighting, or changes the resource's service lifetime.
func (c *Controller) PrepareOpenRGBAppImageStage(ctx context.Context, envelope ManagedServiceArtifact) (string, error) {
	if ctx == nil {
		return "", errors.New("OpenRGB staging requires a context")
	}
	appDir, err := c.OpenRGBAppImageStagePath(envelope)
	if err != nil {
		return "", err
	}
	source, selected, err := c.ManagedServiceSource(ctx, openRGBResourceName)
	if err != nil || !selected || source.Mode != "managed" && source.Mode != "host" {
		return "", errors.Join(errors.New("OpenRGB payload staging requires an explicitly selected managed acquisition or exact host bundle owner"), err)
	}
	current, err := c.InspectManagedServiceArtifact(ctx, openRGBResourceName)
	if err != nil || current != envelope {
		return "", errors.Join(errors.New("OpenRGB stage source differs from the governed acquired artifact"), err)
	}

	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		final := filepath.Dir(appDir)
		if _, err := os.Lstat(final); err == nil {
			retained := envelope
			retained.Path = filepath.Join(final, openRGBSealedEnvelope)
			if _, err := verifyOpenRGBAppImageStage(ctx, retained, appDir); err != nil {
				return "", err
			}
			if err := syncOpenRGBStageDirectory(filepath.Dir(final)); err != nil {
				return "", errors.Join(ErrOpenRGBStageCommitUnknown, err)
			}
			return appDir, nil
		} else if !os.IsNotExist(err) {
			return "", err
		}
		if err := requireOpenRGBNativeStagePublisher(); err != nil {
			return "", err
		}
	}
	parent := filepath.Dir(filepath.Dir(appDir))
	releaseParent, err := openRGBStageParentLease(ctx, parent)
	if err != nil {
		return "", err
	}
	defer releaseParent()
	final := filepath.Dir(appDir)
	owner, err := openRGBStagePreparationOwner()
	if err != nil {
		return "", err
	}
	profile, err := openRGBReleaseProfileForCurrentHost()
	if err != nil {
		return "", err
	}
	if source.Mode == "host" && envelope.SHA256 == profile.PayloadSHA256 {
		sourceRoot, err := openRGBAdoptedBundleRoot(envelope.Path, profile)
		if err != nil {
			return "", err
		}
		owner.extract = func(ctx context.Context, _ *os.File, destination string) error {
			if runtime.GOOS == "windows" {
				return copyOpenRGBAdoptedBundleWithIO(ctx, sourceRoot, destination, profile, openRGBSealedTreeIO())
			}
			return copyOpenRGBAdoptedBundle(ctx, sourceRoot, destination, profile)
		}
	}
	owner.payloadRelative = profile.Payload
	owner.bundleDirectory = profile.bundleDirectory()
	owner.profile = &profile
	return prepareOpenRGBAppImageLocked(ctx, envelope, final, profile.servingTreeSHA256(), profile.PayloadSHA256, owner)
}

type openRGBStagePreparation struct {
	profile         *openRGBReleaseProfile
	payloadRelative string
	bundleDirectory string
	extract         func(context.Context, *os.File, string) error
	seal            func(context.Context, string) error
	verifySeal      func(context.Context, string) error
	publish         func(string, string) error
	syncDir         func(string) error
}

func prepareOpenRGBAppImageLocked(ctx context.Context, envelope ManagedServiceArtifact, final, treeSHA, payloadSHA string, owner openRGBStagePreparation) (string, error) {
	release, err := acquireOpenRGBStageLock(ctx, final)
	if err != nil {
		return "", err
	}
	defer release()
	return prepareOpenRGBAppImageAt(ctx, envelope, final, treeSHA, payloadSHA, owner)
}

// The caller holds the canonical per-stage lock. Tests use private fixtures and
// injected extraction to cover failure/retry without a host tool or service.
func prepareOpenRGBAppImageAt(ctx context.Context, envelope ManagedServiceArtifact, final, treeSHA, payloadSHA string, owner openRGBStagePreparation) (result string, returnErr error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	appDir := filepath.Join(final, owner.bundlePath())
	verify := func(root string) error {
		info, err := os.Lstat(root)
		if err != nil || !info.IsDir() || info.Mode().Perm()&0o222 != 0 {
			return errors.Join(errors.New("OpenRGB published stage root is not read-only"), err)
		}
		if _, err := verifyOpenRGBReleaseDir(ctx, filepath.Join(root, owner.bundlePath()), treeSHA, payloadSHA, owner.payloadPath(), owner.verifySeal); err != nil {
			return err
		}
		if owner.profile != nil {
			if err := owner.profile.verifyServingStageLayout(ctx, root); err != nil {
				return err
			}
			if err := owner.profile.verifyServingModulesAbsent(ctx, filepath.Join(root, owner.bundlePath())); err != nil {
				return err
			}
			if err := owner.profile.verifyServingProfile(ctx, root, envelope); err != nil {
				return err
			}
			if owner.profile.OS == "darwin" {
				if _, err := owner.profile.darwinLoaderSystemImages(ctx, filepath.Join(root, owner.bundlePath())); err != nil {
					return err
				}
			}
		}
		archive := envelope
		archive.Path = filepath.Join(root, openRGBSealedEnvelope)
		return verifyOpenRGBStageEnvelope(ctx, archive)
	}
	if _, err := os.Lstat(final); err == nil {
		if err := verify(final); err != nil {
			return "", fmt.Errorf("existing OpenRGB stage is unqualified: %w", err)
		}
		if err := owner.syncDir(filepath.Dir(final)); err != nil {
			return "", errors.Join(ErrOpenRGBStageCommitUnknown, err)
		}
		return appDir, nil
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
	snapshot := filepath.Join(pending, openRGBSealedEnvelope)
	if err := copyOpenRGBStageEnvelope(ctx, envelope, snapshot); err != nil {
		return "", err
	}
	file, err := os.Open(snapshot)
	if err != nil {
		return "", err
	}
	err = owner.extract(ctx, file, filepath.Join(pending, owner.bundlePath()))
	_ = file.Close()
	if err != nil {
		return "", err
	}
	if owner.profile != nil {
		if err := owner.profile.deriveServingTree(ctx, filepath.Join(pending, owner.bundlePath())); err != nil {
			return "", err
		}
		if err := owner.profile.writeServingProfile(ctx, pending, envelope); err != nil {
			return "", err
		}
	}
	if err := os.Chmod(snapshot, 0o444); err != nil {
		return "", err
	}
	if err := owner.seal(ctx, pending); err != nil {
		return "", err
	}
	if err := verify(pending); err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := owner.publish(pending, final); err != nil {
		return "", err
	}
	if err := owner.syncDir(filepath.Dir(final)); err != nil {
		return "", errors.Join(ErrOpenRGBStageCommitUnknown, err)
	}
	return appDir, nil
}

func copyOpenRGBStageEnvelope(ctx context.Context, envelope ManagedServiceArtifact, destination string) error {
	file, err := openOpenRGBSourceFile(envelope.Path, nil)
	if err != nil {
		return err
	}
	defer file.Close()
	write := func(dst io.Writer) error {
		const maxEnvelopeBytes = 64 << 20
		hash := sha256.New()
		size, err := io.Copy(io.MultiWriter(dst, hash), io.LimitReader(&managedUpdateContextReader{ctx: ctx, reader: file}, maxEnvelopeBytes+1))
		if err != nil {
			return err
		}
		if size > maxEnvelopeBytes {
			return errors.New("OpenRGB acquired envelope exceeds the qualified size bound")
		}
		if fmt.Sprintf("%x", hash.Sum(nil)) != envelope.SHA256 {
			return errors.New("OpenRGB envelope changed before its verified stage snapshot")
		}
		return nil
	}
	if runtime.GOOS == "windows" {
		return platform.WriteImmutableNativeFile(ctx, destination, write)
	}
	if runtime.GOOS == "darwin" {
		return config.WritePrivateFileDurableStream(destination, 0o500, config.PrivatePathPolicy{OwnerPrincipal: "uid:0"}, write)
	}
	return writeResourceOwnedFileDurableStream(destination, 0o500, write)
}

func verifyOpenRGBStageEnvelope(ctx context.Context, envelope ManagedServiceArtifact) error {
	info, err := os.Lstat(envelope.Path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o222 != 0 {
		return errors.Join(errors.New("OpenRGB retained envelope is not an exact read-only file"), err)
	}
	file, err := os.Open(envelope.Path)
	if err != nil {
		return err
	}
	defer file.Close()
	digest, err := managedUpdateImageDigest(ctx, file)
	if err != nil || digest != envelope.SHA256 {
		return errors.Join(errors.New("sealed OpenRGB envelope checksum differs from its acquired identity"), err)
	}
	return nil
}

func removeOpenRGBPendingStage(path string) error {
	if _, err := os.Lstat(path); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	if err := verifyOpenRGBPendingOwner(path); err != nil {
		return err
	}
	if err := filepath.WalkDir(path, func(path string, entry os.DirEntry, err error) error {
		if err == nil && entry.IsDir() {
			return os.Chmod(path, 0o700)
		}
		return err
	}); err != nil {
		return err
	}
	return os.RemoveAll(path)
}

func (o openRGBStagePreparation) payloadPath() string {
	if o.payloadRelative != "" {
		return o.payloadRelative
	}
	return openRGBAppDirPayload
}

func (o openRGBStagePreparation) bundlePath() string {
	if o.bundleDirectory != "" {
		return o.bundleDirectory
	}
	return openRGBSealedAppDir
}
