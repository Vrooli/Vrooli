package docschema

import "testing"

func TestParseMarkdownReferences(t *testing.T) {
	content := "" +
		"Inline example: `[CODE: examples/ignored.go]`.\n" +
		"```markdown\n" +
		"[CODE: examples/fenced.go]\n" +
		"```\n" +
		"See [CODE: api/server.go] for runtime.\n" +
		"Reference [DOC: docs/QUICKSTART.md#setup] and [REQ: OT-P0-001].\n"

	refs := ParseMarkdownReferences(content)
	if len(refs) != 3 {
		t.Fatalf("expected 3 references, got %d", len(refs))
	}
	if refs[0].Kind != ReferenceKindCode || refs[0].Target != "api/server.go" || refs[0].Line != 5 {
		t.Fatalf("unexpected code reference: %#v", refs[0])
	}
	if refs[1].Kind != ReferenceKindDoc || refs[1].Target != "docs/QUICKSTART.md#setup" {
		t.Fatalf("unexpected doc reference: %#v", refs[1])
	}
	if refs[2].Kind != ReferenceKindReq || refs[2].Target != "OT-P0-001" {
		t.Fatalf("unexpected req reference: %#v", refs[2])
	}
}

func TestParseDocCommentReferences(t *testing.T) {
	content := "var _ = \"// DOC: docs/ignored.md\"\n" +
		"// DOC: docs/reference/api-endpoints.md#health\n" +
		"func handler() {}\n" +
		"// DOC: PRD.md#overview\n"

	refs := ParseDocCommentReferences(content)
	if len(refs) != 2 {
		t.Fatalf("expected 2 references, got %d", len(refs))
	}
	if refs[0].Kind != ReferenceKindDoc || refs[0].Target != "docs/reference/api-endpoints.md#health" || refs[0].Line != 2 {
		t.Fatalf("unexpected doc comment ref: %#v", refs[0])
	}
	if refs[1].Target != "PRD.md#overview" {
		t.Fatalf("unexpected doc comment ref: %#v", refs[1])
	}
}
