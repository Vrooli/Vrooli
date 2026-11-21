package snapshot

import "context"

type Repository interface {
	Create(ctx context.Context, s Snapshot) (Snapshot, error)
	List(ctx context.Context) ([]Snapshot, error)
	Get(ctx context.Context, id string) (Snapshot, error)
	Count(ctx context.Context) (int, error)
}
