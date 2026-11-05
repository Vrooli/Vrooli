package renderjobs

import (
	"math"
	"strings"
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

func TestGeometrySessionSameScopeRefinementUsesPersistedParentConfirmation(t *testing.T) {
	c, ctx, _ := integrationCoordinator(t, &integrationOps{}, false)
	addGeometryPlate(c.media.(*integrationMedia))
	created, err := c.CreateGeometrySession(ctx, GeometrySessionRequest{
		ProjectID: "project", CandidateID: "candidate-base", IdempotencyKey: "session-confirmation",
		Brief: "compact field display", EnvelopeMM: [3]float64{80, 150, 12},
		ReferenceMediaIDs: []string{"active", "plate"}, FundingRoute: GeometryFundingLocal,
		ConfirmLocalAI: true,
	})
	if err != nil || len(created.Children) != 1 {
		t.Fatalf("create explicitly confirmed parent session: children=%d err=%v", len(created.Children), err)
	}
	_, err = c.RefineGeometrySession(ctx, created.SessionID, GeometrySessionRefinementRequest{
		GeometrySessionRequest: GeometrySessionRequest{
			CandidateID: "candidate-refined", IdempotencyKey: "session-confirmation/refine",
			Brief: created.Policy.Brief, EnvelopeMM: created.Policy.EnvelopeMM,
			ReferenceMediaIDs: append([]string(nil), created.Policy.ReferenceMediaIDs...),
			FundingRoute:      GeometryFundingLocal,
			// No second local-AI checkbox is sent for unchanged scope.
			ConfirmLocalAI: false,
		},
		CorrectionSourceJobID: created.Children[0].JobID,
	})
	if err == nil || !strings.Contains(err.Error(), "prior creation step must finish") {
		t.Fatalf("same-scope continuation should pass persisted parent confirmation and reach the baseline-state guard, got %v", err)
	}
}

func TestGeometrySessionChangedScopeReturnsChoiceWithoutDispatch(t *testing.T) {
	c, ctx, _ := integrationCoordinator(t, &integrationOps{}, false)
	addGeometryPlate(c.media.(*integrationMedia))
	created, err := c.CreateGeometrySession(ctx, GeometrySessionRequest{
		ProjectID: "project", CandidateID: "candidate-base", IdempotencyKey: "session-expansion",
		Brief: "compact field display", EnvelopeMM: [3]float64{80, 150, 12},
		ReferenceMediaIDs: []string{"active", "plate"}, FundingRoute: GeometryFundingLocal,
		ConfirmLocalAI: true,
	})
	if err != nil || len(created.Children) != 1 {
		t.Fatalf("create parent session: children=%d err=%v", len(created.Children), err)
	}
	snapshot, err := c.RefineGeometrySession(ctx, created.SessionID, GeometrySessionRefinementRequest{
		GeometrySessionRequest: GeometrySessionRequest{
			CandidateID: "candidate-expanded", IdempotencyKey: "session-expansion/linked",
			Brief: "add a physical keyboard", EnvelopeMM: created.Policy.EnvelopeMM,
			ReferenceMediaIDs: append([]string(nil), created.Policy.ReferenceMediaIDs...),
			FundingRoute:      GeometryFundingLocal, ConfirmLocalAI: false,
		},
	})
	if err != nil || !snapshot.NeedsConfirmation || len(snapshot.ChangedScope) != 1 || snapshot.ChangedScope[0] != "brief" || snapshot.ChildrenAdmitted != 1 || len(snapshot.Children) != 1 {
		t.Fatalf("scope change must pause without dispatch: snapshot=%+v err=%v", snapshot, err)
	}
}
