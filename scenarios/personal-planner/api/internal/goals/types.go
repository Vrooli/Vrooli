package goals

import "fmt"

const (
	StatusActive       = "active"
	ProgressManual     = "manual"
	ProgressMilestones = "milestones"
	MilestoneOpen      = "open"
	MilestoneDone      = "complete"
)

type (
	Measure struct {
		Unit              string `json:"unit"`
		TargetAmountMinor int64  `json:"target_amount_minor"`
		CurrencyCode      string `json:"currency_code"`
		Basis             string `json:"basis"`
		Recurrence        string `json:"recurrence"`
		Period            string `json:"period"`
	}
	Goal struct {
		ID, Title, Purpose, Status, ProgressMethod       string
		ProgressBasisPoints, TargetBasisPoints, Revision int64
		TargetDate                                       string
		Measure                                          *Measure
	}
	TargetDateHistory struct {
		GoalID             string `json:"goal_id"`
		PreviousTargetDate string `json:"previous_target_date"`
		TargetDate         string `json:"target_date"`
		Reason             string `json:"reason"`
		ChangedAt          string `json:"changed_at"`
		Revision           int64  `json:"revision"`
	}
	CreateInput struct {
		Title, Purpose, ProgressMethod string
		TargetBasisPoints              int64
		TargetDate                     string
		Measure                        *Measure
	}
	ErrInvalidGoal struct{ Field, Reason string }
	Milestone      struct {
		ID, GoalID, Title, Criteria, DueDate, Status, LinkedWorkItemID string
		PrerequisiteMilestoneIDs                                       []string
		Revision                                                       int64
	}
	CreateMilestoneInput struct {
		GoalID, Title, Criteria, DueDate, LinkedWorkItemID string
		PrerequisiteMilestoneIDs                           []string
	}
)

func (e ErrInvalidGoal) Error() string { return fmt.Sprintf("%s: %s", e.Field, e.Reason) }

type ErrGoalNotFound struct{ ID string }

func (e ErrGoalNotFound) Error() string { return fmt.Sprintf("goal %q not found", e.ID) }

type ErrRevisionConflict struct{ ID string }

func (e ErrRevisionConflict) Error() string {
	return fmt.Sprintf("goal %q changed; reload before updating", e.ID)
}

type ErrMilestoneNotFound struct{ ID string }

func (e ErrMilestoneNotFound) Error() string { return fmt.Sprintf("milestone %q not found", e.ID) }

type ErrUnverifiedGoalPrincipal struct{}

func (ErrUnverifiedGoalPrincipal) Error() string { return "verified goal principal required" }

type ErrPrerequisitesIncomplete struct{ ID string }

func (e ErrPrerequisitesIncomplete) Error() string {
	return fmt.Sprintf("milestone %q has incomplete prerequisites", e.ID)
}
