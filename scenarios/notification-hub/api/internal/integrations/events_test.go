package integrations

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"notification-hub/internal/hub"
	"notification-hub/internal/modules"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/storage"
)

// [REQ:NOTIFICA-P1-003]
func TestEventWebhookAcceptsDurableEventsPayload(t *testing.T) {
	primary := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), primary, modules.AllSchemas()...))
	service := hub.New(database.NewFromPrimary(primary), nil, nil)
	secret := "event-secret"
	body := []byte(`{"event_id":"evt-1","event_type":"job.completed.v1","source_scenario":"worker","payload":{"title":"Job complete","body":"done","sensitivity_label":"public"}}`)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/events", bytes.NewReader(body))
	req.Header.Set("X-Vrooli-Events-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	req.Header.Set("X-Vrooli-Events-Event-ID", "evt-1")
	recorder := httptest.NewRecorder()
	EventWebhook(service, secret).ServeHTTP(recorder, req)
	require.Equal(t, http.StatusAccepted, recorder.Code)
	var response map[string]string
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, "evt-1", response["event_id"])
}

func TestEventWebhookRejectsProducerOwnedCopy(t *testing.T) {
	primary := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), primary, modules.AllSchemas()...))
	service := hub.New(database.NewFromPrimary(primary), nil, nil)
	secret := "event-secret"
	body := []byte(`{"event_id":"evt-copy","event_type":"incident.opened.v1","title":"producer copy","payload":{"severity":"critical"}}`)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/events", bytes.NewReader(body))
	req.Header.Set("X-Vrooli-Events-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	recorder := httptest.NewRecorder()
	EventWebhook(service, secret).ServeHTTP(recorder, req)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), ErrProducerOwnedCopy.Error())
}

func TestEventWebhookRecordsCriticalUnroutableDeliveryAttempt(t *testing.T) {
	primary := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), primary, modules.AllSchemas()...))
	db := database.NewFromPrimary(primary)
	service := hub.New(db, nil, nil)
	secret := "event-secret"
	body := []byte(`{"event_id":"evt-critical-host","event_type":"incident.opened.v1","source_scenario":"vrooli-autoheal","payload":{"incident_id":"inc-host","severity":"critical","status":"open","source_check_id":"host-kernel-module-drift"}}`)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/events", bytes.NewReader(body))
	req.Header.Set("X-Vrooli-Events-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	req.Header.Set("X-Vrooli-Events-Event-ID", "evt-critical-host")
	recorder := httptest.NewRecorder()
	EventWebhook(service, secret).ServeHTTP(recorder, req)
	require.Equal(t, http.StatusAccepted, recorder.Code)
	var response map[string]string
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Eventually(t, func() bool {
		var outcome, reason, sensitivity string
		err := db.QueryRowContext(context.Background(), `SELECT a.outcome, a.reason, n.sensitivity_label FROM delivery_attempts a JOIN notifications n ON n.id = a.notification_id WHERE n.idempotency_key = ?`, "evt-critical-host").Scan(&outcome, &reason, &sensitivity)
		return err == nil && outcome == "unroutable" && reason != "" && sensitivity == "critical"
	}, time.Second, 10*time.Millisecond)
}

func TestEventWebhookCreatesDurableApprovalAskForRemediationFactEvent(t *testing.T) {
	primary := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), primary, modules.AllSchemas()...))
	db := database.NewFromPrimary(primary)
	service := hub.New(db, nil, nil)
	secret := "event-secret"
	body := []byte(`{"event_id":"evt-approval-ask","event_type":"incident.remediation_approval_requested.v1","source_scenario":"vrooli-autoheal","payload":{"incident_id":"inc-approval","incident_title":"Kernel module drift","severity":"critical","status":"open","message":"operator review required","candidate_id":"candidate-1","candidate_title":"Review coupling"}}`)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/events", bytes.NewReader(body))
	req.Header.Set("X-Vrooli-Events-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	req.Header.Set("X-Vrooli-Events-Event-ID", "evt-approval-ask")
	recorder := httptest.NewRecorder()
	EventWebhook(service, secret).ServeHTTP(recorder, req)
	require.Equal(t, http.StatusAccepted, recorder.Code)
	var response map[string]string
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.NotEmpty(t, response["ask_id"])
	var question, state string
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT question, state FROM asks WHERE id = ?`, response["ask_id"]).Scan(&question, &state))
	require.Equal(t, "pending", state)
	require.Contains(t, question, "Review coupling")
	require.NotContains(t, question, "sensitivity_label")
}

func TestEnsureEventSubscriptionIsIdempotent(t *testing.T) {
	var posts int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`[{"id":1,"event_pattern":"**","delivery_target":"https://hub.test/events","enabled":true}]`))
			return
		}
		posts++
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()
	require.NoError(t, EnsureEventSubscription(context.Background(), server.URL, "https://hub.test/events", "**"))
	require.Zero(t, posts)
}

func TestEnsureEventSubscriptionReconcilesLegacyPatternWithFullUpdate(t *testing.T) {
	var updateBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`[{"id":7,"event_pattern":"incident.*","delivery_target":"https://hub.test/events","enabled":true}]`))
			return
		}
		require.Equal(t, http.MethodPut, r.Method)
		require.NoError(t, json.NewDecoder(r.Body).Decode(&updateBody))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	require.NoError(t, EnsureEventSubscription(context.Background(), server.URL, "https://hub.test/events", "incident.**"))
	require.Equal(t, "notification-hub-events", updateBody["name"])
	require.Equal(t, "notification-hub", updateBody["owner_scenario"])
	require.Equal(t, "webhook", updateBody["delivery_type"])
	require.Equal(t, "incident.**", updateBody["event_pattern"])
	require.Equal(t, "https://hub.test/events", updateBody["delivery_target"])
}

func TestEnsureEventSubscriptionReportsUnconfiguredIntegration(t *testing.T) {
	require.ErrorIs(t, EnsureEventSubscription(context.Background(), "", "", ""), ErrEventIntegrationUnconfigured)
}

func TestRenderEventCoversIncidentLifecycleAndFallback(t *testing.T) {
	for _, eventType := range []string{"incident.opened.v1", "incident.severity_changed.v1", "incident.resolved.v1", "unknown.v1"} {
		rendered := RenderEvent(eventType, json.RawMessage(`{"check_id":"host-kernel-module-drift","severity":"critical","message":"driver drift"}`))
		if rendered.Title == "" || rendered.Body == "" || rendered.SensitivityLabel == "" {
			t.Fatalf("%s rendered = %+v", eventType, rendered)
		}
	}
}

func TestRenderEventUsesLiveOperatorTemplateForFacts(t *testing.T) {
	rendered := RenderEventWithTemplates(
		"incident.opened.v1",
		json.RawMessage(`{"check_id":"disk-space","severity":"warning","message":"low disk"}`),
		map[string]EventTemplate{"incident.opened.v1": {Title: "{{severity}}: {{check_id}}", Body: "{{message}}"}},
	)
	require.Equal(t, "warning: disk-space", rendered.Title)
	require.Equal(t, "low disk", rendered.Body)
	require.Equal(t, "sensitive", rendered.SensitivityLabel)
}

func TestOperatorStateRecipientReadsTheOperatorSetting(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_STORAGE_ROOT", root)
	resolve := OperatorStateRecipient()
	if got := resolve(context.Background()); got != "" {
		t.Fatalf("missing state resolved %q", got)
	}
	resolver, err := storage.NewResolver(storage.ResolverConfig{AppID: "vrooli", Profile: storage.ProfileAuto})
	if err != nil {
		t.Fatal(err)
	}
	paths, err := resolver.Resolve(storage.Options{ScenarioID: "vrooli-onboarding"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(paths.StateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(paths.StateDir, "operator-state.json"), []byte(`{"version":"1","updated_at":"now","notifications":{"recipient":" operator@host "}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := resolve(context.Background()); got != "operator@host" {
		t.Fatalf("resolved %q, want operator@host", got)
	}
}

func signedEventRequest(t *testing.T, secret, eventID string, body []byte) *http.Request {
	t.Helper()
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/events", bytes.NewReader(body))
	req.Header.Set("X-Vrooli-Events-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	req.Header.Set("X-Vrooli-Events-Event-ID", eventID)
	return req
}

// [REQ:NOTIFICA-P1-003]
func TestEventWebhookFailsClosedUntilTheSecretResolves(t *testing.T) {
	primary := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), primary, modules.AllSchemas()...))
	service := hub.New(database.NewFromPrimary(primary), nil, nil)
	body := []byte(`{"event_id":"evt-secret","event_type":"incident.opened.v1","source_scenario":"vrooli-autoheal","payload":{"severity":"warning","message":"disk low"}}`)
	secret := ""
	handler := EventWebhookFromSecretSource(service, func() string { return secret }, nil, nil)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, signedEventRequest(t, "stored-secret", "evt-secret", body))
	require.Equal(t, http.StatusUnauthorized, recorder.Code)

	secret = "stored-secret"
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, signedEventRequest(t, "stored-secret", "evt-secret", body))
	require.Equal(t, http.StatusAccepted, recorder.Code)

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, signedEventRequest(t, "wrong-secret", "evt-secret", body))
	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestResolveEventsWebhookSecretPrefersTheEnvironmentOverride(t *testing.T) {
	secret, err := ResolveEventsWebhookSecret(func(name string) string {
		if name == "VROOLI_EVENTS_WEBHOOK_SECRET" {
			return " override-secret "
		}
		return ""
	})
	require.NoError(t, err)
	require.Equal(t, "override-secret", secret)
}

func TestCachedSecretKeepsTheFirstResolvedValue(t *testing.T) {
	calls := 0
	source := CachedSecret(func() (string, error) {
		calls++
		return "resolved", nil
	}, nil)
	require.Equal(t, "resolved", source())
	require.Equal(t, "resolved", source())
	require.Equal(t, 1, calls)
}

// [REQ:NOTIFICA-P1-003]
func TestEventWebhookTurnsDecisionRequestIntoOneStructuredAsk(t *testing.T) {
	primary := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), primary, modules.AllSchemas()...))
	db := database.NewFromPrimary(primary)
	service := hub.New(db, nil, nil)
	secret := "event-secret"
	body := []byte(`{"event_id":"evt-decision","event_type":"agent_manager.effort.decision_requested.v1","source_scenario":"agent-manager","payload":{"subject":"BAS","question":"Close the goal or re-aim at quality?","options":[{"key":"close","label":"Close the goal"},{"key":"re-aim","label":"Re-aim at quality"}],"recommended":"re-aim","recommendation_reason":"quality gaps remain","default_answer":"re-aim","reversible":true,"deadline":"2030-01-01T00:00:00Z","correlation":{"decision_id":"BAS-D-065"}}}`)
	handler := EventWebhook(service, secret)

	var askIDs []string
	for i := 0; i < 2; i++ {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, signedEventRequest(t, secret, "evt-decision", body))
		require.Equal(t, http.StatusAccepted, recorder.Code, recorder.Body.String())
		var response map[string]string
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
		askIDs = append(askIDs, response["ask_id"])
	}
	require.Equal(t, askIDs[0], askIDs[1], "a redelivered event returns the same ask")

	ask, err := service.GetAsk(context.Background(), askIDs[0])
	require.NoError(t, err)
	require.Equal(t, []hub.AskOption{{Key: "close", Label: "Close the goal"}, {Key: "re-aim", Label: "Re-aim at quality"}}, ask.Options)
	require.Equal(t, "re-aim", ask.Recommended)
	require.Equal(t, "re-aim", ask.DefaultAnswer)
	require.True(t, ask.Reversible)
	require.Equal(t, "agent-manager", ask.Source)
	require.Equal(t, "evt-decision", ask.RequesterRef)
	n, _, err := service.Get(context.Background(), ask.NotificationID)
	require.NoError(t, err)
	require.Equal(t, "Decision needed: BAS", n.Title)
	require.Equal(t, "high", n.Urgency)
	var notifications int
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM notifications`).Scan(&notifications))
	require.Equal(t, 1, notifications)
}

func TestEventWebhookAcceptsAgentManagerAuthorizationFYI(t *testing.T) {
	primary := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), primary, modules.AllSchemas()...))
	db := database.NewFromPrimary(primary)
	service := hub.New(db, nil, nil)
	secret := "event-secret"
	eventID := "authorization-generation-1"
	consentURL := "https://agent-manager.example.test/api/v1/authorizations/123e4567-e89b-12d3-a456-426614174000/approve"
	body := []byte(`{"event_id":"authorization-generation-1","event_type":"agent_manager.authorization_requested.notification.v1","source_scenario":"agent-manager","payload":{"title":"Authorization requested: swarm-bookkeeping","body":"codex requests swarm-bookkeeping for swarm-manager:backlog/execute/obsolete. The 16-hour permission lifetime starts only after you approve on the authenticated Agent Manager page; this notification grants no authority.","category":"authorization-requested","urgency":"normal","sensitivity":"private","context_url":"` + consentURL + `","dedupe_key":"grant-1","correlation":{"authorization_id":"grant-1","run_id":"run-1","policy":"swarm-bookkeeping"}}}`)
	handler := EventWebhook(service, secret)
	for i := 0; i < 2; i++ {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, signedEventRequest(t, secret, eventID, body))
		require.Equal(t, http.StatusAccepted, recorder.Code, recorder.Body.String())
	}
	var count int
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM notifications WHERE idempotency_key = ?`, eventID).Scan(&count))
	require.Equal(t, 1, count, "event redelivery must reuse one Hub notification")
	var title, notificationBody, category, contextURL string
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT title, body, category, context_url FROM notifications WHERE idempotency_key = ?`, eventID).Scan(&title, &notificationBody, &category, &contextURL))
	require.Equal(t, "Authorization requested: swarm-bookkeeping", title)
	require.Contains(t, notificationBody, "starts only after you approve")
	require.Equal(t, "authorization-requested", category)
	require.Equal(t, consentURL, contextURL)
}

func TestEventWebhookTurnsThresholdHeartbeatEpisodeIntoIdempotentFYI(t *testing.T) {
	primary := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), primary, modules.AllSchemas()...))
	db := database.NewFromPrimary(primary)
	service := hub.New(db, nil, nil)
	secret := "event-secret"
	const episodeID = "episode-heartbeat-team-a-agent-b-success-1"
	handler := EventWebhook(service, secret)
	for _, tc := range []struct {
		eventID string
		streak  int
		outcome string
	}{
		{eventID: "heartbeat-attempt-3", streak: 3, outcome: "request_submitted"},
		{eventID: "heartbeat-attempt-4", streak: 4, outcome: "request_suppressed"},
	} {
		body := []byte(fmt.Sprintf(`{"event_id":%q,"event_type":"prompt-manager.heartbeat.attempt.v1","source_scenario":"prompt-manager","payload":{"teamId":"team-a","agentId":"agent-b","status":"failed","attentionEligible":true,"consecutiveFailures":%d,"failureEpisodeId":%q,"recoveryOutcome":%q,"recoveryRequestId":"recovery-123","reviewUrl":"https://pm.example/teams/team-a"}}`, tc.eventID, tc.streak, episodeID, tc.outcome))
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, signedEventRequest(t, secret, tc.eventID, body))
		require.Equal(t, http.StatusAccepted, recorder.Code, recorder.Body.String())
	}
	var count int
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM notifications WHERE idempotency_key = ?`, episodeID).Scan(&count))
	require.Equal(t, 1, count, "later attempts in one failure episode must reuse the FYI")
	var title, notificationBody, sensitivity, category, contextURL string
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT title, body, sensitivity_label, category, context_url FROM notifications WHERE idempotency_key = ?`, episodeID).Scan(&title, &notificationBody, &sensitivity, &category, &contextURL))
	require.Equal(t, "Prompt Manager heartbeat needs review", title)
	require.Contains(t, notificationBody, "3 consecutive failed heartbeats")
	require.Contains(t, notificationBody, "recovery request (recovery-123) was submitted")
	require.Contains(t, notificationBody, "This does not mean it was approved or completed.")
	require.Equal(t, "private", sensitivity)
	require.Equal(t, "prompt_manager_heartbeat", category)
	require.Equal(t, "https://pm.example/teams/team-a", contextURL)
}

func TestEventWebhookDoesNotAlertForNonthresholdOrNonfailureHeartbeatAttempts(t *testing.T) {
	primary := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), primary, modules.AllSchemas()...))
	db := database.NewFromPrimary(primary)
	service := hub.New(db, nil, nil)
	secret := "event-secret"
	for _, tc := range []struct {
		id, status string
		streak     int
		eligible   bool
	}{
		{id: "heartbeat-attempt-2", status: "failed", streak: 2},
		{id: "heartbeat-attempt-pending", status: "approval_pending", streak: 2},
		{id: "heartbeat-attempt-observer", status: "observer_unavailable", streak: 3},
		{id: "heartbeat-attempt-uncertain", status: "failed", streak: 3, eligible: false},
	} {
		body := []byte(fmt.Sprintf(`{"event_id":%q,"event_type":"prompt-manager.heartbeat.attempt.v1","source_scenario":"prompt-manager","payload":{"teamId":"team-a","agentId":"agent-b","status":%q,"attentionEligible":%t,"consecutiveFailures":%d,"recoveryOutcome":"not_requested"}}`, tc.id, tc.status, tc.eligible, tc.streak))
		recorder := httptest.NewRecorder()
		EventWebhook(service, secret).ServeHTTP(recorder, signedEventRequest(t, secret, tc.id, body))
		require.Equal(t, http.StatusAccepted, recorder.Code, recorder.Body.String())
	}
	var count int
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM notifications`).Scan(&count))
	require.Zero(t, count, "nonthreshold, nonfailure, pending, observer-unavailable, and uncertain attempts are not attention-worthy")
}

func TestEventWebhookMapsRetainedOwnerStateToScopedFYIWithoutDecisionActions(t *testing.T) {
	primary := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), primary, modules.AllSchemas()...))
	db := database.NewFromPrimary(primary)
	service := hub.New(db, nil, nil)
	secret := "event-secret"
	eventID := "retained-owner-event-1"
	body := []byte(`{"event_id":"retained-owner-event-1","event_type":"prompt-manager.heartbeat.owner-state.v1","source_scenario":"prompt-manager","payload":{"teamId":"team-a","agentId":"agent-a","ownerObligationId":"obligation-a","retainedOwnerState":"dispatch_uncertain","status":"dispatch_uncertain","observedAt":"2026-10-17T01:02:03Z","reviewUrl":"https://pm.example/teams/team-a"}}`)
	handler := EventWebhook(service, secret)
	for i := 0; i < 2; i++ {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, signedEventRequest(t, secret, eventID, body))
		require.Equal(t, http.StatusAccepted, recorder.Code, recorder.Body.String())
	}
	var count int
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM notifications WHERE category='prompt_manager_heartbeat'`).Scan(&count))
	require.Equal(t, 1, count, "redelivery of one frozen owner-state fact must reuse one FYI")
	var id, title, notificationBody, category, contextURL string
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT id, title, body, category, context_url FROM notifications WHERE category='prompt_manager_heartbeat'`).Scan(&id, &title, &notificationBody, &category, &contextURL))
	require.Equal(t, "Prompt Manager heartbeat owner state needs review", title)
	require.Contains(t, notificationBody, "did not establish whether it accepted the run")
	require.NotContains(t, notificationBody, "approve")
	require.NotContains(t, notificationBody, "clear")
	require.Equal(t, "prompt_manager_heartbeat", category)
	require.Equal(t, "https://pm.example/teams/team-a", contextURL)
	require.Eventually(t, func() bool {
		notification, _, err := service.Get(context.Background(), id)
		return err == nil && notification.State == "unroutable"
	}, time.Second, 10*time.Millisecond, "finish the accepted owner's delivery attempt before closing the test database")
}

func TestEventWebhookMapsPreparationInterruptedToStableScopedFYIWithoutAsk(t *testing.T) {
	primary := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), primary, modules.AllSchemas()...))
	db := database.NewFromPrimary(primary)
	service := hub.New(db, nil, nil)
	secret := "event-secret"
	eventID := "preparation-interrupted-event-1"
	body := []byte(`{"event_id":"preparation-interrupted-event-1","event_type":"prompt-manager.heartbeat.owner-state.v1","source_scenario":"prompt-manager","payload":{"teamId":"team-a","agentId":"agent-a","ownerObligationId":"obligation-preparation-1","retainedOwnerState":"preparation_interrupted","observedAt":"2026-10-17T01:02:03Z","reviewUrl":"https://pm.example/teams/team-a"}}`)
	handler := EventWebhook(service, secret)
	for i := 0; i < 2; i++ {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, signedEventRequest(t, secret, eventID, body))
		require.Equal(t, http.StatusAccepted, recorder.Code, recorder.Body.String())
	}

	var notifications, asks int
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM notifications WHERE category='prompt_manager_heartbeat'`).Scan(&notifications))
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM asks`).Scan(&asks))
	require.Equal(t, 1, notifications, "redelivery must reuse the retained obligation's scoped FYI")
	require.Zero(t, asks, "preparation interruption is informational and must not create an Ask")
	var id, title, notificationBody, category, contextURL, idempotencyKey string
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT id, title, body, category, context_url, idempotency_key FROM notifications WHERE category='prompt_manager_heartbeat'`).Scan(&id, &title, &notificationBody, &category, &contextURL, &idempotencyKey))
	require.Equal(t, "Prompt Manager heartbeat owner state needs review", title)
	require.Contains(t, notificationBody, "before sending the first Agent Manager task or run request")
	require.Contains(t, notificationBody, "The obligation remains retained")
	require.Contains(t, notificationBody, "ordinary enabled scheduled admission or a fresh authorized manual admission")
	require.NotContains(t, notificationBody, "owner state could not be confirmed")
	require.NotContains(t, notificationBody, "approve")
	require.NotContains(t, notificationBody, "clear")
	require.Equal(t, "prompt_manager_heartbeat", category)
	require.Equal(t, "https://pm.example/teams/team-a", contextURL)
	require.Equal(t, "prompt-manager:team-a:agent-a:obligation-preparation-1:preparation_interrupted", idempotencyKey)
	require.Eventually(t, func() bool {
		notification, _, err := service.Get(context.Background(), id)
		return err == nil && notification.State == "unroutable"
	}, time.Second, 10*time.Millisecond, "finish the accepted owner's delivery attempt before closing the test database")
}

func TestEventWebhookRejectsPreparationInterruptedOwnerStateWithRunID(t *testing.T) {
	primary := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), primary, modules.AllSchemas()...))
	db := database.NewFromPrimary(primary)
	service := hub.New(db, nil, nil)
	secret := "event-secret"
	eventID := "preparation-interrupted-with-run"
	body := []byte(`{"event_id":"preparation-interrupted-with-run","event_type":"prompt-manager.heartbeat.owner-state.v1","source_scenario":"prompt-manager","payload":{"teamId":"team-a","agentId":"agent-a","ownerObligationId":"obligation-preparation-2","retainedOwnerState":"preparation_interrupted","runId":"run-should-not-exist","observedAt":"2026-10-17T01:02:03Z","reviewUrl":"https://pm.example/teams/team-a"}}`)
	recorder := httptest.NewRecorder()
	EventWebhook(service, secret).ServeHTTP(recorder, signedEventRequest(t, secret, eventID, body))
	require.Equal(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())
	var notifications, asks int
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM notifications`).Scan(&notifications))
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM asks`).Scan(&asks))
	require.Zero(t, notifications, "an internally contradictory pre-task owner fact must not be published")
	require.Zero(t, asks)
}

func TestHeartbeatOwnerStateMapperDoesNotCreateAnApprovalAsk(t *testing.T) {
	primary := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), primary, modules.AllSchemas()...))
	db := database.NewFromPrimary(primary)
	service := hub.New(db, nil, nil)
	secret := "event-secret"
	body := []byte(`{"event_id":"retained-owner-approval-pending","event_type":"prompt-manager.heartbeat.owner-state.v1","source_scenario":"prompt-manager","payload":{"teamId":"team-a","agentId":"agent-a","ownerObligationId":"obligation-a","retainedOwnerState":"approval_pending","status":"approval_pending","observedAt":"2026-10-17T01:02:03Z"}}`)
	recorder := httptest.NewRecorder()
	EventWebhook(service, secret).ServeHTTP(recorder, signedEventRequest(t, secret, "retained-owner-approval-pending", body))
	require.Equal(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())
	var notifications, asks int
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM notifications`).Scan(&notifications))
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM asks`).Scan(&asks))
	require.Zero(t, notifications)
	require.Zero(t, asks)
}

func TestHeartbeatFailureEpisodesForDifferentTeamsDoNotCollapse(t *testing.T) {
	primary := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), primary, modules.AllSchemas()...))
	db := database.NewFromPrimary(primary)
	service := hub.New(db, nil, nil)
	secret := "event-secret"
	for _, tc := range []struct {
		eventID, teamID, episodeID string
	}{
		{eventID: "team-a-attempt", teamID: "team-a", episodeID: "episode-team-a"},
		{eventID: "team-b-attempt", teamID: "team-b", episodeID: "episode-team-b"},
	} {
		body := []byte(fmt.Sprintf(`{"event_id":%q,"event_type":"prompt-manager.heartbeat.attempt.v1","source_scenario":"prompt-manager","payload":{"teamId":%q,"agentId":"agent-1","status":"failed","attentionEligible":true,"consecutiveFailures":3,"failureEpisodeId":%q,"recoveryOutcome":"not_requested"}}`, tc.eventID, tc.teamID, tc.episodeID))
		recorder := httptest.NewRecorder()
		EventWebhook(service, secret).ServeHTTP(recorder, signedEventRequest(t, secret, tc.eventID, body))
		require.Equal(t, http.StatusAccepted, recorder.Code, recorder.Body.String())
	}
	var count int
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM notifications`).Scan(&count))
	require.Equal(t, 2, count, "different team failure episodes must have independent dedupe identities")
}

// [REQ:NOTIFICA-P1-003]
func TestEventWebhookRejectsUnsafeDecisionRequest(t *testing.T) {
	primary := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), primary, modules.AllSchemas()...))
	service := hub.New(database.NewFromPrimary(primary), nil, nil)
	secret := "event-secret"
	body := []byte(`{"event_id":"evt-unsafe","event_type":"agent_manager.effort.decision_requested.v1","source_scenario":"agent-manager","payload":{"question":"Delete the product surface?","options":[{"key":"delete"},{"key":"keep"}],"recommended":"delete","default_answer":"delete","reversible":false}}`)
	recorder := httptest.NewRecorder()
	EventWebhook(service, secret).ServeHTTP(recorder, signedEventRequest(t, secret, "evt-unsafe", body))
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), "reversible")
}

// [REQ:NOTIFICA-P1-003]
func TestEnsureDecisionSubscriptionAddsItsOwnSubscriptionBesideTheIncidentOne(t *testing.T) {
	var created map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`[{"id":1,"name":"notification-hub-events","event_pattern":"incident.**","delivery_target":"https://hub.test/events","enabled":true}]`))
			return
		}
		require.Equal(t, http.MethodPost, r.Method, "the incident subscription must not be rewritten")
		require.NoError(t, json.NewDecoder(r.Body).Decode(&created))
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	require.NoError(t, EnsureDecisionSubscription(context.Background(), server.URL, "https://hub.test/events"))
	require.Equal(t, "notification-hub-decisions", created["name"])
	require.Equal(t, DecisionRequestPattern, created["event_pattern"])
}

func TestEnsureHeartbeatAttemptSubscriptionAddsExactPatternBesideExistingSubscriptions(t *testing.T) {
	var created []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`[{"id":1,"name":"notification-hub-events","event_pattern":"incident.**","delivery_target":"https://hub.test/events","enabled":true}]`))
			return
		}
		require.Equal(t, http.MethodPost, r.Method)
		var item map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&item))
		created = append(created, item)
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()
	require.NoError(t, EnsureHeartbeatAttemptSubscription(context.Background(), server.URL, "https://hub.test/events"))
	require.Len(t, created, 2)
	require.Equal(t, heartbeatAttemptSubscriptionName, created[0]["name"])
	require.Equal(t, HeartbeatAttemptEventPattern, created[0]["event_pattern"])
	require.Equal(t, heartbeatOwnerStateSubscriptionName, created[1]["name"])
	require.Equal(t, HeartbeatOwnerStateEventPattern, created[1]["event_pattern"])
}
