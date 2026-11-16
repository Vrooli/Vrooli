package release

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/vrooli/binaryfetch"
	"github.com/vrooli/vrooli/packages/cloudrelease"

	"scenario-to-cloud/apierrors"
)

// DefaultVerifyBudget bounds the archive dry pass.
const DefaultVerifyBudget = 10 * time.Minute

// VerifyOptions select the policy a release is verified under.
type VerifyOptions struct {
	// TrustMode is the provenance policy; zero resolves from configuration.
	TrustMode TrustMode
	// PublicKeyPath is the trust anchor; defaults to repoRoot-relative
	// install/vrooli-release.pub when RepoRoot is set.
	PublicKeyPath string
	// RepoRoot locates the default trust anchor.
	RepoRoot string
	// Platform, when set, is the target platform the release must serve. A
	// mismatch is unsupported_capability naming the artifact.
	Platform *Platform
	// TimeBudget bounds the archive dry pass; defaults to DefaultVerifyBudget.
	TimeBudget time.Duration
}

// Check is one verification step's verdict.
type Check struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Code   string `json:"code,omitempty"`
}

// VerifyReport lists every check that ran. Verified is true only when all
// passed; the error returned beside it names the first refusal.
type VerifyReport struct {
	ReleaseDigest string                     `json:"release_digest"`
	Verified      bool                       `json:"verified"`
	Checks        []Check                    `json:"checks"`
	Policy        string                     `json:"policy"`
	TrustMode     string                     `json:"trust_mode"`
	SignerKeyID   string                     `json:"signer_key_id,omitempty"`
	Archive       binaryfetch.ArchiveSummary `json:"archive"`
}

// Verify performs every cloud-side release trust check on a stored release
// directory without writing anything: completion marker, manifest shape,
// release digest, bundle sha256, native CLI sha256 and platform, archive
// bounds (dry pass), and provenance under the trust mode. A refusal is a
// typed release_verification_failed (or unsupported_capability for a
// platform mismatch) whose details.check names the failed step.
func Verify(ctx context.Context, dir string, opts VerifyOptions) (VerifyReport, error) {
	report := VerifyReport{Checks: []Check{}}
	mode, err := resolveVerifyTrustMode(opts.TrustMode)
	if err != nil {
		return report, err
	}
	opts.TrustMode = mode
	if opts.PublicKeyPath == "" && opts.RepoRoot != "" {
		opts.PublicKeyPath = filepath.Join(opts.RepoRoot, DefaultPublicKeyRelPath)
	}
	report.TrustMode = string(mode)
	state := verifyState{ctx: ctx, dir: dir, opts: opts, report: report}
	if err := state.run(); err != nil {
		return state.report, err
	}
	return state.report, nil
}

type verifyState struct {
	ctx           context.Context
	dir           string
	opts          VerifyOptions
	report        VerifyReport
	manifest      *cloudrelease.Manifest
	manifestBytes []byte
}

func resolveVerifyTrustMode(mode TrustMode) (TrustMode, error) {
	if mode == "" {
		return ResolveTrustMode(nil)
	}
	if err := mode.Validate(); err != nil {
		return "", apierrors.Newf(apierrors.CodeInvalidRequest, "%v", err)
	}
	return mode, nil
}

func (v *verifyState) run() error {
	checks := []func() error{
		v.checkComplete, v.readManifest, v.checkReleaseDigest,
		v.checkBundle, v.checkPlatform, v.checkNativeCLI,
		v.checkArchiveBounds, v.checkProvenance,
	}
	for _, check := range checks {
		if err := check(); err != nil {
			return err
		}
	}
	v.report.Verified = true
	return nil
}

func (v *verifyState) pass(id string) {
	v.report.Checks = append(v.report.Checks, Check{ID: id, Status: "passed"})
}

func (v *verifyState) fail(id string, err *apierrors.Error) error {
	v.report.Checks = append(v.report.Checks, Check{ID: id, Status: "failed", Code: err.Code})
	return err.WithDetail("check", id)
}

func refuse(reason, format string, args ...any) *apierrors.Error {
	return apierrors.Newf(apierrors.CodeReleaseVerificationFailed, format, args...).WithDetail("reason", reason)
}

func (v *verifyState) checkComplete() error {
	if _, err := os.Stat(filepath.Join(v.dir, cloudrelease.CompleteMarker)); err != nil {
		return v.fail("complete", refuse(ReasonIncomplete, "release at %s has no completion marker; it is not a release", v.dir))
	}
	v.pass("complete")
	return nil
}

func (v *verifyState) readManifest() error {
	data, err := os.ReadFile(filepath.Join(v.dir, cloudrelease.ManifestFileName))
	if err != nil {
		return v.fail("manifest", refuse("release_manifest_invalid", "read release manifest: %v", err))
	}
	manifest, err := cloudrelease.ParseManifest(data)
	if err != nil {
		return v.fail("manifest", refuse("release_manifest_invalid", "%v", err))
	}
	v.manifestBytes, v.manifest = data, &manifest
	v.report.ReleaseDigest = manifest.ReleaseDigest
	v.report.Policy = manifest.Provenance.Policy
	v.pass("manifest")
	return nil
}

func (v *verifyState) checkReleaseDigest() error {
	expected, err := cloudrelease.ComputeReleaseDigest(*v.manifest)
	if err != nil {
		return v.fail("release_digest", refuse("release_manifest_invalid", "compute release digest: %v", err))
	}
	if expected != v.manifest.ReleaseDigest || filepath.Base(v.dir) != v.manifest.ReleaseDigest {
		return v.fail("release_digest", refuse("release_digest_mismatch", "release_digest does not match the manifest identity").WithDetail("declared", v.manifest.ReleaseDigest).WithDetail("computed", expected))
	}
	v.pass("release_digest")
	return nil
}

func (v *verifyState) checkBundle() error {
	bundleSum, _, err := fileSHA256(filepath.Join(v.dir, cloudrelease.BundleFileName))
	if err != nil {
		return v.fail("bundle_sha256", refuse("archive_unreadable", "read bundle: %v", err))
	}
	if bundleSum != v.manifest.BundleSHA256 {
		return v.fail("bundle_sha256", refuse("bundle_sha256_mismatch", "bundle sha256 does not match the manifest").WithDetail("declared", v.manifest.BundleSHA256).WithDetail("observed", bundleSum))
	}
	v.pass("bundle_sha256")
	return nil
}

func (v *verifyState) checkPlatform() error {
	platform := v.opts.Platform
	if platform != nil && (platform.GOOS != v.manifest.NativeCLI.GOOS || platform.GOARCH != v.manifest.NativeCLI.GOARCH) {
		return v.fail("platform", apierrors.Newf(apierrors.CodeUnsupportedCapability, "release native control plane is %s/%s; the target needs %s", v.manifest.NativeCLI.GOOS, v.manifest.NativeCLI.GOARCH, platform).
			WithDetail("artifact", cloudrelease.NativeCLIFileName(v.manifest.NativeCLI.GOOS, v.manifest.NativeCLI.GOARCH)).WithDetail("declared", v.manifest.NativeCLI.GOOS+"/"+v.manifest.NativeCLI.GOARCH).WithDetail("required", platform.String()))
	}
	v.pass("platform")
	return nil
}

func (v *verifyState) checkNativeCLI() error {
	name := cloudrelease.NativeCLIFileName(v.manifest.NativeCLI.GOOS, v.manifest.NativeCLI.GOARCH)
	sum, _, err := fileSHA256(filepath.Join(v.dir, name))
	if err != nil {
		return v.fail("native_cli", refuse("native_cli_mismatch", "read %s: %v", name, err).WithDetail("artifact", name))
	}
	if sum != v.manifest.NativeCLI.SHA256 {
		return v.fail("native_cli", refuse("native_cli_mismatch", "%s sha256 does not match native_cli.sha256", name).WithDetail("artifact", name).WithDetail("declared", v.manifest.NativeCLI.SHA256).WithDetail("observed", sum))
	}
	v.pass("native_cli")
	return nil
}

func (v *verifyState) checkArchiveBounds() error {
	budget := v.opts.TimeBudget
	if budget <= 0 {
		budget = DefaultVerifyBudget
	}
	deadline := time.Now().Add(budget)
	if ctxDeadline, ok := v.ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}
	summary, err := binaryfetch.InspectArchiveBounded(filepath.Join(v.dir, cloudrelease.BundleFileName), cloudrelease.ArchiveFormat, v.manifest.ExtractOptions(deadline))
	v.report.Archive = summary
	if err != nil {
		reason := "archive_unreadable"
		var archiveErr *binaryfetch.ArchiveError
		if errors.As(err, &archiveErr) {
			reason = "archive_" + string(archiveErr.Violation)
		}
		return v.fail("archive_bounds", refuse(reason, "%v", err))
	}
	v.pass("archive_bounds")
	return nil
}

func (v *verifyState) checkProvenance() error {
	keyID, err := verifyProvenance(v.dir, *v.manifest, v.manifestBytes, v.opts.TrustMode, v.opts.PublicKeyPath)
	if err != nil {
		return v.fail("provenance", refuse("untrusted_provenance", "%v", err).WithDetail("policy", v.manifest.Provenance.Policy).WithDetail("trust_mode", string(v.opts.TrustMode)))
	}
	v.report.SignerKeyID = keyID
	v.pass("provenance")
	return nil
}
