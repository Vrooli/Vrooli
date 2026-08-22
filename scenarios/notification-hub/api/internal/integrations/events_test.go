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

func TestEnsureEventSubscriptionIsIdempotent(t *testing.T) {
	var posts int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`[{"delivery_target":"https://hub.test/events","enabled":true}]`))
			return
		}
		posts++
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()
	require.NoError(t, EnsureEventSubscription(context.Background(), server.URL, "https://hub.test/events", "**"))
	require.Zero(t, posts)
}
