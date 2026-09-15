// Package followup owns the typed recovery instruction shared by proposal,
// review, and backlog domains. It deliberately has no domain imports.
package followup

type Disposition string

const (
	DispositionRun      Disposition = "follow_up_run"
	DispositionReplan   Disposition = "replan"
	DispositionNewItems Disposition = "new_items"
)

type Contract struct {
	Steering    string      `json:"steering"`
	Disposition Disposition `json:"disposition"`
	Items       []ItemSpec  `json:"items,omitempty"`
}

// ItemSpec is the transport-safe child-work shape for a new_items recovery.
// Domain adapters validate the kind and create it through their own lifecycle.
type ItemSpec struct {
	Kind        string   `json:"kind"`
	Name        string   `json:"name"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Priority    int      `json:"priority,omitempty"`
	DependsOn   []string `json:"depends_on,omitempty"`
}
