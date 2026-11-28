package agentmanager

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	"swarm-manager/internal/workflowcontract"
)

type accountingHandler struct {
	apiconnect.UnimplementedAgentManagerServiceHandler
	accounting   *api.RunAccounting
	requestedRun string
}

func (h *accountingHandler) GetRunAccounting(_ context.Context, r *connect.Request[api.GetRunAccountingRequest]) (*connect.Response[api.RunAccounting], error) {
	h.requestedRun = r.Msg.GetRunId()
	return connect.NewResponse(h.accounting), nil
}

func accountingService(t *testing.T, accounting *api.RunAccounting) (*AgentService, *accountingHandler) {
	t.Helper()
	h := &accountingHandler{accounting: accounting}
	path, handler := apiconnect.NewAgentManagerServiceHandler(h)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := NewHTTPClientWithResolver(func(context.Context) (string, error) { return server.URL, nil }, server.Client())
	return &AgentService{client: client, enabled: true}, h
}

func TestGetGoalRunUsageMapsOwnerAccounting(t *testing.T) {
	service, h := accountingService(t, &api.RunAccounting{RunId: "run-1", Terminal: true, Tokens: 1200, Turns: 4, TokensKnown: true, ChargeMicroUsd: 350, ChargeMeasured: true, WallSeconds: 90})
	usage, terminal, err := service.GetGoalRunUsage(t.Context(), "run-1")
	if err != nil {
		t.Fatal(err)
	}
	if h.requestedRun != "run-1" {
		t.Fatalf("requested run=%q", h.requestedRun)
	}
	want := workflowcontract.Usage{Tokens: 1200, Turns: 4, WallSeconds: 90, ChargeMicroUSD: 350, TokensKnown: true, ChargeMeasured: true}
	if !terminal || usage == nil || *usage != want {
		t.Fatalf("usage=%+v terminal=%v, want %+v terminal", usage, terminal, want)
	}
}

func TestGetGoalRunUsageNeverReportsALiveRunAsKnown(t *testing.T) {
	service, _ := accountingService(t, &api.RunAccounting{RunId: "run-1", Terminal: false, Tokens: 10, TokensKnown: true, ChargeMeasured: true, WallSeconds: 5})
	usage, terminal, err := service.GetGoalRunUsage(t.Context(), "run-1")
	if err != nil {
		t.Fatal(err)
	}
	if terminal || usage.TokensKnown || usage.ChargeMeasured {
		t.Fatalf("live run reported settled: %+v terminal=%v", usage, terminal)
	}
}

func TestGetRunAccountingRejectsAnAnswerForAnotherRun(t *testing.T) {
	service, _ := accountingService(t, &api.RunAccounting{RunId: "other-run", Terminal: true, TokensKnown: true, ChargeMeasured: true, WallSeconds: 5})
	if _, err := service.client.GetRunAccounting(t.Context(), "run-1"); err == nil {
		t.Fatal("accounting for a different run was accepted")
	}
}
