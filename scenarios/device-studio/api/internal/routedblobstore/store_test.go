package routedblobstore

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

func TestStoreUsesRequestRoutedDataRoot(t *testing.T) {
	primary, leased := filepath.Join(t.TempDir(), "primary"), filepath.Join(t.TempDir(), "leased")
	roots := filerouting.New(storage.Paths{DataDir: primary})
	if err := roots.InstallTestRoots(storage.Paths{DataDir: leased}, "lease-media", time.Minute); err != nil {
		t.Fatal(err)
	}
	store, err := New(roots, "media")
	if err != nil {
		t.Fatal(err)
	}
	ctx := database.WithTestMode(context.Background())
	if err := store.Put(ctx, "projects/p1/media/object", strings.NewReader("original bytes"), "image/png"); err != nil {
		t.Fatal(err)
	}
	leasedPath := filepath.Join(leased, "media", "projects", "p1", "media", "object")
	if got, err := os.ReadFile(leasedPath); err != nil || string(got) != "original bytes" {
		t.Fatalf("leased object = %q, %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(primary, "media", "projects", "p1", "media", "object")); !os.IsNotExist(err) {
		t.Fatalf("primary object exists under test mode: %v", err)
	}
	r, mime, err := store.Get(ctx, "projects/p1/media/object")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	data, err := io.ReadAll(r)
	if err != nil || string(data) != "original bytes" || mime != "image/png" {
		t.Fatalf("Get = %q, %q, %v", data, mime, err)
	}
	if err := store.Delete(ctx, "projects/p1/media/object"); err != nil {
		t.Fatal(err)
	}
	if stats := roots.LeaseStats(); stats.TestRootWrites != 2 || stats.PrimaryWritesDuringTestMode != 0 {
		t.Fatalf("file lease stats = %+v", stats)
	}
}

func TestStoreFailsClosedForUnleasedTestMode(t *testing.T) {
	primary := filepath.Join(t.TempDir(), "primary")
	store, err := New(filerouting.New(storage.Paths{DataDir: primary}), "media")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Put(database.WithTestMode(context.Background()), "key", strings.NewReader("no"), "text/plain"); err == nil {
		t.Fatal("unleased test-mode write unexpectedly succeeded")
	}
	if _, err := os.Stat(filepath.Join(primary, "media", "key")); !os.IsNotExist(err) {
		t.Fatalf("unleased request wrote to primary root: %v", err)
	}
}

func TestNewRejectsPathNamespaces(t *testing.T) {
	for _, namespace := range []string{"", ".", "..", "nested/path", `nested\path`} {
		if _, err := New(filerouting.New(storage.Paths{DataDir: t.TempDir()}), namespace); err == nil {
			t.Errorf("New namespace %q unexpectedly succeeded", namespace)
		}
	}
}
