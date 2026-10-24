package versionledger

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"react-component-library/internal/librarywalk"
)

// Why this file exists.
//
// Retention used to answer "is this version still needed?" from three sources,
// and all three shared one blind spot. The source scan followed only relative
// specifiers, so it never saw the way library assets actually compose:
//
//	import { IconButton } from "@vrooli/react-component-library/IconButton/2.0.1"
//
// The pinned-dependency table was filled from `component.json.dependencies[]`,
// a field the setup-maturity work retired in favour of per-version locks, so it
// reads a column nothing writes any more. And both of those iterate versions
// found in SQLite, so an asset the indexer never recorded cannot appear as a
// referrer under any of them.
//
// The consequence was a real deletion: IconButton@2.0.1 was retired while 28
// versions still imported it, because the one component that made the edge
// visible had no index row.
//
// `versions/<v>/dependencies.json` is the authoritative per-version record of
// what a version imports. It is generated from parsed source, it exists for
// every source-bearing version, and — the property that matters here — it is on
// disk, so it is true whether or not the index has caught up. Reading it makes
// retention independent of index freshness, which is what let all three guards
// fail at the same time.

// lockDependency mirrors one entry of a generated version lock.
type lockDependency struct {
	LibraryID string `json:"libraryId"`
	Version   string `json:"version"`
}

// versionLock mirrors `versions/<version>/dependencies.json`.
type versionLock struct {
	LibraryID    string           `json:"libraryId"`
	Version      string           `json:"version"`
	Dependencies []lockDependency `json:"dependencies"`
}

// retiredDirName holds quarantined asset trees. Their imports must never keep a
// live version alive: retired source is the one place a dangling reference is
// expected, and honouring it would pin versions to content nothing can reach.
const retiredDirName = ".retired"

// lockReferences builds the reverse dependency graph from every generated
// version lock under the library root: for each lock, the version that owns it
// becomes a referrer of each version it depends on.
//
// The walk is filesystem-first and deliberately independent of the index. A
// version absent from SQLite still contributes its edges, which is exactly the
// case that produced the IconButton deletion.
func (r *Repository) lockReferences(ctx context.Context) (map[string][]VersionReference, error) {
	byVersion := make(map[string][]VersionReference)
	root := filepath.Clean(r.sourceRoot)
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			// A repository pointed at a tree with no library is a legitimate
			// state for lightweight fixtures. It contributes no edges.
			return byVersion, nil
		}
		return nil, fmt.Errorf("inspect library root for version locks: %w", err)
	}

	var lockPaths []string
	err := librarywalk.WalkContext(context.Background(), root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			if os.IsNotExist(walkErr) {
				return filepath.SkipDir
			}
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == retiredDirName || entry.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Name() == "dependencies.json" {
			lockPaths = append(lockPaths, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan version locks: %w", err)
	}
	// Stable order keeps the cleanup plan hash reproducible across runs.
	sort.Strings(lockPaths)

	for _, lockPath := range lockPaths {
		raw, err := os.ReadFile(lockPath)
		if err != nil {
			return nil, fmt.Errorf("read version lock %s: %w", lockPath, err)
		}
		var lock versionLock
		if err := json.Unmarshal(raw, &lock); err != nil {
			// A malformed lock must not silently reduce protection. Retention
			// is a deletion boundary; an unreadable record is a refusal, not an
			// absent edge.
			return nil, fmt.Errorf("decode version lock %s: %w", lockPath, err)
		}
		owner := strings.TrimSpace(lock.LibraryID)
		ownerVersion := strings.TrimSpace(lock.Version)
		if owner == "" || ownerVersion == "" {
			return nil, fmt.Errorf("version lock %s names no owning library id and version", lockPath)
		}
		ownerPath, relErr := filepath.Rel(root, lockPath)
		if relErr != nil {
			ownerPath = lockPath
		}
		for _, dependency := range lock.Dependencies {
			target := strings.TrimSpace(dependency.LibraryID)
			targetVersion := strings.TrimSpace(dependency.Version)
			if target == "" || targetVersion == "" {
				continue
			}
			if target == owner && targetVersion == ownerVersion {
				continue
			}
			key := sourceReferenceKey(target, targetVersion)
			byVersion[key] = appendUniqueReference(byVersion[key], VersionReference{
				Kind:            "version-lock",
				OwnerLibraryID:  owner,
				OwnerVersion:    ownerVersion,
				OwnerPath:       filepath.ToSlash(ownerPath),
				ImportSpecifier: target + "/" + targetVersion,
				Evidence:        "generated version lock records this dependency",
			})
		}
	}
	if err := r.mirroredLockReferences(ctx, byVersion); err != nil {
		return nil, err
	}
	return byVersion, nil
}

// mirroredLockReferences adds edges declared by versions that live in cold
// storage. Eviction moves a version's bytes into the durable mirror and removes
// its directory, so its lock is no longer on disk — but the version is still
// real and can be materialized again. Its dependencies must therefore keep
// their targets alive, or restoring it later would find them gone.
func (r *Repository) mirroredLockReferences(ctx context.Context, byVersion map[string][]VersionReference) error {
	if r.db == nil {
		return nil
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT c.library_id, v.version, f.content
		FROM component_version_files f
		JOIN component_versions v ON v.id = f.version_id
		JOIN components c ON c.id = v.component_id
		WHERE f.path = 'dependencies.json'
		  AND lower(v.presence) = 'evicted'
		  AND lower(v.status) <> 'retired'
		ORDER BY c.library_id, v.version`)
	if err != nil {
		// A store without the mirror tables simply contributes no cold edges.
		lowered := strings.ToLower(err.Error())
		if strings.Contains(lowered, "no such table") || strings.Contains(lowered, "no such column") {
			return nil
		}
		return fmt.Errorf("read mirrored version locks: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var owner, ownerVersion, content string
		if err := rows.Scan(&owner, &ownerVersion, &content); err != nil {
			return err
		}
		var lock versionLock
		if err := json.Unmarshal([]byte(content), &lock); err != nil {
			return fmt.Errorf("decode mirrored version lock for %s@%s: %w", owner, ownerVersion, err)
		}
		for _, dependency := range lock.Dependencies {
			target := strings.TrimSpace(dependency.LibraryID)
			targetVersion := strings.TrimSpace(dependency.Version)
			if target == "" || targetVersion == "" {
				continue
			}
			if target == owner && targetVersion == ownerVersion {
				continue
			}
			key := sourceReferenceKey(target, targetVersion)
			byVersion[key] = appendUniqueReference(byVersion[key], VersionReference{
				Kind:            "version-lock",
				OwnerLibraryID:  owner,
				OwnerVersion:    ownerVersion,
				OwnerPath:       "(cold storage mirror)",
				ImportSpecifier: target + "/" + targetVersion,
				Evidence:        "evicted version's mirrored lock records this dependency",
			})
		}
	}
	return rows.Err()
}

// IndexDrift reports assets and versions that exist on disk but carry no row in
// the index, and versions recorded as materialized that are no longer on disk.
//
// Retention treats this as a refusal condition rather than a warning. Every
// reference query is a search for evidence that a version is still needed, and
// a missing row is indistinguishable from an absent reference — so an index
// that disagrees with the filesystem makes "nothing references this" unsound in
// exactly one direction: the deleting one.
type IndexDrift struct {
	UnindexedAssets   []string
	UnindexedVersions []string
}

// Empty reports whether the index agrees with the filesystem closely enough to
// authorise deletion.
func (d IndexDrift) Empty() bool {
	return len(d.UnindexedAssets) == 0 && len(d.UnindexedVersions) == 0
}

func (d IndexDrift) Error() string {
	var parts []string
	if len(d.UnindexedAssets) > 0 {
		parts = append(parts, fmt.Sprintf("%d asset(s) on disk are missing from the index (%s)", len(d.UnindexedAssets), strings.Join(boundedList(d.UnindexedAssets, 8), ", ")))
	}
	if len(d.UnindexedVersions) > 0 {
		parts = append(parts, fmt.Sprintf("%d version(s) on disk are missing from the index (%s)", len(d.UnindexedVersions), strings.Join(boundedList(d.UnindexedVersions, 8), ", ")))
	}
	return strings.Join(parts, "; ")
}

func boundedList(items []string, limit int) []string {
	if len(items) <= limit {
		return items
	}
	out := append([]string(nil), items[:limit]...)
	return append(out, fmt.Sprintf("and %d more", len(items)-limit))
}

// IndexDrift compares the library tree against the index and reports what the
// index has not seen. Only the deleting direction is reported: a row with no
// directory is the cold-version tier working as designed, while a directory
// with no row is a referrer that cannot defend its dependencies.
func (r *Repository) IndexDrift(ctx context.Context) (IndexDrift, error) {
	var drift IndexDrift
	disk, err := r.diskAssets()
	if err != nil {
		return drift, err
	}
	if len(disk) == 0 {
		return drift, nil
	}
	// Rows are matched to directories through `source_path`, the same link the
	// reference scans already resolve against. Deriving a directory name from
	// `library_id` would bake in a naming convention that the store does not
	// actually guarantee.
	rows, err := r.db.QueryContext(ctx, `SELECT source_path FROM component_versions`)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no such table") || strings.Contains(strings.ToLower(err.Error()), "no such column") {
			// A store with no component tables cannot authorise deletion in the
			// first place; there is no drift to report against it.
			return drift, nil
		}
		return drift, err
	}
	defer rows.Close()
	indexedAssetDirs := map[string]bool{}
	indexedVersionDirs := map[string]bool{}
	for rows.Next() {
		var sourcePath string
		if err := rows.Scan(&sourcePath); err != nil {
			return drift, err
		}
		if strings.TrimSpace(sourcePath) == "" {
			continue
		}
		versionDir := filepath.ToSlash(filepath.Dir(filepath.Clean(filepath.FromSlash(sourcePath))))
		indexedVersionDirs[versionDir] = true
		if assetDir, ok := assetDirOfVersionDir(versionDir); ok {
			indexedAssetDirs[assetDir] = true
		}
	}
	if err := rows.Err(); err != nil {
		return drift, err
	}
	for assetDir, versionDirs := range disk {
		if !indexedAssetDirs[assetDir] {
			drift.UnindexedAssets = append(drift.UnindexedAssets, assetDir)
			// Its versions are covered by the asset-level report; listing them
			// again would bury the names that matter under a longer list.
			continue
		}
		for versionDir := range versionDirs {
			if !indexedVersionDirs[versionDir] {
				drift.UnindexedVersions = append(drift.UnindexedVersions, versionDir)
			}
		}
	}
	sort.Strings(drift.UnindexedAssets)
	sort.Strings(drift.UnindexedVersions)
	return drift, nil
}

// assetDirOfVersionDir turns `components/Button/versions/1.0.0` into
// `components/Button`. It reports false for any shape that is not a version
// directory, so an unexpected layout contributes nothing rather than a wrong
// parent.
func assetDirOfVersionDir(versionDir string) (string, bool) {
	parent := filepath.ToSlash(filepath.Dir(versionDir))
	if filepath.Base(parent) != "versions" {
		return "", false
	}
	return filepath.ToSlash(filepath.Dir(parent)), true
}

// diskAssets enumerates manifest-bearing asset directories and their version
// directories, as library-root-relative slash paths — the same shape the index
// stores in `source_path`.
func (r *Repository) diskAssets() (map[string]map[string]bool, error) {
	out := map[string]map[string]bool{}
	root := filepath.Clean(r.sourceRoot)
	kinds, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return nil, fmt.Errorf("read library root: %w", err)
	}
	for _, kind := range kinds {
		if !kind.IsDir() || kind.Name() == retiredDirName {
			continue
		}
		assets, err := os.ReadDir(filepath.Join(root, kind.Name()))
		if err != nil {
			return nil, fmt.Errorf("read library kind %s: %w", kind.Name(), err)
		}
		for _, asset := range assets {
			if !asset.IsDir() {
				continue
			}
			assetDir := filepath.Join(root, kind.Name(), asset.Name())
			if _, err := os.Stat(filepath.Join(assetDir, "component.json")); err != nil {
				// Only manifest-bearing directories are assets; anything else
				// is shared source the index is not expected to carry.
				continue
			}
			relativeAssetDir := filepath.ToSlash(filepath.Join(kind.Name(), asset.Name()))
			versions := map[string]bool{}
			entries, err := os.ReadDir(filepath.Join(assetDir, "versions"))
			if err == nil {
				for _, entry := range entries {
					if entry.IsDir() {
						versions[relativeAssetDir+"/versions/"+entry.Name()] = true
					}
				}
			} else if !os.IsNotExist(err) {
				return nil, fmt.Errorf("read versions of %s: %w", asset.Name(), err)
			}
			out[relativeAssetDir] = versions
		}
	}
	return out, nil
}
