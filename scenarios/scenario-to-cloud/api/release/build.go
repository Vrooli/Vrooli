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
	if in.Closure != nil {
		in.ClosureDigest = in.Closure.Digest
	}
	if in.ClosureDigest == "" {
		in.ClosureDigest = in.Manifest.Dependencies.ClosureDigest
	}
	if strings.TrimSpace(in.ClosureDigest) == "" {
		return apierrors.New(apierrors.CodeClosureUnavailable, "release build requires a closure digest; resolve the closure first")
	}
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
	return nil
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
	cleanup := func() { _ = os.RemoveAll(work) }

	// 1. Bundle (deterministic tar.gz) and its content manifest.
	spec, err := bundle.MiniVrooliBundleSpec(in.RepoRoot, in.Manifest)
	if err != nil {
		cleanup()
		return Release{}, apierrors.Internal("resolve bundle spec", err)
	}
	artifact, err := bundle.BuildMiniVrooliBundle(in.RepoRoot, work, in.Manifest)
	if err != nil {
		cleanup()
		return Release{}, apierrors.Internal("build bundle", err)
	}
	bundlePath := filepath.Join(work, cloudrelease.BundleFileName)
	if err := os.Rename(artifact.Path, bundlePath); err != nil {
		cleanup()
		return Release{}, apierrors.Internal("place bundle", err)
	}
	content, err := bundle.ContentManifest(in.RepoRoot, spec)
	if err != nil {
		cleanup()
		return Release{}, apierrors.Internal("content manifest", err)
	}

	// 2. Native control plane for the target platform.
	nativePath := filepath.Join(work, cloudrelease.NativeCLIFileName(in.Platform.GOOS, in.Platform.GOARCH))
	native, repro, err := buildNativeCLI(ctx, in.RepoRoot, in.NativeCLI, in.Platform, nativePath)
	if err != nil {
		cleanup()
		return Release{}, apierrors.Internal("build native control plane", err)
	}

	// 3. Configuration digest and release identity.
	configurationDigest, err := ConfigurationDigest(in.Manifest)
	if err != nil {
		cleanup()
		return Release{}, apierrors.Internal("configuration digest", err)
	}
	policy := cloudrelease.PolicyDevelopmentLocalUnsigned
	if in.TrustMode == TrustProduction {
		policy = cloudrelease.PolicyProductionSigned
	}
	manifest := cloudrelease.Manifest{
		SchemaVersion:       cloudrelease.ManifestSchemaVersion,
		BundleSHA256:        artifact.Sha256,
		NativeCLI:           cloudrelease.NativeCLI{SHA256: native.SHA256, GOOS: in.Platform.GOOS, GOARCH: in.Platform.GOARCH},
		ClosureDigest:       in.ClosureDigest,
		ConfigurationDigest: configurationDigest,
		Provenance:          cloudrelease.Provenance{Builder: in.Builder, Policy: policy},
		Limits:              cloudrelease.DefaultLimits(),
	}
	manifest.ReleaseDigest, err = cloudrelease.ComputeReleaseDigest(manifest)
	if err != nil {
		cleanup()
		return Release{}, apierrors.Internal("release digest", err)
	}
	now := in.Now().UTC()

	// An identical release already stored is the same release.
	if store.Complete(manifest.ReleaseDigest) {
		cleanup()
		existing, err := store.Load(manifest.ReleaseDigest)
		if err != nil {
			return Release{}, err
		}
		if _, err := store.Protect(existing.Digest, in.Manifest.Scenario.ID, in.LeaseTTL, now); err != nil {
			return Release{}, apierrors.Internal("renew release lease", err)
		}
		return existing, nil
	}

	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		cleanup()
		return Release{}, apierrors.Internal("encode release manifest", err)
	}
	manifestBytes = append(manifestBytes, '\n')
	if err := os.WriteFile(filepath.Join(work, cloudrelease.ManifestFileName), manifestBytes, 0o644); err != nil {
		cleanup()
		return Release{}, apierrors.Internal("write release manifest", err)
	}

	// 4. Provenance: sign in production, declare unsigned otherwise.
	provenance := domain.ReleaseProvenance{Policy: policy, TrustMode: string(in.TrustMode)}
	if in.TrustMode == TrustProduction {
		stage, err := writeProvenanceStage(work, manifestBytes)
		if err != nil {
			cleanup()
			return Release{}, apierrors.Internal("write provenance stage", err)
		}
		signature, err := in.Signer.Sign(ctx, stage)
		if err != nil {
			cleanup()
			return Release{}, apierrors.Newf(apierrors.CodeReleaseVerificationFailed, "sign release manifest: %v", err).WithDetail("reason", "signing_failed")
		}
		keyID, err := verifyProvenance(work, manifest, manifestBytes, TrustProduction, in.PublicKeyPath)
		if err != nil {
			cleanup()
			return Release{}, apierrors.Newf(apierrors.CodeReleaseVerificationFailed, "signature does not verify against %s: %v", in.PublicKeyPath, err).WithDetail("reason", "untrusted_signer")
		}
		if keyID != signature.KeyID {
			cleanup()
			return Release{}, apierrors.New(apierrors.CodeReleaseVerificationFailed, "signer key id does not match the verified signature").WithDetail("reason", "untrusted_signer")
		}
		provenance.SignerKeyID = keyID
		provenance.SignatureFile = filepath.ToSlash(filepath.Join(cloudrelease.ProvenanceDirName, provenanceSignatureFile))
	}

	// 5. inputs.json: every material input, references only.
	inputs := buildInputsRecord(ctx, in, manifest, artifact, content, spec, native, repro, provenance, now)
	inputsBytes, err := json.MarshalIndent(inputs, "", "  ")
	if err != nil {
		cleanup()
		return Release{}, apierrors.Internal("encode inputs", err)
	}
	if err := os.WriteFile(filepath.Join(work, cloudrelease.InputsFileName), append(inputsBytes, '\n'), 0o644); err != nil {
		cleanup()
		return Release{}, apierrors.Internal("write inputs", err)
	}

	// 6. Promote: rename into place, then the completion marker last.
	final, _ := store.Path(manifest.ReleaseDigest)
	if err := os.RemoveAll(final); err != nil {
		cleanup()
		return Release{}, apierrors.Internal("clear incomplete release", err)
	}
	if err := os.Rename(work, final); err != nil {
		cleanup()
		return Release{}, apierrors.Internal("promote release", err)
	}
	if err := os.WriteFile(filepath.Join(final, cloudrelease.CompleteMarker), []byte(now.Format(time.RFC3339Nano)+"\n"), 0o644); err != nil {
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
