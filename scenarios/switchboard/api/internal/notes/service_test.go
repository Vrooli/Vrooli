package notes_test

import (
	"context"
	"errors"
	"testing"
	"time"
	"switchboard/internal/notes"

	"github.com/stretchr/testify/require"

	mocks "switchboard/internal/notes/mocks"
)

func TestService_CountInWindow_DelegatesToRepository(t *testing.T) {
	repo := mocks.NewFakeRepository()
	repo.CountOut = 11
	svc := notes.NewService(repo)

	from := time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	n, err := svc.CountInWindow(context.Background(), from, to)
	require.NoError(t, err)
	require.Equal(t, 11, n)
	require.Equal(t, [][2]time.Time{{from, to}}, repo.CountWindows,
		"service must pass the window through unchanged")
}

func TestService_CountInWindow_PropagatesError(t *testing.T) {
	repo := mocks.NewFakeRepository()
	repo.CountErr = errors.New("boom")
	svc := notes.NewService(repo)

	_, err := svc.CountInWindow(context.Background(), time.Time{}, time.Time{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "boom")
}

func TestService_Create_RejectsEmptyTitle(t *testing.T) {
	repo := mocks.NewFakeRepository()
	svc := notes.NewService(repo)

	_, err := svc.Create(context.Background(), notes.CreateInput{Title: ""})
	require.Error(t, err)
	var inv notes.ErrInvalidNote
	require.True(t, errors.As(err, &inv), "expected ErrInvalidNote, got %T: %v", err, err)
	require.Equal(t, "title", inv.Field)
	require.Equal(t, int64(0), repo.CreateCalls.Load(),
		"validation must reject before reaching the repository")
}

func TestService_Create_TrimsWhitespace(t *testing.T) {
	repo := mocks.NewFakeRepository()
	svc := notes.NewService(repo)

	_, err := svc.Create(context.Background(), notes.CreateInput{Title: "   "})
	require.Error(t, err)
	var inv notes.ErrInvalidNote
	require.True(t, errors.As(err, &inv))
	require.Equal(t, "title", inv.Field,
		"whitespace-only title must rejected the same as empty")
}

func TestService_Create_DelegatesToRepo(t *testing.T) {
	repo := mocks.NewFakeRepository()
	svc := notes.NewService(repo)

	got, err := svc.Create(context.Background(), notes.CreateInput{Title: "hello", Body: "world"})
	require.NoError(t, err)
	require.NotEmpty(t, got.ID, "repo populates ID")
	require.Equal(t, "hello", got.Title)
	require.Equal(t, "world", got.Body)
	require.Equal(t, int64(1), repo.CreateCalls.Load())
}

func TestService_Create_TrimsBeforePersisting(t *testing.T) {
	repo := mocks.NewFakeRepository()
	svc := notes.NewService(repo)

	got, err := svc.Create(context.Background(), notes.CreateInput{Title: "  hello  ", Body: "world"})
	require.NoError(t, err)
	require.Equal(t, "hello", got.Title,
		"the trimmed value, not the raw input, is what gets persisted")
}

func TestService_Create_PropagatesRepoError(t *testing.T) {
	want := errors.New("repo boom")
	repo := mocks.NewFakeRepository()
	repo.CreateErr = want
	svc := notes.NewService(repo)

	_, err := svc.Create(context.Background(), notes.CreateInput{Title: "hello"})
	require.ErrorIs(t, err, want)
}

func TestService_Get_PropagatesNotFound(t *testing.T) {
	repo := mocks.NewFakeRepository()
	svc := notes.NewService(repo)

	_, err := svc.Get(context.Background(), "ghost")
	require.Error(t, err)
	var nf notes.ErrNoteNotFound
	require.True(t, errors.As(err, &nf), "service must propagate the typed sentinel verbatim")
	require.Equal(t, "ghost", nf.ID)
}

func TestService_Get_ReturnsRepoNote(t *testing.T) {
	repo := mocks.NewFakeRepository()
	repo.Items = []notes.Note{{ID: "abc", Title: "found"}}
	svc := notes.NewService(repo)

	got, err := svc.Get(context.Background(), "abc")
	require.NoError(t, err)
	require.Equal(t, "abc", got.ID)
	require.Equal(t, "found", got.Title)
}

// repoLimitSpy records the limit value Repository.List was called with
// so the service test can prove the default-substitution + pass-through
// contract without depending on the fake's accidental behaviour.
type repoLimitSpy struct {
	notes.Repository
	gotLimit int
}

func (r *repoLimitSpy) List(ctx context.Context, limit int) ([]notes.Note, error) {
	r.gotLimit = limit
	return nil, nil
}

func TestService_List_AppliesDefaultLimit(t *testing.T) {
	spy := &repoLimitSpy{Repository: mocks.NewFakeRepository()}
	svc := notes.NewService(spy)

	_, err := svc.List(context.Background(), 0)
	require.NoError(t, err)
	require.Equal(t, 100, spy.gotLimit,
		"limit <= 0 must be substituted with defaultListLimit (100)")
}

func TestService_List_AppliesDefaultOnNegative(t *testing.T) {
	spy := &repoLimitSpy{Repository: mocks.NewFakeRepository()}
	svc := notes.NewService(spy)

	_, err := svc.List(context.Background(), -3)
	require.NoError(t, err)
	require.Equal(t, 100, spy.gotLimit,
		"negative limit is the same as 0 — both mean 'use default'")
}

func TestService_List_HonorsExplicitLimit(t *testing.T) {
	spy := &repoLimitSpy{Repository: mocks.NewFakeRepository()}
	svc := notes.NewService(spy)

	_, err := svc.List(context.Background(), 5)
	require.NoError(t, err)
	require.Equal(t, 5, spy.gotLimit,
		"a positive caller-supplied limit must pass through unchanged")
}

func TestService_List_PropagatesRepoError(t *testing.T) {
	want := errors.New("list boom")
	repo := mocks.NewFakeRepository()
	repo.ListErr = want
	svc := notes.NewService(repo)

	_, err := svc.List(context.Background(), 5)
	require.ErrorIs(t, err, want)
}
