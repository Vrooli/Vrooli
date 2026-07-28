package planrepair

import (
	"context"
	"testing"

	plansv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/plans"
	"swarm-manager/internal/planclient"
)

type fakeCandidateClient struct {
	input planclient.CandidateRevisionInput
}

func (f *fakeCandidateClient) CreateCandidateRevision(_ context.Context, input planclient.CandidateRevisionInput) (*plansv1.CandidateRevision, error) {
	f.input = input
	return &plansv1.CandidateRevision{Id: "candidate-1"}, nil
}

func (f *fakeCandidateClient) PreviewCandidateRevision(_ context.Context, id string) (*plansv1.CandidateRevisionPreview, error) {
	return &plansv1.CandidateRevisionPreview{Candidate: &plansv1.CandidateRevision{Id: id}, QualityStatus: "pass"}, nil
}

func (f *fakeCandidateClient) ApplyCandidateRevision(_ context.Context, id, _ string, _ bool) (*plansv1.ApplyCandidateRevisionResponse, error) {
	return &plansv1.ApplyCandidateRevisionResponse{Candidate: &plansv1.CandidateRevision{Id: id}}, nil
}

func TestCanonicalizeCreatesCandidateWithoutReplacingCanonicalPlan(t *testing.T) {
	client := &fakeCandidateClient{}
	preview, err := Canonicalize(context.Background(), client, "plan-1", "sha256:base", "exec-1", TerminalResult{Outcome: "ready", CandidatePlan: []byte(`{"title":"Repaired plan","purpose":"Use the repaired contract."}`)})
	if err != nil {
		t.Fatal(err)
	}
	if preview.GetCandidate().GetId() != "candidate-1" || client.input.PlanID != "plan-1" || client.input.ExpectedBaseContentHash != "sha256:base" || client.input.ProposalProvenance != "swarm-manager:repair/exec-1" {
		t.Fatalf("candidate = %#v, input = %#v", preview, client.input)
	}
}
