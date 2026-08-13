package homeintegration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"

	db "github.com/vrooli/api-core/databasetest"
)

func TestSQLiteRepositoryPersistsEventsAndInvocations(t *testing.T) {
	// [REQ:NM-P0-007] Home Automation action/event audit data is persisted in domain-owned storage.
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(Schema), apidb.SchemaProviderFunc(Schema)))
	repo := NewSQLiteRepository(d)

	event, err := repo.SaveEvent(context.Background(), Event{ID: "event-1", Type: "network.device.new_seen", Summary: "New device observed with redacted details.", OccurredAt: fixedNow(), PublishStatus: "pending"})
	require.NoError(t, err)
	event, err = repo.UpdateEventPublish(context.Background(), event.ID, "publish_failed", "home automation unavailable")
	require.NoError(t, err)
	require.Equal(t, "publish_failed", event.PublishStatus)

	_, err = repo.SaveInvocation(context.Background(), Invocation{ID: "invoke-1", ActionName: "network.health.run", Status: "accepted", Approved: false, Message: "accepted", Params: map[string]string{"device": "[redacted]"}, EventID: event.ID, CreatedAt: fixedNow()})
	require.NoError(t, err)

	events, err := repo.ListEvents(context.Background(), 10)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, "network.device.new_seen", events[0].Type)
	require.Equal(t, "home automation unavailable", events[0].PublishError)
}
