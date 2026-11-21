package identity

import "testing"

// Backlog stamps the digest at acceptance and execution recomputes it at queue
// time. Both must agree that a declared default and an absent field are the same
// contract, and that a non-default value is a different one.
func TestPlanAcceptanceDigestTreatsDeclaredDefaultsAsAbsent(t *testing.T) {
	base := PlanAcceptanceContract{Kind: "execute", Name: "item", Title: "Item"}

	explicit := base
	explicit.Continuation = "manual"
	explicit.ScopePolicy = "fixed"
	if explicit.Digest() != base.Digest() {
		t.Fatal("spelling out the default continuation and scope policy changed the digest")
	}

	continued := base
	continued.Continuation = "until-allowance"
	if continued.Digest() == base.Digest() {
		t.Fatal("a non-default continuation did not change the digest")
	}

	extended := base
	extended.ScopePolicy = "extend-with-record"
	if extended.Digest() == base.Digest() {
		t.Fatal("a non-default scope policy did not change the digest")
	}
}
