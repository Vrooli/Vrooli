package proposals

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

const (
	runE1 = "11111111-1111-4111-8111-111111111111"
	runE2 = "22222222-2222-4222-8222-222222222222"
	runE3 = "33333333-3333-4333-8333-333333333333"
)

func createEpoch(t *testing.T, svc *Service, repo Repo, epoch, run string, paths ...string) CreateResult {
	t.Helper()
	result, err := svc.Create(context.Background(), repo, CreateRequest{
		Work:    Work{EffortRef: "effort:bas", Epoch: epoch, RunIDs: []string{testRun, run}},
		Subject: "bas: work of " + epoch, Body: "Outcome of " + epoch + ".", Paths: paths, Actor: agentActor,
	})
	if err != nil {
		t.Fatalf("create %s: %v", epoch, err)
	}
	return result
}

func paths(files []File) []string {
	var result []string
	for _, file := range files {
		result = append(result, file.Path)
	}
	return result
}

func TestCreateFoldsOverlappingProposalsAndLeavesOthersSeparate(t *testing.T) {
	svc := testService(t)
	git := NewMemoryGit(map[string]string{"a.go": "a1", "b.go": "b1", "c.go": "c1"})
	repo := repoFor(git)
	git.Write("a.go", "a2")
	git.Write("b.go", "b2")
	e1 := createEpoch(t, svc, repo, "E1", runE1, "a.go", "b.go").View
	git.Write("c.go", "c2")
	e2 := createEpoch(t, svc, repo, "E2", runE2, "c.go").View

	git.Write("b.go", "b3")
	git.Write("d.go", "d1")
	result := createEpoch(t, svc, repo, "E3", runE3, "b.go", "d.go")
	combined := result.View

	if got := paths(combined.Files); !reflect.DeepEqual(got, []string{"a.go", "b.go", "d.go"}) {
		t.Fatalf("combined files = %v", got)
	}
	if a := fileByPath(t, combined.Files, "a.go"); a.Source != "folded:"+e1.ID || a.BlobID != memoryBlobID("a2") {
		t.Fatalf("carried a.go = %#v", a)
	}
	if !reflect.DeepEqual(result.SupersededIDs, []string{e1.ID}) || combined.Freshness.State != FreshnessFresh {
		t.Fatalf("result = %#v freshness = %#v", result.SupersededIDs, combined.Freshness)
	}
	if len(combined.Folds) != 1 || combined.Folds[0].ID != e1.ID || combined.Folds[0].Epoch != "E1" || combined.Folds[0].Reason != FoldOverlap {
		t.Fatalf("folds = %#v", combined.Folds)
	}
	rendered := combined.Message.Rendered
	for _, want := range []string{
		"Vrooli-Epoch: effort:bas#E1\nVrooli-Epoch: effort:bas#E3\n",
		"Vrooli-Run: " + runE1, "Vrooli-Run: " + runE3, "Vrooli-Proposal: " + combined.ID,
		"- E1 (" + e1.ID + "): bas: work of E1",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered message missing %q:\n%s", want, rendered)
		}
	}
	if strings.Contains(rendered, "#E2") || strings.Contains(rendered, runE2) {
		t.Fatalf("separate E2 leaked into the combined message:\n%s", rendered)
	}
	if created := combined.Events[0].Detail; !strings.Contains(created, "folded "+e1.ID+" (E1, overlapping_paths: b.go)") {
		t.Fatalf("created event = %q", created)
	}

	open, count, err := svc.List(context.Background(), repo, ListFilter{})
	if err != nil || count != 2 || len(open) != 2 || open[0].ID != e2.ID || open[1].ID != combined.ID {
		t.Fatalf("open = %v count = %d err = %v", open, count, err)
	}
	superseded, _, err := svc.List(context.Background(), repo, ListFilter{States: []State{StateSuperseded}})
	if err != nil || len(superseded) != 1 {
		t.Fatalf("superseded = %#v err = %v", superseded, err)
	}
	old := superseded[0]
	last := old.Events[len(old.Events)-1]
	if old.ID != e1.ID || old.SupersededBy != combined.ID || last.Action != "superseded" || !strings.Contains(last.Detail, "folded into "+combined.ID+" (E3): shares b.go") {
		t.Fatalf("history of folded proposal = %#v", old.Proposal)
	}
	if !reflect.DeepEqual(paths(old.Files), []string{"a.go", "b.go"}) {
		t.Fatalf("folded proposal lost its own record: %v", paths(old.Files))
	}
}

func TestDriftedProposalFoldsForwardAtCurrentContent(t *testing.T) {
	svc := testService(t)
	git := NewMemoryGit(map[string]string{"a.go": "a1", "c.go": "c1"})
	repo := repoFor(git)
	git.Write("a.go", "a2")
	e1 := createEpoch(t, svc, repo, "E1", runE1, "a.go").View
	git.Write("a.go", "a3 from a later epoch that did not list it")
	git.Write("c.go", "c2")

	combined := createEpoch(t, svc, repo, "E2", runE2, "c.go").View
	if got := paths(combined.Files); !reflect.DeepEqual(got, []string{"a.go", "c.go"}) {
		t.Fatalf("files = %v", got)
	}
	if a := fileByPath(t, combined.Files, "a.go"); a.BlobID != memoryBlobID("a3 from a later epoch that did not list it") {
		t.Fatalf("a.go was not carried at current content: %#v", a)
	}
	if combined.Freshness.State != FreshnessFresh || len(combined.Folds) != 1 || combined.Folds[0].Reason != FoldDrifted {
		t.Fatalf("freshness = %#v folds = %#v", combined.Freshness, combined.Folds)
	}
	old, err := svc.Get(context.Background(), repo, e1.ID)
	if err != nil || old.State != StateSuperseded || !strings.Contains(old.Events[len(old.Events)-1].Detail, "content drifted") {
		t.Fatalf("old = %#v err = %v", old.Proposal, err)
	}
}

func TestManyOpenProposalsNeverRefuseAnAcceptance(t *testing.T) {
	svc := testService(t)
	git := NewMemoryGit(map[string]string{})
	repo := repoFor(git)
	for n := 1; n <= 8; n++ {
		path := fmt.Sprintf("f%d.go", n)
		git.Write(path, "new")
		createEpoch(t, svc, repo, fmt.Sprintf("E%d", n), runE1, path)
	}
	if _, count, err := svc.List(context.Background(), repo, ListFilter{}); err != nil || count != 8 {
		t.Fatalf("open count = %d err = %v", count, err)
	}
}

func TestSameEpochReplacementKeepsCarryingFoldedEpochs(t *testing.T) {
	svc := testService(t)
	git := NewMemoryGit(map[string]string{"a.go": "a1", "b.go": "b1"})
	repo := repoFor(git)
	git.Write("a.go", "a2")
	e1 := createEpoch(t, svc, repo, "E1", runE1, "a.go").View
	git.Write("a.go", "a3")
	git.Write("b.go", "b2")
	first := createEpoch(t, svc, repo, "E2", runE2, "a.go", "b.go").View

	// The orchestrator re-submits E2 listing only its own file.
	result := createEpoch(t, svc, repo, "E2", runE2, "b.go")
	if result.SupersededID != first.ID {
		t.Fatalf("superseded = %v", result.SupersededIDs)
	}
	view := result.View
	if got := paths(view.Files); !reflect.DeepEqual(got, []string{"a.go", "b.go"}) {
		t.Fatalf("files = %v", got)
	}
	if len(view.Folds) != 1 || view.Folds[0].ID != e1.ID || !strings.Contains(view.Message.Rendered, "Vrooli-Epoch: effort:bas#E1\n") {
		t.Fatalf("folds = %#v message:\n%s", view.Folds, view.Message.Rendered)
	}
}

func TestFoldedPathCommittedByHandIsExcludedNotDropped(t *testing.T) {
	svc := testService(t)
	git := NewMemoryGit(map[string]string{"a.go": "a1", "b.go": "b1"})
	repo := repoFor(git)
	git.Write("a.go", "a2")
	git.Write("b.go", "b2")
	e1 := createEpoch(t, svc, repo, "E1", runE1, "a.go", "b.go").View
	git.CommitOutside(map[string]string{"a.go": "a2"})
	git.Write("b.go", "b3")

	view := createEpoch(t, svc, repo, "E2", runE2, "b.go").View
	if got := paths(view.Files); !reflect.DeepEqual(got, []string{"b.go"}) {
		t.Fatalf("files = %v", got)
	}
	if len(view.Excluded) != 1 || view.Excluded[0].Path != "a.go" || view.Excluded[0].Reason != "now_clean" || !strings.Contains(view.Excluded[0].Detail, e1.ID) {
		t.Fatalf("excluded = %#v", view.Excluded)
	}
}

func TestApplyOfACombinedProposalStagesExactlyTheUnion(t *testing.T) {
	svc := testService(t)
	git := NewMemoryGit(map[string]string{"a.go": "a1", "gone.go": "g1", "b.go": "b1", "other.go": "o1"})
	repo := repoFor(git)
	git.Write("a.go", "a2")
	git.Delete("gone.go")
	createEpoch(t, svc, repo, "E1", runE1, "a.go", "gone.go")
	git.Write("a.go", "a3")
	git.Write("b.go", "b2")
	git.Write("other.go", "unrelated work stays uncommitted")
	combined := createEpoch(t, svc, repo, "E2", runE2, "a.go", "b.go").View

	result, err := svc.Apply(humanApplyContext(OperationApply), repo, ApplyRequest{ID: combined.ID, Revision: combined.Revision, Actor: humanActor}, git.CommitIndex)
	if err != nil || !result.Success || !result.Verified {
		t.Fatalf("result = %#v err = %v", result, err)
	}
	if !reflect.DeepEqual(git.StageCalls(), [][]string{{"a.go", "b.go", "gone.go"}}) {
		t.Fatalf("stage calls = %v", git.StageCalls())
	}
	commit := git.commits[git.head]
	if commit.message != combined.Message.Rendered || commit.tree["other.go"] != memoryBlobID("o1") {
		t.Fatalf("commit = %#v", commit)
	}
	if _, kept := commit.tree["gone.go"]; kept {
		t.Fatalf("folded deletion was not committed: %#v", commit.tree)
	}
}
