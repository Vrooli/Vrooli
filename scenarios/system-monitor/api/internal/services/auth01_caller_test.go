package services

import (
	"context"
	"github.com/vrooli/api-core/owneridentity"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/agentmanager"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/services/mocks"
	"net/http"
	"testing"
	"time"
)

func TestAuth01InvestigationRejectsBeforeRepositoryAvailabilityOrWorker(t *testing.T) {
	s := &InvestigationService{}
	if _, err := s.TriggerInvestigation(context.Background(), false, "fixture"); err != owneridentity.ErrCreateRunCaller {
		t.Fatalf("anonymous reached effects%v", err)
	}
}

type auth01FixtureVerifier struct{}

func (auth01FixtureVerifier) Validate(context.Context, string) (owneridentity.Identity, error) {
	return owneridentity.Identity{Subject: "fixture", Scopes: []string{"agent-manager:write"}, ExpiresAt: time.Now().Add(time.Hour)}, nil
}
func auth01FixtureCaller(t *testing.T, ctx context.Context) context.Context {
	t.Helper()
	out, err := owneridentity.AuthorizeCreateRunCaller(ctx, http.Header{"Authorization": []string{"Bearer fixture-owner"}}, auth01FixtureVerifier{}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return out
}

type auth01WorkerSpy struct {
	*mocks.AgentExecutor
	received chan error
}

func (s *auth01WorkerSpy) Execute(ctx context.Context, _ agentmanager.ExecuteRequest) (*agentmanager.ExecuteResult, error) {
	s.received <- owneridentity.RequireCreateRunCaller(ctx, time.Now())
	return &agentmanager.ExecuteResult{Success: true, Output: "fixture"}, nil
}
func TestAuth01ManualInvestigationWorkerRetainsQualifiedCaller(t *testing.T) {
	svc := newTestInvestigationService(t, RealClock{})
	defer svc.Shutdown()
	spy := &auth01WorkerSpy{AgentExecutor: mocks.NewAgentExecutor(), received: make(chan error, 1)}
	svc.agentSvc = spy
	if _, err := svc.TriggerInvestigation(auth01FixtureCaller(t, context.Background()), false, "fixture"); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-spy.received:
		if err != nil {
			t.Fatal("worker lost proof", err)
		}
	case <-time.After(time.Second):
		t.Fatal("worker not called")
	}
}
