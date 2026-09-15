package validation

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"brand-manager/internal/profiles"
)

func writePNG(t *testing.T, path string, w, h int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.NRGBA{10, 20, 30, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func seedCompliantWebPublic(t *testing.T, root string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	serviceJSON := `{"branding":{"brand":"aquila","targets":["web-public-v1"]}}`
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "service.json"), []byte(serviceJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(root, profiles.WebPublicV1.Root)
	for _, target := range profiles.WebPublicV1.Targets {
		if target.ID == "manifest" {
			continue
		}
		rel := filepath.Join(base, target.Path)
		switch target.Format {
		case "png":
			writePNG(t, rel, target.Width, target.Height)
		case "svg":
			if err := os.MkdirAll(filepath.Dir(rel), 0o755); err != nil {
				t.Fatal(err)
			}
			_ = os.WriteFile(rel, []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512"></svg>`), 0o644)
		}
	}
	if err := os.WriteFile(filepath.Join(root, indexHTMLRel), []byte(`<html><head><!-- brand-manager:icons:start --><!-- brand-manager:icons:end --></head></html>`), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := `{"icons":[{"src":"icon-192.png","sizes":"192x192","type":"image/png","purpose":"any"}]}`
	if err := os.WriteFile(filepath.Join(base, "site.webmanifest"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
}

func hasRule(result *ScanResult, id string) bool {
	for _, f := range result.Findings {
		if f.RuleID == id {
			return true
		}
	}
	return false
}

// targetProblems returns the declared-icon-targets finding's problem strings.
func targetProblems(result *ScanResult) []string {
	for _, f := range result.Findings {
		if f.RuleID != "declared-icon-targets" {
			continue
		}
		raw, ok := f.Evidence["problems"]
		if !ok {
			return nil
		}
		switch v := raw.(type) {
		case []string:
			return v
		case []any:
			out := make([]string, 0, len(v))
			for _, item := range v {
				out = append(out, fmt.Sprint(item))
			}
			return out
		}
	}
	return nil
}

func problemsContain(result *ScanResult, substr string) bool {
	for _, p := range targetProblems(result) {
		if strings.Contains(p, substr) {
			return true
		}
	}
	return false
}

// writeOpaqueMaskableViolation writes a 512x512 opaque PNG with a distinct
// colour block in the top-left corner, deliberately outside the 0.40 safe zone.
func writeOpaqueMaskableViolation(t *testing.T, path string, edge int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	img := image.NewNRGBA(image.Rect(0, 0, edge, edge))
	bg := color.NRGBA{10, 20, 30, 255}
	mark := color.NRGBA{240, 240, 40, 255}
	for y := 0; y < edge; y++ {
		for x := 0; x < edge; x++ {
			if x < edge/8 && y < edge/8 {
				img.SetNRGBA(x, y, mark)
				continue
			}
			img.SetNRGBA(x, y, bg)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDeclaredIconTargetsPassesCompliantSet(t *testing.T) {
	root := t.TempDir()
	seedCompliantWebPublic(t, root)
	res := ScanScenario("web-console", root)
	if hasRule(res, "declared-icon-targets") {
		t.Fatalf("expected no declared-icon-targets finding, got %+v", res.Findings)
	}
}

func TestDeclaredIconTargetsFlagsWrongDimension(t *testing.T) {
	root := t.TempDir()
	seedCompliantWebPublic(t, root)
	// Deliberately wrong size for favicon-16.
	writePNG(t, filepath.Join(root, profiles.WebPublicV1.Root, "favicon-16.png"), 256, 256)
	res := ScanScenario("web-console", root)
	if !hasRule(res, "declared-icon-targets") {
		t.Fatalf("expected a declared-icon-targets finding for a mislabeled icon")
	}
}

func TestDeclaredIconTargetsFlagsMissingMarkerBlock(t *testing.T) {
	root := t.TempDir()
	seedCompliantWebPublic(t, root)
	if err := os.WriteFile(filepath.Join(root, indexHTMLRel), []byte(`<html><head></head></html>`), 0o644); err != nil {
		t.Fatal(err)
	}
	res := ScanScenario("web-console", root)
	if !hasRule(res, "declared-icon-targets") {
		t.Fatalf("expected a declared-icon-targets finding for a missing marker block")
	}
}

func TestDeclaredIconTargetsFlagsPlaceholderBytes(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "platforms", "electron"), 0o755); err != nil {
		t.Fatal(err)
	}
	serviceJSON := `{"branding":{"brand":"aquila","targets":["electron-v1"]}}`
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "service.json"), []byte(serviceJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	placeholder, err := os.ReadFile(filepath.Join("testdata", "desktop-placeholder-663.png"))
	if err != nil {
		t.Fatalf("read placeholder fixture: %v", err)
	}
	// The fixture is 256x256, so it matches the icon-256 dimensions exactly and
	// only the placeholder check can fire.
	assets := filepath.Join(root, "platforms", "electron", "assets")
	for _, target := range profiles.ElectronV1.Targets {
		if target.Format != "png" {
			continue
		}
		if target.ID == "icon-256" {
			if err := os.MkdirAll(assets, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(assets, target.Path), placeholder, 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		writePNG(t, filepath.Join(assets, target.Path), target.Width, target.Height)
	}
	res := ScanScenario("web-console", root)
	if !problemsContain(res, "placeholder bytes") {
		t.Fatalf("expected a placeholder-bytes problem, got %v", targetProblems(res))
	}
}

func TestDeclaredIconTargetsFlagsMaskableSafeZoneViolation(t *testing.T) {
	root := t.TempDir()
	seedCompliantWebPublic(t, root)
	writeOpaqueMaskableViolation(t, filepath.Join(root, profiles.WebPublicV1.Root, "maskable-icon-512.png"), 512)
	res := ScanScenario("web-console", root)
	if !problemsContain(res, "safe zone") {
		t.Fatalf("expected a safe-zone problem, got %v", targetProblems(res))
	}
}

func TestDeclaredIconTargetsFlagsAbsoluteManifestSrc(t *testing.T) {
	root := t.TempDir()
	seedCompliantWebPublic(t, root)
	manifest := `{"icons":[{"src":"/icon-192.png","sizes":"192x192","type":"image/png","purpose":"any"}]}`
	if err := os.WriteFile(filepath.Join(root, profiles.WebPublicV1.Root, "site.webmanifest"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	res := ScanScenario("web-console", root)
	if !problemsContain(res, "must be relative") {
		t.Fatalf("expected a relative-src problem, got %v", targetProblems(res))
	}
}
