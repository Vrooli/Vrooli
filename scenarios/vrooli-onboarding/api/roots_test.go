package main

import (
	"path/filepath"
	"testing"
)

func TestResolveRootsCoversEnvironmentCombinations(t *testing.T) {
	cases := []struct {
		name, repo, storage, bundle, source, repoRoot, storageRoot, catalogRoot string
	}{
		{name: "empty"},
		{name: "repo", repo: "/repo", source: "VROOLI_ROOT", repoRoot: "/repo", storageRoot: "/repo/.vrooli", catalogRoot: "/repo"},
		{name: "storage", storage: "/storage", source: "VROOLI_STORAGE_ROOT", storageRoot: "/storage"},
		{name: "bundle", bundle: "/bundle", source: "BUNDLE_ROOT", storageRoot: "/bundle/app-data", catalogRoot: "/bundle/catalog"},
		{name: "repo-storage", repo: "/repo", storage: "/storage", source: "VROOLI_ROOT", repoRoot: "/repo", storageRoot: "/storage", catalogRoot: "/repo"},
		{name: "repo-bundle", repo: "/repo", bundle: "/bundle", source: "VROOLI_ROOT", repoRoot: "/repo", storageRoot: "/repo/.vrooli", catalogRoot: "/repo"},
		{name: "storage-bundle", storage: "/storage", bundle: "/bundle", source: "VROOLI_STORAGE_ROOT+BUNDLE_ROOT", storageRoot: "/storage", catalogRoot: "/bundle/catalog"},
		{name: "all", repo: "/repo", storage: "/storage", bundle: "/bundle", source: "VROOLI_ROOT", repoRoot: "/repo", storageRoot: "/storage", catalogRoot: "/repo"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("VROOLI_ROOT", tc.repo)
			t.Setenv("VROOLI_STORAGE_ROOT", tc.storage)
			t.Setenv("BUNDLE_ROOT", tc.bundle)
			roots, err := resolveRoots()
			if tc.source == "" {
				if err == nil {
					t.Fatal("empty environment must fail")
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveRoots: %v", err)
			}
			expectedStorage, expectedCatalog := tc.storageRoot, tc.catalogRoot
			if expectedStorage != "" {
				expectedStorage = filepath.Clean(expectedStorage)
			}
			if expectedCatalog != "" {
				expectedCatalog = filepath.Clean(expectedCatalog)
			}
			if roots.Source != tc.source || roots.RepoRoot != tc.repo || roots.StorageRoot != expectedStorage || roots.CatalogRoot != expectedCatalog {
				t.Fatalf("roots = %+v", roots)
			}
		})
	}
}
