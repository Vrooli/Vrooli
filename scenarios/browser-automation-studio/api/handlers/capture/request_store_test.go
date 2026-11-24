package capture

import (
	"context"
	"errors"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestSQLCaptureRequestStoreIdempotencyAndChangedRequestConflict(t *testing.T) {
	db, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(`CREATE TABLE capture_requests (request_token TEXT PRIMARY KEY, request_hash TEXT NOT NULL, request_json TEXT NOT NULL, execution_id TEXT NOT NULL UNIQUE, state TEXT NOT NULL, response_json TEXT NOT NULL DEFAULT '', error TEXT NOT NULL DEFAULT '', created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP)`)
	require.NoError(t, err)
	store := NewSQLCaptureRequestStore(db)
	ctx := context.Background()
	input := CaptureRequestRecord{RequestToken: "rcl-state:one", RequestHash: "same", RequestJSON: `{}`, ExecutionID: "exec-1"}
	first, created, err := store.Create(ctx, input)
	require.NoError(t, err)
	require.True(t, created)
	require.Equal(t, "in_progress", first.State)

	replay, created, err := store.Create(ctx, CaptureRequestRecord{RequestToken: input.RequestToken, RequestHash: input.RequestHash, RequestJSON: input.RequestJSON, ExecutionID: "exec-retry"})
	require.NoError(t, err)
	require.False(t, created)
	require.Equal(t, input.ExecutionID, replay.ExecutionID)

	_, _, err = store.Create(ctx, CaptureRequestRecord{RequestToken: input.RequestToken, RequestHash: "changed", RequestJSON: `{"url":"https://other.test"}`, ExecutionID: "exec-other"})
	require.ErrorIs(t, err, ErrRequestTokenConflict)

	require.NoError(t, store.Complete(ctx, input.RequestToken, `{"executionId":"exec-1"}`))
	completed, err := store.Get(ctx, input.RequestToken)
	require.NoError(t, err)
	require.Equal(t, "completed", completed.State)
	require.Equal(t, `{"executionId":"exec-1"}`, completed.ResponseJSON)
}

func TestNormalizeRequestToken(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  string
		bad   bool
	}{
		{input: "  RCL.state:one  ", want: "RCL.state:one"},
		{input: "", bad: true},
		{input: "has space", bad: true},
		{input: "../unsafe", bad: true},
		{input: string(make([]byte, 129)), bad: true},
	} {
		got, err := NormalizeRequestToken(tc.input)
		if tc.bad {
			require.Error(t, err)
			continue
		}
		require.NoError(t, err)
		require.Equal(t, tc.want, got)
	}
}

func TestSQLCaptureRequestStoreChangedInputReturnsSpecificConflict(t *testing.T) {
	db, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE capture_requests (request_token TEXT PRIMARY KEY, request_hash TEXT NOT NULL, request_json TEXT NOT NULL, execution_id TEXT NOT NULL UNIQUE, state TEXT NOT NULL, response_json TEXT NOT NULL DEFAULT '', error TEXT NOT NULL DEFAULT '', created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP)`)
	require.NoError(t, err)
	store := NewSQLCaptureRequestStore(db)
	_, _, err = store.Create(context.Background(), CaptureRequestRecord{RequestToken: "key", RequestHash: "one", ExecutionID: "id1"})
	require.NoError(t, err)
	_, _, err = store.Create(context.Background(), CaptureRequestRecord{RequestToken: "key", RequestHash: "two", ExecutionID: "id2"})
	require.True(t, errors.Is(err, ErrRequestTokenConflict))
}
