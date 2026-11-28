package hub_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	hub "notification-hub/internal/hub"
	"notification-hub/internal/testutil/hubtest"
)

func TestTemplateTitlesPersistAndBackfillOpenGenericTitles(t *testing.T) {
	service, db := testService(t)
	ctx := context.Background()
	spec := hub.AskSpec{Recipient: "alice", Source: "alice", Kind: "status", Question: "Portfolio update", SensitivityLabel: "private", IdempotencyKey: "title-test", TemplateName: "daily-briefing", TemplateData: map[string]any{"meta": map[string]any{"title": "Portfolio status for 2026-11-16"}}}
	created, err := service.AskDecision(ctx, spec, "")
	require.NoError(t, err)
	require.Equal(t, "Daily briefing · Mon 16 Nov", created.Ask.Title)
	_, err = db.ExecContext(ctx, `UPDATE notifications SET title='Status update' WHERE id=?`, created.Ask.NotificationID)
	require.NoError(t, err)
	// A new service performs the same backfill as startup on existing storage.
	restarted := hubtest.New(t, db, nil, nil)
	got, err := restarted.GetAsk(ctx, created.Ask.ID)
	require.NoError(t, err)
	require.Equal(t, "Daily briefing · Mon 16 Nov", got.Title)
	_, err = db.ExecContext(ctx, `UPDATE notifications SET title='Owner title' WHERE id=?`, created.Ask.NotificationID)
	require.NoError(t, err)
	restarted = hubtest.New(t, db, nil, nil)
	got, err = restarted.GetAsk(ctx, created.Ask.ID)
	require.NoError(t, err)
	require.Equal(t, "Owner title", got.Title)
}
