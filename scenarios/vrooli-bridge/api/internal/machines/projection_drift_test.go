package machines

import (
	"context"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/packages/proto/privilegedops"
)

func connectionPolicy() PolicySnapshot {
	return PolicySnapshot{ProfileID: "managed-connection", ProfileVersion: "v1", Scenarios: []string{"vrooli-bridge"}, RequiredCapabilities: []string{privilegedops.CapabilitySSHManagement}}
}

func trustedSSH() TrustRecord {
	return TrustRecord{HostKeyState: HostKeyVerified, ConnectionState: ConnectionTrusted, ClientKeyRef: "ssh-key://machine-1", SSHUser: "operator"}
}

func driftKinds(items []DriftItem) map[string]DriftItem {
	out := make(map[string]DriftItem, len(items))
	for _, item := range items {
		out[item.Kind+":"+item.Name] = item
	}
	return out
}

// A machine Bridge onboarded over SSH has a verified host key, a trusted
// connection, its own client key, and a login. That is SSH management; the
// node cannot report it, so it must not be drift.
func TestVerifiedSSHTrustSatisfiesSSHManagement(t *testing.T) {
	projection := Projection{HasNode: true, Presence: PresenceSnapshot{Connected: true}, Node: NodeSnapshot{ID: "node-1"}}
	projection = WithControlPlaneCapabilities(projection, trustedSSH())

	drift := driftKinds(ComputeDrift(Machine{}, connectionPolicy(), projection))
	if _, ok := drift["capability:"+privilegedops.CapabilitySSHManagement]; ok {
		t.Fatalf("drift = %+v, want SSH management satisfied by verified trust", drift)
	}
	if reasons := EvaluateReadiness(Machine{Lifecycle: LifecycleActive}, trustedSSH(), connectionPolicy(), projection).Reasons; len(reasons) != 0 {
		t.Fatalf("readiness reasons = %v, want none for a trusted connected machine", reasons)
	}
}

func TestUnverifiedTrustLeavesSSHManagementMissing(t *testing.T) {
	for name, trust := range map[string]TrustRecord{
		"unverified host key": {HostKeyState: HostKeyUnverified, ConnectionState: ConnectionTrusted, ClientKeyRef: "ssh-key://k", SSHUser: "u"},
		"no login":            {HostKeyState: HostKeyVerified, ConnectionState: ConnectionTrusted, ClientKeyRef: "ssh-key://k"},
		"no client key":       {HostKeyState: HostKeyVerified, ConnectionState: ConnectionTrusted, SSHUser: "u"},
	} {
		t.Run(name, func(t *testing.T) {
			projection := WithControlPlaneCapabilities(Projection{HasNode: true, Presence: PresenceSnapshot{Connected: true}}, trust)
			drift := driftKinds(ComputeDrift(Machine{}, connectionPolicy(), projection))
			if _, ok := drift["capability:"+privilegedops.CapabilitySSHManagement]; !ok {
				t.Fatalf("drift = %+v, want SSH management missing", drift)
			}
			if _, ok := drift["profile:managed-connection"]; !ok {
				t.Fatalf("drift = %+v, want the unapplied profile reported while its requirements are unmet", drift)
			}
		})
	}
}

// A connection-only profile asks for a connected agent and capabilities
// Bridge can see. A machine that visibly has them conforms, whether or not an
// apply was ever recorded.
func TestConnectedConformingMachineHasNoUnappliedProfileDrift(t *testing.T) {
	projection := WithControlPlaneCapabilities(Projection{HasNode: true, Presence: PresenceSnapshot{Connected: true}}, trustedSSH())
	if drift := ComputeDrift(Machine{}, connectionPolicy(), projection); len(drift) != 0 {
		t.Fatalf("drift = %+v, want none", drift)
	}
}

func TestProfileThatPlacesScenariosStillNeedsARecordedApply(t *testing.T) {
	policy := PolicySnapshot{ProfileID: "development-runner", ProfileVersion: "v1", Scenarios: []string{"test-genie", "vrooli-bridge"}}
	projection := Projection{HasNode: true, Presence: PresenceSnapshot{Connected: true}}
	if _, ok := driftKinds(ComputeDrift(Machine{}, policy, projection))["profile:development-runner"]; !ok {
		t.Fatal("want profile drift: Bridge cannot observe test-genie on the node")
	}
}

func TestUnappliedProfileNamesTheLastApplyOutcome(t *testing.T) {
	projection := Projection{HasNode: true, Node: NodeSnapshot{ConfigurationState: "onboarding_apply_failed", ConfigurationUnmet: []string{"a", "b", "c", "d"}}}
	item, ok := driftKinds(ComputeDrift(Machine{}, connectionPolicy(), projection))["profile:managed-connection"]
	if !ok {
		t.Fatal("want profile drift")
	}
	for _, want := range []string{"onboarding_apply_failed", "4 unmet", "a, b, c"} {
		if !strings.Contains(item.Reason, want) {
			t.Fatalf("reason = %q, want it to contain %q", item.Reason, want)
		}
	}
}

type missingNodeReader struct{}

func (missingNodeReader) GetNode(context.Context, string) (NodeSnapshot, error) {
	return NodeSnapshot{}, ErrNodeMissing
}

type offlinePresence struct{}

func (offlinePresence) GetPresence(context.Context, string) (PresenceSnapshot, error) {
	return PresenceSnapshot{}, nil
}

// A lineage that still names a deleted node is a machine without a current
// node, not an unreadable machine.
func TestComposeTreatsADeletedLineageNodeAsNoCurrentNode(t *testing.T) {
	machine := Machine{ID: "m-1", Lineage: []NodeLineage{{NodeID: "gone", Current: true}}}
	projection, err := Compose(context.Background(), machine, missingNodeReader{}, offlinePresence{})
	if err != nil {
		t.Fatalf("Compose() error = %v, want the machine to stay readable", err)
	}
	if projection.HasNode {
		t.Fatal("HasNode = true for a deleted node")
	}
	reasons := EvaluateReadiness(machine, TrustRecord{}, connectionPolicy(), projection).Reasons
	found := false
	for _, reason := range reasons {
		found = found || reason == "no_current_node"
	}
	if !found {
		t.Fatalf("readiness reasons = %v, want no_current_node", reasons)
	}
}
