package experiment

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"

	"audio-tools/internal/database"
	"audio-tools/internal/testutil/db"
	"audio-tools/internal/testutil/mocks"
)

type internalMemBlobs struct {
	mu   sync.Mutex
	data map[string][]byte
}

func (m *internalMemBlobs) Put(_ context.Context, key string, data []byte, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.data == nil {
		m.data = map[string][]byte{}
	}
	m.data[key] = append([]byte(nil), data...)
	return nil
}

func (m *internalMemBlobs) Get(_ context.Context, key string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	data, ok := m.data[key]
	if !ok {
		return nil, errors.New("missing blob")
	}
	return append([]byte(nil), data...), nil
}

func (m *internalMemBlobs) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}

func newInternalSchemaDB(t *testing.T) *sql.DB {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(database.SystemSchema),
		apidb.SchemaProviderFunc(Schema),
	))
	return d
}

func TestManager_EvictsTerminalEntriesAndFallsBackToDB(t *testing.T) {
	clk := mocks.NewFakeClock(time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC))
	repo := NewSQLiteRepository(newInternalSchemaDB(t), clk)
	svc := NewService(repo, &internalMemBlobs{})
	mgr := NewManager(Config{
		Service: svc,
		Clock:   clk,
		Runner: func(context.Context, Experiment, func(int, string)) (RunResult, error) {
			return RunResult{Report: []byte(`{"ok":true}`)}, nil
		},
	})
	require.NoError(t, mgr.Start(context.Background()))
	t.Cleanup(mgr.Close)

	ids := make([]string, 0, 5)
	for i := 0; i < 5; i++ {
		exp, err := mgr.Submit(context.Background(), SubmitSpec{Name: "evict"})
		require.NoError(t, err)
		done, err := mgr.Wait(context.Background(), exp.ID)
		require.NoError(t, err)
		require.Equal(t, StatusSucceeded, done.Status)
		ids = append(ids, exp.ID)
	}

	mgr.mu.Lock()
	entryCount := len(mgr.entries)
	mgr.mu.Unlock()
	require.Zero(t, entryCount, "terminal entries should not remain resident")

	got, err := mgr.Get(context.Background(), ids[0])
	require.NoError(t, err)
	require.Equal(t, StatusSucceeded, got.Status)
	require.Equal(t, ids[0], got.ID)
}

type failRecoverRepo struct {
	Repository
}

func (failRecoverRepo) ListNonTerminal(context.Context) ([]Experiment, error) {
	return nil, errors.New("recover failed")
}

func TestManager_StartRecoverFailureDoesNotExposeStarted(t *testing.T) {
	svc := NewService(failRecoverRepo{}, &internalMemBlobs{})
	mgr := NewManager(Config{
		Service: svc,
		Runner: func(context.Context, Experiment, func(int, string)) (RunResult, error) {
			return RunResult{}, nil
		},
	})

	require.Error(t, mgr.Start(context.Background()))
	_, err := mgr.Submit(context.Background(), SubmitSpec{Name: "must-not-submit"})
	require.ErrorIs(t, err, ErrNotStarted)
}
