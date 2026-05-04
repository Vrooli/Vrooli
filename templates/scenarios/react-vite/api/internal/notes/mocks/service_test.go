package mocks

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"{{SCENARIO_ID}}/internal/notes"
)

func TestFakeService_CreateRecordsInputAndSynthesisesNote(t *testing.T) {
	var f FakeService
	got, err := f.Create(context.Background(), notes.CreateInput{Title: "hi", Body: "there"})
	require.NoError(t, err)
	require.NotEmpty(t, got.ID, "default Create must synthesise an ID")
	require.Equal(t, "hi", got.Title)
	require.Equal(t, "there", got.Body)
	require.Equal(t, int64(1), f.CreateCalls.Load())
	require.Len(t, f.CreateInputs, 1)
	require.Equal(t, "hi", f.CreateInputs[0].Title)
}

func TestFakeService_CreateOutOverridesSynthesis(t *testing.T) {
	canned := notes.Note{ID: "fixed", Title: "canned", Body: "body"}
	f := &FakeService{CreateOut: &canned}
	got, err := f.Create(context.Background(), notes.CreateInput{Title: "ignored"})
	require.NoError(t, err)
	require.Equal(t, "fixed", got.ID, "CreateOut must take precedence over synthesis")
	require.Equal(t, "canned", got.Title)
}

func TestFakeService_CreateErrSurfaces(t *testing.T) {
	want := errors.New("create boom")
	f := &FakeService{CreateErr: want}
	_, err := f.Create(context.Background(), notes.CreateInput{Title: "x"})
	require.ErrorIs(t, err, want)
	require.Len(t, f.CreateInputs, 1, "input still recorded — proves the call reached the fake")
}

func TestFakeService_GetByIDReturnsMatch(t *testing.T) {
	f := &FakeService{GetByID: map[string]notes.Note{"abc": {ID: "abc", Title: "found"}}}
	got, err := f.Get(context.Background(), "abc")
	require.NoError(t, err)
	require.Equal(t, "found", got.Title)
}

func TestFakeService_GetReturnsNotFoundOnMiss(t *testing.T) {
	var f FakeService
	_, err := f.Get(context.Background(), "ghost")
	var nf notes.ErrNoteNotFound
	require.True(t, errors.As(err, &nf))
	require.Equal(t, "ghost", nf.ID)
}

func TestFakeService_GetErrOverridesMatch(t *testing.T) {
	want := errors.New("get boom")
	f := &FakeService{
		GetErr:  want,
		GetByID: map[string]notes.Note{"abc": {ID: "abc"}},
	}
	_, err := f.Get(context.Background(), "abc")
	require.ErrorIs(t, err, want, "GetErr must override GetByID matches so tests can drive 500s independently")
}

func TestFakeService_ListReturnsCopiedSlice(t *testing.T) {
	f := &FakeService{ListOut: []notes.Note{{ID: "a"}, {ID: "b"}}}
	got, err := f.List(context.Background(), 10)
	require.NoError(t, err)
	require.Len(t, got, 2)

	got[0].ID = "mutated"
	require.Equal(t, "a", f.ListOut[0].ID, "returned slice must be a copy")
}

func TestFakeService_ListErrSurfaces(t *testing.T) {
	want := errors.New("list boom")
	f := &FakeService{ListErr: want}
	_, err := f.List(context.Background(), 5)
	require.ErrorIs(t, err, want)
}

// TestFakeService_RaceCleanWhenSharedAcrossGoroutines pins the
// race-cleanliness of the input-recording path. Mirrors the
// FakeRepository race test to keep the patterns symmetrical for
// scenarios that copy these mocks.
func TestFakeService_RaceCleanWhenSharedAcrossGoroutines(t *testing.T) {
	t.Parallel()
	const goroutines = 50
	var f FakeService
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_, _ = f.Create(context.Background(), notes.CreateInput{Title: "t"})
		}()
	}
	wg.Wait()
	require.Equal(t, int64(goroutines), f.CreateCalls.Load())
	require.Len(t, f.CreateInputs, goroutines)
}
