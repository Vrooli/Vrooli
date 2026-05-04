package docschema

import (
	"fmt"
	"strings"
	"time"
)

// FrontmatterIssue represents a single problem with a doc's YAML frontmatter
// or required body shape.
type FrontmatterIssue struct {
	DocPath  string `json:"doc_path"`
	Code     string `json:"code"`
	Field    string `json:"field,omitempty"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
}

// FrontmatterSchema declares the validation contract for a doc type's
// frontmatter block. All keys are top-level; nested objects are recognised but
// their contents are not validated beyond presence.
type FrontmatterSchema struct {
	// Required top-level keys. Missing keys produce an issue.
	RequiredKeys []string
	// EnumValues maps a key to the closed set of allowed string values.
	EnumValues map[string][]string
	// DateKeys must parse as YYYY-MM-DD when present.
	DateKeys []string
	// ListKeys must be non-empty lists when present.
	ListKeys []string
	// ObjectKeys must be objects (i.e. have indented sub-keys) when present.
	ObjectKeys []string
}

// PerfAuditFrontmatterSchema is the canonical schema for
// `docs/perf/<date>-<slug>.md` files. Adding a new required key here requires
// updating the embedded PERF-AUDIT.md template in lockstep.
var PerfAuditFrontmatterSchema = FrontmatterSchema{
	RequiredKeys: []string{"date", "scenario", "interactions", "status"},
	EnumValues: map[string][]string{
		"status": {"open", "in-progress", "fixed", "won't-fix", "superseded"},
	},
	DateKeys:   []string{"date"},
	ListKeys:   []string{"interactions"},
	ObjectKeys: []string{"traces"},
}

// frontmatter is the parsed result of a document's leading YAML block.
type frontmatter struct {
	scalars map[string]string
	lists   map[string][]string
	objects map[string]bool // present + non-empty?
	keys    map[string]bool // every top-level key seen, regardless of shape
	raw     string          // raw frontmatter text (between --- markers)
	present bool            // true iff we found delimited frontmatter
}

// extractFrontmatter pulls the leading `---\n...\n---\n` block off a markdown
// document, if present. Returns parsed frontmatter and the body without the
// frontmatter block.
func extractFrontmatter(content string) (*frontmatter, string) {
	fm := &frontmatter{
		scalars: map[string]string{},
		lists:   map[string][]string{},
		objects: map[string]bool{},
		keys:    map[string]bool{},
	}
	const delim = "---"
	// Strip UTF-8 BOM if present (avoids "\xef\xbb\xbf---" tripping the prefix check).
	const utf8BOM = "\xef\xbb\xbf"
	trimmed := strings.TrimPrefix(content, utf8BOM)
	if !strings.HasPrefix(trimmed, delim+"\n") && !strings.HasPrefix(trimmed, delim+"\r\n") {
		return fm, content
	}
	// Skip the opening delim line.
	rest := trimmed[len(delim):]
	rest = strings.TrimLeft(rest, "\r\n")
	// Find the closing delim (must be at the start of a line and followed by EOL or EOF).
	closeIdx := -1
	scan := rest
	offset := 0
	for {
		idx := strings.Index(scan, "\n"+delim)
		if idx < 0 {
			break
		}
		// `\n---` must be followed by end-of-line or EOF to count as the closing delim.
		end := offset + idx + 1 + len(delim)
		if end >= len(rest) || rest[end] == '\n' || rest[end] == '\r' {
			closeIdx = offset + idx
			break
		}
		// False positive (e.g. `--- something`). Continue scanning past this match.
		offset = end
		if offset >= len(rest) {
			break
		}
		scan = rest[offset:]
	}
	if closeIdx < 0 {
		return fm, content
	}
	raw := rest[:closeIdx]
	body := rest[closeIdx:]
	body = strings.TrimPrefix(body, "\n")
	body = strings.TrimPrefix(body, delim)
	body = strings.TrimLeft(body, "\r\n")
	fm.raw = raw
	fm.present = true
	parseFrontmatterBlock(raw, fm)
	return fm, body
}

// parseFrontmatterBlock reads a constrained subset of YAML:
//   - `key: value` -> scalar
//   - `key:` then indented `  - item` lines -> list of strings
//   - `key:` then indented `  subkey: value` lines -> object
//   - `# comment` lines are ignored
//
// More complex YAML (anchors, multi-line strings, nested lists, quoted keys)
// is not supported. Frontmatter for our doc types stays flat.
func parseFrontmatterBlock(raw string, fm *frontmatter) {
	lines := strings.Split(raw, "\n")
	currentKey := ""
	currentKind := "" // "list" | "object"
	for _, rawLine := range lines {
		line := strings.TrimRight(rawLine, "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// A line starting with whitespace continues the current key's block.
		if line != trimmed {
			if currentKey == "" {
				continue
			}
			inner := strings.TrimSpace(line)
			if strings.HasPrefix(inner, "- ") || inner == "-" {
				if currentKind == "" {
					currentKind = "list"
				}
				if currentKind != "list" {
					continue
				}
				val := strings.TrimSpace(strings.TrimPrefix(inner, "-"))
				val = stripInlineComment(val)
				val = unquote(val)
				if val != "" {
					fm.lists[currentKey] = append(fm.lists[currentKey], val)
				}
				continue
			}
			if colon := strings.Index(inner, ":"); colon > 0 {
				if currentKind == "" {
					currentKind = "object"
				}
				if currentKind != "object" {
					continue
				}
				fm.objects[currentKey] = true
			}
			continue
		}

		// A new top-level key. Reset block context.
		colon := strings.Index(line, ":")
		if colon <= 0 {
			currentKey = ""
			currentKind = ""
			continue
		}
		key := strings.TrimSpace(line[:colon])
		val := stripInlineComment(strings.TrimSpace(line[colon+1:]))
		fm.keys[key] = true
		currentKey = key
		currentKind = ""
		if val == "" {
			// key: with no value — block follows on indented lines.
			continue
		}
		fm.scalars[key] = unquote(val)
	}
}

// stripInlineComment removes a trailing ` # comment` from a value, but only
// when the `#` is preceded by whitespace (so URLs containing `#fragment` are
// preserved).
func stripInlineComment(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] == '#' && i > 0 && (s[i-1] == ' ' || s[i-1] == '\t') {
			return strings.TrimRight(s[:i], " \t")
		}
	}
	return s
}

func unquote(s string) string {
	if len(s) >= 2 {
		first, last := s[0], s[len(s)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// ValidateFrontmatter checks the parsed frontmatter against schema. Returns
// issues with DocPath unset; callers fill it in before reporting.
func ValidateFrontmatter(content string, schema FrontmatterSchema) []FrontmatterIssue {
	var issues []FrontmatterIssue
	fm, _ := extractFrontmatter(content)
	if !fm.present {
		issues = append(issues, FrontmatterIssue{
			Code:     "perf-audit:missing-frontmatter",
			Message:  "missing YAML frontmatter block; expected `---` delimited header",
			Severity: "error",
		})
		return issues
	}

	for _, key := range schema.RequiredKeys {
		if !fm.keys[key] {
			issues = append(issues, FrontmatterIssue{
				Code:     "perf-audit:missing-key",
				Field:    key,
				Message:  fmt.Sprintf("required frontmatter key missing: %s", key),
				Severity: "error",
			})
		}
	}

	for key, allowed := range schema.EnumValues {
		val, ok := fm.scalars[key]
		if !ok {
			continue
		}
		matched := false
		for _, candidate := range allowed {
			if val == candidate {
				matched = true
				break
			}
		}
		if !matched {
			issues = append(issues, FrontmatterIssue{
				Code:     "perf-audit:invalid-enum",
				Field:    key,
				Message:  fmt.Sprintf("frontmatter %s=%q not in allowed set %v", key, val, allowed),
				Severity: "error",
			})
		}
	}

	for _, key := range schema.DateKeys {
		val, ok := fm.scalars[key]
		if !ok {
			continue
		}
		if _, err := time.Parse("2006-01-02", val); err != nil {
			issues = append(issues, FrontmatterIssue{
				Code:     "perf-audit:invalid-date",
				Field:    key,
				Message:  fmt.Sprintf("frontmatter %s=%q is not a YYYY-MM-DD date", key, val),
				Severity: "error",
			})
		}
	}

	for _, key := range schema.ListKeys {
		if !fm.keys[key] {
			continue
		}
		if items, ok := fm.lists[key]; !ok || len(items) == 0 {
			issues = append(issues, FrontmatterIssue{
				Code:     "perf-audit:empty-list",
				Field:    key,
				Message:  fmt.Sprintf("frontmatter %s must be a non-empty list", key),
				Severity: "error",
			})
		}
	}

	for _, key := range schema.ObjectKeys {
		if !fm.keys[key] {
			continue
		}
		// Object can be empty in our schema (traces is optional content).
		// Just ensure it isn't accidentally given a scalar value.
		if val, hasScalar := fm.scalars[key]; hasScalar && strings.TrimSpace(val) != "" {
			issues = append(issues, FrontmatterIssue{
				Code:     "perf-audit:expected-object",
				Field:    key,
				Message:  fmt.Sprintf("frontmatter %s must be a YAML object (sub-keys), not a scalar value", key),
				Severity: "error",
			})
		}
	}

	return issues
}
