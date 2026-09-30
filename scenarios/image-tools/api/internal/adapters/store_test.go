package adapters

import (
	"context"
	"database/sql"
	"testing"

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

func TestStoreOverlayDefaultsToSeedThenOverrides(t *testing.T) {
	ctx := context.Background()
	st := NewStore(newStateDB(t))
	overlay, err := st.LoadOverlay(ctx)
	if err != nil {
		t.Fatalf("load overlay: %v", err)
	}
	if len(overlay) != 0 {
		t.Fatalf("fresh overlay should be empty, got %v", overlay)
	}

	if err := st.SetEnabled(ctx, "lcm-lora-sdv1-5", false); err != nil {
		t.Fatalf("set enabled: %v", err)
	}
	if err := st.SetEnabled(ctx, "lcm-lora-sdv1-5", true); err != nil { // upsert
		t.Fatalf("re-set enabled: %v", err)
	}
	overlay, err = st.LoadOverlay(ctx)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if len(overlay) != 1 || !overlay["lcm-lora-sdv1-5"] {
		t.Fatalf("expected single re-enabled override, got %v", overlay)
	}

	r, _ := Load()
	a, _ := r.ByID("lcm-lora-sdv1-5")
	if !EffectiveEnabled(a, overlay) {
		t.Fatal("EffectiveEnabled should honor the override")
	}
	if EffectiveEnabled(a, map[string]bool{"lcm-lora-sdv1-5": false}) {
		t.Fatal("EffectiveEnabled should honor a disabling override")
	}
}
