package design

import (
	"fmt"
	"strings"
)

// renderDesignLanguage generates a canonical DESIGN.md export from a brand. The
// output is deterministic and pure (no clock, no I/O): front matter, an identity
// section, a color-system table, typography, voice, visual patterns derived from
// the color/typography choices, and any freeform notes. Empty facets render as
// "_Not yet defined._" placeholders rather than being omitted, so the document
// doubles as a fill-in template for a partially-authored brand.
func renderDesignLanguage(brand Brand) string {
	var sb strings.Builder

	sb.WriteString("---\n")
	fmt.Fprintf(&sb, "id: %s\n", brand.ID)
	fmt.Fprintf(&sb, "name: %q\n", brand.Name)
	fmt.Fprintf(&sb, "version: %d\n", brand.Version)
	sb.WriteString("source: brand-manager\n")
	sb.WriteString("---\n\n")
	fmt.Fprintf(&sb, "# %s DESIGN.md\n\n", brand.Name)
	if brand.Description != "" {
		fmt.Fprintf(&sb, "> %s\n\n", brand.Description)
	}

	// Identity
	if brand.Identity.HasAny() {
		writeSection(&sb, "Identity", []mdField{
			{"Display Name", brand.Identity.DisplayName},
			{"Tagline", brand.Identity.Tagline},
			{"Logo", backtick(brand.Identity.LogoPath)},
			{"Favicon", backtick(brand.Identity.FaviconPath)},
			{"Icon", backtick(brand.Identity.IconPath)},
		})
	} else {
		writeSection(&sb, "Identity", nil)
	}

	// Color System
	sb.WriteString("## Color System\n\n")
	if brand.Colors.HasAny() {
		sb.WriteString("| Role | Value |\n|------|-------|\n")
		for _, e := range []mdField{
			{"Primary", brand.Colors.Primary},
			{"Secondary", brand.Colors.Secondary},
			{"Accent", brand.Colors.Accent},
			{"Background", brand.Colors.Background},
			{"Surface", brand.Colors.Surface},
			{"Text", brand.Colors.Text},
			{"Error", brand.Colors.Error},
		} {
			if e.value != "" {
				fmt.Fprintf(&sb, "| %s | `%s` |\n", e.label, e.value)
			}
		}
	} else {
		sb.WriteString("_Not yet defined._\n")
	}
	sb.WriteString("\n")

	// Typography
	if brand.Typography.HasAny() {
		writeSection(&sb, "Typography", []mdField{
			{"Heading Font", brand.Typography.HeadingFont},
			{"Body Font", brand.Typography.BodyFont},
			{"Mono Font", brand.Typography.MonoFont},
			{"Base Font Size", brand.Typography.BaseFontSize},
		})
	} else {
		writeSection(&sb, "Typography", nil)
	}

	// Voice & Personality
	if brand.Voice.HasAny() {
		kw := ""
		if len(brand.Voice.Keywords) > 0 {
			kw = strings.Join(brand.Voice.Keywords, ", ")
		}
		writeSection(&sb, "Voice & Personality", []mdField{
			{"Tone", brand.Voice.Tone},
			{"Style", brand.Voice.Style},
			{"Keywords", kw},
		})
	} else {
		writeSection(&sb, "Voice & Personality", nil)
	}

	// Visual Patterns & Metaphors — derived from the brand's own choices.
	sb.WriteString("## Visual Patterns & Metaphors\n\n")
	sb.WriteString("Derive visual patterns from the brand's color system and typography:\n\n")
	if brand.Colors.Primary != "" {
		fmt.Fprintf(&sb, "- **Primary accent** (`%s`) anchors interactive elements, CTAs, and focus states\n", brand.Colors.Primary)
	}
	if brand.Colors.Surface != "" && brand.Colors.Background != "" {
		fmt.Fprintf(&sb, "- **Surface/background layering** (`%s` on `%s`) creates depth without shadow overuse\n", brand.Colors.Surface, brand.Colors.Background)
	}
	if brand.Typography.HeadingFont != "" {
		fmt.Fprintf(&sb, "- **Heading typeface** (%s) conveys hierarchy; reserve for titles and key labels\n", brand.Typography.HeadingFont)
	}
	sb.WriteString("\n")

	// Notes
	if brand.Notes != "" {
		sb.WriteString("## Notes\n\n")
		sb.WriteString(brand.Notes)
		sb.WriteString("\n\n")
	}

	sb.WriteString("---\n\n")
	fmt.Fprintf(&sb, "_Generated from Brand Manager — brand `%s` v%d_\n", brand.ID, brand.Version)

	return sb.String()
}

// mdField is a label-value pair for markdown rendering.
type mdField struct{ label, value string }

// writeFields writes non-empty fields as markdown bullet points. Returns true if
// at least one field was written.
func writeFields(sb *strings.Builder, fields []mdField) bool {
	wrote := false
	for _, f := range fields {
		if f.value != "" {
			fmt.Fprintf(sb, "- **%s**: %s\n", f.label, f.value)
			wrote = true
		}
	}
	return wrote
}

// writeSection writes a markdown section header and its fields, or the
// "_Not yet defined._" placeholder when no field has a value.
func writeSection(sb *strings.Builder, title string, fields []mdField) {
	fmt.Fprintf(sb, "## %s\n\n", title)
	if !writeFields(sb, fields) {
		sb.WriteString("_Not yet defined._\n")
	}
	sb.WriteString("\n")
}

// backtick wraps a non-empty string in backticks for markdown code formatting.
func backtick(s string) string {
	if s == "" {
		return ""
	}
	return "`" + s + "`"
}
