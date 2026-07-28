package forest

import "context"

type Repository interface {
	CreateSummary(context.Context, Summary, []Edge) (Summary, error)
	Frontier(context.Context) ([]Summary, error)
	Rebuild(context.Context) error
}
