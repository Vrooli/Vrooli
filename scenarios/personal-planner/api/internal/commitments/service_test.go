package commitments

import (
	"context"
	"testing"
)

type fakeRepository struct {
	commitment Commitment
	revision   Revision
}

func (f *fakeRepository) List(context.Context) ([]Commitment, error) {
	return []Commitment{f.commitment}, nil
}
func (f *fakeRepository) Create(_ context.Context, c Commitment) (Commitment, error) {
	c.ID = "c-1"
	f.commitment = c
	return c, nil
}
func (f *fakeRepository) UpdateState(_ context.Context, id, state string, revision int64) (Commitment, error) {
	f.commitment.ID = id
	f.commitment.State = state
	f.commitment.Revision = revision + 1
	return f.commitment, nil
}
func (f *fakeRepository) Revise(_ context.Context, in ReviseInput) (Commitment, Revision, error) {
	f.commitment.ID = in.ID
	f.commitment.PromisedBoundary = in.PromisedBoundary
	f.commitment.Revision = in.ExpectedRevision + 1
	f.revision = Revision{CommitmentID: in.ID, PromisedBoundary: in.PromisedBoundary, Revision: f.commitment.Revision}
	return f.commitment, f.revision, nil
}

func TestServicePreservesExplicitPromiseBoundaries(t *testing.T) {
	repo := &fakeRepository{}
	service := NewService(repo)
	created, err := service.Create(context.Background(), CreateInput{Result: "Send the brief", PromisedBoundary: "2026-10-01"})
	if err != nil || created.State != StateProposed || created.Risk != RiskUnknown || created.AcknowledgmentStatus != AcknowledgmentUnknown {
		t.Fatalf("created=%+v err=%v", created, err)
	}
	updated, err := service.UpdateState(context.Background(), created.ID, StateActive, created.Revision)
	if err != nil || updated.State != StateActive {
		t.Fatalf("updated=%+v err=%v", updated, err)
	}
	revised, revision, err := service.Revise(context.Background(), ReviseInput{ID: created.ID, PromisedBoundary: "2026-10-03", Reason: "scope changed", ExpectedRevision: updated.Revision})
	if err != nil || revised.PromisedBoundary != "2026-10-03" || revision.Reason != "" || revision.Revision != revised.Revision {
		t.Fatalf("revised=%+v revision=%+v err=%v", revised, revision, err)
	}
}

func TestServiceRejectsPromisesWithoutBoundaries(t *testing.T) {
	_, err := NewService(&fakeRepository{}).Create(context.Background(), CreateInput{Result: "Something useful"})
	if err == nil {
		t.Fatal("expected missing boundary error")
	}
}
