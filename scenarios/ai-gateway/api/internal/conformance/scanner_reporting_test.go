package conformance

import "testing"

func TestTitleForRule(t *testing.T) {
	if got := titleForRule("ai.direct_ollama_http"); got != "Direct Ollama HTTP usage" {
		t.Fatalf("titleForRule known rule = %q", got)
	}
	if got := titleForRule("ai.unknown"); got != "AI conformance finding" {
		t.Fatalf("titleForRule unknown rule = %q", got)
	}
}
