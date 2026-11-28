package fetch

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const companionSHA = "451f4fe16113bff5a5d2269ed5ad43b0592e9a14"

type companionRecorder struct {
	repoFetcherFunc
	fetched []string
}

func (c *companionRecorder) Companion(_ context.Context, repo RepoSpec, hubCache string) error {
	c.fetched = append(c.fetched, repo.RepoID)
	snapshot := companionSnapshotDir(hubCache, repo)
	if err := os.MkdirAll(snapshot, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(snapshot, "model_index.json"), []byte("{}"), 0o644)
}

func TestHubCompanionsMustBePinnedAndNarrow(t *testing.T) {
	good := RepoSpec{RepoID: "org/config", Revision: companionSHA, AllowPatterns: []string{"*.json"}}
	for name, tc := range map[string]struct {
		mutate func(*RepoSpec)
		want   string
	}{
		"ok":            {func(*RepoSpec) {}, ""},
		"branch":        {func(r *RepoSpec) { r.Revision = "main" }, "40-character commit SHA"},
		"unrestricted":  {func(r *RepoSpec) { r.AllowPatterns = nil }, "allow_patterns is required"},
		"not an org/id": {func(r *RepoSpec) { r.RepoID = "config" }, "org/name"},
	} {
		repo := good
		tc.mutate(&repo)
		err := ValidateHubCompanions(Source{HubCompanions: []RepoSpec{repo}})
		if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
			t.Fatalf("%s: err=%v, want %q", name, err, tc.want)
		}
	}
}

func TestInstallPublishesOnlyAfterCompanionsLandInTheManagedHubCache(t *testing.T) {
	root := useManagedRoot(t)
	fetcher := &companionRecorder{repoFetcherFunc: func(context.Context, RepoSpec, string, func(int64, int64)) error { return nil }}
	acquirer := Acquirer{Root: root, Download: acquireDownloader(func(_ context.Context, _, path string, _ func(int64, int64)) error {
		return os.WriteFile(path, []byte(strings.Repeat("w", 2048)), 0o600)
	}), RepoFetch: fetcher}
	source := Source{
		Assets:        []Asset{{URL: "https://models.example/m.safetensors", Filename: "m.safetensors", MinBytes: 1024}},
		HubCompanions: []RepoSpec{{RepoID: "org/config", Revision: companionSHA, AllowPatterns: []string{"*.json"}}},
	}
	dest := filepath.Join(root, "models", "m")
	if _, err := acquirer.Acquire(context.Background(), source, dest, "", 1, nil); err != nil {
		t.Fatal(err)
	}
	hub := filepath.Join(root, "cache", "huggingface", "hub")
	if len(fetcher.fetched) != 1 || len(MissingCompanions(source)) != 0 {
		t.Fatalf("companion not placed in the managed hub cache: fetched=%v missing=%v", fetcher.fetched, MissingCompanions(source))
	}
	ref, err := os.ReadFile(filepath.Join(hub, "models--org--config", "refs", "main"))
	if err != nil || strings.TrimSpace(string(ref)) != companionSHA {
		t.Fatalf("a pinned companion must resolve as main offline: %q %v", ref, err)
	}
	// Already present: nothing is fetched again.
	if err := acquirer.EnsureCompanions(context.Background(), source, nil); err != nil || len(fetcher.fetched) != 1 {
		t.Fatalf("present companion refetched: %v %v", err, fetcher.fetched)
	}
}

func TestCompanionFailureBlocksPublication(t *testing.T) {
	root := useManagedRoot(t)
	acquirer := Acquirer{Root: root, Download: acquireDownloader(func(_ context.Context, _, path string, _ func(int64, int64)) error {
		return os.WriteFile(path, []byte(strings.Repeat("w", 2048)), 0o600)
	}), RepoFetch: repoFetcherFunc(func(context.Context, RepoSpec, string, func(int64, int64)) error { return nil })}
	source := Source{
		Assets:        []Asset{{URL: "https://models.example/m.safetensors", Filename: "m.safetensors", MinBytes: 1024}},
		HubCompanions: []RepoSpec{{RepoID: "org/config", Revision: companionSHA, AllowPatterns: []string{"*.json"}}},
	}
	dest := filepath.Join(root, "models", "m")
	if _, err := acquirer.Acquire(context.Background(), source, dest, "", 1, nil); err == nil {
		t.Fatal("an install whose companion cannot be placed must not publish")
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatalf("model published without its companion: %v", err)
	}
}
