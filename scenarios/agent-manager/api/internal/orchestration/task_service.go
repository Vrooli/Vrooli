package orchestration

import (
	"context"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// TaskService is the task-lifecycle boundary used by the HTTP surface.
// It deliberately contains only task operations, so callers cannot acquire
// unrelated run or workflow capabilities by depending on task handling.
type TaskService interface {
	CreateTask(context.Context, *domain.Task) (*domain.Task, error)
	GetTask(context.Context, uuid.UUID) (*domain.Task, error)
	ListTasks(context.Context, ListOptions) ([]*domain.Task, error)
	UpdateTask(context.Context, *domain.Task) (*domain.Task, error)
	CancelTask(context.Context, uuid.UUID) error
	DeleteTask(context.Context, uuid.UUID) error
}

var _ TaskService = (*Orchestrator)(nil)
