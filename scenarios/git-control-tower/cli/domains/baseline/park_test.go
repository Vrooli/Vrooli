package baseline

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

// TestMain clears VROOLI_AGENT_IDENTITY_TOKEN for the whole package so a
// `baseline diff` test run inside an agent-manager run (the agent's token leaking
// into the test process) cannot accidentally PARK the agent's own run. Tests that
// exercise the park path opt back in via t.Setenv.
func TestMain(m *testing.M) {
	os.Unsetenv(cliutil.EnvIdentityToken)
	os.Exit(m.Run())
}

func TestParkForDiff_NotAgentControlled(t *testing.T) {
	_, parked, err := parkForDiff("web-console", "pre-launch")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parked {
		t.Fatal("expected parked=false outside an agent-manager run")
	}
}

// TestParkForDiff_ParksInsideAgentManagerRun proves the gct producer parks with
// the git-control-tower producer key and a "<scenario>/<name>" await key.
func TestParkForDiff_ParksInsideAgentManagerRun(t *testing.T) {
	const runID = "run-uuid-gct"
	am := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/identity/verify":
			json.NewEncoder(w).Encode(cliutil.VerifyResult{
				Valid:     true,
				Claims:    &cliutil.VerifiedClaims{RunID: runID},
				RunStatus: "running",
			})
		case "/api/v1/runs/" + runID + "/park":
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			if body["producer"] != cliutil.ParkProducerGCT {
				t.Errorf("producer = %v, want %q", body["producer"], cliutil.ParkProducerGCT)
			}
			if body["key"] != "web-console/pre-launch" {
				t.Errorf("key = %v, want web-console/pre-launch", body["key"])
			}
			json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"message": "PARKED — waiting on git-control-tower:web-console/pre-launch",
			})
		default:
			t.Errorf("unexpected AM request: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer am.Close()

	t.Setenv("AGENT_MANAGER_API_BASE", am.URL)
	t.Setenv(cliutil.EnvIdentityToken, "agent-token")

	park, parked, err := parkForDiff("web-console", "pre-launch")
	if err != nil {
		t.Fatalf("parkForDiff: %v", err)
	}
	if !parked {
		t.Fatal("expected parked=true inside an agent-manager run")
	}
	if park == nil || !strings.Contains(park.Message, "PARKED") {
		t.Fatalf("expected PARKED message, got %+v", park)
	}
}
