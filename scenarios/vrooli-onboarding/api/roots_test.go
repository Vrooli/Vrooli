package main

import (
	"path/filepath"
	"strings"
	"testing"
)

type mapEnvironment map[string]string

func (e mapEnvironment) Getenv(key string) string { return e[key] }

func TestResolveRootsUsesInjectedEnvironment(t *testing.T) {
	previous := rootEnvironment
	rootEnvironment = mapEnvironment{"VROOLI_ROOT": "/fixed-repo"}
	t.Cleanup(func() { rootEnvironment = previous })

	roots, err := resolveRoots()
	if err != nil {
		t.Fatalf("resolveRoots: %v", err)
	}
	if roots.RepoRoot != "/fixed-repo" || strings.Contains(roots.StateRoot, filepath.Join("fixed-repo", ".vrooli")) {
		t.Fatalf("roots = %+v", roots)
	}
	if !strings.HasSuffix(roots.StateRoot, filepath.Join("state", "vrooli", onboardingScenarioName)) {
		t.Fatalf("state root = %q, want a contract-routed onboarding state namespace", roots.StateRoot)
	}
}

func TestResolveRootsCoversEnvironmentCombinations(t *testing.T) {
	cases := []struct {
		name, repo, storage, bundle, source, repoRoot, stateRoot, catalogRoot string
	}{
		{name: "empty"},
		{name: "repo", repo: "/repo", source: "VROOLI_ROOT", repoRoot: "/repo", catalogRoot: "/repo"},
		{name: "storage", storage: "/storage", source: "VROOLI_STORAGE_ROOT"},
		{name: "bundle", bundle: "/bundle", source: "BUNDLE_ROOT", stateRoot: "/bundle/app-data/state/vrooli/vrooli-onboarding", catalogRoot: "/bundle/catalog"},
		{name: "repo-storage", repo: "/repo", storage: "/storage", source: "VROOLI_ROOT", repoRoot: "/repo", stateRoot: "/storage/state/vrooli/vrooli-onboarding", catalogRoot: "/repo"},
		{name: "repo-bundle", repo: "/repo", bundle: "/bundle", source: "VROOLI_ROOT", repoRoot: "/repo", catalogRoot: "/repo"},
		{name: "storage-bundle", storage: "/storage", bundle: "/bundle", source: "VROOLI_STORAGE_ROOT+BUNDLE_ROOT", stateRoot: "/storage/state/vrooli/vrooli-onboarding", catalogRoot: "/bundle/catalog"},
		{name: "all", repo: "/repo", storage: "/storage", bundle: "/bundle", source: "VROOLI_ROOT", repoRoot: "/repo", stateRoot: "/storage/state/vrooli/vrooli-onboarding", catalogRoot: "/repo"},
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
			expectedState, expectedCatalog := tc.stateRoot, tc.catalogRoot
			if expectedState != "" {
				expectedState = filepath.Clean(expectedState)
			}
			if expectedCatalog != "" {
				expectedCatalog = filepath.Clean(expectedCatalog)
			}
			if roots.Source != tc.source || roots.RepoRoot != tc.repo || (expectedState != "" && roots.StateRoot != expectedState) || roots.CatalogRoot != expectedCatalog {
				t.Fatalf("roots = %+v", roots)
			}
			if strings.Contains(roots.StateRoot, filepath.Join("repo", ".vrooli")) {
				t.Fatalf("state root must not be repo-local: %q", roots.StateRoot)
			}
		})
	}
}
