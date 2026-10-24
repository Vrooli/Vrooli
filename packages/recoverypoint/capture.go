package recoverypoint

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const stagingSuffix = ".staging"

// CaptureRequest describes one recovery point to capture.
type CaptureRequest struct {
	DeploymentID    string
	RecoveryPointID string
	// Dir is the final recovery point directory. Capture stages beside it
	// and promotes with a rename, so Dir either holds a complete recovery
	// point or does not exist.
	Dir      string
	Bindings []Binding
	Refs     Refs
	// KeyRef is recorded; the material is resolved through Keys.
	KeyRef string
	Keys   KeyResolver
	Sealer Sealer

	Providers Registry
	// Boundary defaults to ApplicationHooks without a runner.
	Boundary Boundary

	Provider         string
	ProviderRef      string
	RetentionPolicy  string
	MigrationPosture string
	Now              func() time.Time
}

// Capture produces an encrypted, checksummed recovery point. When Dir already
// holds a valid recovery point with the same id the manifest is returned
// unchanged (idempotent); any other pre-existing content is refused.
func Capture(ctx context.Context, req CaptureRequest) (Manifest, error) {
	if err := validateCapture(req); err != nil {
		return Manifest{}, err
	}
	if existing, err := ReadManifest(req.Dir); err == nil {
		if existing.ID == req.RecoveryPointID && existing.DeploymentID == req.DeploymentID {
			return existing, nil
		}
		return Manifest{}, newError(CodeRecoveryPointExists, "%s already holds recovery point %s", req.Dir, existing.ID)
	} else if !errors.Is(err, os.ErrNotExist) {
		return Manifest{}, err
	}
	if _, statErr := os.Stat(req.Dir); statErr == nil {
		return Manifest{}, newError(CodeRecoveryPointExists, "%s exists without a manifest; refusing to reuse it", req.Dir)
	}
	material, err := req.Keys.ResolveKey(ctx, req.KeyRef)
	if err != nil {
		return Manifest{}, keyUnavailable(req.KeyRef, err)
	}
	now := time.Now
	if req.Now != nil {
		now = req.Now
	}
	boundary := req.Boundary
	if boundary == nil {
		boundary = ApplicationHooks{}
	}
	stage := req.Dir + stagingSuffix
	_ = os.RemoveAll(stage)
	if err := os.MkdirAll(stage, 0o755); err != nil {
		return Manifest{}, newError(CodeCaptureFailed, "create staging dir: %v", err)
	}
	manifest := Manifest{
		FormatVersion: FormatVersion, ID: req.RecoveryPointID, DeploymentID: req.DeploymentID,
		Bindings: req.Bindings, Refs: req.Refs, CapturedAt: now().UTC(),
		Provider: req.Provider, ProviderRef: req.ProviderRef, Checksums: map[string]Inventory{},
		Encrypted: true, KeyRef: req.KeyRef, RetentionPolicy: req.RetentionPolicy,
		MigrationPosture: req.MigrationPosture,
	}
	if manifest.Refs.CredentialVersionRefs == nil {
		manifest.Refs.CredentialVersionRefs = []string{}
	}
	if err := captureBindings(ctx, req, boundary, material, stage, &manifest); err != nil {
		_ = os.RemoveAll(stage)
		return Manifest{}, err
	}
	manifest.ConsistencyToken = consistencyToken(manifest.Consistency)
	written, err := WriteManifest(stage, manifest)
	if err != nil {
		_ = os.RemoveAll(stage)
		return Manifest{}, err
	}
	if err := os.MkdirAll(filepath.Dir(req.Dir), 0o755); err != nil {
		_ = os.RemoveAll(stage)
		return Manifest{}, newError(CodeCaptureFailed, "create recovery point parent: %v", err)
	}
	if err := os.Rename(stage, req.Dir); err != nil {
		_ = os.RemoveAll(stage)
		return Manifest{}, newError(CodeCaptureFailed, "promote recovery point: %v", err)
	}
	return written, nil
}

func captureBindings(ctx context.Context, req CaptureRequest, boundary Boundary, material []byte, stage string, manifest *Manifest) error {
	for _, b := range req.Bindings {
		provider, err := req.Providers.Lookup(b)
		if err != nil {
			return err
		}
		record, release, err := boundary.Enter(ctx, b, provider.Mode())
		if err != nil {
			return err
		}
		result, captureErr := provider.Capture(ctx, b, stage)
		if releaseErr := release(ctx); releaseErr != nil && captureErr == nil {
			captureErr = newError(CodeCaptureFailed, "%v", releaseErr)
		}
		if captureErr != nil {
			return captureErr
		}
		artifact, err := sealArtifact(req.Sealer, material, req.RecoveryPointID, b.ID, result.ArtifactPath, stage)
		if err != nil {
			return err
		}
		manifest.Consistency = append(manifest.Consistency, record)
		manifest.Checksums[b.ID] = result.Inventory
		manifest.Artifacts = append(manifest.Artifacts, artifact)
	}
	return nil
}

func sealArtifact(sealer Sealer, material []byte, recoveryPointID, binding, plainPath, stage string) (Artifact, error) {
	plain, err := os.ReadFile(plainPath) //nolint:gosec // provider artifact inside the staging dir
	if err != nil {
		return Artifact{}, newError(CodeCaptureFailed, "binding %s: read artifact: %v", binding, err)
	}
	sealed, err := sealer.Seal(material, plain, recoveryPointID, binding)
	if err != nil {
		return Artifact{}, newError(CodeCaptureFailed, "binding %s: seal artifact: %v", binding, err)
	}
	file := binding + ".sealed"
	if err := os.WriteFile(filepath.Join(stage, file), sealed, 0o600); err != nil {
		return Artifact{}, newError(CodeCaptureFailed, "binding %s: write sealed artifact: %v", binding, err)
	}
	if err := os.Remove(plainPath); err != nil {
		return Artifact{}, newError(CodeCaptureFailed, "binding %s: remove plaintext artifact: %v", binding, err)
	}
	return Artifact{Binding: binding, File: file, Bytes: int64(len(sealed)), SHA256: BytesSHA256(sealed), PlainSHA256: BytesSHA256(plain), PlainBytes: int64(len(plain))}, nil
}

func consistencyToken(records []ConsistencyRecord) string {
	parts := make([]string, 0, len(records))
	for _, r := range records {
		parts = append(parts, r.Binding+":"+r.Mode+":"+r.WriteQuiescence+":"+r.Token)
	}
	sort.Strings(parts)
	sum := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return hex.EncodeToString(sum[:])
}

func validateCapture(req CaptureRequest) error {
	if !identifierPattern.MatchString(req.DeploymentID) {
		return newError(CodeInvalidArgument, "deployment id %q is not a valid identifier", req.DeploymentID)
	}
	if !identifierPattern.MatchString(req.RecoveryPointID) {
		return newError(CodeInvalidArgument, "recovery point id %q is not a valid identifier", req.RecoveryPointID)
	}
	if strings.TrimSpace(req.Dir) == "" || !filepath.IsAbs(req.Dir) {
		return newError(CodeInvalidArgument, "recovery point dir must be an absolute path")
	}
	if len(req.Bindings) == 0 {
		return newError(CodeInvalidArgument, "at least one binding is required")
	}
	seen := map[string]bool{}
	for _, b := range req.Bindings {
		if err := b.Validate(); err != nil {
			return err
		}
		if seen[b.ID] {
			return newError(CodeInvalidArgument, "binding %s is declared twice", b.ID)
		}
		seen[b.ID] = true
	}
	if strings.TrimSpace(req.KeyRef) == "" {
		return newError(CodeInvalidArgument, "key reference is required; recovery points are always encrypted")
	}
	if req.Keys == nil {
		return newError(CodeInvalidArgument, "key resolver is required")
	}
	if req.Providers == nil {
		return newError(CodeInvalidArgument, "provider registry is required")
	}
	switch req.MigrationPosture {
	case "", PostureGreenfield, PostureGreenfieldWithData, PostureProductionEvolution:
	default:
		return newError(CodeInvalidArgument, "migration posture %q is unknown", req.MigrationPosture)
	}
	return nil
}

func keyUnavailable(keyRef string, cause error) *Error {
	return &Error{
		Code:    CodeRecoveryKeyMissing,
		Message: fmt.Sprintf("recovery key %s is unavailable: %v", keyRef, cause),
		Blocker: keyRef,
		Details: map[string]any{"key_ref": keyRef, "required": "recovery key material resolvable from the reference outside the target failure domain"},
	}
}
