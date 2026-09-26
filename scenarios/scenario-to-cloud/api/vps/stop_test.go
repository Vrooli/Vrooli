package vps

import (
	"context"
	"os"
	"strings"
	"testing"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/execplan"
	"scenario-to-cloud/identity"
)

type identityTargetRef = identity.TargetRef

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// [REQ:STC-P0-028] workload.stop is one scoped lifecycle stop through the
// privilege broker: the argv names exactly one scenario and its workdir and
// carries no process pattern or port; the same inputs build the same
// invocation on every attempt (a replayed stop is a receipt replay).
func TestWorkloadStopIsAScopedLifecycleStop(t *testing.T) {
	cc := edgeContext("", "")
	action := execplan.Action{ID: execplan.OpWorkloadStop, OwnerOperation: execplan.OpWorkloadStop, Inputs: map[string]string{"workdir": "/root/Vrooli", "scenario": "app", "ports": "api=3001,ui=3000"}}
	first, err := ActionCommands(action, cc)
	if err != nil {
		t.Fatal(err)
	}
	second, _ := ActionCommands(action, cc)
	if strings.Join(first[0].Command.Argv(), " ") != strings.Join(second[0].Command.Argv(), " ") {
		t.Fatal("the stop invocation must be deterministic across attempts")
	}
	argv := strings.Join(first[0].Command.Argv(), " ")
	if !strings.Contains(argv, "--action process.stop.scoped") || !strings.Contains(argv, "--step workload.stop") {
		t.Fatalf("argv = %s", argv)
	}
	for _, forbidden := range []string{"pkill", "kill", "3000", "3001", "ss -"} {
		if strings.Contains(argv, forbidden) {
			t.Fatalf("stop must not carry %q: %s", forbidden, argv)
		}
	}
	var subject string
	for i, a := range first[0].Command.Args {
		if a == "--subject" {
			subject = first[0].Command.Args[i+1]
		}
	}
	raw, _ := DecodeJSONArg(subject)
	if string(raw) != `{"process":{"scenario":"app","workdir":"/root/Vrooli"}}` {
		t.Fatalf("subject = %s", raw)
	}
}

// [REQ:STC-P0-028] Retirement runs in owner order (route, runtime, grants,
// data, artifacts); an irreversible data disposition is refused with the
// missing owner and the retained disposition leaves every binding in place;
// active and previous artifacts are reported as retained.
func TestRetireOrderAndDataDisposition(t *testing.T) {
	stubHealth(t)
	target := newFakeTarget()
	w := loadFixtures(t).Workloads["sql-uploads"]
	h := newHarness(t, target, w, "v1")
	if _, execErr := h.execute(context.Background(), h.compile(execplan.ScopeFull, execplan.Observations{}), Identity{OperationID: "op-1", Fence: 1}); execErr != nil {
		t.Fatalf("deploy: %v", execErr)
	}
	if _, err := CompilePlan(context.Background(), PlanRequest{Manifest: h.manifest, BundlePath: h.bundle, Closure: h.closure, Scope: execplan.ScopeRetire}); err == nil || !apierrors.Is(err, apierrors.CodeInvalidRequest) {
		t.Fatalf("retire with persistent data and no retention policy must be refused: %v", err)
	}
	deleting, err := CompilePlan(context.Background(), PlanRequest{Manifest: h.manifest, BundlePath: h.bundle, Closure: h.closure, Scope: execplan.ScopeRetire, Policy: execplan.Policy{RetentionPolicy: execplan.RetentionDelete}, Deployment: ptr(domain0(h))})
	if err != nil {
		t.Fatal(err)
	}
	if ids := strings.Join(deleting.ActionIDs(), ","); ids != strings.Join([]string{execplan.OpEdgeRouteRetire, execplan.OpWorkloadStop, execplan.OpGrantsRevoke, execplan.OpDataRetire, execplan.OpArtifactsRetire}, ",") {
		t.Fatalf("retire order = %s", ids)
	}
	if deleting.Action(execplan.OpDataRetire).Effect != execplan.EffectDataWrite {
		t.Fatal("an irreversible disposition is a data write")
	}
	_, execErr := h.execute(context.Background(), deleting, Identity{OperationID: "op-2", Fence: 2})
	if execErr == nil || execErr.ActionID != execplan.OpDataRetire || !apierrors.Is(execErr.Err, apierrors.CodeUnsupportedCapability) {
		t.Fatalf("delete disposition must be refused by the missing owner, got %v", execErr)
	}
	if len(target.persistent["uploads"]) == 0 && target.persistent["uploads"] == nil {
		t.Fatal("persistent data must survive a refused deletion")
	}
	if target.running[h.manifest.Scenario.ID] || target.routes[h.manifest.Edge.Domain] != 0 {
		t.Fatalf("route and runtime must be retired before the data step: running=%v routes=%v", target.running, target.routes)
	}
	retaining, err := CompilePlan(context.Background(), PlanRequest{Manifest: h.manifest, BundlePath: h.bundle, Closure: h.closure, Scope: execplan.ScopeRetire, Policy: execplan.Policy{RetentionPolicy: execplan.RetentionRetain}, Deployment: ptr(domain0(h))})
	if err != nil {
		t.Fatal(err)
	}
	trace, execErr := h.execute(context.Background(), retaining, Identity{OperationID: "op-3", Fence: 3})
	if execErr != nil {
		t.Fatalf("retain: %v (%s)", execErr, execErr.ActionID)
	}
	last := trace.Actions[len(trace.Actions)-1]
	if last.ID != execplan.OpArtifactsRetire || !strings.Contains(last.Detail, "retained 1 ("+h.release) {
		t.Fatalf("artifact retirement must report the active release as retained: %+v", last)
	}
	if !strings.Contains(retaining.Presentation.RecoveryNote, "retain") {
		t.Fatalf("preview must state the data disposition: %q", retaining.Presentation.RecoveryNote)
	}
}
