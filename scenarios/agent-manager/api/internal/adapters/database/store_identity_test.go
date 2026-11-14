package database

import (
	"path/filepath"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/vrooli/api-core/storage"
)

func TestStoreOwnerIDPersistsAcrossReopenAndIsUniquePerDatabase(t *testing.T) {
	root := t.TempDir()
	open := func(name string) *DB {
		t.Helper()
		dsn, err := storage.SQLiteDSNAt(filepath.Join(root, "store space & plus+ ü", name), storage.SQLiteTuning{JournalSizeLimitBytes: walJournalSizeLimit})
		if err != nil {
			t.Fatal(err)
		}
		raw, err := sqlx.Connect("sqlite", dsn)
		if err != nil {
			t.Fatal(err)
		}
		raw.SetMaxOpenConns(1)
		t.Cleanup(func() { _ = raw.Close() })
		wrapped := NewDB(raw, nil)
		if err := wrapped.InitializeSchema(); err != nil {
			t.Fatal(err)
		}
		return wrapped
	}

	first := open("first.db")
	firstID, err := first.StoreOwnerID(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	reopened := open("first.db")
	reopenedID, err := reopened.StoreOwnerID(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if reopenedID != firstID {
		t.Fatalf("reopened store owner ID = %q, want stable ID %q", reopenedID, firstID)
	}

	second := open("second.db")
	secondID, err := second.StoreOwnerID(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if secondID == firstID {
		t.Fatalf("independent stores share owner ID %q", firstID)
	}
}
