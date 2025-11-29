package validators

import (
	"scenario-completeness-scoring/pkg/collectors"
)

// ConvertRequirements converts collectors.Requirement slice to validators.Requirement slice
func ConvertRequirements(reqs []collectors.Requirement) []Requirement {
	result := make([]Requirement, len(reqs))
	for i, r := range reqs {
		validations := make([]Validation, len(r.Validation))
		for j, v := range r.Validation {
			validations[j] = Validation{
				Type:       v.Type,
				Ref:        v.Ref,
				WorkflowID: "", // Not present in collectors.ValidationRef
				Status:     v.Status,
			}
		}
		result[i] = Requirement{
			ID:                  r.ID,
			Title:               r.Title,
			Status:              r.Status,
			Priority:            r.Priority,
			Category:            r.Category,
			PRDRef:              r.PRDRef,
			OperationalTargetID: r.OperationalTargetID,
			Children:            r.Children,
			Validation:          validations,
		}
	}
	return result
}
