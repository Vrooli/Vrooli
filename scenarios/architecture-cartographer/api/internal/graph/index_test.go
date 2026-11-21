package graph

import "testing"

func TestSnapshotIndexResolvesPackageAndFileOwners(t *testing.T) {
	snapshot := GraphSnapshot{
		Packages: []PackageNode{{ID: "package:api", ImportPath: "example/api", RepoPath: "api"}},
		Files:    []FileNode{{ID: "file:api/main.go", Path: "api/main.go", PackageID: "package:api"}},
	}
	index := IndexSnapshot(snapshot)
	if got := index.Package("package:api"); got.ID != "package:api" {
		t.Fatalf("package lookup = %+v", got)
	}
	if got := index.ResolvePackage("file:api/main.go"); got.ID != "package:api" {
		t.Fatalf("file owner lookup = %+v", got)
	}
	if got := index.SymbolPath(SymbolNode{FileID: "file:api/main.go", PackageID: "package:api"}); got != "api/main.go" {
		t.Fatalf("symbol path from file = %q", got)
	}
	if got := index.SymbolPath(SymbolNode{PackageID: "package:api"}); got != "api" {
		t.Fatalf("symbol path package fallback = %q", got)
	}
	if got := index.ResolvePackage("missing"); got.ID != "" {
		t.Fatalf("unknown lookup = %+v, want zero package", got)
	}
}
