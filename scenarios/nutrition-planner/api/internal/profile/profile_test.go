package profile

import (
	"context"
	"database/sql"
	"testing"

	"github.com/vrooli/api-core/schedule"
	_ "modernc.org/sqlite"
)

func TestPresetExpansionKeepsRestrictionsSeparate(t *testing.T) {
	rules := ExpandPreset("vegan")
	for _, want := range []string{"animal_products", "milk", "eggs", "meat", "fish", "shellfish"} {
		found := false
		for _, got := range rules {
			if got == want {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing rule %q in %#v", want, rules)
		}
	}
	if got := ExpandPreset("unknown"); len(got) != 0 {
		t.Fatalf("unknown preset should fall back to everything, got %#v", got)
	}
}

func TestDraftIsSeparateUntilApply(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	s := NewService(NewSQLiteRepository(db, schedule.System()))
	ctx := context.Background()
	if _, err = s.SaveDraft(ctx, "w1", `{"step":2,"preset":"vegan"}`); err != nil {
		t.Fatal(err)
	}
	p, err := s.Get(ctx, "w1")
	if err != nil {
		t.Fatal(err)
	}
	if p.DraftJSON == "" || p.Preset != "everything" {
		t.Fatalf("draft mutated active profile: %#v", p)
	}
	p, err = s.Apply(ctx, ApplyInput{WorkspaceID: "w1", Preset: "vegan", Allergies: []string{"peanut"}, Appliances: []string{}})
	if err != nil {
		t.Fatal(err)
	}
	if p.Preset != "vegan" || len(p.ActiveRules) == 0 || len(p.Allergies) != 1 {
		t.Fatalf("apply %#v", p)
	}
}
