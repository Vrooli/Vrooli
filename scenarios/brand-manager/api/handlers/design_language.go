// Package handlers - design language file generation from brand data.
// [REQ:BM-REQ-DESIGN-GEN] [REQ:BM-REQ-DESIGN-CONTENT]
package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"brand-manager/domain"

	"github.com/gorilla/mux"
)

// designLanguageResponse is the API response for design language generation.
type designLanguageResponse struct {
	BrandID  string `json:"brand_id"`
	Markdown string `json:"markdown"`
}

// GenerateDesignLanguage handles POST /api/v1/brands/{id}/design-language.
// [REQ:BM-REQ-DESIGN-GEN] [REQ:BM-REQ-DESIGN-CONTENT]
func (h *Handlers) GenerateDesignLanguage(w http.ResponseWriter, r *http.Request) {
	brandID := mux.Vars(r)["id"]

	brand, done := getOrNotFound(w, func() (*domain.Brand, error) {
		return h.brands.GetByID(r.Context(), brandID)
	}, "brand")
	if done {
		return
	}

	md := renderDesignLanguage(brand)

	writeJSON(w, http.StatusOK, designLanguageResponse{
		BrandID:  brandID,
		Markdown: md,
	})
}

// mdField is a label-value pair for markdown rendering.
type mdField struct{ label, value string }

// writeFields writes non-empty fields as markdown bullet points.
// Returns true if at least one field was written.
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

// writeSection writes a markdown section header and fields, or "_Not yet defined._" if empty.
func writeSection(sb *strings.Builder, title string, fields []mdField) {
	fmt.Fprintf(sb, "## %s\n\n", title)
	if !writeFields(sb, fields) {
		sb.WriteString("_Not yet defined._\n")
	}
	sb.WriteString("\n")
}

// renderDesignLanguage generates a DESIGN_LANGUAGE.md from brand data.
// [REQ:BM-REQ-DESIGN-CONTENT]
func renderDesignLanguage(brand *domain.Brand) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "# %s — Design Language\n\n", brand.Name)
	if brand.Description != "" {
		fmt.Fprintf(&sb, "> %s\n\n", brand.Description)
	}
	sb.WriteString("---\n\n")

	// Identity
	if brand.Identity != nil {
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
	if brand.Colors != nil {
		sb.WriteString("| Role | Value |\n|------|-------|\n")
		for _, e := range []mdField{
			{"Primary", brand.Colors.Primary}, {"Secondary", brand.Colors.Secondary},
			{"Accent", brand.Colors.Accent}, {"Background", brand.Colors.Background},
			{"Surface", brand.Colors.Surface}, {"Text", brand.Colors.Text},
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
	if brand.Typography != nil {
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
	if brand.Voice != nil {
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

	// Visual Patterns & Metaphors
	sb.WriteString("## Visual Patterns & Metaphors\n\n")
	sb.WriteString("Derive visual patterns from the brand's color system and typography:\n\n")
	if brand.Colors != nil && brand.Colors.Primary != "" {
		fmt.Fprintf(&sb, "- **Primary accent** (`%s`) anchors interactive elements, CTAs, and focus states\n", brand.Colors.Primary)
	}
	if brand.Colors != nil && brand.Colors.Surface != "" && brand.Colors.Background != "" {
		fmt.Fprintf(&sb, "- **Surface/background layering** (`%s` on `%s`) creates depth without shadow overuse\n", brand.Colors.Surface, brand.Colors.Background)
	}
	if brand.Typography != nil && brand.Typography.HeadingFont != "" {
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

// backtick wraps a non-empty string in backticks for markdown code formatting.
func backtick(s string) string {
	if s == "" {
		return ""
	}
	return "`" + s + "`"
}
