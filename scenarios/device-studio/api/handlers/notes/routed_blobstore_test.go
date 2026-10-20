package notes

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
)

func TestRoutedFilesystemBlobStoreUsesLeasedDataRoot(t *testing.T) {
	primary := filepath.Join(t.TempDir(), "primary")
	leased := filepath.Join(t.TempDir(), "leased")
	roots := filerouting.New(storage.Paths{DataDir: primary})
	if err := roots.InstallTestRoots(storage.Paths{DataDir: leased}, "lease-1", time.Minute); err != nil {
		t.Fatal(err)
	}
	store, err := newRoutedFilesystemBlobStore(roots)
	if err != nil {
		t.Fatal(err)
	}
	ctx := database.WithTestMode(context.Background())
	if err := store.Put(ctx, "notes/n1/a.bin", strings.NewReader("leased bytes"), "application/octet-stream"); err != nil {
		t.Fatal(err)
	}
	leasedPath := filepath.Join(leased, "attachments", "notes", "n1", "a.bin")
	if got, err := os.ReadFile(leasedPath); err != nil || string(got) != "leased bytes" {
		t.Fatalf("leased bytes = %q, %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(primary, "attachments", "notes", "n1", "a.bin")); !os.IsNotExist(err) {
		t.Fatalf("primary attachment unexpectedly exists: %v", err)
	}
	reader, mime, err := store.Get(ctx, "notes/n1/a.bin")
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	got, err := io.ReadAll(reader)
	if err != nil || string(got) != "leased bytes" || mime != "application/octet-stream" {
		t.Fatalf("leased read = %q mime=%q err=%v", got, mime, err)
	}
	if stats := roots.LeaseStats(); stats.TestRootWrites != 1 || stats.PrimaryWritesDuringTestMode != 0 {
		t.Fatalf("file lease stats = %+v, want one isolated write and no primary write", stats)
	}
}

func TestRoutedFilesystemBlobStoreUsesPrimaryOutsideTestMode(t *testing.T) {
	primary := filepath.Join(t.TempDir(), "primary")
	roots := filerouting.New(storage.Paths{DataDir: primary})
	store, err := newRoutedFilesystemBlobStore(roots)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Put(context.Background(), "notes/n1/a.bin", strings.NewReader("ordinary bytes"), "text/plain"); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(filepath.Join(primary, "attachments", "notes", "n1", "a.bin")); err != nil || string(got) != "ordinary bytes" {
		t.Fatalf("primary bytes = %q, %v", got, err)
	}
}

func TestRoutedFilesystemBlobStoreFailsClosedWithoutTestLease(t *testing.T) {
	primary := filepath.Join(t.TempDir(), "primary")
	roots := filerouting.New(storage.Paths{DataDir: primary})
	store, err := newRoutedFilesystemBlobStore(roots)
	if err != nil {
		t.Fatal(err)
	}
	err = store.Put(database.WithTestMode(context.Background()), "notes/n1/a.bin", strings.NewReader("must not persist"), "text/plain")
	if err == nil {
		t.Fatal("test-mode write without lease unexpectedly succeeded")
	}
	if _, statErr := os.Stat(filepath.Join(primary, "attachments", "notes", "n1", "a.bin")); !os.IsNotExist(statErr) {
		t.Fatalf("unleased test-mode request reached primary root: %v", statErr)
	}
}
