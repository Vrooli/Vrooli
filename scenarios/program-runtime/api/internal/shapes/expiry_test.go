package shapes

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func insertExpiryShape(t *testing.T, db *sql.DB, key, state, last string) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO program_shapes(shape_key,binding_ids,binding_count,first_seen,last_seen,exemplar_program_id,state) VALUES(?,?,?,?,?,?,?)`, key, `["a","b"]`, 2, last, last, key, state)
	require.NoError(t, err)
}

func TestExpireOnlyDeletesOldObservedShapes(t *testing.T) {
	d := newShapeDB(t)
	old := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339Nano)
	recent := time.Date(2026, 8, 5, 0, 0, 0, 0, time.UTC).Format(time.RFC3339Nano)
	insertExpiryShape(t, d, "old", "observed", old)
	insertExpiryShape(t, d, "nominated-old", "nominated", old)
	insertExpiryShape(t, d, "covered-old", "covered", old)
	insertExpiryShape(t, d, "recent", "observed", recent)
	_, err := d.Exec(`INSERT INTO program_shape_sessions(shape_key,session_id) VALUES('old','s1')`)
	require.NoError(t, err)
	repo := NewRepository(d)
	deleted, err := repo.Expire(context.Background(), time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC), ShapeWindow)
	require.NoError(t, err)
	require.EqualValues(t, 1, deleted)
	_, err = repo.Get(context.Background(), "old")
	require.ErrorIs(t, err, ErrNotFound)
	var sessions int
	require.NoError(t, d.QueryRow(`SELECT COUNT(*) FROM program_shape_sessions WHERE shape_key='old'`).Scan(&sessions))
	require.Zero(t, sessions)
	for _, key := range []string{"nominated-old", "covered-old", "recent"} {
		_, err = repo.Get(context.Background(), key)
		require.NoError(t, err)
	}
}
