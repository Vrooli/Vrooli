package facets

import "context"

type Repository interface {
	Seed(context.Context) error
	List(context.Context) ([]Definition, error)
	Validate(context.Context, string) error
	CompactionEligible(context.Context, string) (bool, error)
	SetPin(context.Context, string, bool) error
	Pinned(context.Context, string) (bool, error)
	Assign(context.Context, Assignment) (Assignment, error)
	Assignments(context.Context, string) ([]Assignment, error)
	MarkSuperseded(context.Context, string, string) error
}
