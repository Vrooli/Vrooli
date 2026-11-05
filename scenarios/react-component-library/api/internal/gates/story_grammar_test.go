package gates

import (
	"encoding/json"
	"testing"

	"react-component-library/internal/components"
)

func TestStoryGrammarBoundaryRedundancyRequiresAxisOnlyArguments(t *testing.T) {
	contract := &components.StoryContract{Args: components.StoryArgsSchema{Fields: []components.StoryField{
		{Path: "tone", Kind: components.StoryFieldEnum},
		{Path: "children", Kind: components.StoryFieldStructured},
	}}}

	axisOnly := map[string]json.RawMessage{"tone": json.RawMessage(`"warning"`)}
	if !storyGrammarBoundaryArgsAreAxisOnly(contract, axisOnly) {
		t.Fatal("an enum-only boundary should be classified as axis-only")
	}

	meaningfulBoundary := map[string]json.RawMessage{
		"tone":     json.RawMessage(`"warning"`),
		"children": json.RawMessage(`"Long content"`),
	}
	if storyGrammarBoundaryArgsAreAxisOnly(contract, meaningfulBoundary) {
		t.Fatal("a boundary with a non-enum condition must not be classified as axis-only")
	}
}

func TestStoryGrammarAllowsDocumentedInvisibleStoryWithoutExpectation(t *testing.T) {
	contract := &components.StoryContract{Stories: []components.StoryDefinition{
		{ID: "portal", Role: "anatomy", RendersNothing: true, RendersNothingReason: "The host supplies children after hydration."},
	}}
	if storyNeedsExpectation(contract.Stories[0]) {
		t.Fatal("an explicitly invisible story must not require a visual expectation")
	}
	if !storyNeedsExpectation(components.StoryDefinition{ID: "rendered", Role: "anatomy"}) {
		t.Fatal("a rendered story without an expectation must remain a finding")
	}
}

func TestStoryGrammarPreservesBoundaryWithExplicitStateMetadata(t *testing.T) {
	story := components.StoryDefinition{Role: "boundary", States: []string{"error"}}
	if !storyGrammarBoundaryHasExplicitState(story) {
		t.Fatal("boundary state metadata should preserve a meaningful boundary frame")
	}
}
