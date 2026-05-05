package relationshiprefs

import (
	"regexp"
	"strings"
)

// Kind identifies the relationship-reference type.
type Kind string

const (
	KindCode Kind = "CODE"
	KindDoc  Kind = "DOC"
	KindReq  Kind = "REQ"
)

var (
	bracketRefPattern = regexp.MustCompile(`\[(CODE|DOC|REQ):\s*([^\]]+)\]`)
	docCommentPattern = regexp.MustCompile(`^\s*(?://|/\*|#)\s*DOC:\s*([^\s\*\n]+)`)
	fencePattern      = regexp.MustCompile("^(```|~~~)")
	inlineCodePattern = regexp.MustCompile("`[^`]*`")
)

// Reference is one parsed relationship reference.
type Reference struct {
	Kind   Kind
	Value  string
	Raw    string
	Line   int
	Column int
}

// ExtractMarkdownRefs extracts bracketed [CODE:], [DOC:], and [REQ:]
// references from markdown outside fenced code blocks and outside inline code.
func ExtractMarkdownRefs(content string) []Reference {
	var refs []Reference
	lines := strings.Split(content, "\n")
	inFence := false
	fenceMarker := ""
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if fenceMatch := fencePattern.FindStringSubmatch(trimmed); fenceMatch != nil {
			marker := fenceMatch[1]
			if inFence && marker == fenceMarker {
				inFence = false
				fenceMarker = ""
			} else if !inFence {
				inFence = true
				fenceMarker = marker
			}
			continue
		}
		if inFence {
			continue
		}

		searchLine := inlineCodePattern.ReplaceAllString(line, "")
		for _, match := range bracketRefPattern.FindAllStringSubmatchIndex(searchLine, -1) {
			if len(match) < 6 {
				continue
			}
			raw := searchLine[match[0]:match[1]]
			kind := Kind(searchLine[match[2]:match[3]])
			value := strings.TrimSpace(searchLine[match[4]:match[5]])
			if value == "" {
				continue
			}
			refs = append(refs, Reference{
				Kind:   kind,
				Value:  value,
				Raw:    raw,
				Line:   i + 1,
				Column: match[0] + 1,
			})
		}
	}
	return refs
}

// ExtractDocCommentRefs extracts standalone DOC: comments from code content.
//
// Supported forms are // DOC:, /* DOC:, and # DOC:. The match is anchored to
// the start of the line after whitespace so syntax mentions and string literals
// do not become references.
func ExtractDocCommentRefs(content string) []Reference {
	var refs []Reference
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		for _, match := range docCommentPattern.FindAllStringSubmatchIndex(line, -1) {
			if len(match) < 4 {
				continue
			}
			value := strings.TrimSpace(line[match[2]:match[3]])
			if value == "" {
				continue
			}
			refs = append(refs, Reference{
				Kind:   KindDoc,
				Value:  value,
				Raw:    strings.TrimSpace(line[match[0]:match[1]]),
				Line:   i + 1,
				Column: match[0] + 1,
			})
		}
	}
	return refs
}

// TargetPath strips the fragment and numeric line suffix from a file-like
// relationship reference value.
func TargetPath(value string) string {
	value = strings.TrimSpace(value)
	if idx := strings.Index(value, "#"); idx != -1 {
		value = value[:idx]
	}
	if idx := strings.LastIndex(value, ":"); idx != -1 {
		suffix := value[idx+1:]
		if suffix != "" && isDigits(suffix) {
			value = value[:idx]
		}
	}
	return strings.TrimSpace(value)
}

func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
