package hostcapability

import "testing"

func TestGuardFreezesEveryCoupledMember(t *testing.T) {
	set := CoupledSet{Name: "gpu-driver-health", Members: []string{"kernel", "module"}}
	result := ReconcileCoupledSet(UpdateGuard, set)
	if len(result.FrozenMembers) != len(set.Members) {
		t.Fatalf("frozen members = %v, want %v", result.FrozenMembers, set.Members)
	}
	if result.Policy == "" || result.Policy == "kernel" {
		t.Fatalf("policy = %q, want owning header and all members", result.Policy)
	}
}

func TestObserveDoesNotProduceHostPolicy(t *testing.T) {
	result := ReconcileCoupledSet(UpdateObserve, CoupledSet{Name: "gpu-driver-health", Members: []string{"kernel", "module"}})
	if result.Policy != "" || len(result.FrozenMembers) != 0 {
		t.Fatalf("observe result = %+v, want no host policy", result)
	}
}

func TestOwnIsTypedNotImplemented(t *testing.T) {
	result := ReconcileCoupledSet(UpdateOwn, CoupledSet{Name: "gpu-driver-health", Members: []string{"kernel"}})
	if result.Implemented || result.Reason == "" {
		t.Fatalf("own result = %+v, want typed not-implemented", result)
	}
}
