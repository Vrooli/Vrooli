package agentmanager

import (
	"testing"
	"time"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

func TestDefaultProfileRefUsesDeclaredProfileOnly(t *testing.T) {
	svc := NewAgentService(Config{
		ProfileKey: "test-genie/generation",
		Timeout:    5 * time.Second,
		Enabled:    true,
	})

	ref := svc.defaultProfileRef()
	if ref.ProfileKey != "test-genie/generation" {
		t.Fatalf("expected default profile ref to use profile key, got %q", ref.ProfileKey)
	}
	if ref.Defaults != nil || ref.UpdateExisting {
		t.Fatal("expected default profile ref to use only the reconciled profile key")
	}
}

func TestMapRunStatusRoundTrip(t *testing.T) {
	cases := map[domainpb.RunStatus]string{
		domainpb.RunStatus_RUN_STATUS_PENDING:      "pending",
		domainpb.RunStatus_RUN_STATUS_STARTING:     "pending",
		domainpb.RunStatus_RUN_STATUS_RUNNING:      "running",
		domainpb.RunStatus_RUN_STATUS_NEEDS_REVIEW: "running",
		domainpb.RunStatus_RUN_STATUS_COMPLETE:     "completed",
		domainpb.RunStatus_RUN_STATUS_FAILED:       "failed",
		domainpb.RunStatus_RUN_STATUS_CANCELLED:    "stopped",
		domainpb.RunStatus_RUN_STATUS_UNSPECIFIED:  "unknown",
	}

	for input, expected := range cases {
		if got := MapRunStatus(input); got != expected {
			t.Fatalf("MapRunStatus(%v) = %q, want %q", input, got, expected)
		}
	}

	if got := MapStatusToRun("completed"); got != domainpb.RunStatus_RUN_STATUS_COMPLETE {
		t.Fatalf("expected completed to map back to RUN_STATUS_COMPLETE, got %v", got)
	}
	if got := MapStatusToRun("bogus"); got != domainpb.RunStatus_RUN_STATUS_UNSPECIFIED {
		t.Fatalf("expected unknown status to map to unspecified, got %v", got)
	}
}
