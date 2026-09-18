package planning

import "fmt"

type SwapRequest struct {
	Date                  string
	ReplacementID         string
	ReplacementName       string
	ReplaceMatchingFuture bool
}

type SwapChange struct {
	Date       string `json:"date"`
	BeforeID   string `json:"beforeId"`
	BeforeName string `json:"beforeName"`
	AfterID    string `json:"afterId"`
	AfterName  string `json:"afterName"`
}

type SwapPreview struct {
	Draft   Draft        `json:"draft"`
	Changes []SwapChange `json:"changes"`
}

func PreviewSwap(draft Draft, request SwapRequest) (SwapPreview, error) {
	if request.Date == "" {
		return SwapPreview{}, fmt.Errorf("date is required")
	}
	if request.ReplacementID == "" || request.ReplacementName == "" {
		return SwapPreview{}, fmt.Errorf("replacement recipe is required")
	}
	out := draft
	out.Occurrences = append([]Occurrence(nil), draft.Occurrences...)
	changes := make([]SwapChange, 0, 1)
	found := false
	for i := range out.Occurrences {
		occurrence := &out.Occurrences[i]
		if occurrence.Date < request.Date || (!request.ReplaceMatchingFuture && occurrence.Date != request.Date) {
			continue
		}
		if request.ReplaceMatchingFuture && occurrence.Date != request.Date && occurrence.RecipeID != draftOccurrence(draft, request.Date).RecipeID {
			continue
		}
		if occurrence.Date == request.Date {
			found = true
		}
		if occurrence.Locked {
			return SwapPreview{}, fmt.Errorf("%s is locked and cannot be swapped", occurrence.Date)
		}
		changes = append(changes, SwapChange{Date: occurrence.Date, BeforeID: occurrence.RecipeID, BeforeName: occurrence.RecipeName, AfterID: request.ReplacementID, AfterName: request.ReplacementName})
		occurrence.RecipeID = request.ReplacementID
		occurrence.RecipeName = request.ReplacementName
		occurrence.Reason = "Manually swapped in the selected scope"
	}
	if !found {
		return SwapPreview{}, fmt.Errorf("no planned occurrence for %s", request.Date)
	}
	return SwapPreview{Draft: out, Changes: changes}, nil
}

func draftOccurrence(draft Draft, date string) Occurrence {
	for _, occurrence := range draft.Occurrences {
		if occurrence.Date == date {
			return occurrence
		}
	}
	return Occurrence{}
}
