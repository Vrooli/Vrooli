package scenarioruntime

import "testing"

func TestDependencyDemandIdentityIsVariantScopedAndBounded(t *testing.T) {
	live := DependencyConsumerID("program-runtime", "")
	shadow := DependencyConsumerID("program-runtime", "shadow")
	if live == shadow || live != "scenario-dependency:program-runtime" || shadow != "scenario-dependency:program-runtime@shadow" {
		t.Fatalf("consumer identities live=%q shadow=%q", live, shadow)
	}
	if !IsDependencyConsumerID(live) || IsDependencyConsumerID("program-runtime") {
		t.Fatalf("dependency consumer recognition is incorrect")
	}
	if key, ok := DependencyConsumerInstance(shadow); !ok || key.Slug() != "program-runtime@shadow" {
		t.Fatalf("consumer instance = %#v, ok=%v", key, ok)
	}
	liveLease := DependencyLeaseID(live, "ai-gateway", "")
	shadowLease := DependencyLeaseID(shadow, "ai-gateway", "")
	if liveLease == shadowLease || len(liveLease) > 200 || len(shadowLease) > 200 {
		t.Fatalf("lease identities live=%q shadow=%q", liveLease, shadowLease)
	}
}
