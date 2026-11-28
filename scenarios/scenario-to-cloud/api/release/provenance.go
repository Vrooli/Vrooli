package release

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"

	"scenario-to-cloud/apierrors"

	"github.com/vrooli/vrooli/packages/cloudrelease"
)

// TrustMode is the verification policy a build or verification runs under.
// It is resourcedeployment.ArtifactTrustMode: "development-local" accepts an
// explicitly unsigned release; "production" requires a signature from the
// configured release authority.
type TrustMode = resourcedeployment.ArtifactTrustMode

// Trust modes.
const (
	TrustDevelopmentLocal = resourcedeployment.ArtifactTrustDevelopmentLocal
	TrustProduction       = resourcedeployment.ArtifactTrustProduction
)

// TrustModeEnv is the configuration variable the trust mode is resolved from.
const TrustModeEnv = "SCENARIO_TO_CLOUD_ARTIFACT_TRUST"

// DefaultPublicKeyRelPath is the repository-visible trust anchor the release
// authority publishes.
const DefaultPublicKeyRelPath = "install/vrooli-release.pub"

// ResolveTrustMode reads the trust mode from configuration. Unset means
// development-local; an unknown value is refused rather than defaulted.
func ResolveTrustMode(lookup func(string) (string, bool)) (TrustMode, error) {
	if lookup == nil {
		lookup = os.LookupEnv
	}
	value, ok := lookup(TrustModeEnv)
	if !ok || strings.TrimSpace(value) == "" {
		return TrustDevelopmentLocal, nil
	}
	mode := TrustMode(strings.TrimSpace(value))
	if err := mode.Validate(); err != nil {
		return "", apierrors.Newf(apierrors.CodeInvalidRequest, "%s=%q: %v", TrustModeEnv, value, err)
	}
	return mode, nil
}

// Signer produces a detached signature for a staged release manifest. The
// stage directory holds a resource-deployment release-manifest.json listing
// the artifacts to sign; the signer writes release-manifest.sig.json beside
// it. Private keys never enter this module.
type Signer interface {
	Sign(ctx context.Context, stageDir string) (resourcedeployment.ReleaseSignature, error)
}

// ArgvSigner signs through the control plane's release authority
// (`vrooli release-authority sign --stage <dir> --json`). The authority holds
// the key in the native credential store and refuses a stage whose artifact
// hashes do not match, so the cloud side never handles key material.
type ArgvSigner struct {
	// Vrooli is the control-plane executable; defaults to "vrooli" on PATH.
	Vrooli string
	// RepoRoot is the working directory (the authority resolves the trust
	// anchor relative to it).
	RepoRoot string
}

// Sign implements Signer.
func (s ArgvSigner) Sign(ctx context.Context, stageDir string) (resourcedeployment.ReleaseSignature, error) {
	binary := s.Vrooli
	if strings.TrimSpace(binary) == "" {
		binary = "vrooli"
	}
	cmd := exec.CommandContext(ctx, binary, "release-authority", "sign", "--stage", stageDir, "--json")
	cmd.Dir = s.RepoRoot
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return resourcedeployment.ReleaseSignature{}, fmt.Errorf("release-authority sign: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	var envelope resourcedeployment.ReleaseSignature
	if err := json.Unmarshal(bytes.TrimSpace(out), &envelope); err != nil {
		return resourcedeployment.ReleaseSignature{}, fmt.Errorf("release-authority sign: parse envelope: %w", err)
	}
	return envelope, nil
}

// Provenance stage file names.
const (
	provenanceManifestFile  = "release-manifest.json"
	provenanceSignatureFile = "release-manifest.sig.json"
	provenanceRole          = "cloud-release-manifest"
	provenanceUpstream      = "scenario-to-cloud release build"
)

// writeProvenanceStage lays out the signing stage: a copy of the cloud
// manifest and a resource-deployment manifest that pins its sha256. Signing
// the pin signs, transitively, every digest the cloud manifest carries
// (bundle, native CLI, closure, configuration, release digest).
func writeProvenanceStage(releaseDir string, manifestBytes []byte) (string, error) {
	stage := filepath.Join(releaseDir, cloudrelease.ProvenanceDirName)
	if err := os.MkdirAll(stage, 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(stage, cloudrelease.ProvenanceManifestCopy), manifestBytes, 0o644); err != nil {
		return "", err
	}
	sum := sha256.Sum256(manifestBytes)
	rd := resourcedeployment.ReleaseManifest{SchemaVersion: "v1", Artifacts: []resourcedeployment.ReleaseArtifact{{
		Name: cloudrelease.ProvenanceManifestCopy, SHA256: hex.EncodeToString(sum[:]), Role: provenanceRole, UpstreamProvenance: provenanceUpstream,
	}}}
	if _, err := rd.CanonicalBytes(); err != nil {
		return "", err
	}
	raw, err := json.MarshalIndent(rd, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(stage, provenanceManifestFile), append(raw, '\n'), 0o644); err != nil {
		return "", err
	}
	return stage, nil
}

// verifyProvenance checks the signing stage against the trust mode. In
// production it requires a valid signature by the trusted key and that the
// signed copy equals the release manifest byte for byte; in development-local
// it requires the manifest to declare the unsigned policy explicitly.
func verifyProvenance(releaseDir string, manifest cloudrelease.Manifest, manifestBytes []byte, mode TrustMode, publicKeyPath string) (keyID string, err error) {
	stage := filepath.Join(releaseDir, cloudrelease.ProvenanceDirName)
	switch mode {
	case TrustDevelopmentLocal:
		if manifest.Provenance.Policy != cloudrelease.PolicyDevelopmentLocalUnsigned && manifest.Provenance.Policy != cloudrelease.PolicyProductionSigned {
			return "", fmt.Errorf("provenance policy %q is not declared", manifest.Provenance.Policy)
		}
		if manifest.Provenance.Policy == cloudrelease.PolicyDevelopmentLocalUnsigned {
			return "", nil
		}
		// A signed release verifies in development too; a broken signature
		// is a fault, not something to wave through.
		fallthrough
	case TrustProduction:
		if manifest.Provenance.Policy != cloudrelease.PolicyProductionSigned {
			return "", fmt.Errorf("release declares provenance policy %q; production requires %q", manifest.Provenance.Policy, cloudrelease.PolicyProductionSigned)
		}
		_, signature, err := resourcedeployment.VerifyReleaseDirectory(stage, TrustProduction, publicKeyPath)
		if err != nil {
			return "", err
		}
		signed, err := os.ReadFile(filepath.Join(stage, cloudrelease.ProvenanceManifestCopy))
		if err != nil {
			return "", err
		}
		if !bytes.Equal(signed, manifestBytes) {
			return "", fmt.Errorf("signed manifest copy does not match %s", cloudrelease.ManifestFileName)
		}
		return signature.KeyID, nil
	default:
		return "", fmt.Errorf("unknown trust mode %q", mode)
	}
}
