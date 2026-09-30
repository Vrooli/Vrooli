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

// goalProjection tracks the effect of earlier mutations in the same list so a
// single envelope can create a milestone and then populate it. Without this,
// structure and membership must be split across two operator decisions, and
// the intermediate state is a milestone with no members — exactly the
// half-built shape the acceptance-criteria rule exists to prevent.
type goalProjection struct {
	created  map[string]struct{}
	archived map[string]struct{}
	targets  map[string]struct{}
	items    map[string]int
}

func newGoalProjection() *goalProjection {
	return &goalProjection{created: map[string]struct{}{}, archived: map[string]struct{}{}, targets: map[string]struct{}{}, items: map[string]int{}}
}

// milestoneExists reports whether the named milestone exists at this point in
// the list: either it was already on the goal, or an earlier mutation created
// it. A milestone archived earlier in the same list no longer exists.
func (g *goalProjection) milestoneExists(name string, state GoalState) bool {
	if _, archived := g.archived[name]; archived {
		return false
	}
	if _, created := g.created[name]; created {
		return true
	}
	_, existing := state.Milestones[name]
	return existing
}

// targetExists reports whether the ref is a goal target at this point in the
// list, counting targets added by an earlier mutation.
func (g *goalProjection) targetExists(ref string, state GoalState) bool {
	if _, added := g.targets[ref]; added {
		return true
	}
	_, existing := state.Targets[ref]
	return existing
}

// inClosure reports whether the ref is assignable at this point in the list.
// A target added earlier in the same list is in scope by definition: the goal's
// closure is derived from its targets.
func (g *goalProjection) inClosure(ref string, state GoalState) bool {
	if _, added := g.targets[ref]; added {
		return true
	}
	_, existing := state.Closure[ref]
	return existing
}

// itemExists reports whether the ref names an item the goal can already see,
// counting items created earlier in the same list.
func (g *goalProjection) itemExists(ref string, state GoalState) bool {
	if _, staged := g.items[ref]; staged {
		return true
	}
	if _, inClosure := state.Closure[ref]; inClosure {
		return true
	}
	_, isTarget := state.Targets[ref]
	return isTarget
}

// ValidateGoal validates a mutation list intended for a goal target.
//
// A goal proposal may shape the goal graph and create the work that graph
// needs: add_item is admitted because goal planning, discovery, and milestone
// review all exist to find work that has no covering item yet. Every other
// item operation stays with the item path, where that item's own context is
// hydrated — a goal proposal must not silently retarget, archive, or change
// the status of work it is not showing the operator.
//
// Mutations are validated in order against a running projection of the list's
// own effects, so a create-then-populate sequence is valid in one envelope.
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
	projection := newGoalProjection()
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
		if err := validateGoalMutation(m, state, projection); err != nil {
			problems = append(problems, fmt.Errorf("%s: %w", prefix, err))
		}
	}
	if len(problems) == 0 {
		return nil
	}
	return errors.Join(append([]error{ErrInvalidProposal}, problems...)...)
}

func validateGoalMutation(m Mutation, state GoalState, projection *goalProjection) error {
	milestone := strings.TrimSpace(m.MilestoneName)
	switch m.Op {
	case OpCreateMilestone:
		if m.GoalMilestone == nil || strings.TrimSpace(m.GoalMilestone.Name) == "" || strings.TrimSpace(m.GoalMilestone.Title) == "" {
			return fmt.Errorf("create_milestone requires goal_milestone name and title")
		}
		if projection.milestoneExists(m.GoalMilestone.Name, state) {
			return fmt.Errorf("milestone %q already exists", m.GoalMilestone.Name)
		}
		if !hasAcceptanceCriteria(m.GoalMilestone) {
			return fmt.Errorf("create_milestone requires acceptance_criteria: a milestone without a definition of done can never be reviewed or closed out")
		}
		projection.created[m.GoalMilestone.Name] = struct{}{}
		delete(projection.archived, m.GoalMilestone.Name)
	case OpUpdateMilestone:
		if m.GoalMilestone == nil || strings.TrimSpace(m.GoalMilestone.Name) == "" {
			return fmt.Errorf("update_milestone requires goal_milestone name")
		}
		if !projection.milestoneExists(m.GoalMilestone.Name, state) {
			return fmt.Errorf("unknown milestone %q", m.GoalMilestone.Name)
		}
		// update_milestone replaces the milestone definition wholesale, so an
		// omitted criteria list erases the definition of done rather than
		// leaving it untouched. Require the full intended set every time.
		if !hasAcceptanceCriteria(m.GoalMilestone) {
			return fmt.Errorf("update_milestone requires acceptance_criteria: the update replaces the milestone definition, so omitting them erases the definition of done")
		}
	case OpArchiveMilestone:
		if milestone == "" || !projection.milestoneExists(milestone, state) {
			return fmt.Errorf("archive_milestone requires an existing milestone_name")
		}
		if current, existing := state.Milestones[milestone]; existing && current.Open && !m.DetachOpen {
			return fmt.Errorf("milestone %q has open member items; set detach_open explicitly", milestone)
		}
		projection.archived[milestone] = struct{}{}
		delete(projection.created, milestone)
	case OpAssignMilestoneItems, OpUnassignMilestoneItems:
		if milestone == "" || !projection.milestoneExists(milestone, state) {
			return fmt.Errorf("%s requires an existing milestone_name", m.Op)
		}
		if len(m.Items) == 0 {
			return fmt.Errorf("%s requires items", m.Op)
		}
		for _, item := range m.Items {
			if !projection.inClosure(item, state) {
				return fmt.Errorf("item %q is outside goal closure", item)
			}
		}
	case OpAddItem:
		// Goal planning discovers work that has no covering item yet. Without
		// this op a goal workflow could only reference items someone else had
		// already written down, so intent could never grow its own work.
		if m.Item == nil {
			return fmt.Errorf("add_item requires item spec")
		}
		if err := validateItemSpec(*m.Item); err != nil {
			return err
		}
		ref := m.Item.Ref()
		if projection.itemExists(ref, state) {
			return fmt.Errorf("%w: %s", ErrDuplicateItem, ref)
		}
		for _, dependency := range m.Item.DependsOn {
			if dependency == ref {
				return fmt.Errorf("depends_on must not reference self: %s", ref)
			}
		}
		projection.items[ref] = 1
	case OpAddGoalTarget:
		if len(m.Targets) == 0 {
			return fmt.Errorf("add_goal_target requires targets")
		}
		for _, target := range m.Targets {
			projection.targets[target] = struct{}{}
		}
	case OpRemoveGoalTarget:
		if len(m.Targets) == 0 {
			return fmt.Errorf("remove_goal_target requires targets")
		}
		for _, target := range m.Targets {
			if !projection.targetExists(target, state) {
				return fmt.Errorf("target %q is not a current goal target", target)
			}
			delete(projection.targets, target)
		}
	default:
		return fmt.Errorf("op %q is not valid for a goal proposal; supported: %s", m.Op, goalOpsList())
	}
	return nil
}

// goalOpsList renders GoalOps for an error message so the rejection names the
// vocabulary the caller should have used.
func goalOpsList() string {
	names := make([]string, 0, len(GoalOps()))
	for _, op := range GoalOps() {
		names = append(names, string(op))
	}
	return strings.Join(names, ", ")
}

// hasAcceptanceCriteria reports whether the spec carries at least one
// non-blank criterion. A milestone is the only place a goal's definition of
// done can live: milestone review reads it, and close-out is gated on the
// review verdict. A criteria-less milestone is therefore permanently
// unverifiable, so the proposal layer refuses to create one.
func hasAcceptanceCriteria(spec *GoalMilestone) bool {
	for _, criterion := range spec.AcceptanceCriteria {
		if strings.TrimSpace(criterion) != "" {
			return true
		}
	}
	return false
}
