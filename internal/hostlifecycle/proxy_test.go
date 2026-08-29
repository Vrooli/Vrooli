package hostlifecycle

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrooli/vrooli/internal/lifecycle"
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

func TestStartOptionsRequest(t *testing.T) {
	req := StartOptionsRequest("restart", "prompt-manager", lifecycle.StartOptions{
		BestEffort:           true,
		CleanStale:           true,
		AcceptCredentialLoss: true,
		CustomPath:           "/tmp/scenario",
	})
	if req.Action != "restart" || req.Name != "prompt-manager" {
		t.Fatalf("unexpected action/name: %+v", req)
	}
	if !req.BestEffort || !req.CleanStale || !req.AcceptCredentialLoss || req.CustomPath != "/tmp/scenario" {
		t.Fatalf("options not copied: %+v", req)
	}
}
