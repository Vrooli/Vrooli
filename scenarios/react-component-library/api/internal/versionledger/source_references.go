package versionledger

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// sourceImportRE intentionally accepts static, dynamic, and export imports.
// Cleanup is a safety boundary, so a false positive is preferable to deleting
// a version that a surviving story or component still needs.
var sourceImportRE = regexp.MustCompile(`(?:from\s*|import\s*)["']([^"']+)["']`)

type sourceVersion struct {
	libraryID string
	version   string
	status    string
	directory string
}

type adoptedSourceFile struct {
	adoptionID, scenario, adoptedPath string
	sourceLibraryID, sourceVersion    string
}

func (r *Repository) sourceReferences(ctx context.Context) (map[string][]VersionReference, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT c.library_id, v.version, v.status, v.source_path FROM components c JOIN component_versions v ON v.component_id = c.id`)
	if err != nil {
		// Lightweight lifecycle fixtures and pre-migration stores may not yet
		// carry source_path. Their SQL reference checks remain valid; there is
		// simply no source graph to inspect until the catalog is reindexed.
		if strings.Contains(strings.ToLower(err.Error()), "no such column") {
			return map[string][]VersionReference{}, nil
		}
		return nil, err
	}
	defer rows.Close()
	var versions []sourceVersion
	for rows.Next() {
		var v sourceVersion
		var sourcePath string
		if err := rows.Scan(&v.libraryID, &v.version, &v.status, &sourcePath); err != nil {
			return nil, err
		}
		if strings.TrimSpace(sourcePath) == "" {
			continue
		}
		v.directory = filepath.Clean(filepath.Join(r.sourceRoot, filepath.Dir(sourcePath)))
		versions = append(versions, v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Longest directory first prevents a nested version directory from being
	// attributed to a parent asset when assets happen to share a path prefix.
	sort.Slice(versions, func(i, j int) bool { return len(versions[i].directory) > len(versions[j].directory) })
	byVersion := make(map[string][]VersionReference)
	for _, owner := range versions {
		if strings.EqualFold(owner.status, "retired") {
			continue
		}
		files := []string{}
		if err := filepath.WalkDir(owner.directory, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				if os.IsNotExist(walkErr) {
					return filepath.SkipDir
				}
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			if ext == ".ts" || ext == ".tsx" || ext == ".js" || ext == ".jsx" {
				files = append(files, path)
			}
			return nil
		}); err != nil {
			return nil, fmt.Errorf("scan source references for %s@%s: %w", owner.libraryID, owner.version, err)
		}
		sort.Strings(files)
		for _, file := range files {
			body, err := os.ReadFile(file)
			if err != nil {
				return nil, fmt.Errorf("read source reference file %s: %w", file, err)
			}
			for _, match := range sourceImportRE.FindAllStringSubmatch(string(body), -1) {
				specifier := match[1]
				if !strings.HasPrefix(specifier, ".") {
					continue
				}
				target := resolveSourceImport(filepath.Dir(file), specifier)
				for _, targetVersion := range versions {
					if targetVersion.libraryID == owner.libraryID && targetVersion.version == owner.version {
						continue
					}
					if !pathWithin(targetVersion.directory, target) {
						continue
					}
					key := targetVersion.libraryID + "@" + targetVersion.version
					rel, _ := filepath.Rel(r.sourceRoot, file)
					ref := VersionReference{
						Kind: "source-import", OwnerLibraryID: owner.libraryID, OwnerVersion: owner.version,
						OwnerPath: filepath.ToSlash(rel), ImportSpecifier: specifier,
						Evidence: "relative import resolves into a surviving version folder",
					}
					byVersion[key] = appendUniqueReference(byVersion[key], ref)
					break
				}
			}
		}
	}
	adoptedRefs, err := r.adoptedSourceReferences(ctx)
	if err != nil {
		return nil, err
	}
	for key, refs := range adoptedRefs {
		byVersion[key] = appendUniqueReferences(byVersion[key], refs)
	}
	return byVersion, nil
}

// adoptedSourceReferences follows relative imports inside files copied into a
// scenario. The adoption ledger is the authority for attribution; filesystem
// layout is used only to resolve the importing file and its relative target.
func (r *Repository) adoptedSourceReferences(ctx context.Context) (map[string][]VersionReference, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT f.adoption_id, a.scenario, f.adopted_path, f.source_library_id, f.source_version FROM adoption_files f JOIN adoption_records a ON a.id = f.adoption_id WHERE f.source_library_id <> '' AND f.source_version <> ''`)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no such table") || strings.Contains(strings.ToLower(err.Error()), "no such column") {
			return map[string][]VersionReference{}, nil
		}
		return nil, fmt.Errorf("read adopted source provenance: %w", err)
	}
	var files []adoptedSourceFile
	for rows.Next() {
		var file adoptedSourceFile
		if err := rows.Scan(&file.adoptionID, &file.scenario, &file.adoptedPath, &file.sourceLibraryID, &file.sourceVersion); err != nil {
			rows.Close()
			return nil, err
		}
		files = append(files, file)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	byPath := make(map[string]adoptedSourceFile, len(files))
	for _, file := range files {
		root := filepath.Join(filepath.Dir(r.sourceRoot), file.scenario)
		path := filepath.Clean(filepath.Join(root, filepath.FromSlash(file.adoptedPath)))
		if pathWithin(root, path) {
			byPath[path] = file
		}
	}
	refs := make(map[string][]VersionReference)
	for _, owner := range files {
		root := filepath.Join(filepath.Dir(r.sourceRoot), owner.scenario)
		ownerPath := filepath.Clean(filepath.Join(root, filepath.FromSlash(owner.adoptedPath)))
		body, err := os.ReadFile(ownerPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read adopted source %s: %w", owner.adoptedPath, err)
		}
		for _, match := range sourceImportRE.FindAllStringSubmatch(string(body), -1) {
			specifier := match[1]
			if !strings.HasPrefix(specifier, ".") {
				continue
			}
			target := resolveSourceImport(filepath.Dir(ownerPath), specifier)
			targetFile, ok := byPath[target]
			if !ok || (targetFile.sourceLibraryID == owner.sourceLibraryID && targetFile.sourceVersion == owner.sourceVersion) {
				continue
			}
			key := sourceReferenceKey(targetFile.sourceLibraryID, targetFile.sourceVersion)
			refs[key] = appendUniqueReference(refs[key], VersionReference{
				Kind: "adopted-source-import", OwnerLibraryID: owner.sourceLibraryID,
				OwnerVersion: owner.sourceVersion, OwnerPath: filepath.ToSlash(owner.adoptedPath),
				ImportSpecifier: specifier, Evidence: "adopted file resolves to another adopted source file",
				OwnerScenario: owner.scenario, AdoptionID: owner.adoptionID,
			})
		}
	}
	return refs, nil
}

func resolveSourceImport(directory, specifier string) string {
	base := filepath.Clean(filepath.Join(directory, filepath.FromSlash(specifier)))
	for _, candidate := range []string{base, base + ".ts", base + ".tsx", base + ".js", base + ".jsx", filepath.Join(base, "index.ts"), filepath.Join(base, "index.tsx")} {
		if _, err := os.Stat(candidate); err == nil {
			return filepath.Clean(candidate)
		}
	}
	return base
}

func pathWithin(directory, target string) bool {
	rel, err := filepath.Rel(directory, target)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && !filepath.IsAbs(rel)
}

func appendUniqueReference(references []VersionReference, candidate VersionReference) []VersionReference {
	for _, ref := range references {
		if ref.Kind == candidate.Kind && ref.OwnerLibraryID == candidate.OwnerLibraryID && ref.OwnerVersion == candidate.OwnerVersion && ref.OwnerPath == candidate.OwnerPath && ref.ImportSpecifier == candidate.ImportSpecifier && ref.OwnerScenario == candidate.OwnerScenario && ref.AdoptionID == candidate.AdoptionID {
			return references
		}
	}
	return append(references, candidate)
}

func appendUniqueReferences(references, candidates []VersionReference) []VersionReference {
	for _, candidate := range candidates {
		references = appendUniqueReference(references, candidate)
	}
	return references
}

func sourceReferenceKey(libraryID, version string) string { return libraryID + "@" + version }
