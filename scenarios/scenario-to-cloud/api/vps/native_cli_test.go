package vps

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/execplan"
	"scenario-to-cloud/reach"
)

// [REQ:STC-P0-027] Delivery places the bundle, the release manifest and the
// native control plane for the negotiated platform through reach.Deliver
// with the manifest's digest; a platform mismatch is refused before any
// byte moves and a bare bundle is refused before staging.
func TestReleaseDeliverPlacesReleaseSetForNegotiatedPlatform(t *testing.T) {
	stubHealth(t)
	target := newFakeTarget()
	w := loadFixtures(t).Workloads["stateless-web"]
	h := newHarness(t, target, w, "deliver")
	plan := h.compile(execplan.ScopeInstall, execplan.Observations{})
	deliver := plan.Action(execplan.OpReleaseDeliver)
	if deliver.Inputs["release_id"] != h.release {
		t.Fatalf("plan must bind the canonical release id: %q", deliver.Inputs["release_id"])
	}
	trace, execErr := h.execute(context.Background(), plan, Identity{OperationID: "op-1", Fence: 1})
	if execErr != nil {
		t.Fatalf("install: %v (%s)", execErr, execErr.ActionID)
	}
	roles := map[string]string{}
	for _, f := range target.delivered {
		roles[f.Role] = f.RemotePath
	}
	if roles["bundle"] != deliver.Inputs["destination"] || roles["release_manifest"] != deliver.Inputs["release_manifest"] || roles["native_cli"] != "/root/Vrooli/.vrooli/bin/vrooli" {
		t.Fatalf("delivered = %v", roles)
	}
	for _, f := range target.delivered {
		if f.Role == "native_cli" && (f.Mode != 0o755 || filepath.Base(f.LocalPath) != "vrooli-linux-amd64") {
			t.Fatalf("native cli delivery = %+v", f)
		}
	}
	var negotiated bool
	for _, call := range trace.Calls {
		if call.Kind == "negotiate" && call.ActionID == execplan.OpReleaseDeliver {
			negotiated = true
		}
	}
	if !negotiated {
		t.Fatal("delivery must negotiate the target platform before choosing the binary")
	}

	arm := newFakeTarget()
	arm.platform = "linux/arm64"
	h2 := newHarness(t, arm, w, "arm")
	_, execErr = h2.execute(context.Background(), h2.compile(execplan.ScopeInstall, execplan.Observations{}), Identity{OperationID: "op-1", Fence: 1})
	if execErr == nil || execErr.ActionID != execplan.OpReleaseDeliver || !apierrors.Is(execErr.Err, apierrors.CodeUnsupportedCapability) {
		t.Fatalf("platform mismatch must be refused at delivery: %v", execErr)
	}
	if len(arm.delivered) != 0 {
		t.Fatalf("nothing may be delivered on a platform mismatch: %v", arm.delivered)
	}

	bare := filepath.Join(t.TempDir(), "mini-vrooli.tar.gz")
	writeFile(t, bare, "bare")
	plan, err := CompilePlan(context.Background(), PlanRequest{Manifest: h.manifest, BundlePath: bare, Closure: h.closure, Scope: execplan.ScopeInstall})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Action(execplan.OpReleaseStage).Inputs["release_id"] != "" {
		t.Fatal("a bare bundle has no release id")
	}
	_, execErr = ExecutePlan(context.Background(), ExecuteRequest{Plan: plan, Manifest: h.manifest, BundlePath: bare, DeploymentID: h.depID, Runtime: Runtime{Reach: newFakeTarget(), Target: targetRef(), Identity: Identity{OperationID: "op-1", Fence: 1}}})
	if execErr == nil || execErr.ActionID != execplan.OpReleaseVerify || !apierrors.Is(execErr.Err, apierrors.CodeReleaseVerificationFailed) {
		t.Fatalf("a bare bundle must be refused at verification with release_verification_failed: %v", execErr)
	}
}

// [REQ:STC-P0-024] A Bridge-bound deployment gets a typed reach_unavailable
// for artifact delivery instead of a silent SSH fallback.
func TestReleaseDeliverSurfacesTransportRefusal(t *testing.T) {
	w := loadFixtures(t).Workloads["stateless-web"]
	h := newHarness(t, newFakeTarget(), w, "bridge")
	plan := h.compile(execplan.ScopeInstall, execplan.Observations{})
	refusing := &refusingReach{fakeTarget: newFakeTarget()}
	_, execErr := ExecutePlan(context.Background(), ExecuteRequest{Plan: plan, Manifest: h.manifest, BundlePath: h.bundle, DeploymentID: h.depID, Runtime: Runtime{Reach: refusing, Target: targetRef(), Identity: Identity{OperationID: "op-1", Fence: 1}}})
	if execErr == nil || execErr.ActionID != execplan.OpReleaseDeliver || !reach.IsKind(execErr.Err, reach.KindUnavailable) {
		t.Fatalf("expected reach_unavailable at delivery, got %v", execErr)
	}
	if !strings.Contains(execErr.Error(), "bridge artifact delivery") {
		t.Fatalf("refusal must name the pending owner: %v", execErr)
	}
}

type refusingReach struct{ *fakeTarget }

func (r *refusingReach) Deliver(context.Context, identityTargetRef, reach.Delivery) (reach.DeliveryReceipt, error) {
	return reach.DeliveryReceipt{}, &reach.Error{Kind: reach.KindUnavailable, Transport: "bridge", Detail: "bridge artifact delivery is not configured"}
}
