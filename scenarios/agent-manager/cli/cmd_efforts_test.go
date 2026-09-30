package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

func TestEffortCLIUsesSharedBoardAndExplicitMutationRequests(t *testing.T) {
	services, server := newContractServices(t)
	app := &App{services: services}
	if err := app.cmdEffort([]string{"board", "--effort-ref", "effort:any", "--json"}); err != nil {
		t.Fatal(err)
	}
	if err := app.cmdEffort([]string{"enroll"}); err == nil {
		t.Fatal("mutation accepted without a request")
	}
	path := filepath.Join(t.TempDir(), "request.json")
	if err := os.WriteFile(path, []byte(`{"enrollment":{"effort_ref":"effort:any","work_shape":"investigation"},"idempotency_key":"one"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := app.cmdEffort([]string{"enroll", "--request-file", path, "--json"}); err != nil {
		t.Fatal(err)
	}
	requests := server.Requests()
	if len(requests) != 2 || requests[0].Path != "/agent_manager.v1.AgentManagerService/GetEffortBoard" || requests[1].Path != "/agent_manager.v1.AgentManagerService/EnrollEffort" {
		t.Fatal("CLI bypassed canonical RPC projection", requests)
	}
	// A partially populated wire row remains a useful human response, not a panic.
	if err := app.cmdEffort([]string{"board"}); err != nil {
		t.Fatal(err)
	}
}

func TestEffortCLICompactBoardPrintsJoinedOwnerFacts(t *testing.T) {
	board := &pb.EffortBoard{ActiveCount: 1, QuotaObservations: []*pb.EffortQuotaObservation{{Provider: "openai", Pool: "primary", Window: "daily", Standing: "reported", EvidenceRef: "quota:1"}}, Rows: []*pb.EffortBoardRow{{Enrollment: &pb.EffortEnrollment{EffortRef: "effort:arbitrary", DisplayName: "Arbitrary effort", TargetRevision: "accepted"}, RuntimeState: "active", Freshness: pb.EffortFreshness_EFFORT_FRESHNESS_FRESH, OutcomeStanding: &pb.EffortOutcomeStanding{State: "unknown"}, EvidenceRefs: []string{"checkpoint:changed"}, PendingOperations: []string{"owner:wait"}, Usage: &pb.EffortUsage{Partial: true}}}}
	output := captureStdout(t, func() error { printCompactEffortBoard(board); return nil })
	for _, want := range []string{"checkpoint:changed", "Named waits: owner:wait", "agent-manager:GetEffortBoard:effort:arbitrary", "Usage: tokens=unknown", "Quota observations (shared owner cut):", "openai/primary/daily", "quota:1"} {
		if !strings.Contains(output, want) {
			t.Fatalf("compact owner projection omitted %q: %s", want, output)
		}
	}
}

func TestEffortLocalOwnerDoesNotElevateAgentRequests(t *testing.T) {
	services, recorder := newContractServices(t)
	app := &App{services: services}
	file := filepath.Join(t.TempDir(), "request.json")
	if err := os.WriteFile(file, []byte(`{"authority":"WATCH_AUTHORITY_FAMILY_PARENT"}`), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(cliutil.EnvIdentityToken, "identified-agent-fixture")
	if err := app.cmdEffort([]string{"reconcile-metadata", "--request-file", file, "--local-owner"}); err == nil || !strings.Contains(err.Error(), "identified agent") {
		t.Fatalf("agent must not invoke owner exchange: %v", err)
	}
	t.Setenv(cliutil.EnvIdentityToken, "")
	if err := app.cmdEffort([]string{"reconcile-metadata", "--request-file", file, "--local-owner"}); err == nil || !strings.Contains(err.Error(), "operator authority") {
		t.Fatalf("agent authority must not be upgraded: %v", err)
	}
	if err := app.cmdEffort([]string{"board", "--local-owner"}); err == nil {
		t.Fatal("read must not exchange unnecessary credentials")
	}
	if len(recorder.Requests()) != 0 {
		t.Fatal("rejected override made an API request")
	}
}
