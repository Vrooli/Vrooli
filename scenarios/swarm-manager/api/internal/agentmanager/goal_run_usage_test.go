package agentmanager

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"swarm-manager/internal/workflowcontract"
)

func accountingService(t *testing.T, body string, path *string) *AgentService {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if path != nil {
			*path = r.URL.Path
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	client := NewHTTPClientWithResolver(func(context.Context) (string, error) { return server.URL, nil }, server.Client())
	return &AgentService{client: client, enabled: true}
}

func TestGetGoalRunUsageMapsOwnerAccounting(t *testing.T) {
	var path string
	service := accountingService(t, `{"run_id":"run-1","terminal":true,"tokens":"1200","turns":"4","tokens_known":true,"charge_micro_usd":"350","charge_measured":true,"wall_seconds":"90"}`, &path)
	usage, terminal, err := service.GetGoalRunUsage(t.Context(), "run-1")
	if err != nil {
		t.Fatal(err)
	}
	if path != "/api/v1/runs/run-1/accounting" {
		t.Fatalf("path = %q", path)
	}
	want := workflowcontract.Usage{Tokens: 1200, Turns: 4, WallSeconds: 90, ChargeMicroUSD: 350, TokensKnown: true, ChargeMeasured: true}
	if !terminal || usage == nil || *usage != want {
		t.Fatalf("usage = %+v terminal=%v, want %+v terminal", usage, terminal, want)
	}
}

func TestGetGoalRunUsageNeverReportsALiveRunAsKnown(t *testing.T) {
	service := accountingService(t, `{"run_id":"run-1","terminal":false,"tokens":"10","tokens_known":true,"charge_measured":true,"wall_seconds":"5"}`, nil)
	usage, terminal, err := service.GetGoalRunUsage(t.Context(), "run-1")
	if err != nil {
		t.Fatal(err)
	}
	if terminal || usage.TokensKnown || usage.ChargeMeasured {
		t.Fatalf("a live run was reported as settled: %+v terminal=%v", usage, terminal)
	}
}

func TestGetRunAccountingRejectsAnAnswerForAnotherRun(t *testing.T) {
	service := accountingService(t, `{"run_id":"other-run","terminal":true,"tokens_known":true,"charge_measured":true,"wall_seconds":"5"}`, nil)
	if _, err := service.client.GetRunAccounting(t.Context(), "run-1"); err == nil {
		t.Fatal("accounting for a different run was accepted")
	}
}
