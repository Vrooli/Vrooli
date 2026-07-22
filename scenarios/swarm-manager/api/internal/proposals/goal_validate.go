package proposals

import (
	"errors"
	"fmt"
	"strings"
)

// GoalState is the minimal, read-only goal snapshot needed to validate a
// goal-targeted mutation list. It is deliberately data-only so callers can
// build it from goals.Service without coupling this package to goals.
type GoalState struct {
	Name       string
	Version    string
	Milestones map[string]GoalMilestoneState
	Closure    map[string]struct{}
	Targets    map[string]struct{}
}

type GoalMilestoneState struct {
	Items   map[string]struct{}
	Open    bool
	Archive bool
}

// ValidateGoal validates a mutation list intended for a goal target. Item
// operations are rejected here: a goal proposal may only mutate the goal
// graph, keeping the authority boundary unambiguous.
func ValidateGoal(p Proposal, state GoalState) error {
	if p.Form != FormMutationList {
		return fmt.Errorf("%w: goal proposal must use mutation_list", ErrInvalidProposal)
	}
	if strings.TrimSpace(p.BaseVersion) == "" {
		return fmt.Errorf("%w: goal proposal base version is required", ErrInvalidProposal)
	}
	if strings.TrimSpace(p.BaseVersion) != strings.TrimSpace(state.Version) {
		return fmt.Errorf("%w: proposal base version %q does not match goal version %q", ErrInvalidProposal, p.BaseVersion, state.Version)
	}
	if len(p.Mutations) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	var problems []error
	for i, m := range p.Mutations {
		prefix := fmt.Sprintf("mutations[%d]", i)
		if strings.TrimSpace(m.ID) == "" {
			problems = append(problems, fmt.Errorf("%s: id is required", prefix))
		} else if _, duplicate := seen[m.ID]; duplicate {
			problems = append(problems, fmt.Errorf("%s: duplicate id %q", prefix, m.ID))
		} else {
			seen[m.ID] = struct{}{}
		}
		if err := validateGoalMutation(m, state); err != nil {
			problems = append(problems, fmt.Errorf("%s: %w", prefix, err))
		}
	}
	if len(problems) == 0 {
		return nil
	}
	return errors.Join(append([]error{ErrInvalidProposal}, problems...)...)
}

func validateGoalMutation(m Mutation, state GoalState) error {
	milestone := strings.TrimSpace(m.MilestoneName)
	switch m.Op {
	case OpCreateMilestone:
		if m.GoalMilestone == nil || strings.TrimSpace(m.GoalMilestone.Name) == "" || strings.TrimSpace(m.GoalMilestone.Title) == "" {
			return fmt.Errorf("create_milestone requires goal_milestone name and title")
		}
		if _, exists := state.Milestones[m.GoalMilestone.Name]; exists {
			return fmt.Errorf("milestone %q already exists", m.GoalMilestone.Name)
		}
	case OpUpdateMilestone:
		if m.GoalMilestone == nil || strings.TrimSpace(m.GoalMilestone.Name) == "" {
			return fmt.Errorf("update_milestone requires goal_milestone name")
		}
		if _, exists := state.Milestones[m.GoalMilestone.Name]; !exists {
			return fmt.Errorf("unknown milestone %q", m.GoalMilestone.Name)
		}
	case OpArchiveMilestone:
		current, exists := state.Milestones[milestone]
		if milestone == "" || !exists {
			return fmt.Errorf("archive_milestone requires an existing milestone_name")
		}
		if current.Open && !m.DetachOpen {
			return fmt.Errorf("milestone %q has open member items; set detach_open explicitly", milestone)
		}
	case OpAssignMilestoneItems, OpUnassignMilestoneItems:
		if _, exists := state.Milestones[milestone]; milestone == "" || !exists {
			return fmt.Errorf("%s requires an existing milestone_name", m.Op)
		}
		if len(m.Items) == 0 {
			return fmt.Errorf("%s requires items", m.Op)
		}
		for _, item := range m.Items {
			if _, exists := state.Closure[item]; !exists {
				return fmt.Errorf("item %q is outside goal closure", item)
			}
		}
	case OpAddGoalTarget:
		if len(m.Targets) == 0 {
			return fmt.Errorf("add_goal_target requires targets")
		}
	case OpRemoveGoalTarget:
		if len(m.Targets) == 0 {
			return fmt.Errorf("remove_goal_target requires targets")
		}
		for _, target := range m.Targets {
			if _, exists := state.Targets[target]; !exists {
				return fmt.Errorf("target %q is not a current goal target", target)
			}
		}
	default:
		return fmt.Errorf("op %q is not valid for a goal proposal", m.Op)
	}
	return nil
}
