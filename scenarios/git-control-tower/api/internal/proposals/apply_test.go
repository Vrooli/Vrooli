package proposals

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func indexSnapshot(r *MemoryGit) map[string]string {
	snapshot := map[string]string{}
	for path, blob := range r.index {
		snapshot[path] = blob
	}
	return snapshot
}

func TestApplyStagesExactPathsIncludingDeletionsAndCommitsRenderedMessage(t *testing.T) {
	svc := testService(t)
	git := NewMemoryGit(map[string]string{"a.go": "a1", "gone.go": "g1", "other.go": "o1"})
	repo := repoFor(git)
	git.Write("a.go", "a2")
	git.Write("new.go", "n1")
	git.Delete("gone.go")
	git.Write("other.go", "unrelated work stays uncommitted")
	view := createProposal(t, svc, repo, "a.go", "new.go", "gone.go")
	base := git.head

	result, err := svc.Apply(humanApplyContext(OperationApply), repo, ApplyRequest{ID: view.ID, Revision: 1, Actor: humanActor}, git.CommitIndex)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success || result.Refusal != nil || !result.Verified || result.CommitOID != git.head || result.CommitOID == base {
		t.Fatalf("result = %#v", result)
	}
	if !reflect.DeepEqual(git.stageCalls, [][]string{{"a.go", "gone.go", "new.go"}}) {
		t.Fatalf("stage calls = %v", git.stageCalls)
	}
	commit := git.commits[git.head]
	if commit.message != view.Message.Rendered || commit.parent != base {
		t.Fatalf("commit = %#v", commit)
	}
	if _, kept := commit.tree["gone.go"]; kept || commit.tree["new.go"] != memoryBlobID("n1") || commit.tree["other.go"] != memoryBlobID("o1") {
		t.Fatalf("commit tree = %#v", commit.tree)
	}
	if result.View.State != StateCommitted || result.View.CommitOID != git.head || result.View.CommittedAt == nil {
		t.Fatalf("proposal = %#v", result.View.Proposal)
	}
	last := result.View.Events[len(result.View.Events)-1]
	if last.Action != "committed" || last.Actor.Subject != "operator" {
		t.Fatalf("last event = %#v", last)
	}
}

func TestApplyRefusesContentDriftNamingEachFile(t *testing.T) {
	svc := testService(t)
	git := NewMemoryGit(map[string]string{"a.go": "a1", "b.go": "b1", "c.go": "c1"})
	repo := repoFor(git)
	for _, path := range []string{"a.go", "b.go", "c.go"} {
		git.Write(path, path+"-2")
	}
	view := createProposal(t, svc, repo, "a.go", "b.go", "c.go")
	git.Write("a.go", "drifted")
	git.Delete("c.go")
	head, index := git.head, indexSnapshot(git)

	result, err := svc.Apply(humanApplyContext(OperationApply), repo, ApplyRequest{ID: view.ID, Revision: 1, Actor: humanActor}, git.CommitIndex)
	if err != nil {
		t.Fatal(err)
	}
	if result.Success || result.Refusal == nil || result.Refusal.Code != RefuseContentDrift || !reflect.DeepEqual(result.Refusal.Paths, []string{"a.go", "c.go"}) {
		t.Fatalf("result = %#v", result)
	}
	if git.head != head || !reflect.DeepEqual(indexSnapshot(git), index) || len(git.stageCalls) != 0 {
		t.Fatal("a refused apply changed the repository")
	}
	if result.View.State != StateOpen || result.View.Events[len(result.View.Events)-1].Action != "apply_refused" {
		t.Fatalf("view = %#v", result.View.Proposal)
	}
}

func TestApplyRefusesBaseMoveThatTouchesProposalPaths(t *testing.T) {
	svc := testService(t)
	git := NewMemoryGit(map[string]string{"a.go": "a1", "b.go": "b1", "elsewhere.go": "e1"})
	repo := repoFor(git)
	git.Write("a.go", "a2")
	git.Write("b.go", "b2")
	view := createProposal(t, svc, repo, "a.go")

	git.CommitOutside(map[string]string{"elsewhere.go": "e2"})
	moved, err := svc.Apply(humanApplyContext(OperationApply), repo, ApplyRequest{ID: view.ID, Revision: 1, Actor: humanActor}, git.CommitIndex)
	if err != nil || !moved.Success {
		t.Fatalf("an unrelated base move must not block: %#v err=%v", moved, err)
	}

	second := createProposalFor(t, svc, repo, "E28", "b.go")
	git.CommitOutside(map[string]string{"b.go": "b-other"})
	git.Write("b.go", "b2")
	head := git.head
	result, err := svc.Apply(humanApplyContext(OperationApply), repo, ApplyRequest{ID: second.ID, Revision: 1, Actor: humanActor}, git.CommitIndex)
	if err != nil {
		t.Fatal(err)
	}
	if result.Refusal == nil || result.Refusal.Code != RefuseBaseMoved || !reflect.DeepEqual(result.Refusal.Paths, []string{"b.go"}) || git.head != head {
		t.Fatalf("result = %#v", result)
	}
}

func createProposalFor(t *testing.T, svc *Service, repo Repo, epoch string, paths ...string) View {
	t.Helper()
	work := epochWork()
	work.Epoch = epoch
	result, err := svc.Create(context.Background(), repo, CreateRequest{Work: work, Subject: "bas: " + epoch, Paths: paths, Actor: agentActor})
	if err != nil {
		t.Fatal(err)
	}
	return result.View
}

func TestApplyRefusesForeignStagedPaths(t *testing.T) {
	svc := testService(t)
	git := NewMemoryGit(map[string]string{"a.go": "a1", "mine.go": "m1"})
	repo := repoFor(git)
	git.Write("a.go", "a2")
	git.Write("mine.go", "operator's staged work")
	view := createProposal(t, svc, repo, "a.go")
	git.StageDirect("mine.go")
	index := indexSnapshot(git)

	result, err := svc.Apply(humanApplyContext(OperationApply), repo, ApplyRequest{ID: view.ID, Revision: 1, Actor: humanActor}, git.CommitIndex)
	if err != nil {
		t.Fatal(err)
	}
	if result.Refusal == nil || result.Refusal.Code != RefuseForeignStaged || !reflect.DeepEqual(result.Refusal.Paths, []string{"mine.go"}) {
		t.Fatalf("result = %#v", result)
	}
	if !reflect.DeepEqual(indexSnapshot(git), index) {
		t.Fatal("refusal changed the operator's staged work")
	}
}

// GCT-005: a committed proposal never commits again, whatever intent the
// second call carries.
func TestApplyReplayIsRefused(t *testing.T) {
	svc := testService(t)
	git := NewMemoryGit(map[string]string{"a.go": "a1"})
	repo := repoFor(git)
	git.Write("a.go", "a2")
	view := createProposal(t, svc, repo, "a.go")
	ctx := humanApplyContext(OperationApply)
	if first, err := svc.Apply(ctx, repo, ApplyRequest{ID: view.ID, Revision: 1, Actor: humanActor}, git.CommitIndex); err != nil || !first.Success {
		t.Fatalf("first = %#v err=%v", first, err)
	}
	head, commits := git.head, len(git.commits)
	replay, err := svc.Apply(ctx, repo, ApplyRequest{ID: view.ID, Revision: 1, Actor: humanActor}, git.CommitIndex)
	if err != nil {
		t.Fatal(err)
	}
	if replay.Success || replay.Refusal == nil || replay.Refusal.Code != RefuseState || git.head != head || len(git.commits) != commits {
		t.Fatalf("replay = %#v", replay)
	}
}

func TestApplyRefusesAStaleRevision(t *testing.T) {
	svc := testService(t)
	git := NewMemoryGit(map[string]string{"a.go": "a1"})
	repo := repoFor(git)
	git.Write("a.go", "a2")
	view := createProposal(t, svc, repo, "a.go")
	subject := "operator subject"
	if _, _, err := svc.Edit(context.Background(), repo, EditRequest{ID: view.ID, ExpectedRevision: 1, Subject: &subject, Actor: humanActor}); err != nil {
		t.Fatal(err)
	}
	result, err := svc.Apply(humanApplyContext(OperationApply), repo, ApplyRequest{ID: view.ID, Revision: 1, Actor: humanActor}, git.CommitIndex)
	if err != nil || result.Refusal == nil || result.Refusal.Code != RefuseRevision || len(git.stageCalls) != 0 {
		t.Fatalf("result = %#v err=%v", result, err)
	}
}

func TestApplyRestoresIndexWhenCommitFails(t *testing.T) {
	svc := testService(t)
	git := NewMemoryGit(map[string]string{"a.go": "a1", "gone.go": "g1"})
	repo := repoFor(git)
	git.Write("a.go", "a-staged-earlier")
	git.StageDirect("a.go")
	git.Write("a.go", "a-final")
	git.Delete("gone.go")
	view := createProposal(t, svc, repo, "a.go", "gone.go")
	head, index := git.head, indexSnapshot(git)

	failing := func(context.Context, string) (CommitOutcome, error) {
		return CommitOutcome{Failure: "precommit failed", Precommit: &PrecommitOutcome{Status: "failed", Summary: "Precommit checks failed", ExitCode: 2}}, nil
	}
	result, err := svc.Apply(humanApplyContext(OperationApply), repo, ApplyRequest{ID: view.ID, Revision: 1, Actor: humanActor}, failing)
	if err != nil {
		t.Fatal(err)
	}
	if result.Success || result.Refusal == nil || result.Refusal.Code != RefusePrecommit || result.Precommit == nil || !strings.Contains(result.Error, "index was restored") {
		t.Fatalf("result = %#v", result)
	}
	if git.head != head || !reflect.DeepEqual(indexSnapshot(git), index) {
		t.Fatalf("index not restored: %#v want %#v", indexSnapshot(git), index)
	}
	if result.View.State != StateOpen {
		t.Fatalf("failed commit changed state: %s", result.View.State)
	}
}

func TestApplyRefusesWhenAWriterRacesTheStagedIndex(t *testing.T) {
	svc := testService(t)
	git := NewMemoryGit(map[string]string{"a.go": "a1", "b.go": "b1"})
	repo := repoFor(git)
	git.Write("a.go", "a2")
	git.Write("b.go", "b2")
	view := createProposal(t, svc, repo, "a.go")
	index := indexSnapshot(git)
	git.afterStage = func() {
		git.StageDirect("b.go") // an external writer stages between our stage and commit
		git.index["a.go"] = memoryBlobID("rewritten by a hook")
	}
	result, err := svc.Apply(humanApplyContext(OperationApply), repo, ApplyRequest{ID: view.ID, Revision: 1, Actor: humanActor}, git.CommitIndex)
	if err != nil {
		t.Fatal(err)
	}
	if result.Refusal == nil || result.Refusal.Code != RefuseIndexMismatch || !reflect.DeepEqual(result.Refusal.Paths, []string{"a.go", "b.go"}) {
		t.Fatalf("result = %#v", result)
	}
	if git.index["a.go"] != index["a.go"] {
		t.Fatal("proposal path index entry was not restored")
	}
}

func TestApplyReportsACommitThatDiffersFromTheProposal(t *testing.T) {
	svc := testService(t)
	git := NewMemoryGit(map[string]string{"a.go": "a1"})
	repo := repoFor(git)
	git.Write("a.go", "a2")
	view := createProposal(t, svc, repo, "a.go")
	reformatting := func(ctx context.Context, message string) (CommitOutcome, error) {
		git.Write("a.go", "a2 reformatted by hook")
		git.StageDirect("a.go")
		return git.CommitIndex(ctx, message+"\n\nSigned-off-by: hook")
	}
	result, err := svc.Apply(humanApplyContext(OperationApply), repo, ApplyRequest{ID: view.ID, Revision: 1, Actor: humanActor}, reformatting)
	if err != nil || !result.Success || result.Verified || len(result.Notes) != 2 || result.View.State != StateCommitted {
		t.Fatalf("result = %#v err=%v", result, err)
	}
}

// GCT-003: the domain writer refuses agents and humans without a consumed
// apply intent, independent of transport.
func TestApplyRequiresHumanWithConsumedApplyIntent(t *testing.T) {
	svc := testService(t)
	git := NewMemoryGit(map[string]string{"a.go": "a1"})
	repo := repoFor(git)
	git.Write("a.go", "a2")
	view := createProposal(t, svc, repo, "a.go")
	for name, ctx := range map[string]context.Context{
		"no principal":      context.Background(),
		"commit intent":     humanApplyContext("repo.commit"),
		"agent with intent": agentContextWithIntent(),
	} {
		_, err := svc.Apply(ctx, repo, ApplyRequest{ID: view.ID, Revision: 1, Actor: humanActor}, git.CommitIndex)
		if !errors.Is(err, ErrForbidden) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
	if len(git.stageCalls) != 0 {
		t.Fatal("a refused caller reached the writer")
	}
}
