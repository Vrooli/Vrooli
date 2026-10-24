package objectives

import (
	"context"
	"errors"
	"testing"

	"prompt-manager/internal/testsqlite"

	"github.com/vrooli/api-core/database"
)

func newTestService(t *testing.T, teamKnown TeamLookup) *Service {
	t.Helper()
	db := testsqlite.Open(t)
	if err := database.EnsureSchemas(context.Background(), db.Primary(), database.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("apply objectives schema: %v", err)
	}
	return NewService(NewRepository(db), teamKnown)
}

func mustObjective(t *testing.T, svc *Service, id string, class Class) Objective {
	t.Helper()
	obj, err := svc.UpsertObjective(context.Background(), Objective{
		ID:             id,
		Title:          "Objective " + id,
		Class:          class,
		EvidenceSource: "owner evidence for " + id,
	}, "")
	if err != nil {
		t.Fatalf("create objective %s: %v", id, err)
	}
	return obj
}

func teamRevision(t *testing.T, svc *Service, teamID string) string {
	t.Helper()
	rev, err := svc.TeamAttachmentRevision(context.Background(), teamID)
	if err != nil {
		t.Fatalf("team revision %s: %v", teamID, err)
	}
	return rev
}

func TestObjectiveMeaningRevisionTracksAcknowledgement(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t, nil)
	obj := mustObjective(t, svc, "T1", ClassTerminal)
	if obj.MeaningRevision == "" {
		t.Fatal("expected a computed meaning revision")
	}

	att, err := svc.Attach(ctx, Attachment{ObjectiveID: "T1", TeamID: "marketing-crew", Role: RolePrimary, Coverage: CoverageFull}, teamRevision(t, svc, "marketing-crew"))
	if err != nil {
		t.Fatalf("attach: %v", err)
	}
	if !att.RestatementPending {
		t.Fatal("a fresh attachment with no acknowledgement must be restatement-pending")
	}
	if att.AttachmentRevision == "" {
		t.Fatal("expected a team attachment revision")
	}

	acked, err := svc.Acknowledge(ctx, "T1", "marketing-crew", obj.MeaningRevision)
	if err != nil {
		t.Fatalf("acknowledge: %v", err)
	}
	if acked.RestatementPending {
		t.Fatal("acknowledging the current revision must clear restatement-pending")
	}

	changed, err := svc.UpsertObjective(ctx, Objective{ID: "T1", Title: "Objective T1", Class: ClassTerminal, EvidenceSource: "revised owner evidence"}, obj.MeaningRevision)
	if err != nil {
		t.Fatalf("meaning update: %v", err)
	}
	if changed.MeaningRevision == obj.MeaningRevision {
		t.Fatal("changing evidence expectations must move the meaning revision")
	}

	atts, err := svc.ListTeamAttachments(ctx, "marketing-crew")
	if err != nil {
		t.Fatalf("list attachments: %v", err)
	}
	if len(atts) != 1 || !atts[0].RestatementPending {
		t.Fatalf("expected the meaning edit to re-pend restatement, got %+v", atts)
	}
}

func TestReorderSeparatesOrderFromMeaning(t *testing.T) {
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
	if _, err := svc.Acknowledge(ctx, "T1", "marketing-crew", t1.MeaningRevision); err != nil {
		t.Fatalf("ack T1: %v", err)
	}
	if _, err := svc.Acknowledge(ctx, "I1", "marketing-crew", i1.MeaningRevision); err != nil {
		t.Fatalf("ack I1: %v", err)
	}
	before := teamRevision(t, svc, "marketing-crew")

	reordered, err := svc.ReorderTeamAttachments(ctx, "marketing-crew", []string{"I1", "T1"}, before)
	if err != nil {
		t.Fatalf("reorder: %v", err)
	}
	if len(reordered) != 2 || reordered[0].ObjectiveID != "I1" || reordered[1].ObjectiveID != "T1" {
		t.Fatalf("expected I1 then T1, got %+v", reordered)
	}
	if reordered[0].RestatementPending || reordered[1].RestatementPending {
		t.Fatal("a pure reorder must not invalidate an acknowledgement")
	}
	if after := teamRevision(t, svc, "marketing-crew"); after == before {
		t.Fatal("reordering must move the team attachment revision")
	}

	var storedT1 Objective
	objs, err := svc.ListObjectives(ctx)
	if err != nil {
		t.Fatalf("list objectives: %v", err)
	}
	for _, o := range objs {
		if o.ID == "T1" {
			storedT1 = o
		}
	}
	if storedT1.MeaningRevision != t1.MeaningRevision {
		t.Fatal("reordering must not change objective meaning revisions")
	}
}

func TestMutationGuards(t *testing.T) {
	ctx := context.Background()
	known := map[string]bool{"marketing-crew": true}
	svc := newTestService(t, func(teamID string) bool { return known[teamID] })
	t1 := mustObjective(t, svc, "T1", ClassTerminal)
	mustObjective(t, svc, "T2", ClassInstrumental)

	if _, err := svc.Attach(ctx, Attachment{ObjectiveID: "MISSING", TeamID: "marketing-crew"}, ""); !errors.Is(err, ErrUnknownObjective) {
		t.Fatalf("expected unknown objective, got %v", err)
	}
	if _, err := svc.Attach(ctx, Attachment{ObjectiveID: "T1", TeamID: "marketing-crew", Role: "boss"}, ""); !errors.Is(err, ErrInvalidRole) {
		t.Fatalf("expected invalid role, got %v", err)
	}
	if _, err := svc.Attach(ctx, Attachment{ObjectiveID: "T1", TeamID: "unknown-team"}, ""); !errors.Is(err, ErrUnknownTeam) {
		t.Fatalf("expected unknown team, got %v", err)
	}

	if _, err := svc.Attach(ctx, Attachment{ObjectiveID: "T1", TeamID: "marketing-crew"}, ""); err != nil {
		t.Fatalf("attach: %v", err)
	}
	if _, err := svc.Attach(ctx, Attachment{ObjectiveID: "T1", TeamID: "marketing-crew"}, teamRevision(t, svc, "marketing-crew")); !errors.Is(err, ErrDuplicateLink) {
		t.Fatalf("expected duplicate link, got %v", err)
	}
	if _, err := svc.Attach(ctx, Attachment{ObjectiveID: "T2", TeamID: "marketing-crew"}, "stale"); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected stale team revision conflict, got %v", err)
	}

	if _, err := svc.UpsertObjective(ctx, Objective{ID: "T1", Title: "changed", Class: ClassTerminal}, "stale"); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected stale meaning revision conflict, got %v", err)
	}

	if err := svc.DeleteObjective(ctx, "T1", t1.MeaningRevision); !errors.Is(err, ErrActiveReferences) {
		t.Fatalf("expected active references refusal, got %v", err)
	}
}

func TestRelationCycleRejected(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t, nil)
	mustObjective(t, svc, "T1", ClassTerminal)
	mustObjective(t, svc, "I1", ClassInstrumental)
	mustObjective(t, svc, "I2", ClassInstrumental)

	if err := svc.AddRelation(ctx, "I1", "T1"); err != nil {
		t.Fatalf("I1 -> T1: %v", err)
	}
	if err := svc.AddRelation(ctx, "I2", "I1"); err != nil {
		t.Fatalf("I2 -> I1: %v", err)
	}
	if err := svc.AddRelation(ctx, "I1", "I2"); !errors.Is(err, ErrCycle) {
		t.Fatalf("expected cycle refusal, got %v", err)
	}
	if err := svc.AddRelation(ctx, "T1", "I1"); !errors.Is(err, ErrInvalidClass) {
		t.Fatalf("expected non-instrumental source refusal, got %v", err)
	}
	if err := svc.AddRelation(ctx, "I1", "MISSING"); !errors.Is(err, ErrUnknownObjective) {
		t.Fatalf("expected unknown relation target, got %v", err)
	}
}

func TestValidateReportsCurrentStateFindings(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t, nil)
	mustObjective(t, svc, "T1", ClassTerminal)
	_, err := svc.UpsertObjective(ctx, Objective{ID: "T2", Title: "No evidence", Class: ClassTerminal}, "")
	if err != nil {
		t.Fatalf("create T2: %v", err)
	}

	result, err := svc.Validate(ctx)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if result.Errors != 0 {
		t.Fatalf("expected no validation errors, got %+v", result.Findings)
	}
	var served, unmeasurable bool
	for _, f := range result.Findings {
		if f.Rule == "objective_unserved" && f.ObjectiveID == "T1" {
			served = true
		}
		if f.Rule == "objective_unmeasurable" && f.ObjectiveID == "T2" {
			unmeasurable = true
		}
	}
	if !served || !unmeasurable {
		t.Fatalf("expected unserved and unmeasurable findings, got %+v", result.Findings)
	}
}
