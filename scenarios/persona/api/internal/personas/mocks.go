package personas

import "context"

type FakeRepository struct {
	CreateFunc  func(context.Context, Persona) (Persona, error)
	GetFunc     func(context.Context, string) (Persona, error)
	ListFunc    func(context.Context, bool, int) ([]Persona, error)
	ArchiveFunc func(context.Context, string) (Persona, error)
}

func (f FakeRepository) Create(ctx context.Context, p Persona) (Persona, error) {
	return f.CreateFunc(ctx, p)
}

func (f FakeRepository) Get(ctx context.Context, id string) (Persona, error) {
	return f.GetFunc(ctx, id)
}

func (f FakeRepository) List(ctx context.Context, includeArchived bool, limit int) ([]Persona, error) {
	return f.ListFunc(ctx, includeArchived, limit)
}

func (f FakeRepository) Archive(ctx context.Context, id string) (Persona, error) {
	return f.ArchiveFunc(ctx, id)
}
