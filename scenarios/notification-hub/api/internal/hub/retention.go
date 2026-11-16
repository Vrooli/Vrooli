package hub

import (
	"context"
	"time"
)

// Retention: how long the Hub keeps its history, and the pass that enforces it.
// The rules are in docs/concepts/DATA.md ("Retention And Deletion").
//
// Deletes name every dependent table. Only some declare a foreign key, and
// whether a connection enforces foreign keys is a driver setting; what a delete
// means must not depend on it.

const (
	// retentionPeriod is how long a notification, and a resolved ask with
	// everything attached to it, is kept.
	retentionPeriod = 90 * 24 * time.Hour
	// retentionInterval is how often the maintenance loop runs the pass.
	retentionInterval = 24 * time.Hour
)

// notificationDependents are the tables whose rows exist only for a notification.
var notificationDependents = []string{"notification_events", "delivery_attempts", "receipts", "routing_decisions", "holds", "suppressions"}

// askDependents are the tables whose rows exist only for an ask, leaves first.
// ask_attachments and ask_attachment_entries are the references that keep an
// attachment blob alive: once they go, the next attachment sweep reclaims it.
var askDependents = []string{"answer_identity", "answer_image_refs", "answers", "escalation_steps", "ask_revisions", "ask_event_ids", "ask_context_requests", "ask_attachment_entries", "ask_attachments"}

// resolvedAskBefore selects the asks Retain removes: resolved ones whose
// resolution is older than the cutoff. An open ask is never removed, however
// old: its deadline and escalations still run.
const resolvedAskBefore = `state NOT IN ('pending', 'escalated') AND COALESCE(NULLIF(resolved_at, ''), updated_at) < ?`

// RetentionResult is what one retention pass removed.
type RetentionResult struct {
	Asks          int64
	Notifications int64
	// Orphans are dependent rows whose notification or ask no longer exists.
	Orphans int64
}

// Retain removes what is older than the cutoff: resolved asks with their
// answers, revisions, escalations, context requests and attachments, then
// notifications created before it with their events, attempts, receipts and
// routing records. A notification an ask still references stays until its ask
// goes, so an open ask never loses its notification. Attachment blobs that lose
// their last reference are reclaimed by SweepAttachmentBlobs.
//
// The pass also removes orphans: dependent rows left behind by a delete that did
// not name every dependent, whose notification or ask is already gone. It is the
// same rule applied to history, and idempotent: with no orphans it deletes nothing.
func (s *Service) Retain(ctx context.Context, before time.Time) (RetentionResult, error) {
	cutoff := storedTime(before)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return RetentionResult{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var result RetentionResult
	for _, table := range askDependents {
		if _, err := tx.ExecContext(ctx, `DELETE FROM `+table+` WHERE ask_id IN (SELECT id FROM asks WHERE `+resolvedAskBefore+`)`, cutoff); err != nil {
			return RetentionResult{}, err
		}
	}
	removed, err := tx.ExecContext(ctx, `DELETE FROM asks WHERE `+resolvedAskBefore, cutoff)
	if err != nil {
		return RetentionResult{}, err
	}
	if result.Asks, err = removed.RowsAffected(); err != nil {
		return RetentionResult{}, err
	}

	const stale = `SELECT id FROM notifications WHERE created_at < ? AND id NOT IN (SELECT notification_id FROM asks)`
	for _, table := range notificationDependents {
		if _, err := tx.ExecContext(ctx, `DELETE FROM `+table+` WHERE notification_id IN (`+stale+`)`, cutoff); err != nil {
			return RetentionResult{}, err
		}
	}
	removed, err = tx.ExecContext(ctx, `DELETE FROM notifications WHERE id IN (`+stale+`)`, cutoff)
	if err != nil {
		return RetentionResult{}, err
	}
	if result.Notifications, err = removed.RowsAffected(); err != nil {
		return RetentionResult{}, err
	}

	for _, orphan := range []struct {
		tables []string
		key    string
		owner  string
	}{
		{notificationDependents, "notification_id", "notifications"},
		{askDependents, "ask_id", "asks"},
	} {
		for _, table := range orphan.tables {
			removed, err := tx.ExecContext(ctx, `DELETE FROM `+table+` WHERE `+orphan.key+` NOT IN (SELECT id FROM `+orphan.owner+`)`)
			if err != nil {
				return RetentionResult{}, err
			}
			count, err := removed.RowsAffected()
			if err != nil {
				return RetentionResult{}, err
			}
			result.Orphans += count
		}
	}
	return result, tx.Commit()
}

// retentionDue reports whether the daily pass should run now, and claims the
// slot so concurrent callers run it once. The first call after a start is due.
func (s *Service) retentionDue() bool {
	now := s.clock.Now().UnixNano()
	last := s.lastRetention.Load()
	if last != 0 && now-last < int64(retentionInterval) {
		return false
	}
	return s.lastRetention.CompareAndSwap(last, now)
}

// runRetention is the maintenance loop's daily step.
func (s *Service) runRetention(ctx context.Context) {
	if !s.retentionDue() {
		return
	}
	result, err := s.Retain(ctx, s.clock.Now().Add(-retentionPeriod))
	if err != nil {
		s.log.Printf("retention: %v", err)
		return
	}
	if result.Asks > 0 || result.Notifications > 0 {
		s.log.Printf("retention: removed %d asks and %d notifications older than %d days", result.Asks, result.Notifications, int(retentionPeriod/(24*time.Hour)))
	}
	if result.Orphans > 0 {
		s.log.Printf("retention: removed %d orphaned rows whose notification or ask no longer exists", result.Orphans)
	}
}
