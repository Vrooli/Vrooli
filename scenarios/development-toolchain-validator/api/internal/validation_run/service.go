package validation_run

import (
	"context"
	"errors"
	"strings"

	vr "development-toolchain-validator/internal/validation_record"

	"github.com/vrooli/api-core/schedule"

	"github.com/google/uuid"
)

// Service is the application-layer surface the validation_run handler
// depends on. Start is async — the in-process worker actually advances
// the lifecycle.
type Service interface {
	Start(ctx context.Context, in StartInput) (Run, error)
	Get(ctx context.Context, id string) (Run, error)
	ListActive(ctx context.Context) ([]Run, error)
}

type service struct {
	repo  Repository
	clock schedule.Clock

	// Notify is called whenever a new run is queued so the worker can
	// pick it up promptly. May be nil; the worker also polls periodically.
	Notify func()
}

// NewService constructs the production Service. Notify is optional.
func NewService(repo Repository, clk schedule.Clock, notify func()) Service {
	return &service{repo: repo, clock: clk, Notify: notify}
}

var _ Service = (*service)(nil)

func (s *service) Start(ctx context.Context, in StartInput) (Run, error) {
	in.SubjectID = strings.TrimSpace(in.SubjectID)
	in.GoldenSlug = strings.TrimSpace(in.GoldenSlug)
	if in.TupleKind == vr.TupleKindUnspecified {
		return Run{}, ErrInvalidRun{Field: "tuple_kind", Reason: "required"}
	}
	if in.SubjectID == "" {
		return Run{}, ErrInvalidRun{Field: "subject_id", Reason: "required"}
	}
	if in.GoldenSlug == "" {
		return Run{}, ErrInvalidRun{Field: "golden_slug", Reason: "required"}
	}
	r := Run{
		ID:         uuid.NewString(),
		TupleKind:  in.TupleKind,
		SubjectID:  in.SubjectID,
		GoldenSlug: in.GoldenSlug,
		Status:     StatusQueued,
		CreatedAt:  s.clock.Now().UTC(),
		ForceReRun: in.Force,
	}
	if err := s.repo.Create(ctx, r); err != nil {
		return Run{}, err
	}
	if s.Notify != nil {
		s.Notify()
	}
	return r, nil
}

func (s *service) Get(ctx context.Context, id string) (Run, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Run{}, ErrInvalidRun{Field: "id", Reason: "required"}
	}
	r, err := s.repo.Get(ctx, id)
	if err != nil {
		return Run{}, err
	}
	return r, nil
}

func (s *service) ListActive(ctx context.Context) ([]Run, error) {
	return s.repo.ListActive(ctx)
}

// IsNotFound reports whether err is an ErrRunNotFound. Useful for
// handlers that want to special-case the "no row" sentinel without
// importing the sentinel type.
func IsNotFound(err error) bool {
	var nf ErrRunNotFound
	return errors.As(err, &nf)
}
