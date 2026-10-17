package resourcedeployment

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	activationStagingDir   = ".staging"
	activationGenerations  = "generations"
	activationCurrentEntry = "current"
)

// ActivationRequest describes one immutable release promotion. SourceDir is a
// release directory containing release-manifest.json and its declared
// artifacts. InstallDir is owner-managed state; it is never replaced as a
// whole, so the currently active generation survives a failed attempt.
type ActivationRequest struct {
	SourceDir     string
	InstallDir    string
	OperationID   string
	GenerationID  string
	TrustMode     ArtifactTrustMode
	PublicKeyPath string
}

// ActivationReceipt is the bounded evidence returned after promotion. Bytes
// transferred excludes content reused from an interrupted attempt; every
// reused entry was re-hashed before it was counted.
type ActivationReceipt struct {
	OperationID        string `json:"operation_id"`
	GenerationID       string `json:"generation_id"`
	PreviousGeneration string `json:"previous_generation,omitempty"`
	ManifestDigest     string `json:"manifest_digest"`
	BytesTransferred   int64  `json:"bytes_transferred"`
	ReusedBytes        int64  `json:"reused_bytes"`
	RevalidatedFiles   int    `json:"revalidated_files"`
	State              string `json:"state"`
}

const ActivationStateActivated = "activated"

// ActivateRelease verifies, stages, and atomically promotes a release. A
// failed verification leaves the previous current generation untouched and
// intentionally keeps the operation stage so a later retry can revalidate and
// reuse complete entries without trusting partial bytes.
func ActivateRelease(ctx context.Context, req ActivationRequest) (ActivationReceipt, error) {
	if err := validateActivationRequest(req); err != nil {
		return ActivationReceipt{}, err
	}
	source, err := filepath.Abs(filepath.Clean(req.SourceDir))
	if err != nil {
		return ActivationReceipt{}, fmt.Errorf("resolve release source: %w", err)
	}
	install, err := filepath.Abs(filepath.Clean(req.InstallDir))
	if err != nil {
		return ActivationReceipt{}, fmt.Errorf("resolve activation directory: %w", err)
	}
	manifest, err := LoadReleaseManifest(source)
	if err != nil {
		return ActivationReceipt{}, err
	}
	canonical, err := manifest.CanonicalBytes()
	if err != nil {
		return ActivationReceipt{}, err
	}
	manifestSum := sha256.Sum256(canonical)
	receipt := ActivationReceipt{
		OperationID:    req.OperationID,
		GenerationID:   req.GenerationID,
		ManifestDigest: "sha256:" + hex.EncodeToString(manifestSum[:]),
	}
	previous, err := activeGeneration(install)
	if err != nil {
		return receipt, err
	}
	receipt.PreviousGeneration = previous

	stage := filepath.Join(install, activationStagingDir, req.OperationID)
	if err := os.MkdirAll(stage, 0o750); err != nil {
		return receipt, fmt.Errorf("create activation stage: %w", err)
	}
	if err := removeUndeclaredStageEntries(stage, manifest); err != nil {
		return receipt, err
	}

	for _, artifact := range manifest.Artifacts {
		if err := ctx.Err(); err != nil {
			return receipt, err
		}
		sourcePath := filepath.Join(source, artifact.Name)
		stagePath := filepath.Join(stage, artifact.Name)
		if err := ensureRegularArtifact(sourcePath, "release source", artifact.Name); err != nil {
			return receipt, err
		}
		if size, ok, err := verifiedFileSize(stagePath, artifact.SHA256); err != nil {
			return receipt, err
		} else if ok {
			receipt.ReusedBytes += size
			receipt.RevalidatedFiles++
			continue
		}
		written, err := copyVerifiedFile(ctx, sourcePath, stagePath, artifact.SHA256)
		if err != nil {
			return receipt, err
		}
		receipt.BytesTransferred += written
	}
	if err := writeActivationMetadata(ctx, source, stage, canonical, req.TrustMode); err != nil {
		return receipt, err
	}
	if _, _, err := VerifyReleaseDirectory(stage, req.TrustMode, req.PublicKeyPath); err != nil {
		return receipt, fmt.Errorf("verify staged release: %w", err)
	}

	generations := filepath.Join(install, activationGenerations)
	if err := os.MkdirAll(generations, 0o750); err != nil {
		return receipt, fmt.Errorf("create activation generations: %w", err)
	}
	generation := filepath.Join(generations, req.GenerationID)
	if _, err := os.Lstat(generation); err == nil {
		// A process can terminate after the stage rename and before replacing
		// current. Treat a matching, already-verified generation as resumable
		// work for the same operation instead of stranding it behind the old
		// generation. A different generation with the same identity is a hard
		// collision and must never be overwritten.
		if _, _, verifyErr := VerifyReleaseDirectory(generation, req.TrustMode, req.PublicKeyPath); verifyErr != nil {
			return receipt, fmt.Errorf("activation generation %q already exists and is not the requested verified release: %w", req.GenerationID, verifyErr)
		}
		if err := atomicallyActivate(install, req.GenerationID); err != nil {
			return receipt, err
		}
		receipt.State = ActivationStateActivated
		return receipt, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return receipt, fmt.Errorf("inspect activation generation: %w", err)
	}
	if err := os.Rename(stage, generation); err != nil {
		return receipt, fmt.Errorf("promote verified generation: %w", err)
	}
	if err := atomicallyActivate(install, req.GenerationID); err != nil {
		// The old current link remains active if activation could not replace it.
		// The verified generation is retained for an explicit recovery attempt.
		return receipt, err
	}
	receipt.State = ActivationStateActivated
	return receipt, nil
}

func validateActivationRequest(req ActivationRequest) error {
	if strings.TrimSpace(req.SourceDir) == "" || strings.TrimSpace(req.InstallDir) == "" {
		return errors.New("activation source and install directories are required")
	}
	if !safeActivationSegment(req.OperationID) || !safeActivationSegment(req.GenerationID) {
		return errors.New("activation operation and generation ids must be safe path segments")
	}
	return req.TrustMode.Validate()
}

func safeActivationSegment(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && value != "." && value != ".." && filepath.Base(value) == value && !strings.ContainsAny(value, `/\\`)
}

func ensureRegularArtifact(path, label, name string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("%s artifact %q is unavailable: %w", label, name, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%s artifact %q must be a regular file; symlinks and directories are not allowed", label, name)
	}
	return nil
}

func verifiedFileSize(path, expected string) (int64, bool, error) {
	if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
		return 0, false, nil
	} else if err != nil {
		return 0, false, fmt.Errorf("inspect staged artifact %q: %w", path, err)
	}
	if err := ensureRegularArtifact(path, "staged", filepath.Base(path)); err != nil {
		return 0, false, nil
	}
	got, size, err := fileSHA256(path)
	if err != nil {
		return 0, false, err
	}
	if !strings.EqualFold(got, strings.TrimSpace(expected)) {
		return 0, false, nil
	}
	return size, true, nil
}

func copyVerifiedFile(ctx context.Context, source, destination, expected string) (int64, error) {
	parent := filepath.Dir(destination)
	if err := os.MkdirAll(parent, 0o750); err != nil {
		return 0, fmt.Errorf("create staged artifact directory: %w", err)
	}
	input, err := os.Open(source)
	if err != nil {
		return 0, fmt.Errorf("open release artifact: %w", err)
	}
	defer input.Close()
	tmp, err := os.CreateTemp(parent, ".activation-*")
	if err != nil {
		return 0, fmt.Errorf("create staged artifact: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return 0, fmt.Errorf("secure staged artifact: %w", err)
	}
	hash := sha256.New()
	written, err := io.Copy(io.MultiWriter(tmp, hash), input)
	if err == nil {
		err = ctx.Err()
	}
	if err == nil {
		err = tmp.Sync()
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return 0, fmt.Errorf("stage release artifact: %w", err)
	}
	got := hex.EncodeToString(hash.Sum(nil))
	if !strings.EqualFold(got, strings.TrimSpace(expected)) {
		return 0, fmt.Errorf("release artifact %q changed during staging: got %s, want %s", filepath.Base(source), got, expected)
	}
	if err := os.Rename(tmpPath, destination); err != nil {
		return 0, fmt.Errorf("install staged artifact: %w", err)
	}
	return written, nil
}

func fileSHA256(path string) (string, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", 0, fmt.Errorf("read staged artifact: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return "", 0, fmt.Errorf("stat staged artifact: %w", err)
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", 0, fmt.Errorf("hash staged artifact: %w", err)
	}
	return hex.EncodeToString(hash.Sum(nil)), info.Size(), nil
}

func writeActivationMetadata(ctx context.Context, source, stage string, canonical []byte, mode ArtifactTrustMode) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := atomicWrite(filepath.Join(stage, "release-manifest.json"), canonical, 0o640); err != nil {
		return fmt.Errorf("stage release manifest: %w", err)
	}
	if mode != ArtifactTrustProduction {
		return nil
	}
	signature, err := os.ReadFile(filepath.Join(source, "release-manifest.sig.json"))
	if err != nil {
		return fmt.Errorf("read release signature: %w", err)
	}
	if err := atomicWrite(filepath.Join(stage, "release-manifest.sig.json"), signature, 0o640); err != nil {
		return fmt.Errorf("stage release signature: %w", err)
	}
	return nil
}

func atomicWrite(path string, content []byte, mode os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".activation-metadata-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func removeUndeclaredStageEntries(stage string, manifest ReleaseManifest) error {
	keep := map[string]struct{}{"release-manifest.json": {}}
	if _, err := os.Stat(filepath.Join(stage, "release-manifest.sig.json")); err == nil {
		keep["release-manifest.sig.json"] = struct{}{}
	}
	for _, artifact := range manifest.Artifacts {
		keep[artifact.Name] = struct{}{}
	}
	entries, err := os.ReadDir(stage)
	if err != nil {
		return fmt.Errorf("inspect activation stage: %w", err)
	}
	for _, entry := range entries {
		if _, ok := keep[entry.Name()]; ok {
			continue
		}
		if err := os.RemoveAll(filepath.Join(stage, entry.Name())); err != nil {
			return fmt.Errorf("remove undeclared staged entry %q: %w", entry.Name(), err)
		}
	}
	return nil
}

func activeGeneration(install string) (string, error) {
	target, err := os.Readlink(filepath.Join(install, activationCurrentEntry))
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read active generation: %w", err)
	}
	const prefix = activationGenerations + string(filepath.Separator)
	if !strings.HasPrefix(target, prefix) || !safeActivationSegment(strings.TrimPrefix(target, prefix)) {
		return "", errors.New("active generation link is outside the managed generations directory")
	}
	return strings.TrimPrefix(target, prefix), nil
}

func atomicallyActivate(install, generationID string) error {
	current := filepath.Join(install, activationCurrentEntry)
	if info, err := os.Lstat(current); err == nil && info.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("active generation entry %q is not a managed symlink", current)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect active generation entry: %w", err)
	}
	tmp := filepath.Join(install, ".current-"+generationID)
	_ = os.Remove(tmp)
	if err := os.Symlink(filepath.Join(activationGenerations, generationID), tmp); err != nil {
		return fmt.Errorf("prepare active generation link: %w", err)
	}
	if err := os.Rename(tmp, current); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("activate generation: %w", err)
	}
	return nil
}
