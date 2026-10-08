package models

import (
	"bytes"
	"context"
	"testing"

	apidb "github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"
)

func TestUnknownLicenceDefaultsRestricted(t *testing.T) {
	r := New(true, Model{ID: "unknown"})
	if _, err := r.Resolve("unknown"); err == nil {
		t.Fatal("unknown licence resolved in permissive build")
	}
}
func TestPermissiveModelResolves(t *testing.T) {
	r := New(true, Model{ID: "ace", Licence: LicencePermissive})
	if _, err := r.Resolve("ace"); err != nil {
		t.Fatal(err)
	}
}

func TestResolveForUsesFreeVRAMGate(t *testing.T) {
	r := New(true, Model{ID: "ace", DisplayName: "ACE-Step", Licence: LicencePermissive, License: "MIT", CommercialUse: "yes", BaseModelLineage: "ACE-Step", KnownRisks: "remote code", Hardware: Hardware{MinVRAMBytes: 7}})
	if _, err := r.ResolveFor("ace", Host{FreeVRAMBytes: 6}); err == nil {
		t.Fatal("model resolved below free-VRAM gate")
	}
}

func TestVerifyChecksumAndDiskFloor(t *testing.T) {
	data := []byte("weights")
	if _, err := VerifyChecksum(bytes.NewReader(data), ""); err == nil {
		t.Fatal("empty checksum accepted")
	}
	if err := CheckDisk(9, 4, 6); err == nil {
		t.Fatal("install passed below disk floor")
	}
}

func TestRegistryValidateRequiresCapabilityLabels(t *testing.T) {
	r := New(false, Model{ID: "ace", DisplayName: "ACE-Step", Licence: LicencePermissive})
	if err := r.Validate(); err == nil {
		t.Fatal("incomplete registry entry passed validation")
	}
}

func TestLoadSeedProvidesValidatedAceStepEntry(t *testing.T) {
	r, err := LoadSeed(true)
	if err != nil {
		t.Fatal(err)
	}
	model, err := r.Resolve("ACE-Step/Ace-Step1.5:acestep-v15-turbo")
	if err != nil || model.Variant != "acestep-v15-turbo" {
		t.Fatalf("model=%+v err=%v", model, err)
	}
}

func TestOverlayPersistsFourTableState(t *testing.T) {
	d := db.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatal(err)
	}
	base := New(true, Model{ID: "ace", DisplayName: "ACE", Licence: LicencePermissive, License: "MIT", CommercialUse: "yes", BaseModelLineage: "ACE", KnownRisks: "none"})
	overlay := NewOverlay(d, base)
	ctx := context.Background()
	if err := overlay.SetEnabled(ctx, "ace", true); err != nil {
		t.Fatal(err)
	}
	if err := overlay.SetInstalled(ctx, "ace", "checksum", 42); err != nil {
		t.Fatal(err)
	}
	if err := overlay.SetCustom(ctx, Model{ID: "custom", DisplayName: "Custom", Licence: LicencePermissive, License: "MIT", CommercialUse: "yes", BaseModelLineage: "ACE", KnownRisks: "none"}); err != nil {
		t.Fatal(err)
	}
	if err := overlay.SetOperationDefault(ctx, "compose", "ace"); err != nil {
		t.Fatal(err)
	}
	got, err := overlay.ResolveOperation(ctx, "compose")
	if err != nil || got.ID != "ace" {
		t.Fatalf("resolved=%+v err=%v", got, err)
	}
	for _, table := range []string{"model_state", "model_install", "model_custom", "model_operation_default"} {
		var count int
		if err := d.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("%s count=%d, want 1", table, count)
		}
	}
}
