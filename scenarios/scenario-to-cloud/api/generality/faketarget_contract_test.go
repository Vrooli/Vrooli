package generality

import (
	"context"
	"fmt"
	"testing"

	"scenario-to-cloud/identity"
	"scenario-to-cloud/reach"
	"scenario-to-cloud/vps"
)

// The target owner retains the displaced edge snippet and restores it when
// route rollback follows a deployment update. Keep that lifecycle behavior in
// the generality fake so a passing newcomer update does not hide unsafe
// rollback semantics.
func TestFakeTargetEdgeRollbackRestoresDisplacedRoute(t *testing.T) {
	target := newFakeTarget("edge-owner")
	ctx := context.Background()
	apply := func(operation, step string, fence uint64, port int) {
		t.Helper()
		spec, err := vps.EncodeJSONArg(map[string]any{
			"domain": "newcomer.example.test",
			"routes": []map[string]any{{"host": "app.newcomer.example.test", "upstream_port": port, "listener_id": "newcomer-service/web"}},
		})
		if err != nil {
			t.Fatal(err)
		}
		args := []string{"--deployment", "dep-newcomer", "--operation", operation, "--step", step, "--fence", fmt.Sprint(fence)}
		args = append(args, "--spec", spec)
		result, err := target.Exec(ctx, identity.TargetRef{}, reach.Command{Verb: "cloud-target edge route-apply", Args: args, Effectful: true, RequiredScope: "vrooli:write"})
		if err != nil || result.ExitCode != 0 {
			t.Fatalf("route apply port %d: result=%+v err=%v", port, result, err)
		}
	}
	apply("op-v1", "edge", 1, 3000)
	apply("op-v2", "edge", 2, 3100)

	result, err := target.Exec(ctx, identity.TargetRef{}, reach.Command{
		Verb:      "cloud-target edge route-rollback",
		Args:      []string{"--deployment", "dep-newcomer", "--operation", "op-rollback", "--step", "edge-rollback", "--fence", "3"},
		Effectful: true, RequiredScope: "vrooli:write",
	})
	if err != nil || result.ExitCode != 0 {
		t.Fatalf("route rollback: result=%+v err=%v", result, err)
	}
	if got := target.snapshot().Routes["app.newcomer.example.test"]; got.UpstreamPort != 3000 {
		t.Fatalf("rollback route = %+v, want displaced v1 upstream on port 3000", got)
	}
}
