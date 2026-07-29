package channelmanager

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestStoreRoundTrip(t *testing.T) {
	db, e := sql.Open("sqlite", "file:cm-store?mode=memory&cache=shared")
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	if _, e = db.Exec(Schema()); e != nil {
		t.Fatal(e)
	}
	s, e := New([]Platform{{ID: "x", DailyCeiling: 1, ActionKinds: []string{"engage"}, Formats: testFormats()}}, nil)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.CreateIdentity(Identity{ID: "i", PlatformID: "x", Purpose: "brand", EnvironmentRef: "env", VaultRef: "vault://x"}); e != nil {
		t.Fatal(e)
	}
	store := NewStore(db)
	if e = store.Save(context.Background(), s); e != nil {
		t.Fatal(e)
	}
	reloaded, e := New([]Platform{{ID: "x", DailyCeiling: 1, ActionKinds: []string{"engage"}, Formats: testFormats()}}, nil)
	if e != nil {
		t.Fatal(e)
	}
	if e = store.Load(context.Background(), reloaded); e != nil {
		t.Fatal(e)
	}
	if reloaded.Identities["i"].VaultRef != "vault://x" {
		t.Fatal("persisted identity not restored")
	}
}
