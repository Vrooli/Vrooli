package hub

import (
	"context"
	"testing"
	"time"

	"github.com/vrooli/api-core/database"

	"notification-hub/internal/testutil/filedb"
)

// Maintain runs one pass of the background maintenance loop, as its timer does.
func Maintain(s *Service, ctx context.Context) { s.maintain(ctx) }

// SetUnreferencedBlobLimit changes how many bytes of uploads no ask references
// the Service accepts, so a test need not upload a gibibyte.
func SetUnreferencedBlobLimit(s *Service, limit int64) {
	s.attachments.cfg.Load().unreferencedLimit = limit
}

// FileDSN is the DSN of a fresh file-backed SQLite database for a test that
// needs the real routed pool; see filedb for why it must not be a bare path.
func FileDSN(t *testing.T, name string) string { return filedb.DSN(t, name) }

// OpenTestDB opens a routed SQLite database on a FileDSN.
func OpenTestDB(t *testing.T, name string) *database.RoutedDB { return filedb.Open(t, name) }

// AwaitingCountSQL is the query CountAwaiting runs for the owner.
const AwaitingCountSQL = awaitingCountSQL

// AskListSQL is the page query for the owner of "alice", with a cursor in the
// middle of the list, so a test can read its plan.
func AskListSQL(state string, order AskOrder) (string, []any) {
	columns, _ := orderColumns(order, closingCutoff(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)))
	cursor := &askListCursor{Version: 3, Keys: make([]string, len(columns))}
	for i, column := range columns {
		cursor.Keys[i] = "2026-01-01T00:00:00Z"
		if column.numeric {
			cursor.Keys[i] = "1"
		}
	}
	return askListSQL(askBaseWhere("alice", "", nil, state, ""), nil, nil, columns, cursor, 51)
}

// StoredTime is the text a timestamp is persisted as.
func StoredTime(t time.Time) string { return storedTime(t) }

// NormalizeStoredTimes runs the migration to the fixed-width timestamp form, as boot does.
func NormalizeStoredTimes(s *Service, ctx context.Context) { s.normalizeStoredTimes(ctx) }
