package main

import (
	"encoding/json"
	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDeliveryCLIUsesExactEnrollmentAndOwnerClock(t *testing.T) {
	var mutation map[string]any
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case apiconnect.AgentManagerServiceGetEffortBoardProcedure:
			w.Write([]byte(`{"rows":[{"enrollment":{"effortRef":"effort:fixture","revision":"17","targetRevision":"selected-1","destinationRef":"path:.vrooli/work/delivery/fixture/CHARTER.md"}}]}`))
		case apiconnect.AgentManagerServiceEnrollEffortProcedure:
			if err := json.NewDecoder(r.Body).Decode(&mutation); err != nil {
				t.Error(err)
			}
			w.Write([]byte(`{}`))
		default:
			t.Errorf("unexpected owner path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	api := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions { return cliutil.APIBaseOptions{DefaultBase: server.URL} }, nil)
	app := &App{services: NewServices(api)}
	if err := app.cmdEffort([]string{"delivery", "start", "--effort-ref", "effort:fixture", "--commission", "disposable", "--budget", "1h", "--workers", "2", "--json"}); err != nil {
		t.Fatal(err)
	}
	e, ok := mutation["enrollment"].(map[string]any)
	if !ok {
		t.Fatal("missing owner enrollment", mutation)
	}
	d, ok := e["delivery"].(map[string]any)
	if !ok {
		t.Fatal("missing commission", e)
	}
	if calls != 2 || mutation["expected_revision"] != "17" || d["budget_seconds"] != "3600" || d["max_workers"] != float64(2) || d["commission_id"] != "disposable" || d["started_at"] != nil || e["destination_ref"] != "path:.vrooli/work/delivery/fixture/CHARTER.md" {
		t.Fatalf("lost exact revision/policy or supplied caller clock: %+v", mutation)
	}
	t.Setenv(cliutil.EnvIdentityToken, "identified-agent")
	if err := app.cmdEffort([]string{"delivery", "start", "--effort-ref", "effort:fixture", "--commission", "other", "--local-owner"}); err == nil {
		t.Fatal("identified agent acquired local owner authority")
	}
	if calls != 2 {
		t.Fatal("rejected authority made a request")
	}
}
