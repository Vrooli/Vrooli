package work

import (
	"context"
	"strings"
)

type Service interface {
	Create(context.Context, CreateInput) (WorkItem, error)
	Get(context.Context, string) (WorkItem, error)
	List(context.Context, int) ([]WorkItem, error)
	TodayPlan(context.Context) (TodayPlan, error)
}

type service struct{ repo Repository }

func NewService(repo Repository) Service { return &service{repo: repo} }

func (s *service) Create(ctx context.Context, in CreateInput) (WorkItem, error) {
	if strings.TrimSpace(in.Title) == "" {
		return WorkItem{}, ErrInvalidWorkItem{"title", "required"}
	}
	if in.RemainingMinutes < 0 {
		return WorkItem{}, ErrInvalidWorkItem{"remaining_minutes", "must be non-negative"}
	}
	return s.repo.Create(ctx, WorkItem{Title: strings.TrimSpace(in.Title), Description: in.Description, RemainingMinutes: in.RemainingMinutes, SourceLabel: in.SourceLabel})
}
func (s *service) Get(ctx context.Context, id string) (WorkItem, error) { return s.repo.Get(ctx, id) }
func (s *service) List(ctx context.Context, limit int) ([]WorkItem, error) {
	if limit <= 0 {
		limit = 100
	}
	return s.repo.List(ctx, limit)
}

func (s *service) TodayPlan(ctx context.Context) (TodayPlan, error) {
	items, err := s.List(ctx, 100)
	if err != nil {
		return TodayPlan{}, err
	}
	plan := TodayPlan{AvailableMinutes: 480, Entries: make([]TodayPlanEntry, 0, len(items))}
	minute := 9 * 60
	for _, item := range items {
		duration := item.RemainingMinutes
		if duration <= 0 {
			continue
		}
		if duration > 120 {
			duration = 120
		}
		plan.Entries = append(plan.Entries, TodayPlanEntry{WorkItemID: item.ID, Title: item.Title, SourceLabel: item.SourceLabel, StartMinutes: minute, DurationMinutes: duration})
		plan.PlannedMinutes += duration
		minute += duration + 15
	}
	plan.BreathingRoom = plan.AvailableMinutes - plan.PlannedMinutes
	if plan.BreathingRoom < 0 {
		plan.BreathingRoom = 0
	}
	return plan, nil
}
