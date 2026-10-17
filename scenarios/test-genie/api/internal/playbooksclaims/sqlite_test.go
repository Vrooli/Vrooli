package playbooksclaims_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"test-genie/internal/playbooksclaims"
	"test-genie/internal/testsqlite"
)

func newRepo(t *testing.T) *playbooksclaims.SqliteRepository {
	t.Helper()
	db := testsqlite.Open(t)
	return playbooksclaims.NewSqliteRepository(db)
}

func acquire(t *testing.T, repo *playbooksclaims.SqliteRepository, scenario, runID string, now time.Time) (playbooksclaims.Claim, error) {
	t.Helper()
	return repo.TryAcquire(context.Background(), playbooksclaims.AcquireInput{
		ScenarioName: scenario,
		RunID:        runID,
		Mode:         playbooksclaims.ModeRouted,
		StartedBy:    "tester",
	}, now, playbooksclaims.TTL)
}

func TestTryAcquire_Fresh(t *testing.T) {
	repo := newRepo(t)
	now := time.Unix(1700000000, 0).UTC()

	got, err := acquire(t, repo, "scenario-a", "run-1", now)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	if got.RunID != "run-1" || got.ScenarioName != "scenario-a" {
		t.Fatalf("unexpected claim: %+v", got)
	}
	if !got.ExpiresAt.After(now) {
		t.Fatalf("expires_at not in future: %v vs %v", got.ExpiresAt, now)
	}
}

func TestTryAcquire_RejectsLiveDuplicate(t *testing.T) {
	repo := newRepo(t)
	now := time.Unix(1700000000, 0).UTC()

	if _, err := acquire(t, repo, "scenario-a", "run-1", now); err != nil {
		t.Fatalf("first acquire: %v", err)
	}

	_, err := acquire(t, repo, "scenario-a", "run-2", now)
	if !playbooksclaims.IsBusy(err) {
		t.Fatalf("expected ErrBusy, got %v", err)
	}
	var busy *playbooksclaims.ErrBusy
	if !errors.As(err, &busy) {
		t.Fatalf("not ErrBusy: %v", err)
	}
	if busy.Holder.RunID != "run-1" {
		t.Fatalf("holder run mismatch: %q", busy.Holder.RunID)
	}
}

func TestTryAcquire_StealsExpired(t *testing.T) {
	repo := newRepo(t)
	t0 := time.Unix(1700000000, 0).UTC()
	if _, err := acquire(t, repo, "scenario-a", "run-1", t0); err != nil {
		t.Fatalf("first acquire: %v", err)
	}

	// Advance past TTL — second acquire should steal.
	t1 := t0.Add(playbooksclaims.TTL + time.Second)
	got, err := acquire(t, repo, "scenario-a", "run-2", t1)
	if err != nil {
		t.Fatalf("steal acquire: %v", err)
	}
	if got.RunID != "run-2" {
		t.Fatalf("expected run-2 owner, got %q", got.RunID)
	}
}

func TestHeartbeat_ExtendsAndRejectsMismatch(t *testing.T) {
	repo := newRepo(t)
	t0 := time.Unix(1700000000, 0).UTC()
	if _, err := acquire(t, repo, "scenario-a", "run-1", t0); err != nil {
		t.Fatalf("acquire: %v", err)
	}

	t1 := t0.Add(30 * time.Second)
	c, err := repo.Heartbeat(context.Background(), "scenario-a", "run-1", t1, playbooksclaims.TTL)
	if err != nil {
		t.Fatalf("heartbeat: %v", err)
	}
	if !c.ExpiresAt.After(t0.Add(playbooksclaims.TTL)) {
		t.Fatalf("expires_at not extended: %v", c.ExpiresAt)
	}

	_, err = repo.Heartbeat(context.Background(), "scenario-a", "run-other", t1, playbooksclaims.TTL)
	if !errors.Is(err, playbooksclaims.ErrLeaseMismatch) {
		t.Fatalf("expected ErrLeaseMismatch, got %v", err)
	}
}

func TestRelease(t *testing.T) {
	repo := newRepo(t)
	t0 := time.Unix(1700000000, 0).UTC()
	if _, err := acquire(t, repo, "scenario-a", "run-1", t0); err != nil {
		t.Fatalf("acquire: %v", err)
	}

	if err := repo.Release(context.Background(), "scenario-a", "run-wrong"); !errors.Is(err, playbooksclaims.ErrLeaseMismatch) {
		t.Fatalf("expected ErrLeaseMismatch, got %v", err)
	}

	if err := repo.Release(context.Background(), "scenario-a", "run-1"); err != nil {
		t.Fatalf("release: %v", err)
	}

	if err := repo.Release(context.Background(), "scenario-a", "run-1"); !errors.Is(err, playbooksclaims.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestForceBreak(t *testing.T) {
	repo := newRepo(t)
	t0 := time.Unix(1700000000, 0).UTC()
	if _, err := acquire(t, repo, "scenario-a", "run-1", t0); err != nil {
		t.Fatalf("acquire: %v", err)
	}

	c, err := repo.ForceBreak(context.Background(), "scenario-a")
	if err != nil {
		t.Fatalf("force-break: %v", err)
	}
	if c.RunID != "run-1" {
		t.Fatalf("unexpected broken claim: %+v", c)
	}

	if _, err := acquire(t, repo, "scenario-a", "run-2", t0); err != nil {
		t.Fatalf("re-acquire after break: %v", err)
	}
}

func TestList(t *testing.T) {
	repo := newRepo(t)
	t0 := time.Unix(1700000000, 0).UTC()
	if _, err := acquire(t, repo, "scenario-a", "run-1", t0); err != nil {
		t.Fatalf("acquire a: %v", err)
	}
	if _, err := acquire(t, repo, "scenario-b", "run-2", t0); err != nil {
		t.Fatalf("acquire b: %v", err)
	}

	claims, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(claims) != 2 {
		t.Fatalf("expected 2 claims, got %d", len(claims))
	}
}

func TestConcurrentAcquire(t *testing.T) {
	repo := newRepo(t)
	t0 := time.Unix(1700000000, 0).UTC()

	const n = 10
	var wg sync.WaitGroup
	var wins int32
	var busies int32

	start := make(chan struct{})
	for i := 0; i < n; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			runID := "run-" + itoa(i)
			_, err := acquire(t, repo, "scenario-race", runID, t0)
			if err == nil {
				atomic.AddInt32(&wins, 1)
			} else if playbooksclaims.IsBusy(err) {
				atomic.AddInt32(&busies, 1)
			} else {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	close(start)
	wg.Wait()

	if wins != 1 {
		t.Fatalf("expected exactly 1 winner, got %d", wins)
	}
	if busies != n-1 {
		t.Fatalf("expected %d busy errors, got %d", n-1, busies)
	}
}

func itoa(i int) string {
	const digits = "0123456789"
	if i == 0 {
		return "0"
	}
	var b [20]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = digits[i%10]
		i /= 10
	}
	return string(b[pos:])
}
