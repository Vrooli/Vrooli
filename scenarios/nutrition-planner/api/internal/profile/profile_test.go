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
	p, err = s.Apply(ctx, ApplyInput{WorkspaceID: "w1", Preset: "vegan", Allergies: []string{"peanut"}, Appliances: []string{"stove"}, Cookware: []string{"saucepan"}, Tools: []string{"chef_knife"}})
	if err != nil {
		t.Fatal(err)
	}
	if p.Preset != "vegan" || len(p.ActiveRules) == 0 || len(p.Allergies) != 1 {
		t.Fatalf("apply %#v", p)
	}
	p, err = s.Get(ctx, "w1")
	if err != nil || len(p.Appliances) != 1 || p.Appliances[0] != "stove" || len(p.Cookware) != 1 || p.Cookware[0] != "saucepan" || len(p.Tools) != 1 || p.Tools[0] != "chef_knife" {
		t.Fatalf("equipment selections did not round-trip: %#v, %v", p, err)
	}
	p, err = s.Apply(ctx, ApplyInput{WorkspaceID: "w1", Preset: p.Preset, Appliances: p.Appliances, Cookware: []string{}, Tools: []string{}})
	if err != nil || len(p.Cookware) != 0 || len(p.Tools) != 0 {
		t.Fatalf("empty equipment selections should save: %#v, %v", p, err)
	}
}

func TestSQLiteRepositoryAddsEquipmentColumnsToExistingProfiles(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	legacySchema := `CREATE TABLE profiles (
		workspace_id TEXT PRIMARY KEY, revision INTEGER NOT NULL, preset TEXT NOT NULL,
		preset_version INTEGER NOT NULL, active_rules_json TEXT NOT NULL,
		excluded_groups_json TEXT NOT NULL, allergies_json TEXT NOT NULL,
		appliances_json TEXT NOT NULL, cost_weight TEXT NOT NULL,
		effort_weight TEXT NOT NULL, variety_weight TEXT NOT NULL,
		draft_json TEXT NOT NULL, updated_at TEXT NOT NULL
	)`
	if _, err = db.Exec(legacySchema); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO profiles(workspace_id,revision,preset,preset_version,active_rules_json,excluded_groups_json,allergies_json,appliances_json,cost_weight,effort_weight,variety_weight,draft_json,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`, "w1", 1, "vegan", 1, `[]`, `[]`, `[]`, `["stove"]`, "0.2", "0.4", "0.4", "", "2026-10-09T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	repo := NewSQLiteRepository(db, schedule.System())
	saved, err := repo.Apply(context.Background(), ApplyInput{WorkspaceID: "w1", Preset: "vegan", Appliances: []string{"stove"}, Cookware: []string{"saucepan"}, Tools: []string{"chef_knife"}, CostWeight: .2, EffortWeight: .4, VarietyWeight: .4})
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := repo.Get(context.Background(), "w1")
	if err != nil {
		t.Fatal(err)
	}
	if saved.Revision != 2 || loaded.Revision != 2 || len(loaded.Appliances) != 1 || loaded.Appliances[0] != "stove" || len(loaded.Cookware) != 1 || loaded.Cookware[0] != "saucepan" || len(loaded.Tools) != 1 || loaded.Tools[0] != "chef_knife" {
		t.Fatalf("legacy profile data was not preserved and extended: saved=%#v loaded=%#v", saved, loaded)
	}
	if len(loaded.KitchenDevices) != 0 {
		t.Fatalf("legacy profile should have no inferred physical devices: %#v", loaded.KitchenDevices)
	}
}

func TestKitchenDevicesRoundTripSeparatelyFromEquipmentSelections(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	repo := NewSQLiteRepository(db, schedule.System())
	ctx := context.Background()
	base, err := repo.Apply(ctx, ApplyInput{WorkspaceID: "w1", Preset: "everything", Appliances: []string{"stove"}, Cookware: []string{"saucepan"}, Tools: []string{"spatula"}})
	if err != nil {
		t.Fatal(err)
	}
	want := []KitchenDevice{
		{ID: "device-range-main", Type: "range", Count: 1, Capacity: "5 burners", Capabilities: []string{"cooktop", "oven"}},
		{ID: "device-multicooker-main", Type: "multicooker", Count: 1, Capacity: "6 qt", Capabilities: []string{"pressure_cook"}},
	}
	if _, err = repo.Apply(ctx, ApplyInput{WorkspaceID: "w1", Preset: base.Preset, ExcludedGroups: base.ExcludedGroups, Allergies: base.Allergies, Appliances: base.Appliances, Cookware: base.Cookware, Tools: base.Tools, KitchenDevices: want, CostWeight: base.CostWeight, EffortWeight: base.EffortWeight, VarietyWeight: base.VarietyWeight}); err != nil {
		t.Fatal(err)
	}
	loaded, err := repo.Get(ctx, "w1")
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.KitchenDevices) != 2 || loaded.KitchenDevices[0].ID != want[0].ID || loaded.KitchenDevices[0].Type != "range" || loaded.KitchenDevices[0].Count != 1 || loaded.KitchenDevices[0].Capacity != "5 burners" || len(loaded.KitchenDevices[0].Capabilities) != 2 || loaded.KitchenDevices[1].ID != want[1].ID || loaded.KitchenDevices[1].Capabilities[0] != "pressure_cook" {
		t.Fatalf("physical devices did not round-trip distinctly: %#v", loaded.KitchenDevices)
	}
	if loaded.Revision != base.Revision+1 || len(loaded.Appliances) != 1 || loaded.Appliances[0] != "stove" || len(loaded.Cookware) != 1 || loaded.Cookware[0] != "saucepan" || len(loaded.Tools) != 1 || loaded.Tools[0] != "spatula" {
		t.Fatalf("device save changed category selections or revision: %#v", loaded)
	}
	empty, err := repo.Apply(ctx, ApplyInput{WorkspaceID: "w1", Preset: base.Preset, ExcludedGroups: base.ExcludedGroups, Allergies: base.Allergies, Appliances: base.Appliances, Cookware: base.Cookware, Tools: base.Tools, KitchenDevices: []KitchenDevice{}, CostWeight: base.CostWeight, EffortWeight: base.EffortWeight, VarietyWeight: base.VarietyWeight})
	if err != nil || empty.KitchenDevices == nil || len(empty.KitchenDevices) != 0 {
		t.Fatalf("empty device list should be retained: %#v, %v", empty.KitchenDevices, err)
	}
}
