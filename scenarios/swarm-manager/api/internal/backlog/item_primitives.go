package backlog

import (
	"strings"
)

// ItemAttacher is the minimal milestone-side hook Service.Create uses
// to record a new item as a member of an milestone without rewriting
// the item file. Satisfied by both milestones.Service and the batch
// handler's MilestoneAssigner.
type ItemAttacher interface {
	RememberItem(milestoneName, ref string) error
}

// ItemPatch is a struct-based patch describing updatable BacklogItem
// fields. A nil pointer leaves the field unchanged; a non-nil pointer
// applies the value (including explicit empty values that clear the
// field). This shape is the single source of truth for item mutation used
// by both the HTTP PATCH handler and the proposals.OpUpdateItem path, so
// adding a new updatable field requires touching exactly one place.
type ItemPatch struct {
	Title           *string
	Description     *string
	Status          *string
	Priority        *int
	Tags            *[]string
	DependsOn       *[]string
	Milestone      *string
	Effort          *string
	AcceptanceAllow *[]string
	AcceptanceDeny  *[]string
	Creates         *[]string
	SpawnedFrom     *string
	PlanRef         *PlanRef
	PlanRefSet      bool
	Note            *string
}

// ApplyItemPatch mutates item in-place according to patch. Callers remain
// responsible for dependency validation (Store.ValidateDependencies),
// effort normalization, and status-transition gating — those live above
// this helper because different callers validate differently (PATCH
// handler rejects at request time; proposals rejects at Validate time).
func ApplyItemPatch(item *BacklogItem, patch ItemPatch) {
	contractChanged := patch.Title != nil || patch.Description != nil ||
		patch.AcceptanceAllow != nil || patch.AcceptanceDeny != nil ||
		patch.Creates != nil || patch.PlanRefSet
	if patch.Title != nil {
		item.Title = strings.TrimSpace(*patch.Title)
	}
	if patch.Description != nil {
		item.Description = *patch.Description
	}
	if patch.Status != nil {
		item.Status = BacklogStatus(*patch.Status)
	}
	if patch.Priority != nil {
		item.Priority = *patch.Priority
	}
	if patch.Tags != nil {
		item.Tags = cloneStringsOrEmpty(*patch.Tags)
	}
	if patch.DependsOn != nil {
		item.DependsOn = cloneStrings(*patch.DependsOn)
	}
	if patch.Milestone != nil {
		item.Milestone = strings.TrimSpace(*patch.Milestone)
	}
	if patch.Effort != nil {
		item.Effort = strings.ToUpper(strings.TrimSpace(*patch.Effort))
	}
	if patch.AcceptanceAllow != nil {
		item.AcceptanceAllow = cloneStrings(*patch.AcceptanceAllow)
	}
	if patch.AcceptanceDeny != nil {
		item.AcceptanceDeny = cloneStrings(*patch.AcceptanceDeny)
	}
	if patch.Creates != nil {
		item.Creates = cloneStrings(*patch.Creates)
	}
	if patch.SpawnedFrom != nil {
		item.SpawnedFrom = strings.TrimSpace(*patch.SpawnedFrom)
	}
	if patch.PlanRefSet {
		item.PlanRef = normalizePlanRef(patch.PlanRef)
		// A changed (or explicitly rebound) plan reference always needs a new
		// acceptance. Keeping a historical acceptance here would authorize a
		// different canonical plan under the old decision.
		item.PlanAcceptance = nil
	}
	if patch.Note != nil {
		item.Note = strings.TrimSpace(*patch.Note)
	}
	if contractChanged {
		item.PlanAcceptance = nil
	}
}
