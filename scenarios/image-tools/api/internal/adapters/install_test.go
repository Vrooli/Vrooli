package adapters

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"

	apidb "github.com/vrooli/api-core/database"

	db "github.com/vrooli/api-core/databasetest"
	"image-tools/internal/fetch"
)

type downloaderFunc func(ctx context.Context, rawURL, dest string, emit func(int64, int64)) error

func (fn downloaderFunc) Download(ctx context.Context, rawURL, dest string, emit func(int64, int64)) error {
	return fn(ctx, rawURL, dest, emit)
}

const installTestAdapterID = "install-test-adapter"

func newInstaller(t *testing.T) (*Installer, *int) {
	t.Helper()
	d := db.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("ensure schemas: %v", err)
	}
	reg, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	downloads := 0
	payload := bytes.Repeat([]byte("fake-adapter-weights\n"), 128)
	in := &Installer{
		Root:   t.TempDir(),
		Reg:    reg,
		Custom: NewCustomStore(d),
		State:  NewInstallStore(d),
		Download: downloaderFunc(func(_ context.Context, _, dest string, _ func(int64, int64)) error {
			downloads++
			return os.WriteFile(dest, payload, 0o644)
		}),
		DiskAvail: func(string) (int64, error) { return int64(50) << 30, nil },
	}
	return in, &downloads
}

func TestInstallDownloadsAndIsIdempotent(t *testing.T) {
	ctx := context.Background()
	in, downloads := newInstaller(t)
	if err := in.AddCustom(ctx, Adapter{
		ID:           installTestAdapterID,
		Name:         "Install Test",
		Kind:         KindLoRA,
		Architecture: "sd15",
		Weight:       "none",
		ScaleRange:   ScaleRange{Min: 0, Max: 2, Default: 1},
		SizeMBApprox: 8,
		Source: Source{Assets: []fetch.Asset{{
			URL: "https://example.test/a.safetensors", Filename: "a.safetensors", Kind: fetch.ArtifactGeneric, MinBytes: 1,
		}}},
	}); err != nil {
		t.Fatalf("add custom: %v", err)
	}

	rec, err := in.Install(ctx, installTestAdapterID, nil)
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if !rec.Installed || rec.Checksum == "" {
		t.Fatalf("expected installed record with checksum, got %+v", rec)
	}
	if *downloads != 1 {
		t.Fatalf("expected one download, got %d", *downloads)
	}
	if !in.Installed(ctx, installTestAdapterID) {
		t.Fatal("expected Installed=true after install")
	}

	// Idempotent: second install re-downloads nothing.
	if _, err := in.Install(ctx, installTestAdapterID, nil); err != nil {
		t.Fatalf("re-install: %v", err)
	}
	if *downloads != 1 {
		t.Fatalf("re-install should not re-download, got %d", *downloads)
	}

	// Remove clears the record.
	if err := in.Remove(ctx, installTestAdapterID); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if in.Installed(ctx, installTestAdapterID) {
		t.Fatal("expected not installed after remove")
	}
}

func TestAddCustomRejectsSeedCollision(t *testing.T) {
	ctx := context.Background()
	in, _ := newInstaller(t)
	err := in.AddCustom(ctx, Adapter{ID: "lcm-lora-sdv1-5", Name: "dupe", Kind: KindLoRA, Architecture: "sd15"})
	if !errors.Is(err, ErrCustomShadowsSeed) {
		t.Fatalf("expected ErrCustomShadowsSeed, got %v", err)
	}
}

func TestInstallRepoRequiresPinnedRevision(t *testing.T) {
	ctx := context.Background()
	in, _ := newInstaller(t)
	in.RepoFetch = repoFetcherFunc(func(context.Context, fetch.RepoSpec, string, func(int64, int64)) error { return nil })
	if err := in.AddCustom(ctx, Adapter{
		ID: "repo-adapter", Name: "Repo", Kind: KindControlNet, Architecture: "sd15",
		Source: Source{Repo: fetch.RepoSpec{RepoID: "x/y", Revision: ""}},
	}); err != nil {
		t.Fatalf("add custom: %v", err)
	}
	_, err := in.Install(ctx, "repo-adapter", nil)
	if !errors.Is(err, ErrRepoRevisionUnpinned) {
		t.Fatalf("expected ErrRepoRevisionUnpinned, got %v", err)
	}
}

type repoFetcherFunc func(ctx context.Context, repo fetch.RepoSpec, destDir string, emit func(int64, int64)) error

func (fn repoFetcherFunc) Snapshot(ctx context.Context, repo fetch.RepoSpec, destDir string, emit func(int64, int64)) error {
	return fn(ctx, repo, destDir, emit)
}
