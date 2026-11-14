package planning

import "fmt"

type SwapRequest struct {
	Date                  string
	SlotName              string
	ReplacementID         string
	ReplacementRevision   int64
	ReplacementName       string
	ReplaceMatchingFuture bool
}

type SwapChange struct {
	Date           string `json:"date"`
	SlotName       string `json:"slotName"`
	BeforeID       string `json:"beforeId"`
	BeforeRevision int64  `json:"beforeRevision"`
	BeforeName     string `json:"beforeName"`
	AfterID        string `json:"afterId"`
	AfterName      string `json:"afterName"`
}

type SwapPreview struct {
	Draft                Draft                 `json:"draft"`
	Changes              []SwapChange          `json:"changes"`
	RelatedOccurrences   []RelatedOccurrence   `json:"relatedOccurrences,omitempty"`
	PreparedBatchImpacts []PreparedBatchImpact `json:"preparedBatchImpacts,omitempty"`
}

// RelatedOccurrence identifies a later use of the meal being replaced. It is
// a review dependency only; the planner does not infer or edit a batch/leftover
// relationship from two matching recipe IDs.
type RelatedOccurrence struct {
	Date           string `json:"date"`
	SlotName       string `json:"slotName"`
	RecipeRevision int64  `json:"recipeRevision"`
	RecipeName     string `json:"recipeName"`
	Locked         bool   `json:"locked"`
}

// PreparedBatchImpact reports actual available portions associated with a
// recipe removed by a swap. It is review context only; plans never allocate
// or consume batch portions implicitly.
type PreparedBatchImpact struct {
	BatchID        string `json:"batchId"`
	RecipeID       string `json:"recipeId"`
	RecipeName     string `json:"recipeName"`
	RecipeRevision int64  `json:"recipeRevision"`
	Available      string `json:"available"`
	Unit           string `json:"unit"`
}

func PreviewSwap(draft Draft, request SwapRequest) (SwapPreview, error) {
	if request.Date == "" {
		return SwapPreview{}, fmt.Errorf("date is required")
	}
	if request.SlotName == "" {
		return SwapPreview{}, fmt.Errorf("slot name is required")
	}
	if request.ReplacementID == "" || request.ReplacementName == "" {
		return SwapPreview{}, fmt.Errorf("replacement recipe is required")
	}
	out := draft
	out.Occurrences = append([]Occurrence(nil), draft.Occurrences...)
	changes := make([]SwapChange, 0, 1)
	found := false
	var replacedRecipeID string
	for i := range out.Occurrences {
		occurrence := &out.Occurrences[i]
		if occurrence.SlotName != request.SlotName || occurrence.Date < request.Date || (!request.ReplaceMatchingFuture && occurrence.Date != request.Date) {
			continue
		}
		if request.ReplaceMatchingFuture && occurrence.Date != request.Date && occurrence.RecipeID != draftOccurrence(draft, request.Date, request.SlotName).RecipeID {
			continue
		}
		if occurrence.Date == request.Date {
			found = true
			replacedRecipeID = occurrence.RecipeID
		}
		if occurrence.Locked {
			return SwapPreview{}, fmt.Errorf("%s is locked and cannot be swapped", occurrence.Date)
		}
		changes = append(changes, SwapChange{Date: occurrence.Date, SlotName: occurrence.SlotName, BeforeID: occurrence.RecipeID, BeforeRevision: occurrence.RecipeRevision, BeforeName: occurrence.RecipeName, AfterID: request.ReplacementID, AfterName: request.ReplacementName})
		occurrence.RecipeID = request.ReplacementID
		occurrence.RecipeRevision = request.ReplacementRevision
		occurrence.RecipeName = request.ReplacementName
		occurrence.Reason = "Manually swapped in the selected scope"
	}
	if !found {
		return SwapPreview{}, fmt.Errorf("no planned occurrence for %s", request.Date)
	}
	related := make([]RelatedOccurrence, 0)
	if replacedRecipeID != "" && !request.ReplaceMatchingFuture {
		for _, occurrence := range draft.Occurrences {
			if occurrence.Date <= request.Date || occurrence.RecipeID != replacedRecipeID || occurrence.RecipeID == request.ReplacementID {
				continue
			}
			related = append(related, RelatedOccurrence{Date: occurrence.Date, SlotName: occurrence.SlotName, RecipeRevision: occurrence.RecipeRevision, RecipeName: occurrence.RecipeName, Locked: occurrence.Locked})
		}
	}
	return SwapPreview{Draft: out, Changes: changes, RelatedOccurrences: related}, nil
}

func draftOccurrence(draft Draft, date, slotName string) Occurrence {
	for _, occurrence := range draft.Occurrences {
		if occurrence.Date == date && occurrence.SlotName == slotName {
			return occurrence
		}
	}
	return Occurrence{}
}
