package models

import (
	"bytes"
	"context"
	"errors"
	"strings"
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

func TestSA3DeclarationIsRestrictedUnqualifiedAndExcludedFromPermissiveLane(t *testing.T) {
	mixed, e := LoadSeed(false)
	if e != nil {
		t.Fatal(e)
	}
	m, e := mixed.Resolve("stabilityai/stable-audio-3-medium")
	if e != nil || m.Licence != LicenceRestricted || m.Installed || m.Revision != "d4d3579395a8ac0df4ab13b5b1d7138ce7290f81" || m.CommercialUse != "conditional" {
		t.Fatalf("false registry identity/qualification %+v %v", m, e)
	}
	permissive, e := LoadSeed(true)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = permissive.Resolve(m.ID); e == nil {
		t.Fatal("restricted model resolved in permissive lane")
	}
	for _, item := range permissive.List() {
		if item.Licence != LicencePermissive {
			t.Fatal("restricted model leaked into permissive list")
		}
	}
}

func TestCompositionSelectionUsesCapabilityAndPreferenceNotBackendName(t *testing.T) {
	preferred := Model{ID: "preferred", Variant: "future-default", Engine: "not-ace", CompositionPreference: 1, Licence: LicencePermissive, Operations: []string{"compose"}, SungLyrics: Capability{Supported: true, Source: "https://example.test/preferred"}}
	alternative := Model{ID: "alternative", Variant: "other", Engine: "another-engine", CompositionPreference: 2, Licence: LicencePermissive, Operations: []string{"compose"}, SungLyrics: Capability{Supported: true}}
	for _, needsLyrics := range []bool{false, true} {
		r := New(false, preferred, alternative)
		m, err := r.SelectComposition("", needsLyrics)
		if err != nil || m.ID != preferred.ID {
			t.Fatalf("capable preferred model was displaced (lyrics=%v): %+v %v", needsLyrics, m, err)
		}
	}
	preferred.SungLyrics = Capability{Reason: "No intelligible singing", Source: "https://example.test/limitation"}
	r := New(false, preferred, alternative)
	m, err := r.SelectComposition("", true)
	if err != nil || m.ID != alternative.ID {
		t.Fatalf("incapable preference not filtered: %+v %v", m, err)
	}
	if _, err := r.SelectComposition(preferred.Variant, true); err == nil || !strings.Contains(err.Error(), preferred.SungLyrics.Source) {
		t.Fatalf("explicit unsupported choice did not preserve refusal/source: %v", err)
	}
	if err := r.ValidateComposition(preferred.Engine, "", true); err == nil {
		t.Fatal("recorded unsupported engine silently rerouted")
	}
	if err := r.ValidateComposition(alternative.Engine, "", true); err != nil {
		t.Fatalf("recorded capable engine re-resolved default: %v", err)
	}
}

func TestSeedCompositionLyricsCapabilitiesAndExactVariants(t *testing.T) {
	r, err := LoadSeed(false)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		variant string
		lyrics  bool
		want    string
	}{
		{"", false, "stable-audio-3"},
		{"", true, "ace-step"},
		{"acestep-v15-turbo", true, "ace-step"},
		{"acestep-v15-sft", true, "ace-step"},
	} {
		m, err := r.SelectComposition(tc.variant, tc.lyrics)
		if err != nil || m.Engine != tc.want || (tc.variant != "" && m.Variant != tc.variant) {
			t.Errorf("selection %+v: %+v %v", tc, m, err)
		}
	}
	if _, err := r.SelectComposition("stable-audio-3-medium", true); err == nil || !strings.Contains(err.Error(), "intelligible vocals") || !strings.Contains(err.Error(), "77e0398c") {
		t.Fatalf("actual SA3 limitation lacks pinned source: %v", err)
	}
	sft, err := r.SelectComposition("acestep-v15-sft", true)
	if err != nil || sft.Revision != "c410d249e71ea9385a7b586865e65b1473e1098d" || sft.Checksum != "d4dd3a93870f06720027965b90771f529ab02094b3d29e2518f1d5e097e1af7e" || sft.Installed {
		t.Fatalf("SFT aliases Turbo closure or installation: %+v %v", sft, err)
	}
	r, err = LoadSeed(true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.SelectComposition("", false); !errors.Is(err, ErrRestrictedModel) {
		t.Fatalf("license refusal silently changed preferred engine: %v", err)
	}
	if _, err := r.SelectComposition("unknown", true); err == nil {
		t.Fatal("unknown explicit variant silently fell back")
	}
}
