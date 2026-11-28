package integrations

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"

	"notification-hub/internal/modules"
	"notification-hub/internal/testutil/hubtest"
)

// A context link the operator would tap must pass hub.ValidateContextURL. On the
// event path a link that fails is dropped, never the notification, and the drop is
// counted against the producing scenario.

func dropsFor(source string) uint64 { return DroppedContextLinks()[source] }

func TestEventContextURLKeepsEveryFormALiveProducerSends(t *testing.T) {
	before := dropsFor("web-console")
	for _, link := range []string{
		"", "/planner/today", "/asks/ask-1?from=notice",
		"https://agent-manager.example.test/api/v1/authorizations/native/approve",
		"http://127.0.0.1:18227/runs/1", "https://host.example:8443/review?x=1#top",
	} {
		require.Equal(t, link, eventContextURL("web-console", "x.notification.v1", link))
	}
	require.Equal(t, "https://trimmed.example/a", eventContextURL("web-console", "x.notification.v1", "  https://trimmed.example/a "))
	require.Equal(t, before, dropsFor("web-console"), "a valid link is never counted as dropped")
}

func TestEventContextURLDropsAndCountsAnUnsafeLink(t *testing.T) {
	for _, link := range []string{
		"javascript:alert(1)", "data:text/html;base64,PHNjcmlwdD4=", "//evil.example/x",
		"https://trusted.example@evil.example/login", "evil.example/x", "file:///etc/passwd",
	} {
		before := dropsFor("planner")
		require.Empty(t, eventContextURL("planner", "planner.reminder.notification.v1", link), link)
		require.Equal(t, before+1, dropsFor("planner"), "%q is counted against its producer", link)
	}
}

func TestEveryEventBuilderDropsAnUnsafeLinkAndKeepsTheNotice(t *testing.T) {
	const bad = "javascript:alert(1)"
	before := dropsFor("planner")

	facts, ok := parseNotificationFacts(json.RawMessage(`{"body":"remember","context_url":"` + bad + `"}`))
	require.True(t, ok)
	in := notificationSendInput("alice", "planner", "planner.reminder.notification.v1", "evt-1", facts)
	require.Equal(t, "remember", in.Body, "the notice survives")
	require.Empty(t, in.ContextURL)

	encoded, err := json.Marshal(map[string]any{
		"question": "Apply this change?", "options": []map[string]string{{"key": "apply", "label": "Apply"}, {"key": "reject", "label": "Reject"}},
		"recommended": "apply", "big_picture": "Purpose", "context_url": bad,
	})
	require.NoError(t, err)
	spec, err := DecisionAskSpec("alice", "planner", "planner.change.decision_requested.v1", "evt-2", encoded, time.Now().UTC())
	require.NoError(t, err)
	require.Equal(t, "Apply this change?", spec.Question, "the ask survives")
	require.Empty(t, spec.ContextURL)

	require.Equal(t, before+2, dropsFor("planner"))
}

func TestHeartbeatNoticesDropAnUnsafeLinkAndKeepTheNotice(t *testing.T) {
	before := dropsFor("prompt-manager")
	attempt := json.RawMessage(`{"teamId":"t","agentId":"a","status":"failed","attentionEligible":true,"consecutiveFailures":3,"failureEpisodeId":"ep-1","recoveryOutcome":"not_requested","reviewUrl":"javascript:alert(1)"}`)
	in, ok, err := heartbeatAttemptFYI(HeartbeatAttemptEventType, "prompt-manager", "evt-1", attempt)
	require.NoError(t, err)
	require.True(t, ok, "the notice survives")
	require.Empty(t, in.ContextURL)

	owner := json.RawMessage(`{"teamId":"t","agentId":"a","ownerObligationId":"o-1","retainedOwnerState":"owner_unreachable","reviewUrl":"//evil.example/x"}`)
	in, ok, err = heartbeatAttemptFYI(HeartbeatOwnerStateEventType, "prompt-manager", "evt-2", owner)
	require.NoError(t, err)
	require.True(t, ok, "the notice survives")
	require.Empty(t, in.ContextURL)
	require.Equal(t, before+2, dropsFor("prompt-manager"))
}

func TestAnEventWithAnUnsafeLinkIsStoredWithoutTheLinkNotRejected(t *testing.T) {
	primary := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), primary, modules.AllSchemas()...))
	db := database.NewFromPrimary(primary)
	service := hubtest.New(t, db, nil, nil)
	const secret = "event-secret"
	handler := EventWebhook(service, secret)
	before := dropsFor("planner")

	for id, link := range map[string]string{"evt-bad": "javascript:alert(1)", "evt-good": "/planner/today"} {
		body := []byte(`{"event_id":"` + id + `","event_type":"planner.reminder.notification.v1","source_scenario":"planner","payload":{"body":"Stretch","context_url":"` + link + `"}}`)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, signedEventRequest(t, secret, id, body))
		require.Equal(t, http.StatusAccepted, recorder.Code, "%s: a bad link must not lose the notice: %s", id, recorder.Body.String())
	}
	stored := func(id string) (body, link string) {
		require.NoError(t, db.QueryRowContext(context.Background(), `SELECT body, context_url FROM notifications WHERE idempotency_key = ?`, id).Scan(&body, &link))
		return body, link
	}
	body, link := stored("evt-bad")
	require.Equal(t, "Stretch", body)
	require.Empty(t, link, "the unsafe link is not stored")
	_, link = stored("evt-good")
	require.Equal(t, "/planner/today", link)
	require.Equal(t, before+1, dropsFor("planner"))
}
