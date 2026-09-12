// Package graphindex provides shared graph indexes used by scoring signals.
package graphindex

import (
	"sort"

	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/signals"
)

// PackageForFile returns the package id for a file id in the snapshot.
func PackageForFile(fileID string, snap graph.GraphSnapshot) string {
	files, _ := filePackages(nil, snap)
	return files[fileID]
}

// PackageFor returns the package id for either a file id or package id.
func PackageFor(id string, snap graph.GraphSnapshot) string {
	files, pkgs := filePackages(nil, snap)
	if pkgID := files[id]; pkgID != "" {
		return pkgID
	}
	if _, ok := pkgs[id]; ok {
		return id
	}
	return ""
}

// PackageForFileIn returns the package id for a file using GraphContext caches.
func PackageForFileIn(fileID string, gctx signals.GraphContext) string {
	files, _ := filePackages(gctx.Caches, gctx.Snapshot)
	return files[fileID]
}

// PackageForIn returns the package id for either a file id or package id using
// GraphContext caches.
func PackageForIn(id string, gctx signals.GraphContext) string {
	files, pkgs := filePackages(gctx.Caches, gctx.Snapshot)
	if pkgID := files[id]; pkgID != "" {
		return pkgID
	}
	if _, ok := pkgs[id]; ok {
		return id
	}
	return ""
}

func filePackages(caches *signals.Caches, snap graph.GraphSnapshot) (map[string]string, map[string]struct{}) {
	if caches != nil {
		if files, pkgs := caches.FilePackagesSnapshot(); files != nil || pkgs != nil {
			return files, pkgs
		}
	}
	files := make(map[string]string, len(snap.Files))
	for _, f := range snap.Files {
		if f.ID != "" && f.PackageID != "" {
			files[f.ID] = f.PackageID
		}
	}
	pkgs := make(map[string]struct{}, len(snap.Packages))
	for _, p := range snap.Packages {
		if p.ID != "" {
			pkgs[p.ID] = struct{}{}
		}
	}
	if caches != nil {
		caches.SetFilePackages(files, pkgs)
	}
	return files, pkgs
}

// DomainPackages maps package_id -> domain_name based on the derived domain
// map's path patterns over each package directory.
func DomainPackages(gctx signals.GraphContext) map[string]string {
	if gctx.Caches != nil {
		if cached := gctx.Caches.DomainPackagesSnapshot(); cached != nil {
			return cached
		}
	}
	out := make(map[string]string, len(gctx.Snapshot.Packages))
	for _, p := range gctx.Snapshot.Packages {
		if p.RepoPath == "" {
			continue
		}
		if dom := DomainForPath(p.RepoPath, gctx.DomainMap); dom != "" {
			out[p.ID] = dom
		}
	}
	if gctx.Caches != nil {
		gctx.Caches.SetDomainPackages(out)
	}
	return out
}

// ExportedSymbolsForFile returns exported symbol names for a file id.
func ExportedSymbolsForFile(fileID string, gctx signals.GraphContext) []string {
	byFile := exportedSymbolsByFile(gctx.Caches, gctx.Snapshot)
	return append([]string(nil), byFile[fileID]...)
}

func exportedSymbolsByFile(caches *signals.Caches, snap graph.GraphSnapshot) map[string][]string {
	if caches != nil {
		if cached := caches.ExportedSymbolsByFileSnapshot(); cached != nil {
			return cached
		}
	}
	out := make(map[string][]string)
	for _, s := range snap.Symbols {
		if s.FileID != "" && s.Exported {
			out[s.FileID] = append(out[s.FileID], s.Name)
		}
	}
	for fileID := range out {
		sort.Strings(out[fileID])
	}
	if caches != nil {
		caches.SetExportedSymbolsByFile(out)
	}
	return out
}

// PackageImporters returns package ids importing the given package id.
func PackageImporters(pkgID string, gctx signals.GraphContext) []string {
	byPackage := packageImporters(gctx.Caches, gctx.Snapshot)
	return append([]string(nil), byPackage[pkgID]...)
}

func packageImporters(caches *signals.Caches, snap graph.GraphSnapshot) map[string][]string {
	if caches != nil {
		if cached := caches.PackageImportersSnapshot(); cached != nil {
			return cached
		}
	}
	seen := make(map[string]map[string]struct{})
	ctx := signals.GraphContext{Snapshot: snap, Caches: caches}
	for _, e := range snap.Imports {
		if e.ToPackageID == "" {
			continue
		}
		from := PackageForIn(e.From, ctx)
		if from == "" || from == e.ToPackageID {
			continue
		}
		if seen[e.ToPackageID] == nil {
			seen[e.ToPackageID] = make(map[string]struct{})
		}
		seen[e.ToPackageID][from] = struct{}{}
	}
	out := make(map[string][]string, len(seen))
	for pkgID, importers := range seen {
		out[pkgID] = make([]string, 0, len(importers))
		for importer := range importers {
			out[pkgID] = append(out[pkgID], importer)
		}
		sort.Strings(out[pkgID])
	}
	if caches != nil {
		caches.SetPackageImporters(out)
	}
	return out
}

// PackageImportingTests returns test files importing the given package id.
func PackageImportingTests(pkgID string, gctx signals.GraphContext) []graph.FileNode {
	byPackage := packageImportingTests(gctx.Caches, gctx.Snapshot)
	return append([]graph.FileNode(nil), byPackage[pkgID]...)
}

func packageImportingTests(caches *signals.Caches, snap graph.GraphSnapshot) map[string][]graph.FileNode {
	if caches != nil {
		if cached := caches.PackageImportingTestsSnapshot(); cached != nil {
			return cached
		}
	}
	rawImportersByPackage := make(map[string]map[string]struct{})
	for _, e := range snap.Imports {
		if e.ToPackageID == "" || e.From == "" {
			continue
		}
		if rawImportersByPackage[e.ToPackageID] == nil {
			rawImportersByPackage[e.ToPackageID] = make(map[string]struct{})
		}
		rawImportersByPackage[e.ToPackageID][e.From] = struct{}{}
	}
	testFilesByImporter := make(map[string][]graph.FileNode)
	for _, f := range snap.Files {
		if !f.IsTest {
			continue
		}
		if f.ID != "" {
			testFilesByImporter[f.ID] = append(testFilesByImporter[f.ID], f)
		}
		if f.PackageID != "" {
			testFilesByImporter[f.PackageID] = append(testFilesByImporter[f.PackageID], f)
		}
	}
	out := make(map[string][]graph.FileNode, len(rawImportersByPackage))
	for pkgID, importers := range rawImportersByPackage {
		seenFiles := make(map[string]struct{})
		for importer := range importers {
			for _, f := range testFilesByImporter[importer] {
				if f.ID == "" {
					continue
				}
				if _, ok := seenFiles[f.ID]; ok {
					continue
				}
				seenFiles[f.ID] = struct{}{}
				out[pkgID] = append(out[pkgID], f)
			}
		}
		sort.SliceStable(out[pkgID], func(i, j int) bool {
			return out[pkgID][i].Path < out[pkgID][j].Path
		})
	}
	if caches != nil {
		caches.SetPackageImportingTests(out)
	}
	return out
}

// DomainForPath returns the first declared domain whose path patterns cover
// path. It uses the canonical domains.PathMatches implementation.
func DomainForPath(path string, domainMap domains.DerivedDomainMap) string {
	for _, d := range domainMap.Domains {
		for _, pattern := range d.Paths {
			if domains.PathMatches(path, pattern) {
				return d.Name
			}
		}
	}
	return ""
}
