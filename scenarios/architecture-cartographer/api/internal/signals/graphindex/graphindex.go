// Package graphindex provides shared graph indexes used by scoring signals.
package graphindex

import (
	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/signals"
)

// PackageForFile returns the package id for a file id in the snapshot.
func PackageForFile(fileID string, snap graph.GraphSnapshot) string {
	for _, f := range snap.Files {
		if f.ID == fileID {
			return f.PackageID
		}
	}
	return ""
}

// PackageFor returns the package id for either a file id or package id.
func PackageFor(id string, snap graph.GraphSnapshot) string {
	if pkgID := PackageForFile(id, snap); pkgID != "" {
		return pkgID
	}
	for _, p := range snap.Packages {
		if p.ID == id {
			return p.ID
		}
	}
	return ""
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
