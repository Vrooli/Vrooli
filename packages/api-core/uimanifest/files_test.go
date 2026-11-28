package uimanifest

import "testing"

func TestResolveFileFallsBackWhenUndeclared(t *testing.T) {
	var m Manifest
	if got := m.ResolveFile("designTokens", "ui/src/design-tokens.css"); got != "ui/src/design-tokens.css" {
		t.Fatalf("expected fallback, got %q", got)
	}
}

func TestResolveFileAndLocalePattern(t *testing.T) {
	m := Manifest{Files: map[string]FileDeclaration{
		"designTokens":    {Path: "ui/src/theme/tokens.css", ManagedRegion: &ManagedRegion{Begin: "/* a */", End: "/* b */"}},
		"localeCatalogue": {Path: "ui/src/i18n/{locale}.json", DefaultLocale: "en"},
	}}
	if got := m.ResolveFile("designTokens", "x"); got != "ui/src/theme/tokens.css" {
		t.Fatalf("declared path not honoured: %q", got)
	}
	if got := m.ResolveLocaleFile("localeCatalogue", "", "x"); got != "ui/src/i18n/en.json" {
		t.Fatalf("default locale not applied: %q", got)
	}
	if got := m.ResolveLocaleFile("localeCatalogue", "ja", "x"); got != "ui/src/i18n/ja.json" {
		t.Fatalf("locale not substituted: %q", got)
	}
	begin, end := m.ManagedRegionMarkers("designTokens", "B", "E")
	if begin != "/* a */" || end != "/* b */" {
		t.Fatalf("markers not honoured: %q %q", begin, end)
	}
	begin, end = m.ManagedRegionMarkers("localeCatalogue", "B", "E")
	if begin != "B" || end != "E" {
		t.Fatalf("marker fallback not applied: %q %q", begin, end)
	}
}

func TestMergeOverlaysFiles(t *testing.T) {
	base := Manifest{Files: map[string]FileDeclaration{"designTokens": {Path: "a.css"}, "appEntry": {Path: "main.tsx"}}}
	overlay := Manifest{Files: map[string]FileDeclaration{"designTokens": {Path: "b.css"}}}
	merged := merge(base, overlay)
	if merged.Files["designTokens"].Path != "b.css" || merged.Files["appEntry"].Path != "main.tsx" {
		t.Fatalf("overlay merge wrong: %+v", merged.Files)
	}
}

func TestValidateRejectsEscapingFilePath(t *testing.T) {
	m := Manifest{Contract: Contract{Kind: "scenario-ui", Schema: "scenario-ui-manifest/v2"}, Slots: map[string]Slot{"page": {Dir: "ui/src/pages"}}, Files: map[string]FileDeclaration{"designTokens": {Path: "../outside.css"}}}
	if err := validate(m, "manifest.json"); err == nil {
		t.Fatal("expected an error for a path escaping the scenario")
	}
}
