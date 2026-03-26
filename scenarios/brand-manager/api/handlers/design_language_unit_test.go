package handlers

import (
	"strings"
	"testing"

	"brand-manager/domain"
)

// Unit tests for pure functions in design_language.go.
// [REQ:BM-REQ-DESIGN-GEN] [REQ:BM-REQ-DESIGN-CONTENT]

func TestRenderDesignLanguage_FullBrand(t *testing.T) {
	brand := &domain.Brand{
		ID:          "b1",
		Name:        "Full Brand",
		Description: "Comprehensive test brand",
		Version:     3,
		Identity: &domain.Identity{
			DisplayName: "Full Display",
			Tagline:     "Build great things",
			LogoPath:    "/assets/logo.svg",
			FaviconPath: "/assets/favicon.ico",
			IconPath:    "/assets/icon.png",
		},
		Colors: &domain.Colors{
			Primary:    "#3498db",
			Secondary:  "#2ecc71",
			Accent:     "#e74c3c",
			Background: "#ffffff",
			Surface:    "#f0f0f0",
			Text:       "#333333",
			Error:      "#cc0000",
		},
		Typography: &domain.Typography{
			HeadingFont:  "Inter",
			BodyFont:     "Open Sans",
			MonoFont:     "Fira Code",
			BaseFontSize: "16px",
		},
		Voice: &domain.Voice{
			Tone:     "professional",
			Style:    "concise",
			Keywords: []string{"reliable", "modern", "fast"},
		},
		Notes: "Extra design guidance here.",
	}

	md := renderDesignLanguage(brand)

	// Structure checks
	for _, section := range []string{
		"# Full Brand — Design Language",
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

	// Description blockquote
	if !strings.Contains(md, "> Comprehensive test brand") {
		t.Error("missing description blockquote")
	}

	// Identity fields
	for _, want := range []string{"Full Display", "Build great things", "`/assets/logo.svg`", "`/assets/favicon.ico`", "`/assets/icon.png`"} {
		if !strings.Contains(md, want) {
			t.Errorf("missing identity field: %s", want)
		}
	}

	// Color table
	for _, want := range []string{"#3498db", "#2ecc71", "#e74c3c", "#ffffff", "#f0f0f0", "#333333", "#cc0000"} {
		if !strings.Contains(md, want) {
			t.Errorf("missing color: %s", want)
		}
	}
	if !strings.Contains(md, "| Primary |") {
		t.Error("missing color table header row")
	}

	// Typography
	for _, want := range []string{"Inter", "Open Sans", "Fira Code", "16px"} {
		if !strings.Contains(md, want) {
			t.Errorf("missing typography field: %s", want)
		}
	}

	// Voice
	if !strings.Contains(md, "reliable, modern, fast") {
		t.Error("missing keywords")
	}

	// Visual patterns derived from brand data
	if !strings.Contains(md, "Primary accent") {
		t.Error("missing visual pattern for primary accent")
	}
	if !strings.Contains(md, "Surface/background layering") {
		t.Error("missing visual pattern for surface/background")
	}
	if !strings.Contains(md, "Heading typeface") {
		t.Error("missing visual pattern for heading typeface")
	}

	// Notes
	if !strings.Contains(md, "Extra design guidance here.") {
		t.Error("missing notes content")
	}

	// Footer
	if !strings.Contains(md, "brand `b1` v3") {
		t.Error("missing footer with brand ID and version")
	}
}

func TestRenderDesignLanguage_MinimalBrand(t *testing.T) {
	brand := &domain.Brand{
		ID:      "b2",
		Name:    "Bare Brand",
		Version: 1,
	}

	md := renderDesignLanguage(brand)

	// Should have "Not yet defined" for all nil sections
	count := strings.Count(md, "_Not yet defined._")
	if count < 3 {
		t.Errorf("expected at least 3 'Not yet defined' placeholders, got %d", count)
	}

	// No description blockquote
	if strings.Contains(md, ">") {
		t.Error("should not have blockquote when description is empty")
	}

	// No visual patterns derived
	if strings.Contains(md, "Primary accent") {
		t.Error("should not have primary accent pattern when no colors")
	}

	// No notes section
	if strings.Contains(md, "## Notes") {
		t.Error("should not have notes section when notes are empty")
	}
}

func TestRenderDesignLanguage_PartialColors(t *testing.T) {
	brand := &domain.Brand{
		ID:      "b3",
		Name:    "Partial Color",
		Version: 1,
		Colors: &domain.Colors{
			Primary: "#ff0000",
			Text:    "#000000",
		},
	}

	md := renderDesignLanguage(brand)

	// Present colors
	if !strings.Contains(md, "#ff0000") {
		t.Error("missing primary color")
	}
	if !strings.Contains(md, "#000000") {
		t.Error("missing text color")
	}

	// Absent colors should not appear in table
	if strings.Contains(md, "Secondary") {
		t.Error("secondary should not appear when empty")
	}

	// Primary accent pattern should appear
	if !strings.Contains(md, "Primary accent") {
		t.Error("missing primary accent visual pattern")
	}
	// Surface/background should not (no surface/bg defined)
	if strings.Contains(md, "Surface/background layering") {
		t.Error("should not have surface/background pattern when those colors are empty")
	}
}

func TestRenderDesignLanguage_VoiceNoKeywords(t *testing.T) {
	brand := &domain.Brand{
		ID:      "b4",
		Name:    "Voice Test",
		Version: 1,
		Voice: &domain.Voice{
			Tone:  "casual",
			Style: "verbose",
		},
	}

	md := renderDesignLanguage(brand)

	if !strings.Contains(md, "casual") {
		t.Error("missing tone")
	}
	if !strings.Contains(md, "verbose") {
		t.Error("missing style")
	}
	// Keywords line should not appear (empty)
	if strings.Contains(md, "**Keywords**:") {
		t.Error("should not show Keywords when empty")
	}
}

func TestWriteFields_AllEmpty(t *testing.T) {
	var sb strings.Builder
	wrote := writeFields(&sb, []mdField{
		{"A", ""},
		{"B", ""},
	})
	if wrote {
		t.Error("expected false when all fields empty")
	}
	if sb.Len() != 0 {
		t.Errorf("expected empty output, got %q", sb.String())
	}
}

func TestWriteFields_Mixed(t *testing.T) {
	var sb strings.Builder
	wrote := writeFields(&sb, []mdField{
		{"Filled", "value"},
		{"Empty", ""},
		{"Also Filled", "v2"},
	})
	if !wrote {
		t.Error("expected true when at least one field has value")
	}
	out := sb.String()
	if !strings.Contains(out, "**Filled**: value") {
		t.Error("missing Filled field")
	}
	if strings.Contains(out, "**Empty**") {
		t.Error("empty field should not appear")
	}
	if !strings.Contains(out, "**Also Filled**: v2") {
		t.Error("missing Also Filled field")
	}
}

func TestWriteSection_EmptyFields(t *testing.T) {
	var sb strings.Builder
	writeSection(&sb, "Test Section", nil)
	out := sb.String()

	if !strings.Contains(out, "## Test Section") {
		t.Error("missing section header")
	}
	if !strings.Contains(out, "_Not yet defined._") {
		t.Error("missing placeholder for empty section")
	}
}

func TestBacktick_EmptyReturnsEmpty(t *testing.T) {
	if got := backtick(""); got != "" {
		t.Errorf("backtick(\"\") = %q, want empty", got)
	}
}

func TestBacktick_WrapsValue(t *testing.T) {
	if got := backtick("path/to/file"); got != "`path/to/file`" {
		t.Errorf("backtick = %q, want `path/to/file`", got)
	}
}
