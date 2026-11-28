package lifecycle

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/scenario"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

func TestEnvironmentIdentityTransitionRetainsWorkAcrossQuiesceLoss(t *testing.T) {
	values := map[string]string{
		"SCENARIO_NAME":              "operator-workbench",
		"VROOLI_RUNTIME_INSTANCE_ID": "instance-7",
	}
	getenv := func(key string) string { return values[key] }
	quiesceCalls := 0
	p, err := NewProviderFromEnv(getenv, Hooks{
		Snapshot: func(context.Context) (Inventory, error) {
			return Inventory{Complete: true, Work: []*commonv1.LifecycleWorkItem{{Id: "draft-42", Kind: "operator-draft", Status: "saved"}}}, nil
		},
		Quiesce: func(context.Context) error {
			quiesceCalls++
			if quiesceCalls == 1 {
				return errors.New("injected quiesce provider loss")
			}
			return nil
		},
		Resume: func(context.Context) error { return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.scenario != "operator-workbench" {
		t.Fatalf("legacy environment resolved scenario %q", p.scenario)
	}
	ctx := context.Background()
	prepared, err := p.Prepare(ctx, connect.NewRequest(&commonv1.LifecyclePrepareRequest{OperationId: "replace-1", Reason: "configuration transition"}))
	if err != nil {
		t.Fatal(err)
	}
	fence := prepared.Msg.GetStanding()
	drain := &commonv1.LifecycleDrainRequest{OperationId: fence.GetOperationId(), FenceToken: fence.GetFenceToken(), Revision: fence.GetRevision(), QuiesceComponents: true}
	if _, err := p.Drain(ctx, connect.NewRequest(drain)); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("first drain error = %v, want injected provider loss", err)
	}
	status, err := p.Status(ctx, connect.NewRequest(&commonv1.LifecycleStatusRequest{OperationId: fence.GetOperationId(), FenceToken: fence.GetFenceToken(), Revision: fence.GetRevision()}))
	if err != nil {
		t.Fatal(err)
	}
	if got := status.Msg.GetStanding().GetWork(); len(got) != 1 || got[0].GetId() != "draft-42" || got[0].GetStatus() != "saved" {
		t.Fatalf("work after provider loss = %v, want saved draft-42", got)
	}

	// A control-plane reconfiguration may supply the canonical variable while
	// retaining the old alias during rollout. The lifecycle owner must still
	// resolve the same identity and retry the same fence without replaying work.
	values["VROOLI_SCENARIO"] = "operator-workbench"
	values["SCENARIO_NAME"] = "stale-alias"
	if got := scenario.IdentityFromEnv(getenv); got != p.scenario {
		t.Fatalf("reconfigured identity = %q, want %q", got, p.scenario)
	}
	if _, err := p.Drain(ctx, connect.NewRequest(drain)); err != nil {
		t.Fatalf("recovery drain: %v", err)
	}
	if quiesceCalls != 2 {
		t.Fatalf("quiesce calls = %d, want one retry", quiesceCalls)
	}
}
