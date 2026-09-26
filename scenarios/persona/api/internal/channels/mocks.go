package channels

import "context"

type FakeRepository struct {
	CreateFunc func(context.Context, Channel) (Channel, error)
	ListFunc   func(context.Context, string) ([]Channel, error)
	GetFunc    func(context.Context, string) (Channel, error)
}

func (f FakeRepository) Create(ctx context.Context, c Channel) (Channel, error) {
	return f.CreateFunc(ctx, c)
}

func (f FakeRepository) List(ctx context.Context, id string) ([]Channel, error) {
	return f.ListFunc(ctx, id)
}

func (f FakeRepository) Get(ctx context.Context, id string) (Channel, error) {
	return f.GetFunc(ctx, id)
}
