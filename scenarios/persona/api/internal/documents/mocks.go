package documents

import "context"

type FakeRepository struct {
	CreateFunc        func(context.Context, Binding) (Binding, error)
	ListFunc          func(context.Context, string) ([]Binding, error)
	GetFunc           func(context.Context, string, string) (Binding, error)
	CreateReleaseFunc func(context.Context, Release) (Release, error)
	GetReleaseFunc    func(context.Context, string, string) (Release, error)
}

func (f FakeRepository) Create(ctx context.Context, b Binding) (Binding, error) {
	return f.CreateFunc(ctx, b)
}

func (f FakeRepository) List(ctx context.Context, id string) ([]Binding, error) {
	return f.ListFunc(ctx, id)
}

func (f FakeRepository) Get(ctx context.Context, personaID, documentID string) (Binding, error) {
	return f.GetFunc(ctx, personaID, documentID)
}

func (f FakeRepository) CreateRelease(ctx context.Context, release Release) (Release, error) {
	if f.CreateReleaseFunc == nil {
		return release, nil
	}
	return f.CreateReleaseFunc(ctx, release)
}

func (f FakeRepository) GetRelease(ctx context.Context, handoffID, documentID string) (Release, error) {
	if f.GetReleaseFunc == nil {
		return Release{}, ErrReleaseNotFound
	}
	return f.GetReleaseFunc(ctx, handoffID, documentID)
}

type FakeAuthority struct {
	CheckFunc   func(context.Context) error
	ReleaseFunc func(context.Context, string, string) (string, error)
}

func (f FakeAuthority) Check(ctx context.Context) error { return f.CheckFunc(ctx) }
func (f FakeAuthority) Release(ctx context.Context, documentID, handoffID string) (string, error) {
	return f.ReleaseFunc(ctx, documentID, handoffID)
}
