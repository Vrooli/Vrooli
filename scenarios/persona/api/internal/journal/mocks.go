package journal

import "context"

type FakeRepository struct {
	AppendFunc func(context.Context, Entry) (Entry, error)
	ListFunc   func(context.Context, string, int) ([]Entry, error)
}

func (f FakeRepository) Append(ctx context.Context, e Entry) (Entry, error) {
	return f.AppendFunc(ctx, e)
}

func (f FakeRepository) List(ctx context.Context, id string, limit int) ([]Entry, error) {
	return f.ListFunc(ctx, id, limit)
}
