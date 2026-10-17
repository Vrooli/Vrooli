package generation

import (
	"encoding/json"
	"fmt"
	"strings"
)

// parseGeneratedJSON extracts a JSON object from an AI response that may wrap it
// in markdown fences or prose. Ported from the old handlers.parseGeneratedJSON.
func parseGeneratedJSON(text string) (map[string]any, error) {
	text = strings.TrimSpace(text)

	// Strip a leading/trailing markdown code fence if present.
	if strings.HasPrefix(text, "```") {
		lines := strings.Split(text, "\n")
		if len(lines) >= 3 {
			text = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}

	// Narrow to the outermost {...} so trailing prose is ignored.
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start >= 0 && end > start {
		text = text[start : end+1]
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, fmt.Errorf("response was not valid JSON: %w", err)
	}
	return result, nil
}

// colorsFromJSON maps a parsed colors object onto a *Colors. Unknown keys are
// ignored; missing keys leave the corresponding field empty (the brands domain
// then leaves the stored value unchanged on merge).
func colorsFromJSON(data map[string]any) *Colors {
	return &Colors{
		Primary:    str(data, "primary"),
		Secondary:  str(data, "secondary"),
		Accent:     str(data, "accent"),
		Background: str(data, "background"),
		Surface:    str(data, "surface"),
		Text:       str(data, "text"),
		Error:      str(data, "error"),
	}
}

// typographyFromJSON maps a parsed typography object onto a *Typography.
func typographyFromJSON(data map[string]any) *Typography {
	return &Typography{
		HeadingFont:  str(data, "heading_font"),
		BodyFont:     str(data, "body_font"),
		MonoFont:     str(data, "mono_font"),
		BaseFontSize: str(data, "base_font_size"),
	}
}

// voiceFromJSON maps a parsed voice object onto a *Voice.
func voiceFromJSON(data map[string]any) *Voice {
	v := &Voice{
		Tone:  str(data, "tone"),
		Style: str(data, "style"),
	}
	if kw, ok := data["keywords"].([]any); ok {
		for _, k := range kw {
			if s, ok := k.(string); ok && strings.TrimSpace(s) != "" {
				v.Keywords = append(v.Keywords, s)
			}
		}
	}
	return v
}

// str returns data[key] as a string, or "" when absent or non-string.
func str(data map[string]any, key string) string {
	if v, ok := data[key].(string); ok {
		return v
	}
	return ""
}
