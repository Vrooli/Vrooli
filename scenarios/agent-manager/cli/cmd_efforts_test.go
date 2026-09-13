package main

import (
	"github.com/vrooli/cli-core/cliutil"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEffortCLISeparatesRecoveryVerificationFromDeliveryAndBenefit(t *testing.T) {
	for _, state := range []string{"pending", "progress-observed", "owner-wait", "failed"} {
		t.Run(state, func(t *testing.T) {
			d := &pb.EffortDirective{DirectiveId: "directive", Delivery: pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_DELIVERED, Assessment: "unknown", RecoveryExpectation: &pb.EffortRecoveryExpectation{ProgressCondition: "retained owner regression passes"}, RecoveryVerification: &pb.EffortRecoveryVerification{State: state, Reason: "owner receipt comparison", Verifier: "verified-actor", NextOwnerCondition: "assignment:next-check"}}
			output := captureStdout(t, func() error { printEffortDirective(d); return nil })
			for _, want := range []string{"delivery=DELIVERED", "assessment=unknown", "Recovery: " + state, "retained owner regression passes", "verifier=verified-actor", "next=assignment:next-check"} {
				if !strings.Contains(output, want) {
					t.Fatalf("missing distinct recovery evidence %q in %s", want, output)
				}
			}
		})
	}
}

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

func TestEffortLocalOwnerDoesNotElevateAgentRequests(t *testing.T) {
	services, recorder := newContractServices(t)
	app := &App{services: services}
	file := filepath.Join(t.TempDir(), "request.json")
	if err := os.WriteFile(file, []byte(`{"authority":"WATCH_AUTHORITY_FAMILY_PARENT"}`), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(cliutil.EnvIdentityToken, "identified-agent-fixture")
	if err := app.cmdEffort([]string{"assess", "--request-file", file, "--local-owner"}); err == nil || !strings.Contains(err.Error(), "identified agent") {
		t.Fatalf("agent must not invoke owner exchange: %v", err)
	}
	t.Setenv(cliutil.EnvIdentityToken, "")
	if err := app.cmdEffort([]string{"assess", "--request-file", file, "--local-owner"}); err == nil || !strings.Contains(err.Error(), "operator authority") {
		t.Fatalf("agent authority must not be upgraded: %v", err)
	}
	if err := app.cmdEffort([]string{"board", "--local-owner"}); err == nil {
		t.Fatal("read must not exchange unnecessary credentials")
	}
	if len(recorder.Requests()) != 0 {
		t.Fatal("rejected override made an API request")
	}
}

func TestEffortDispatcherCLIUsesTypedOwnerRPCs(t *testing.T) {
	services, recorder := newContractServices(t)
	app := &App{services: services}
	file := filepath.Join(t.TempDir(), "dispatch.json")
	if err := os.WriteFile(file, []byte(`{"effort_ref":"service:supervision","idempotency_key":"operator-operation"}`), 0600); err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{"issue-dispatch", "revoke-dispatch"} {
		if err := app.cmdEffort([]string{command}); err == nil {
			t.Fatal("missing typed request accepted")
		}
		if err := app.cmdEffort([]string{command, "--request-file", file, "--json"}); err != nil {
			t.Fatal(err)
		}
	}
	requests := recorder.Requests()
	if len(requests) != 2 || !strings.HasSuffix(requests[0].Path, "/IssueSupervisorDispatch") || !strings.HasSuffix(requests[1].Path, "/RevokeSupervisorDispatch") {
		t.Fatal("dispatcher CLI bypassed owner RPC")
	}
}
