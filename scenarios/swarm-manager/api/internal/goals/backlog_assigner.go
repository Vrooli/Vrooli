package goals

import (
	"errors"
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

// errStructureIsGoalOwned explains why the batch seam cannot define goal
// structure. The seam carries no acceptance criteria, so every milestone it
// could write would be permanently unverifiable, and its goal permanently
// un-closeable. Batch item creation attaches work to structure the operator
// already approved; it does not invent that structure.
const errStructureIsGoalOwned = "goal structure is not writable through backlog grouping: create the goal and its milestone (with acceptance criteria) through the goal API, then attach items to it"

// Create previously auto-created a goal named after the milestone whenever the
// referenced goal was absent, producing a goal titled with its own slug that
// wrapped a single identically-named, criteria-less milestone. That is the
// mechanism behind the mirrored goal/milestone pairs in the store, so the seam
// no longer writes structure at all.
func (a *BacklogMilestoneAssigner) Create(spec backlog.MilestoneSpec) error {
	if _, _, err := parseMilestoneRef(spec.Name); err != nil {
		return err
	}
	return errors.New(errStructureIsGoalOwned)
}

func (a *BacklogMilestoneAssigner) Update(spec backlog.MilestoneSpec) error {
	if _, _, err := parseMilestoneRef(spec.Name); err != nil {
		return err
	}
	return errors.New(errStructureIsGoalOwned)
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
