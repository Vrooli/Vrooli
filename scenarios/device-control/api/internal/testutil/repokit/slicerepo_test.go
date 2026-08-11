package repokit_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"device-control/internal/testutil/repokit"

	"github.com/stretchr/testify/require"
)

// thing is a stand-in domain entity used only by these tests. Picked over
// any real scenario type so the substrate stays decoupled from a particular
// domain — exercising the generic against a synthetic fixture is the
// canonical pattern for shared testkits.
type thing struct {
	ID   string
	Name string
}

type errNotFound struct{ ID string }

func (e errNotFound) Error() string { return fmt.Sprintf("thing %q not found", e.ID) }

func newThingsRepo() *repokit.SliceRepo[thing] {
	return repokit.NewSliceRepo[thing](
		func(t thing) string { return t.ID },
		func(t *thing, id string) { t.ID = id },
		func(id string) error { return errNotFound{ID: id} },
	)
}

func TestSliceRepo_Create_AssignsIDWhenEmpty(t *testing.T) {
	r := newThingsRepo()
	got, err := r.Create(context.Background(), thing{Name: "alpha"})
	require.NoError(t, err)
	require.NotEmpty(t, got.ID, "Create must populate empty ID")
	require.Equal(t, "alpha", got.Name)
	require.Equal(t, int64(1), r.CreateCalls.Load())
	require.Len(t, r.Items, 1)
}

func TestSliceRepo_Create_PreservesNonEmptyID(t *testing.T) {
	r := newThingsRepo()
	got, err := r.Create(context.Background(), thing{ID: "fixed", Name: "alpha"})
	require.NoError(t, err)
	require.Equal(t, "fixed", got.ID, "supplied ID must round-trip unchanged")
}

func TestSliceRepo_Create_ReturnsCreateErr(t *testing.T) {
	want := errors.New("boom")
	r := newThingsRepo()
	r.CreateErr = want

	_, err := r.Create(context.Background(), thing{Name: "x"})
	require.ErrorIs(t, err, want)
	require.Empty(t, r.Items, "CreateErr must prevent state mutation")
}

func TestSliceRepo_Get_ReturnsItem(t *testing.T) {
	r := newThingsRepo()
	r.Items = []thing{{ID: "a", Name: "alpha"}, {ID: "b", Name: "beta"}}

	got, err := r.Get(context.Background(), "b")
	require.NoError(t, err)
	require.Equal(t, "beta", got.Name)
}

func TestSliceRepo_Get_ReturnsNotFoundOnMiss(t *testing.T) {
	r := newThingsRepo()
	_, err := r.Get(context.Background(), "ghost")
	require.Error(t, err)
	var nf errNotFound
	require.True(t, errors.As(err, &nf), "expected typed sentinel, got %T", err)
	require.Equal(t, "ghost", nf.ID)
}

func TestSliceRepo_Get_ReturnsGetErrOverridingMatch(t *testing.T) {
	want := errors.New("storage bonk")
	r := newThingsRepo()
	r.Items = []thing{{ID: "a", Name: "alpha"}}
	r.GetErr = want

	_, err := r.Get(context.Background(), "a")
	require.ErrorIs(t, err, want, "GetErr must take precedence over a real match")
}

func TestSliceRepo_List_ReturnsRowsInInsertionOrder(t *testing.T) {
	r := newThingsRepo()
	for _, name := range []string{"a", "b", "c"} {
		_, err := r.Create(context.Background(), thing{Name: name})
		require.NoError(t, err)
	}

	got, err := r.List(context.Background(), 10)
	require.NoError(t, err)
	require.Len(t, got, 3)
	require.Equal(t, "a", got[0].Name)
	require.Equal(t, "c", got[2].Name)
}

func TestSliceRepo_List_RespectsLimit(t *testing.T) {
	r := newThingsRepo()
	for i := 0; i < 5; i++ {
		_, err := r.Create(context.Background(), thing{Name: fmt.Sprintf("n%d", i)})
		require.NoError(t, err)
	}

	got, err := r.List(context.Background(), 2)
	require.NoError(t, err)
	require.Len(t, got, 2)

	none, err := r.List(context.Background(), 0)
	require.NoError(t, err)
	require.Empty(t, none, "limit <= 0 returns no rows")
}

func TestSliceRepo_List_ReturnsListErr(t *testing.T) {
	want := errors.New("list bonk")
	r := newThingsRepo()
	r.ListErr = want

	_, err := r.List(context.Background(), 5)
	require.ErrorIs(t, err, want)
}

// TestSliceRepo_RaceFreeUnderConcurrentCreates pins the lock-vs-counter
// contract: counters are atomic, mutating the slice goes through mu. With
// `go test -race`, missing locks would surface here.
func TestSliceRepo_RaceFreeUnderConcurrentCreates(t *testing.T) {
	r := newThingsRepo()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = r.Create(context.Background(), thing{Name: "x"})
		}()
	}
	wg.Wait()

	require.Equal(t, int64(50), r.CreateCalls.Load())
	require.Len(t, r.Items, 50)
}

// TestSliceRepo_PanicsWithoutExtractors documents the failure mode when a
// caller forgets to wire GetID / SetID / NotFound. Better a clear panic
// than a silent zero-string match.
func TestSliceRepo_PanicsWithoutExtractors(t *testing.T) {
	r := &repokit.SliceRepo[thing]{}
	require.Panics(t, func() {
		_, _ = r.Create(context.Background(), thing{})
	})
}

func TestSliceRepo_Get_PanicsWithoutNotFound(t *testing.T) {
	r := &repokit.SliceRepo[thing]{
		GetID: func(t thing) string { return t.ID },
		SetID: func(t *thing, id string) { t.ID = id },
	}
	require.Panics(t, func() {
		_, _ = r.Get(context.Background(), "x")
	})
}
