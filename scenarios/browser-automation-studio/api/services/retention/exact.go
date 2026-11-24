package retention

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/executionartifactlock"
	"github.com/vrooli/browser-automation-studio/internal/executioninventory"
	"github.com/vrooli/browser-automation-studio/services/evidence"
)

var (
	ErrExactExecutionMissing   = errors.New("execution not found")
	ErrExactExecutionActive    = errors.New("execution is not terminal")
	ErrExactArtifactShared     = errors.New("artifact directory is referenced by another execution")
	ErrExactArtifactOverlap    = errors.New("artifact path overlaps another execution")
	ErrExactInventoryInvalid   = errors.New("execution artifact inventory is missing or invalid")
	ErrExactArtifactMissing    = errors.New("execution artifact directory is missing")
	ErrExactConfirmation       = errors.New("confirmation must exactly match the execution UUID")
	ErrExactPreviewChanged     = errors.New("execution or artifacts changed since preview")
	ErrExactExternalArtifacts  = errors.New("exact purge cannot prove exclusive ownership of external evidence artifacts")
	ErrExactVerificationFailed = errors.New("exact purge verification failed")
)

// ExactPurgePreview contains only non-content metadata safe for operator review.
type ExactPurgePreview struct {
	ExecutionID        string `json:"execution_id"`
	ArtifactFiles      int    `json:"artifact_files"`
	ArtifactBytes      int64  `json:"artifact_bytes"`
	ArtifactSHA256     string `json:"artifact_sha256"`
	ArtifactPathSHA256 string `json:"artifact_path_sha256"`
	ExternalEvidence   int    `json:"external_evidence"`
	Status             string `json:"status"`
	Ready              bool   `json:"ready"`
}

// PreviewExactPurge selects one terminal row and proves its artifact directory
// is present and exclusively referenced. It never returns execution content.
func (s *Service) PreviewExactPurge(ctx context.Context, id uuid.UUID) (ExactPurgePreview, error) {
	if s == nil || s.store == nil || s.fs == nil || strings.TrimSpace(s.recordingsRoot) == "" {
		return ExactPurgePreview{}, ErrRecordingsRootNotConfigured
	}
	release, err := executionartifactlock.Acquire(ctx, s.recordingsRoot, id)
	if err != nil {
		return ExactPurgePreview{}, err
	}
	defer release()
	return s.inspectExact(ctx, id)
}

// PurgeExact deletes one execution only after the caller repeats the UUID and
// the artifact metadata still matches the preview. It verifies the row is gone.
func (s *Service) PurgeExact(ctx context.Context, id uuid.UUID, confirmation string, preview ExactPurgePreview) (ExactPurgePreview, error) {
	if s == nil || s.store == nil || s.fs == nil || strings.TrimSpace(s.recordingsRoot) == "" {
		return ExactPurgePreview{}, ErrRecordingsRootNotConfigured
	}
	if strings.TrimSpace(confirmation) != id.String() || preview.ExecutionID != id.String() || !preview.Ready {
		return ExactPurgePreview{}, ErrExactConfirmation
	}
	if preview.ExternalEvidence > 0 {
		return ExactPurgePreview{}, ErrExactExternalArtifacts
	}
	s.exactPurgeMu.Lock()
	defer s.exactPurgeMu.Unlock()
	release, err := executionartifactlock.Acquire(ctx, s.recordingsRoot, id)
	if err != nil {
		return ExactPurgePreview{}, err
	}
	defer release()

	current, err := s.inspectExact(ctx, id)
	if err != nil {
		return ExactPurgePreview{}, err
	}
	if current != preview {
		return ExactPurgePreview{}, ErrExactPreviewChanged
	}
	entry, err := s.store.GetExecution(ctx, id)
	if err != nil {
		return ExactPurgePreview{}, err
	}
	root, err := canonicalRecordingsRoot(s.recordingsRoot)
	if err != nil {
		return ExactPurgePreview{}, err
	}
	if entry == nil || entry.ID != id || entry.Status != preview.Status || filepath.Clean(entry.ResultPath) != filepath.Join(root, id.String(), "result.json") {
		return ExactPurgePreview{}, ErrExactPreviewChanged
	}
	dir, err := canonicalExecutionDir(s.recordingsRoot, id)
	if err != nil {
		return ExactPurgePreview{}, err
	}
	deleter, ok := s.fs.(containedDeleter)
	if !ok {
		return ExactPurgePreview{}, errors.New("contained artifact deletion is unavailable")
	}
	// The filesystem and execution row do not share a transaction. Remove the
	// exclusive directory first so a database failure leaves the row visible for
	// diagnosis (with a missing-artifact refusal on retry), never a hidden orphan.
	if err := deleter.DeleteContained(ctx, filepath.Dir(dir), dir); err != nil {
		return ExactPurgePreview{}, fmt.Errorf("delete exact artifact directory: %w", err)
	}
	if err := s.store.DeleteExecution(ctx, id); err != nil {
		return ExactPurgePreview{}, fmt.Errorf("delete exact execution row: %w", err)
	}
	if _, err := s.store.GetExecution(ctx, id); !errors.Is(err, database.ErrNotFound) {
		return ExactPurgePreview{}, fmt.Errorf("%w: execution row remains or could not be read", ErrExactVerificationFailed)
	}
	if _, exists, err := s.fs.DirSize(dir); err != nil || exists {
		return ExactPurgePreview{}, fmt.Errorf("%w: artifact directory remains or could not be checked", ErrExactVerificationFailed)
	}
	return current, nil
}

func (s *Service) inspectExact(ctx context.Context, id uuid.UUID) (ExactPurgePreview, error) {
	entry, err := s.store.GetExecution(ctx, id)
	if errors.Is(err, database.ErrNotFound) || entry == nil && err == nil {
		return ExactPurgePreview{}, ErrExactExecutionMissing
	}
	if err != nil {
		return ExactPurgePreview{}, fmt.Errorf("read execution metadata: %w", err)
	}
	if !database.IsTerminalStatus(entry.Status) {
		return ExactPurgePreview{}, ErrExactExecutionActive
	}
	root, err := canonicalRecordingsRoot(s.recordingsRoot)
	if err != nil {
		return ExactPurgePreview{}, err
	}
	dir := filepath.Join(root, id.String())
	if entry.ID == uuid.Nil || entry.ID != id || filepath.Clean(entry.ResultPath) != filepath.Join(dir, "result.json") {
		return ExactPurgePreview{}, ErrExactInventoryInvalid
	}
	all, total, err := s.store.ListExecutions(ctx, database.ExecutionQuery{})
	if err != nil {
		return ExactPurgePreview{}, fmt.Errorf("check artifact ownership: %w", err)
	}
	if total != len(all) {
		return ExactPurgePreview{}, ErrExactInventoryInvalid
	}
	seen := make(map[uuid.UUID]struct{}, len(all))
	for _, other := range all {
		if other == nil || other.ID == uuid.Nil || other.WorkflowID == uuid.Nil || other.Status == "" {
			return ExactPurgePreview{}, ErrExactInventoryInvalid
		}
		if _, duplicate := seen[other.ID]; duplicate {
			return ExactPurgePreview{}, ErrExactInventoryInvalid
		}
		seen[other.ID] = struct{}{}
		if other.ID == id {
			continue
		}
		otherDir := filepath.Join(root, other.ID.String())
		if strings.TrimSpace(other.ResultPath) != "" {
			otherResult := filepath.Clean(other.ResultPath)
			if pathsOverlap(dir, filepath.Dir(otherResult)) || pathsOverlap(dir, otherDir) {
				return ExactPurgePreview{}, ErrExactArtifactOverlap
			}
			if otherResult != filepath.Join(otherDir, "result.json") {
				return ExactPurgePreview{}, ErrExactInventoryInvalid
			}
		}
	}
	if _, found := seen[id]; !found {
		return ExactPurgePreview{}, ErrExactInventoryInvalid
	}
	artifactInfo, err := os.Lstat(dir)
	if errors.Is(err, os.ErrNotExist) {
		return ExactPurgePreview{}, ErrExactArtifactMissing
	}
	if err != nil || !artifactInfo.IsDir() || artifactInfo.Mode()&os.ModeSymlink != 0 {
		return ExactPurgePreview{}, ErrExactInventoryInvalid
	}
	if err := verifyPurgeTreeOwnership(root, dir); err != nil {
		return ExactPurgePreview{}, fmt.Errorf("verify execution artifact ownership: %w", err)
	}
	manifest, err := verifyExecutionManifest(dir, id, entry.WorkflowID)
	if err != nil {
		return ExactPurgePreview{}, fmt.Errorf("verify execution evidence manifest: %w", err)
	}
	externalEvidence, err := verifyDirectoryInventory(dir, id, entry.WorkflowID, manifest)
	if err != nil {
		return ExactPurgePreview{}, fmt.Errorf("verify execution directory inventory: %w", err)
	}
	files, size, digest, exists, err := inspectArtifactTree(dir)
	if err != nil {
		return ExactPurgePreview{}, fmt.Errorf("inspect execution artifacts: %w", err)
	}
	if !exists {
		return ExactPurgePreview{}, ErrExactArtifactMissing
	}
	pathDigest := sha256.Sum256([]byte(filepath.Clean(dir)))
	return ExactPurgePreview{ExecutionID: id.String(), ArtifactFiles: files, ArtifactBytes: size, ArtifactSHA256: digest, ArtifactPathSHA256: hex.EncodeToString(pathDigest[:]), ExternalEvidence: externalEvidence, Status: entry.Status, Ready: true}, nil
}

// verifyPurgeTreeOwnership defines the filesystem threat boundary for exact
// deletion: BAS's recordings root and target tree must be owned by this local
// service identity and must not be writable by another UID. Writers running as
// the same local operator must participate in executionartifactlock; an
// unrelated same-UID process is within the trusted local-operator boundary.
func verifyPurgeTreeOwnership(root, target string) error {
	if err := verifyPurgeAncestors(root); err != nil {
		return fmt.Errorf("recordings root ancestor: %w", err)
	}
	rootInfo, err := os.Stat(root)
	if err != nil || !rootInfo.IsDir() || !ownedByEffectiveUser(rootInfo) || rootInfo.Mode().Perm()&0o022 != 0 {
		return fmt.Errorf("recordings root ownership/permissions: %w", ErrExactInventoryInvalid)
	}
	err = filepath.WalkDir(target, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, err := os.Lstat(path)
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !ownedByEffectiveUser(info) || info.Mode().Perm()&0o022 != 0 {
			return fmt.Errorf("execution artifact ownership/permissions at %s: %w", path, ErrExactInventoryInvalid)
		}
		return nil
	})
	if err != nil {
		return ErrExactInventoryInvalid
	}
	return nil
}

func verifyPurgeAncestors(root string) error {
	current := filepath.Clean(root)
	for {
		info, err := os.Lstat(current)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("unsafe ancestor %s: %w", current, ErrExactInventoryInvalid)
		}
		permissions := info.Mode().Perm()
		sticky := info.Mode()&os.ModeSticky != 0
		if permissions&0o020 != 0 && !sticky || permissions&0o002 != 0 && !sticky {
			return fmt.Errorf("writable ancestor %s: %w", current, ErrExactInventoryInvalid)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return nil
		}
		current = parent
	}
}

func ownedByEffectiveUser(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Uid == uint32(os.Geteuid())
}

func canonicalRecordingsRoot(raw string) (string, error) {
	root, err := filepath.Abs(filepath.Clean(strings.TrimSpace(raw)))
	if err != nil {
		return "", ErrExactInventoryInvalid
	}
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", ErrExactInventoryInvalid
	}
	return filepath.Clean(resolved), nil
}

func canonicalExecutionDir(raw string, id uuid.UUID) (string, error) {
	root, err := canonicalRecordingsRoot(raw)
	if err != nil {
		return "", err
	}
	dir := filepath.Join(root, id.String())
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil || filepath.Clean(resolved) != dir {
		return "", ErrExactInventoryInvalid
	}
	return dir, nil
}

func pathsOverlap(a, b string) bool {
	a, errA := filepath.Abs(filepath.Clean(a))
	b, errB := filepath.Abs(filepath.Clean(b))
	if errA != nil || errB != nil {
		return true
	}
	return a == b || pathContains(a, b) || pathContains(b, a)
}

func pathContains(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	return err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

type exactEvidenceManifest struct {
	ID            string `json:"id"`
	SchemaVersion string `json:"schema_version"`
	ExecutionID   string `json:"execution_id"`
	WorkflowID    string `json:"workflow_id"`
	Evidence      struct {
		ID            string `json:"id"`
		ExecutionID   string `json:"execution_id"`
		SchemaVersion string `json:"schema_version"`
		Artifacts     []struct {
			ID          string          `json:"id"`
			Kind        json.RawMessage `json:"kind"`
			SizeBytes   int64           `json:"size_bytes"`
			SHA256      string          `json:"sha256"`
			ExecutionID string          `json:"execution_id"`
			Provenance  struct {
				Source       *string `json:"source"`
				ArtifactType *string `json:"artifact_type"`
			} `json:"provenance"`
		} `json:"artifacts"`
	} `json:"evidence"`
}

func verifyExecutionManifest(dir string, id, workflowID uuid.UUID) (exactEvidenceManifest, error) {
	f, err := os.Open(filepath.Join(dir, "evidence.proto.json"))
	if err != nil {
		return exactEvidenceManifest{}, ErrExactInventoryInvalid
	}
	defer f.Close()
	var manifest exactEvidenceManifest
	decoder := json.NewDecoder(f)
	if err := decoder.Decode(&manifest); err != nil {
		return exactEvidenceManifest{}, ErrExactInventoryInvalid
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return exactEvidenceManifest{}, ErrExactInventoryInvalid
	}
	if _, err := uuid.Parse(manifest.ID); err != nil {
		return exactEvidenceManifest{}, ErrExactInventoryInvalid
	}
	if _, err := uuid.Parse(manifest.Evidence.ID); err != nil {
		return exactEvidenceManifest{}, ErrExactInventoryInvalid
	}
	if manifest.ID == "" || manifest.ExecutionID != id.String() || manifest.SchemaVersion != "bas-replay/v1" ||
		manifest.Evidence.ID == "" || manifest.Evidence.ExecutionID != id.String() || manifest.Evidence.SchemaVersion != "bas-evidence/v1" ||
		manifest.WorkflowID != workflowID.String() {
		return exactEvidenceManifest{}, ErrExactInventoryInvalid
	}
	seen := make(map[string]struct{}, len(manifest.Evidence.Artifacts))
	for _, artifact := range manifest.Evidence.Artifacts {
		kind := strings.Trim(string(artifact.Kind), `"`)
		if artifact.ID == "" || artifact.ExecutionID != id.String() || !strings.HasPrefix(kind, "ARTIFACT_KIND_") || kind == "ARTIFACT_KIND_UNSPECIFIED" || artifact.SizeBytes < 0 || len(artifact.SHA256) != sha256.Size*2 {
			return exactEvidenceManifest{}, ErrExactInventoryInvalid
		}
		if _, err := uuid.Parse(artifact.ID); err != nil {
			return exactEvidenceManifest{}, ErrExactInventoryInvalid
		}
		if _, err := hex.DecodeString(artifact.SHA256); err != nil {
			return exactEvidenceManifest{}, ErrExactInventoryInvalid
		}
		if _, exists := seen[artifact.ID]; exists {
			return exactEvidenceManifest{}, ErrExactInventoryInvalid
		}
		seen[artifact.ID] = struct{}{}
	}
	return manifest, nil
}

type exactDirectoryInventory struct {
	SchemaVersion string `json:"schema_version"`
	ExecutionID   string `json:"execution_id"`
	WorkflowID    string `json:"workflow_id"`
	Files         []struct {
		Path   string `json:"path"`
		Bytes  int64  `json:"bytes"`
		SHA256 string `json:"sha256"`
	} `json:"files"`
	ArtifactBindings []exactArtifactBinding `json:"artifact_bindings"`
}

// exactArtifactBinding is a private, writer-authored statement of where an
// evidence artifact's bytes were persisted. The public evidence manifest stays
// storage independent and never gains a filesystem path or object key.
type exactArtifactBinding struct {
	ArtifactID string `json:"artifact_id"`
	Scope      string `json:"scope"`
	Storage    string `json:"storage"`
	Path       string `json:"path,omitempty"`
	ObjectKey  string `json:"object_key,omitempty"`
	Bytes      int64  `json:"bytes"`
	SHA256     string `json:"sha256"`
}

type exactDirectoryFile struct {
	bytes  int64
	digest string
}

func verifyDirectoryInventory(dir string, id, workflowID uuid.UUID, evidence exactEvidenceManifest) (int, error) {
	f, err := os.Open(filepath.Join(dir, "purge-inventory.v1.json"))
	if err != nil {
		return 0, ErrExactInventoryInvalid
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	var inventory exactDirectoryInventory
	if dec.Decode(&inventory) != nil {
		return 0, ErrExactInventoryInvalid
	}
	var trailing any
	if err := dec.Decode(&trailing); !errors.Is(err, io.EOF) {
		return 0, ErrExactInventoryInvalid
	}
	if inventory.SchemaVersion != "bas-execution-directory/v1" || inventory.ExecutionID != id.String() || inventory.WorkflowID != workflowID.String() || len(inventory.Files) == 0 {
		return 0, ErrExactInventoryInvalid
	}
	want := make(map[string]exactDirectoryFile, len(inventory.Files))
	for _, entry := range inventory.Files {
		if !safeInventoryPath(entry.Path) || entry.Bytes < 0 || len(entry.SHA256) != sha256.Size*2 {
			return 0, ErrExactInventoryInvalid
		}
		digest := strings.ToLower(entry.SHA256)
		if _, err := hex.DecodeString(digest); err != nil {
			return 0, ErrExactInventoryInvalid
		}
		if _, duplicate := want[entry.Path]; duplicate {
			return 0, ErrExactInventoryInvalid
		}
		want[entry.Path] = exactDirectoryFile{bytes: entry.Bytes, digest: digest}
	}
	got, err := inspectExactFiles(dir)
	if err != nil || len(got) != len(want) {
		return 0, ErrExactInventoryInvalid
	}
	for path, actual := range got {
		listed, ok := want[path]
		if !ok || listed != actual {
			return 0, ErrExactInventoryInvalid
		}
	}
	if err := reconcileEvidenceFiles(got, evidence, inventory.ArtifactBindings, id); err != nil {
		return 0, err
	}
	external := 0
	for _, binding := range inventory.ArtifactBindings {
		if binding.Storage == "external" {
			external++
		}
	}
	return external, nil
}

func safeInventoryPath(path string) bool {
	if path == "" || filepath.IsAbs(path) || strings.Contains(path, "\\") {
		return false
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
	return clean == path && clean != "." && clean != ".." && !strings.HasPrefix(clean, "../")
}

func inspectExactFiles(root string) (map[string]exactDirectoryFile, error) {
	files := make(map[string]exactDirectoryFile)
	executionID, err := uuid.Parse(filepath.Base(filepath.Clean(root)))
	if err != nil || executionID == uuid.Nil {
		return nil, ErrExactInventoryInvalid
	}
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return errors.New("symlink in execution directory")
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if rel != "." && rel != "artifacts" && !strings.HasPrefix(filepath.ToSlash(rel), "artifacts/") {
				return errors.New("unrecognized execution directory")
			}
			if rel != "." && !executioninventory.IsOwnedDirectory(filepath.ToSlash(rel), executionID.String()) {
				return errors.New("unowned execution directory")
			}
			return nil
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			return errors.New("non-regular execution file")
		}
		rel = filepath.ToSlash(rel)
		if rel == "purge-inventory.v1.json" {
			return nil
		}
		if !safeInventoryPath(rel) || !executioninventory.IsOwnedPath(rel, executionID.String()) {
			return errors.New("unsafe execution file path")
		}
		integrity, err := evidence.InspectFileIntegrity(path)
		if err != nil {
			return err
		}
		files[rel] = exactDirectoryFile{bytes: integrity.SizeBytes, digest: integrity.SHA256}
		return nil
	})
	return files, err
}

func reconcileEvidenceFiles(files map[string]exactDirectoryFile, evidence exactEvidenceManifest, bindings []exactArtifactBinding, executionID uuid.UUID) error {
	artifacts := make(map[string]struct {
		bytes  int64
		digest string
	}, len(evidence.Evidence.Artifacts))
	for _, artifact := range evidence.Evidence.Artifacts {
		artifacts[artifact.ID] = struct {
			bytes  int64
			digest string
		}{bytes: artifact.SizeBytes, digest: strings.ToLower(artifact.SHA256)}
	}
	seenIDs := make(map[string]struct{}, len(bindings))
	seenPaths := make(map[string]struct{}, len(bindings))
	seenEvidence := 0
	for _, binding := range bindings {
		if binding.ArtifactID == "" || binding.Bytes < 0 || len(binding.SHA256) != sha256.Size*2 {
			return ErrExactInventoryInvalid
		}
		if _, err := uuid.Parse(binding.ArtifactID); err != nil {
			return ErrExactInventoryInvalid
		}
		if _, exists := seenIDs[binding.ArtifactID]; exists {
			return ErrExactInventoryInvalid
		}
		seenIDs[binding.ArtifactID] = struct{}{}
		if _, err := hex.DecodeString(binding.SHA256); err != nil {
			return ErrExactInventoryInvalid
		}
		switch binding.Scope {
		case "evidence":
			artifact, ok := artifacts[binding.ArtifactID]
			if !ok || binding.Bytes != artifact.bytes || strings.ToLower(binding.SHA256) != artifact.digest {
				return ErrExactInventoryInvalid
			}
			seenEvidence++
			switch binding.Storage {
			case "inline", "external":
				if binding.Path != "" || (binding.Storage == "inline" && binding.ObjectKey != "") || (binding.Storage == "external" && !executioninventory.IsExecutionObjectKey(binding.ObjectKey, executionID.String())) {
					return ErrExactInventoryInvalid
				}
			case "execution_file":
				if binding.ObjectKey != "" {
					return ErrExactInventoryInvalid
				}
				if err := verifyBoundExecutionFile(files, binding, seenPaths); err != nil {
					return err
				}
			default:
				return ErrExactInventoryInvalid
			}
		case "auxiliary":
			if binding.Storage == "external" {
				if binding.Path != "" || !executioninventory.IsExecutionObjectKey(binding.ObjectKey, executionID.String()) {
					return ErrExactInventoryInvalid
				}
			} else if binding.Storage == "execution_file" {
				if binding.ObjectKey != "" {
					return ErrExactInventoryInvalid
				}
				if err := verifyBoundExecutionFile(files, binding, seenPaths); err != nil {
					return err
				}
			} else {
				return ErrExactInventoryInvalid
			}
		default:
			return ErrExactInventoryInvalid
		}
	}
	if len(artifacts) != seenEvidence {
		return ErrExactInventoryInvalid
	}
	// Every physical artifact file must have one identity-bearing writer
	// binding; digest-only matching can let inline or external evidence mask it.
	for path := range files {
		if strings.HasPrefix(path, "artifacts/") {
			if _, ok := seenPaths[path]; !ok {
				return ErrExactInventoryInvalid
			}
		}
	}
	return nil
}

func verifyBoundExecutionFile(files map[string]exactDirectoryFile, binding exactArtifactBinding, seenPaths map[string]struct{}) error {
	if !safeInventoryPath(binding.Path) || !strings.HasPrefix(binding.Path, "artifacts/") {
		return ErrExactInventoryInvalid
	}
	if _, exists := seenPaths[binding.Path]; exists {
		return ErrExactInventoryInvalid
	}
	seenPaths[binding.Path] = struct{}{}
	file, ok := files[binding.Path]
	if !ok || file.bytes != binding.Bytes || file.digest != strings.ToLower(binding.SHA256) {
		return ErrExactInventoryInvalid
	}
	return nil
}

func inspectArtifactTree(root string) (int, int64, string, bool, error) {
	info, err := os.Lstat(root)
	if errors.Is(err, os.ErrNotExist) {
		return 0, 0, "", false, nil
	}
	if err != nil {
		return 0, 0, "", false, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return 0, 0, "", false, errors.New("artifact root is not a real directory")
	}
	type file struct {
		rel  string
		size int64
		sum  string
	}
	var files []file
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return errors.New("artifact tree contains a symbolic link")
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return errors.New("artifact tree contains a non-regular file")
		}
		integrity, err := evidence.InspectFileIntegrity(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, file{rel: filepath.ToSlash(rel), size: integrity.SizeBytes, sum: integrity.SHA256})
		return nil
	})
	if err != nil {
		return 0, 0, "", false, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].rel < files[j].rel })
	aggregate := sha256.New()
	var total int64
	for _, f := range files {
		total += f.size
		_, _ = fmt.Fprintf(aggregate, "%s\x00%d\x00%s\n", f.rel, f.size, f.sum)
	}
	return len(files), total, hex.EncodeToString(aggregate.Sum(nil)), true, nil
}
