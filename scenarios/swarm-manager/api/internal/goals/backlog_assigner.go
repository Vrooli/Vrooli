package goals

import (
	"fmt"
	"strings"

	"swarm-manager/internal/backlog"
)

// BacklogMilestoneAssigner is the sole bridge through which backlog item
// membership is written. Membership is owned by a goal's embedded milestone;
// a backlog item carries only the corresponding goal/milestone reference.
//
// It deliberately implements the former batch seam while callers are moved to
// the milestone-native request shape. Unqualified names are rejected so this
// bridge cannot recreate the retired milestone model.
type BacklogMilestoneAssigner struct{ goals *Service }

func NewBacklogMilestoneAssigner(service *Service) *BacklogMilestoneAssigner {
	return &BacklogMilestoneAssigner{goals: service}
}

func parseMilestoneRef(ref string) (string, string, error) {
	parts := strings.Split(strings.TrimSpace(ref), "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", fmt.Errorf("milestone reference must be goal/milestone")
	}
	return parts[0], parts[1], nil
}

func (a *BacklogMilestoneAssigner) Get(ref string) (*backlog.MilestoneSnapshot, error) {
	goalName, milestoneName, err := parseMilestoneRef(ref)
	if err != nil {
		return nil, err
	}
	goal, err := a.goals.Get(goalName)
	if err != nil {
		return nil, err
	}
	for _, milestone := range goal.Goal.Milestones {
		if milestone.Name == milestoneName {
			return &backlog.MilestoneSnapshot{Name: ref, Title: milestone.Title, Description: milestone.Description, Items: append([]string(nil), milestone.Items...)}, nil
		}
	}
	return nil, fmt.Errorf("milestone %q not found", ref)
}

// Create and Update retain compatibility with the batch seam only for an
// already explicit goal/milestone reference. Goal-only metadata such as
// milestone dependencies and plan refs has no goal/milestone equivalent.
func (a *BacklogMilestoneAssigner) Create(spec backlog.MilestoneSpec) error {
	goalName, milestoneName, err := parseMilestoneRef(spec.Name)
	if err != nil {
		return err
	}
	if len(spec.DependsOn) != 0 || spec.PlanRef != nil {
		return fmt.Errorf("legacy grouping metadata is not supported for milestones")
	}
	if _, err := a.goals.Get(goalName); err != nil {
		if _, createErr := a.goals.Create(CreateRequest{Name: goalName, Title: goalName, Priority: spec.Priority}); createErr != nil {
			return createErr
		}
	}
	_, err = a.goals.CreateMilestone(goalName, Milestone{Name: milestoneName, Title: spec.Title, Description: spec.Description})
	return err
}

func (a *BacklogMilestoneAssigner) Update(spec backlog.MilestoneSpec) error {
	goalName, milestoneName, err := parseMilestoneRef(spec.Name)
	if err != nil {
		return err
	}
	if len(spec.DependsOn) != 0 || spec.PlanRef != nil {
		return fmt.Errorf("legacy grouping metadata is not supported for milestones")
	}
	_, err = a.goals.UpdateMilestone(goalName, Milestone{Name: milestoneName, Title: spec.Title, Description: spec.Description})
	return err
}

func (a *BacklogMilestoneAssigner) Replace(snapshot backlog.MilestoneSnapshot) error {
	return a.Update(backlog.MilestoneSpec{Name: snapshot.Name, Title: snapshot.Title, Description: snapshot.Description})
}

func (a *BacklogMilestoneAssigner) Delete(ref string) error {
	return fmt.Errorf("deleting a goal or milestone through backlog grouping is not supported: %s", ref)
}

func (a *BacklogMilestoneAssigner) AddItems(ref string, items []string) error {
	goalName, milestoneName, err := parseMilestoneRef(ref)
	if err != nil {
		return err
	}
	if _, err := a.goals.AddTargets(goalName, items); err != nil {
		return err
	}
	_, err = a.goals.AssignMilestoneItems(goalName, milestoneName, items)
	return err
}

func (a *BacklogMilestoneAssigner) RememberItem(ref, item string) error {
	return a.AddItems(ref, []string{item})
}

func (a *BacklogMilestoneAssigner) ForgetItem(ref, item string) error {
	goalName, milestoneName, err := parseMilestoneRef(ref)
	if err != nil {
		return err
	}
	_, err = a.goals.UnassignMilestoneItems(goalName, milestoneName, []string{item})
	return err
}
