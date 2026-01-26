package docschema

import (
	"regexp"
	"strings"
)

// DOC: scenarios/prompt-manager/skills/core/documentation-health.md#bidirectional-reference-format

type ReferenceKind string

const (
	ReferenceKindCode ReferenceKind = "code"
	ReferenceKindDoc  ReferenceKind = "doc"
	ReferenceKindReq  ReferenceKind = "req"
)

type Reference struct {
	Kind   ReferenceKind
	Target string
	Raw    string
	Line   int
}

var (
	markdownReference   = regexp.MustCompile(`\[(CODE|DOC|REQ):\s*([^\]]+)\]`)
	docCommentReference = regexp.MustCompile(`^\s*//\s*DOC:\s*(\S+)`)
)

// ParseMarkdownReferences extracts [CODE:], [DOC:], and [REQ:] references from markdown.
func ParseMarkdownReferences(content string) []Reference {
	lines := strings.Split(content, "\n")
	var refs []Reference
	for i, line := range lines {
		matches := markdownReference.FindAllStringSubmatch(line, -1)
		if len(matches) == 0 {
			continue
		}
		for _, match := range matches {
			refs = append(refs, Reference{
				Kind:   referenceKindFromTag(match[1]),
				Target: strings.TrimSpace(match[2]),
				Raw:    match[0],
				Line:   i + 1,
			})
		}
	}
	return refs
}

// ParseDocCommentReferences extracts // DOC: references from code.
func ParseDocCommentReferences(content string) []Reference {
	lines := strings.Split(content, "\n")
	var refs []Reference
	for i, line := range lines {
		match := docCommentReference.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		refs = append(refs, Reference{
			Kind:   ReferenceKindDoc,
			Target: strings.TrimSpace(match[1]),
			Raw:    strings.TrimSpace(line),
			Line:   i + 1,
		})
	}
	return refs
}

func referenceKindFromTag(tag string) ReferenceKind {
	switch strings.ToUpper(strings.TrimSpace(tag)) {
	case "CODE":
		return ReferenceKindCode
	case "REQ":
		return ReferenceKindReq
	default:
		return ReferenceKindDoc
	}
}
