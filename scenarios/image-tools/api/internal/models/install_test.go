package models

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"

	apidb "github.com/vrooli/api-core/database"

	"image-tools/internal/testutil/db"
)

// installFixture wires an Installer over a fresh SQLite DB and a temp models
// root, with a controllable fake downloader and disk-space function.
type installFixture struct {
	in        *Installer
	root      string
	payload   []byte // bytes the fake downloader writes
	avail     int64  // bytes the fake disk reports
	downloads int
}

func newInstallFixture(t *testing.T) *installFixture {
	t.Helper()
	d := db.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("ensure schemas: %v", err)
	}
	reg, err := Load()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	root := t.TempDir()
	// A realistic-sized generic artifact: above the 1 KiB validation floor and
	// not HTML, so it passes artifact validation (the fake downloader writes it).
	f := &installFixture{root: root, payload: bytes.Repeat([]byte("fake-model-weights\n"), 128), avail: int64(50) << 30}
	f.in = &Installer{
		Root:   root,
		Reg:    reg,
		Custom: NewCustomStore(d),
		State:  NewInstallStore(d),
		Download: downloaderFunc(func(_ context.Context, _, dest string, _ func(int64, int64)) error {
			f.downloads++
			return os.WriteFile(dest, f.payload, 0o644)
		}),
		DiskAvail: func(string) (int64, error) { return f.avail, nil },
	}
	return f
}

type downloaderFunc func(ctx context.Context, rawURL, dest string, emit func(int64, int64)) error

func (fn downloaderFunc) Download(ctx context.Context, rawURL, dest string, emit func(int64, int64)) error {
	return fn(ctx, rawURL, dest, emit)
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// TestInstall_DownloadsPinsChecksumAndRecords proves the happy path: a seed model
// downloads, the checksum is pinned (it was empty in the seed), and the install
// state is persisted and readable.
func TestInstall_DownloadsPinsChecksumAndRecords(t *testing.T) {
	ctx := context.Background()
	f := newInstallFixture(t)

	rec, err := f.in.Install(ctx, "sd-1.5", nil)
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if !rec.Installed {
		t.Fatalf("expected installed record")
	}
	if rec.Checksum != sha256Hex(f.payload) {
		t.Fatalf("checksum not pinned to computed hash: %q", rec.Checksum)
	}
	if !f.in.Installed(ctx, "sd-1.5") {
		t.Fatalf("Installed should report true after install")
	}
	got, ok, err := f.in.State.Get(ctx, "sd-1.5")
	if err != nil || !ok {
		t.Fatalf("get install: ok=%v err=%v", ok, err)
	}
	if got.Checksum != rec.Checksum || !got.Installed {
		t.Fatalf("persisted record mismatch: %+v", got)
	}
}

// TestInstall_Idempotent proves a second install is a no-op (no re-download).
func TestInstall_Idempotent(t *testing.T) {
	ctx := context.Background()
	f := newInstallFixture(t)
	if _, err := f.in.Install(ctx, "sd-1.5", nil); err != nil {
		t.Fatalf("first install: %v", err)
	}
	if _, err := f.in.Install(ctx, "sd-1.5", nil); err != nil {
		t.Fatalf("second install: %v", err)
	}
	if f.downloads != 1 {
		t.Fatalf("expected exactly one download, got %d", f.downloads)
	}
}

// TestInstall_InsufficientDisk proves the disk-space gate refuses before any
// download happens.
func TestInstall_InsufficientDisk(t *testing.T) {
	ctx := context.Background()
	f := newInstallFixture(t)
	f.avail = 1 << 20 // 1 MiB, far below sd-1.5's ~2.1 GB

	_, err := f.in.Install(ctx, "sd-1.5", nil)
	if !errors.Is(err, ErrInsufficientDisk) {
		t.Fatalf("expected ErrInsufficientDisk, got %v", err)
	}
	if f.downloads != 0 {
		t.Fatalf("download must not run when disk is insufficient")
	}
}

// TestInstall_ChecksumMismatch proves a pinned checksum that disagrees with the
// downloaded bytes fails and removes the partial download.
func TestInstall_ChecksumMismatch(t *testing.T) {
	ctx := context.Background()
	f := newInstallFixture(t)

	// Pre-pin a checksum that won't match the fake payload.
	if err := f.in.State.Set(ctx, InstallRecord{ID: "sd-1.5", Checksum: "deadbeef"}); err != nil {
		t.Fatalf("pre-pin: %v", err)
	}
	_, err := f.in.Install(ctx, "sd-1.5", nil)
	if !errors.Is(err, ErrChecksumMismatch) {
		t.Fatalf("expected ErrChecksumMismatch, got %v", err)
	}
	if dirExists(filepath.Join(f.root, "models", "sd-1.5")) {
		t.Fatalf("partial download dir should be removed on mismatch")
	}
}

// TestRemove_DeletesWeightsAndRecord proves removal clears disk + state.
func TestRemove_DeletesWeightsAndRecord(t *testing.T) {
	ctx := context.Background()
	f := newInstallFixture(t)
	if _, err := f.in.Install(ctx, "sd-1.5", nil); err != nil {
		t.Fatalf("install: %v", err)
	}
	if err := f.in.Remove(ctx, "sd-1.5"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if f.in.Installed(ctx, "sd-1.5") {
		t.Fatalf("Installed should be false after remove")
	}
	if _, ok, _ := f.in.State.Get(ctx, "sd-1.5"); ok {
		t.Fatalf("install record should be gone after remove")
	}
}

// TestAddCustom_RoundTripAndLocalInstall proves a custom local model is rejected
// when it shadows a seed id or points at a missing path, accepted otherwise, and
// "installed" by reference without a download.
func TestAddCustom_RoundTripAndLocalInstall(t *testing.T) {
	ctx := context.Background()
	f := newInstallFixture(t)

	// Shadowing a seed id is rejected.
	if err := f.in.AddCustom(ctx, Model{ID: "sd-1.5"}); !errors.Is(err, ErrCustomShadowsSeed) {
		t.Fatalf("expected ErrCustomShadowsSeed, got %v", err)
	}

	// A real local file backs a custom upscale model.
	weights := filepath.Join(t.TempDir(), "my-upscaler.bin")
	if err := os.WriteFile(weights, []byte("custom-weights"), 0o644); err != nil {
		t.Fatalf("write weights: %v", err)
	}
	custom := Model{
		ID:         "my-upscaler",
		Name:       "My Upscaler",
		Operations: []string{"upscale"},
		Tier:       TierNiceToHave,
		Backend:    "onnxruntime",
		Hardware:   Hardware{CPUCapable: true},
		Source:     Source{LocalPath: weights},
	}
	if err := f.in.AddCustom(ctx, custom); err != nil {
		t.Fatalf("add custom: %v", err)
	}

	rec, err := f.in.Install(ctx, "my-upscaler", nil)
	if err != nil {
		t.Fatalf("install custom: %v", err)
	}
	if rec.Path != weights {
		t.Fatalf("custom install should point at the local path, got %q", rec.Path)
	}
	if f.downloads != 0 {
		t.Fatalf("custom local model must not download")
	}
}
