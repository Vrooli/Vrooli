package plans

import "fmt"

import "plan-manager/internal/planmodel"

// ErrPlanNotFound is returned when no plan matches an id or slug.
type ErrPlanNotFound = planmodel.ErrPlanNotFound

// ErrInvalidPlan is returned when a plan fails structural validation
// (e.g. empty title/slug) at the service boundary.
type ErrInvalidPlan = planmodel.ErrInvalidPlan

// ErrPhaseNotFound is returned when a phase id is not on the named plan.
type ErrPhaseNotFound = planmodel.ErrPhaseNotFound

// ErrTemplateNotFound is returned when a template id is unknown.
type ErrTemplateNotFound struct{ ID string }

func (e ErrTemplateNotFound) Error() string { return fmt.Sprintf("template %q not found", e.ID) }
