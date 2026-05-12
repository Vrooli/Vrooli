package hostlifecycle

import (
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

func TestStartOptionsRequest(t *testing.T) {
	req := StartOptionsRequest("restart", "prompt-manager", lifecycle.StartOptions{
		BestEffort: true,
		CleanStale: true,
		CustomPath: "/tmp/scenario",
	})
	if req.Action != "restart" || req.Name != "prompt-manager" {
		t.Fatalf("unexpected action/name: %+v", req)
	}
	if !req.BestEffort || !req.CleanStale || req.CustomPath != "/tmp/scenario" {
		t.Fatalf("options not copied: %+v", req)
	}
}
