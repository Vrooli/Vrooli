package fetch

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

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

	h1, size1, err := TreeManifestHash(a)
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
	h2, _, err := TreeManifestHash(b)
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
	h3, _, err := TreeManifestHash(b)
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
	h4, _, err := TreeManifestHash(b)
	if err != nil {
		t.Fatalf("hash b changed: %v", err)
	}
	if h4 == h1 {
		t.Fatal("changing a file did not change the manifest hash")
	}
}

func TestHFSnapshotFetcher(t *testing.T) {
	ctx := context.Background()

	// Validation: empty repo_id / revision are refused before invoking python.
	f := &HFSnapshotFetcher{Python: "/nonexistent-python"}
	if err := f.Snapshot(ctx, RepoSpec{Revision: "abc"}, t.TempDir(), nil); err == nil {
		t.Fatal("expected error for empty repo_id")
	}
	if err := f.Snapshot(ctx, RepoSpec{RepoID: "Org/Repo"}, t.TempDir(), nil); err == nil {
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
	if err := okF.Snapshot(ctx, RepoSpec{RepoID: "Org/Repo", Revision: "deadbeef"}, dest, func(_, _ int64) { emitted = true }); err != nil {
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
	if err := badF.Snapshot(ctx, RepoSpec{RepoID: "Org/Repo", Revision: "deadbeef"}, t.TempDir(), nil); err == nil {
		t.Fatal("expected error when interpreter exits non-zero")
	}
}
