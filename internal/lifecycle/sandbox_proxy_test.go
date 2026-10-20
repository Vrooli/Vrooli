package lifecycle

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInSandbox(t *testing.T) {
	t.Setenv("VROOLI_SANDBOX_MERGED", "")
	if InSandbox() {
		t.Fatal("empty sandbox env should not be treated as sandbox")
	}
	t.Setenv("VROOLI_SANDBOX_MERGED", "/tmp/merged")
	if !InSandbox() {
		t.Fatal("non-empty sandbox merged path should be treated as sandbox")
	}
}

func TestWorkspaceSandboxBaseURLRequiresDedicatedTransport(t *testing.T) {
	t.Setenv(HostLifecycleBaseEnv, "")
	if _, err := workspaceSandboxBaseURL(); err == nil {
		t.Fatal("expected missing host lifecycle transport error")
	}

	t.Setenv(HostLifecycleBaseEnv, "not-a-url")
	if _, err := workspaceSandboxBaseURL(); err == nil {
		t.Fatal("expected invalid host lifecycle transport error")
	}
}

func TestRunScenarioUsesDedicatedTransport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/host/vrooli/scenario" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("method = %q", r.Method)
		}
		_, _ = w.Write([]byte(`{"success":true,"action":"port","name":"plan-manager","stdout":"19834\n"}`))
	}))
	defer server.Close()
	t.Setenv(HostLifecycleBaseEnv, server.URL+"/")

	resp, err := RunScenario(t.Context(), ScenarioRequest{Action: "port", Name: "plan-manager", PortName: "API_PORT"})
	if err != nil {
		t.Fatalf("RunScenario() error = %v", err)
	}
	if resp.Stdout != "19834\n" {
		t.Fatalf("stdout = %q", resp.Stdout)
	}
}

func TestRunCLIRefreshUsesTypedHostTransport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/host/vrooli/cli-refresh" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("method = %q", r.Method)
		}
		var got CLIRefreshRequest
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		want := CLIRefreshRequest{Kind: "scenario", Name: "scenario-qa", RunningExecutable: "/home/agent/.vrooli/bin/scenario-qa"}
		if got != want {
			t.Fatalf("request = %+v, want %+v", got, want)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":true,"exit_code":0}`))
	}))
	defer server.Close()
	t.Setenv(HostLifecycleBaseEnv, server.URL)

	resp, err := RunCLIRefresh(t.Context(), CLIRefreshRequest{
		Kind: "scenario", Name: "scenario-qa", RunningExecutable: "/home/agent/.vrooli/bin/scenario-qa",
	})
	if err != nil {
		t.Fatalf("RunCLIRefresh() error = %v", err)
	}
	if !resp.Success {
		t.Fatalf("RunCLIRefresh() response = %+v, want success", resp)
	}
}

func TestRunCLIRefreshPropagatesHostDenial(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"success":false,"exit_code":1,"stderr":"canonical target mismatch"}`))
	}))
	defer server.Close()
	t.Setenv(HostLifecycleBaseEnv, server.URL)

	_, err := RunCLIRefresh(t.Context(), CLIRefreshRequest{Kind: "scenario", Name: "scenario-qa", RunningExecutable: "/tmp/not-canonical"})
	if err == nil || !strings.Contains(err.Error(), "canonical target mismatch") {
		t.Fatalf("RunCLIRefresh() error = %v, want host target denial", err)
	}
}

func TestStartOptionsRequest(t *testing.T) {
	req := StartOptionsRequest("restart", "prompt-manager", StartOptions{
		BestEffort:           true,
		CleanStale:           true,
		DemandManaged:        true,
		AcceptCredentialLoss: true,
		CustomPath:           "/tmp/scenario",
	})
	if req.Action != "restart" || req.Name != "prompt-manager" {
		t.Fatalf("unexpected action/name: %+v", req)
	}
	if !req.BestEffort || !req.CleanStale || !req.DemandManaged || !req.AcceptCredentialLoss || req.CustomPath != "/tmp/scenario" {
		t.Fatalf("options not copied: %+v", req)
	}
}
