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

// [REQ:BAS-P0-101]
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

func TestEvidenceBudgetsLoadManifestAndOverrideCapacity(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("SCENARIO_ROOT", root)
	t.Setenv("BAS_OWNER_RETENTION_MAX_AGE", "")
	for _, name := range []string{"RECORDINGS", "CAPTURES"} {
		t.Setenv("BAS_"+name+"_RETENTION_MAX_AGE", "")
		t.Setenv("BAS_"+name+"_RETENTION_MAX_BYTES", "")
	}
	budgets, err := evidenceBudgets()
	if err != nil || budgets["recordings"].MaxBytes != 20<<30 || budgets["captures"].MaxBytes != 5<<30 {
		t.Fatalf("budgets=%+v err=%v", budgets, err)
	}
	t.Setenv("BAS_CAPTURES_RETENTION_MAX_BYTES", "2GiB")
	budgets, err = evidenceBudgets()
	if err != nil || budgets["captures"].MaxBytes != 2<<30 {
		t.Fatalf("override=%+v err=%v", budgets, err)
	}
	t.Setenv("BAS_CAPTURES_RETENTION_MAX_BYTES", "0")
	if _, err = evidenceBudgets(); err == nil {
		t.Fatal("zero capacity silently disables retention")
	}
}

func TestEvidenceBudgetsFromLifecycleRepoWorkingDirectory(t *testing.T) {
	repoRoot, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("SCENARIO_ROOT", "")
	t.Setenv("VROOLI_ROOT", repoRoot)
	t.Chdir(repoRoot)
	budgets, err := evidenceBudgets()
	if err != nil || len(budgets) != 2 {
		t.Fatalf("lifecycle budgets=%+v err=%v", budgets, err)
	}
}

func TestEvidenceRetentionBatchSizeIsBoundedAndConfigurable(t *testing.T) {
	t.Setenv("BAS_EVIDENCE_RETENTION_BATCH_SIZE", "")
	n, err := evidenceRetentionBatchSize()
	if err != nil || n != 2000 {
		t.Fatalf("default=%d err=%v", n, err)
	}
	t.Setenv("BAS_EVIDENCE_RETENTION_BATCH_SIZE", "100000")
	n, err = evidenceRetentionBatchSize()
	if err != nil || n != 100000 {
		t.Fatalf("catchup=%d err=%v", n, err)
	}
	for _, raw := range []string{"0", "-1", "100001", "invalid"} {
		t.Setenv("BAS_EVIDENCE_RETENTION_BATCH_SIZE", raw)
		if _, err = evidenceRetentionBatchSize(); err == nil {
			t.Fatalf("accepted %q", raw)
		}
	}
}

// [REQ:BAS-P0-101]
func TestEvidenceRetentionIncludesNestedArtifactBundles(t *testing.T) {
	root := t.TempDir()
	old, active, newest := uuid.New(), uuid.New(), uuid.New()
	repo := &evidenceRepository{states: map[uuid.UUID]string{old: "completed", active: "running", newest: "completed"}}
	entries := []struct {
		path  string
		bytes int
		age   time.Duration
	}{
		{old.String(), 10, 3 * time.Hour},
		{filepath.Join("artifacts", old.String()), 60, 2 * time.Hour},
		{filepath.Join("artifacts", active.String()), 20, 4 * time.Hour},
		{filepath.Join("artifacts", newest.String()), 10, time.Hour},
	}
	for _, entry := range entries {
		path := filepath.Join(root, entry.path)
		if err := os.MkdirAll(path, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(path, "artifact"), make([]byte, entry.bytes), 0600); err != nil {
			t.Fatal(err)
		}
		at := time.Now().Add(-entry.age)
		if err := os.Chtimes(path, at, at); err != nil {
			t.Fatal(err)
		}
	}
	service := &ownerCleanupService{root: root, repo: repo}
	result, err := service.enforceEvidenceBudget(context.Background(), "recordings", retention.Budget{Name: "recordings", MaxBytes: 30, MaxAge: 7 * 24 * time.Hour}, 0)
	if err != nil || result.After.Bytes != 30 || result.FreedBytes != 70 || result.Incomplete {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	for _, id := range []uuid.UUID{active, newest} {
		if _, err := os.Stat(filepath.Join(root, "artifacts", id.String(), "artifact")); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "artifacts", old.String())); !os.IsNotExist(err) {
		t.Fatalf("expired bundle survived: %v", err)
	}
}
