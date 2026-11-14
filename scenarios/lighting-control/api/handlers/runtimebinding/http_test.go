package runtimebinding

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrooli/api-core/identity"
	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/control"
)

type fakeService struct {
	binds, quiesces, reconciles int
}

func (s *fakeService) BindBackend(context.Context, string, string) error { s.binds++; return nil }
func (s *fakeService) Quiesce(context.Context) error                     { s.quiesces++; return nil }
func (s *fakeService) Reconcile(context.Context) ([]*controlv1.ReconcileZoneResult, error) {
	s.reconciles++
	return nil, nil
}

func TestBindRequiresVerifiedHumanWithScenarioWriteScope(t *testing.T) {
	service := &fakeService{}
	mux := http.NewServeMux()
	Register(mux, service, "lighting-api-generation-7")
	body := `{"address":"127.0.0.1:6742","incarnation":"openrgb:42:born-1","generation":"lighting-api-generation-7"}`
	for _, principal := range []identity.Principal{
		{},
		{Kind: identity.ActorAgent, Subject: "agent", Verified: true, Scopes: []string{"lighting-control:write"}},
		{Kind: identity.ActorHuman, Subject: "operator", Verified: false, Scopes: []string{"lighting-control:write"}},
		{Kind: identity.ActorHuman, Subject: "operator", Verified: true, Scopes: []string{"other:write"}},
		{Kind: identity.ActorHuman, Subject: "operator", Verified: true, Scopes: []string{"lighting-control:write"}},
	} {
		req := httptest.NewRequest(http.MethodPost, "/internal/lighting/backend/bind", strings.NewReader(body))
		if principal.Kind != "" {
			req = req.WithContext(identity.WithPrincipal(req.Context(), principal))
		}
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, req)
		if response.Code != http.StatusForbidden {
			t.Fatalf("principal %+v status = %d, want forbidden", principal, response.Code)
		}
	}
	if service.binds != 0 || service.reconciles != 0 {
		t.Fatalf("unauthorized action reached service: %+v", service)
	}

	principal := identity.Principal{Kind: identity.ActorHuman, Subject: "operator", Verified: true, Scopes: []string{"lighting-control:write", "lighting-control:backend-owner"}}
	req := httptest.NewRequest(http.MethodPost, "/internal/lighting/backend/bind", strings.NewReader(body))
	req = req.WithContext(identity.WithPrincipal(req.Context(), principal))
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, req)
	if response.Code != http.StatusOK || service.binds != 1 || service.reconciles != 1 {
		t.Fatalf("authorized bind status=%d service=%+v body=%s", response.Code, service, response.Body.String())
	}
}

func TestBindRejectsUnknownAndTrailingJSONFields(t *testing.T) {
	service := &fakeService{}
	mux := http.NewServeMux()
	Register(mux, service, "generation-7")
	principal := identity.Principal{Kind: identity.ActorHuman, Subject: "operator", Verified: true, Scopes: []string{"lighting-control:write", "lighting-control:backend-owner"}}
	for _, body := range []string{
		`{"address":"127.0.0.1:6742","incarnation":"x","generation":"generation-7","argv":["shell"]}`,
		`{"address":"127.0.0.1:6742","incarnation":"x","generation":"generation-7"}{}`,
	} {
		req := httptest.NewRequest(http.MethodPost, "/internal/lighting/backend/bind", strings.NewReader(body))
		req = req.WithContext(identity.WithPrincipal(req.Context(), principal))
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, req)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("body %s status = %d, want bad request", body, response.Code)
		}
	}
	if service.binds != 0 {
		t.Fatalf("invalid request reached bind: %+v", service)
	}
	stale := httptest.NewRequest(http.MethodPost, "/internal/lighting/backend/bind", strings.NewReader(`{"address":"127.0.0.1:6742","incarnation":"x","generation":"stale-generation"}`))
	stale = stale.WithContext(identity.WithPrincipal(stale.Context(), principal))
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, stale)
	if response.Code != http.StatusConflict || service.binds != 0 {
		t.Fatalf("stale generation status=%d, service=%+v", response.Code, service)
	}
}
