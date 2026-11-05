package control

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	testutil "github.com/vrooli/cli-core/cliapptest"
)

func TestFlowForgetRequiresExplicitCacheOnlyConfirmation(t *testing.T) {
	requests := 0
	server := testutil.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != http.MethodDelete || r.URL.Path != "/api/v1/flows/runs/run-1/cache" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		var body struct {
			Acknowledge bool `json:"acknowledge_volatile_cache_only"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || !body.Acknowledge {
			t.Fatalf("cache-only acknowledgment missing: %+v, err=%v", body, err)
		}
		_, _ = w.Write([]byte(`{"run_id":"run-1","scope":"volatile_run_cache","cache_deleted":true,"evidence_preserved":true,"audits_unchanged":true,"saved_flow_definitions_unchanged":true,"retained_store_unchanged":true}`))
	}))
	core, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{Name: "device-control-test", Version: "test", DefaultAPIBase: server.URL, AllowAnonymous: true})
	if err != nil {
		t.Fatal(err)
	}
	var cmd cliapp.Command
	for _, candidate := range FlowGroup(core).Subcommands {
		if candidate.Name == "forget" {
			cmd = candidate
		}
	}
	if cmd.RunCtx == nil {
		t.Fatal("missing forget command")
	}
	var out bytes.Buffer
	ctx, err := cliapp.NewTestRunContextFromArgs(cmd.Args, []string{"run-1", "--json"}, core, &out, &out)
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.RunCtx(ctx); err == nil {
		t.Fatal("forget command succeeded without explicit confirmation")
	}
	if requests != 0 {
		t.Fatalf("sent %d requests without confirmation", requests)
	}

	ctx, err = cliapp.NewTestRunContextFromArgs(cmd.Args, []string{"run-1", "--confirm-cache-only", "--json"}, core, &out, &out)
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.RunCtx(ctx); err != nil {
		t.Fatal(err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}
