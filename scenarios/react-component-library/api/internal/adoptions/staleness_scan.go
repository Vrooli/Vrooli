package adoptions

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"react-component-library/internal/librarywalk"

	"react-component-library/internal/components"
)

// ComponentCatalogReader is the library seam used by record-less
// staleness scans. It intentionally matches the subset exposed by the
// components service so callers can reuse the indexed catalog.
type ComponentCatalogReader interface {
	List(ctx context.Context, q components.SearchQuery) ([]components.Component, error)
	GetVersion(ctx context.Context, componentID, version string) (components.ComponentVersion, error)
}

// FileScanFinding reports a vendored component provenance header whose
// version is no longer acceptable against the indexed component catalog.
type FileScanFinding struct {
	Path           string
	LibraryID      string
	ComponentID    string
	AdoptedVersion string
	LatestVersion  string
	Status         LibraryVersionStatus
	Detail         string
}

// ScanVendoredFiles walks root looking for @vrooliComponentSource /
// @vrooliComponentVersion headers and classifies each copy without
// requiring an adoption DB row. This reaches scaffolded scenarios and
// templates that have vendored files but no registry record yet.
func ScanVendoredFiles(ctx context.Context, root string, catalog ComponentCatalogReader) ([]FileScanFinding, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, fmt.Errorf("root is required")
	}
	componentsByLibraryID, err := indexedComponents(ctx, catalog)
	if err != nil {
		return nil, err
	}
	var findings []FileScanFinding
	err = librarywalk.WalkContext(ctx, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if shouldSkipVendoredScanDir(d.Name()) && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		if !isSourceFile(path) {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		libraryID := provenanceField(string(raw), "@vrooliComponentSource")
		adoptedVersion := provenanceField(string(raw), "@vrooliComponentVersion")
		if libraryID == "" && adoptedVersion == "" {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			rel = path
		}
		rel = filepath.ToSlash(rel)
		component, ok := componentsByLibraryID[libraryID]
		if !ok {
			findings = append(findings, FileScanFinding{
				Path: rel, LibraryID: libraryID, AdoptedVersion: adoptedVersion,
				Status: LibraryVersionStatusMissing, Detail: "component removed from library",
			})
			return nil
		}
		row := Adoption{ComponentID: component.ID, LibraryID: component.LibraryID, AdoptedVersion: adoptedVersion}
		version, vErr := catalog.GetVersion(ctx, component.ID, adoptedVersion)
		if vErr != nil {
			findings = append(findings, FileScanFinding{
				Path: rel, LibraryID: libraryID, ComponentID: component.ID, AdoptedVersion: adoptedVersion,
				LatestVersion: firstNonEmpty(component.LatestVersion, component.Version),
				Status:        LibraryVersionStatusUnknown,
				Detail:        fmt.Sprintf("adopted version %s not found in library", emptyOrVersion(adoptedVersion)),
			})
			return nil
		}
		status, detail, _ := libraryStatusFor(row, component, version.Status)
		if status == LibraryVersionStatusBehind || status == LibraryVersionStatusDeprecated {
			findings = append(findings, FileScanFinding{
				Path: rel, LibraryID: libraryID, ComponentID: component.ID, AdoptedVersion: adoptedVersion,
				LatestVersion: firstNonEmpty(component.LatestVersion, component.Version),
				Status:        status,
				Detail:        detail,
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return findings, nil
}

func indexedComponents(ctx context.Context, catalog ComponentCatalogReader) (map[string]components.Component, error) {
	rows, err := catalog.List(ctx, components.SearchQuery{Limit: 1000})
	if err != nil {
		return nil, fmt.Errorf("list components: %w", err)
	}
	out := make(map[string]components.Component, len(rows))
	for _, row := range rows {
		if row.LibraryID != "" {
			out[row.LibraryID] = row
		}
	}
	return out, nil
}

func shouldSkipVendoredScanDir(name string) bool {
	switch name {
	case ".git", "node_modules", "dist", "build", ".vite", "coverage":
		return true
	default:
		return false
	}
}

func isSourceFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".ts", ".tsx", ".js", ".jsx":
		return true
	default:
		return false
	}
}

func provenanceField(source, key string) string {
	re := regexp.MustCompile(regexp.QuoteMeta(key) + `\s+([^\n\r*]+)`)
	matches := re.FindStringSubmatch(source)
	if len(matches) < 2 {
		return ""
	}
	return strings.TrimSpace(matches[1])
}
