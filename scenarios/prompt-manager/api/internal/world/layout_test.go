package world

import (
	"os"
	"strings"
	"testing"
)

func TestLoadLayoutEmptyWhenMissing(t *testing.T) {
	s := newTestStore(t)
	layout, err := s.LoadLayout("park")
	if err != nil {
		t.Fatalf("LoadLayout: %v", err)
	}
	if layout.Scene != "park" || len(layout.Overrides) != 0 || len(layout.Decor) != 0 {
		t.Fatalf("expected empty park layout, got %+v", layout)
	}
	if _, err := s.LoadLayout("moon"); err == nil {
		t.Fatal("unknown scene must be rejected")
	}
}

func TestSaveLayoutRoundTripPerScene(t *testing.T) {
	s := newTestStore(t)
	rot := 1.25
	in := Layout{Scene: "office", Overrides: []Override{{PlaceID: "room:team-a", Position: &Vec2{X: 4, Z: -3}, Rotation: &rot}, {PlaceID: "room:team-b", Removed: true}}, Decor: []Decor{{ID: "d1", PropID: "plant_small", Position: Vec2{X: 1, Z: 1}, Rotation: 0.5, Scale: 1}}}
	saved, err := s.SaveLayout(in)
	if err != nil {
		t.Fatalf("SaveLayout: %v", err)
	}
	if saved.UpdatedAt == "" {
		t.Fatal("updatedAt not stamped")
	}
	loaded, err := s.LoadLayout("office")
	if err != nil {
		t.Fatalf("LoadLayout: %v", err)
	}
	if len(loaded.Overrides) != 2 || loaded.Overrides[0].Position.X != 4 || *loaded.Overrides[0].Rotation != rot || !loaded.Overrides[1].Removed {
		t.Fatalf("overrides did not round trip: %+v", loaded.Overrides)
	}
	if len(loaded.Decor) != 1 || loaded.Decor[0].PropID != "plant_small" {
		t.Fatalf("decor did not round trip: %+v", loaded.Decor)
	}
	park, err := s.LoadLayout("park")
	if err != nil || len(park.Overrides) != 0 {
		t.Fatalf("park layout must stay independent: %+v %v", park, err)
	}
}

func TestSaveLayoutValidation(t *testing.T) {
	s := newTestStore(t)
	bad := []struct {
		name   string
		layout Layout
	}{
		{"scene", Layout{Scene: "moon"}},
		{"placeId", Layout{Scene: "park", Overrides: []Override{{PlaceID: ""}}}},
		{"duplicate placeId", Layout{Scene: "park", Overrides: []Override{{PlaceID: "a"}, {PlaceID: "a"}}}},
		{"propId", Layout{Scene: "park", Decor: []Decor{{ID: "x", Scale: 1}}}},
		{"scale", Layout{Scene: "park", Decor: []Decor{{ID: "x", PropID: "p", Scale: 99}}}},
	}
	for _, tc := range bad {
		if _, err := s.SaveLayout(tc.layout); err == nil || !strings.Contains(err.Error(), tc.name) {
			t.Fatalf("%s: expected error naming it, got %v", tc.name, err)
		}
	}
}

func TestLoadLayoutMalformedIsAnError(t *testing.T) {
	s := newTestStore(t)
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(s.layoutPath("park"), []byte("[]"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := s.LoadLayout("park"); err == nil {
		t.Fatal("malformed layout must be an error")
	}
}
