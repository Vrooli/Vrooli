package release

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/packages/cloudrelease"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/bundle"
	"scenario-to-cloud/domain"
)

// DefaultBuilderIdentity names this builder in provenance records.
const DefaultBuilderIdentity = "scenario-to-cloud"

// BuildInputs are the pinned inputs of one release build.
type BuildInputs struct {
	// RepoRoot is the Vrooli repository the bundle and binary are built from.
	RepoRoot string
	// StoreDir is the bundle store directory; releases live in releases/.
	StoreDir string
	// Manifest is the cloud manifest. Secrets are never written anywhere.
	Manifest domain.CloudManifest
	// Closure is the resolved closure document when available; its digest
	// wins over ClosureDigest and Manifest.Dependencies.ClosureDigest.
	Closure *domain.Closure
	// ClosureDigest is used when Closure is nil.
	ClosureDigest string
	// Platform is the target control-plane platform.
	Platform Platform
	// NativeCLI pins the control-plane build.
	NativeCLI NativeCLIOptions
	// TrustMode is the provenance policy; zero resolves from configuration.
	TrustMode TrustMode
	// Signer is required for production; ignored in development-local.
	Signer Signer
	// PublicKeyPath is the trust anchor a production signature is checked
	// against right after signing; defaults to RepoRoot/install/vrooli-release.pub.
	PublicKeyPath string
	// Builder is the builder identity; defaults to DefaultBuilderIdentity.
	Builder string
	// LeaseTTL protects the release from GC; defaults to DefaultLeaseTTL.
	LeaseTTL time.Duration
	// Now is the clock; defaults to time.Now.
	Now func() time.Time
}

func (in *BuildInputs) normalize() error {
	if err := in.validateBuildTarget(); err != nil {
		return err
	}
	if err := in.normalizeClosureDigest(); err != nil {
		return err
	}
	if err := in.normalizeTrustPolicy(); err != nil {
		return err
	}
	in.applyBuildDefaults()
	return nil
}

func (in *BuildInputs) validateBuildTarget() error {
	if strings.TrimSpace(in.RepoRoot) == "" || strings.TrimSpace(in.StoreDir) == "" {
		return apierrors.New(apierrors.CodeInvalidRequest, "release build requires repo_root and store_dir")
	}
	if strings.TrimSpace(in.Manifest.Scenario.ID) == "" {
		return apierrors.New(apierrors.CodeManifestInvalid, "release build requires manifest.scenario.id")
	}
	if !in.Platform.Supported() {
		return apierrors.Newf(apierrors.CodeUnsupportedCapability, "no native control plane can be built for %s", in.Platform).
			WithDetail("artifact", "native_cli").WithDetail("platform", in.Platform.String())
	}
	return nil
}

func (in *BuildInputs) normalizeClosureDigest() error {
	if in.Closure != nil {
		in.ClosureDigest = in.Closure.Digest
	}
	if in.ClosureDigest == "" {
		in.ClosureDigest = in.Manifest.Dependencies.ClosureDigest
	}
	if strings.TrimSpace(in.ClosureDigest) == "" {
		return apierrors.New(apierrors.CodeClosureUnavailable, "release build requires a closure digest; resolve the closure first")
	}
	return nil
}

func (in *BuildInputs) normalizeTrustPolicy() error {
	if in.TrustMode == "" {
		mode, err := ResolveTrustMode(nil)
		if err != nil {
			return err
		}
		in.TrustMode = mode
	} else if err := in.TrustMode.Validate(); err != nil {
		return apierrors.Newf(apierrors.CodeInvalidRequest, "%v", err)
	}
	if in.TrustMode == TrustProduction && in.Signer == nil {
		return apierrors.New(apierrors.CodeReleaseVerificationFailed, "production trust mode requires a release signer; none configured").WithDetail("reason", "signer_unavailable")
	}
	return nil
}

func (in *BuildInputs) applyBuildDefaults() {
	if in.PublicKeyPath == "" {
		in.PublicKeyPath = filepath.Join(in.RepoRoot, DefaultPublicKeyRelPath)
	}
	if in.Builder == "" {
		in.Builder = DefaultBuilderIdentity
	}
	if in.LeaseTTL <= 0 {
		in.LeaseTTL = DefaultLeaseTTL
	}
	if in.Now == nil {
		in.Now = time.Now
	}
}

// Build produces the release artifact set and stores it under
// releases/<release_digest>. An identical release that already exists is
// returned as-is with its lease renewed; nothing is rebuilt twice into the
// same directory. Every failure leaves the store without a partial release.
func Build(ctx context.Context, in BuildInputs) (Release, error) {
	if err := in.normalize(); err != nil {
		return Release{}, err
	}
	store := Store{Dir: in.StoreDir}
	if err := os.MkdirAll(store.ReleasesDir(), 0o755); err != nil {
		return Release{}, apierrors.Internal("create releases dir", err)
	}
	work, err := buildingDir(store.ReleasesDir())
	if err != nil {
		return Release{}, apierrors.Internal("create build dir", err)
	}
	defer func() { _ = os.RemoveAll(work) }()
	return buildReleaseInDirectory(ctx, in, store, work)
}

type releaseBuildArtifacts struct {
	spec    bundle.MiniBundleSpec
	bundle  domain.BundleArtifact
	content []bundle.ContentEntry
	native  domain.ReleaseNativeCLI
	repro   domain.ReleaseReproducibility
}

func buildReleaseInDirectory(ctx context.Context, in BuildInputs, store Store, work string) (Release, error) {
	artifacts, err := buildReleaseArtifacts(ctx, in, work)
	if err != nil {
		return Release{}, err
	}
	manifest, err := releaseManifest(in, artifacts)
	if err != nil {
		return Release{}, err
	}
	now := in.Now().UTC()
	if store.Complete(manifest.ReleaseDigest) {
		return renewExistingRelease(store, manifest.ReleaseDigest, in, now)
	}
	manifestBytes, err := writeReleaseManifest(work, manifest)
	if err != nil {
		return Release{}, err
	}
	provenance, err := writeReleaseProvenance(ctx, in, work, manifest, manifestBytes)
	if err != nil {
		return Release{}, err
	}
	if err := writeReleaseInputs(ctx, in, manifest, artifacts, provenance, now, work); err != nil {
		return Release{}, err
	}
	return promoteRelease(store, manifest, in, work, now)
}

func buildReleaseArtifacts(ctx context.Context, in BuildInputs, work string) (releaseBuildArtifacts, error) {
	spec, err := bundle.MiniVrooliBundleSpec(in.RepoRoot, in.Manifest)
	if err != nil {
		return releaseBuildArtifacts{}, apierrors.Internal("resolve bundle spec", err)
	}
	built, err := bundle.BuildMiniVrooliBundle(in.RepoRoot, work, in.Manifest)
	if err != nil {
		return releaseBuildArtifacts{}, apierrors.Internal("build bundle", err)
	}
	bundlePath := filepath.Join(work, cloudrelease.BundleFileName)
	if err := os.Rename(built.Path, bundlePath); err != nil {
		return releaseBuildArtifacts{}, apierrors.Internal("place bundle", err)
	}
	content, err := bundle.ContentManifest(in.RepoRoot, spec)
	if err != nil {
		return releaseBuildArtifacts{}, apierrors.Internal("content manifest", err)
	}
	nativePath := filepath.Join(work, cloudrelease.NativeCLIFileName(in.Platform.GOOS, in.Platform.GOARCH))
	native, repro, err := buildNativeCLI(ctx, in.RepoRoot, in.NativeCLI, in.Platform, nativePath)
	if err != nil {
		return releaseBuildArtifacts{}, apierrors.Internal("build native control plane", err)
	}
	return releaseBuildArtifacts{spec: spec, bundle: built, content: content, native: native, repro: repro}, nil
}

func releaseManifest(in BuildInputs, artifacts releaseBuildArtifacts) (cloudrelease.Manifest, error) {
	configurationDigest, err := ConfigurationDigest(in.Manifest)
	if err != nil {
		return cloudrelease.Manifest{}, apierrors.Internal("configuration digest", err)
	}
	policy := releasePolicy(in.TrustMode)
	manifest := cloudrelease.Manifest{
		SchemaVersion: cloudrelease.ManifestSchemaVersion,
		BundleSHA256:  artifacts.bundle.Sha256,
		NativeCLI:     cloudrelease.NativeCLI{SHA256: artifacts.native.SHA256, GOOS: in.Platform.GOOS, GOARCH: in.Platform.GOARCH},
		ClosureDigest: in.ClosureDigest, ConfigurationDigest: configurationDigest,
		Provenance: cloudrelease.Provenance{Builder: in.Builder, Policy: policy}, Limits: cloudrelease.DefaultLimits(),
	}
	manifest.ReleaseDigest, err = cloudrelease.ComputeReleaseDigest(manifest)
	if err != nil {
		return cloudrelease.Manifest{}, apierrors.Internal("release digest", err)
	}
	return manifest, nil
}

func releasePolicy(mode TrustMode) string {
	if mode == TrustProduction {
		return cloudrelease.PolicyProductionSigned
	}
	return cloudrelease.PolicyDevelopmentLocalUnsigned
}

func renewExistingRelease(store Store, digest string, in BuildInputs, now time.Time) (Release, error) {
	existing, err := store.Load(digest)
	if err != nil {
		return Release{}, err
	}
	if _, err := store.Protect(existing.Digest, in.Manifest.Scenario.ID, in.LeaseTTL, now); err != nil {
		return Release{}, apierrors.Internal("renew release lease", err)
	}
	return existing, nil
}

func writeReleaseManifest(work string, manifest cloudrelease.Manifest) ([]byte, error) {
	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, apierrors.Internal("encode release manifest", err)
	}
	encoded = append(encoded, '\n')
	if err := os.WriteFile(filepath.Join(work, cloudrelease.ManifestFileName), encoded, 0o644); err != nil {
		return nil, apierrors.Internal("write release manifest", err)
	}
	return encoded, nil
}

func writeReleaseProvenance(ctx context.Context, in BuildInputs, work string, manifest cloudrelease.Manifest, manifestBytes []byte) (domain.ReleaseProvenance, error) {
	policy := releasePolicy(in.TrustMode)
	provenance := domain.ReleaseProvenance{Policy: policy, TrustMode: string(in.TrustMode)}
	if in.TrustMode != TrustProduction {
		return provenance, nil
	}
	stage, err := writeProvenanceStage(work, manifestBytes)
	if err != nil {
		return domain.ReleaseProvenance{}, apierrors.Internal("write provenance stage", err)
	}
	signature, err := in.Signer.Sign(ctx, stage)
	if err != nil {
		return domain.ReleaseProvenance{}, apierrors.Newf(apierrors.CodeReleaseVerificationFailed, "sign release manifest: %v", err).WithDetail("reason", "signing_failed")
	}
	keyID, err := verifyProvenance(work, manifest, manifestBytes, TrustProduction, in.PublicKeyPath)
	if err != nil {
		return domain.ReleaseProvenance{}, apierrors.Newf(apierrors.CodeReleaseVerificationFailed, "signature does not verify against %s: %v", in.PublicKeyPath, err).WithDetail("reason", "untrusted_signer")
	}
	if keyID != signature.KeyID {
		return domain.ReleaseProvenance{}, apierrors.New(apierrors.CodeReleaseVerificationFailed, "signer key id does not match the verified signature").WithDetail("reason", "untrusted_signer")
	}
	provenance.SignerKeyID = keyID
	provenance.SignatureFile = filepath.ToSlash(filepath.Join(cloudrelease.ProvenanceDirName, provenanceSignatureFile))
	return provenance, nil
}

func writeReleaseInputs(ctx context.Context, in BuildInputs, manifest cloudrelease.Manifest, artifacts releaseBuildArtifacts, provenance domain.ReleaseProvenance, now time.Time, work string) error {
	inputs := buildInputsRecord(ctx, in, manifest, artifacts.bundle, artifacts.content, artifacts.spec, artifacts.native, artifacts.repro, provenance, now)
	encoded, err := json.MarshalIndent(inputs, "", "  ")
	if err != nil {
		return apierrors.Internal("encode inputs", err)
	}
	if err := os.WriteFile(filepath.Join(work, cloudrelease.InputsFileName), append(encoded, '\n'), 0o644); err != nil {
		return apierrors.Internal("write inputs", err)
	}
	return nil
}

func promoteRelease(store Store, manifest cloudrelease.Manifest, in BuildInputs, work string, now time.Time) (Release, error) {
	final, _ := store.Path(manifest.ReleaseDigest)
	if err := os.RemoveAll(final); err != nil {
		return Release{}, apierrors.Internal("clear incomplete release", err)
	}
	if err := os.Rename(work, final); err != nil {
		return Release{}, apierrors.Internal("promote release", err)
	}
	marker := []byte(now.Format(time.RFC3339Nano) + "\n")
	if err := os.WriteFile(filepath.Join(final, cloudrelease.CompleteMarker), marker, 0o644); err != nil {
		return Release{}, apierrors.Internal("write completion marker", err)
	}
	if _, err := store.Protect(manifest.ReleaseDigest, in.Manifest.Scenario.ID, in.LeaseTTL, now); err != nil {
		return Release{}, apierrors.Internal("claim release lease", err)
	}
	return store.Load(manifest.ReleaseDigest)
}

func buildInputsRecord(ctx context.Context, in BuildInputs, manifest cloudrelease.Manifest, artifact domain.BundleArtifact, content []bundle.ContentEntry, spec bundle.MiniBundleSpec, native domain.ReleaseNativeCLI, repro domain.ReleaseReproducibility, provenance domain.ReleaseProvenance, now time.Time) domain.ReleaseInputs {
	limitations := []string{}
	commit, dirty, dirtyPaths, sourceLimits := sourceFacts(ctx, in.RepoRoot, spec.IncludeRoots)
	limitations = append(limitations, sourceLimits...)
	artifacts, artifactLimits := resourceArtifacts(in.RepoRoot, in.Closure)
	limitations = append(limitations, artifactLimits...)
	if dirty {
		limitations = append(limitations, fmt.Sprintf("working tree differs from commit %s in %d included path(s); the content manifest digest is the source identity, not the commit", short(commit), dirtyPaths))
	}
	if repro.NativeCLI != domain.ReproducibilityReproducible {
		limitations = append(limitations, "native_cli reproducibility: "+repro.NativeCLI+": "+repro.Reason)
	}
	limitations = append(limitations, "the builder reads the working tree in place; no isolated source snapshot is claimed")
	if provenance.Policy == cloudrelease.PolicyDevelopmentLocalUnsigned {
		limitations = append(limitations, "release manifest is unsigned under development-local trust; production verification will refuse it")
	}
	goFlags := envValue(buildEnv(in.Platform), "GOFLAGS")
	owner := currentOwner()
	return domain.ReleaseInputs{
		SchemaVersion: domain.ReleaseInputsSchemaVersion,
		ReleaseDigest: manifest.ReleaseDigest,
		BuiltAt:       now.Format(time.RFC3339Nano),
		Builder:       domain.ReleaseBuilder{Identity: in.Builder, Node: owner.Node, User: owner.User},
		Source: domain.ReleaseSource{
			Commit: commit, Dirty: dirty, DirtyPaths: dirtyPaths, Snapshot: domain.ReleaseSnapshotWorkingTree,
			ContentManifestSHA256: bundle.ContentManifestDigest(content), FileCount: len(content),
			IncludeRoots: append([]string{}, spec.IncludeRoots...), Excludes: append([]string{}, spec.Excludes...),
		},
		Bundle: domain.ReleaseBundle{FileName: cloudrelease.BundleFileName, SHA256: artifact.Sha256, SizeBytes: artifact.SizeBytes},
		Dependencies: domain.ReleaseDependencies{
			ClosureDigest: in.ClosureDigest, AnalyzerTool: in.Manifest.Dependencies.Analyzer.Tool,
			AnalyzerFingerprint: in.Manifest.Dependencies.Analyzer.Fingerprint, AnalyzerGeneratedAt: in.Manifest.Dependencies.Analyzer.GeneratedAt,
			Scenarios: sortedCopy(in.Manifest.Dependencies.Scenarios), Resources: sortedCopy(in.Manifest.Dependencies.Resources),
		},
		NativeCLI:         native,
		ResourceArtifacts: artifacts,
		Toolchain:         domain.ReleaseToolchain{GoVersion: native.GoVersion, HostGOOS: runtime.GOOS, HostGOARCH: runtime.GOARCH, GoFlags: strings.Fields(goFlags)},
		Configuration:     domain.ReleaseConfiguration{Digest: manifest.ConfigurationDigest, Schema: ConfigurationSchema(in.Manifest), Rule: ConfigurationDigestRule},
		Credentials:       credentialRefs(in.Manifest),
		Provenance:        provenance,
		Reproducibility:   repro,
		Limitations:       limitations,
	}
}

func buildingDir(releasesDir string) (string, error) {
	var nonce [6]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return "", err
	}
	dir := filepath.Join(releasesDir, ".building-"+hex.EncodeToString(nonce[:]))
	return dir, os.MkdirAll(dir, 0o755)
}

func sortedCopy(values []string) []string {
	out := append([]string{}, values...)
	sort.Strings(out)
	return out
}

func short(digest string) string {
	if len(digest) > 12 {
		return digest[:12]
	}
	return digest
}
