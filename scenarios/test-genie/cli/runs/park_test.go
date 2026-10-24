package runs

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

// TestMain clears VROOLI_AGENT_IDENTITY_TOKEN for the whole package so a `runs
// wait` test run inside an agent-manager run (the agent's token leaking into the
// test process) cannot accidentally PARK the agent's own run. Tests that exercise
// the park path opt back in via t.Setenv.
func TestMain(m *testing.M) {
	os.Unsetenv(cliutil.EnvIdentityToken)
	os.Exit(m.Run())
}

// TestRunWaitJSONParksInsideAgentManagerRun proves that the canonical agent
// wait parks. Agent Manager performs the actual JSON wait from its non-agent
// process and resumes the agent with the terminal snapshot.
func TestRunWaitJSONParksInsideAgentManagerRun(t *testing.T) {
	const runID = "run-uuid-tg"
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
			if body["producer"] != cliutil.ParkProducerTestGenie {
				t.Errorf("producer = %v", body["producer"])
			}
			if body["key"] != "demo/R" {
				t.Errorf("key = %v, want demo/R", body["key"])
			}
			json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"message": "PARKED — waiting on test-genie:demo/R",
			})
		default:
			t.Errorf("unexpected AM request: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer am.Close()

	t.Setenv("AGENT_MANAGER_API_BASE", am.URL)
	t.Setenv(cliutil.EnvIdentityToken, "agent-token")

	// Contain a park-failure regression: if park did not short-circuit, runWait
	// would build the client and hit this (empty) stream server rather than a real
	// backend.
	withStreamServer(t, &streamServer{})

	var buf bytes.Buffer
	if err := runWait(nil, []string{"--json", "demo", "R"}, &buf); err != nil {
		t.Fatalf("runWait should park cleanly, got: %v", err)
	}
	if !strings.Contains(buf.String(), "PARKED") {
		t.Fatalf("expected the parked tool-result, got: %q", buf.String())
	}
}
