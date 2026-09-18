package planning

import "testing"

func TestPreviewSwapScopesOneOccurrence(t *testing.T) {
	draft := Draft{Occurrences: []Occurrence{{Date: "2026-09-20", RecipeID: "a", RecipeName: "A"}, {Date: "2026-09-21", RecipeID: "a", RecipeName: "A"}}}
	preview, err := PreviewSwap(draft, SwapRequest{Date: "2026-09-20", ReplacementID: "b", ReplacementName: "B"})
	if err != nil || len(preview.Changes) != 1 || preview.Draft.Occurrences[1].RecipeID != "a" {
		t.Fatalf("preview=%+v err=%v", preview, err)
	}
}

func TestPreviewSwapCanReplaceMatchingFutureOccurrences(t *testing.T) {
	draft := Draft{Occurrences: []Occurrence{{Date: "2026-09-20", RecipeID: "a", RecipeName: "A"}, {Date: "2026-09-21", RecipeID: "a", RecipeName: "A"}, {Date: "2026-09-22", RecipeID: "b", RecipeName: "B"}}}
	preview, err := PreviewSwap(draft, SwapRequest{Date: "2026-09-20", ReplacementID: "c", ReplacementName: "C", ReplaceMatchingFuture: true})
	if err != nil || len(preview.Changes) != 2 || preview.Draft.Occurrences[2].RecipeID != "b" {
		t.Fatalf("preview=%+v err=%v", preview, err)
	}
}

func TestPreviewSwapRejectsLockedOccurrence(t *testing.T) {
	_, err := PreviewSwap(Draft{Occurrences: []Occurrence{{Date: "2026-09-20", RecipeID: "a", Locked: true}}}, SwapRequest{Date: "2026-09-20", ReplacementID: "b", ReplacementName: "B"})
	if err == nil {
		t.Fatal("locked occurrence accepted")
	}
}
