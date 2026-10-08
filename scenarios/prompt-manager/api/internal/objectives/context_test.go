package objectives

import (
	"context"
	"testing"
)

func teamContext(t *testing.T, svc *Service, teamID string) TeamObjectiveContext {
	t.Helper()
	projection, err := BuildProjection(context.Background(), svc)
	if err != nil {
		t.Fatalf("build projection: %v", err)
	}
	for _, tc := range BuildTeamObjectiveContexts(projection) {
		if tc.TeamID == teamID {
			return tc
		}
	}
	t.Fatalf("team %q has no runtime objective context", teamID)
	return TeamObjectiveContext{}
}

// TestTeamObjectiveContextOrderAndRevisions is the executable fixture for the
// Phase 6 runtime read contract. It proves the three consumer guarantees the
// plan requires: the read is in the team's priority order, it carries the
// authority's current meaning revision, and a strict meaning change re-pends
// restatement while a pure reorder does not.
func TestTeamObjectiveContextOrderAndRevisions(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t, nil)
	t1 := mustObjective(t, svc, "T1", ClassTerminal)
	i1 := mustObjective(t, svc, "I1", ClassInstrumental)

	if _, err := svc.Attach(ctx, Attachment{ObjectiveID: "T1", TeamID: "marketing-crew", Role: RolePrimary}, teamRevision(t, svc, "marketing-crew")); err != nil {
		t.Fatalf("attach T1: %v", err)
	}
	if _, err := svc.Attach(ctx, Attachment{ObjectiveID: "I1", TeamID: "marketing-crew"}, teamRevision(t, svc, "marketing-crew")); err != nil {
		t.Fatalf("attach I1: %v", err)
	}

	before := teamContext(t, svc, "marketing-crew")
	if len(before.Objectives) != 2 || before.Objectives[0].ObjectiveID != "T1" || before.Objectives[1].ObjectiveID != "I1" {
		t.Fatalf("expected T1 then I1 in priority order, got %+v", before.Objectives)
	}
	if before.Objectives[0].MeaningRevision != t1.MeaningRevision || before.Objectives[1].MeaningRevision != i1.MeaningRevision {
		t.Fatalf("context must carry the authority meaning revisions, got %+v", before.Objectives)
	}
	if !before.Objectives[0].RestatementPending || !before.Objectives[1].RestatementPending {
		t.Fatalf("a never-acknowledged attachment must be restatement-pending, got %+v", before.Objectives)
	}
	if before.AttachmentRevision == "" {
		t.Fatal("context must carry the team attachment revision")
	}

	for id, obj := range map[string]Objective{"T1": t1, "I1": i1} {
		if _, err := svc.Acknowledge(ctx, id, "marketing-crew", obj.MeaningRevision); err != nil {
			t.Fatalf("acknowledge %s: %v", id, err)
		}
	}
	acked := teamContext(t, svc, "marketing-crew")
	if acked.Objectives[0].RestatementPending || acked.Objectives[1].RestatementPending {
		t.Fatalf("acknowledging the current revision must clear restatement, got %+v", acked.Objectives)
	}

	// A pure reorder moves the attachment revision but must not re-pend either
	// acknowledgement.
	reordered, err := svc.ReorderTeamAttachments(ctx, "marketing-crew", []string{"I1", "T1"}, acked.AttachmentRevision)
	if err != nil {
		t.Fatalf("reorder: %v", err)
	}
	if reordered[0].ObjectiveID != "I1" || reordered[1].ObjectiveID != "T1" {
		t.Fatalf("expected I1 then T1 after reorder, got %+v", reordered)
	}
	after := teamContext(t, svc, "marketing-crew")
	if after.Objectives[0].ObjectiveID != "I1" || after.Objectives[1].ObjectiveID != "T1" {
		t.Fatalf("context must follow the reordered priority, got %+v", after.Objectives)
	}
	if after.Objectives[0].RestatementPending || after.Objectives[1].RestatementPending {
		t.Fatal("a pure reorder must not invalidate an acknowledgement")
	}
	if after.AttachmentRevision == acked.AttachmentRevision {
		t.Fatal("reordering must move the team attachment revision")
	}

	// A meaning change re-pends only the objective whose meaning moved.
	if _, err := svc.UpsertObjective(ctx, Objective{ID: "T1", Title: "Objective T1", Class: ClassTerminal, EvidenceSource: "revised owner evidence"}, t1.MeaningRevision); err != nil {
		t.Fatalf("restate T1: %v", err)
	}
	restated := teamContext(t, svc, "marketing-crew")
	var refT1, refI1 TeamObjectiveReference
	for _, ref := range restated.Objectives {
		switch ref.ObjectiveID {
		case "T1":
			refT1 = ref
		case "I1":
			refI1 = ref
		}
	}
	if !refT1.RestatementPending {
		t.Fatal("a strict meaning change must re-pend restatement")
	}
	if refT1.MeaningRevision == t1.MeaningRevision {
		t.Fatal("a strict meaning change must move the objective meaning revision")
	}
	if refI1.RestatementPending || refI1.MeaningRevision != i1.MeaningRevision {
		t.Fatalf("an unchanged objective must not re-pend or move, got %+v", refI1)
	}
}
