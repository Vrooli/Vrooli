package deployment

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/execplan"
	"scenario-to-cloud/identity"
	"scenario-to-cloud/operations"
	"scenario-to-cloud/reach"
)

type fakeReach struct {
	commands []reach.Command
	targets  []identity.TargetRef
	result   reach.Result
	err      error
}

func (f *fakeReach) Exec(_ context.Context, target identity.TargetRef, cmd reach.Command) (reach.Result, error) {
	f.commands = append(f.commands, cmd)
	f.targets = append(f.targets, target)
	return f.result, f.err
}

func (f *fakeReach) Deliver(context.Context, identity.TargetRef, reach.Delivery) (reach.DeliveryReceipt, error) {
	return reach.DeliveryReceipt{}, nil
}

func (f *fakeReach) Negotiate(context.Context, identity.TargetRef) (reach.Capabilities, error) {
	return reach.Capabilities{}, nil
}

type fakeRefRepo struct{ ref *identity.DeploymentRef }

func (r fakeRefRepo) GetDeploymentRef(context.Context, string) (*identity.DeploymentRef, error) {
	return r.ref, nil
}

// [REQ:STC-P0-024] Receipt reads travel through reach as argv on the bound
// transport; a missing native owner becomes ErrNativeCLIAbsent so the worker
// records an unknown effect instead of replaying.
func TestReachTargetReceiptsReadsThroughBoundTransport(t *testing.T) {
	target := identity.TargetRef{MachineID: "m-1", NodeID: "n-1", EnrollmentGeneration: 7, Transport: identity.TransportBridge}
	ref := &identity.DeploymentRef{ID: "dep-1", Target: target}
	plan, _ := json.Marshal(execplan.Plan{DeploymentID: "dep-1", Target: execplan.Target{MachineID: "m-1", NodeID: "n-1", EnrollmentGeneration: 7, Transport: string(identity.TransportBridge)}})
	fr := &fakeReach{result: reach.Result{ExitCode: 0, Stdout: `{"schema_version":1,"outcome":"succeeded","fence":4}`}}
	reader := &ReachTargetReceipts{Reach: fr, Repo: fakeRefRepo{ref: ref}}
	receipt, err := reader.Read(context.Background(), &domain.CloudOperation{ID: "op-1", DeploymentID: "dep-1", Plan: plan}, "release.stage")
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Found || receipt.Outcome != operations.StepSucceeded || receipt.Fence != 4 {
		t.Fatalf("receipt = %+v", receipt)
	}
	if fr.targets[0] != target || fr.commands[0].Verb != "cloud-target receipt get" || fr.commands[0].RequiredScope != "vrooli:read" {
		t.Fatalf("command = %+v target = %+v", fr.commands[0], fr.targets[0])
	}
	for _, arg := range fr.commands[0].Args {
		if err := reach.ValidateArgs([]string{arg}); err != nil {
			t.Fatalf("argument %q is not argv-safe: %v", arg, err)
		}
	}

	absent := &ReachTargetReceipts{Reach: &fakeReach{err: &reach.Error{Kind: reach.KindProtocolUnsupported}}, Repo: fakeRefRepo{ref: ref}}
	if _, err := absent.Read(context.Background(), &domain.CloudOperation{ID: "op-1", DeploymentID: "dep-1", Plan: plan}, "release.stage"); !errors.Is(err, operations.ErrNativeCLIAbsent) {
		t.Fatalf("expected ErrNativeCLIAbsent, got %v", err)
	}

	revoked := &ReachTargetReceipts{Reach: &fakeReach{err: &reach.Error{Kind: reach.KindEnrollmentRevoked}}, Repo: fakeRefRepo{ref: ref}}
	if _, err := revoked.Read(context.Background(), &domain.CloudOperation{ID: "op-1", DeploymentID: "dep-1", Plan: plan}, "release.stage"); !reach.IsKind(err, reach.KindEnrollmentRevoked) {
		t.Fatalf("revocation must surface as a typed reach error, got %v", err)
	}

	unsafe := &ReachTargetReceipts{Reach: fr, Repo: fakeRefRepo{ref: ref}}
	if _, err := unsafe.Read(context.Background(), &domain.CloudOperation{ID: "op;1", DeploymentID: "dep-1", Plan: plan}, "x"); err == nil {
		t.Fatal("unsafe identifier must be refused before reach")
	}
	changed := &ReachTargetReceipts{Reach: fr, Repo: fakeRefRepo{ref: &identity.DeploymentRef{ID: "dep-1", Target: identity.TargetRef{MachineID: "m-2", NodeID: "n-2", EnrollmentGeneration: 8, Transport: identity.TransportBridge}}}}
	before := len(fr.commands)
	if _, err := changed.Read(context.Background(), &domain.CloudOperation{ID: "op-1", DeploymentID: "dep-1", Plan: plan}, "release.stage"); err == nil || len(fr.commands) != before {
		t.Fatalf("changed deployment binding must refuse before receipt read, err=%v commands=%d", err, len(fr.commands)-before)
	}
}

func TestAdmittedTargetBindingMustStillMatchBeforeRecovery(t *testing.T) {
	dep := &domain.Deployment{ID: "dep-1", Target: identity.TargetRef{MachineID: "m-1", NodeID: "n-1", EnrollmentGeneration: 7, Transport: identity.TransportBridge}}
	plan := &execplan.Plan{DeploymentID: "dep-1", Target: execplan.Target{MachineID: "m-1", NodeID: "n-1", EnrollmentGeneration: 7, Transport: string(identity.TransportBridge)}}
	if err := verifyAdmittedTargetBinding(dep, plan); err != nil {
		t.Fatalf("matching admitted target should pass: %v", err)
	}
	dep.Target.EnrollmentGeneration++
	if err := verifyAdmittedTargetBinding(dep, plan); err == nil {
		t.Fatal("recovery must refuse a changed enrollment generation")
	}
}
