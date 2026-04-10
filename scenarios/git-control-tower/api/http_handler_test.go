package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// fakeRepoService implements the minimum needed for handler context tests.
// We use a nil RepoService so that resolveRepo falls back to GitRunner.ResolveRepoRoot.

func TestRepoRead_DoesNotAcquireLock(t *testing.T) {
	rl := NewRepoLock()
	git := NewFakeGitRunner()
	ctx := context.Background()

	// Hold the write lock — a RepoWrite call would block.
	unlock, err := rl.Acquire(ctx, git.RepoRoot)
	if err != nil {
		t.Fatalf("unexpected error acquiring lock: %v", err)
	}
	defer unlock()

	// RepoRead should succeed immediately even though the lock is held.
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/repo/status", nil)
	hctx := RepoRead(w, r, git, nil, 1*time.Second)
	if hctx == nil {
		t.Fatal("RepoRead returned nil — should succeed without lock")
	}
	defer hctx.Cancel()

	if hctx.RepoDir != git.RepoRoot {
		t.Fatalf("expected RepoDir %q, got %q", git.RepoRoot, hctx.RepoDir)
	}
}

func TestRepoWrite_AcquiresLock(t *testing.T) {
	rl := NewRepoLock()
	git := NewFakeGitRunner()
	ctx := context.Background()

	// Hold the lock.
	unlock, err := rl.Acquire(ctx, git.RepoRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// RepoWrite with a short timeout should fail because the lock is held.
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/repo/stage", nil)
	hctx := RepoWrite(w, r, git, nil, rl, 100*time.Millisecond)
	if hctx != nil {
		hctx.Cancel()
		t.Fatal("RepoWrite should return nil when lock is held and timeout expires")
	}

	// The response should be 503 Service Unavailable.
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", w.Code)
	}

	// Release the lock — next RepoWrite should succeed.
	unlock()

	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest("POST", "/repo/stage", nil)
	hctx2 := RepoWrite(w2, r2, git, nil, rl, 1*time.Second)
	if hctx2 == nil {
		t.Fatal("RepoWrite should succeed after lock is released")
	}
	hctx2.Cancel()
}

func TestRepoRead_ConcurrentWithWriteLock(t *testing.T) {
	rl := NewRepoLock()
	git := NewFakeGitRunner()
	ctx := context.Background()

	// Hold the write lock for the duration of the test.
	unlock, err := rl.Acquire(ctx, git.RepoRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer unlock()

	// Launch 10 concurrent reads — they should all succeed immediately.
	const readers = 10
	var wg sync.WaitGroup
	errs := make(chan string, readers)

	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/repo/status", nil)
			hctx := RepoRead(w, r, git, nil, 500*time.Millisecond)
			if hctx == nil {
				errs <- "RepoRead returned nil while write lock was held"
				return
			}
			hctx.Cancel()
		}()
	}

	wg.Wait()
	close(errs)

	for msg := range errs {
		t.Error(msg)
	}
}

func TestRepoWrite_SerializesWrites(t *testing.T) {
	rl := NewRepoLock()
	git := NewFakeGitRunner()

	const writers = 5
	var mu sync.Mutex
	var order []int
	var wg sync.WaitGroup

	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/repo/stage", nil)
			hctx := RepoWrite(w, r, git, nil, rl, 5*time.Second)
			if hctx == nil {
				t.Errorf("writer %d: RepoWrite returned nil", id)
				return
			}
			// Simulate work while holding the lock.
			time.Sleep(5 * time.Millisecond)
			mu.Lock()
			order = append(order, id)
			mu.Unlock()
			hctx.Cancel()
		}(i)
	}

	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	if len(order) != writers {
		t.Fatalf("expected %d completions, got %d", writers, len(order))
	}
}

func TestRepoRead_InvalidRepo(t *testing.T) {
	git := NewFakeGitRunner()
	git.IsRepository = false
	git.RepoRoot = ""

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/repo/status", nil)
	hctx := RepoRead(w, r, git, nil, 1*time.Second)
	if hctx != nil {
		hctx.Cancel()
		t.Fatal("expected nil for invalid repo")
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestRepoWrite_InvalidRepo(t *testing.T) {
	rl := NewRepoLock()
	git := NewFakeGitRunner()
	git.IsRepository = false
	git.RepoRoot = ""

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/repo/stage", nil)
	hctx := RepoWrite(w, r, git, nil, rl, 1*time.Second)
	if hctx != nil {
		hctx.Cancel()
		t.Fatal("expected nil for invalid repo")
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
