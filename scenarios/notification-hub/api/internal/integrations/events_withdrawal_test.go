package integrations

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"notification-hub/internal/hub"
	"notification-hub/internal/modules"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
)

func TestEventWebhookWithdrawsOnlyExactDecisionAskWithoutAnswering(t *testing.T) {
	primary := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), primary, modules.AllSchemas()...))
	db := database.NewFromPrimary(primary)
	service := hub.New(db, nil, nil)
	spec := hub.AskSpec{
		Recipient: "operator", Question: "Apply the run changes?",
		Options:     []hub.AskOption{{Key: "approve", Label: "Approve"}, {Key: "reject", Label: "Reject"}},
		Recommended: "approve", Reversible: true, Deadline: time.Now().Add(time.Hour),
		SensitivityLabel: "private", IdempotencyKey: "exact-request-ref", Source: "agent-manager",
		SourceEventType: "agent_manager.run.decision_requested.v1",
	}
	ask, _, err := service.AskWithSpec(context.Background(), spec)
	require.NoError(t, err)
	secret := "withdrawal-test-secret"
	handler := EventWebhook(service, secret)
	post := func(eventID, source, sourceType, ref, reason string) *httptest.ResponseRecorder {
		body := []byte(`{"event_type":"agent_manager.run.decision_withdrawn.v1","source_scenario":"` + source + `","payload":{"requester_ref":"` + ref + `","source_event_type":"` + sourceType + `","reason":"` + reason + `"}}`)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, signedEventRequest(t, secret, eventID, body))
		return recorder
	}

	wrongType := post("withdraw-wrong-type", "agent-manager", "agent_manager.other.decision_requested.v1", spec.IdempotencyKey, "verified_empty_diff")
	require.Equal(t, http.StatusBadRequest, wrongType.Code)
	stillOpen, err := service.GetAsk(context.Background(), ask.ID)
	require.NoError(t, err)
	require.Equal(t, hub.AskPending, stillOpen.State)
	wrongSource := post("withdraw-wrong-source", "other-producer", spec.SourceEventType, spec.IdempotencyKey, "verified_empty_diff")
	require.Equal(t, http.StatusBadRequest, wrongSource.Code)
	wrongRef := post("withdraw-wrong-ref", "agent-manager", spec.SourceEventType, "not-the-request-ref", "verified_empty_diff")
	require.Equal(t, http.StatusAccepted, wrongRef.Code)
	stillOpen, err = service.GetAsk(context.Background(), ask.ID)
	require.NoError(t, err)
	require.Equal(t, hub.AskPending, stillOpen.State)

	exact := post("withdraw-exact", "agent-manager", spec.SourceEventType, spec.IdempotencyKey, "verified_empty_diff")
	require.Equal(t, http.StatusAccepted, exact.Code)
	withdrawn, err := service.GetAsk(context.Background(), ask.ID)
	require.NoError(t, err)
	require.Equal(t, hub.AskWithdrawn, withdrawn.State)
	require.Empty(t, withdrawn.Answer)
	require.Empty(t, withdrawn.AnsweredBy)
	duplicate := post("withdraw-duplicate", "agent-manager", spec.SourceEventType, spec.IdempotencyKey, "verified_empty_diff")
	require.Equal(t, http.StatusAccepted, duplicate.Code)
	withdrawn, err = service.GetAsk(context.Background(), ask.ID)
	require.NoError(t, err)
	require.Equal(t, hub.AskWithdrawn, withdrawn.State)

	newSpec := spec
	newSpec.IdempotencyKey = "newer-review-generation"
	newAsk, _, err := service.AskWithSpec(context.Background(), newSpec)
	require.NoError(t, err)
	superseded := post("withdraw-superseded", "agent-manager", spec.SourceEventType, newSpec.IdempotencyKey, "superseded_by_new_review")
	require.Equal(t, http.StatusAccepted, superseded.Code)
	newAsk, err = service.GetAsk(context.Background(), newAsk.ID)
	require.NoError(t, err)
	require.Equal(t, hub.AskWithdrawn, newAsk.State)
	require.Contains(t, newAsk.Reason, "newer review request")
}

func TestEventWebhookWithdrawalBeforeDelayedRequestCreatesNoAskOrNotification(t *testing.T) {
	primary := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), primary, modules.AllSchemas()...))
	db := database.NewFromPrimary(primary)
	service := hub.New(db, nil, nil)
	secret := "withdrawal-ordering-secret"
	handler := EventWebhook(service, secret)
	post := func(eventID, eventType string, payload any) *httptest.ResponseRecorder {
		encoded, err := json.Marshal(map[string]any{"event_type": eventType, "source_scenario": "agent-manager", "payload": payload})
		require.NoError(t, err)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, signedEventRequest(t, secret, eventID, encoded))
		return recorder
	}
	requestRef := "request-delivered-after-withdrawal"
	withdrawal := post("withdrawal-arrives-first", DecisionWithdrawalEventType, DecisionWithdrawal{
		RequesterRef: requestRef, SourceEventType: RunApprovalDecisionEventType, Reason: "verified_empty_diff",
	})
	require.Equal(t, http.StatusAccepted, withdrawal.Code)
	request := post(requestRef, RunApprovalDecisionEventType, map[string]any{
		"question":    "Apply this run?",
		"options":     []map[string]string{{"key": "approve", "label": "Approve"}, {"key": "reject", "label": "Reject"}},
		"recommended": "approve",
		"reversible":  true,
	})
	require.Equal(t, http.StatusAccepted, request.Code)
	var response map[string]any
	require.NoError(t, json.Unmarshal(request.Body.Bytes(), &response))
	require.Equal(t, true, response["withdrawn"])
	require.Equal(t, requestRef, response["event_id"])
	for query := range map[string]bool{"asks": true, "notifications": true, "answers": true} {
		var count int
		require.NoError(t, db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM `+query).Scan(&count))
		require.Zero(t, count, query+" must remain absent after the delayed request is acknowledged")
	}
	var state, sourceType string
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT state, source_event_type FROM decision_request_lifecycle WHERE source='agent-manager' AND requester_ref=?`, requestRef).Scan(&state, &sourceType))
	require.Equal(t, "withdrawn", state)
	require.Equal(t, RunApprovalDecisionEventType, sourceType)
}

func TestEnsureSchemasAddsWithdrawalLedgerWithoutChangingExistingAsks(t *testing.T) {
	primary := databasetest.NewSQLite(t)
	ctx := context.Background()
	require.NoError(t, database.EnsureSchemas(ctx, primary, modules.AllSchemas()...))
	db := database.NewFromPrimary(primary)
	service := hub.New(db, nil, nil)
	spec := hub.AskSpec{
		Recipient: "operator", Question: "Apply the run changes?",
		Options:     []hub.AskOption{{Key: "approve", Label: "Approve"}, {Key: "reject", Label: "Reject"}},
		Recommended: "approve", Reversible: true, Deadline: time.Now().Add(time.Hour),
		SensitivityLabel: "private", IdempotencyKey: "legacy-request-ref", Source: "agent-manager",
		SourceEventType: RunApprovalDecisionEventType,
	}
	created, notification, err := service.AskWithSpec(ctx, spec)
	require.NoError(t, err)

	// Model a pre-feature database: the existing Ask and notification rows are
	// retained, while the newly introduced additive owner table is absent.
	_, err = primary.ExecContext(ctx, `DROP TABLE decision_request_lifecycle`)
	require.NoError(t, err)
	require.NoError(t, database.EnsureSchemas(ctx, primary, modules.AllSchemas()...))

	retained, err := service.GetAsk(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, hub.AskPending, retained.State)
	require.Equal(t, created.ID, retained.ID)
	retainedNotification, _, err := service.Get(ctx, notification.ID)
	require.NoError(t, err)
	require.Equal(t, notification.ID, retainedNotification.ID)

	// Replaying the same exact request after schema ensure remains idempotent.
	// The legacy Ask is not rewritten just to populate new lifecycle state; a
	// later exact producer withdrawal can bind and resolve it lazily.
	replayed, replayedNotification, err := service.AskWithSpec(ctx, spec)
	require.NoError(t, err)
	require.Equal(t, created.ID, replayed.ID)
	require.Equal(t, notification.ID, replayedNotification.ID)
	var asks, notifications, lifecycleRows int
	require.NoError(t, primary.QueryRowContext(ctx, `SELECT COUNT(*) FROM asks WHERE source=? AND requester_ref=?`, spec.Source, spec.IdempotencyKey).Scan(&asks))
	require.NoError(t, primary.QueryRowContext(ctx, `SELECT COUNT(*) FROM notifications WHERE id=?`, notification.ID).Scan(&notifications))
	require.NoError(t, primary.QueryRowContext(ctx, `SELECT COUNT(*) FROM decision_request_lifecycle WHERE source=? AND requester_ref=?`, spec.Source, spec.IdempotencyKey).Scan(&lifecycleRows))
	require.Equal(t, 1, asks)
	require.Equal(t, 1, notifications)
	require.Zero(t, lifecycleRows, "schema ensure must not synthesize lifecycle history for preexisting Asks")

	changed, err := service.WithdrawDecisionAsk(ctx, hub.DecisionWithdrawal{
		Source: spec.Source, RequesterRef: spec.IdempotencyKey, SourceEventType: spec.SourceEventType, Reason: "verified_empty_diff",
	})
	require.NoError(t, err)
	require.True(t, changed)
	retained, err = service.GetAsk(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, hub.AskWithdrawn, retained.State)
	require.NoError(t, primary.QueryRowContext(ctx, `SELECT COUNT(*) FROM decision_request_lifecycle WHERE source=? AND requester_ref=?`, spec.Source, spec.IdempotencyKey).Scan(&lifecycleRows))
	require.Equal(t, 1, lifecycleRows, "first real lifecycle event records exact legacy Ask provenance without rewriting prior data")
}

func TestEnsureDecisionWithdrawalSubscriptionUsesExactProducerRoute(t *testing.T) {
	var created map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`[]`))
			return
		}
		require.Equal(t, http.MethodPost, r.Method)
		require.NoError(t, json.NewDecoder(r.Body).Decode(&created))
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()
	require.NoError(t, EnsureDecisionWithdrawalSubscription(context.Background(), server.URL, "https://hub.test/events"))
	require.Equal(t, "notification-hub-decision-withdrawals", created["name"])
	require.Equal(t, DecisionWithdrawalPattern, created["event_pattern"])
}
