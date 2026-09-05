package orchestration

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"
)

type clockTaskRepository struct {
	repository.TaskRepository
	captured *domain.Task
}

func (r *clockTaskRepository) Create(_ context.Context, task *domain.Task) error {
	copy := *task
	r.captured = &copy
	return nil
}

func TestOrchestratorClockControlsDurableCreationTimestamps(t *testing.T) {
	now := time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)
	tasks := &clockTaskRepository{}
	o := New(nil, tasks, nil, WithClock(func() time.Time { return now }))
	_, err := o.CreateTask(context.Background(), &domain.Task{Title: "clock", ScopePath: ".", ProjectRoot: "."})
	if err != nil {
		t.Fatal(err)
	}
	if tasks.captured == nil || !tasks.captured.CreatedAt.Equal(now) || !tasks.captured.UpdatedAt.Equal(now) {
		t.Fatalf("captured task timestamps = %+v, want %v", tasks.captured, now)
	}
}

func TestFilterFromPresetAtUsesExplicitReferenceTime(t *testing.T) {
	now := time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)
	filter := FilterFromPresetAt(TimePreset24H, now)
	if !filter.Window.End.Equal(now) || !filter.Window.Start.Equal(now.Add(-24*time.Hour)) {
		t.Fatalf("window = %+v, want reference time %v", filter.Window, now)
	}
}
