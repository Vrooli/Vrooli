package notes

import (
	"context"
	"strings"
	"time"
)

// defaultListLimit caps the rows returned by Service.List when the
// caller passes 0. Lives here (next to the only code that interprets
// it) rather than in the transport layer — this is business policy,
// not transport policy. Future scenarios that need configurable limits
// add a WithDefaultLimit option to NewService.
const defaultListLimit = 100

// Service is the application-layer surface the notes handlers depend
// on. Owns validation, default substitution, and any cross-handler
// policy that doesn't belong in transport. The handler is intentionally
// thin around it: decode → call service → translate errors.
type Service interface {
	// Create validates in (Title required after whitespace trim) and
	// delegates persistence to the underlying Repository. Returns
	// ErrInvalidNote on validation failure; otherwise propagates any
	// repository error.
	Create(ctx context.Context, in CreateInput) (Note, error)

	// Get is a thin pass-through to Repository.Get. ErrNoteNotFound
	// propagates verbatim — the handler does the errors.As translation.
	Get(ctx context.Context, id string) (Note, error)

	// List substitutes defaultListLimit when limit <= 0; otherwise
	// passes the caller's limit through unchanged.
	List(ctx context.Context, limit int) ([]Note, error)

	// CountInWindow returns how many notes were created in the half-open
	// range [from, to). It is the application-layer entry point the
	// `notes count` measure computes against (both the CountNotes RPC and
	// the measures-go serve registry call through here, so the measure and
	// the RPC can never diverge).
	CountInWindow(ctx context.Context, from, to time.Time) (int, error)
}

type service struct {
	repo Repository
}

// NewService constructs the production Service. Repository is the only
// dependency today; future seams (audit log, cache, webhook) join here
// as additional Deps fields without changing the Service interface.
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// Compile-time guarantee.
var _ Service = (*service)(nil)

func (s *service) Create(ctx context.Context, in CreateInput) (Note, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return Note{}, ErrInvalidNote{Field: "title", Reason: "required"}
	}
	return s.repo.Create(ctx, Note{Title: title, Body: in.Body})
}

func (s *service) Get(ctx context.Context, id string) (Note, error) {
	return s.repo.Get(ctx, id)
}

func (s *service) List(ctx context.Context, limit int) ([]Note, error) {
	if limit <= 0 {
		limit = defaultListLimit
	}
	return s.repo.List(ctx, limit)
}

func (s *service) CountInWindow(ctx context.Context, from, to time.Time) (int, error) {
	return s.repo.Count(ctx, from, to)
}
