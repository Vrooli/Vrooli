package models

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"image-tools/internal/catalogstate"

	apidb "github.com/vrooli/api-core/database"

	db "github.com/vrooli/api-core/databasetest"
)

func newStateDB(t *testing.T) *sql.DB {
	t.Helper()
	d := db.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("ensure schemas: %v", err)
	}
	return d
}

func TestStoreOverlayDefaultsToSeed(t *testing.T) {
	st := NewStore(newStateDB(t))
	overlay, err := st.LoadOverlay(context.Background())
	if err != nil {
		t.Fatalf("load overlay: %v", err)
	}
	if len(overlay) != 0 {
		t.Fatalf("fresh overlay should be empty, got %v", overlay)
	}
}

func TestStoreSetOverridePersistsAndUpserts(t *testing.T) {
	ctx := context.Background()
	st := NewStore(newStateDB(t))

	if err := st.SetOverride(ctx, catalogstate.Override{ID: "sd-1.5", Enabled: false, SetBy: "test"}, true, time.Now()); err != nil {
		t.Fatalf("set enabled false: %v", err)
	}
	overlay, err := st.LoadOverlay(ctx)
	if err != nil {
		t.Fatalf("load overlay: %v", err)
	}
	if v, ok := overlay["sd-1.5"]; !ok || v {
		t.Fatalf("expected sd-1.5 overridden to false, got %v (present=%v)", v, ok)
	}

	// Re-toggle: upsert must replace, not insert a duplicate.
	if err := st.SetOverride(ctx, catalogstate.Override{ID: "sd-1.5", Enabled: true, SetBy: "test"}, true, time.Now()); err != nil {
		t.Fatalf("set enabled true: %v", err)
	}
	overlay, err = st.LoadOverlay(ctx)
	if err != nil {
		t.Fatalf("reload overlay: %v", err)
	}
	if len(overlay) != 1 {
		t.Fatalf("upsert should keep one row, got %d", len(overlay))
	}
	if !overlay["sd-1.5"] {
		t.Fatalf("expected sd-1.5 re-enabled to true")
	}
}

func TestStoreSetOverrideRejectsEmptyID(t *testing.T) {
	st := NewStore(newStateDB(t))
	if err := st.SetOverride(context.Background(), catalogstate.Override{ID: "", Enabled: true, SetBy: "test"}, true, time.Now()); err == nil {
		t.Fatal("expected error for empty model id")
	}
}

func TestEnabledWithOverlayPrecedence(t *testing.T) {
	r, err := Load()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	// Pick a seeded-enabled default model to flip off via overlay.
	var enabledID string
	for _, m := range r.Models() {
		if m.Enabled {
			enabledID = m.ID
			break
		}
	}
	if enabledID == "" {
		t.Fatal("seed has no enabled model to exercise overlay")
	}

	overlay := map[string]bool{enabledID: false}
	isEnabled := r.EnabledWithOverlay(overlay)
	if isEnabled(enabledID) {
		t.Fatalf("overlay should disable %q", enabledID)
	}

	m, _ := r.ByID(enabledID)
	if EffectiveEnabled(m, overlay) {
		t.Fatalf("EffectiveEnabled should reflect overlay disable for %q", enabledID)
	}
	if EffectiveEnabled(m, nil) != m.Enabled {
		t.Fatal("nil overlay should fall back to seed default")
	}

	// A model absent from the overlay keeps its seed default.
	if isEnabled(enabledID + "-does-not-exist") {
		t.Fatal("unknown id should report not-enabled")
	}
}

// The overlay is honoured for every license class: the shared rights owner
// (RequireLocalUsePermitted, selection and execution) is the gate, so a stored
// enable must not be silently masked, including for non-commercial weights
// (operator stance 2026-11-21: best models for normal use).
func TestEffectiveEnabledHonoursOverlayForEveryLicenseClass(t *testing.T) {
	r, err := Load()
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	var conditional Model
	for _, model := range r.Models() {
		if model.CapabilityLabels.CommercialUse == CommercialUseConditional {
			conditional = model
			break
		}
	}
	if conditional.ID == "" {
		t.Fatal("seed should include a conditional-rights model for this regression")
	}
	overlay := map[string]bool{conditional.ID: true}
	if !EffectiveEnabled(conditional, overlay) {
		t.Fatalf("stored enable for conditional-rights model %q was masked", conditional.ID)
	}
	if !r.EnabledWithOverlay(overlay)(conditional.ID) {
		t.Fatalf("registry selector ignored stored enable for %q", conditional.ID)
	}
	nonCommercial := conditional
	nonCommercial.CapabilityLabels.CommercialUse = CommercialUseNo
	if !EffectiveEnabled(nonCommercial, overlay) {
		t.Fatalf("stored enable for a non-commercial model was masked")
	}
	if EffectiveEnabled(nonCommercial, map[string]bool{conditional.ID: false}) {
		t.Fatalf("stored disable for a non-commercial model was ignored")
	}
}
