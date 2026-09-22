package work

import (
	"context"
	"strings"
	"time"
)

type Service interface {
	Create(context.Context, CreateInput) (WorkItem, error)
	Get(context.Context, string) (WorkItem, error)
	List(context.Context, int) ([]WorkItem, error)
	TodayPlan(context.Context) (TodayPlan, error)
	Snooze(context.Context, string, string, string) error
	UpdateEstimate(context.Context, string, int, string) error
	Complete(context.Context, string) error
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

func (s *service) Snooze(ctx context.Context, id, until, reason string) error {
	if strings.TrimSpace(id) == "" {
		return ErrInvalidWorkItem{"id", "required"}
	}
	if _, err := time.ParseInLocation("2006-01-02", until, time.Local); err != nil {
		return ErrInvalidWorkItem{"until", "must be YYYY-MM-DD"}
	}
	return s.repo.Snooze(ctx, id, until, strings.TrimSpace(reason))
}

func (s *service) Complete(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return ErrInvalidWorkItem{"id", "required"}
	}
	return s.repo.Complete(ctx, id)
}

func (s *service) UpdateEstimate(ctx context.Context, id string, minutes int, reason string) error {
	if strings.TrimSpace(id) == "" {
		return ErrInvalidWorkItem{"id", "required"}
	}
	if minutes < 0 {
		return ErrInvalidWorkItem{"remaining_minutes", "must be non-negative"}
	}
	reason = strings.TrimSpace(reason)
	if len([]rune(reason)) > 200 {
		return ErrInvalidWorkItem{"reason", "must be 200 characters or fewer"}
	}
	return s.repo.UpdateEstimate(ctx, id, minutes, reason)
}
