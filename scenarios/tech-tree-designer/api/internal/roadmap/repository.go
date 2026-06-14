package roadmap

import "context"

type Repository interface {
	ListSectors(ctx context.Context) ([]Sector, error)
	UpsertSector(ctx context.Context, sector Sector) (Sector, error)
	ListMilestones(ctx context.Context) ([]Milestone, error)
	UpsertMilestone(ctx context.Context, milestone Milestone) (Milestone, error)
}
