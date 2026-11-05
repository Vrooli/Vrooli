package projects

import "context"

type Repository interface {
	Create(context.Context, Project) (Project, error)
	Get(context.Context, string) (Project, error)
	List(context.Context, int, *Project) ([]Project, error)
	Rename(context.Context, RenameInput, string, int64) (Project, error)
}
