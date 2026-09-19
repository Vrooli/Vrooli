package planning

type Candidate struct {
	ID                       string
	Name                     string
	Eligible                 bool
	Cost, Effort, Repetition float64
	InputRevision            string
}
type (
	Slot struct {
		Date     string `json:"date"`
		SlotName string `json:"slotName,omitempty"`
		Mode     string `json:"mode,omitempty"`
		Quantity string `json:"quantity,omitempty"`
		Locked   bool   `json:"locked"`
		RecipeID string `json:"recipeId"`
	}
	Occurrence struct {
		Date       string `json:"date"`
		SlotName   string `json:"slotName,omitempty"`
		Mode       string `json:"mode,omitempty"`
		Quantity   string `json:"quantity,omitempty"`
		RecipeID   string `json:"recipeId"`
		RecipeName string `json:"recipeName"`
		Reason     string `json:"reason"`
		Locked     bool   `json:"locked"`
	}
	Unresolved struct {
		Date         string   `json:"date"`
		Code         string   `json:"code"`
		Message      string   `json:"message"`
		CandidateIDs []string `json:"candidateIds,omitempty"`
	}
	Draft struct {
		Occurrences      []Occurrence   `json:"occurrences"`
		Unresolved       []Unresolved   `json:"unresolved"`
		InputReferences  []string       `json:"inputReferences"`
		RunID            string         `json:"runId"`
		Seed             int64          `json:"seed"`
		ObjectiveVersion string         `json:"objectiveVersion"`
		Evaluation       PlanEvaluation `json:"evaluation"`
	}
	GenerateInput struct {
		Slots                                      []Slot
		Candidates                                 []Candidate
		Seed                                       int64
		CostWeight, EffortWeight, RepetitionWeight float64
		InputReferences                            []string
	}
)
