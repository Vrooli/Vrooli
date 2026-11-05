package styles

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	db "github.com/vrooli/api-core/databasetest"
	stylespb "github.com/vrooli/vrooli/packages/proto/gen/go/music-tools/v1/styles"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestLyricsSurvivePortableAndDatabaseRoundTrips(t *testing.T) {
	const lyrics = "[Verse]\nWe write our own tomorrow\nCafé lights come alive\n"
	for _, legacy := range []bool{false, true} {
		t.Run(fmt.Sprint(legacy), func(t *testing.T) {
			d := db.NewSQLite(t)
			if _, err := d.Exec(Schema()); err != nil {
				t.Fatal(err)
			}
			store, err := NewStoreWithDB(d)
			if err != nil {
				t.Fatal(err)
			}
			doc := map[string]any{"id": "song", "name": "Song", "caption": "bright energetic pop", "lyrics": lyrics, "bpm": 150, "duration": 30}
			if legacy {
				delete(doc, "bpm")
				delete(doc, "duration")
				doc["params"] = map[string]any{"bpm": 150, "duration": 30}
			}
			data, _ := json.Marshal(doc)
			if _, err := store.Import(data); err != nil {
				t.Fatal(err)
			}
			reloaded, err := NewStoreWithDB(d)
			if err != nil {
				t.Fatal(err)
			}
			exported, err := reloaded.Export("song")
			if err != nil {
				t.Fatal(err)
			}
			var wire stylespb.Style
			if err := protojson.Unmarshal(exported, &wire); err != nil || wire.Lyrics != lyrics {
				t.Fatalf("lyrics lost: %q err=%v", wire.Lyrics, err)
			}
			copy := NewStore()
			if _, err := copy.Import(exported); err != nil {
				t.Fatal(err)
			}
			again, _ := copy.Export("song")
			if err := protojson.Unmarshal(again, &wire); err != nil || wire.Lyrics != lyrics {
				t.Fatalf("portable lyrics changed: %q err=%v", wire.Lyrics, err)
			}
		})
	}
}

func TestPortableStyleRejectsUnrecognizedOrInvalidFields(t *testing.T) {
	for name, payload := range map[string]string{
		"unknown canonical field": `{"id":"invalid","name":"Invalid","caption":"drums","bpm":200,"duration":48,"guidence_scale":1}`,
		"invalid canonical value": `{"id":"invalid","name":"Invalid","caption":"drums","bpm":"fast","duration":48}`,
		"unknown nested field":    `{"id":"invalid","name":"Invalid","caption":"drums","params":{"bpm":200,"duration":48,"stepz":8}}`,
		"mixed representations":   `{"id":"invalid","name":"Invalid","caption":"drums","bpm":200,"params":{"duration":48}}`,
		"trailing legacy value":   `{"id":"invalid","name":"Invalid","caption":"drums","params":{"bpm":200,"duration":48}} {}`,
	} {
		t.Run(name, func(t *testing.T) {
			store := NewStore()
			if _, err := store.Import([]byte(payload)); !errors.Is(err, ErrDecode) {
				t.Fatalf("invalid portable style must return a decode error, got %v", err)
			}
			if _, ok := store.Get("invalid"); ok {
				t.Fatal("rejected style was persisted")
			}
		})
	}
}

func TestPortableStyleKeepsExistingCreationAndBuiltinRules(t *testing.T) {
	store := NewStore()
	// Creation still permits an incomplete draft; compilation owns BPM/duration requirements.
	created, err := store.Import([]byte(`{"id":"draft","name":"Draft","caption":"drums","builtin":true}`))
	if err != nil || created.Builtin || created.CreatedAt.IsZero() || created.CreatedAt != created.UpdatedAt {
		t.Fatalf("draft/custom creation changed: %+v err=%v", created, err)
	}
	if _, err := store.Compile(created.ID); !errors.Is(err, ErrInvalid) {
		t.Fatalf("incomplete draft compile error=%v", err)
	}
	if _, err := store.Import([]byte(`{"id":"launch-trap","name":"Shadow","caption":"drums","bpm":200,"duration":48}`)); !errors.Is(err, ErrBuiltinCollision) {
		t.Fatalf("built-in collision error=%v", err)
	}
}

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

func TestCompileRejectsInvalidSamplerDraftsAndKeepsDefaults(t *testing.T) {
	for _, tc := range []struct {
		name, params string
		valid        bool
	}{
		{"defaults", `"inference_steps":0,"guidance_scale":0`, true},
		{"boundaries", `"inference_steps":100,"guidance_scale":20`, true},
		{"negative steps", `"inference_steps":-1`, false},
		{"too many steps", `"inference_steps":101`, false},
		{"negative guidance", `"guidance_scale":-1`, false},
		{"high guidance", `"guidance_scale":21`, false},
		{"nan guidance", `"guidanceScale":"NaN"`, false},
		{"infinite guidance", `"guidanceScale":"Infinity"`, false},
		{"negative infinite guidance", `"guidanceScale":"-Infinity"`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := NewStore()
			input := `{"id":"draft","name":"Draft","caption":"drums","bpm":150,"duration":30,` + tc.params + `}`
			if _, err := store.Import([]byte(input)); err != nil {
				t.Fatalf("draft creation changed: %v", err)
			}
			compiled, err := store.Compile("draft")
			if tc.valid {
				if err != nil {
					t.Fatalf("valid sampler rejected: %v", err)
				}
				if compiled.StyleID != "draft" {
					t.Fatal("wrong compiled identity")
				}
			} else if !errors.Is(err, ErrInvalid) {
				t.Fatalf("invalid sampler compiled: %+v %v", compiled, err)
			}
		})
	}
}

func TestCompileLyricsRemainBackendNeutralWithoutStructuredTempo(t *testing.T) {
	const text = "[Verse]\nKeep my exact words\n\n[Chorus]\nTogether\n"
	store := NewStore()
	_, err := store.Create(Style{ID: "generic-song", Name: "Generic song", Caption: "a new original song", Lyrics: text, Params: Params{Duration: 30}})
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := store.Compile("generic-song")
	if err != nil || compiled.Lyrics != text || compiled.Params.BPM != 0 {
		t.Fatalf("shared style contract imposed backend controls or changed lyrics: %+v %v", compiled, err)
	}
}
