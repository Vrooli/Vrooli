package backend

import "testing"

func TestPromptHashIgnoresOnlyTheSelection(t *testing.T) {
	// [REQ:P0-017i] An answer names the prompt it answers by hash. Moving the
	// selection does not change the prompt; its text or options do.
	base := PendingPrompt{Kind: "question", Text: "Pick one", Options: []PromptOption{{Key: "1", Label: "Red", Selected: true}, {Key: "2", Label: "Blue"}}}
	moved := base
	moved.Options = []PromptOption{{Key: "1", Label: "Red"}, {Key: "2", Label: "Blue", Selected: true}}
	if PromptHash(&base) == "" || PromptHash(&base) != PromptHash(&moved) {
		t.Fatalf("hash changed with the selection: %q vs %q", PromptHash(&base), PromptHash(&moved))
	}
	retext := base
	retext.Text = "Pick another"
	relabel := base
	relabel.Options = []PromptOption{{Key: "1", Label: "Green", Selected: true}, {Key: "2", Label: "Blue"}}
	if PromptHash(&retext) == PromptHash(&base) || PromptHash(&relabel) == PromptHash(&base) {
		t.Fatal("hash did not change with the prompt's text or options")
	}
	if PromptHash(nil) != "" {
		t.Fatal("a missing prompt has no hash")
	}
}
