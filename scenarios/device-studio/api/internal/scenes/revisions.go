package scenes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"device-studio/internal/revisionstore"
)

var (
	ErrRevisionNotFound = revisionstore.ErrNotFound
	ErrRevisionConflict = revisionstore.ErrConflict
	ErrInvalidCandidate = errors.New("invalid scene candidate")
)

type RevisionState string

const RevisionCandidate RevisionState = "candidate"

type Revision struct {
	SceneID   string         `json:"scene_id"`
	Revision  uint32         `json:"revision"`
	SHA256    string         `json:"sha256"`
	State     RevisionState  `json:"state"`
	Candidate SceneCandidate `json:"candidate"`
	CreatedAt time.Time      `json:"created_at"`
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
		return nil, errors.New("scene revision repository is required")
	}
	if now == nil {
		now = time.Now
	}
	return &RevisionService{repo: repo, now: now}, nil
}

func (s *RevisionService) SaveCandidate(ctx context.Context, input SceneCandidate, expectedHead uint32) (Revision, error) {
	candidate, canonical, digest, err := CanonicalizeCandidate(input)
	if err != nil {
		return Revision{}, fmt.Errorf("%w: %v", ErrInvalidCandidate, err)
	}
	return s.repo.SaveCandidate(ctx, Revision{SceneID: candidate.SceneID, SHA256: digest, State: RevisionCandidate, Candidate: candidate, CreatedAt: s.now().UTC(), canonical: canonical}, expectedHead)
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
	store *revisionstore.SQLite[SceneCandidate]
}

func NewSQLiteRevisionRepository(db revisionstore.Database) (RevisionRepository, error) {
	store, err := revisionstore.NewSQLite[SceneCandidate](db, revisionstore.SceneRevisions, revisionstore.Codec[SceneCandidate]{
		ID:     func(c SceneCandidate) string { return c.SceneID },
		Decode: decodeCanonicalScene,
	})
	if err != nil {
		return nil, err
	}
	return &sqliteRevisionRepository{store: store}, nil
}

func (r *sqliteRevisionRepository) SaveCandidate(ctx context.Context, proposed Revision, expected uint32) (Revision, error) {
	stored, err := r.store.Save(ctx, revisionstore.Record[SceneCandidate]{ID: proposed.SceneID, SHA256: proposed.SHA256, Value: proposed.Candidate, Canonical: proposed.canonical, CreatedAt: proposed.CreatedAt}, expected)
	return fromStoredScene(stored), err
}
func (r *sqliteRevisionRepository) GetRevision(ctx context.Context, id string, revision uint32) (Revision, error) {
	stored, err := r.store.Get(ctx, id, revision)
	return fromStoredScene(stored), err
}
func (r *sqliteRevisionRepository) GetHead(ctx context.Context, id string) (Revision, error) {
	stored, err := r.store.Head(ctx, id)
	return fromStoredScene(stored), err
}
func (r *sqliteRevisionRepository) ListRevisions(ctx context.Context, id string, after uint32, limit int) ([]Revision, error) {
	stored, err := r.store.List(ctx, id, after, limit)
	if err != nil {
		return nil, err
	}
	rows := make([]Revision, len(stored))
	for i := range stored {
		rows[i] = fromStoredScene(stored[i])
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
		rows[i] = fromStoredScene(stored[i])
	}
	return rows, nil
}

func fromStoredScene(stored revisionstore.Record[SceneCandidate]) Revision {
	return Revision{SceneID: stored.ID, Revision: stored.Revision, SHA256: stored.SHA256, State: RevisionCandidate, Candidate: stored.Value, CreatedAt: stored.CreatedAt, canonical: stored.Canonical}
}

func decodeCanonicalScene(data []byte) (SceneCandidate, string, error) {
	if len(data) == 0 || len(data) > MaxCanonicalJSON {
		return SceneCandidate{}, "", errors.New("scene candidate byte length is invalid")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var candidate SceneCandidate
	if err := decoder.Decode(&candidate); err != nil {
		return SceneCandidate{}, "", fmt.Errorf("decode scene candidate: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return SceneCandidate{}, "", errors.New("scene candidate has trailing JSON")
	}
	canonical, encoded, digest, err := CanonicalizeCandidate(candidate)
	if err != nil {
		return SceneCandidate{}, "", err
	}
	if !bytes.Equal(encoded, data) {
		return SceneCandidate{}, "", errors.New("scene candidate JSON is not canonical")
	}
	return canonical, digest, nil
}
