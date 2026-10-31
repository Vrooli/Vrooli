package profiles

import (
	"os"
	"path/filepath"
	"testing"
)

func writeScenario(t *testing.T, dir, serviceJSON string, withElectron bool) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".vrooli", "service.json"), []byte(serviceJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	if withElectron {
		if err := os.MkdirAll(filepath.Join(dir, "platforms", "electron"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

func TestProfileTablesMatchConceptDocument(t *testing.T) {
	if len(WebPublicV1.Targets) != 11 {
		t.Fatalf("web-public-v1 has %d targets, want 11", len(WebPublicV1.Targets))
	}
	if WebPublicV1.Root != "ui/public/public" {
		t.Fatalf("web-public-v1 root = %q", WebPublicV1.Root)
	}
	og, ok := targetByID(WebPublicV1, "og-image")
	if !ok || og.Width != 1200 || og.Height != 630 || og.Variant != VariantSocialCard {
		t.Fatalf("og-image target = %+v", og)
	}
	mask, ok := targetByID(WebPublicV1, "maskable-512")
	if !ok || mask.Variant != VariantMaskable || mask.Wiring != WiringManifestMaskable {
		t.Fatalf("maskable-512 target = %+v", mask)
	}
	if len(ElectronV1.Targets) != 10 {
		t.Fatalf("electron-v1 has %d targets, want 10", len(ElectronV1.Targets))
	}
	ico, ok := targetByID(ElectronV1, "ico")
	if !ok || ico.Format != "ico" || len(ico.Sizes) != 7 {
		t.Fatalf("ico target = %+v", ico)
	}
}

func TestResolveReturnsBothProfilesWhenElectronExists(t *testing.T) {
	dir := t.TempDir()
	writeScenario(t, dir, `{"branding":{"brand":"aquila","targets":["web-public-v1","electron-v1"]}}`, true)
	got, err := Resolve(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != "web-public-v1" || got[1].ID != "electron-v1" {
		t.Fatalf("resolved = %+v", got)
	}
}

func TestResolveDropsElectronWithoutDir(t *testing.T) {
	dir := t.TempDir()
	writeScenario(t, dir, `{"branding":{"brand":"aquila","targets":["web-public-v1","electron-v1"]}}`, false)
	got, err := Resolve(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "web-public-v1" {
		t.Fatalf("resolved = %+v", got)
	}
}

func TestResolveErrorsWithoutBranding(t *testing.T) {
	dir := t.TempDir()
	writeScenario(t, dir, `{"service":{"name":"x"}}`, false)
	if _, err := Resolve(dir); err == nil {
		t.Fatal("expected ErrNoBranding")
	}
}

func targetByID(p Profile, id string) (Target, bool) {
	for _, t := range p.Targets {
		if t.ID == id {
			return t, true
		}
	}
	return Target{}, false
}
