package safety

import "testing"

func TestInvariantsCarryTheFinancialSafetyBoundary(t *testing.T) {
	joined := make(map[string]bool)
	for _, invariant := range Invariants() {
		joined[invariant] = true
	}

	for _, required := range []string{
		"AgentSpend exposes no policy-mutating method.",
		"Unverifiable agent identity fails closed.",
		"Only operator-owned funds are representable.",
		"Settlement retries never move value twice for one idempotency key.",
		"Evidence is append-only and retained for every spend attempt.",
	} {
		if !joined[required] {
			t.Fatalf("missing safety invariant %q", required)
		}
	}
}

func TestRegisterExposesLocalInvariantCommand(t *testing.T) {
	group := Register()
	if group.Name != "safety" {
		t.Fatalf("group name = %q, want safety", group.Name)
	}
	if len(group.Subcommands) != 1 || group.Subcommands[0].Name != "invariants" {
		t.Fatalf("subcommands = %#v, want one invariants command", group.Subcommands)
	}
}
