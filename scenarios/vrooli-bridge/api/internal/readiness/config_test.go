package readiness

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestStoreSavedEndpointWinsOverFallback(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	store := NewStore(db, Endpoint{URL: "http://192.168.1.173:18767", Mode: "lan", Source: "derived"})
	if got, err := store.Resolve(context.Background()); err != nil || got.Source != "derived" {
		t.Fatalf("fallback = %#v, %v", got, err)
	}
	if _, err := store.Save(context.Background(), "https://bridge.example.test", "manual"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Save(context.Background(), "https://bridge.example.test", "wireguardish"); err == nil {
		t.Fatal("accepted an invalid reachability mode")
	}
	got, err := store.Resolve(context.Background())
	if err != nil || got.URL != "https://bridge.example.test" || got.Mode != "manual" || got.Source != "configured" {
		t.Fatalf("saved = %#v, %v", got, err)
	}
}
