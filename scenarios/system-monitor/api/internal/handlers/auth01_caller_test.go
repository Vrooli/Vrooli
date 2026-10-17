package handlers

import (
	"connectrpc.com/connect"
	"context"
	"github.com/vrooli/api-core/owneridentity"
	investigationspb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/investigations"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/config"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAuth01InvestigationIngressRejectsBeforeEffects(t *testing.T) {
	h := &InvestigationHandler{}
	r := httptest.NewRequest(http.MethodPost, "/trigger", nil)
	w := httptest.NewRecorder()
	h.HandleTriggerInvestigation(w, r)
	if w.Code != 401 {
		t.Fatal("REST anonymous accepted")
	}
	_, err := h.TriggerInvestigation(context.Background(), connect.NewRequest(&investigationspb.TriggerInvestigationRequest{}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatal("Connect anonymous accepted")
	}
}

type auth01Verifier struct{}

func (auth01Verifier) Validate(context.Context, string) (owneridentity.Identity, error) {
	return owneridentity.Identity{Subject: "fixture", Scopes: []string{"agent-manager:write"}, ExpiresAt: time.Now().Add(time.Hour)}, nil
}

type auth01InvestigationSpy struct {
	InvestigationManager
	calls int
	t     *testing.T
}

func (s *auth01InvestigationSpy) TriggerInvestigation(ctx context.Context, _ bool, _ string) (*models.Investigation, error) {
	s.calls++
	if owneridentity.RequireCreateRunCaller(ctx, time.Now()) != nil {
		s.t.Error("qualified proof lost")
	}
	return &models.Investigation{ID: "fixture"}, nil
}
func TestAuth01ManualInvestigationRESTAndConnectPreserveQualifiedCaller(t *testing.T) {
	spy := &auth01InvestigationSpy{t: t}
	h := &InvestigationHandler{config: &config.Config{}, investigationSvc: spy, runCallerValidator: auth01Verifier{}}
	r := httptest.NewRequest(http.MethodPost, "/trigger", nil)
	r.Header.Set("Authorization", "Bearer fixture-owner")
	w := httptest.NewRecorder()
	h.HandleTriggerInvestigation(w, r)
	if w.Code != http.StatusAccepted || spy.calls != 1 {
		t.Fatalf("REST status%d calls%d", w.Code, spy.calls)
	}
	req := connect.NewRequest(&investigationspb.TriggerInvestigationRequest{})
	req.Header().Set("Authorization", "Bearer fixture-owner")
	_, err := h.TriggerInvestigation(context.Background(), req)
	if err != nil || spy.calls != 2 {
		t.Fatalf("Connect %v calls%d", err, spy.calls)
	}
}
