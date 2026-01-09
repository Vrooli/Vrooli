package sync

import (
	"context"

	"test-genie/internal/requirements/parsing"
	"test-genie/internal/requirements/types"
)

// StatusUpdater updates validation and requirement statuses.
type StatusUpdater struct {
	// Configuration options
}

// NewStatusUpdater creates a new StatusUpdater.
func NewStatusUpdater() *StatusUpdater {
	return &StatusUpdater{}
}

// UpdateStatuses updates validation statuses based on enriched data.
func (u *StatusUpdater) UpdateStatuses(ctx context.Context, index *parsing.ModuleIndex, evidence *types.EvidenceBundle) ([]Change, error) {
	if index == nil || evidence == nil {
		return nil, nil
	}

	var changes []Change

	for _, module := range index.Modules {
		select {
		case <-ctx.Done():
			return changes, ctx.Err()
		default:
		}

		moduleChanges := u.updateModule(module)
		changes = append(changes, moduleChanges...)
	}

	return changes, nil
}

// updateModule updates statuses for all requirements in a module.
func (u *StatusUpdater) updateModule(module *types.RequirementModule) []Change {
	var changes []Change

	for i := range module.Requirements {
		req := &module.Requirements[i]
		reqChanges := u.updateRequirement(req, module.FilePath)
		changes = append(changes, reqChanges...)
	}

	return changes
}

// updateRequirement updates statuses for a requirement and its validations.
func (u *StatusUpdater) updateRequirement(req *types.Requirement, filePath string) []Change {
	var changes []Change

	// Update validation statuses
	for i := range req.Validations {
		val := &req.Validations[i]
		if valChange := u.updateValidationStatus(val, req.ID, filePath); valChange != nil {
			changes = append(changes, *valChange)
		}
	}

	// Update requirement status based on validation results
	if reqChange := u.updateRequirementStatus(req, filePath); reqChange != nil {
		changes = append(changes, *reqChange)
	}

	return changes
}

// updateValidationStatus updates a single validation status.
func (u *StatusUpdater) updateValidationStatus(val *types.Validation, reqID, filePath string) *Change {
	if val.LiveStatus == "" || val.LiveStatus == types.LiveUnknown {
		return nil
	}

	newStatus := types.DeriveValidationStatus(val.LiveStatus)
	if newStatus == val.Status {
		return nil // No change
	}

	change := &Change{
		Type:          ChangeTypeStatusUpdate,
		FilePath:      filePath,
		RequirementID: reqID,
		Field:         "validation.status",
		OldValue:      string(val.Status),
		NewValue:      string(newStatus),
	}

	// Apply the change
	val.Status = newStatus

	return change
}

// updateRequirementStatus updates requirement status based on its validations.
func (u *StatusUpdater) updateRequirementStatus(req *types.Requirement, filePath string) *Change {
	if len(req.Validations) == 0 {
		return nil
	}

	// Collect validation statuses
	var valStatuses []types.ValidationStatus
	for _, val := range req.Validations {
		valStatuses = append(valStatuses, val.Status)
	}

	newStatus := types.DeriveRequirementStatus(req.Status, valStatuses)
	if newStatus == req.Status {
		return nil // No change
	}

	change := &Change{
		Type:          ChangeTypeStatusUpdate,
		FilePath:      filePath,
		RequirementID: req.ID,
		Field:         "status",
		OldValue:      string(req.Status),
		NewValue:      string(newStatus),
	}

	// Apply the change
	req.Status = newStatus

	return change
}

// ShouldUpdateValidationStatus determines if a validation status should be updated.
func ShouldUpdateValidationStatus(current types.ValidationStatus, live types.LiveStatus) bool {
	if live == "" || live == types.LiveUnknown {
		return false
	}

	derived := types.DeriveValidationStatus(live)
	return derived != current
}

// ShouldUpdateRequirementStatus determines if a requirement status should be updated.
func ShouldUpdateRequirementStatus(current types.DeclaredStatus, validations []types.Validation) bool {
	if len(validations) == 0 {
		return false
	}

	var valStatuses []types.ValidationStatus
	for _, val := range validations {
		valStatuses = append(valStatuses, val.Status)
	}

	derived := types.DeriveRequirementStatus(current, valStatuses)
	return derived != current
}
