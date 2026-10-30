package main

import (
	"encoding/json"
	"strings"
	"testing"

	clitest "agent-manager/cli/internal/testutil"

	"github.com/vrooli/cli-core/cliutil"
)

func reportSpendApp(t *testing.T) *App {
	t.Helper()
	server := clitest.NewRecordingServerForRequests(t, func(request clitest.Request) string {
		switch request.Path {
		case "/api/v1/runs/run-1/report":
			return `{"run_id":"run-1","status":"complete","turns":1,"tokens":"10","cost_usd":0,"event_counts":{},"diff":{"bytes":"0","available":{"state":"available"}},"receipts_availability":{"state":"unobserved"}}`
		case "/api/v1/runs/run-1/invocation-facts":
			return `{"facts":[{"id":"fact-1"}]}`
		case "/agent_manager.v1.AgentManagerService/GetRunAccounting":
			return `{"model":"gpt-6-sol","tokensKnown":true,"nonCacheTokens":"100"}`
		default:
			return `{}`
		}
	})
	api := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions {
		return cliutil.APIBaseOptions{DefaultBase: server.URL()}
	}, nil)
	return &App{services: NewServices(api)}
}

// P4-F7: `usage run` is folded into `run report`, which now carries the run's
// weighted token spend and, with --facts, its invocation facts.
func TestRunReportCarriesWeightedSpend(t *testing.T) {
	app := reportSpendApp(t)
	output := captureStdout(t, func() error { return app.runReport([]string{"run-1", "--weights", "sol=7"}) })
	if !strings.Contains(output, "non-cache=100") || !strings.Contains(output, "x7 = 700") {
		t.Fatalf("report must print the weighted spend with the requested tier weights:\n%s", output)
	}
}

func TestRunReportJSONNestsSpendBesideTheReport(t *testing.T) {
	app := reportSpendApp(t)
	output := captureStdout(t, func() error { return app.runReport([]string{"run-1", "--json"}) })
	var decoded struct {
		RunID      string      `json:"run_id"`
		UsageSpend tokenReport `json:"usage_spend"`
	}
	if err := json.Unmarshal([]byte(output), &decoded); err != nil {
		t.Fatalf("report JSON: %v\n%s", err, output)
	}
	if decoded.RunID != "run-1" || decoded.UsageSpend.TotalNonCache != 100 || len(decoded.UsageSpend.Rows) != 1 {
		t.Fatalf("report JSON must keep the report and add usage_spend: %+v", decoded)
	}
}

func TestRunReportFactsPrintsInvocationFacts(t *testing.T) {
	app := reportSpendApp(t)
	output := captureStdout(t, func() error { return app.runReport([]string{"run-1", "--facts"}) })
	if !strings.Contains(output, "fact-1") || strings.Contains(output, "Turns:") {
		t.Fatalf("--facts must print the facts instead of the report:\n%s", output)
	}
}
