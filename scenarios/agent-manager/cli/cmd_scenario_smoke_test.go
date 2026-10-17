package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

func TestScenarioSmokeHasAppliedProvenance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("projectRoot"); got != "/project" {
			t.Fatalf("projectRoot = %q", got)
		}
		_, _ = w.Write([]byte(`{"runGroups":[{"runId":"run-1","files":[{"state":"applied"}]}]}`))
	}))
	defer server.Close()

	found, err := scenarioSmokeHasAppliedProvenance(server.URL, "/project", "run-1")
	if err != nil || !found {
		t.Fatalf("scenarioSmokeHasAppliedProvenance() = (%t, %v), want (true, nil)", found, err)
	}
}

func TestScenarioSmokeHasAppliedProvenanceRejectsMissingAppliedFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"runGroups":[{"runId":"run-1","files":[{"state":"pending-review"}]}]}`))
	}))
	defer server.Close()

	found, err := scenarioSmokeHasAppliedProvenance(server.URL, "/project", "run-1")
	if err != nil || found {
		t.Fatalf("scenarioSmokeHasAppliedProvenance() = (%t, %v), want (false, nil)", found, err)
	}
}

func TestIsTerminalRunStatus(t *testing.T) {
	if !isTerminalRunStatus(domainpb.RunStatus_RUN_STATUS_COMPLETE) {
		t.Fatal("complete status must be terminal")
	}
	if isTerminalRunStatus(domainpb.RunStatus_RUN_STATUS_RUNNING) {
		t.Fatal("running status must not be terminal")
	}
}

func TestScenarioSmokeProfileEligible(t *testing.T) {
	if !scenarioSmokeProfileEligible(&domainpb.AgentProfile{RoleRef: "code.default"}) {
		t.Fatal("unrestricted non-protected profile must be eligible")
	}
	if scenarioSmokeProfileEligible(&domainpb.AgentProfile{AllowedTools: []string{"read"}}) {
		t.Fatal("restricted profile must not be eligible")
	}
	if scenarioSmokeProfileEligible(&domainpb.AgentProfile{SandboxConfig: &domainpb.SandboxConfig{Mode: domainpb.SandboxMode_SANDBOX_MODE_PROTECTED}}) {
		t.Fatal("protected profile must not be eligible")
	}
}
