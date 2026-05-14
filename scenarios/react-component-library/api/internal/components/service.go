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
	ListVersions(ctx context.Context, componentID string, limit int) ([]ComponentVersion, error)
	GetVersion(ctx context.Context, componentID, version string) (ComponentVersion, error)
	GetVersionContent(ctx context.Context, componentID, version string) (Content, error)
	UpdateContent(ctx context.Context, id string, in WriteContentInput) (Content, error)
}

// ContentChangeListener is an optional sink the service invokes after a
// successful UpdateContent write. Wired by main.go to drive the
// versions recorder (req 11). Listener errors are logged at the
// handler edge and do not roll back the save — the file on disk is
// already updated. Keep listeners idempotent and fast.
type ContentChangeListener interface {
	OnContentSaved(ctx context.Context, c Component, content Content) error
}

type service struct {
	repo     Repository
	content  ContentStore
	listener ContentChangeListener
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

// SetContentChangeListener installs the post-save listener. Designed
// to be called once at boot before any requests land; not concurrency-
// safe with in-flight UpdateContent.
func SetContentChangeListener(svc Service, l ContentChangeListener) {
	if s, ok := svc.(*service); ok {
		s.listener = l
	}
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

func (s *service) ListVersions(ctx context.Context, componentID string, limit int) ([]ComponentVersion, error) {
	return s.repo.ListVersions(ctx, componentID, limit)
}

func (s *service) GetVersion(ctx context.Context, componentID, version string) (ComponentVersion, error) {
	return s.repo.GetVersion(ctx, componentID, version)
}

func (s *service) GetVersionContent(ctx context.Context, componentID, version string) (Content, error) {
	v, err := s.repo.GetVersion(ctx, componentID, version)
	if err != nil {
		return Content{}, err
	}
	return Content{Body: v.Content, SourcePath: v.SourcePath, SHA256: v.ContentSHA256}, nil
}

func (s *service) UpdateContent(ctx context.Context, id string, in WriteContentInput) (Content, error) {
	if s.content == nil {
		return Content{}, errNoContentStore
	}
	c, err := s.repo.Get(ctx, id)
	if err != nil {
		return Content{}, err
	}
	written, err := s.content.Write(ctx, c, in)
	if err != nil {
		return Content{}, err
	}
	if s.listener != nil {
		// Listener errors are non-fatal — the file is already written.
		// Callers swallow + log at the handler edge.
		_ = s.listener.OnContentSaved(ctx, c, written)
	}
	return written, nil
}

// errNoContentStore signals that the service was constructed without a
// ContentStore. Surfaces as Internal at the transport edge — it's a
// wiring bug, not a caller error.
var errNoContentStore = contentStoreUnconfiguredError{}

type contentStoreUnconfiguredError struct{}

func (contentStoreUnconfiguredError) Error() string {
	return "components service: ContentStore not configured"
}
