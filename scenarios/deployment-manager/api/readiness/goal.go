package readiness

import (
	"fmt"
	"strings"
	"time"
)

type GoalMilestone struct {
	Name               string   `json:"name"`
	Title              string   `json:"title"`
	Description        string   `json:"description"`
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
	findings := make(map[string]Finding, len(verdict.Findings))
	for _, finding := range verdict.Findings {
		findings[finding.ItemID] = finding
	}
	milestones := make([]GoalMilestone, 0, len(verdict.Findings))
	for _, item := range checklist.Items {
		finding, unresolved := findings[item.ID]
		if !unresolved {
			continue
		}
		acceptance := item.AcceptanceCriteria
		if acceptance == "" {
			acceptance = item.Acceptance.Sentence()
		}
		milestones = append(milestones, GoalMilestone{
			Name: item.ID, Title: item.Title,
			Description: fmt.Sprintf("review=%s/%s disposition=%s severity=%s producer=%s evidence=%s run=%s observed=%s freshness=%s detail=%s verification=%s remediation=%s:%s waiver_eligible=%t",
				scenario, commit, finding.Status, finding.Severity, finding.Signal.Source, finding.Signal.Reference,
				finding.Signal.RunID, finding.Signal.ObservedAt.UTC().Format(time.RFC3339), item.Freshness.Basis,
				finding.Signal.Detail, verificationRoute(item), item.Remediation.Skill, item.Remediation.Topic, item.Waiver.Eligible),
			AcceptanceCriteria: []string{acceptance},
		})
	}
	return GoalSpec{
		Name:        fmt.Sprintf("readiness/%s/%s", scenario, commit),
		Title:       fmt.Sprintf("Readiness review: %s at %s", scenario, commit),
		Description: fmt.Sprintf("Aggregated readiness verdict for %s at commit %s: approved=%t, findings=%d. Criteria registry: docs/scenario-qa/methods/readiness/.", scenario, commit, verdict.Approved, len(verdict.Findings)),
		Priority:    0, Scenario: scenario, Commit: commit, Trigger: strings.TrimSpace(trigger), ServesDeliverable: strings.TrimSpace(deliverable), Milestones: milestones,
	}, nil
}

func verificationRoute(item Item) string {
	if item.Producer != nil {
		return item.Producer.Binding
	}
	return "human_review:" + item.HumanReview.Kind
}
