package hub_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"

	hub "notification-hub/internal/hub"
)

// Deletes must leave no orphans whether or not the connection enforces foreign
// keys: production opens SQLite with them on, but a test pool, a restored
// database or a future driver setting must not change what a delete means.
var foreignKeyModes = map[string]string{"enforced": "ON", "not enforced": "OFF"}

func setForeignKeys(t *testing.T, db *database.RoutedDB, mode string) {
	t.Helper()
	// The pool holds one connection, so the pragma applies to every statement.
	_, err := db.ExecContext(context.Background(), "PRAGMA foreign_keys = "+mode)
	require.NoError(t, err)
	var got int
	require.NoError(t, db.QueryRowContext(context.Background(), "PRAGMA foreign_keys").Scan(&got))
	want := 0
	if mode == "ON" {
		want = 1
	}
	require.Equal(t, want, got)
}

func rowCount(t *testing.T, db *database.RoutedDB, query string, args ...any) int {
	t.Helper()
	var n int
	require.NoError(t, db.QueryRowContext(context.Background(), query, args...).Scan(&n))
	return n
}

func TestRetain_LeavesNoOrphanedHistory(t *testing.T) {
	for name, mode := range foreignKeyModes {
		t.Run("foreign keys "+name, func(t *testing.T) {
			service, db := testService(t)
			setForeignKeys(t, db, mode)
			ctx := context.Background()
			old := insertPending(t, db, "alice", "old", "public")
			fresh := insertPending(t, db, "alice", "fresh", "public")
			_, err := db.ExecContext(ctx, `UPDATE notifications SET created_at='2020-01-01T00:00:00Z' WHERE id=?`, old)
			require.NoError(t, err)
			for _, id := range []string{old, fresh} {
				// Settled, so the service's own worker leaves it alone.
				_, err = db.ExecContext(ctx, `INSERT INTO notification_events (notification_id, state, reason, created_at) VALUES (?, 'delivered', 'test', '2020-01-01T00:00:00Z')`, id)
				require.NoError(t, err)
				insertProjectionAttempt(t, db, id, "2020-01-01T00:00:00Z")
				_, err = db.ExecContext(ctx, `INSERT INTO receipts (id, notification_id, channel, delivered_at) VALUES (?, ?, 'push', '2020-01-01T00:00:00Z')`, "receipt-"+id, id)
				require.NoError(t, err)
				_, err = db.ExecContext(ctx, `INSERT INTO routing_decisions (id, notification_id, channel, approved, content_mode, reason, created_at) VALUES (?, ?, 'push', 1, 'full', 'test', '2020-01-01T00:00:00Z')`, "decision-"+id, id)
				require.NoError(t, err)
			}

			removed, err := service.Retain(ctx, time.Date(2021, time.January, 1, 0, 0, 0, 0, time.UTC))

			require.NoError(t, err)
			require.EqualValues(t, 1, removed.Notifications)
			for _, table := range []string{"notification_events", "delivery_attempts", "receipts", "routing_decisions"} {
				require.Zero(t, rowCount(t, db, `SELECT COUNT(*) FROM `+table+` WHERE notification_id = ?`, old), "%s rows of the removed notification", table)
				require.Positive(t, rowCount(t, db, `SELECT COUNT(*) FROM `+table+` WHERE notification_id = ?`, fresh), "%s rows of the kept notification", table)
			}
		})
	}
}

func TestRemoveDevice_LeavesNoOrphanedAddressesOrSubscriptions(t *testing.T) {
	for name, mode := range foreignKeyModes {
		t.Run("foreign keys "+name, func(t *testing.T) {
			service, db := testService(t)
			setForeignKeys(t, db, mode)
			ctx := context.Background()
			gone, err := service.UpsertDevice(ctx, "alice", hub.Device{Name: "gone", MachineID: "local"})
			require.NoError(t, err)
			kept, err := service.UpsertDevice(ctx, "alice", hub.Device{Name: "kept", MachineID: "local"})
			require.NoError(t, err)
			for _, device := range []hub.Device{gone, kept} {
				_, err = service.UpsertChannelAddress(ctx, "alice", hub.ChannelAddress{DeviceID: device.ID, Channel: "email", Address: device.Name + "@example.test", ApprovedLabels: []string{"public"}})
				require.NoError(t, err)
			}

			require.NoError(t, service.RemoveDevice(ctx, "alice", gone.ID))

			require.Zero(t, rowCount(t, db, `SELECT COUNT(*) FROM channel_addresses WHERE device_id = ?`, gone.ID))
			require.Equal(t, 1, rowCount(t, db, `SELECT COUNT(*) FROM channel_addresses WHERE device_id = ?`, kept.ID))
			require.ErrorIs(t, service.RemoveDevice(ctx, "alice", gone.ID), sql.ErrNoRows, "removing it again finds nothing")
			require.ErrorIs(t, service.RemoveDevice(ctx, "bob", kept.ID), sql.ErrNoRows, "another recipient cannot remove it")
			require.Equal(t, 1, rowCount(t, db, `SELECT COUNT(*) FROM channel_addresses WHERE device_id = ?`, kept.ID))
		})
	}
}

// A delete that did not name every dependent left rows behind whose notification
// or ask is gone (the live database had six). The retention pass clears them,
// whether or not a connection enforces foreign keys, and leaves everything else.
func TestRetain_RemovesHistoricalOrphansAndIsIdempotent(t *testing.T) {
	service, db := testService(t)
	setForeignKeys(t, db, "OFF")
	ctx := context.Background()
	const stamp = "2026-11-01T00:00:00.000000000Z"
	kept := insertPending(t, db, "alice", "kept", "public")
	gone := insertPending(t, db, "alice", "gone", "public")
	for _, id := range []string{kept, gone} {
		_, err := db.ExecContext(ctx, `INSERT INTO notification_events (notification_id, state, reason, created_at) VALUES (?, 'delivered', 'test', ?)`, id, stamp)
		require.NoError(t, err)
		insertProjectionAttempt(t, db, id, stamp)
		_, err = db.ExecContext(ctx, `INSERT INTO receipts (id, notification_id, channel, delivered_at) VALUES (?, ?, 'push', ?)`, "receipt-"+id, id, stamp)
		require.NoError(t, err)
		_, err = db.ExecContext(ctx, `INSERT INTO routing_decisions (id, notification_id, channel, approved, content_mode, reason, created_at) VALUES (?, ?, 'push', 1, 'full', 'test', ?)`, "decision-"+id, id, stamp)
		require.NoError(t, err)
	}
	// An older delete removed the notification but not what hung from it, and an
	// ask's step outlived the ask.
	_, err := db.ExecContext(ctx, `DELETE FROM notifications WHERE id = ?`, gone)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO escalation_steps (id, ask_id, ordinal, channel, outcome, reason, created_at) VALUES ('step-missing', 'missing-ask', 1, 'push', 'delivered', 'test', ?)`, stamp)
	require.NoError(t, err)
	orphans := rowCount(t, db, `SELECT COUNT(*) FROM escalation_steps WHERE ask_id = 'missing-ask'`)
	keptRows := map[string]int{}
	for _, table := range []string{"notification_events", "delivery_attempts", "receipts", "routing_decisions"} {
		count := rowCount(t, db, `SELECT COUNT(*) FROM `+table+` WHERE notification_id = ?`, gone)
		require.Positive(t, count, "the fixture leaves orphans in %s", table)
		orphans += count
		keptRows[table] = rowCount(t, db, `SELECT COUNT(*) FROM `+table+` WHERE notification_id = ?`, kept)
		require.Positive(t, keptRows[table])
	}

	removed, err := service.Retain(ctx, time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.EqualValues(t, orphans, removed.Orphans)
	require.Zero(t, removed.Asks)
	require.Zero(t, removed.Notifications, "nothing is past the cutoff")
	for _, table := range []string{"notification_events", "delivery_attempts", "receipts", "routing_decisions"} {
		require.Zero(t, rowCount(t, db, `SELECT COUNT(*) FROM `+table+` WHERE notification_id = ?`, gone), "%s orphans", table)
		require.Equal(t, keptRows[table], rowCount(t, db, `SELECT COUNT(*) FROM `+table+` WHERE notification_id = ?`, kept), "%s rows of the kept notification", table)
	}
	require.Zero(t, rowCount(t, db, `SELECT COUNT(*) FROM escalation_steps WHERE ask_id = 'missing-ask'`))
	require.Equal(t, 1, rowCount(t, db, `SELECT COUNT(*) FROM notifications WHERE id = ?`, kept))

	again, err := service.Retain(ctx, time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.Zero(t, again.Orphans, "a second pass finds nothing")
}
