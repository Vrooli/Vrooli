package docschema

import (
	"strings"

	"github.com/vrooli/api-core/relationshiprefs"
)

// DOC: ../prompt-manager/skills/core/documentation-health.md#bidirectional-reference-format

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

// ParseMarkdownReferences extracts [CODE:], [DOC:], and [REQ:] references from markdown.
func ParseMarkdownReferences(content string) []Reference {
	var refs []Reference
	for _, ref := range relationshiprefs.ExtractMarkdownRefs(content) {
		refs = append(refs, Reference{
			Kind:   referenceKindFromShared(ref.Kind),
			Target: strings.TrimSpace(ref.Value),
			Raw:    ref.Raw,
			Line:   ref.Line,
		})
	}
	return refs
}

// ParseDocCommentReferences extracts DOC references from code.
func ParseDocCommentReferences(content string) []Reference {
	var refs []Reference
	for _, ref := range relationshiprefs.ExtractDocCommentRefs(content) {
		refs = append(refs, Reference{
			Kind:   ReferenceKindDoc,
			Target: strings.TrimSpace(ref.Value),
			Raw:    ref.Raw,
			Line:   ref.Line,
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

func referenceKindFromShared(kind relationshiprefs.Kind) ReferenceKind {
	switch kind {
	case relationshiprefs.KindCode:
		return ReferenceKindCode
	case relationshiprefs.KindReq:
		return ReferenceKindReq
	default:
		return ReferenceKindDoc
	}
}
