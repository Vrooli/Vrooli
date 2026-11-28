package documents

import "context"

type Repository interface {
	Create(context.Context, Binding) (Binding, error)
	List(context.Context, string) ([]Binding, error)
	Get(context.Context, string, string) (Binding, error)
	CreateRelease(context.Context, Release) (Release, error)
	GetRelease(context.Context, string, string) (Release, error)
}
