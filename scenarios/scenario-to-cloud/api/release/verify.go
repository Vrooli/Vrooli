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
	fail := func(id string, err *apierrors.Error) (VerifyReport, error) {
		report.Checks = append(report.Checks, Check{ID: id, Status: "failed", Code: err.Code})
		return report, err.WithDetail("check", id)
	}
	pass := func(id string) { report.Checks = append(report.Checks, Check{ID: id, Status: "passed"}) }
	refuse := func(reason, format string, args ...any) *apierrors.Error {
		return apierrors.Newf(apierrors.CodeReleaseVerificationFailed, format, args...).WithDetail("reason", reason)
	}

	if opts.TrustMode == "" {
		mode, err := ResolveTrustMode(nil)
		if err != nil {
			return report, err
		}
		opts.TrustMode = mode
	} else if err := opts.TrustMode.Validate(); err != nil {
		return report, apierrors.Newf(apierrors.CodeInvalidRequest, "%v", err)
	}
	if opts.PublicKeyPath == "" && opts.RepoRoot != "" {
		opts.PublicKeyPath = filepath.Join(opts.RepoRoot, DefaultPublicKeyRelPath)
	}
	report.TrustMode = string(opts.TrustMode)

	if _, err := os.Stat(filepath.Join(dir, cloudrelease.CompleteMarker)); err != nil {
		return fail("complete", refuse(ReasonIncomplete, "release at %s has no completion marker; it is not a release", dir))
	}
	pass("complete")

	manifestBytes, err := os.ReadFile(filepath.Join(dir, cloudrelease.ManifestFileName))
	if err != nil {
		return fail("manifest", refuse("release_manifest_invalid", "read release manifest: %v", err))
	}
	manifest, err := cloudrelease.ParseManifest(manifestBytes)
	if err != nil {
		return fail("manifest", refuse("release_manifest_invalid", "%v", err))
	}
	report.ReleaseDigest = manifest.ReleaseDigest
	report.Policy = manifest.Provenance.Policy
	pass("manifest")

	expected, err := cloudrelease.ComputeReleaseDigest(manifest)
	if err != nil {
		return fail("release_digest", refuse("release_manifest_invalid", "compute release digest: %v", err))
	}
	if expected != manifest.ReleaseDigest || filepath.Base(dir) != manifest.ReleaseDigest {
		return fail("release_digest", refuse("release_digest_mismatch", "release_digest does not match the manifest identity").WithDetail("declared", manifest.ReleaseDigest).WithDetail("computed", expected))
	}
	pass("release_digest")

	bundleSum, _, err := fileSHA256(filepath.Join(dir, cloudrelease.BundleFileName))
	if err != nil {
		return fail("bundle_sha256", refuse("archive_unreadable", "read bundle: %v", err))
	}
	if bundleSum != manifest.BundleSHA256 {
		return fail("bundle_sha256", refuse("bundle_sha256_mismatch", "bundle sha256 does not match the manifest").WithDetail("declared", manifest.BundleSHA256).WithDetail("observed", bundleSum))
	}
	pass("bundle_sha256")

	if opts.Platform != nil && (opts.Platform.GOOS != manifest.NativeCLI.GOOS || opts.Platform.GOARCH != manifest.NativeCLI.GOARCH) {
		return fail("platform", apierrors.Newf(apierrors.CodeUnsupportedCapability, "release native control plane is %s/%s; the target needs %s", manifest.NativeCLI.GOOS, manifest.NativeCLI.GOARCH, opts.Platform).
			WithDetail("artifact", cloudrelease.NativeCLIFileName(manifest.NativeCLI.GOOS, manifest.NativeCLI.GOARCH)).WithDetail("declared", manifest.NativeCLI.GOOS+"/"+manifest.NativeCLI.GOARCH).WithDetail("required", opts.Platform.String()))
	}
	pass("platform")

	nativeName := cloudrelease.NativeCLIFileName(manifest.NativeCLI.GOOS, manifest.NativeCLI.GOARCH)
	nativeSum, _, err := fileSHA256(filepath.Join(dir, nativeName))
	if err != nil {
		return fail("native_cli", refuse("native_cli_mismatch", "read %s: %v", nativeName, err).WithDetail("artifact", nativeName))
	}
	if nativeSum != manifest.NativeCLI.SHA256 {
		return fail("native_cli", refuse("native_cli_mismatch", "%s sha256 does not match native_cli.sha256", nativeName).WithDetail("artifact", nativeName).WithDetail("declared", manifest.NativeCLI.SHA256).WithDetail("observed", nativeSum))
	}
	pass("native_cli")

	budget := opts.TimeBudget
	if budget <= 0 {
		budget = DefaultVerifyBudget
	}
	deadline := time.Now().Add(budget)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}
	summary, err := binaryfetch.InspectArchiveBounded(filepath.Join(dir, cloudrelease.BundleFileName), cloudrelease.ArchiveFormat, manifest.ExtractOptions(deadline))
	report.Archive = summary
	if err != nil {
		reason := "archive_unreadable"
		var archiveErr *binaryfetch.ArchiveError
		if errors.As(err, &archiveErr) {
			reason = "archive_" + string(archiveErr.Violation)
		}
		return fail("archive_bounds", refuse(reason, "%v", err))
	}
	pass("archive_bounds")

	keyID, err := verifyProvenance(dir, manifest, manifestBytes, opts.TrustMode, opts.PublicKeyPath)
	if err != nil {
		return fail("provenance", refuse("untrusted_provenance", "%v", err).WithDetail("policy", manifest.Provenance.Policy).WithDetail("trust_mode", string(opts.TrustMode)))
	}
	report.SignerKeyID = keyID
	pass("provenance")
	report.Verified = true
	return report, nil
}
