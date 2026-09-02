package readiness

import (
	"fmt"
	"strings"
)

type GoalMilestone struct {
	Name               string   `json:"name"`
	Title              string   `json:"title"`
	AcceptanceCriteria []string `json:"acceptance_criteria"`
}

type GoalSpec struct {
	Name              string          `json:"name"`
	Title             string          `json:"title"`
	Description       string          `json:"description"`
	Priority          int             `json:"priority"`
	Scenario          string          `json:"scenario"`
	Commit            string          `json:"commit"`
	Trigger           string          `json:"trigger,omitempty"`
	ServesDeliverable string          `json:"serves_deliverable,omitempty"`
	Milestones        []GoalMilestone `json:"milestones"`
}

// BuildGoalSpec converts the deployment-owned checklist into the existing
// swarm-manager goal shape. Commit identity is retained in deployment-manager
// fields and the deterministic goal name, not added to swarm-manager's model.
func BuildGoalSpec(scenario, commit, deliverable, trigger string, checklist Checklist, verdict Verdict) (GoalSpec, error) {
	if err := checklist.Validate(); err != nil {
		return GoalSpec{}, err
	}
	if strings.TrimSpace(scenario) == "" || strings.TrimSpace(commit) == "" {
		return GoalSpec{}, fmt.Errorf("scenario and commit are required")
	}
	if verdict.Scenario != scenario || verdict.Commit != commit {
		return GoalSpec{}, fmt.Errorf("verdict identity does not match scenario and commit")
	}
	milestones := make([]GoalMilestone, 0, len(checklist.Items))
	for _, item := range checklist.Items {
		milestones = append(milestones, GoalMilestone{
			Name: item.ID, Title: item.Title, AcceptanceCriteria: []string{item.AcceptanceCriteria},
		})
	}
	return GoalSpec{
		Name:        fmt.Sprintf("readiness/%s/%s", scenario, commit),
		Title:       fmt.Sprintf("Readiness review: %s at %s", scenario, commit),
		Description: fmt.Sprintf("Aggregated readiness verdict for %s at commit %s: approved=%t, findings=%d. Criteria registry: docs/scenario-qa/methods/readiness/.", scenario, commit, verdict.Approved, len(verdict.Findings)),
		Priority:    0, Scenario: scenario, Commit: commit, Trigger: strings.TrimSpace(trigger), ServesDeliverable: strings.TrimSpace(deliverable), Milestones: milestones,
	}, nil
}
