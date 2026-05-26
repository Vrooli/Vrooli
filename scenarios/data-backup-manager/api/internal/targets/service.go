package targets

import (
	"context"
	"errors"
	"strings"
)

// defaultListLimit caps List when the caller passes 0.
const defaultListLimit = 100

// Service is the application surface the targets handlers depend on. It owns
// validation and the idempotent-upsert decision (OT-P0-001): a re-register
// with an identical spec is a no-op (UpdatedAt unchanged); a changed spec
// updates in place; an absent key creates. The catalog is therefore
// reconstructable — scenarios re-register on boot and converge to the same
// rows.
type Service interface {
	// Register upserts a target keyed by (owner, name). Returns the resulting
	// target. ErrInvalidTarget on validation failure.
	Register(ctx context.Context, in RegisterInput) (Target, error)

	// Deregister removes a target by (owner, name). Returns whether a row was
	// removed.
	Deregister(ctx context.Context, owner, name string) (bool, error)

	// Get returns a target by id. ErrTargetNotFound propagates verbatim.
	Get(ctx context.Context, id string) (Target, error)

	// List returns targets, optionally filtered by owner. limit <= 0 uses the
	// default.
	List(ctx context.Context, owner string, limit int) ([]Target, error)
}

type service struct {
	repo Repository
}

// NewService constructs the production Service.
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// Compile-time guarantee.
var _ Service = (*service)(nil)

func (s *service) Register(ctx context.Context, in RegisterInput) (Target, error) {
	owner := strings.TrimSpace(in.Owner)
	if owner == "" {
		return Target{}, ErrInvalidTarget{Field: "owner", Reason: "required"}
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return Target{}, ErrInvalidTarget{Field: "name", Reason: "required"}
	}
	if !in.SourceKind.Valid() {
		return Target{}, ErrInvalidTarget{Field: "source_kind", Reason: "must be one of filesystem, sqlite, postgres, redis, qdrant, object-storage"}
	}
	locator := strings.TrimSpace(in.Locator)
	if locator == "" {
		return Target{}, ErrInvalidTarget{Field: "locator", Reason: "required"}
	}

	desired := Target{Owner: owner, Name: name, SourceKind: in.SourceKind, Locator: locator}

	existing, err := s.repo.GetByOwnerName(ctx, owner, name)
	if err != nil {
		var notFound ErrTargetNotFound
		if errors.As(err, &notFound) {
			return s.repo.Create(ctx, desired)
		}
		return Target{}, err
	}

	// Idempotent no-op: identical spec re-register leaves the row (and its
	// UpdatedAt) untouched.
	if existing.sameSpec(desired) {
		return existing, nil
	}

	// Changed spec: update in place, preserving identity and CreatedAt.
	existing.SourceKind = desired.SourceKind
	existing.Locator = desired.Locator
	return s.repo.Update(ctx, existing)
}

func (s *service) Deregister(ctx context.Context, owner, name string) (bool, error) {
	owner = strings.TrimSpace(owner)
	name = strings.TrimSpace(name)
	if owner == "" {
		return false, ErrInvalidTarget{Field: "owner", Reason: "required"}
	}
	if name == "" {
		return false, ErrInvalidTarget{Field: "name", Reason: "required"}
	}
	return s.repo.DeleteByOwnerName(ctx, owner, name)
}

func (s *service) Get(ctx context.Context, id string) (Target, error) {
	if strings.TrimSpace(id) == "" {
		return Target{}, ErrInvalidTarget{Field: "id", Reason: "required"}
	}
	return s.repo.GetByID(ctx, id)
}

func (s *service) List(ctx context.Context, owner string, limit int) ([]Target, error) {
	if limit <= 0 {
		limit = defaultListLimit
	}
	return s.repo.List(ctx, strings.TrimSpace(owner), limit)
}
