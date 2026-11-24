package routing

import (
	"testing"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
)

func TestBuildCorrectionCandidateRankedReference(t *testing.T) {
	response := &routingv1.QueryResponse{Ranked: []*routingv1.SearchHit{
		{ProviderId: "docs", Id: "unrelated", Type: "doc", Path: "docs/other", Title: "Unrelated"},
		{ProviderId: "skills", Id: "manager-skill", Type: "skill", Path: "skills/manager", Title: "Manager Skill"},
	}}

	candidates := buildCorrectionCandidates("Managr", response)
	if len(candidates) != 1 {
		t.Fatalf("candidate count = %d, want 1: %+v", len(candidates), candidates)
	}
	candidate := candidates[0]
	if candidate.GetStrength() != routingv1.CorrectionCandidate_STRENGTH_BAND_POSSIBLE {
		t.Fatalf("strength = %s, want POSSIBLE", candidate.GetStrength())
	}
	ref := candidate.GetEvidence()
	if ref == nil {
		t.Fatal("candidate has no evidence reference")
	}
	if ref.GetProviderId() != "skills" || ref.GetRankedIndex() != 1 || ref.GetHitId() != "manager-skill" || ref.GetType() != "skill" || ref.GetPath() != "skills/manager" {
		t.Fatalf("ranked evidence = %+v, want exact provider, rank, and hit identity", ref)
	}
}
