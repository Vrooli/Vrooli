package integrations

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
