// Package initiatives provides CRUD operations and rollup computation for
// initiative groupings of backlog items.
package initiatives

// Initiative represents a named grouping of backlog items into a coherent
// work stream. Stored as individual JSON files under .vrooli/initiatives/.
type Initiative struct {
	Name        string   `json:"name"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Status      string   `json:"status"` // active, completed, archived
	Items       []string `json:"items"`  // "kind/name" references
	Created     string   `json:"created"`
	Updated     string   `json:"updated"`
	Note        string   `json:"note,omitempty"`
}

// RollupStatus provides aggregated status counts for an initiative's items.
type RollupStatus struct {
	Total      int `json:"total"`
	Completed  int `json:"completed"`
	InProgress int `json:"in_progress"`
	Failed     int `json:"failed"`
	Pending    int `json:"pending"`
}

// InitiativeWithRollup pairs an initiative with its computed rollup status.
type InitiativeWithRollup struct {
	Initiative Initiative   `json:"initiative"`
	Rollup     RollupStatus `json:"rollup"`
}

// CreateRequest holds validated fields for creating a new initiative.
type CreateRequest struct {
	Name        string   `json:"name"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Status      string   `json:"status,omitempty"`
	Items       []string `json:"items,omitempty"`
}

// UpdateRequest holds validated fields for updating an existing initiative.
type UpdateRequest struct {
	Title       *string   `json:"title,omitempty"`
	Description *string   `json:"description,omitempty"`
	Status      *string   `json:"status,omitempty"`
	Items       *[]string `json:"items,omitempty"`
	Note        *string   `json:"note,omitempty"`
}

// HasChanges reports whether the update request contains at least one field.
func (r UpdateRequest) HasChanges() bool {
	return r.Title != nil || r.Description != nil || r.Status != nil || r.Items != nil || r.Note != nil
}

// ValidateStatus returns true if the status string is valid.
func ValidateStatus(status string) bool {
	switch status {
	case "active", "completed", "archived":
		return true
	default:
		return false
	}
}
