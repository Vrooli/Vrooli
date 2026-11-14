package handlers

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"agent-manager/internal/domain"
)

type providerReadinessFixture struct {
	value domain.ProviderReadiness
}

func (f providerReadinessFixture) ReadProviderReadiness(context.Context) domain.ProviderReadiness {
	return f.value
}

func TestProviderReadinessHandlerReturnsNoStoreOwnerObservation(t *testing.T) {
	observed := time.Date(2026, 11, 13, 15, 0, 0, 0, time.UTC)
	reader := providerReadinessFixture{value: domain.ProviderReadiness{
		Provider: "workspace-sandbox", State: domain.ProviderReadinessUnknown,
		Known: false, ObservedAt: &observed,
		Message: "Workspace Sandbox readiness could not be confirmed.",
	}}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/api/v1/provider-readiness", nil)
	ProviderReadinessHandler(reader).ServeHTTP(recorder, request)
	if recorder.Code != 200 || recorder.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("response code/cache = %d/%q", recorder.Code, recorder.Header().Get("Cache-Control"))
	}
	var got domain.ProviderReadiness
	if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode readiness response: %v", err)
	}
	if got.Provider != "workspace-sandbox" || got.State != domain.ProviderReadinessUnknown || got.Known || got.Message == "" || got.ObservedAt == nil || !got.ObservedAt.Equal(observed) {
		t.Fatalf("provider readiness response = %+v", got)
	}
}
