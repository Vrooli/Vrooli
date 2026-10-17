package recoverypoint

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"
)

// VerifyRequest checks a recovery point without restoring it.
type VerifyRequest struct {
	Dir string
	// Keys is optional: when present every artifact is also opened and its
	// plaintext checksum compared, proving the key reference still resolves.
	Keys   KeyResolver
	Sealer Sealer
	Now    func() time.Time
}

// VerifyReport is the outcome of a verification.
type VerifyReport struct {
	RecoveryPointID string    `json:"recovery_point_id"`
	DeploymentID    string    `json:"deployment_id"`
	Refs            Refs      `json:"refs"`
	CapturedAt      time.Time `json:"captured_at"`
	VerifiedAt      time.Time `json:"verified_at"`
	// RecoveryPointAge is how old the recovery point was at verification.
	RecoveryPointAge time.Duration        `json:"recovery_point_age_ns"`
	Bindings         []string             `json:"bindings"`
	Checksums        map[string]Inventory `json:"checksums"`
	ManifestDigest   string               `json:"manifest_digest"`
	ArtifactsIntact  bool                 `json:"artifacts_intact"`
	KeyResolved      bool                 `json:"key_resolved"`
	ArtifactsOpened  bool                 `json:"artifacts_opened"`
	Outcome          string               `json:"outcome"`
	Error            *Error               `json:"error,omitempty"`
}

// Verify checks the manifest digest and every sealed artifact's size and
// checksum; with a key resolver it also opens each artifact.
func Verify(ctx context.Context, req VerifyRequest) (VerifyReport, error) {
	now := time.Now
	if req.Now != nil {
		now = req.Now
	}
	report := VerifyReport{VerifiedAt: now().UTC(), Outcome: OutcomeFailed}
	fail := func(err error) (VerifyReport, error) {
		report.Error = AsError(err, CodeVerifyFailed)
		return report, err
	}
	manifest, err := ReadManifest(req.Dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fail(newError(CodeRecoveryPointCorrupt, "%s holds no recovery point manifest", req.Dir))
		}
		return fail(err)
	}
	report.RecoveryPointID, report.DeploymentID, report.Refs, report.CapturedAt = manifest.ID, manifest.DeploymentID, manifest.Refs, manifest.CapturedAt
	report.Bindings, report.Checksums, report.ManifestDigest = manifest.BindingIDs(), manifest.Checksums, manifest.Digest
	report.RecoveryPointAge = report.VerifiedAt.Sub(manifest.CapturedAt)
	if err := verifySealedArtifacts(req.Dir, manifest); err != nil {
		return fail(err)
	}
	report.ArtifactsIntact = true
	if req.Keys != nil {
		material, err := req.Keys.ResolveKey(ctx, manifest.KeyRef)
		if err != nil {
			return fail(keyUnavailable(manifest.KeyRef, err))
		}
		report.KeyResolved = true
		for _, artifact := range manifest.Artifacts {
			sealed, err := os.ReadFile(filepath.Join(req.Dir, artifact.File)) //nolint:gosec // recovery point directory
			if err != nil {
				return fail(newError(CodeRecoveryPointCorrupt, "artifact %s: %v", artifact.File, err))
			}
			plain, err := req.Sealer.Open(material, sealed, manifest.ID, artifact.Binding)
			if err != nil {
				return fail(err)
			}
			if BytesSHA256(plain) != artifact.PlainSHA256 {
				return fail(newError(CodeRecoveryPointCorrupt, "binding %s: decrypted artifact does not match its recorded checksum", artifact.Binding))
			}
		}
		report.ArtifactsOpened = true
	}
	report.Outcome = OutcomeSucceeded
	return report, nil
}
