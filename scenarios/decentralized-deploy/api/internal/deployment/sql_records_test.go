package deployment

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	_ "modernc.org/sqlite"
)

func TestSQLRecordRepositoryPersistsLifecycleReceipts(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "deployment.sqlite")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	first, err := NewSQLRecordRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	want := Record{
		ID:      "showcase-1",
		Storage: StoredContent{Provider: "kubo", CID: "bafy-root", URI: "ipfs://bafy-root", RepoID: "0123456789abcdef0123456789abcdef"},
		Names:   []NamedContent{{Provider: "ens-anvil", Name: "example.eth", Target: "ipfs://bafy-root"}, {Provider: "arns-arlocal", Name: "example", Target: "ipfs://bafy-root"}},
		State:   "deployed",
	}
	if err := first.Put(context.Background(), want); err != nil {
		t.Fatal(err)
	}
	second, err := NewSQLRecordRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	got, err := second.Get(context.Background(), want.ID)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("got=%+v err=%v want=%+v", got, err, want)
	}
	if _, err := second.Get(context.Background(), "missing"); !errors.Is(err, ErrRecordNotFound) {
		t.Fatalf("missing record error=%v", err)
	}
	want.State = "renewed"
	if err := second.Put(context.Background(), want); err != nil {
		t.Fatal(err)
	}
	got, err = first.Get(context.Background(), want.ID)
	if err != nil || got.State != "renewed" {
		t.Fatalf("updated state=%q err=%v", got.State, err)
	}
}
