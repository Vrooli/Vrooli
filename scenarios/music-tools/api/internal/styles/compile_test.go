package styles

import (
	"testing"

	db "github.com/vrooli/api-core/databasetest"
)

func TestCompileIsDeterministic(t *testing.T) {
	store := NewStore()
	first, err := store.Compile("launch-trap")
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.Compile("launch-trap")
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("compile changed: %#v != %#v", first, second)
	}
	if first.Caption == "" || first.Params.BPM != 144 {
		t.Fatalf("bad compiled style: %#v", first)
	}
}

func TestCustomStyleCannotShadowBuiltin(t *testing.T) {
	_, err := NewStore().Create(Style{ID: "launch-trap", Name: "shadow", Caption: "other", Params: Params{BPM: 120, Duration: 10}})
	if err != ErrBuiltinCollision {
		t.Fatalf("expected builtin collision, got %v", err)
	}
}

func TestCustomStyleRoundTripsThroughStore(t *testing.T) {
	store := NewStore()
	want := Style{ID: "cold-metal", Name: "Cold Metal", Caption: "cold detuned metallic lead", Params: Params{BPM: 128, Duration: 30, Steps: 8, Guidance: 1, Variant: "acestep-v15-turbo"}}
	if _, err := store.Create(want); err != nil {
		t.Fatal(err)
	}
	got, ok := store.Get(want.ID)
	if !ok {
		t.Fatal("created style not found")
	}
	if got.ID != want.ID || got.Caption != want.Caption || got.Params != want.Params {
		t.Fatalf("style changed: got %#v want %#v", got, want)
	}
}

func TestCustomStyleExportsDeletesAndImports(t *testing.T) {
	store := NewStore()
	want, err := store.Create(Style{ID: "portable", Name: "Portable", Caption: "warm synth", Params: Params{BPM: 120, Duration: 20, Variant: "acestep-v15-turbo"}})
	if err != nil {
		t.Fatal(err)
	}
	data, err := store.Export(want.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(want.ID); err != nil {
		t.Fatal(err)
	}
	got, err := store.Import(data)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != want.ID || got.Caption != want.Caption || got.Params != want.Params {
		t.Fatalf("style changed across portable round trip: got %#v want %#v", got, want)
	}
}

func TestCustomStylesPersistThroughDatabaseStore(t *testing.T) {
	d := db.NewSQLite(t)
	if _, err := d.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	first, err := NewStoreWithDB(d)
	if err != nil {
		t.Fatal(err)
	}
	want, err := first.Create(Style{ID: "persisted", Name: "Persisted", Caption: "warm bass", Params: Params{BPM: 118, Duration: 22, Variant: "acestep-v15-turbo"}})
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewStoreWithDB(d)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := second.Get(want.ID)
	if !ok || got.Caption != want.Caption || got.Params != want.Params {
		t.Fatalf("loaded=%+v found=%v want=%+v", got, ok, want)
	}
	if _, err := second.Compile(want.ID); err != nil {
		t.Fatal(err)
	}
	if err := second.Delete(want.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok := second.Get(want.ID); ok {
		t.Fatal("deleted style still present")
	}
}
