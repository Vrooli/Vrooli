package targets

import (
	"strings"
	"testing"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/shared"
)

func TestTargetRowsExposeDispatchabilityAndPlatform(t *testing.T) {
	rows := targetRows([]*sharedv1.Target{{Id: "bridge-node:node-a", Kind: "bridge-node", Label: "Build node A", Os: "linux", Arch: "amd64", Dispatchable: true, State: sharedv1.TargetState_TARGET_STATE_DISPATCHABLE}})
	if len(rows) != 1 || !strings.Contains(rows[0], "bridge-node:node-a") || !strings.Contains(rows[0], "dispatchable=true") || !strings.Contains(rows[0], "linux/amd64") {
		t.Fatalf("targetRows() = %#v", rows)
	}
}

func TestReadinessRowsPreserveRecoveryFacts(t *testing.T) {
	rows := readinessRows(&sharedv1.Target{Readiness: []*sharedv1.ReadinessFact{{Label: "Heartbeat fresh", Passed: false, Detail: "last heartbeat is stale"}}})
	if len(rows) != 1 || !strings.Contains(rows[0], "Heartbeat fresh") || !strings.Contains(rows[0], "passed=false") || !strings.Contains(rows[0], "last heartbeat is stale") {
		t.Fatalf("readinessRows() = %#v", rows)
	}
}
