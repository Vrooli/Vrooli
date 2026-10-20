package shots

import "context"

type Repository interface {
	Create(context.Context, Shot, Revision) error
	Save(context.Context, string, int64, Revision) (Shot, error)
	Get(context.Context, string) (Shot, error)
	List(context.Context, string, int, *Shot) ([]Shot, error)
	GetRevision(context.Context, string, int64) (Revision, error)
	ListRevisions(context.Context, string, int, int64) ([]Revision, error)
}
