package profile

import "context"

type (
	Service interface {
		Get(context.Context, string) (Profile, error)
		SaveDraft(context.Context, string, string) (Profile, error)
		Apply(context.Context, ApplyInput) (Profile, error)
	}
	service struct{ repo Repository }
)

func NewService(r Repository) Service                                  { return &service{r} }
func (s *service) Get(ctx context.Context, id string) (Profile, error) { return s.repo.Get(ctx, id) }
func (s *service) SaveDraft(ctx context.Context, id, draft string) (Profile, error) {
	return s.repo.SaveDraft(ctx, id, draft)
}

func (s *service) Apply(ctx context.Context, in ApplyInput) (Profile, error) {
	return s.repo.Apply(ctx, in)
}
