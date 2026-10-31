package planning

import "testing"

func TestPreviewSwapScopesOneOccurrence(t *testing.T) {
	draft := Draft{Occurrences: []Occurrence{{Date: "2026-09-20", SlotName: "breakfast", RecipeID: "a", RecipeName: "A"}, {Date: "2026-09-20", SlotName: "dinner", RecipeID: "d", RecipeName: "Dinner"}, {Date: "2026-09-21", SlotName: "breakfast", RecipeID: "a", RecipeName: "A"}}}
	preview, err := PreviewSwap(draft, SwapRequest{Date: "2026-09-20", SlotName: "breakfast", ReplacementID: "b", ReplacementName: "B"})
	if err != nil || len(preview.Changes) != 1 || preview.Draft.Occurrences[1].RecipeID != "d" || preview.Draft.Occurrences[2].RecipeID != "a" {
		t.Fatalf("preview=%+v err=%v", preview, err)
	}
}

func TestPreviewSwapCanReplaceMatchingFutureOccurrences(t *testing.T) {
	draft := Draft{Occurrences: []Occurrence{{Date: "2026-09-20", SlotName: "dinner", RecipeID: "a", RecipeName: "A"}, {Date: "2026-09-21", SlotName: "dinner", RecipeID: "a", RecipeName: "A"}, {Date: "2026-09-22", SlotName: "dinner", RecipeID: "b", RecipeName: "B"}, {Date: "2026-09-21", SlotName: "lunch", RecipeID: "a", RecipeName: "A"}}}
	preview, err := PreviewSwap(draft, SwapRequest{Date: "2026-09-20", SlotName: "dinner", ReplacementID: "c", ReplacementName: "C", ReplaceMatchingFuture: true})
	if err != nil || len(preview.Changes) != 2 || preview.Draft.Occurrences[2].RecipeID != "b" || preview.Draft.Occurrences[3].RecipeID != "a" {
		t.Fatalf("preview=%+v err=%v", preview, err)
	}
}

func TestPreviewSwapIgnoresLockedSiblingSlot(t *testing.T) {
	draft := Draft{Occurrences: []Occurrence{{Date: "2026-09-20", SlotName: "breakfast", RecipeID: "a", RecipeName: "A"}, {Date: "2026-09-20", SlotName: "dinner", RecipeID: "d", RecipeName: "Dinner", Locked: true}}}
	preview, err := PreviewSwap(draft, SwapRequest{Date: "2026-09-20", SlotName: "breakfast", ReplacementID: "b", ReplacementName: "B"})
	if err != nil || len(preview.Changes) != 1 || preview.Draft.Occurrences[0].RecipeID != "b" || preview.Draft.Occurrences[1].RecipeID != "d" || !preview.Draft.Occurrences[1].Locked {
		t.Fatalf("preview=%+v err=%v", preview, err)
	}
}

func TestPreviewSwapFlagsLaterSameRecipeOccurrenceForExplicitReview(t *testing.T) {
	draft := Draft{Occurrences: []Occurrence{
		{Date: "2026-09-20", SlotName: "dinner", RecipeID: "tofu", RecipeRevision: 7, RecipeName: "Tofu bowls"},
		{Date: "2026-09-21", SlotName: "lunch", RecipeID: "tofu", RecipeRevision: 7, RecipeName: "Tofu bowls", Locked: true},
	}}
	preview, err := PreviewSwap(draft, SwapRequest{Date: "2026-09-20", SlotName: "dinner", ReplacementID: "soup", ReplacementName: "Soup"})
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.RelatedOccurrences) != 1 || preview.RelatedOccurrences[0].Date != "2026-09-21" || preview.RelatedOccurrences[0].SlotName != "lunch" || preview.RelatedOccurrences[0].RecipeRevision != 7 || !preview.RelatedOccurrences[0].Locked {
		t.Fatalf("later same-recipe use must remain intact and require explicit review: %+v", preview.RelatedOccurrences)
	}
	if len(preview.Changes) != 1 || preview.Changes[0].BeforeRevision != 7 {
		t.Fatalf("swap change must retain the exact source recipe revision: %+v", preview.Changes)
	}
	if preview.Draft.Occurrences[1].RecipeID != "tofu" {
		t.Fatalf("review context must not silently change the later occurrence: %+v", preview.Draft.Occurrences)
	}
}

func TestPreviewSwapMatchingFutureReplacementsAreChangesNotRelatedUses(t *testing.T) {
	draft := Draft{Occurrences: []Occurrence{
		{Date: "2026-09-20", SlotName: "dinner", RecipeID: "tofu", RecipeName: "Tofu bowls"},
		{Date: "2026-09-21", SlotName: "dinner", RecipeID: "tofu", RecipeName: "Tofu bowls"},
	}}
	preview, err := PreviewSwap(draft, SwapRequest{Date: "2026-09-20", SlotName: "dinner", ReplacementID: "soup", ReplacementName: "Soup", ReplaceMatchingFuture: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Changes) != 2 || len(preview.RelatedOccurrences) != 0 {
		t.Fatalf("explicit future replacements are direct changes, not leftover assumptions: %+v", preview)
	}
}

func TestPreviewSwapRejectsLockedOccurrence(t *testing.T) {
	_, err := PreviewSwap(Draft{Occurrences: []Occurrence{{Date: "2026-09-20", SlotName: "dinner", RecipeID: "a", Locked: true}}}, SwapRequest{Date: "2026-09-20", SlotName: "dinner", ReplacementID: "b", ReplacementName: "B"})
	if err == nil {
		t.Fatal("locked occurrence accepted")
	}
}
