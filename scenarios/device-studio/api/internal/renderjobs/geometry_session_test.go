package renderjobs

import (
	"math"
	"testing"
)

func TestGeometrySessionPolicyPinsSourceScopeAndAllowsDistinctChildCandidates(t *testing.T) {
	base, err := canonicalGeometrySessionPolicy(GeometrySessionRequest{
		ProjectID: "project-1", CandidateID: "candidate-base", IdempotencyKey: "session-key",
		Brief: "unbranded display", EnvelopeMM: [3]float64{900, 645.45, 194.82},
		ReferenceMediaIDs: []string{"media-1"}, FundingRoute: GeometryFundingLocal,
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	if base.TokensPerCall != 4096 || base.FundingRoute != GeometryFundingLocal {
		t.Fatalf("policy did not pin bounded local execution: %#v", base)
	}
	child := base
	child.CandidateID = "candidate-refined"
	if changed := changedGeometrySessionScope(base, child); len(changed) != 0 {
		t.Fatalf("a new immutable output candidate must remain inside the same refinement scope, changed=%v", changed)
	}
	child.EnvelopeMM[0] = 901
	if changed := changedGeometrySessionScope(base, child); len(changed) != 1 || changed[0] != "envelope_mm" {
		t.Fatalf("locked-envelope change must require confirmation, changed=%v", changed)
	}
}

func TestGeometrySessionIdentityIsStableAndProjectScoped(t *testing.T) {
	one := geometrySessionID("project-1", "request-1")
	if one == "" || one != geometrySessionID("project-1", "request-1") {
		t.Fatal("same project/request must derive a stable session identity")
	}
	if one == geometrySessionID("project-2", "request-1") || one == geometrySessionID("project-1", "request-2") {
		t.Fatal("project and operation key must both participate in session identity")
	}
}

func TestGeometrySessionRejectsNonFiniteLockedDimensions(t *testing.T) {
	_, err := canonicalGeometrySessionPolicy(GeometrySessionRequest{
		ProjectID: "project-1", CandidateID: "candidate-1", IdempotencyKey: "request-1",
		Brief: "display", EnvelopeMM: [3]float64{100, math.NaN(), 20}, ReferenceMediaIDs: []string{"media-1"},
	}, "")
	if err == nil {
		t.Fatal("non-finite geometry envelope must be rejected before a durable scope is created")
	}
}
