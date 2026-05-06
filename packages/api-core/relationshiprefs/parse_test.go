package relationshiprefs

import (
	"reflect"
	"strings"
	"testing"
)

func TestExtractMarkdownRefs(t *testing.T) {
	content := strings.Join([]string{
		"Inline syntax example: `[CODE: path/to/file.ext]`",
		"```markdown",
		"- [CODE: src/example.ts#DoThing]",
		"```",
		"Real reference: [CODE: src/main.go#Run]",
		"Doc reference: [DOC: docs/guide.md#intro]",
		"Requirement: [REQ: OT-P0-005]",
	}, "\n")

	got := ExtractMarkdownRefs(content)
	want := []Reference{
		{Kind: KindCode, Value: "src/main.go#Run", Raw: "[CODE: src/main.go#Run]", Line: 5, Column: 17},
		{Kind: KindDoc, Value: "docs/guide.md#intro", Raw: "[DOC: docs/guide.md#intro]", Line: 6, Column: 16},
		{Kind: KindReq, Value: "OT-P0-005", Raw: "[REQ: OT-P0-005]", Line: 7, Column: 14},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ExtractMarkdownRefs() = %#v, want %#v", got, want)
	}
}

func TestExtractDocCommentRefs(t *testing.T) {
	content := strings.Join([]string{
		"package main",
		"// ValidateDocRefs checks // DOC: comments in code point to valid docs.",
		`var _ = "// DOC: docs/ignored.md"`,
		"// DOC: README.md",
		"/* DOC: docs/guide.md */",
		"# DOC: docs/python.md#section",
	}, "\n")

	got := ExtractDocCommentRefs(content)
	want := []Reference{
		{Kind: KindDoc, Value: "README.md", Raw: "// DOC: README.md", Line: 4, Column: 1},
		{Kind: KindDoc, Value: "docs/guide.md", Raw: "/* DOC: docs/guide.md", Line: 5, Column: 1},
		{Kind: KindDoc, Value: "docs/python.md#section", Raw: "# DOC: docs/python.md#section", Line: 6, Column: 1},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ExtractDocCommentRefs() = %#v, want %#v", got, want)
	}
}

func TestTargetPath(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"src/main.go", "src/main.go"},
		{"src/main.go#Run", "src/main.go"},
		{"src/main.go:42", "src/main.go"},
		{"docs/guide.md#intro", "docs/guide.md"},
		{"https://example.com/a:b", "https://example.com/a:b"},
	}
	for _, tt := range tests {
		if got := TargetPath(tt.in); got != tt.want {
			t.Fatalf("TargetPath(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
