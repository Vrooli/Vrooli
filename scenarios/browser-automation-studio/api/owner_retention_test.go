package main

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/retention"
	"github.com/vrooli/browser-automation-studio/database"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type evidenceRepository struct {
	database.Repository
	states  map[uuid.UUID]string
	deleted []uuid.UUID
	err     error
}

func (r *evidenceRepository) GetExecution(_ context.Context, id uuid.UUID) (*database.ExecutionIndex, error) {
	if r.err != nil {
		return nil, r.err
	}
	state, ok := r.states[id]
	if !ok {
		return nil, database.ErrNotFound
	}
	return &database.ExecutionIndex{Status: state}, nil
}
func (r *evidenceRepository) DeleteExecution(_ context.Context, id uuid.UUID) error {
	r.deleted = append(r.deleted, id)
	delete(r.states, id)
	return nil
}

func TestEvidenceRetentionBoundsYoungDataAndProtectsActive(t *testing.T) {
	for _, kind := range []string{"recordings", "captures"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			repo := &evidenceRepository{states: map[uuid.UUID]string{}}
			active, old, newest := uuid.New(), uuid.New(), uuid.New()
			for i, id := range []uuid.UUID{active, old, newest} {
				path := filepath.Join(root, id.String())
				if err := os.Mkdir(path, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(path, "evidence"), make([]byte, 10), 0600); err != nil {
					t.Fatal(err)
				}
				at := time.Now().Add(time.Duration(i-10) * time.Minute)
				if err := os.Chtimes(path, at, at); err != nil {
					t.Fatal(err)
				}
				repo.states[id] = "completed"
			}
			repo.states[active] = "running"
			s := &ownerCleanupService{root: root, capturesRoot: root, repo: repo}
			result, err := s.enforceEvidenceBudget(context.Background(), kind, retention.Budget{Name: kind, MaxAge: 7 * 24 * time.Hour, MaxBytes: 20}, 1)
			if err != nil || result.Deleted != 1 || result.After.Bytes != 20 {
				t.Fatalf("result=%+v err=%v", result, err)
			}
			if _, err := os.Stat(filepath.Join(root, old.String())); !os.IsNotExist(err) {
				t.Fatalf("old data retained: %v", err)
			}
			for _, id := range []uuid.UUID{active, newest} {
				if _, err := os.Stat(filepath.Join(root, id.String())); err != nil {
					t.Fatal(err)
				}
			}
			if kind == "recordings" && (len(repo.deleted) != 1 || repo.deleted[0] != old) {
				t.Fatalf("index cleanup=%v", repo.deleted)
			}
			repo.err = errors.New("repository unavailable")
			if _, err := s.enforceEvidenceBudget(context.Background(), kind, retention.Budget{Name: kind, MaxBytes: 1}, 0); err == nil {
				t.Fatal("repository failure must fail closed")
			}
		})
	}
}
