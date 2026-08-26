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
	"testing"
	"time"

	"notification-hub/internal/hub"
	"notification-hub/internal/modules"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
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
