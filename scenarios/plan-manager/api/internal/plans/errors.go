package plans

import (
	"fmt"
	"strings"

	"plan-manager/internal/planmodel"
)

// ErrPlanNotFound is returned when no plan matches an id or slug.
type ErrPlanNotFound = planmodel.ErrPlanNotFound

// ErrInvalidPlan is returned when a plan fails structural validation
// (e.g. empty title/slug) at the service boundary.
type ErrInvalidPlan = planmodel.ErrInvalidPlan

// ErrPhaseNotFound is returned when a phase id is not on the named plan.
type ErrPhaseNotFound = planmodel.ErrPhaseNotFound

// ErrPhaseTransition reports a lifecycle edge that is absent from the phase
// transition table.
type ErrPhaseTransition struct {
	PlanID  string
	PhaseID string
	From    PhaseStatus
	To      PhaseStatus
}

func (e ErrPhaseTransition) Error() string {
	return fmt.Sprintf("phase %q on plan %q cannot transition from %q to %q", e.PhaseID, e.PlanID, e.From, e.To)
}

// ErrTemplateNotFound is returned when a template id is unknown.
type ErrTemplateNotFound struct{ ID string }

func (e ErrTemplateNotFound) Error() string { return fmt.Sprintf("template %q not found", e.ID) }

type ErrCandidateNotFound struct{ ID string }

func (e ErrCandidateNotFound) Error() string {
	return fmt.Sprintf("candidate revision %q not found", e.ID)
}

type ErrCandidateState struct {
	ID    string
	State CandidateRevisionState
}

func (e ErrCandidateState) Error() string {
	return fmt.Sprintf("candidate revision %q is %s", e.ID, e.State)
}

type ErrCandidateStaleBase struct {
	PlanID   string
	Expected string
	Actual   string
}

func (e ErrCandidateStaleBase) Error() string {
	return fmt.Sprintf("candidate plan %q is stale: expected base hash %q, found %q", e.PlanID, e.Expected, e.Actual)
}

type ErrCandidateExecutionActive struct {
	PlanID       string
	ExecutionIDs []string
}

func (e ErrCandidateExecutionActive) Error() string {
	return fmt.Sprintf("candidate plan %q cannot apply while execution(s) are active: %s", e.PlanID, strings.Join(e.ExecutionIDs, ", "))
}
