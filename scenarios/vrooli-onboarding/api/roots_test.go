package main

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/operatorstate"
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

func TestBundleCatalogRelocationPreservesProfileSessionInPinnedStorage(t *testing.T) {
	storageRoot := filepath.Join(t.TempDir(), "operator-storage")
	bundleOne := filepath.Join(t.TempDir(), "onboarding-bundle-v1")
	bundleTwo := filepath.Join(t.TempDir(), "onboarding-bundle-v2")
	previous := rootEnvironment
	rootEnvironment = mapEnvironment{"VROOLI_STORAGE_ROOT": storageRoot, "BUNDLE_ROOT": bundleOne}
	t.Cleanup(func() { rootEnvironment = previous })

	state := operatorstate.New(operatorstate.Config{
		Now: func() time.Time { return time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC) },
		StatePath: func(context.Context) (string, error) {
			roots, err := resolveRoots()
			if err != nil {
				return "", err
			}
			return filepath.Join(roots.StateRoot, operatorstate.StateFile), nil
		},
	})
	ctx := context.Background()
	initial, err := state.Load(ctx)
	if err != nil {
		t.Fatalf("load initial operator state: %v", err)
	}
	answers := map[string]json.RawMessage{
		"release_target": json.RawMessage(`"desktop"`),
		"audience":       json.RawMessage(`"internal-users"`),
	}
	saved, err := state.SaveProfileSession(ctx, operatorstate.ProfileSession{
		Target: "local", Actor: "operator", Mode: "guided", ProfileID: "develop-and-publish",
		ProfileVersion: "1.0.0", CatalogRevision: "catalog-v1", BaseRevision: operatorstate.Revision(initial),
		Answers: answers,
	}, operatorstate.Revision(initial))
	if err != nil {
		t.Fatalf("save profile work: %v", err)
	}

	rootEnvironment = mapEnvironment{"VROOLI_STORAGE_ROOT": storageRoot, "BUNDLE_ROOT": bundleTwo}
	roots, err := resolveRoots()
	if err != nil {
		t.Fatalf("resolve relocated bundle roots: %v", err)
	}
	if roots.CatalogRoot != filepath.Join(bundleTwo, "catalog") {
		t.Fatalf("catalog root = %q, want relocated bundle catalog", roots.CatalogRoot)
	}
	if roots.StateRoot != filepath.Join(storageRoot, "state", "vrooli", onboardingScenarioName) {
		t.Fatalf("state root = %q, want stable operator storage", roots.StateRoot)
	}
	recovered, err := state.ProfileSession(ctx)
	if err != nil {
		t.Fatalf("recover saved profile work: %v", err)
	}
	if recovered == nil || recovered.ProfileID != "develop-and-publish" || recovered.CatalogRevision != "catalog-v1" || operatorstate.Revision(saved) == "" {
		t.Fatalf("recovered profile session = %#v; saved revision=%q", recovered, operatorstate.Revision(saved))
	}
	if string(recovered.Answers["release_target"]) != `"desktop"` || string(recovered.Answers["audience"]) != `"internal-users"` {
		t.Fatalf("recovered answers = %#v", recovered.Answers)
	}
}
