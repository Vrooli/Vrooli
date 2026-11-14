package goals

import (
	"context"
	"strings"
	"time"
)

type (
	Service interface {
		List(context.Context) ([]Goal, error)
		Create(context.Context, CreateInput) (Goal, error)
		SetTargetDate(context.Context, string, string, string, int64) error
		ListTargetDateHistory(context.Context, string) ([]TargetDateHistory, error)
		UpdateProgress(context.Context, string, int64, int64) (Goal, error)
		ListMilestones(context.Context, string) ([]Milestone, error)
		CreateMilestone(context.Context, CreateMilestoneInput) (Milestone, error)
		UpdateMilestoneStatus(context.Context, string, string, int64) (Milestone, error)
	}
	service struct{ repo Repository }
)

func NewService(repo Repository) Service                  { return &service{repo: repo} }
func (s *service) List(c context.Context) ([]Goal, error) { return s.repo.List(c) }

func (s *service) SetTargetDate(c context.Context, id, targetDate, reason string, expectedRevision int64) error {
	if strings.TrimSpace(id) == "" {
		return ErrInvalidGoal{"id", "required"}
	}
	if expectedRevision < 1 {
		return ErrInvalidGoal{"revision", "must be positive"}
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return ErrInvalidGoal{"reason", "required"}
	}
	targetDate = strings.TrimSpace(targetDate)
	if targetDate != "" {
		if _, err := time.Parse("2006-01-02", targetDate); err != nil {
			return ErrInvalidGoal{"target_date", "must be YYYY-MM-DD or empty"}
		}
	}
	return s.repo.SetTargetDate(c, id, targetDate, reason, expectedRevision)
}

func (s *service) ListTargetDateHistory(c context.Context, id string) ([]TargetDateHistory, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, ErrInvalidGoal{"id", "required"}
	}
	return s.repo.ListTargetDateHistory(c, id)
}
func (s *service) Create(c context.Context, in CreateInput) (Goal, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return Goal{}, ErrInvalidGoal{"title", "required"}
	}
	method := strings.TrimSpace(in.ProgressMethod)
	if method == "" {
		method = ProgressManual
	}
	if method != ProgressManual && method != ProgressMilestones {
		return Goal{}, ErrInvalidGoal{"progress_method", "must be manual or milestones"}
	}
	if in.TargetBasisPoints <= 0 || in.TargetBasisPoints > 10000 {
		return Goal{}, ErrInvalidGoal{"target_basis_points", "must be between 1 and 10000"}
	}
	targetDate := strings.TrimSpace(in.TargetDate)
	if targetDate != "" {
		if _, err := time.Parse("2006-01-02", targetDate); err != nil {
			return Goal{}, ErrInvalidGoal{"target_date", "must be YYYY-MM-DD"}
		}
	}
	measure, err := validateMeasure(in.Measure, targetDate)
	if err != nil {
		return Goal{}, err
	}
	return s.repo.Create(c, Goal{Title: title, Purpose: strings.TrimSpace(in.Purpose), Status: StatusActive, ProgressMethod: method, TargetBasisPoints: in.TargetBasisPoints, TargetDate: targetDate, Measure: measure, Revision: 1})
}

func validateMeasure(input *Measure, targetDate string) (*Measure, error) {
	if input == nil {
		return nil, nil
	}
	measure := *input
	measure.Unit = strings.TrimSpace(strings.ToLower(measure.Unit))
	measure.CurrencyCode = strings.TrimSpace(strings.ToUpper(measure.CurrencyCode))
	measure.Basis = strings.TrimSpace(strings.ToLower(measure.Basis))
	measure.Recurrence = strings.TrimSpace(strings.ToLower(measure.Recurrence))
	measure.Period = strings.TrimSpace(strings.ToLower(measure.Period))
	if measure.Unit != "currency" {
		return nil, ErrInvalidGoal{"measure.unit", "must be currency"}
	}
	if measure.TargetAmountMinor <= 0 {
		return nil, ErrInvalidGoal{"measure.target_amount_minor", "must be positive"}
	}
	if measure.CurrencyCode != "USD" {
		return nil, ErrInvalidGoal{"measure.currency_code", "currently supports USD currency targets"}
	}
	if measure.Basis != "after_costs_before_tax" {
		return nil, ErrInvalidGoal{"measure.basis", "must be after_costs_before_tax"}
	}
	if targetDate == "" {
		return nil, ErrInvalidGoal{"target_date", "required for measured goals"}
	}
	switch {
	case measure.Recurrence == "one_time" && measure.Period == "once":
	case measure.Recurrence == "recurring" && measure.Period == "month":
	default:
		return nil, ErrInvalidGoal{"measure.period", "one_time requires once; recurring requires month"}
	}
	return &measure, nil
}

func (s *service) ListMilestones(c context.Context, goalID string) ([]Milestone, error) {
	if strings.TrimSpace(goalID) == "" {
		return nil, ErrInvalidGoal{"goal_id", "required"}
	}
	return s.repo.ListMilestones(c, goalID)
}

func (s *service) CreateMilestone(c context.Context, in CreateMilestoneInput) (Milestone, error) {
	goalID := strings.TrimSpace(in.GoalID)
	if goalID == "" {
		return Milestone{}, ErrInvalidGoal{"goal_id", "required"}
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return Milestone{}, ErrInvalidGoal{"title", "required"}
	}
	dueDate := strings.TrimSpace(in.DueDate)
	if dueDate != "" {
		if _, err := time.Parse("2006-01-02", dueDate); err != nil {
			return Milestone{}, ErrInvalidGoal{"due_date", "must be YYYY-MM-DD"}
		}
	}
	linkedWorkItemID := strings.TrimSpace(in.LinkedWorkItemID)
	if linkedWorkItemID != "" {
		exists, err := s.repo.WorkItemExists(c, linkedWorkItemID)
		if err != nil {
			return Milestone{}, err
		}
		if !exists {
			return Milestone{}, ErrInvalidGoal{"linked_work_item_id", "work item not found"}
		}
	}
	prerequisites := uniqueIDs(in.PrerequisiteMilestoneIDs)
	for _, id := range prerequisites {
		if id == "" {
			return Milestone{}, ErrInvalidGoal{"prerequisite_milestone_ids", "ids must not be empty"}
		}
	}
	if len(prerequisites) > 0 {
		exists, err := s.repo.MilestonesExistForGoal(c, goalID, prerequisites)
		if err != nil {
			return Milestone{}, err
		}
		if !exists {
			return Milestone{}, ErrInvalidGoal{"prerequisite_milestone_ids", "all prerequisites must belong to the goal"}
		}
	}
	return s.repo.CreateMilestone(c, Milestone{GoalID: goalID, Title: title, Criteria: strings.TrimSpace(in.Criteria), DueDate: dueDate, LinkedWorkItemID: linkedWorkItemID, PrerequisiteMilestoneIDs: prerequisites, Status: MilestoneOpen, Revision: 1})
}

func (s *service) UpdateMilestoneStatus(c context.Context, id, status string, revision int64) (Milestone, error) {
	if strings.TrimSpace(id) == "" {
		return Milestone{}, ErrInvalidGoal{"id", "required"}
	}
	status = strings.TrimSpace(status)
	if status != MilestoneOpen && status != MilestoneDone {
		return Milestone{}, ErrInvalidGoal{"status", "must be open or complete"}
	}
	return s.repo.UpdateMilestoneStatus(c, id, status, revision)
}

func (s *service) UpdateProgress(c context.Context, id string, p, r int64) (Goal, error) {
	if id == "" {
		return Goal{}, ErrInvalidGoal{"id", "required"}
	}
	if p < 0 || p > 10000 {
		return Goal{}, ErrInvalidGoal{"progress_basis_points", "must be between 0 and 10000"}
	}
	return s.repo.UpdateProgress(c, id, p, r)
}

func uniqueIDs(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
