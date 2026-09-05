package graphindex_test

import (
	"testing"

	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/signals"
	"architecture-cartographer/internal/signals/graphindex"
)

func TestPackageLookup(t *testing.T) {
	snap := graph.GraphSnapshot{
		Files:    []graph.FileNode{{ID: "file:a", PackageID: "pkg:a"}},
		Packages: []graph.PackageNode{{ID: "pkg:b"}},
	}
	if got := graphindex.PackageForFile("file:a", snap); got != "pkg:a" {
		t.Fatalf("PackageForFile() = %q, want pkg:a", got)
	}
	if got := graphindex.PackageFor("file:a", snap); got != "pkg:a" {
		t.Fatalf("PackageFor(file) = %q, want pkg:a", got)
	}
	if got := graphindex.PackageFor("pkg:b", snap); got != "pkg:b" {
		t.Fatalf("PackageFor(package) = %q, want pkg:b", got)
	}
}

func TestDomainPackagesUsesCanonicalPathMatcherAndCache(t *testing.T) {
	snap := graph.GraphSnapshot{
		Packages: []graph.PackageNode{
			{ID: "pkg:graph", RepoPath: "api/internal/graph"},
			{ID: "pkg:conflicts", RepoPath: "api/internal/conflicts/detectors/layering"},
		},
	}
	dmap := domains.DerivedDomainMap{
		Domains: []domains.DerivedDomain{
			{Name: "graph", Paths: []string{"api/internal/graph/"}},
			{Name: "conflicts", Paths: []string{"api/internal/conflicts/*"}},
		},
	}
	gctx := signals.NewGraphContext("demo", snap, dmap)

	got := graphindex.DomainPackages(gctx)
	if got["pkg:graph"] != "graph" {
		t.Fatalf("graph package mapped to %q, want graph", got["pkg:graph"])
	}
	if got["pkg:conflicts"] != "" {
		t.Fatalf("one-level glob should not match nested package, got %q", got["pkg:conflicts"])
	}

	got["pkg:graph"] = "mutated"
	cached := graphindex.DomainPackages(gctx)
	if cached["pkg:graph"] != "graph" {
		t.Fatalf("cache should return defensive copies, got %q", cached["pkg:graph"])
	}
}

func TestDomainForPath(t *testing.T) {
	dmap := domains.DerivedDomainMap{
		Domains: []domains.DerivedDomain{
			{Name: "graph", Paths: []string{"api/internal/graph/**"}},
		},
	}
	if got := graphindex.DomainForPath("api/internal/graph/service.go", dmap); got != "graph" {
		t.Fatalf("DomainForPath() = %q, want graph", got)
	}
}
