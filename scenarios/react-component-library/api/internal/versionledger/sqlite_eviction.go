package versionledger

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"react-component-library/internal/librarywalk"
	"react-component-library/internal/sqliterows"
	"strings"

	"github.com/vrooli/api-core/storage"
)

type mirroredFileDigest struct{ path, digest string }

// removeReleaseLedgerEntries keeps the append-only hash ledger aligned with
// the governed removal of a version. Derived locks are deliberately included
// in this cleanup even though they are not immutable; authored entries for a
// removed version must not become orphaned integrity failures.
func (r *Repository) removeReleaseLedgerEntries(sourcePath string) error {
	root, err := r.openSourceRoot()
	if err != nil {
		return err
	}
	defer root.Close()
	const path = "released-version-hashes.json"
	raw, err := readSourceFile(root, path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var ledger struct {
		SchemaVersion int `json:"schemaVersion"`
		Entries       []struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(raw, &ledger); err != nil {
		return err
	}
	prefix := filepath.ToSlash(filepath.Dir(sourcePath)) + "/"
	filtered := ledger.Entries[:0]
	for _, entry := range ledger.Entries {
		if !strings.HasPrefix(filepath.ToSlash(entry.Path), prefix) {
			filtered = append(filtered, entry)
		}
	}
	ledger.Entries = filtered
	out, err := json.MarshalIndent(ledger, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	mode, err := sourceFileMode(root, path)
	if err != nil {
		return err
	}
	return storage.WriteFileAtomicInRoot(root, path, out, mode)
}

func (r *Repository) reclaimRetiredMaterialization(ctx context.Context, c Candidate, sourcePath, manifestPath string) error {
	root, err := r.openSourceRoot()
	if err != nil {
		return err
	}
	defer root.Close()
	versionPath, err := sourceRelativePath(sourcePath)
	if err != nil {
		return fmt.Errorf("refusing to retire path outside library root")
	}
	versionDir := filepath.Dir(versionPath)
	if _, err := root.Stat(versionDir); err == nil {
		if err := r.verifyVersionDirectory(ctx, root, c, versionDir); err != nil {
			return err
		}
		backup, err := r.retiredSourceRelativePath(c.LibraryID, c.Version)
		if err != nil {
			return err
		}
		if _, backupErr := root.Stat(backup); errors.Is(backupErr, os.ErrNotExist) {
			if err := root.MkdirAll(filepath.Dir(backup), 0o755); err != nil {
				return err
			}
			if err := root.Rename(versionDir, backup); err != nil {
				return err
			}
		} else if backupErr != nil {
			return backupErr
		} else if err := root.RemoveAll(versionDir); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := r.updateManifestArray(manifestPath, "evictedVersions", c.Version); err != nil {
		return err
	}
	if _, err := r.db.ExecContext(ctx, `UPDATE component_versions SET status='retired', presence='evicted' WHERE component_id=? AND version=?`, c.ComponentID, c.Version); err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `UPDATE version_ledger SET lifecycle_state='retired' WHERE library_id=? AND version=?`, c.LibraryID, c.Version)
	if err != nil {
		return err
	}
	return r.removeReleaseLedgerEntries(sourcePath)
}

func (r *Repository) verifyVersionDirectory(ctx context.Context, root *os.Root, c Candidate, versionDir string) error {
	files, err := r.mirroredFileDigests(ctx, c.ComponentID, c.Version, "read retirement mirror")
	if err != nil {
		return err
	}
	expected := map[string]string{}
	for _, file := range files {
		expected[filepath.Clean(filepath.FromSlash(file.path))] = file.digest
	}
	if len(expected) == 0 {
		return fmt.Errorf("cannot retire %s@%s: file mirror is empty", c.LibraryID, c.Version)
	}
	seen := map[string]struct{}{}
	absoluteVersionDir := filepath.Join(r.sourceRoot, versionDir)
	if err := librarywalk.WalkContext(context.Background(), absoluteVersionDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(absoluteVersionDir, path)
		if err != nil {
			return err
		}
		digest, ok := expected[filepath.Clean(relative)]
		if !ok {
			return ErrEvictionMirrorMismatch{LibraryID: c.LibraryID, Version: c.Version, Path: filepath.ToSlash(relative), Expected: "absent", Actual: "unexpected"}
		}
		body, err := root.ReadFile(filepath.Join(versionDir, relative))
		if err != nil {
			return err
		}
		actual := fmt.Sprintf("%x", sha256.Sum256(body))
		if actual != digest {
			return ErrEvictionMirrorMismatch{LibraryID: c.LibraryID, Version: c.Version, Path: filepath.ToSlash(relative), Expected: digest, Actual: actual}
		}
		seen[filepath.Clean(relative)] = struct{}{}
		return nil
	}); err != nil {
		return err
	}
	for path, digest := range expected {
		if _, ok := seen[path]; !ok {
			return ErrEvictionMirrorMismatch{LibraryID: c.LibraryID, Version: c.Version, Path: filepath.ToSlash(path), Expected: digest, Actual: "missing"}
		}
	}
	return nil
}

func (r *Repository) validateAssetRetirement(ctx context.Context, c Candidate, draft, manifestPath string) error {
	if strings.TrimSpace(draft) != "" {
		return fmt.Errorf("asset %s has active draft %s", c.LibraryID, draft)
	}
	root, err := r.openSourceRoot()
	if err != nil {
		return err
	}
	defer root.Close()
	manifest, err := sourceRelativePath(manifestPath)
	if err != nil {
		return fmt.Errorf("refusing to inspect manifest outside library root")
	}
	data, err := readSourceFile(root, manifest)
	if err != nil {
		return err
	}
	var doc struct {
		ReplacedBy []string `json:"replacedBy"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return err
	}
	if len(doc.ReplacedBy) == 0 {
		return fmt.Errorf("asset %s cannot retire its latest version without replacedBy metadata", c.LibraryID)
	}
	var remaining int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM component_versions WHERE component_id=? AND version<>? AND lower(status)<>'retired'`, c.ComponentID, c.Version).Scan(&remaining); err != nil {
		return err
	}
	if remaining != 0 {
		return fmt.Errorf("asset %s still has %d non-retired historical version(s)", c.LibraryID, remaining)
	}
	queries := []string{
		`SELECT COUNT(*) FROM adoption_records WHERE component_id=? AND adopted_version=?`,
		`SELECT COUNT(*) FROM adoption_files WHERE source_library_id=? AND source_version=?`,
		`SELECT COUNT(*) FROM component_asset_dependencies WHERE library_id=? AND version=?`,
	}
	for index, query := range queries {
		identity := c.ComponentID
		if index > 0 {
			identity = c.LibraryID
		}
		var count int
		if err := r.db.QueryRowContext(ctx, query, identity, c.Version).Scan(&count); err != nil {
			return err
		}
		if count != 0 {
			return fmt.Errorf("version %s is still referenced and is not safe to retire", c.Version)
		}
	}
	graph, err := r.BuildReachability(ctx)
	if err != nil {
		return fmt.Errorf("build source reference graph: %w", err)
	}
	if len(graph.References[sourceReferenceKey(c.LibraryID, c.Version)]) != 0 || hasUnreadableVersion(graph.Unreadable, c.LibraryID, c.Version) {
		return fmt.Errorf("version %s is still referenced and is not safe to retire", c.Version)
	}
	return nil
}

func (r *Repository) evictVersion(ctx context.Context, c Candidate, version, manifestPath, suppliedPlanHash, scopeComponentID string) error {
	// Preserve the caller's scope spelling (UUID or stable library id) while
	// recomputing the confirmation hash. PlanCleanup includes the scope in its
	// hash, so silently replacing a library id with the resolved UUID would
	// reject a valid operator plan.
	items, currentPlanHash, err := r.PlanCleanup(ctx, CleanupScope{ComponentID: scopeComponentID})
	if err != nil {
		return err
	}
	// The RPC supplies its confirmation through the legacy Transition API. A
	// direct archive call is intentionally rejected unless the caller uses the
	// plan-aware helper below; this keeps old callers from accidentally gaining
	// destructive behavior.
	if suppliedPlanHash == "" || suppliedPlanHash != currentPlanHash {
		return fmt.Errorf("archive plan changed or --plan-hash is missing; regenerate with versions reap")
	}
	allowed := false
	for _, item := range items {
		if item.Candidate.Version == version && item.Eligible {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("version %s is still referenced and is not safe to archive", version)
	}
	var sourcePath string
	if err := r.db.QueryRowContext(ctx, `SELECT source_path FROM component_versions WHERE component_id=? AND version=?`, c.ComponentID, version).Scan(&sourcePath); err != nil {
		return err
	}
	root, err := r.openSourceRoot()
	if err != nil {
		return err
	}
	defer root.Close()
	source, err := sourceRelativePath(sourcePath)
	if err != nil {
		return fmt.Errorf("refusing to archive path outside library root")
	}
	versionDir := filepath.Dir(source)
	files, err := r.mirroredFileDigests(ctx, c.ComponentID, version, "read archive mirror")
	if err != nil {
		return err
	}
	checked := 0
	for _, file := range files {
		filePath, pathErr := sourceRelativePath(filepath.Join(versionDir, filepath.FromSlash(file.path)))
		if pathErr != nil {
			return ErrEvictionMirrorMismatch{LibraryID: c.LibraryID, Version: version, Path: file.path, Expected: file.digest, Actual: "invalid path"}
		}
		actualBytes, readErr := root.ReadFile(filePath)
		if readErr != nil {
			return ErrEvictionMirrorMismatch{LibraryID: c.LibraryID, Version: version, Path: file.path, Expected: file.digest, Actual: "missing"}
		}
		actual := fmt.Sprintf("%x", sha256.Sum256(actualBytes))
		if actual != file.digest {
			return ErrEvictionMirrorMismatch{LibraryID: c.LibraryID, Version: version, Path: file.path, Expected: file.digest, Actual: actual}
		}
		checked++
	}
	if checked == 0 {
		return fmt.Errorf("cannot archive %s@%s: file mirror is empty", c.LibraryID, version)
	}
	if err := root.RemoveAll(versionDir); err != nil {
		return fmt.Errorf("remove archived version directory: %w", err)
	}
	if err := r.updateManifestArray(manifestPath, "evictedVersions", version); err != nil {
		return err
	}
	if _, err := r.db.ExecContext(ctx, `UPDATE component_versions SET presence='evicted' WHERE component_id=? AND version=?`, c.ComponentID, version); err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `UPDATE version_ledger SET lifecycle_state='archived' WHERE library_id=? AND version=?`, c.LibraryID, version)
	if err != nil {
		return err
	}
	return r.removeReleaseLedgerEntries(sourcePath)
}

func (r *Repository) mirroredFileDigests(ctx context.Context, componentID, version, queryContext string) ([]mirroredFileDigest, error) {
	return sqliterows.Collect(ctx, r.db, `SELECT path, content_sha256 FROM component_version_files WHERE version_id=(SELECT id FROM component_versions WHERE component_id=? AND version=?) ORDER BY path`, []any{componentID, version}, false, func(row sqliterows.Scanner) (mirroredFileDigest, error) {
		var file mirroredFileDigest
		if err := row.Scan(&file.path, &file.digest); err != nil {
			return mirroredFileDigest{}, err
		}
		return file, nil
	}, sqliterows.ErrorContext{Query: queryContext})
}

func (r *Repository) retiredSourcePath(libraryID, version string) string {
	key := sha256.Sum256([]byte(libraryID + "@" + version))
	return filepath.Join(r.sourceRoot, ".retired", hex.EncodeToString(key[:]))
}

func (r *Repository) retiredSourceRelativePath(libraryID, version string) (string, error) {
	absolute, err := filepath.Rel(r.sourceRoot, r.retiredSourcePath(libraryID, version))
	if err != nil {
		return "", err
	}
	return sourceRelativePath(absolute)
}

func (r *Repository) restoreRetiredSource(sourcePath, libraryID, version string) error {
	root, err := r.openSourceRoot()
	if err != nil {
		return err
	}
	defer root.Close()
	backup, err := r.retiredSourceRelativePath(libraryID, version)
	if err != nil {
		return err
	}
	if _, err := root.Stat(backup); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	destination, err := sourceRelativePath(sourcePath)
	if err != nil {
		return fmt.Errorf("refusing to restore path outside library root")
	}
	versionDir := filepath.Dir(destination)
	if err := root.MkdirAll(filepath.Dir(versionDir), 0o755); err != nil {
		return err
	}
	_ = root.RemoveAll(versionDir)
	return root.Rename(backup, versionDir)
}

func (r *Repository) updateManifest(manifestPath, version string) error {
	return r.updateManifestArray(manifestPath, "deprecatedVersions", version)
}

func (r *Repository) updateManifestArray(manifestPath, field, version string) error {
	root, err := r.openSourceRoot()
	if err != nil {
		return err
	}
	defer root.Close()
	path, err := sourceRelativePath(manifestPath)
	if err != nil {
		return fmt.Errorf("refusing to update manifest outside library root")
	}
	data, err := readSourceFile(root, path)
	if err != nil {
		return err
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return err
	}
	values := []string{}
	if raw, ok := doc[field].([]any); ok {
		for _, value := range raw {
			if v, ok := value.(string); ok {
				values = append(values, v)
			}
		}
	}
	for _, value := range values {
		if value == version {
			return nil
		}
	}
	values = append(values, version)
	doc[field] = values
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	mode, err := sourceFileMode(root, path)
	if err != nil {
		return err
	}
	return storage.WriteFileAtomicInRoot(root, path, out, mode)
}

func (r *Repository) clearDraftManifest(manifestPath string) error {
	root, err := r.openSourceRoot()
	if err != nil {
		return err
	}
	defer root.Close()
	path, err := sourceRelativePath(manifestPath)
	if err != nil {
		return fmt.Errorf("refusing to update manifest outside library root")
	}
	data, err := readSourceFile(root, path)
	if err != nil {
		return err
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return err
	}
	doc["draft"] = ""
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	mode, err := sourceFileMode(root, path)
	if err != nil {
		return err
	}
	return storage.WriteFileAtomicInRoot(root, path, out, mode)
}
