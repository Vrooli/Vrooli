package executionwriter

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/google/uuid"
	corestorage "github.com/vrooli/api-core/storage"
	"github.com/vrooli/browser-automation-studio/internal/executioninventory"
	"github.com/vrooli/browser-automation-studio/services/evidence"
)

const purgeInventoryFileName = "purge-inventory.v1.json"

type purgeInventoryFile struct {
	Path   string `json:"path"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
}

type purgeArtifactBinding struct {
	ArtifactID string `json:"artifact_id"`
	Scope      string `json:"scope"`
	Storage    string `json:"storage"`
	Path       string `json:"path,omitempty"`
	ObjectKey  string `json:"object_key,omitempty"`
	Bytes      int64  `json:"bytes"`
	SHA256     string `json:"sha256"`
}

type purgeInventory struct {
	SchemaVersion    string                 `json:"schema_version"`
	ExecutionID      string                 `json:"execution_id"`
	WorkflowID       string                 `json:"workflow_id"`
	Files            []purgeInventoryFile   `json:"files"`
	ArtifactBindings []purgeArtifactBinding `json:"artifact_bindings"`
}

// ReadPurgeInventoryArtifactPaths reads writer-owned path references from an
// already-open execution directory. Recovery uses this to avoid maintaining
// a second, partial decoder for the writer's inventory contract.
func ReadPurgeInventoryArtifactPaths(executionRoot *os.Root, executionID uuid.UUID) (map[string]struct{}, error) {
	paths := make(map[string]struct{})
	if executionRoot == nil || executionID == uuid.Nil {
		return nil, fmt.Errorf("execution inventory read requires an open root and execution UUID")
	}
	info, err := executionRoot.Lstat(purgeInventoryFileName)
	if err != nil {
		if os.IsNotExist(err) {
			return paths, nil
		}
		return nil, fmt.Errorf("inspect execution artifact inventory: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("execution artifact inventory is not a regular file")
	}
	file, err := executionRoot.Open(purgeInventoryFileName)
	if err != nil {
		return nil, fmt.Errorf("open execution artifact inventory: %w", err)
	}
	raw, readErr := io.ReadAll(file)
	closeErr := file.Close()
	if readErr != nil {
		return nil, fmt.Errorf("read execution artifact inventory: %w", readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close execution artifact inventory: %w", closeErr)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, fmt.Errorf("decode execution artifact inventory: %w", err)
	}
	for _, required := range []string{"schema_version", "execution_id", "workflow_id", "files", "artifact_bindings"} {
		if _, ok := fields[required]; !ok {
			return nil, fmt.Errorf("execution artifact inventory is missing %s", required)
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var inventory purgeInventory
	if err := decoder.Decode(&inventory); err != nil {
		return nil, fmt.Errorf("decode execution artifact inventory: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, fmt.Errorf("decode execution artifact inventory: trailing content")
	}
	if inventory.SchemaVersion != "bas-execution-directory/v1" || inventory.ExecutionID != executionID.String() {
		return nil, fmt.Errorf("execution artifact inventory identity is uncertain")
	}
	workflowID, err := uuid.Parse(inventory.WorkflowID)
	if err != nil || workflowID == uuid.Nil {
		return nil, fmt.Errorf("execution artifact inventory workflow identity is uncertain")
	}
	if inventory.Files == nil {
		return nil, fmt.Errorf("execution artifact inventory file list is missing")
	}
	for _, entry := range inventory.Files {
		path, err := validatedInventoryPath(entry.Path)
		if err != nil {
			return nil, err
		}
		if entry.Bytes < 0 || !validPurgeInventoryDigest(entry.SHA256) {
			return nil, fmt.Errorf("execution artifact inventory file receipt is invalid")
		}
		paths[path] = struct{}{}
	}
	for _, binding := range inventory.ArtifactBindings {
		if binding.ArtifactID == "" || binding.Bytes < 0 || !validPurgeInventoryDigest(binding.SHA256) {
			return nil, fmt.Errorf("execution artifact inventory binding is invalid")
		}
		switch binding.Storage {
		case "execution_file":
			if binding.ObjectKey != "" {
				return nil, fmt.Errorf("execution artifact inventory binding is ambiguous")
			}
			path, err := validatedInventoryPath(binding.Path)
			if err != nil {
				return nil, err
			}
			paths[path] = struct{}{}
		case "external":
			if binding.Path != "" || !executioninventory.IsExecutionObjectKey(binding.ObjectKey, executionID.String()) {
				return nil, fmt.Errorf("execution artifact inventory external binding is invalid")
			}
		case "inline":
			if binding.Path != "" || binding.ObjectKey != "" {
				return nil, fmt.Errorf("execution artifact inventory inline binding is ambiguous")
			}
		default:
			return nil, fmt.Errorf("execution artifact inventory binding storage is unknown")
		}
	}
	return paths, nil
}

func validPurgeInventoryDigest(raw string) bool {
	if len(raw) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(raw)
	return err == nil
}

func validatedInventoryPath(raw string) (string, error) {
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(raw)))
	if raw == "" || strings.Contains(raw, `\`) || filepath.IsAbs(raw) || clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || clean != raw {
		return "", fmt.Errorf("execution artifact inventory path is invalid")
	}
	return clean, nil
}

// writePurgeInventory records the complete canonical directory file set after
// each writer mutation. It contains only paths, byte counts and digests.
func (r *FileWriter) writePurgeInventory(ctx context.Context, executionID, workflowID uuid.UUID, bindings ...[]purgeArtifactBinding) error {
	if r == nil || r.root == nil || executionID == uuid.Nil || workflowID == uuid.Nil {
		return fmt.Errorf("execution directory inventory cannot be written")
	}
	root, err := r.root.Root(ctx)
	if err != nil {
		return err
	}
	dir := filepath.Join(root, executionID.String())
	files, err := inventoryDirectoryFiles(dir)
	if err != nil {
		return fmt.Errorf("inventory execution directory: %w", err)
	}
	var artifactBindings []purgeArtifactBinding
	if len(bindings) > 0 {
		artifactBindings = bindings[0]
	} else {
		// Checkpoint writes do not change evidence ownership. Preserve the last
		// writer-authored bindings instead of deriving them from a filesystem scan.
		previous, readErr := os.ReadFile(filepath.Join(dir, purgeInventoryFileName))
		if readErr == nil {
			var old purgeInventory
			if json.Unmarshal(previous, &old) != nil || old.ExecutionID != executionID.String() || old.WorkflowID != workflowID.String() {
				return fmt.Errorf("existing execution inventory is invalid")
			}
			artifactBindings = old.ArtifactBindings
		} else if !os.IsNotExist(readErr) {
			return readErr
		}
	}
	inventory := purgeInventory{SchemaVersion: "bas-execution-directory/v1", ExecutionID: executionID.String(), WorkflowID: workflowID.String(), Files: files, ArtifactBindings: artifactBindings}
	raw, err := json.Marshal(inventory)
	if err != nil {
		return fmt.Errorf("encode execution directory inventory: %w", err)
	}
	if err := corestorage.WriteFileAtomic(filepath.Join(dir, purgeInventoryFileName), raw, 0o600); err != nil {
		return fmt.Errorf("write execution directory inventory: %w", err)
	}
	r.root.RecordWrite(ctx)
	return nil
}

func (r *FileWriter) purgeArtifactBindings(ctx context.Context, executionID uuid.UUID, result *ExecutionResultData) ([]purgeArtifactBinding, error) {
	if result == nil {
		return nil, fmt.Errorf("execution result is unavailable for artifact inventory")
	}
	root, err := r.root.Root(ctx)
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(root, executionID.String())
	result.mu.Lock()
	artifacts := append([]ArtifactData(nil), result.Artifacts...)
	auxiliary := append([]PurgeArtifactReceipt(nil), result.PurgeAuxiliary...)
	result.mu.Unlock()
	out := make([]purgeArtifactBinding, 0, len(artifacts)+len(auxiliary))
	for _, artifact := range artifacts {
		if evidence.KindFor(artifact.ArtifactType).String() == "ARTIFACT_KIND_UNSPECIFIED" {
			continue
		}
		class := artifact.PurgeStorageClass
		if class == "" {
			return nil, fmt.Errorf("evidence artifact %s has no writer storage receipt", artifact.ArtifactID)
		}
		path := ""
		if class == "execution_file" {
			if artifact.PurgeStoragePath == "" {
				return nil, fmt.Errorf("disk-backed evidence artifact %s has no writer path", artifact.ArtifactID)
			}
			path, isExecutionFile, pathErr := classifyPurgeStoragePath(root, dir, executionID, artifact.PurgeStoragePath)
			if pathErr != nil {
				return nil, fmt.Errorf("evidence artifact %s has an unsupported storage path: %w", artifact.ArtifactID, pathErr)
			}
			if !isExecutionFile {
				class = "external"
				path = ""
			}
			path = filepath.ToSlash(path)
		} else if class == "external" && artifact.PurgeStoragePath != "" {
			return nil, fmt.Errorf("external evidence artifact %s has an ambiguous local path", artifact.ArtifactID)
		} else if class != "inline" && class != "external" {
			return nil, fmt.Errorf("evidence artifact %s has an unknown storage class", artifact.ArtifactID)
		}
		if artifact.SizeBytes == nil || strings.TrimSpace(artifact.SHA256) == "" {
			return nil, fmt.Errorf("evidence artifact %s is missing integrity metadata", artifact.ArtifactID)
		}
		objectKey := ""
		if class == "external" {
			objectKey = artifact.PurgeObjectKey
			if !executioninventory.IsExecutionObjectKey(objectKey, executionID.String()) {
				return nil, fmt.Errorf("external evidence artifact %s has no UUID-scoped storage receipt", artifact.ArtifactID)
			}
		} else if artifact.PurgeObjectKey != "" && class == "inline" {
			return nil, fmt.Errorf("inline evidence artifact %s has an ambiguous object key", artifact.ArtifactID)
		}
		out = append(out, purgeArtifactBinding{ArtifactID: artifact.ArtifactID, Scope: "evidence", Storage: class, Path: path, ObjectKey: objectKey, Bytes: *artifact.SizeBytes, SHA256: strings.ToLower(artifact.SHA256)})
	}
	for _, receipt := range auxiliary {
		if receipt.Storage == "external" {
			if receipt.Path != "" || !executioninventory.IsExecutionObjectKey(receipt.ObjectKey, executionID.String()) || receipt.Bytes < 0 || len(receipt.SHA256) != sha256.Size*2 {
				return nil, fmt.Errorf("external auxiliary artifact receipt is invalid")
			}
			if _, err := hex.DecodeString(receipt.SHA256); err != nil {
				return nil, fmt.Errorf("auxiliary artifact digest is invalid")
			}
			out = append(out, purgeArtifactBinding{ArtifactID: receipt.ArtifactID, Scope: "auxiliary", Storage: "external", ObjectKey: receipt.ObjectKey, Bytes: receipt.Bytes, SHA256: strings.ToLower(receipt.SHA256)})
			continue
		}
		path, isExecutionFile, pathErr := classifyPurgeStoragePath(root, dir, executionID, receipt.Path)
		if receipt.Storage != "execution_file" || receipt.Path == "" || pathErr != nil || !isExecutionFile || receipt.Bytes < 0 || len(receipt.SHA256) != sha256.Size*2 {
			return nil, fmt.Errorf("auxiliary artifact receipt is invalid")
		}
		if _, err := hex.DecodeString(receipt.SHA256); err != nil {
			return nil, fmt.Errorf("auxiliary artifact digest is invalid")
		}
		out = append(out, purgeArtifactBinding{ArtifactID: receipt.ArtifactID, Scope: "auxiliary", Storage: receipt.Storage, Path: filepath.ToSlash(path), Bytes: receipt.Bytes, SHA256: strings.ToLower(receipt.SHA256)})
	}
	return out, nil
}

func classifyPurgeStoragePath(root, executionDir string, executionID uuid.UUID, rawPath string) (string, bool, error) {
	path, err := filepath.Abs(filepath.Clean(strings.TrimSpace(rawPath)))
	if err != nil || rawPath == "" {
		return "", false, fmt.Errorf("storage path is invalid")
	}
	if rel, err := filepath.Rel(executionDir, path); err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel) {
		return rel, true, nil
	}
	// FileStorage's generic artifact API keeps objects at recordingsRoot/
	// artifacts/<execution-id>/<name>. Exact execution purge deliberately
	// deletes only recordingsRoot/<execution-id>, so record this location as
	// external and leave it to the storage owner's exact-object cleanup API.
	rel, relErr := filepath.Rel(root, path)
	if relErr == nil && filepath.ToSlash(filepath.Dir(rel)) == "artifacts/"+executionID.String() && filepath.Base(rel) != "." && filepath.Base(rel) != ".." {
		return "", false, nil
	}
	return "", false, fmt.Errorf("storage path is outside the supported BAS roots")
}

func inventoryDirectoryFiles(root string) ([]purgeInventoryFile, error) {
	files := make([]purgeInventoryFile, 0)
	executionID, err := uuid.Parse(filepath.Base(filepath.Clean(root)))
	if err != nil || executionID == uuid.Nil {
		return nil, fmt.Errorf("execution directory has no canonical UUID")
	}
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symbolic link in execution directory")
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if entry.IsDir() {
			if rel != "." && !executioninventory.IsOwnedDirectory(rel, executionID.String()) {
				return fmt.Errorf("unowned execution directory")
			}
			return nil
		}
		if rel == purgeInventoryFileName {
			return nil
		}
		if !executioninventory.IsOwnedPath(rel, executionID.String()) {
			return fmt.Errorf("unowned execution directory file")
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("non-regular file in execution directory")
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		h := sha256.New()
		n, copyErr := io.Copy(h, f)
		closeErr := f.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		files = append(files, purgeInventoryFile{Path: rel, Bytes: n, SHA256: hex.EncodeToString(h.Sum(nil))})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}
