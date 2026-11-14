// [REQ:GCT-PROPOSAL-CAPTURE]
package proposals

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func refs(ps ...View) []RevisionReference {
	result := make([]RevisionReference, 0, len(ps))
	for _, p := range ps {
		result = append(result, RevisionReference{ID: p.ID, Revision: p.Revision})
	}
	return result
}

func TestExplicitCombinePreservesSourceRevisionsAndExactUnion(t *testing.T) {
	svc := testService(t)
	git := NewMemoryGit(map[string]string{"a.go": "a1", "b.go": "b1", "c.go": "c1"})
	repo := repoFor(git)
	git.Write("a.go", "a2")
	git.Write("b.go", "b2")
	a := createProposalFor(t, svc, repo, "E1", "a.go", "b.go")
	git.Write("c.go", "c2")
	b := createProposalFor(t, svc, repo, "E2", "b.go", "c.go")
	if _, open, _ := svc.List(context.Background(), repo, ListFilter{}); open != 2 {
		t.Fatal("submission silently combined proposals")
	}
	combined, err := svc.Combine(context.Background(), repo, CombineRequest{Sources: refs(a, b), Subject: "bas: reviewed package", Actor: humanActor})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(combined.Paths(), []string{"a.go", "b.go", "c.go"}) || len(combined.Folds) != 2 || combined.Folds[0].Revision != a.Revision {
		t.Fatalf("combined=%#v", combined)
	}
	for _, source := range []View{a, b} {
		old, err := svc.Get(context.Background(), repo, source.ID)
		if err != nil || old.State != StateSuperseded || old.SupersededBy != combined.ID {
			t.Fatalf("source=%#v err=%v", old, err)
		}
		retained, err := svc.Revision(context.Background(), repo, source.ID, source.Revision)
		if err != nil || retained.State != StateOpen || retained.Message.Rendered != source.Message.Rendered {
			t.Fatal("source revision was rewritten")
		}
	}
	applied, err := svc.Apply(humanApplyContext(OperationApply), repo, ApplyRequest{ID: combined.ID, Revision: combined.Revision, Actor: humanActor, AcknowledgePriorWork: true}, git.CommitIndex)
	if err != nil || !applied.Success || !applied.Verified {
		t.Fatalf("apply=%#v err=%v", applied, err)
	}
}

func TestExplicitCombineRejectsDifferentCapturedVersionsWithoutClosingSources(t *testing.T) {
	svc := testService(t)
	git := NewMemoryGit(map[string]string{"a.go": "a1"})
	repo := repoFor(git)
	git.Write("a.go", "a2")
	a := createProposalFor(t, svc, repo, "E1", "a.go")
	git.Write("a.go", "a3")
	b := createProposalFor(t, svc, repo, "E2", "a.go")
	_, err := svc.Combine(context.Background(), repo, CombineRequest{Sources: refs(a, b), Subject: "combine", Actor: humanActor})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("err=%v", err)
	}
	if _, open, _ := svc.List(context.Background(), repo, ListFilter{}); open != 2 {
		t.Fatal("conflict closed a source")
	}
}

func TestExplicitCombineRejectsStaleReviewedSource(t *testing.T) {
	svc := testService(t)
	git := NewMemoryGit(map[string]string{"a.go": "a1", "b.go": "b1"})
	repo := repoFor(git)
	git.Write("a.go", "a2")
	a := createProposalFor(t, svc, repo, "E1", "a.go")
	git.Write("b.go", "b2")
	b := createProposalFor(t, svc, repo, "E2", "b.go")
	body := "operator edits"
	if _, _, err := svc.Edit(context.Background(), repo, EditRequest{ID: a.ID, ExpectedRevision: a.Revision, Body: &body, Actor: humanActor}); err != nil {
		t.Fatal(err)
	}
	_, err := svc.Combine(context.Background(), repo, CombineRequest{Sources: refs(a, b), Subject: "combine", Actor: humanActor})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("err=%v", err)
	}
	if _, open, _ := svc.List(context.Background(), repo, ListFilter{}); open != 2 {
		t.Fatal("stale combine altered inbox")
	}
}

func TestCapturedRevisionSurvivesDriftRefreshAndClosure(t *testing.T) {
	svc := testService(t)
	git := NewMemoryGit(map[string]string{"a.go": "original"})
	repo := repoFor(git)
	git.Write("a.go", "captured")
	v := createProposal(t, svc, repo, "a.go")
	git.Write("a.go", "later")
	updated, _, err := svc.Refresh(context.Background(), repo, v.ID, v.Revision, humanActor)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Close(context.Background(), repo, v.ID, updated.Revision, "obsolete", "", humanActor); err != nil {
		t.Fatal(err)
	}
	_, before, after, err := svc.ReviewFile(context.Background(), repo, v.ID, v.Revision, "a.go")
	if err != nil || string(before) != "original" || string(after) != "captured" {
		t.Fatalf("retained %q→%q err=%v", before, after, err)
	}
	if git.worktree["a.go"] != "later" {
		t.Fatal("closure changed source code")
	}
}

func TestApplyRequiresExplicitPriorWorkAcknowledgment(t *testing.T) {
	svc := testService(t)
	git := NewMemoryGit(map[string]string{"a.go": "a1"})
	repo := repoFor(git)
	git.Write("a.go", "a2")
	v := createProposal(t, svc, repo, "a.go")
	head, index := git.head, indexSnapshot(git)
	got, err := svc.Apply(humanApplyContext(OperationApply), repo, ApplyRequest{ID: v.ID, Revision: v.Revision, Actor: humanActor}, git.CommitIndex)
	if err != nil || got.Refusal == nil || got.Refusal.Code != "prior_work_acknowledgment" || git.head != head || !reflect.DeepEqual(index, indexSnapshot(git)) {
		t.Fatalf("result=%#v err=%v", got, err)
	}
}

func TestEditDuringChecksRefusesPublication(t *testing.T) {
	svc := testService(t)
	git := NewMemoryGit(map[string]string{"a.go": "a1"})
	repo := repoFor(git)
	git.Write("a.go", "a2")
	v := createProposal(t, svc, repo, "a.go")
	head, index := git.head, indexSnapshot(git)
	prepare := func(ctx context.Context, message string) (CommitOutcome, error) {
		candidate, err := git.CommitIndex(ctx, message)
		if err != nil {
			return candidate, err
		}
		subject := "new operator wording"
		_, _, err = svc.Edit(ctx, repo, EditRequest{ID: v.ID, ExpectedRevision: v.Revision, Subject: &subject, Actor: humanActor})
		return candidate, err
	}
	got, err := svc.Apply(humanApplyContext(OperationApply), repo, ApplyRequest{ID: v.ID, Revision: v.Revision, Actor: humanActor, AcknowledgePriorWork: true}, prepare)
	if err != nil || got.Success || got.Error == "" || git.head != head || !reflect.DeepEqual(index, indexSnapshot(git)) {
		t.Fatalf("result=%#v err=%v", got, err)
	}
}
