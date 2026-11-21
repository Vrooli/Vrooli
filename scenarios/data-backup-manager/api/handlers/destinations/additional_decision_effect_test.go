package destinations

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/askreader"
	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"
)

type effectTestPublisher struct{}

func (effectTestPublisher) PublishDomainEvent(context.Context, eventbus.DomainEvent) error {
	return nil
}

type additionalDecisionEffect struct{ called bool }

func (e *additionalDecisionEffect) Handles(string) bool { return true }
func (e *additionalDecisionEffect) Apply(context.Context, operatordecision.DecisionResolution) (bool, error) {
	e.called = true
	return true, nil
}

func TestWebhookRoutesAdditionalDecisionEffect(t *testing.T) {
	t.Setenv("API_PORT", "")
	secret := "test-webhook-secret"
	effect := &additionalDecisionEffect{}
	module, _ := ModuleWithBackupRecovery(nil, nil, nil, t.TempDir(), filepath.Join(t.TempDir(), "state"), nil, effectTestPublisher{}, secret, BackupFailureConfig{DecisionEffect: effect, AskReader: noCredentialReader{}})
	router := mux.NewRouter()
	module.Mount(router)
	payload, err := json.Marshal(map[string]any{
		"event_type": operatordecision.AskResolvedEventType,
		"payload":    operatordecision.DecisionResolution{AskID: "a", State: operatordecision.StateAnswered, Source: recoveryDecisionSource, SourceEventType: "custom.review"},
	})
	if err != nil {
		t.Fatal(err)
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(payload)
	req := httptest.NewRequest(http.MethodPost, recoveryDecisionWebhookPath, bytes.NewReader(payload))
	req.Header.Set(operatordecision.SignatureHeader, hex.EncodeToString(mac.Sum(nil)))
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("webhook status=%d body=%s", resp.Code, resp.Body.String())
	}
	if !effect.called {
		t.Fatal("additional decision effect was not reached through the shared signed callback")
	}
}

// noCredentialReader is a Hub readback for a scenario with no ask-reader credential
// yet, so a test never reads the operator's real credential or the real Hub.
type noCredentialReader struct{}

func (noCredentialReader) ReadAsk(context.Context, string) (askreader.Ask, error) {
	return askreader.Ask{}, askreader.ErrNotIssued
}
