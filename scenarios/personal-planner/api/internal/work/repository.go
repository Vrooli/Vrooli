package work

import "context"

type Repository interface {
	Create(context.Context, WorkItem) (WorkItem, error)
	Get(context.Context, string) (WorkItem, error)
	List(context.Context, int) ([]WorkItem, error)
	Snooze(context.Context, string, string, string) error
	UpdateEstimate(context.Context, string, int, string) error
	Complete(context.Context, string) error
}
