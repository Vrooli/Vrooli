package resources

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/vrooli/binaryfetch"
)

// OpenRGB 1.1 Linux amd64. The tree digest uses binaryfetch's whole-tree
// format on the embedded SquashFS at offset 193728. Keep the envelope as the
// governed and retained artifact even when the native owner executes its ELF.
const (
	openRGBAppImageVersion = "1.1"
	openRGBAppImageSHA256  = "d556c2273b0ef863dbee7ffdb438304b382add5d685c93c60758b42de3b0268d"
	openRGBAppDirSHA256    = "2e99a7ae55e1afd217acba433da3b791f6554c2b702b9418a2aab706efffb1b3"
	openRGBPayloadSHA256   = "48739d378cfcd27e8b8c8cfc6d820364dddf31f0a9819065be1a281a4ee98dab"
	openRGBAppDirPayload   = "usr/bin/OpenRGB"
)

type openRGBAppImageStage struct {
	envelope      ManagedServiceArtifact
	appDir        string
	payloadSHA256 string
	payload       os.FileInfo
	profile       openRGBReleaseProfile
}

// verifyOpenRGBAppImageStage is read-only. The acquisition owner supplies the
// approved AppDir path from its content-addressed stage. The native launcher
// must never extract on startup or accept the path from a command argument.
func verifyOpenRGBAppImageStage(ctx context.Context, envelope ManagedServiceArtifact, appDir string) (openRGBAppImageStage, error) {
	profile, profileErr := openRGBReleaseProfileForCurrentHost()
	if profileErr != nil || profile.validate() != nil || envelope.Resource != openRGBResourceName || envelope.Version != openRGBAppImageVersion || !profile.acceptsArtifactSHA256(envelope.SHA256) {
		return openRGBAppImageStage{}, errOpenRGBReleasePayloadUnqualified
	}
	if !filepath.IsAbs(envelope.Path) || filepath.Clean(envelope.Path) != envelope.Path {
		return openRGBAppImageStage{}, errors.New("OpenRGB AppImage requires a canonical retained envelope path")
	}
	if err := profile.verifyServingStageLayout(ctx, filepath.Dir(appDir)); err != nil {
		return openRGBAppImageStage{}, err
	}
	if err := binaryfetch.VerifyArtifactContext(ctx, envelope.Path, "file", envelope.SHA256); err != nil {
		return openRGBAppImageStage{}, fmt.Errorf("verify retained OpenRGB AppImage envelope: %w", err)
	}
	payload, err := verifyOpenRGBReleaseDir(ctx, appDir, profile.servingTreeSHA256(), profile.PayloadSHA256, profile.Payload, verifyOpenRGBAppDirSeal)
	if err != nil {
		return openRGBAppImageStage{}, errors.Join(errOpenRGBReleasePayloadUnqualified, err)
	}
	if err := profile.verifyServingModulesAbsent(ctx, appDir); err != nil {
		return openRGBAppImageStage{}, err
	}
	if err := profile.verifyServingProfile(ctx, filepath.Dir(appDir), envelope); err != nil {
		return openRGBAppImageStage{}, err
	}
	if err := profile.verifyPayloadABI(filepath.Join(appDir, filepath.FromSlash(profile.Payload))); err != nil {
		return openRGBAppImageStage{}, err
	}
	if err := profile.verifyVendorStartupMode(filepath.Join(appDir, filepath.FromSlash(profile.Payload))); err != nil {
		return openRGBAppImageStage{}, err
	}
	if err := profile.verifyBundleABI(ctx, appDir); err != nil {
		return openRGBAppImageStage{}, err
	}
	if profile.OS == "darwin" {
		if _, err := profile.darwinLoaderSystemImages(ctx, appDir); err != nil {
			return openRGBAppImageStage{}, err
		}
	}
	return openRGBAppImageStage{envelope: envelope, appDir: appDir, payloadSHA256: profile.PayloadSHA256, payload: payload, profile: profile}, nil
}

func verifyOpenRGBAppDir(ctx context.Context, appDir, treeSHA, payloadSHA string, verifySeal func(context.Context, string) error) (os.FileInfo, error) {
	return verifyOpenRGBReleaseDir(ctx, appDir, treeSHA, payloadSHA, openRGBAppDirPayload, verifySeal)
}

func verifyOpenRGBReleaseDir(ctx context.Context, appDir, treeSHA, payloadSHA, payloadRelative string, verifySeal func(context.Context, string) error) (os.FileInfo, error) {
	if !filepath.IsAbs(appDir) || filepath.Clean(appDir) != appDir || appDir == string(filepath.Separator) || openRGBStagePathHasUnsafeCharacters(runtime.GOOS, appDir) || !updateDigest(treeSHA) || !updateDigest(payloadSHA) {
		return nil, errors.New("OpenRGB AppDir requires an exact absolute staged path and pinned digests")
	}
	if err := verifySeal(ctx, appDir); err != nil {
		return nil, fmt.Errorf("OpenRGB AppDir is not sealed for native execution: %w", err)
	}
	if err := binaryfetch.VerifyArtifactContext(ctx, appDir, "dir", treeSHA); err != nil {
		return nil, fmt.Errorf("verify complete staged OpenRGB AppDir: %w", err)
	}
	payloadPath := filepath.Join(appDir, filepath.FromSlash(payloadRelative))
	payload, err := os.Lstat(payloadPath)
	if err != nil || !payload.Mode().IsRegular() || runtime.GOOS != "windows" && payload.Mode().Perm()&0o111 == 0 {
		return nil, errors.Join(errors.New("verified OpenRGB AppDir lacks an executable payload file"), err)
	}
	if err := binaryfetch.VerifyArtifactContext(ctx, payloadPath, "file", payloadSHA); err != nil {
		return nil, fmt.Errorf("verify staged OpenRGB serving ELF: %w", err)
	}
	return payload, nil
}

func (s openRGBAppImageStage) servingInvocation(base managedUpdateInvocation, port int) (managedUpdateInvocation, error) {
	if s.payload == nil || base.Path != s.envelope.Path {
		return managedUpdateInvocation{}, errors.New("OpenRGB launch does not name its verified staged envelope")
	}
	if err := verifyOpenRGBExecInvocation(base, port); err != nil {
		return managedUpdateInvocation{}, err
	}
	base.Path = filepath.Join(s.appDir, s.payloadRelative())
	base.Dir = s.appDir
	if s.profile.OS == "linux" || s.profile.OS == "" {
		base.Env = append(slices.Clone(base.Env), "LD_LIBRARY_PATH="+filepath.Join(s.appDir, "usr", "lib"))
	}
	return base, nil
}

func (s openRGBAppImageStage) openPayload() (*os.File, error) {
	if s.payload == nil {
		return nil, errors.New("OpenRGB staged payload is unverified")
	}
	file, err := os.Open(filepath.Join(s.appDir, s.payloadRelative()))
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil || !os.SameFile(s.payload, info) {
		_ = file.Close()
		return nil, errors.Join(errors.New("verified OpenRGB payload inode changed before exec"), err)
	}
	return file, nil
}

func (s openRGBAppImageStage) payloadRelative() string {
	if s.profile.Payload != "" {
		return filepath.FromSlash(s.profile.Payload)
	}
	return openRGBAppDirPayload
}

func openRGBStagePathHasUnsafeCharacters(goos, path string) bool {
	if goos == "windows" {
		// Only the canonical local drive prefix may contain a colon. A remaining
		// colon is an alternate-data-stream or search-path injection candidate.
		if len(path) < 3 || path[1] != ':' || !((path[0] >= 'A' && path[0] <= 'Z') || (path[0] >= 'a' && path[0] <= 'z')) || (path[2] != '\\' && path[2] != '/') {
			return true
		}
		path = path[2:]
	}
	return strings.ContainsAny(path, ":;$\r\n\x00")
}
