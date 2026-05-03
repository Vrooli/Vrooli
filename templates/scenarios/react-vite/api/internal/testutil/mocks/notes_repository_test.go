package mocks

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"{{SCENARIO_ID}}/internal/notes"
)

func TestFakeRepository_CreateAssignsID(t *testing.T) {
	var f FakeRepository
	got, err := f.Create(context.Background(), notes.Note{Title: "x"})
	require.NoError(t, err)
	require.NotEmpty(t, got.ID, "Create must populate ID when caller leaves it empty")
	require.Equal(t, int64(1), f.CreateCalls.Load())
	require.Len(t, f.Notes, 1)
}

func TestFakeRepository_CreateRespectsCallerID(t *testing.T) {
	var f FakeRepository
	got, err := f.Create(context.Background(), notes.Note{ID: "fixed", Title: "x"})
	require.NoError(t, err)
	require.Equal(t, "fixed", got.ID, "Create must not overwrite a caller-supplied ID")
}

func TestFakeRepository_CreateErrSurfaces(t *testing.T) {
	want := errors.New("create boom")
	f := &FakeRepository{CreateErr: want}
	_, err := f.Create(context.Background(), notes.Note{Title: "x"})
	require.ErrorIs(t, err, want)
	require.Empty(t, f.Notes, "failed Create must not mutate state")
}

func TestFakeRepository_GetReturnsNotFoundByDefault(t *testing.T) {
	var f FakeRepository
	_, err := f.Get(context.Background(), "missing")
	require.Error(t, err)
	var nf notes.ErrNoteNotFound
	require.True(t, errors.As(err, &nf))
	require.Equal(t, "missing", nf.ID)
}

func TestFakeRepository_GetErrSurfaces(t *testing.T) {
	want := errors.New("get boom")
	f := &FakeRepository{GetErr: want, Notes: []notes.Note{{ID: "a"}}}
	// GetErr overrides even when the in-memory store has a match —
	// tests need to be able to drive the internal-error path
	// independently of the not-found path.
	_, err := f.Get(context.Background(), "a")
	require.ErrorIs(t, err, want)
}

func TestFakeRepository_ListReturnsCopiedSlice(t *testing.T) {
	f := &FakeRepository{Notes: []notes.Note{{ID: "a"}, {ID: "b"}}}
	got, err := f.List(context.Background(), 10)
	require.NoError(t, err)
	require.Len(t, got, 2)

	// Mutating the returned slice must not affect the fake's state.
	got[0].ID = "mutated"
	require.Equal(t, "a", f.Notes[0].ID)
}

func TestFakeRepository_ListRespectsLimit(t *testing.T) {
	f := &FakeRepository{Notes: []notes.Note{{ID: "a"}, {ID: "b"}, {ID: "c"}}}
	got, err := f.List(context.Background(), 2)
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, "a", got[0].ID)

	none, err := f.List(context.Background(), 0)
	require.NoError(t, err)
	require.Empty(t, none, "limit <= 0 returns no rows")
}

func TestFakeRepository_ListErrSurfaces(t *testing.T) {
	want := errors.New("list boom")
	f := &FakeRepository{ListErr: want}
	_, err := f.List(context.Background(), 5)
	require.ErrorIs(t, err, want)
}

// TestFakeRepository_RaceCleanWhenSharedAcrossGoroutines is the
// load-bearing regression test for the mutex + atomic counters. Run
// with `go test -race`; without the synchronisation the slice append
// inside Create races and the test trips the race detector.
func TestFakeRepository_RaceCleanWhenSharedAcrossGoroutines(t *testing.T) {
	t.Parallel()
	const goroutines = 50
	var f FakeRepository
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_, _ = f.Create(context.Background(), notes.Note{Title: "t"})
		}()
	}
	wg.Wait()
	require.Equal(t, int64(goroutines), f.CreateCalls.Load())
	require.Len(t, f.Notes, goroutines)
}
