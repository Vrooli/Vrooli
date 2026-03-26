package handlers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"brand-manager/domain"
)

// Unit tests for pure functions in apply.go that don't require HTTP infrastructure.
// [REQ:BM-REQ-APPLY-CSS]

func TestGenerateColorCSS_AllColors(t *testing.T) {
	colors := &domain.Colors{
		Primary:    "#ff0000",
		Secondary:  "#00ff00",
		Accent:     "#0000ff",
		Background: "#ffffff",
		Surface:    "#f5f5f5",
		Text:       "#333333",
		Error:      "#cc0000",
	}
	css := generateColorCSS(colors)

	if !strings.Contains(css, "brand-manager:colors") {
		t.Error("missing header comment")
	}
	for _, want := range []string{
		"--brand-primary: #ff0000",
		"--brand-secondary: #00ff00",
		"--brand-accent: #0000ff",
		"--brand-background: #ffffff",
		"--brand-surface: #f5f5f5",
		"--brand-text: #333333",
		"--brand-error: #cc0000",
	} {
		if !strings.Contains(css, want) {
			t.Errorf("missing CSS variable: %s", want)
		}
	}
	for _, marker := range []string{"brand-manager:primary", "brand-manager:secondary", "brand-manager:accent"} {
		if !strings.Contains(css, marker) {
			t.Errorf("missing marker comment: %s", marker)
		}
	}
}

func TestGenerateColorCSS_PartialColors(t *testing.T) {
	colors := &domain.Colors{
		Primary: "#ff0000",
		Text:    "#333333",
	}
	css := generateColorCSS(colors)

	if !strings.Contains(css, "--brand-primary: #ff0000") {
		t.Error("missing primary")
	}
	if !strings.Contains(css, "--brand-text: #333333") {
		t.Error("missing text")
	}
	if strings.Contains(css, "--brand-secondary") {
		t.Error("empty secondary should be omitted")
	}
	if strings.Contains(css, "--brand-background") {
		t.Error("empty background should be omitted")
	}
}

func TestGenerateColorCSS_EmptyColors(t *testing.T) {
	css := generateColorCSS(&domain.Colors{})

	if !strings.Contains(css, ":root {") {
		t.Error("missing :root selector")
	}
	if !strings.Contains(css, "}") {
		t.Error("missing closing brace")
	}
	if strings.Contains(css, "--brand-") {
		t.Error("no variables should be emitted for empty colors")
	}
}

func TestGenerateTypographyCSS_AllFields(t *testing.T) {
	typo := &domain.Typography{
		HeadingFont:  "Inter",
		BodyFont:     "Open Sans",
		MonoFont:     "Fira Code",
		BaseFontSize: "16px",
	}
	css := generateTypographyCSS(typo)

	if !strings.Contains(css, "brand-manager:typography") {
		t.Error("missing header comment")
	}
	for _, want := range []string{
		"--brand-heading-font: Inter",
		"--brand-body-font: Open Sans",
		"--brand-mono-font: Fira Code",
		"--brand-base-font-size: 16px",
	} {
		if !strings.Contains(css, want) {
			t.Errorf("missing CSS variable: %s", want)
		}
	}
}

func TestGenerateTypographyCSS_PartialFields(t *testing.T) {
	typo := &domain.Typography{
		HeadingFont: "Inter",
	}
	css := generateTypographyCSS(typo)

	if !strings.Contains(css, "--brand-heading-font: Inter") {
		t.Error("missing heading font")
	}
	if strings.Contains(css, "--brand-body-font") {
		t.Error("empty body font should be omitted")
	}
}

func TestColorPairs_AllFieldsPresent(t *testing.T) {
	colors := &domain.Colors{
		Primary:    "p",
		Secondary:  "s",
		Accent:     "a",
		Background: "bg",
		Surface:    "sf",
		Text:       "t",
		Error:      "e",
	}
	pairs := colorPairs(colors)
	if len(pairs) != 7 {
		t.Fatalf("expected 7 pairs, got %d", len(pairs))
	}
	expected := []string{"primary", "secondary", "accent", "background", "surface", "text", "error"}
	for i, want := range expected {
		if pairs[i].name != want {
			t.Errorf("pair[%d].name = %q, want %q", i, pairs[i].name, want)
		}
	}
}

func TestTypographyPairs_AllFieldsPresent(t *testing.T) {
	typo := &domain.Typography{
		HeadingFont:  "h",
		BodyFont:     "b",
		MonoFont:     "m",
		BaseFontSize: "16px",
	}
	pairs := typographyPairs(typo)
	if len(pairs) != 4 {
		t.Fatalf("expected 4 pairs, got %d", len(pairs))
	}
	expected := []string{"heading-font", "body-font", "mono-font", "base-font-size"}
	for i, want := range expected {
		if pairs[i].name != want {
			t.Errorf("pair[%d].name = %q, want %q", i, pairs[i].name, want)
		}
	}
}

func TestWriteFileAtomic_CreatesDirectories(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "dir", "file.txt")
	err := writeFileAtomic(path, []byte("hello"))
	if err != nil {
		t.Fatalf("writeFileAtomic failed: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back failed: %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("content = %q, want %q", string(data), "hello")
	}
	// Verify no .tmp leftover
	if _, err := os.Stat(path + ".tmp"); err == nil {
		t.Error("temp file should not remain after successful write")
	}
}

func TestWriteFileAtomic_OverwritesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	os.WriteFile(path, []byte("old"), 0o644)

	err := writeFileAtomic(path, []byte("new"))
	if err != nil {
		t.Fatalf("writeFileAtomic failed: %v", err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "new" {
		t.Errorf("content = %q, want %q", string(data), "new")
	}
}

func TestCopyFile_Success(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "sub", "dst.txt")

	os.WriteFile(src, []byte("copy-me"), 0o644)
	err := copyFile(src, dst)
	if err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}
	data, _ := os.ReadFile(dst)
	if string(data) != "copy-me" {
		t.Errorf("content = %q, want %q", string(data), "copy-me")
	}
}

func TestCopyFile_SourceMissing(t *testing.T) {
	dir := t.TempDir()
	err := copyFile(filepath.Join(dir, "missing"), filepath.Join(dir, "dst"))
	if err == nil {
		t.Error("expected error for missing source")
	}
}
