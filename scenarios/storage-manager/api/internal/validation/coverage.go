package validation

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	corestorage "github.com/vrooli/api-core/storage"
)

// storageCoverage verifies the owner namespace, not just the declaration
// spelling. Any bytes below a class owner root that are outside every declared
// entry are reported with their measured size. Missing roots are valid for
// lazy writers and are therefore not a coverage failure.
type storageCoverage struct{}

func init() { register(&storageCoverage{}) }

func (storageCoverage) Name() string { return "storage.coverage" }

func (storageCoverage) Applies(ac AnalyzerContext) bool {
	return ac.Owner != nil
}

func (storageCoverage) Analyze(_ context.Context, ac AnalyzerContext) ([]Finding, error) {
	entries := ac.Owner.StorageEntries
	declared := make([]string, 0, len(entries))
	// Walk every resolver-owned class root. A declaration for one class must
	// not hide writes to a sibling class, and an owner with an empty storage
	// block must still be auditable.
	roots := make([]string, 0, len(entries)+5)
	for _, class := range []corestorage.Class{corestorage.ClassConfig, corestorage.ClassData, corestorage.ClassCache, corestorage.ClassLogs, corestorage.ClassState} {
		root, err := resolveEntry(ac, corestorage.StorageEntry{Rung: corestorage.RungOwned, Kind: "dir", Class: class})
		if err == nil {
			roots = append(roots, filepath.Clean(root))
		}
	}
	for _, entry := range entries {
		path, err := resolveEntry(ac, entry)
		if err != nil {
			if _, ok := err.(*corestorage.NotApplicable); ok {
				continue
			}
			return nil, err
		}
		declared = append(declared, filepath.Clean(path))
		if ac.Owner.Kind != corestorage.OwnerScenario && (entry.Path.ByOS != nil || entry.Path.Value != "") {
			// Explicit upstream/system paths are declared roots themselves, but
			// they do not imply that the class namespace is this owner's write
			// root. Only class-only entries establish a namespace walk.
			continue
		}
		root := path
		if entry.Class != "" {
			canonical := entry
			canonical.Path = corestorage.PortablePath{}
			canonical.Subpath = ""
			root, err = resolveEntry(ac, canonical)
			if err != nil {
				return nil, err
			}
		}
		roots = append(roots, filepath.Clean(root))
	}
	declared = uniquePaths(declared)
	roots = uniquePaths(roots)
	if len(roots) == 0 {
		return nil, nil
	}
	var uncovered int64
	var example string
	const maxUncoveredFiles = 1000
	const maxWalkEntries = 10000
	uncoveredFiles := 0
	walkedEntries := 0
	truncated := false
	for _, root := range roots {
		// A declared root already covers its entire subtree. Do not descend it
		// merely because the canonical class-root list also contains it; this is
		// the common case for large artifact stores.
		if coveredBy(root, declared) {
			continue
		}
		info, err := os.Stat(root)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		const maxTopLevelEntries = 10000
		children, readErr := os.ReadDir(root)
		if readErr != nil {
			return nil, readErr
		}
		if len(children) > maxTopLevelEntries {
			uncovered++
			uncoveredFiles++
			if example == "" {
				example = root
			}
			truncated = true
			continue
		}
		walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if path != root {
				walkedEntries++
				if walkedEntries >= maxWalkEntries {
					truncated = true
					if uncovered == 0 {
						uncovered = 1
						uncoveredFiles = 1
						example = root
					}
					return fs.SkipAll
				}
			}
			if d.IsDir() || coveredBy(path, declared) {
				return nil
			}
			fileInfo, statErr := d.Info()
			if statErr != nil {
				return statErr
			}
			uncovered += fileInfo.Size()
			uncoveredFiles++
			if example == "" {
				example = path
			}
			if uncoveredFiles >= maxUncoveredFiles {
				truncated = true
				return fs.SkipAll
			}
			return nil
		})
		if walkErr != nil {
			return nil, fmt.Errorf("scan owner coverage root %s: %w", root, walkErr)
		}
		_ = info
	}
	if uncovered == 0 {
		return nil, nil
	}
	message := fmt.Sprintf("owner has %d measured bytes outside its declared storage roots; example %q", uncovered, example)
	if truncated {
		message += "; file scan truncated after the configured safety cap"
	}
	return []Finding{{
		Code:        "STORAGE_PATH_UNCOVERED",
		Severity:    SeverityError,
		Title:       "Owner storage is outside declared roots",
		Message:     fmt.Sprintf("owner has %d measured bytes outside its declared storage roots; example %q", uncovered, example),
		Location:    ownerManifestLocation(ac),
		Remediation: "Declare the writer's directory with the resolver-selected class and relative subpath, or remove the undeclared write.",
		Analyzer:    "storage.coverage",
	}}, nil
}

func resolveEntry(ac AnalyzerContext, entry corestorage.StorageEntry) (string, error) {
	return corestorage.ResolveOwnerStoragePath(ac.RepoRoot, *ac.Owner, entry, ac.Platform, corestorage.PlatformSeams{})
}

func coveredBy(path string, roots []string) bool {
	for _, root := range roots {
		if path == root || isWithinPath(path, root) {
			return true
		}
	}
	return false
}

func isWithinPath(path, root string) bool {
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(path))
	return err == nil && rel != ".." && !filepath.IsAbs(rel) && rel != "" && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func uniquePaths(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		result = append(result, path)
	}
	sort.Strings(result)
	return result
}
