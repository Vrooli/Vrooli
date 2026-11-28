package quarantine

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/vrooli/api-core/blobstore"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
)

func newStore(t *testing.T, retention time.Duration) (*Store, *blobstore.MemoryBlobStore, func(time.Time)) {
	t.Helper()
	db := dbtest.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}
	blobs := blobstore.NewMemoryBlobStore()
	store := New(db, blobs, retention)
	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }
	return store, blobs, func(at time.Time) { now = at }
}

func TestQuarantineRetainsOpensAndPurges(t *testing.T) {
	ctx := context.Background()
	store, blobs, setNow := newStore(t, 7*24*time.Hour)
	key, err := store.Quarantine(ctx, Item{
		JobID: "job-1", OriginalRef: "out/a.png", ReasonCode: "sexual_unassessable",
		SafetyReview: "refused", Score: 0.93, Evidence: `{"reviewer_version":"rev"}`,
		Finding: `{"refuse":true}`, MIME: "image/png",
	}, []byte("pixels"))
	if err != nil {
		t.Fatal(err)
	}
	if !IsKey(key) || key == "" {
		t.Fatalf("key = %q, want a quarantine key", key)
	}
	items, total, err := store.List(ctx, 10, 0)
	if err != nil || total != 1 || len(items) != 1 {
		t.Fatalf("list = %d/%d/%v", total, len(items), err)
	}
	rc, item, err := store.Open(ctx, items[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(rc)
	_ = rc.Close()
	if string(data) != "pixels" || item.ReasonCode != "sexual_unassessable" {
		t.Fatalf("opened = %q/%+v", data, item)
	}

	// Expiry ends review and the janitor removes both the row and the blob.
	setNow(time.Date(2026, 12, 2, 12, 0, 0, 0, time.UTC))
	if _, _, err := store.Open(ctx, items[0].ID); !errors.Is(err, ErrExpired) {
		t.Fatalf("expired open = %v", err)
	}
	if remaining, _, _ := store.List(ctx, 10, 0); len(remaining) != 0 {
		t.Fatalf("expired entry still listed: %v", remaining)
	}
	removed, err := store.PurgeExpired(ctx, 10)
	if err != nil || removed != 1 {
		t.Fatalf("purge = %d/%v", removed, err)
	}
	if _, _, err := blobs.Get(ctx, key); err == nil {
		t.Fatal("purge left the quarantined blob")
	}
}

func TestQuarantineRequiresAJobAndIsDisabledWithoutRetention(t *testing.T) {
	ctx := context.Background()
	store, _, _ := newStore(t, 0)
	if store.Enabled() {
		t.Fatal("zero retention should disable quarantine")
	}
	if _, err := store.Quarantine(ctx, Item{JobID: "job"}, []byte("x")); !errors.Is(err, ErrDisabled) {
		t.Fatalf("disabled quarantine = %v", err)
	}
	enabled, _, _ := newStore(t, time.Hour)
	if _, err := enabled.Quarantine(ctx, Item{OriginalRef: "out/a.png"}, []byte("x")); err == nil {
		t.Fatal("a quarantine without a job id was accepted")
	}
}

func TestIsKeyRejectsLookalikes(t *testing.T) {
	for key, want := range map[string]bool{
		"quarantine/job/1.png": true,
		"quarantine":           false,
		"out/quarantine/1.png": false,
		"out/a.png":            false,
		"":                     false,
	} {
		if got := IsKey(key); got != want {
			t.Errorf("IsKey(%q) = %v, want %v", key, got, want)
		}
	}
}
