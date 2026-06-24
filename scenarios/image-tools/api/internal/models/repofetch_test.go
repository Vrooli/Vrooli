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

func TestTreeManifestHashDeterministicAndSensitive(t *testing.T) {
	a := t.TempDir()
	writeTree(t, a)

	h1, size1, err := treeManifestHash(a)
	if err != nil {
		t.Fatalf("hash a: %v", err)
	}
	if len(h1) != 64 {
		t.Fatalf("manifest hash is not 64 hex chars: %q", h1)
	}
	if size1 == 0 {
		t.Fatal("manifest reported zero bytes")
	}

	// Identical tree elsewhere → identical hash (path-relative, order-independent).
	b := t.TempDir()
	writeTree(t, b)
	h2, _, err := treeManifestHash(b)
	if err != nil {
		t.Fatalf("hash b: %v", err)
	}
	if h1 != h2 {
		t.Fatalf("identical trees hashed differently: %s vs %s", h1, h2)
	}

	// A .cache bookkeeping dir must be ignored (HF writes one).
	if err := os.MkdirAll(filepath.Join(b, ".cache", "huggingface"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(b, ".cache", "huggingface", "x"), []byte("noise"), 0o644); err != nil {
		t.Fatal(err)
	}
	h3, _, err := treeManifestHash(b)
	if err != nil {
		t.Fatalf("hash b+cache: %v", err)
	}
	if h3 != h1 {
		t.Fatalf(".cache changed the manifest hash: %s vs %s", h3, h1)
	}

	// Changing a tracked file changes the hash.
	if err := os.WriteFile(filepath.Join(b, "vae", "config.json"), []byte(`{"y":99}`), 0o644); err != nil {
		t.Fatal(err)
	}
	h4, _, err := treeManifestHash(b)
	if err != nil {
		t.Fatalf("hash b changed: %v", err)
	}
	if h4 == h1 {
		t.Fatal("changing a file did not change the manifest hash")
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

func TestHFSnapshotFetcher(t *testing.T) {
	ctx := context.Background()

	// Validation: empty repo_id / revision are refused before invoking python.
	f := &HFSnapshotFetcher{Python: "/nonexistent-python"}
	if err := f.Snapshot(ctx, RepoSource{Revision: "abc"}, t.TempDir(), nil); err == nil {
		t.Fatal("expected error for empty repo_id")
	}
	if err := f.Snapshot(ctx, RepoSource{RepoID: "Org/Repo"}, t.TempDir(), nil); err == nil {
		t.Fatal("expected error for empty revision")
	}

	// Success path via a fake interpreter that mimics snapshot_download: argv is
	// [-c, script, repo, revision, dest, allow...]; it materializes dest/$repo file.
	fakePy := filepath.Join(t.TempDir(), "fakepy.sh")
	script := "#!/bin/sh\n# $1=-c $2=script $3=repo $4=rev $5=dest\nmkdir -p \"$5\" && printf x > \"$5/model_index.json\"\n"
	if err := os.WriteFile(fakePy, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()
	emitted := false
	okF := &HFSnapshotFetcher{Python: fakePy}
	if err := okF.Snapshot(ctx, RepoSource{RepoID: "Org/Repo", Revision: "deadbeef"}, dest, func(_, _ int64) { emitted = true }); err != nil {
		t.Fatalf("snapshot success path: %v", err)
	}
	if !emitted {
		t.Error("expected emit to be called on success")
	}
	if _, err := os.Stat(filepath.Join(dest, "model_index.json")); err != nil {
		t.Errorf("fake fetch did not materialize the tree: %v", err)
	}

	// A non-zero interpreter exit surfaces the captured output.
	failPy := filepath.Join(t.TempDir(), "failpy.sh")
	if err := os.WriteFile(failPy, []byte("#!/bin/sh\necho boom 1>&2\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	badF := &HFSnapshotFetcher{Python: failPy}
	if err := badF.Snapshot(ctx, RepoSource{RepoID: "Org/Repo", Revision: "deadbeef"}, t.TempDir(), nil); err == nil {
		t.Fatal("expected error when interpreter exits non-zero")
	}
}
