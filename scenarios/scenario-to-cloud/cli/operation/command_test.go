package operation

import (
	"bytes"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/deployments/deploymentsv1connect"
	operationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/operations"
	"google.golang.org/protobuf/proto"

	"scenario-to-cloud/cli/internal/apierr"
	"scenario-to-cloud/cli/internal/protoout"
	"scenario-to-cloud/cli/internal/testfakes"
	"scenario-to-cloud/cli/internal/transport"
)

func testCommands(t *testing.T) (Commands, *testfakes.Server) {
	t.Helper()
	fake := testfakes.NewServer()
	fake.Deployments.Items = []testfakes.Deployment{{Ref: testfakes.Ref("dep-1", "demo", "production", "203.0.113.10"), Name: "demo", Status: "deployed"}}
	srv := httptest.NewServer(fake.Mux)
	t.Cleanup(srv.Close)
	tr := transport.ForBaseURL(srv.URL)
	return Commands{Client: NewClient(tr), Deployments: deploymentsv1connect.NewDeploymentsServiceClient(tr.HTTP, tr.BaseURL)}, fake
}

func pending(st *operationsv1.OperationStanding) *operationsv1.OperationStanding {
	st.StillPending = true
	st.RecommendedNextCheckSeconds = 90
	return st
}

// TestWaitExitCodesFollowTheStandingContract [REQ:STC-P0-037] [REQ:STC-P0-020]
// proves the CLI exit contract over the generated client: 0 succeeded,
// 1 failed, 3 pending, 124 observer timeout (operation unchanged, reattach
// command printed), and that wait sends the observer bound to the server.
func TestWaitExitCodesFollowTheStandingContract(t *testing.T) {
	c, fake := testCommands(t)
	fake.Operations.Script = []*operationsv1.OperationStanding{testfakes.Standing("op-1", "dep-1", "succeeded", true)}
	if err := c.Run([]string{"wait", "op-1", "--timeout", "45s"}); err != nil {
		t.Fatalf("succeeded must exit 0: %v", err)
	}
	if got := fake.Operations.WaitCalls[0].GetTimeoutSeconds(); got != 45 {
		t.Fatalf("timeout sent = %d, want 45", got)
	}

	fake.Operations.Script = []*operationsv1.OperationStanding{pending(testfakes.Standing("op-1", "dep-1", "running", false))}
	err := c.Run([]string{"wait", "op-1"})
	if apierr.ExitCode(err) != ExitObserverTimeout {
		t.Fatalf("still_pending must exit 124, got %d (%v)", apierr.ExitCode(err), err)
	}
	msg := apierr.Format(err)
	for _, want := range []string{"scenario-to-cloud operation wait op-1", "unchanged", "next check in 90s"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("observer timeout message missing %q: %s", want, msg)
		}
	}
	if len(fake.Operations.Cancelled) != 0 {
		t.Fatal("an observer timeout must never cancel the operation")
	}

	fake.Operations.Script = []*operationsv1.OperationStanding{testfakes.Standing("op-1", "dep-1", "running", false)}
	if code := apierr.ExitCode(c.Run([]string{"get", "op-1"})); code != ExitPending {
		t.Fatalf("non-terminal get must exit 3, got %d", code)
	}
	if code := apierr.ExitCode(c.Run([]string{"resume", "op-1"})); code != ExitPending {
		t.Fatalf("resume without still_pending must exit 3, got %d", code)
	}

	failed := testfakes.Standing("op-1", "dep-1", "failed", true)
	failed.Result = &operationsv1.OperationResult{Outcome: "failed", RecoveryOutcome: "service_restored", CompletedSteps: 1, Message: "activation failed"}
	fake.Operations.Script = []*operationsv1.OperationStanding{failed}
	if code := apierr.ExitCode(c.Run([]string{"wait", "op-1"})); code != ExitFailed {
		t.Fatalf("failed must exit 1 even when recovery restored service, got %d", code)
	}
	for _, state := range []string{"failed_recovery", "cancelled"} {
		fake.Operations.Script = []*operationsv1.OperationStanding{testfakes.Standing("op-1", "dep-1", state, true)}
		if code := apierr.ExitCode(c.Run([]string{"get", "op-1"})); code != ExitFailed {
			t.Fatalf("%s must exit 1, got %d", state, code)
		}
	}
}

// TestJSONOutputIsTheProtoStandingLossless [REQ:STC-P0-037] proves --json
// prints the server's standing as proto JSON that round-trips to an equal
// message.
func TestJSONOutputIsTheProtoStandingLossless(t *testing.T) {
	st := testfakes.Standing("op-9", "dep-1", "running", false)
	st.UnknownEffects = []*operationsv1.UnknownEffect{{Step: "release.activate", Fence: 3, Reason: "reply lost", Retry: "recover", NextAction: "read receipt"}}
	var out bytes.Buffer
	if err := protoout.Write(&out, st); err != nil {
		t.Fatal(err)
	}
	var back operationsv1.OperationStanding
	if err := protoout.Unmarshal(out.Bytes(), &back); err != nil {
		t.Fatalf("round trip: %v\n%s", err, out.String())
	}
	if !proto.Equal(st, &back) {
		t.Fatalf("JSON output is lossy:\n%s", out.String())
	}
	for _, want := range []string{`"operation_id":`, `"plan_digest":`, `"unknown_effects":`} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("proto JSON missing %s:\n%s", want, out.String())
		}
	}
}

// TestHumanStandingUsesSharedRendererAndIdentities [REQ:STC-P0-037] proves
// the human view carries the shared cli-core standing lines plus the
// operation identities (id, deployment, fence, plan digest, request key).
func TestHumanStandingUsesSharedRendererAndIdentities(t *testing.T) {
	var out bytes.Buffer
	st := testfakes.Standing("op-2", "dep-1", "running", false)
	if err := WriteStanding(&out, st); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Operation op-2 (deployment dep-1)", "lifecycle: running", "active phase: release.activate", "fence: 3", "plan digest: sha256:op-2", "request key: key-op-2", "completed steps: release.stage", "next: Wait once"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("human output missing %q:\n%s", want, out.String())
		}
	}
}

// TestListResolvesSelectorAndCancelRecordsIntent [REQ:STC-P0-037] proves
// list accepts the canonical selector forms and cancel records the intent
// once.
func TestListResolvesSelectorAndCancelRecordsIntent(t *testing.T) {
	c, fake := testCommands(t)
	fake.Operations.Script = []*operationsv1.OperationStanding{testfakes.Standing("op-1", "dep-1", "running", false)}
	if err := c.Run([]string{"list", "--scenario", "demo", "--environment", "production"}); err != nil {
		t.Fatalf("list by scenario/environment: %v", err)
	}
	if err := c.Run([]string{"list", "dep-1"}); err != nil {
		t.Fatalf("list by positional id: %v", err)
	}
	if got := fake.Operations.Listed; len(got) != 2 || got[0] != "dep-1" || got[1] != "dep-1" {
		t.Fatalf("listed = %v", got)
	}
	if err := c.Run([]string{"list", "--scenario", "demo"}); apierr.ExitCode(err) != ExitRefused {
		t.Fatalf("scenario without a facet must be refused locally: %v", err)
	}
	fake.Operations.Script = []*operationsv1.OperationStanding{testfakes.Standing("op-1", "dep-1", "cancel_requested", false)}
	if err := c.Run([]string{"cancel", "op-1"}); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if len(fake.Operations.Cancelled) != 1 || fake.Operations.Cancelled[0] != "op-1" {
		t.Fatalf("cancelled = %v", fake.Operations.Cancelled)
	}
}
