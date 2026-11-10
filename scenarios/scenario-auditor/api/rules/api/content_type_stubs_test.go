package api

import "testing"

func TestUnimplementedContentTypeRulesPreserveEmptyResults(t *testing.T) {
	rules := []struct {
		name  string
		check func([]byte, string) []Violation
	}{
		{"csv", CheckCSVContentTypeHeaders},
		{"html", CheckHTMLContentTypeHeaders},
		{"pdf", CheckPDFContentTypeHeaders},
		{"streaming", CheckStreamingContentTypeHeaders},
		{"plain-text", CheckPlainTextContentTypeHeaders},
		{"xml", CheckXMLContentTypeHeaders},
	}
	for _, rule := range rules {
		t.Run(rule.name, func(t *testing.T) {
			if got := rule.check([]byte("response"), "api/handler.go"); got != nil {
				t.Fatalf("unimplemented rule returned %v, want the existing nil result", got)
			}
		})
	}
}
