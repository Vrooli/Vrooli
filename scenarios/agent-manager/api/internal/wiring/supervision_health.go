package wiring

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"agent-manager/internal/adapters/database"

	"github.com/vrooli/api-core/health"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// supervisionStuckFailureThreshold is the consecutive failed-evaluation count
// on a single active cohort watch that flips supervision liveness to degraded.
// Unavailable decisions back off to roughly hourly, so 24 is about a day of a
// watch never completing a decision — long enough to be a real stall, short
// enough to catch the month-long silence that went unnoticed in 2026-09.
const supervisionStuckFailureThreshold = 24

// supervisionLivenessHealthChecker reports degraded when any active cohort
// watch has accumulated a large consecutive-failure streak, i.e. the
// supervision consumer (agent-manager.supervision-evaluate) is no longer
// completing decisions. It is registered Optional so it degrades — never
// fails — the scenario, and it reads only existing supervision tables.
//
// This exists because supervision-evaluate broke on 2026-09-07 and nothing
// surfaced it for a month: unavailable decisions warn once then drop to debug,
// and the portfolio's last_seen counts attempts, not successes. A degraded
// /health signal makes the next such stall visible within a day.
// rowQuerier is the single read the canary needs; both the adapters DB and a
// raw *sql.DB satisfy it, so the check logic is testable without the adapter.
type rowQuerier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func supervisionLivenessHealthChecker(db *database.DB) health.Checker {
	return health.Func("supervision_liveness", func(ctx context.Context) error {
		if db == nil {
			return nil
		}
		return supervisionLivenessCheck(ctx, db)
	})
}

func supervisionLivenessCheck(ctx context.Context, db rowQuerier) error {
	const q = `SELECT f.watch_id, f.failures
		FROM cohort_watch_evaluation_failures f
		JOIN cohort_watches w ON w.watch_id = f.watch_id
		WHERE w.status = ?
		ORDER BY f.failures DESC
		LIMIT 1`
	var watchID string
	var failures int
	err := db.QueryRowContext(ctx, q, int32(domainpb.WatchStatus_WATCH_STATUS_ACTIVE)).Scan(&watchID, &failures)
	if errors.Is(err, sql.ErrNoRows) {
		return nil // no active watches with a failure streak
	}
	if err != nil {
		// A query error (e.g. the tables are absent on a fresh install) must
		// not mask real health; the canary is advisory.
		return nil
	}
	if failures >= supervisionStuckFailureThreshold {
		return fmt.Errorf("supervision watch %s has %d consecutive failed evaluations (threshold %d): the supervision consumer is not completing decisions; inspect with `agent-manager watch get %s`", watchID, failures, supervisionStuckFailureThreshold, watchID)
	}
	return nil
}
