package work

import "context"

type Repository interface {
	Create(context.Context, WorkItem) (WorkItem, error)
	Get(context.Context, string) (WorkItem, error)
	List(context.Context, int) ([]WorkItem, error)
}
