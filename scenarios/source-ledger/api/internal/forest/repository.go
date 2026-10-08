package forest

import "context"

type Repository interface {
	CreateSummary(context.Context, Summary, []Edge) (Summary, error)
	Frontier(context.Context) ([]Summary, error)
	Nodes(context.Context, int) ([]Node, error)
	Rebuild(context.Context) error
}
