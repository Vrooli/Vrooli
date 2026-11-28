package hub_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	hub "notification-hub/internal/hub"
)

const day = 24 * time.Hour

// askRetentionTables are the tables whose rows exist only for an ask.
var askRetentionTables = []string{"answers", "answer_identity", "escalation_steps", "ask_revisions", "ask_event_ids", "ask_context_requests", "ask_attachments", "ask_attachment_entries"}

// fillAsk gives an ask a row in every table that belongs to an ask.
func (e *attachmentEnv) fillAsk(t *testing.T, askID, answer string) {
	t.Helper()
	ctx := context.Background()
	_, err := e.service.RequestContext(ctx, askID, "alice", "What changed?")
	require.NoError(t, err)
	_, err = e.service.AnswerAsk(ctx, hub.AnswerInput{AskID: askID, Answer: answer, Actor: "alice"})
	require.NoError(t, err)
	_, err = e.db.ExecContext(ctx, `INSERT INTO escalation_steps (id, ask_id, ordinal, channel, outcome, reason, created_at) VALUES (?, ?, 1, 'push', 'delivered', 'test', ?)`, "step-"+askID, askID, e.now.Format(time.RFC3339Nano))
	require.NoError(t, err)
}

// askRows counts an ask's rows in every table that belongs to an ask.
func (e *attachmentEnv) askRows(t *testing.T, askID string) map[string]int {
	t.Helper()
	counts := map[string]int{"asks": rowCount(t, e.db, `SELECT COUNT(*) FROM asks WHERE id = ?`, askID)}
	for _, table := range askRetentionTables {
		counts[table] = rowCount(t, e.db, `SELECT COUNT(*) FROM `+table+` WHERE ask_id = ?`, askID)
	}
	return counts
}

func (e *attachmentEnv) notificationExists(t *testing.T, id string) bool {
	t.Helper()
	return rowCount(t, e.db, `SELECT COUNT(*) FROM notifications WHERE id = ?`, id) == 1
}

// filledAsk makes a subject ask with a revision, a file and a folder attachment,
// then resolves it, so it has a row in every table that belongs to an ask.
func (e *attachmentEnv) filledAsk(t *testing.T, key, shared string) hub.AskRecord {
	t.Helper()
	first := e.ask(t, key, e.file(t, "plan.md", "plan "+key), e.file(t, "shared.md", shared),
		e.folder(t, "report", map[string]string{"a.md": "a " + key}))
	revised := subjectSpec(e.now, key+"-rev")
	revised.AcceptsContextRequests = true
	revised.Question = "Backups to the Elements drive are still blocked. Retry or skip?"
	revised.Attachments = []hub.AskAttachment{
		e.file(t, "plan.md", "plan "+key), e.file(t, "shared.md", shared),
		e.folder(t, "report", map[string]string{"a.md": "a " + key}),
	}
	result, err := e.service.AskDecision(context.Background(), revised, "")
	require.NoError(t, err)
	require.Equal(t, hub.AskUpdated, result.Outcome)
	require.Equal(t, first.Ask.ID, result.Ask.ID)
	e.fillAsk(t, first.Ask.ID, "retry")
	return result.Ask
}

func TestRetain_RemovesAResolvedAskWithEverythingBelongingToIt(t *testing.T) {
	for name, mode := range foreignKeyModes {
		t.Run("foreign keys "+name, func(t *testing.T) {
			env := newAttachmentEnv(t)
			setForeignKeys(t, env.db, mode)
			ctx := context.Background()
			old := env.filledAsk(t, "evt-old", "shared bytes")
			for table, count := range env.askRows(t, old.ID) {
				require.Positive(t, count, "the fixture gives the ask a row in %s", table)
			}

			env.advance(t, 100*day)
			fresh := env.filledAsk(t, "evt-fresh", "shared bytes")

			removed, err := env.service.Retain(ctx, env.now.Add(-90*day))

			require.NoError(t, err)
			require.EqualValues(t, 1, removed.Asks)
			for table, count := range env.askRows(t, old.ID) {
				require.Zero(t, count, "%s rows of the removed ask", table)
			}
			require.False(t, env.notificationExists(t, old.NotificationID), "the removed ask's notification goes with it")
			for table, count := range env.askRows(t, fresh.ID) {
				require.Positive(t, count, "%s rows of the kept ask", table)
			}
			require.True(t, env.notificationExists(t, fresh.NotificationID))
		})
	}
}

func TestRetain_NeverRemovesAnOpenAskOrItsNotification(t *testing.T) {
	env := newAttachmentEnv(t)
	ctx := context.Background()
	open := env.ask(t, "evt-open", env.file(t, "plan.md", "open plan"))

	env.advance(t, 200*day)
	removed, err := env.service.Retain(ctx, env.now.Add(-90*day))

	require.NoError(t, err)
	require.Zero(t, removed.Asks)
	require.Zero(t, removed.Notifications)
	require.Equal(t, 1, env.askRows(t, open.Ask.ID)["ask_attachments"])
	require.True(t, env.notificationExists(t, open.Ask.NotificationID), "an open ask still needs its notification")
}

func TestRetain_AnAskResolvedRecentlyKeepsItsOlderNotification(t *testing.T) {
	env := newAttachmentEnv(t)
	ctx := context.Background()
	result := env.ask(t, "evt-slow", env.file(t, "plan.md", "slow plan"))

	env.advance(t, 100*day)
	_, err := env.service.AnswerAsk(ctx, hub.AnswerInput{AskID: result.Ask.ID, Answer: "retry", Actor: "alice"})
	require.NoError(t, err)
	removed, err := env.service.Retain(ctx, env.now.Add(-90*day))

	require.NoError(t, err)
	require.Zero(t, removed.Asks, "the ask was resolved today; its age is the age of its resolution")
	require.Zero(t, removed.Notifications)
	require.True(t, env.notificationExists(t, result.Ask.NotificationID))
}

func TestRetain_BlobsOfRemovedAsksAreReclaimedAndSharedOnesAreKept(t *testing.T) {
	env := newAttachmentEnv(t)
	ctx := context.Background()
	old := env.filledAsk(t, "evt-old", "shared bytes")
	oldOnly := []string{sha("plan evt-old"), sha("a evt-old")}
	shared := sha("shared bytes")
	require.Equal(t, 3, env.blobCount(t))

	env.advance(t, 100*day)
	fresh := decisionSpec(env.now, "plain-fresh")
	fresh.Attachments = []hub.AskAttachment{env.file(t, "shared.md", "shared bytes")}
	_, err := env.service.AskDecision(ctx, fresh, "")
	require.NoError(t, err)

	_, err = env.service.Retain(ctx, env.now.Add(-90*day))
	require.NoError(t, err)
	swept, err := env.service.SweepAttachmentBlobs(ctx)

	require.NoError(t, err)
	require.Equal(t, 2, swept, "the removed ask's own files are reclaimed")
	require.Equal(t, 1, env.blobCount(t))
	missing, err := env.service.StatAttachmentBlobs(ctx, append(oldOnly, shared))
	require.NoError(t, err)
	require.ElementsMatch(t, oldOnly, missing, "only the file another ask still uses survives")
	require.Equal(t, 0, env.askRows(t, old.ID)["asks"])
}

func TestMaintain_AppliesRetentionOncePerDayAndFreesBlobsInTheSamePass(t *testing.T) {
	env := newAttachmentEnv(t)
	ctx := context.Background()
	first := env.filledAsk(t, "evt-first", "first shared")
	env.advance(t, 12*time.Hour)
	second := env.filledAsk(t, "evt-second", "second shared")

	// Ninety days and six hours in: only the first ask is past the cutoff.
	env.advance(t, 90*day+6*time.Hour-12*time.Hour)
	hub.Maintain(env.service, ctx)
	require.Zero(t, env.askRows(t, first.ID)["asks"], "the daily pass ran")
	require.Equal(t, 1, env.askRows(t, second.ID)["asks"])
	require.Equal(t, 3, env.blobCount(t), "the first ask's blobs were swept in the same pass; the second's are kept")

	// Twelve hours later the second ask is past the cutoff, but the pass ran today.
	env.advance(t, 12*time.Hour)
	hub.Maintain(env.service, ctx)
	require.Equal(t, 1, env.askRows(t, second.ID)["asks"], "the pass runs once a day, not every minute")

	env.advance(t, 13*time.Hour)
	hub.Maintain(env.service, ctx)
	require.Zero(t, env.askRows(t, second.ID)["asks"], "the next day's pass removes it")
	require.Zero(t, env.blobCount(t))
}
