package sketch

import (
	"reflect"
	"strings"
	"testing"
)

func TestPlaceholderRequiresIntent(t *testing.T) {
	_, err := AddPlaceholder(Document{}, Placement{Region: "r", Fills: Fill{Placeholder: "new", Intent: "short"}, State: "invented"})
	if err == nil || !strings.Contains(err.Error(), "at least 20") {
		t.Fatalf("error = %v", err)
	}
}

func TestUnplaceThenRestoreIsLossless(t *testing.T) {
	original := Document{Placements: []Placement{{Region: "r", Fills: Fill{Asset: "forms.input", Version: "1.0.0"}, State: "built", Note: "keep me"}}}
	unplaced, err := Unplace(original, "r", "template changed")
	if err != nil {
		t.Fatal(err)
	}
	restored, err := Restore(unplaced, "forms.input", "r")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(restored, original) {
		t.Fatalf("restored = %#v; original = %#v", restored, original)
	}
}

func TestMutationsAreIdempotent(t *testing.T) {
	p := Placement{Region: "r", Fills: Fill{Asset: "forms.input"}, State: "declared"}
	once, err := Place(Document{}, p)
	if err != nil {
		t.Fatal(err)
	}
	twice, err := Place(once, p)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(once, twice) {
		t.Fatalf("twice = %#v", twice)
	}
	n := Note{Scope: "page", Text: "constraint"}
	if got := AddNote(AddNote(twice, n), n); len(got.Notes) != 1 {
		t.Fatalf("notes = %#v", got.Notes)
	}
}

func TestPlaceholderCanBePlacedAndRestored(t *testing.T) {
	p := Placement{Region: "composer", Fills: Fill{Placeholder: "domain-specific-input", Intent: "Collect the domain-specific response payload"}, State: "invented", Note: "Preserve this explanation"}
	once, err := AddPlaceholder(Document{}, p)
	if err != nil {
		t.Fatal(err)
	}
	twice, err := AddPlaceholder(once, p)
	if err != nil || !reflect.DeepEqual(once, twice) {
		t.Fatalf("placeholder retry changed document: %+v, %v", twice, err)
	}
	removed, err := Unplace(twice, "composer", "Move to new composer slot")
	if err != nil {
		t.Fatal(err)
	}
	restored, err := Restore(removed, p.Fills.Placeholder, "composer")
	if err != nil || !reflect.DeepEqual(restored, once) {
		t.Fatalf("placeholder restore lost data: %+v, %v", restored, err)
	}
}

func TestPlacementKindsAreExclusive(t *testing.T) {
	p := Placement{Region: "composer", Fills: Fill{Asset: "controls.button", Placeholder: "custom", Intent: "Collect the domain-specific response payload"}}
	if _, err := Place(Document{}, p); err == nil {
		t.Fatal("asset placement accepted placeholder identity")
	}
	if _, err := AddPlaceholder(Document{}, p); err == nil {
		t.Fatal("placeholder accepted asset identity")
	}
	p.Fills.Asset = ""
	p.Fills.Version = "1.0.0"
	if _, err := AddPlaceholder(Document{}, p); err == nil {
		t.Fatal("placeholder accepted published version")
	}
}
