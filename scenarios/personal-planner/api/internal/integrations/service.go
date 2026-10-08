package integrations

import (
	"context"
	"strings"
)

type Service interface {
	List(context.Context) ([]Connection, error)
	CreateFixture(context.Context, string) (Connection, error)
	Sync(context.Context, SyncInput) (Connection, error)
	Disconnect(context.Context, SyncInput) (Connection, error)
}

type service struct{ repo Repository }

func NewService(repo Repository) Service { return &service{repo: repo} }

func (s *service) List(ctx context.Context) ([]Connection, error) { return s.repo.List(ctx) }

func (s *service) CreateFixture(ctx context.Context, name string) (Connection, error) {
	if strings.TrimSpace(name) == "" {
		name = "Observatory sample calendar"
	}
	return s.repo.CreateFixture(ctx, strings.TrimSpace(name))
}

func (s *service) Sync(ctx context.Context, in SyncInput) (Connection, error) {
	if strings.TrimSpace(in.ID) == "" {
		return Connection{}, ErrInvalid{"id", "required"}
	}
	if in.ExpectedRevision <= 0 {
		return Connection{}, ErrInvalid{"expected_revision", "must be positive"}
	}
	return s.repo.Sync(ctx, in)
}

func (s *service) Disconnect(ctx context.Context, in SyncInput) (Connection, error) {
	if strings.TrimSpace(in.ID) == "" {
		return Connection{}, ErrInvalid{"id", "required"}
	}
	if in.ExpectedRevision <= 0 {
		return Connection{}, ErrInvalid{"expected_revision", "must be positive"}
	}
	return s.repo.Disconnect(ctx, in)
}
