package design

import (
	"strings"
	"testing"
)

// Unit tests for the pure render functions in render.go. They assert the
// document structure and the derive-from-data behaviour without any transport
// or cross-domain wiring.

func TestRenderDesignLanguage_FullBrand(t *testing.T) {
	brand := Brand{
		ID:          "b1",
		Name:        "Full Brand",
		Description: "Comprehensive test brand",
		Version:     3,
		Identity: Identity{
			DisplayName: "Full Display",
			Tagline:     "Build great things",
			LogoPath:    "/assets/logo.svg",
			FaviconPath: "/assets/favicon.ico",
			IconPath:    "/assets/icon.png",
		},
		Colors: Colors{
			Primary:    "#3498db",
			Secondary:  "#2ecc71",
			Accent:     "#e74c3c",
			Background: "#ffffff",
			Surface:    "#f0f0f0",
			Text:       "#333333",
			Error:      "#cc0000",
		},
		Typography: Typography{
			HeadingFont:  "Inter",
			BodyFont:     "Open Sans",
			MonoFont:     "Fira Code",
			BaseFontSize: "16px",
		},
		Voice: Voice{
			Tone:     "professional",
			Style:    "concise",
			Keywords: []string{"reliable", "modern", "fast"},
		},
		Notes: "Extra design guidance here.",
	}

	md := renderDesignLanguage(brand)

	for _, section := range []string{
		"# Full Brand DESIGN.md",
		"## Identity",
		"## Color System",
		"## Typography",
		"## Voice & Personality",
		"## Visual Patterns & Metaphors",
		"## Notes",
	} {
		if !strings.Contains(md, section) {
			t.Errorf("missing section: %q", section)
		}
	}

	if !strings.Contains(md, "> Comprehensive test brand") {
		t.Error("missing description blockquote")
	}
	for _, want := range []string{"Full Display", "Build great things", "`/assets/logo.svg`", "`/assets/favicon.ico`", "`/assets/icon.png`"} {
		if !strings.Contains(md, want) {
			t.Errorf("missing identity field: %s", want)
		}
	}
	for _, want := range []string{"#3498db", "#2ecc71", "#e74c3c", "#ffffff", "#f0f0f0", "#333333", "#cc0000"} {
		if !strings.Contains(md, want) {
			t.Errorf("missing color: %s", want)
		}
	}
	if !strings.Contains(md, "| Primary |") {
		t.Error("missing color table header row")
	}
	for _, want := range []string{"Inter", "Open Sans", "Fira Code", "16px"} {
		if !strings.Contains(md, want) {
			t.Errorf("missing typography field: %s", want)
		}
	}
	if !strings.Contains(md, "reliable, modern, fast") {
		t.Error("missing keywords")
	}
	if !strings.Contains(md, "Primary accent") {
		t.Error("missing visual pattern for primary accent")
	}
	if !strings.Contains(md, "Surface/background layering") {
		t.Error("missing visual pattern for surface/background")
	}
	if !strings.Contains(md, "Heading typeface") {
		t.Error("missing visual pattern for heading typeface")
	}
	if !strings.Contains(md, "Extra design guidance here.") {
		t.Error("missing notes content")
	}
	if !strings.Contains(md, "brand `b1` v3") {
		t.Error("missing footer with brand id and version")
	}
}

func TestRenderDesignLanguage_MinimalBrand(t *testing.T) {
	md := renderDesignLanguage(Brand{ID: "b2", Name: "Bare Brand", Version: 1})

	if !strings.HasPrefix(md, "---\n") {
		t.Error("missing DESIGN.md front matter")
	}
	if count := strings.Count(md, "_Not yet defined._"); count < 3 {
		t.Errorf("expected at least 3 'Not yet defined' placeholders, got %d", count)
	}
	if strings.Contains(md, "\n> ") {
		t.Error("should not have blockquote when description is empty")
	}
	if strings.Contains(md, "Primary accent") {
		t.Error("should not derive primary accent pattern when no colors")
	}
	if strings.Contains(md, "## Notes") {
		t.Error("should not have notes section when notes are empty")
	}
}

func TestRenderDesignLanguage_PartialColors(t *testing.T) {
	md := renderDesignLanguage(Brand{
		ID: "b3", Name: "Partial Color", Version: 1,
		Colors: Colors{Primary: "#ff0000", Text: "#000000"},
	})

	if !strings.Contains(md, "#ff0000") {
		t.Error("missing primary color")
	}
	if !strings.Contains(md, "#000000") {
		t.Error("missing text color")
	}
	if strings.Contains(md, "| Secondary |") {
		t.Error("secondary should not appear in the table when empty")
	}
	if !strings.Contains(md, "Primary accent") {
		t.Error("missing primary accent visual pattern")
	}
	if strings.Contains(md, "Surface/background layering") {
		t.Error("should not derive surface/background pattern when those colors are empty")
	}
}

func TestRenderDesignLanguage_VoiceNoKeywords(t *testing.T) {
	md := renderDesignLanguage(Brand{
		ID: "b4", Name: "Voice Test", Version: 1,
		Voice: Voice{Tone: "casual", Style: "verbose"},
	})

	if !strings.Contains(md, "casual") {
		t.Error("missing tone")
	}
	if !strings.Contains(md, "verbose") {
		t.Error("missing style")
	}
	if strings.Contains(md, "**Keywords**:") {
		t.Error("should not show Keywords when empty")
	}
}

func TestWriteFields(t *testing.T) {
	var empty strings.Builder
	if writeFields(&empty, []mdField{{"A", ""}, {"B", ""}}) {
		t.Error("expected false when all fields empty")
	}
	if empty.Len() != 0 {
		t.Errorf("expected empty output, got %q", empty.String())
	}

	var mixed strings.Builder
	if !writeFields(&mixed, []mdField{{"Filled", "value"}, {"Empty", ""}}) {
		t.Error("expected true when at least one field has value")
	}
	out := mixed.String()
	if !strings.Contains(out, "**Filled**: value") {
		t.Error("missing Filled field")
	}
	if strings.Contains(out, "**Empty**") {
		t.Error("empty field should not appear")
	}
}

func TestBacktick(t *testing.T) {
	if got := backtick(""); got != "" {
		t.Errorf("backtick(\"\") = %q, want empty", got)
	}
	if got := backtick("path/to/file"); got != "`path/to/file`" {
		t.Errorf("backtick = %q, want `path/to/file`", got)
	}
}
