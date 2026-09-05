package main

import (
	"fmt"

	"swarm-manager/internal/goals"
	"swarm-manager/internal/proposals"
)

// goalProposalCreator is the composition adapter for intake proposals. The
// proposal package owns the typed contract; goals.Service remains the only
// writer for goal and milestone state.
type goalProposalCreator struct {
	service *goals.Service
}

func (c goalProposalCreator) CreateGoal(spec proposals.GoalSpec) error {
	if c.service == nil {
		return fmt.Errorf("goal service is not configured")
	}
	if _, err := c.service.Create(goals.CreateRequest{
		Name:        spec.Name,
		Title:       spec.Title,
		Description: spec.Description,
		Priority:    spec.Priority,
		Targets:     spec.Targets,
		SpawnedFrom: spec.SpawnedFrom,
	}); err != nil {
		return err
	}
	if len(spec.Targets) > 0 {
		if _, err := c.service.AddTargets(spec.Name, spec.Targets); err != nil {
			return err
		}
	}
	for _, milestone := range spec.Milestones {
		if _, err := c.service.CreateMilestone(spec.Name, goals.Milestone{
			Name:               milestone.Name,
			Title:              milestone.Title,
			Description:        milestone.Description,
			AcceptanceCriteria: milestone.AcceptanceCriteria,
			DependsOn:          milestone.DependsOn,
			Items:              milestone.Items,
			SpawnedFrom:        milestone.SpawnedFrom,
		}); err != nil {
			return err
		}
		if len(milestone.Items) > 0 {
			if _, err := c.service.AssignMilestoneItems(spec.Name, milestone.Name, milestone.Items); err != nil {
				return err
			}
		}
	}
	return nil
}
