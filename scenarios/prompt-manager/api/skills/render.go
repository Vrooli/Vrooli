package skills

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	RenderFormatXML      = "xml"
	RenderFormatMarkdown = "markdown"
	RenderFormatJSON     = "json"
	RenderFormatCLI      = "cli"
)

// RenderCombined formats skills into a single combined output string.
// Returns the combined output and the normalized format.
func RenderCombined(skills []Response, format string) (string, string, error) {
	normalized := strings.ToLower(strings.TrimSpace(format))
	if normalized == "" {
		normalized = RenderFormatXML
	}

	switch normalized {
	case RenderFormatXML:
		return renderToXML(skills), normalized, nil
	case RenderFormatMarkdown:
		return renderToMarkdown(skills), normalized, nil
	case RenderFormatJSON:
		return renderToJSON(skills), normalized, nil
	case RenderFormatCLI:
		return renderToCLI(skills), normalized, nil
	default:
		return "", "", fmt.Errorf("format must be 'xml', 'markdown', 'json', or 'cli'")
	}
}

func renderToXML(skills []Response) string {
	var b strings.Builder
	b.WriteString("<skills count=\"")
	b.WriteString(fmt.Sprintf("%d", len(skills)))
	b.WriteString("\">\n")

	for _, p := range skills {
		b.WriteString("  <skill id=\"")
		b.WriteString(escapeXML(p.ID))
		b.WriteString("\" name=\"")
		b.WriteString(escapeXML(p.Name))
		b.WriteString("\"><![CDATA[\n")
		b.WriteString(p.Content)
		b.WriteString("\n]]></skill>\n")
	}

	b.WriteString("</skills>")
	return b.String()
}

func renderToMarkdown(skills []Response) string {
	var b strings.Builder
	b.WriteString("# Combined Skills (")
	b.WriteString(fmt.Sprintf("%d", len(skills)))
	b.WriteString(")\n\n")
	b.WriteString("---\n\n")

	for i, p := range skills {
		b.WriteString("## ")
		b.WriteString(fmt.Sprintf("%d", i+1))
		b.WriteString(". ")
		b.WriteString(p.Name)
		b.WriteString("\n\n")

		if p.Description != "" {
			b.WriteString("> ")
			b.WriteString(p.Description)
			b.WriteString("\n\n")
		}

		if len(p.Modes) > 0 {
			b.WriteString("**Modes:** ")
			b.WriteString(strings.Join(p.Modes, " / "))
			b.WriteString("\n")
		}

		if len(p.Tags) > 0 {
			b.WriteString("**Tags:** ")
			for i, tag := range p.Tags {
				if i > 0 {
					b.WriteString(" ")
				}
				b.WriteString("`")
				b.WriteString(tag)
				b.WriteString("`")
			}
			b.WriteString("\n")
		}

		b.WriteString("\n### Content\n\n```\n")
		b.WriteString(p.Content)
		b.WriteString("\n```\n\n---\n\n")
	}

	return b.String()
}

func renderToJSON(skills []Response) string {
	type jsonSkill struct {
		ID          string   `json:"id"`
		Name        string   `json:"name"`
		Description string   `json:"description,omitempty"`
		Modes       []string `json:"modes,omitempty"`
		Tags        []string `json:"tags,omitempty"`
		Content     string   `json:"content"`
	}

	type jsonOutput struct {
		Combined bool        `json:"combined"`
		Count    int         `json:"count"`
		Skills   []jsonSkill `json:"skills"`
	}

	output := jsonOutput{
		Combined: true,
		Count:    len(skills),
		Skills:   make([]jsonSkill, len(skills)),
	}

	for i, p := range skills {
		output.Skills[i] = jsonSkill{
			ID:          p.ID,
			Name:        p.Name,
			Description: p.Description,
			Modes:       p.Modes,
			Tags:        p.Tags,
			Content:     p.Content,
		}
	}

	data, _ := json.MarshalIndent(output, "", "  ")
	return string(data)
}

func renderToCLI(skills []Response) string {
	ids := make([]string, len(skills))
	for i, s := range skills {
		ids[i] = s.ID
	}
	return "prompt-manager skill read " + strings.Join(ids, " ")
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}
