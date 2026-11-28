package manifestvalidation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vrooli/cli-core/cliapp"
	repocontract "github.com/vrooli/repo-contract-go"
)

// FilesystemArchitectureEvidence is the production ArchitectureEvidenceProvider.
// It reads a scenario's committed static primitive-evidence artifact — the
// unforgeable side of the verified-primitive contract — WITHOUT ever executing
// the scenario's commands (plan decision D1). The artifact is generated from the
// scenario's own registration metadata (see cliapp.BuildPrimitiveEvidence); here
// we only read it, so validation stays fully static.
//
// The canonical artifact location is the scenario-local generated path
// (.vrooli/generated/cli-primitive-evidence.json, plan decision D2). A scenario
// still mid-migration may carry the artifact at the deprecated pre-migration path
// (cli/primitive-evidence.json, beside the manifest); the provider falls back to
// it so those scenarios keep validating, preferring the canonical path when both
// exist.
//
// It classifies the artifact into three trust states:
//   - present & fresh  → observed primitives are trusted (can verify declared L4);
//   - missing          → no evidence (empty), declared primitives stay unverified;
//   - malformed / stale → an explicit artifact-level finding, evidence ignored.
type FilesystemArchitectureEvidence struct {
	RepoRoot string
}

// NewFilesystemArchitectureEvidence returns a provider rooted at the repo dir.
func NewFilesystemArchitectureEvidence(repoRoot string) *FilesystemArchitectureEvidence {
	return &FilesystemArchitectureEvidence{RepoRoot: repoRoot}
}

// Evidence resolves the static primitive-evidence artifact for a scenario. It
// mirrors the manifest loader's path resolution: an explicit scenario root in ctx
// (WithScenarioPath — used for deep template/generated validation) takes
// precedence over the repo-contract lookup by scenario name.
//
// A missing artifact is NOT an error: it is the honest rollout state (declared
// primitives classify as unverified). A read error other than "not found"
// degrades to empty evidence (the service logs it). A malformed or stale artifact
// returns a non-error ArchitectureEvidence carrying the corresponding Status, so
// the classifier can emit a precise finding.
func (p *FilesystemArchitectureEvidence) Evidence(ctx context.Context, scenario string) (ArchitectureEvidence, error) {
	manifestPath, err := p.manifestPath(ctx, scenario)
	if err != nil {
		return ArchitectureEvidence{}, err
	}
	scenarioRoot := filepath.Dir(filepath.Dir(manifestPath))
	canonicalPath := cliapp.EvidenceArtifactPath(scenarioRoot)
	legacyPath := filepath.Join(filepath.Dir(manifestPath), cliapp.EvidenceArtifactFilename)

	// Prefer the canonical generated location; fall back to the deprecated
	// pre-migration path so a scenario mid-migration still validates. The finding
	// location reflects whichever path the evidence was actually read from.
	artifactPath := canonicalPath
	raw, err := os.ReadFile(canonicalPath)
	if errors.Is(err, os.ErrNotExist) {
		artifactPath = legacyPath
		raw, err = os.ReadFile(legacyPath)
	}
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Missing artifact at both locations: honest no-evidence state. Declared
			// primitives classify as arch.primitive_unverified — never a false L4.
			// Report the canonical path so remediation points at the right place.
			return ArchitectureEvidence{ArtifactPath: canonicalPath}, nil
		}
		return ArchitectureEvidence{ArtifactPath: artifactPath}, fmt.Errorf("read primitive evidence artifact %q: %w", artifactPath, err)
	}

	artifact, parseErr := cliapp.ParsePrimitiveEvidence(raw)
	if parseErr != nil {
		// Present but broken: a gating finding, not a silent degrade.
		return ArchitectureEvidence{
			ArtifactPath: artifactPath,
			Status:       EvidenceArtifactMalformed,
			Detail:       parseErr.Error(),
		}, nil
	}

	// Staleness: the artifact records the manifest hash it was generated against.
	// If the live manifest has changed since, the evidence describes an older
	// command surface and cannot award verified maturity.
	if artifact.ManifestHash == "" {
		return ArchitectureEvidence{
			ArtifactPath: artifactPath,
			Status:       EvidenceArtifactMalformed,
			Detail:       "missing manifest_hash",
		}, nil
	}
	if manifestRaw, mErr := os.ReadFile(manifestPath); mErr == nil {
		if hashHex(manifestRaw) != artifact.ManifestHash {
			return ArchitectureEvidence{ArtifactPath: artifactPath, Status: EvidenceArtifactStale}, nil
		}
	}

	return ArchitectureEvidence{
		Primitives:   artifact.ObservedPrimitives(),
		ArtifactPath: artifactPath,
	}, nil
}

// manifestPath resolves the scenario's cli/manifest.json, honoring an explicit
// scenario root in ctx (WithScenarioPath) exactly like FilesystemManifestLoader.
func (p *FilesystemArchitectureEvidence) manifestPath(ctx context.Context, scenario string) (string, error) {
	if root := scenarioPathFrom(ctx); root != "" {
		return filepath.Join(root, "cli", "manifest.json"), nil
	}
	if isProjectTarget(scenario) {
		return filepath.Join(p.RepoRoot, "cli", "manifest.json"), nil
	}
	path, err := repocontract.ScenarioCLIManifestPath(p.RepoRoot, scenario)
	if err != nil {
		return "", fmt.Errorf("resolve manifest path: %w", err)
	}
	return path, nil
}

// hashHex is the SHA-256 hex digest used to detect a stale artifact. It must
// match how cliapp.BuildPrimitiveEvidence hashes the manifest.
func hashHex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
