package cloudtarget

import (
	"context"
	"errors"
	"io"
	"os"
	"runtime"
	"time"

	"github.com/vrooli/binaryfetch"
	"github.com/vrooli/vrooli/packages/cloudrelease"
)

// ReleaseArchiveFormat is the only bundle format a target accepts.
const ReleaseArchiveFormat = cloudrelease.ArchiveFormat

// Default bounds applied when a manifest omits a limit.
const (
	DefaultMaxEntries       = cloudrelease.DefaultMaxEntries
	DefaultMaxExpandedBytes = cloudrelease.DefaultMaxExpandedBytes
	DefaultMaxEntryBytes    = cloudrelease.DefaultMaxEntryBytes
	DefaultVerifyBudget     = 10 * time.Minute
)

// The manifest shape, its canonical JSON form and the digest rule are owned
// by packages/cloudrelease so the cloud builder (a separate module that cannot
// import internal/) and this verifier share one implementation.
type (
	// Manifest is the release manifest the cloud side publishes beside a bundle.
	Manifest = cloudrelease.Manifest
	// NativeCLI binds the release to one control-plane binary build.
	NativeCLI = cloudrelease.NativeCLI
	// Provenance names the builder and the policy it was verified under.
	Provenance = cloudrelease.Provenance
	// Limits are the archive budgets declared by the release.
	Limits = cloudrelease.Limits
	// ReleaseIdentity is the exact value set the release digest covers.
	ReleaseIdentity = cloudrelease.ReleaseIdentity
)

// ComputeReleaseDigest returns the digest a manifest must carry.
func ComputeReleaseDigest(m Manifest) (string, error) {
	return cloudrelease.ComputeReleaseDigest(m)
}

// LoadManifest reads and shape-checks a manifest without verifying it.
func LoadManifest(path string) (Manifest, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // operator-provided manifest path
	if err != nil {
		return Manifest{}, refuse(CodeManifestInvalid, "read release manifest: %v", err)
	}
	m, err := cloudrelease.ParseManifest(raw)
	if err != nil {
		return Manifest{}, refuse(CodeManifestInvalid, "%v", err)
	}
	return m, nil
}

// VerifyRequest names the artifacts to verify and the binary to compare.
type VerifyRequest struct {
	ManifestPath string
	ArchivePath  string
	// Executable, GOOS and GOARCH default to this process. Tests override them.
	Executable string
	GOOS       string
	GOARCH     string
	TimeBudget time.Duration
}

// Check is one verification step's verdict.
type Check struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Code   string `json:"code,omitempty"`
}

// VerifyReport is the refuse-before-write result.
type VerifyReport struct {
	ReleaseDigest string                     `json:"release_digest"`
	BundleSHA256  string                     `json:"bundle_sha256"`
	NativeCLI     NativeCLI                  `json:"native_cli"`
	Provenance    Provenance                 `json:"provenance"`
	Archive       binaryfetch.ArchiveSummary `json:"archive"`
	Checks        []Check                    `json:"checks"`
	Verified      bool                       `json:"verified"`
}

func (r VerifyReport) details() map[string]any {
	return map[string]any{
		"release_digest": r.ReleaseDigest, "bundle_sha256": r.BundleSHA256, "native_cli": r.NativeCLI,
		"archive": r.Archive, "checks": r.Checks, "verified": r.Verified,
	}
}

// Verify performs every release trust check without writing anything. The
// report lists every check that ran; the error names the first refusal.
func Verify(ctx context.Context, req VerifyRequest) (VerifyReport, Manifest, error) {
	manifest, err := LoadManifest(req.ManifestPath)
	if err != nil {
		return VerifyReport{}, Manifest{}, err
	}
	report := VerifyReport{ReleaseDigest: manifest.ReleaseDigest, BundleSHA256: manifest.BundleSHA256, NativeCLI: manifest.NativeCLI, Provenance: manifest.Provenance}
	record := func(id string, failure *Error) *Error {
		if failure == nil {
			report.Checks = append(report.Checks, Check{ID: id, Status: "passed"})
			return nil
		}
		report.Checks = append(report.Checks, Check{ID: id, Status: "failed", Code: failure.Code})
		return failure
	}

	expected, err := ComputeReleaseDigest(manifest)
	if err != nil {
		return report, manifest, fail(CodeManifestInvalid, "compute release digest: %v", err)
	}
	if expected != manifest.ReleaseDigest {
		return report, manifest, record("release_digest", refuse(CodeReleaseDigestMismatch, "release_digest does not match the manifest identity").
			withDetails(map[string]any{"declared": manifest.ReleaseDigest, "computed": expected}))
	}
	record("release_digest", nil)

	archiveSum, err := fileSHA256(req.ArchivePath)
	if err != nil {
		return report, manifest, record("bundle_sha256", refuse(CodeArchiveUnreadable, "read archive: %v", err))
	}
	if archiveSum != manifest.BundleSHA256 {
		return report, manifest, record("bundle_sha256", refuse(CodeBundleSHA256Mismatch, "archive sha256 does not match bundle_sha256").
			withDetails(map[string]any{"declared": manifest.BundleSHA256, "observed": archiveSum}))
	}
	record("bundle_sha256", nil)

	executable, goos, goarch := req.Executable, req.GOOS, req.GOARCH
	if executable == "" {
		executable, err = os.Executable()
		if err != nil {
			return report, manifest, record("native_cli", refuse(CodeNativeCLIMismatch, "locate running binary: %v", err))
		}
	}
	if goos == "" {
		goos = runtime.GOOS
	}
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	binarySum, err := fileSHA256(executable)
	if err != nil {
		return report, manifest, record("native_cli", refuse(CodeNativeCLIMismatch, "read running binary: %v", err))
	}
	if binarySum != manifest.NativeCLI.SHA256 || goos != manifest.NativeCLI.GOOS || goarch != manifest.NativeCLI.GOARCH {
		return report, manifest, record("native_cli", refuse(CodeNativeCLIMismatch, "running vrooli binary does not match native_cli").
			withDetails(map[string]any{"declared": manifest.NativeCLI, "observed": NativeCLI{SHA256: binarySum, GOOS: goos, GOARCH: goarch}}))
	}
	record("native_cli", nil)

	budget := req.TimeBudget
	if budget <= 0 {
		budget = DefaultVerifyBudget
	}
	deadline := time.Now().Add(budget)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}
	summary, err := binaryfetch.InspectArchiveBounded(req.ArchivePath, ReleaseArchiveFormat, manifest.ExtractOptions(deadline))
	report.Archive = summary
	if err != nil {
		return report, manifest, record("archive_bounds", archiveError(err))
	}
	record("archive_bounds", nil)
	report.Verified = true
	return report, manifest, nil
}

// archiveError maps a binaryfetch refusal onto the target reason codes.
func archiveError(err error) *Error {
	var archiveErr *binaryfetch.ArchiveError
	if !errors.As(err, &archiveErr) {
		return refuse(CodeArchiveUnreadable, "%v", err)
	}
	code := CodeArchiveUnreadable
	switch archiveErr.Violation {
	case binaryfetch.ViolationTraversal:
		code = CodeArchiveTraversal
	case binaryfetch.ViolationSymlinkEscape:
		code = CodeArchiveSymlinkEscape
	case binaryfetch.ViolationTooLarge:
		code = CodeArchiveTooLarge
	case binaryfetch.ViolationEntryLimit:
		code = CodeArchiveEntryLimit
	case binaryfetch.ViolationUnsupportedEntry:
		code = CodeArchiveUnsupportedEntry
	case binaryfetch.ViolationDeadline:
		code = CodeArchiveTimeBudget
	}
	return refuse(code, "%s", archiveErr.Error()).withDetails(map[string]any{"entry": archiveErr.Entry})
}

func copyAll(dst io.Writer, src io.Reader) (int64, error) {
	return io.Copy(dst, src)
}
