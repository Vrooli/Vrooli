package studio

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestSQLiteStoreSurvivesReload(t *testing.T) {
	db, err := sql.Open("sqlite", "file:studio-store-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	state := New()
	state.Identities["product-v1"] = Identity{ID: "product-v1", Name: "Vrooli", Kind: Product, Traits: map[string]string{"form": "console", "finish": "slate"}}
	state.Specs["spec-1"] = Spec{ID: "spec-1", Template: "{{product}}", Fields: map[string]string{"product": "Vrooli"}, IdentityVersionIDs: []string{"product-v1"}}
	store := NewSQLiteStore(db)
	if err := store.Save(context.Background(), state); err != nil {
		t.Fatal(err)
	}
	reloaded, err := store.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Identities["product-v1"].Name != "Vrooli" || reloaded.Specs["spec-1"].Fields["product"] != "Vrooli" {
		t.Fatalf("state did not survive reload: %#v", reloaded)
	}
}
