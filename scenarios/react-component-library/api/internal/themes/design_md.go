package themes

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ExtractFrontMatter returns the YAML between the leading and closing
// `---` markers of a DESIGN.md file. ok=false means the document does
// not start with `---` or has no closing marker.
func ExtractFrontMatter(src []byte) (string, bool) {
	// Tolerate UTF-8 BOM + leading whitespace.
	src = bytes.TrimPrefix(src, []byte{0xEF, 0xBB, 0xBF})
	src = bytes.TrimLeft(src, " \t\r\n")
	if !bytes.HasPrefix(src, []byte("---")) {
		return "", false
	}
	// Skip the opening "---" line.
	rest := src[3:]
	if idx := bytes.IndexByte(rest, '\n'); idx >= 0 {
		rest = rest[idx+1:]
	} else {
		return "", false
	}
	// Find the closing line that is exactly "---" (followed by EOL or EOF).
	lines := bytes.Split(rest, []byte("\n"))
	var fmLines [][]byte
	closed := false
	for _, line := range lines {
		trimmed := bytes.TrimRight(line, "\r")
		if string(trimmed) == "---" {
			closed = true
			break
		}
		fmLines = append(fmLines, line)
	}
	if !closed {
		return "", false
	}
	return string(bytes.Join(fmLines, []byte("\n"))), true
}

// designMD is the YAML-front-matter shape themes resolves. Matches
// the flow-verifier canonical schema; unknown top-level fields are
// ignored to keep the parser tolerant.
type designMD struct {
	ID         string                       `yaml:"id"`
	Name       string                       `yaml:"name"`
	Colors     map[string]string            `yaml:"colors"`
	Typography map[string]map[string]string `yaml:"typography"`
	Rounded    map[string]string            `yaml:"rounded"`
	Spacing    map[string]string            `yaml:"spacing"`
}

// ParseDesignMDToTheme parses a DESIGN.md byte slice, extracts its YAML
// front-matter, and projects it into a Theme. The scenarioFallbackID
// is used as the Theme.ID when the front-matter omits `id`.
func ParseDesignMDToTheme(src []byte, scenarioFallbackID string) (Theme, error) {
	front, ok := ExtractFrontMatter(src)
	if !ok {
		return Theme{}, ErrInvalidDesignMD{Scenario: scenarioFallbackID, Reason: "missing or malformed front-matter"}
	}
	var d designMD
	if err := yaml.Unmarshal([]byte(front), &d); err != nil {
		return Theme{}, ErrInvalidDesignMD{Scenario: scenarioFallbackID, Reason: fmt.Sprintf("invalid yaml: %v", err)}
	}
	if len(d.Colors) == 0 && len(d.Typography) == 0 && len(d.Rounded) == 0 && len(d.Spacing) == 0 {
		return Theme{}, ErrInvalidDesignMD{Scenario: scenarioFallbackID, Reason: "no theme tokens (colors / typography / rounded / spacing all empty)"}
	}
	tokens := map[string]string{}
	for k, v := range d.Colors {
		tokens["--color-"+slug(k)] = v
	}
	for k, v := range d.Rounded {
		tokens["--rounded-"+slug(k)] = v
	}
	for k, v := range d.Spacing {
		tokens["--spacing-"+slug(k)] = v
	}
	for variant, fields := range d.Typography {
		for prop, v := range fields {
			tokens["--typography-"+slug(variant)+"-"+slug(prop)] = v
		}
	}
	id := strings.TrimSpace(d.ID)
	if id == "" {
		id = scenarioFallbackID
	}
	name := strings.TrimSpace(d.Name)
	if name == "" {
		name = id
	}
	return Theme{
		ID:     "scenario:" + scenarioFallbackID,
		Name:   name,
		Tokens: tokens,
		Source: "scenario:" + scenarioFallbackID,
	}, nil
}

func slug(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")
	return s
}
