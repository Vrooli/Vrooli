package snippets

import (
	"context"
	"time"
)

type Store interface {
	List(ctx context.Context) ([]Snippet, error)
	Upsert(ctx context.Context, req UpsertRequest) (Snippet, error)
	Delete(ctx context.Context, id string) (bool, error)
	Touch(ctx context.Context, id string, now time.Time) (Snippet, error)
}
