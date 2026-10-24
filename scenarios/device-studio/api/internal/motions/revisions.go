package motions

import (
	"context"
	"errors"
	"fmt"
	"time"

	"device-studio/internal/revisionstore"
)

var (
	ErrRevisionNotFound = revisionstore.ErrNotFound
	ErrRevisionConflict = revisionstore.ErrConflict
	ErrInvalidCandidate = errors.New("invalid motion candidate")
)

type RevisionState string

const RevisionCandidate RevisionState = "candidate"

type Revision struct {
	MotionID  string          `json:"motion_id"`
	Revision  uint32          `json:"revision"`
	SHA256    string          `json:"sha256"`
	State     RevisionState   `json:"state"`
	Candidate MotionCandidate `json:"candidate"`
	CreatedAt time.Time       `json:"created_at"`
	canonical []byte
}

type RevisionRepository interface {
	SaveCandidate(context.Context, Revision, uint32) (Revision, error)
	GetRevision(context.Context, string, uint32) (Revision, error)
	GetHead(context.Context, string) (Revision, error)
	ListRevisions(context.Context, string, uint32, int) ([]Revision, error)
	ListHeads(context.Context, string, int) ([]Revision, error)
}

type RevisionService struct {
	repo RevisionRepository
	now  func() time.Time
}

func NewRevisionService(repo RevisionRepository, now func() time.Time) (*RevisionService, error) {
	if repo == nil {
		return nil, errors.New("motion revision repository is required")
	}
	if now == nil {
		now = time.Now
	}
	return &RevisionService{repo: repo, now: now}, nil
}

func (s *RevisionService) SaveCandidate(ctx context.Context, input MotionCandidate, expectedHead uint32) (Revision, error) {
	candidate, canonical, digest, err := CanonicalizeCandidate(input)
	if err != nil {
		return Revision{}, fmt.Errorf("%w: %v", ErrInvalidCandidate, err)
	}
	return s.repo.SaveCandidate(ctx, Revision{MotionID: candidate.MotionID, SHA256: digest, State: RevisionCandidate, Candidate: candidate, CreatedAt: s.now().UTC(), canonical: canonical}, expectedHead)
}
func (s *RevisionService) GetRevision(ctx context.Context, id string, revision uint32) (Revision, error) {
	return s.repo.GetRevision(ctx, id, revision)
}
func (s *RevisionService) GetHead(ctx context.Context, id string) (Revision, error) {
	return s.repo.GetHead(ctx, id)
}
func (s *RevisionService) ListRevisions(ctx context.Context, id string, after uint32, limit int) ([]Revision, error) {
	return s.repo.ListRevisions(ctx, id, after, limit)
}

func (s *RevisionService) ListHeads(ctx context.Context, afterID string, limit int) ([]Revision, error) {
	return s.repo.ListHeads(ctx, afterID, limit)
}

type sqliteRevisionRepository struct {
	store *revisionstore.SQLite[MotionCandidate]
}

func NewSQLiteRevisionRepository(db revisionstore.Database) (RevisionRepository, error) {
	store, err := revisionstore.NewSQLite[MotionCandidate](db, revisionstore.MotionRevisions, revisionstore.Codec[MotionCandidate]{
		ID:     func(c MotionCandidate) string { return c.MotionID },
		Decode: DecodeCandidate,
	})
	if err != nil {
		return nil, err
	}
	return &sqliteRevisionRepository{store: store}, nil
}

func (r *sqliteRevisionRepository) SaveCandidate(ctx context.Context, proposed Revision, expected uint32) (Revision, error) {
	stored, err := r.store.Save(ctx, revisionstore.Record[MotionCandidate]{ID: proposed.MotionID, SHA256: proposed.SHA256, Value: proposed.Candidate, Canonical: proposed.canonical, CreatedAt: proposed.CreatedAt}, expected)
	return fromStoredMotion(stored), err
}
func (r *sqliteRevisionRepository) GetRevision(ctx context.Context, id string, revision uint32) (Revision, error) {
	stored, err := r.store.Get(ctx, id, revision)
	return fromStoredMotion(stored), err
}
func (r *sqliteRevisionRepository) GetHead(ctx context.Context, id string) (Revision, error) {
	stored, err := r.store.Head(ctx, id)
	return fromStoredMotion(stored), err
}
func (r *sqliteRevisionRepository) ListRevisions(ctx context.Context, id string, after uint32, limit int) ([]Revision, error) {
	stored, err := r.store.List(ctx, id, after, limit)
	if err != nil {
		return nil, err
	}
	rows := make([]Revision, len(stored))
	for i := range stored {
		rows[i] = fromStoredMotion(stored[i])
	}
	return rows, nil
}

func (r *sqliteRevisionRepository) ListHeads(ctx context.Context, afterID string, limit int) ([]Revision, error) {
	stored, err := r.store.ListHeads(ctx, afterID, limit)
	if err != nil {
		return nil, err
	}
	rows := make([]Revision, len(stored))
	for i := range stored {
		rows[i] = fromStoredMotion(stored[i])
	}
	return rows, nil
}
func fromStoredMotion(stored revisionstore.Record[MotionCandidate]) Revision {
	return Revision{MotionID: stored.ID, Revision: stored.Revision, SHA256: stored.SHA256, State: RevisionCandidate, Candidate: stored.Value, CreatedAt: stored.CreatedAt, canonical: stored.Canonical}
}
