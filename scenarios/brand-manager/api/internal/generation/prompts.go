package generation

import (
	"fmt"
	"strings"
)

// colorPrompt builds a prompt for generating a brand color palette. Ported from
// the old aigen.ColorPrompt.
func colorPrompt(brandName, description, notes string) string {
	var sb strings.Builder
	sb.WriteString("Generate a professional brand color palette for a software product.\n\n")
	fmt.Fprintf(&sb, "Brand name: %s\n", brandName)
	if description != "" {
		fmt.Fprintf(&sb, "Description: %s\n", description)
	}
	if notes != "" {
		fmt.Fprintf(&sb, "Notes: %s\n", notes)
	}
	sb.WriteString("\nReturn ONLY a JSON object with these hex color values:\n")
	sb.WriteString(`{"primary":"#...","secondary":"#...","accent":"#...","background":"#...","surface":"#...","text":"#...","error":"#..."}`)
	sb.WriteString("\n\nRequirements:\n")
	sb.WriteString("- All colors must be valid 6-digit hex codes (e.g. #1a2b3c)\n")
	sb.WriteString("- Primary on background must have WCAG AA contrast ratio >= 4.5:1\n")
	sb.WriteString("- Text on background must have WCAG AA contrast ratio >= 4.5:1\n")
	sb.WriteString("- Background should be light (#f... range), text should be dark\n")
	sb.WriteString("- Return ONLY the JSON object, no explanation\n")
	return sb.String()
}

// typographyPrompt builds a prompt for generating typography choices. Ported
// from the old aigen.TypographyPrompt.
func typographyPrompt(brandName, description, notes string) string {
	var sb strings.Builder
	sb.WriteString("Suggest professional typography for a software product brand.\n\n")
	fmt.Fprintf(&sb, "Brand name: %s\n", brandName)
	if description != "" {
		fmt.Fprintf(&sb, "Description: %s\n", description)
	}
	if notes != "" {
		fmt.Fprintf(&sb, "Notes: %s\n", notes)
	}
	sb.WriteString("\nReturn ONLY a JSON object:\n")
	sb.WriteString(`{"heading_font":"...","body_font":"...","mono_font":"...","base_font_size":"..."}`)
	sb.WriteString("\n\nRequirements:\n")
	sb.WriteString("- Use widely available Google Fonts or system fonts\n")
	sb.WriteString("- Heading font should be distinctive but readable\n")
	sb.WriteString("- Body font should be highly readable at small sizes\n")
	sb.WriteString("- Mono font for code blocks\n")
	sb.WriteString("- base_font_size as CSS value (e.g. '16px')\n")
	sb.WriteString("- Return ONLY the JSON object, no explanation\n")
	return sb.String()
}

// voicePrompt builds a prompt for generating brand voice/tone. Ported from the
// old aigen.VoicePrompt.
func voicePrompt(brandName, description, notes string) string {
	var sb strings.Builder
	sb.WriteString("Define the brand voice and communication style for a software product.\n\n")
	fmt.Fprintf(&sb, "Brand name: %s\n", brandName)
	if description != "" {
		fmt.Fprintf(&sb, "Description: %s\n", description)
	}
	if notes != "" {
		fmt.Fprintf(&sb, "Notes: %s\n", notes)
	}
	sb.WriteString("\nReturn ONLY a JSON object:\n")
	sb.WriteString(`{"tone":"...","style":"...","keywords":["...","...","..."]}`)
	sb.WriteString("\n\nRequirements:\n")
	sb.WriteString("- tone: one-word descriptor (e.g. 'professional', 'friendly', 'bold')\n")
	sb.WriteString("- style: one sentence describing communication style\n")
	sb.WriteString("- keywords: 3-5 words that capture the brand essence\n")
	sb.WriteString("- Return ONLY the JSON object, no explanation\n")
	return sb.String()
}

// logoPrompt builds a prompt for generating a brand logo image. Ported from the
// old aigen.LogoPrompt.
func logoPrompt(brandName, description, primaryColor string) string {
	prompt := fmt.Sprintf(
		"A minimal, professional logo for '%s'. Clean vector style, simple geometric shapes, modern design. "+
			"White or transparent background. No text in the image.",
		brandName,
	)
	if description != "" {
		prompt += " " + description + "."
	}
	if primaryColor != "" {
		prompt += fmt.Sprintf(" Primary color: %s.", primaryColor)
	}
	return prompt
}

// faviconPrompt builds a prompt for generating a favicon image. Ported from the
// old aigen.FaviconPrompt.
func faviconPrompt(brandName, primaryColor string) string {
	prompt := fmt.Sprintf(
		"A simple, iconic favicon for '%s'. Single symbol or letter, bold and recognizable at 32x32 pixels. "+
			"Clean edges, minimal detail.",
		brandName,
	)
	if primaryColor != "" {
		prompt += fmt.Sprintf(" Color: %s.", primaryColor)
	}
	return prompt
}
