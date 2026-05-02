package operatingmode

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestOverlayStoreLoadMissingFileReturnsEmpty(t *testing.T) {
	store := NewOverlayStore(filepath.Join(t.TempDir(), "nope.json"))
	overrides, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(overrides) != 0 {
		t.Fatalf("expected empty overrides, got %v", overrides)
	}
}

func TestOverlayStoreSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "overrides.json")
	store := NewOverlayStore(path)
	label := "Custom Loop"
	desc := "Reworded for our team."
	if err := store.Save(ModeHolisticLoop, Override{Label: &label, Description: &desc}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got, ok := loaded[ModeHolisticLoop]
	if !ok {
		t.Fatalf("missing override for %q after round trip", ModeHolisticLoop)
	}
	if got.Label == nil || *got.Label != label {
		t.Fatalf("label mismatch: got %v, want %q", got.Label, label)
	}
	if got.Description == nil || *got.Description != desc {
		t.Fatalf("description mismatch: got %v, want %q", got.Description, desc)
	}
}

func TestOverlayStoreClearRemovesEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "overrides.json")
	store := NewOverlayStore(path)
	label := "Custom"
	if err := store.Save(ModeHolisticLoop, Override{Label: &label}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := store.Save(ModeHolisticLoop, Override{}); err != nil {
		t.Fatalf("Save (clear): %v", err)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, ok := loaded[ModeHolisticLoop]; ok {
		t.Fatalf("overlay still present after clear")
	}
}

func TestOverlayStoreSaveRejectsUnknownMode(t *testing.T) {
	store := NewOverlayStore(filepath.Join(t.TempDir(), "overrides.json"))
	label := "x"
	err := store.Save("does-not-exist", Override{Label: &label})
	if err == nil {
		t.Fatalf("expected error for unknown mode")
	}
	if !strings.Contains(err.Error(), "unknown mode") {
		t.Fatalf("error %q does not mention unknown mode", err)
	}
}

func TestOverlayStorePartialOverrideOnlyTouchesPresentField(t *testing.T) {
	store := NewOverlayStore(filepath.Join(t.TempDir(), "overrides.json"))
	label := "Just Label"
	if err := store.Save(ModeHolisticLoop, Override{Label: &label}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, _ := store.Load()
	got := loaded[ModeHolisticLoop]
	if got.Label == nil || *got.Label != label {
		t.Fatalf("label not persisted: %+v", got)
	}
	if got.Description != nil {
		t.Fatalf("description should be nil, got %v", *got.Description)
	}
}

func TestApplyOverlayMergesIntoDefinition(t *testing.T) {
	def := MustDefinition(ModeHolisticLoop)
	originalLabel := def.Label
	if originalLabel == "" {
		t.Fatalf("expected non-empty default label")
	}
	label := "Loopy"
	desc := "Custom"
	merged := applyOverlay(def, Override{Label: &label, Description: &desc})
	if merged.Label != "Loopy" {
		t.Fatalf("label: got %q, want %q", merged.Label, label)
	}
	if merged.Description != "Custom" {
		t.Fatalf("description: got %q, want %q", merged.Description, desc)
	}

	// Empty-string label is ignored (registry default wins) — empty-string
	// description is honored as a clear.
	emptyLabel := ""
	emptyDesc := ""
	cleared := applyOverlay(def, Override{Label: &emptyLabel, Description: &emptyDesc})
	if cleared.Label != originalLabel {
		t.Fatalf("empty label should fall back: got %q, want %q", cleared.Label, originalLabel)
	}
	if cleared.Description != "" {
		t.Fatalf("empty description override should clear: got %q", cleared.Description)
	}
}
