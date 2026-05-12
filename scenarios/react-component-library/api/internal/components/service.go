package components

import (
	"context"
	"strings"
)

// defaultListLimit caps List rows when callers pass 0. Business
// policy, not transport policy — lives next to the only code that
// applies it.
const defaultListLimit = 200

// Service is the application-layer surface handlers and the indexer
// depend on. Owns validation, default substitution, and any cross-
// handler policy that doesn't belong in transport.
type Service interface {
	Upsert(ctx context.Context, in UpsertInput) (Component, error)
	Get(ctx context.Context, id string) (Component, error)
	GetByLibraryID(ctx context.Context, libraryID string) (Component, error)
	List(ctx context.Context, q SearchQuery) ([]Component, error)
	GetContent(ctx context.Context, id string) (Content, error)
	UpdateContent(ctx context.Context, id string, in WriteContentInput) (Content, error)
}

type service struct {
	repo    Repository
	content ContentStore
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// NewServiceWithContent wires the optional ContentStore seam. Read
// paths work without it; GetContent/UpdateContent return an error when
// content is nil. Production constructs the store from the configured
// source root; tests inject a fake.
func NewServiceWithContent(repo Repository, content ContentStore) Service {
	return &service{repo: repo, content: content}
}

var _ Service = (*service)(nil)

func (s *service) Upsert(ctx context.Context, in UpsertInput) (Component, error) {
	in.LibraryID = strings.TrimSpace(in.LibraryID)
	if in.LibraryID == "" {
		return Component{}, ErrInvalidHeader{SourcePath: in.SourcePath, Field: "libraryId", Reason: "required"}
	}
	return s.repo.Upsert(ctx, in)
}

func (s *service) Get(ctx context.Context, id string) (Component, error) {
	return s.repo.Get(ctx, id)
}

func (s *service) GetByLibraryID(ctx context.Context, libraryID string) (Component, error) {
	return s.repo.GetByLibraryID(ctx, libraryID)
}

func (s *service) List(ctx context.Context, q SearchQuery) ([]Component, error) {
	if q.Limit <= 0 {
		q.Limit = defaultListLimit
	}
	return s.repo.List(ctx, q)
}

func (s *service) GetContent(ctx context.Context, id string) (Content, error) {
	if s.content == nil {
		return Content{}, errNoContentStore
	}
	c, err := s.repo.Get(ctx, id)
	if err != nil {
		return Content{}, err
	}
	return s.content.Read(ctx, c)
}

func (s *service) UpdateContent(ctx context.Context, id string, in WriteContentInput) (Content, error) {
	if s.content == nil {
		return Content{}, errNoContentStore
	}
	c, err := s.repo.Get(ctx, id)
	if err != nil {
		return Content{}, err
	}
	return s.content.Write(ctx, c, in)
}

// errNoContentStore signals that the service was constructed without a
// ContentStore. Surfaces as Internal at the transport edge — it's a
// wiring bug, not a caller error.
var errNoContentStore = contentStoreUnconfiguredError{}

type contentStoreUnconfiguredError struct{}

func (contentStoreUnconfiguredError) Error() string {
	return "components service: ContentStore not configured"
}
