package cliutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// parkTestServer stands up an agent-manager stub serving /identity/verify (so
// ParkForAwait can recover the owning run id) and /runs/{id}/park. parkHandler
// receives the decoded park body for assertions and returns the response it
// produces.
func parkTestServer(t *testing.T, runID string, parkHandler func(body map[string]any) (status int, resp map[string]any)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST" && r.URL.Path == "/api/v1/identity/verify":
			json.NewEncoder(w).Encode(VerifyResult{
				Valid:     true,
				Claims:    &VerifiedClaims{RunID: runID},
				RunStatus: "running",
			})
		case r.Method == "POST" && r.URL.Path == "/api/v1/runs/"+runID+"/park":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode park body: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			status, resp := parkHandler(body)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			json.NewEncoder(w).Encode(resp)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestParkForAwait_NotAgentControlled(t *testing.T) {
	t.Setenv(EnvIdentityToken, "")

	result, parked, err := ParkForAwait(ParkRequest{Producer: ParkProducerTestGenie, Key: "demo/R1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parked {
		t.Fatal("expected parked=false outside an agent-manager run")
	}
	if result != nil {
		t.Fatalf("expected nil result, got %+v", result)
	}
}

func TestParkForAwait_Success(t *testing.T) {
	const runID = "run-uuid-abc"
	var gotBody map[string]any
	server := parkTestServer(t, runID, func(body map[string]any) (int, map[string]any) {
		gotBody = body
		return http.StatusOK, map[string]any{
			"success": true,
			"message": "PARKED — agent-manager is now waiting on test-genie:demo/R1 on your behalf.",
		}
	})
	defer server.Close()

	t.Setenv("AGENT_MANAGER_API_BASE", server.URL)
	t.Setenv(EnvIdentityToken, "valid-token")

	result, parked, err := ParkForAwait(ParkRequest{Producer: ParkProducerTestGenie, Key: "demo/R1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !parked {
		t.Fatal("expected parked=true")
	}
	if result == nil || !strings.Contains(result.Message, "PARKED") {
		t.Fatalf("expected PARKED message, got %+v", result)
	}
	if gotBody["producer"] != ParkProducerTestGenie {
		t.Errorf("producer = %v, want %q", gotBody["producer"], ParkProducerTestGenie)
	}
	if gotBody["key"] != "demo/R1" {
		t.Errorf("key = %v, want %q", gotBody["key"], "demo/R1")
	}
	if gotBody["identity_token"] != "valid-token" {
		t.Errorf("identity_token = %v, want %q", gotBody["identity_token"], "valid-token")
	}
	if _, present := gotBody["deadline_unix"]; present {
		t.Errorf("deadline_unix should be omitted for a zero deadline, got %v", gotBody["deadline_unix"])
	}
}

func TestParkForAwait_DeclinedIsError(t *testing.T) {
	const runID = "run-uuid-def"
	server := parkTestServer(t, runID, func(map[string]any) (int, map[string]any) {
		return http.StatusOK, map[string]any{"success": false}
	})
	defer server.Close()

	t.Setenv("AGENT_MANAGER_API_BASE", server.URL)
	t.Setenv(EnvIdentityToken, "valid-token")

	_, parked, err := ParkForAwait(ParkRequest{Producer: ParkProducerTestGenie, Key: "demo/R1"})
	if !parked {
		t.Fatal("expected parked=true (we are in an AM run, the call just failed)")
	}
	if err == nil {
		t.Fatal("expected an error when agent-manager declines to park")
	}
}

func TestParkForAwait_ParkEndpointErrorIsInAMRun(t *testing.T) {
	const runID = "run-uuid-ghi"
	server := parkTestServer(t, runID, func(map[string]any) (int, map[string]any) {
		return http.StatusForbidden, map[string]any{"error": "identity token does not own this run"}
	})
	defer server.Close()

	t.Setenv("AGENT_MANAGER_API_BASE", server.URL)
	t.Setenv(EnvIdentityToken, "valid-token")

	_, parked, err := ParkForAwait(ParkRequest{Producer: ParkProducerGCT, Key: "web/pre-launch"})
	if !parked {
		t.Fatal("expected parked=true (in an AM run) even on a park endpoint error")
	}
	if err == nil {
		t.Fatal("expected an error when the park endpoint rejects the request")
	}
}

func TestParkForAwait_PassesDeadline(t *testing.T) {
	const runID = "run-uuid-jkl"
	var gotBody map[string]any
	server := parkTestServer(t, runID, func(body map[string]any) (int, map[string]any) {
		gotBody = body
		return http.StatusOK, map[string]any{"success": true, "message": "ok"}
	})
	defer server.Close()

	t.Setenv("AGENT_MANAGER_API_BASE", server.URL)
	t.Setenv(EnvIdentityToken, "valid-token")

	deadline := time.Unix(1700000000, 0)
	_, parked, err := ParkForAwait(ParkRequest{Producer: ParkProducerTestGenie, Key: "demo/R2", Deadline: deadline})
	if err != nil || !parked {
		t.Fatalf("ParkForAwait: parked=%v err=%v", parked, err)
	}
	// JSON numbers decode to float64.
	if got, ok := gotBody["deadline_unix"].(float64); !ok || int64(got) != deadline.Unix() {
		t.Errorf("deadline_unix = %v, want %d", gotBody["deadline_unix"], deadline.Unix())
	}
}
