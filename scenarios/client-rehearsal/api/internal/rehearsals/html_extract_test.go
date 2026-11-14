package rehearsals

import "testing"

func TestSafeHTMLTitleExtractionIsBoundedAndDecodesEntities(t *testing.T) {
	title, ok := extractSafeHTMLTitle(`<html><head><TITLE>  Project &amp; Status <em>Board</em> </TITLE></head></html>`)
	if !ok || title != "Project & Status Board" {
		t.Fatalf("unexpected extracted title: %q, ok=%v", title, ok)
	}
	for _, markup := range []string{"", "<html><title> ", "<html><title></title>", "<title>bad\x00title</title>"} {
		if title, ok := extractSafeHTMLTitle(markup); ok {
			t.Errorf("accepted invalid title input %q as %q", markup, title)
		}
	}
}

func TestHTMLExtractNodeAllowsOnlyStaticTitleSelectors(t *testing.T) {
	params := map[string]any{
		"options": map[string]any{},
		"extractionValues": map[string]any{"values": []any{
			map[string]any{"key": "title", "cssSelector": "title"},
		}},
	}
	node, err := normalizeWorkflowNodeVersion("html-node", "Extract title", "n8n-nodes-base.htmlExtract", params, 1)
	if err != nil || node.HTMLExtractFields["title"] != "title" {
		t.Fatalf("normalize bounded title extraction: node=%#v err=%v", node, err)
	}
	params["extractionValues"].(map[string]any)["values"] = []any{map[string]any{"key": "title", "cssSelector": "script"}}
	if _, err := normalizeWorkflowNodeVersion("html-node", "Extract title", "n8n-nodes-base.htmlExtract", params, 1); err == nil {
		t.Fatal("accepted an unsupported or potentially active selector")
	}
}
