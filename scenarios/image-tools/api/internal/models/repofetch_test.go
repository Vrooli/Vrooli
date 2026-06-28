package models

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	apidb "github.com/vrooli/api-core/database"

	"image-tools/internal/testutil/db"
)

// fetcherFunc adapts a function to the RepoFetcher seam.
type fetcherFunc func(ctx context.Context, repo RepoSource, dest string, emit func(int64, int64)) error

func (fn fetcherFunc) Snapshot(ctx context.Context, repo RepoSource, dest string, emit func(int64, int64)) error {
	return fn(ctx, repo, dest, emit)
}

// writeTree materializes a small diffusers-like file tree under dir.
func writeTree(t *testing.T, dir string) {
	t.Helper()
	files := map[string]string{
		"model_index.json":                       `{"_class_name":"QwenImageEditPlusPipeline"}`,
		"transformer/config.json":                `{"x":1}`,
		"transformer/diffusion_pytorch_model.sf": "shard-bytes-shard-bytes",
		"vae/config.json":                        `{"y":2}`,
	}
	for rel, content := range files {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
}

// newRepoInstaller wires an Installer with a custom repo-source model and an
// injected RepoFetcher, returning the installer and a counter of fetch calls.
func newRepoInstaller(t *testing.T, fetch RepoFetcher) (*Installer, string) {
	t.Helper()
	d := db.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("ensure schemas: %v", err)
	}
	reg, err := Load()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	in := &Installer{
		Root:      t.TempDir(),
		Reg:       reg,
		Custom:    NewCustomStore(d),
		State:     NewInstallStore(d),
		RepoFetch: fetch,
		DiskAvail: func(string) (int64, error) { return int64(80) << 30, nil },
	}
	const id = "repo-test-model"
	if err := in.AddCustom(context.Background(), Model{
		ID:           id,
		Name:         "Repo Test Model",
		Operations:   []string{"edit_instruct"},
		Tier:         TierQuality,
		Backend:      "diffusers",
		SizeMBApprox: 32,
		Hardware:     Hardware{GPURequired: true},
		Runtime:      Runtime{Family: "qwen-image-edit-plus"},
		Source:       Source{Repo: RepoSource{RepoID: "Org/Repo", Revision: "deadbeefcafef00ddeadbeefcafef00ddeadbeef"}},
	}); err != nil {
		t.Fatalf("add custom repo model: %v", err)
	}
	return in, id
}

func TestInstallRepo_FetchesPinsManifestAndIsIdempotent(t *testing.T) {
	calls := 0
	in, id := newRepoInstaller(t, fetcherFunc(func(_ context.Context, repo RepoSource, dest string, _ func(int64, int64)) error {
		calls++
		if repo.Revision == "" {
			t.Error("fetcher received empty revision")
		}
		writeTree(t, dest)
		return nil
	}))

	rec, err := in.Install(context.Background(), id, nil)
	if err != nil {
		t.Fatalf("install repo: %v", err)
	}
	if !rec.Installed || rec.Checksum == "" || len(rec.Checksum) != 64 {
		t.Fatalf("expected installed record with pinned manifest checksum, got %+v", rec)
	}
	if rec.SizeBytes == 0 {
		t.Fatal("expected non-zero size")
	}
	if calls != 1 {
		t.Fatalf("expected exactly one fetch, got %d", calls)
	}

	// Idempotent: a second install does not refetch.
	if _, err := in.Install(context.Background(), id, nil); err != nil {
		t.Fatalf("re-install: %v", err)
	}
	if calls != 1 {
		t.Fatalf("re-install refetched (calls=%d)", calls)
	}
}

func TestInstallRepo_PinnedManifestMismatchFails(t *testing.T) {
	in, id := newRepoInstaller(t, fetcherFunc(func(_ context.Context, _ RepoSource, dest string, _ func(int64, int64)) error {
		writeTree(t, dest)
		return nil
	}))
	// Pre-pin a bogus checksum so the fetched manifest won't match.
	if err := in.State.Set(context.Background(), InstallRecord{ID: id, Checksum: "00000000000000000000000000000000000000000000000000000000000000ff"}); err != nil {
		t.Fatalf("seed pinned checksum: %v", err)
	}
	_, err := in.Install(context.Background(), id, nil)
	if err == nil || !errors.Is(err, ErrChecksumMismatch) {
		t.Fatalf("expected checksum mismatch, got %v", err)
	}
	// The bad snapshot is cleaned up.
	if dirExists(in.modelDir(id)) {
		t.Fatal("model dir should be removed after a checksum mismatch")
	}
}

func TestInstallRepo_RequiresRevisionAndFetcher(t *testing.T) {
	// No fetcher configured → repo install fails with a clear no-source error.
	inNoFetch, id := newRepoInstaller(t, nil)
	if _, err := inNoFetch.Install(context.Background(), id, nil); err == nil {
		t.Fatal("expected error when RepoFetch is nil")
	}

	// Repo source without a pinned revision → refused before any fetch.
	in, _ := newRepoInstaller(t, fetcherFunc(func(_ context.Context, _ RepoSource, dest string, _ func(int64, int64)) error {
		writeTree(t, dest)
		return nil
	}))
	if err := in.Custom.Add(context.Background(), Model{
		ID:           "repo-no-rev",
		Name:         "No Revision",
		Operations:   []string{"edit_instruct"},
		Tier:         TierQuality,
		Backend:      "diffusers",
		SizeMBApprox: 1,
		Hardware:     Hardware{GPURequired: true},
		Runtime:      Runtime{Family: "qwen-image-edit-plus"},
		Source:       Source{Repo: RepoSource{RepoID: "Org/Repo"}},
	}); err != nil {
		t.Fatalf("add no-rev model: %v", err)
	}
	if _, err := in.Install(context.Background(), "repo-no-rev", nil); err == nil {
		t.Fatal("expected error for repo source without a pinned revision")
	}
}
