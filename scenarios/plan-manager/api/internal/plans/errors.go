package plans

import "fmt"

// ErrPlanNotFound is returned when no plan matches an id or slug.
type ErrPlanNotFound struct{ ID string }

func (e ErrPlanNotFound) Error() string { return fmt.Sprintf("plan %q not found", e.ID) }

// ErrInvalidPlan is returned when a plan fails structural validation
// (e.g. empty title/slug) at the service boundary.
type ErrInvalidPlan struct{ Reason string }

func (e ErrInvalidPlan) Error() string { return fmt.Sprintf("invalid plan: %s", e.Reason) }

// ErrPhaseNotFound is returned when a phase id is not on the named plan.
type ErrPhaseNotFound struct {
	PlanID  string
	PhaseID string
}

func (e ErrPhaseNotFound) Error() string {
	return fmt.Sprintf("phase %q not found on plan %q", e.PhaseID, e.PlanID)
}

// ErrTemplateNotFound is returned when a template id is unknown.
type ErrTemplateNotFound struct{ ID string }

func (e ErrTemplateNotFound) Error() string { return fmt.Sprintf("template %q not found", e.ID) }
