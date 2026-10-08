package proposals

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"git-control-tower/internal/trailers"
)

func TestAnchorFingerprintsOnlyDirtyFilesInScope(t *testing.T) {
	git := NewMemoryGit(map[string]string{"scenarios/bas/a.go": "a1", "packages/proto/p.proto": "p1", "docs/x.md": "x1"})
	git.Write("scenarios/bas/a.go", "a2")
	git.Write("scenarios/bas/new.go", "n1")
	git.Delete("packages/proto/p.proto")
	git.Write("docs/x.md", "x2")
	anchor, err := testService(t).Anchor(context.Background(), repoFor(git), AnchorRequest{EffortRef: "effort:bas", Epoch: "27", Scopes: []string{"scenarios/bas", "packages/proto/**"}, Actor: agentActor})
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, file := range anchor.Files {
		paths = append(paths, file.Path)
	}
	if !reflect.DeepEqual(paths, []string{"packages/proto/p.proto", "scenarios/bas/a.go", "scenarios/bas/new.go"}) || anchor.Epoch != "E27" || anchor.Head != git.head {
		t.Fatalf("anchor = %#v", anchor)
	}
	if !anchor.Files[0].Deleted || anchor.Files[1].BlobID != memoryBlobID("a2") {
		t.Fatalf("anchor files = %#v", anchor.Files)
	}
}

func TestCreateFlagsMixedFilesAndRecordsUnlistedDelta(t *testing.T) {
	svc := testService(t)
	git := NewMemoryGit(map[string]string{"scenarios/bas/old.go": "o1", "scenarios/bas/clean.go": "c1"})
	repo := repoFor(git)
	git.Write("scenarios/bas/old.go", "weeks-old-change")
	if _, err := svc.Anchor(context.Background(), repo, AnchorRequest{EffortRef: "effort:bas", Epoch: "E27", Scopes: []string{"scenarios/bas"}, Actor: agentActor}); err != nil {
		t.Fatal(err)
	}
	git.Write("scenarios/bas/old.go", "weeks-old-change plus epoch change")
	git.Write("scenarios/bas/new.go", "epoch file")
	git.Write("scenarios/bas/stray.go", "concurrent session")
	git.Delete("scenarios/bas/clean.go")

	view := createProposal(t, svc, repo, "scenarios/bas/old.go", "scenarios/bas/new.go", "scenarios/bas/clean.go")
	if got := flagCodes(fileByPath(t, view.Files, "scenarios/bas/old.go")); !reflect.DeepEqual(got, []string{FlagMixedPrior}) {
		t.Fatalf("old.go flags = %v", got)
	}
	if got := flagCodes(fileByPath(t, view.Files, "scenarios/bas/new.go")); len(got) != 0 {
		t.Fatalf("new.go flags = %v", got)
	}
	deleted := fileByPath(t, view.Files, "scenarios/bas/clean.go")
	if !deleted.Deleted || deleted.Kind != KindDeleted || deleted.BlobID != "" {
		t.Fatalf("deletion = %#v", deleted)
	}
	if len(view.Excluded) != 1 || view.Excluded[0].Path != "scenarios/bas/stray.go" || view.Excluded[0].Reason != "not_listed" {
		t.Fatalf("excluded = %#v", view.Excluded)
	}
	if view.Evidence.AnchorID == "" || view.Freshness.State != FreshnessFresh {
		t.Fatalf("view = %#v", view)
	}
	rendered := view.Message.Rendered
	for _, want := range []string{"bas: unify timeline (E27)\n\nOutcome text.\n\n", "Vrooli-Effort: effort:bas@9", "Vrooli-Epoch: effort:bas#E27", "Vrooli-Run: " + testRun, "Vrooli-Proposal: " + view.ID} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered message missing %q:\n%s", want, rendered)
		}
	}
	if strings.Contains(strings.ToLower(rendered), "co-authored-by") {
		t.Fatalf("agent co-author rendered:\n%s", rendered)
	}
}

func TestCreateRefusesUnsafeIgnoredCleanAndUnchangedPaths(t *testing.T) {
	svc := testService(t)
	git := NewMemoryGit(map[string]string{"scenarios/bas/a.go": "a1", "scenarios/bas/b.go": "b1"})
	repo := repoFor(git)
	git.Write("scenarios/bas/a.go", "prior session")
	git.Write("scenarios/bas/build.log", "ignored")
	git.ignored["scenarios/bas/build.log"] = true
	if _, err := svc.Anchor(context.Background(), repo, AnchorRequest{EffortRef: "effort:bas", Epoch: "E27", Scopes: []string{"scenarios/bas"}, Actor: agentActor}); err != nil {
		t.Fatal(err)
	}
	git.Write("scenarios/bas/c.go", "epoch file")
	cases := map[string][]string{
		"parent traversal":       {"../etc/passwd"},
		"option injection":       {"--exec=evil"},
		"absolute":               {"/etc/passwd"},
		"git dir":                {".git/config"},
		"clean tracked file":     {"scenarios/bas/b.go"},
		"ignored file":           {"scenarios/bas/build.log"},
		"unchanged since anchor": {"scenarios/bas/a.go", "scenarios/bas/c.go"},
	}
	for name, paths := range cases {
		_, err := svc.Create(context.Background(), repo, CreateRequest{Work: epochWork(), Subject: "s", Paths: paths, Actor: agentActor})
		if !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
	if list, _, _ := svc.List(context.Background(), repo, ListFilter{}); len(list) != 0 {
		t.Fatalf("refused creates stored proposals: %#v", list)
	}
}

func TestCreateWithoutAnchorFlagsEveryFile(t *testing.T) {
	git := NewMemoryGit(map[string]string{"a.go": "a1"})
	git.Write("a.go", "a2")
	result, err := testService(t).Create(context.Background(), repoFor(git), CreateRequest{Subject: "fix a", Paths: []string{"a.go"}, Actor: agentActor})
	if err != nil {
		t.Fatal(err)
	}
	if got := flagCodes(result.View.Files[0]); !reflect.DeepEqual(got, []string{FlagNoAnchor}) {
		t.Fatalf("flags = %v", got)
	}
}

func TestAgentMessageErrorsBlockWhileHumanSeesWarnings(t *testing.T) {
	svc := testService(t)
	git := NewMemoryGit(map[string]string{"a.go": "a1"})
	git.Write("a.go", "a2")
	repo := repoFor(git)
	bad := CreateRequest{
		Work: Work{EffortRef: "bas-without-prefix", Epoch: "E1"}, Subject: "s", Paths: []string{"a.go"},
		Trailers: []trailers.Entry{{Key: "Co-Authored-By", Value: "Agent <a@x>"}}, Actor: agentActor,
	}
	_, err := svc.Create(context.Background(), repo, bad)
	var validation *ValidationError
	if !errors.As(err, &validation) || !trailers.HasErrors(validation.Issues) {
		t.Fatalf("agent create err = %v", err)
	}
	bad.Actor = humanActor
	result, err := svc.Create(context.Background(), repo, bad)
	if err != nil || !result.Stored || !trailers.HasErrors(result.Issues) {
		t.Fatalf("human create = %#v err=%v", result, err)
	}
}

func TestSharedPathsAndStagedPathsAreReadTimeFlags(t *testing.T) {
	svc := testService(t)
	git := NewMemoryGit(map[string]string{"shared.go": "s1", "only.go": "o1"})
	repo := repoFor(git)
	git.Write("shared.go", "s2")
	git.Write("only.go", "o2")
	first, err := svc.Create(context.Background(), repo, CreateRequest{Work: Work{EffortRef: "effort:bas", Epoch: "E1"}, Subject: "one", Paths: []string{"shared.go", "only.go"}, Actor: agentActor})
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.Create(context.Background(), repo, CreateRequest{Work: Work{EffortRef: "effort:bas", Epoch: "E2"}, Subject: "two", Paths: []string{"shared.go"}, Actor: agentActor})
	if err != nil {
		t.Fatal(err)
	}
	git.StageDirect("only.go")
	view, err := svc.Get(context.Background(), repo, first.View.ID)
	if err != nil {
		t.Fatal(err)
	}
	shared := fileByPath(t, view.Files, "shared.go")
	if len(shared.Flags) != 2 || shared.Flags[1].Code != FlagOtherOpenProposal || shared.Flags[1].Detail != second.View.ID {
		t.Fatalf("shared flags = %#v", shared.Flags)
	}
	if got := flagCodes(fileByPath(t, view.Files, "only.go")); !reflect.DeepEqual(got, []string{FlagNoAnchor, FlagAlreadyStaged}) {
		t.Fatalf("only.go flags = %v", got)
	}
}

// GCT-039: a regenerated proposal keeps the operator's edited message.
func TestSameEpochSupersedesAndCarriesOperatorEdits(t *testing.T) {
	svc := testService(t)
	git := NewMemoryGit(map[string]string{"a.go": "a1", "b.go": "b1"})
	repo := repoFor(git)
	git.Write("a.go", "a2")
	first := createProposal(t, svc, repo, "a.go")
	subject := "bas: operator's wording (E27)"
	if _, _, err := svc.Edit(context.Background(), repo, EditRequest{ID: first.ID, ExpectedRevision: 1, Subject: &subject, Actor: humanActor}); err != nil {
		t.Fatal(err)
	}
	git.Write("b.go", "b2")
	result, err := svc.Create(context.Background(), repo, CreateRequest{Work: epochWork(), Subject: "agent regenerated", Paths: []string{"a.go", "b.go"}, Actor: agentActor})
	if err != nil {
		t.Fatal(err)
	}
	if result.SupersededID != first.ID || result.View.Message.Subject != subject || !result.View.Message.OperatorEdited {
		t.Fatalf("result = %#v", result)
	}
	old, err := svc.Get(context.Background(), repo, first.ID)
	if err != nil || old.State != StateSuperseded || old.SupersededBy != result.View.ID {
		t.Fatalf("old = %#v err=%v", old.Proposal, err)
	}
}

func TestValidateOnlyStoresNothing(t *testing.T) {
	svc := testService(t)
	git := NewMemoryGit(map[string]string{"a.go": "a1"})
	git.Write("a.go", "a2")
	result, err := svc.Create(context.Background(), repoFor(git), CreateRequest{Work: epochWork(), Subject: "s", Paths: []string{"a.go"}, Actor: agentActor, ValidateOnly: true})
	if err != nil || result.Stored || result.View.Message.Rendered == "" {
		t.Fatalf("result = %#v err=%v", result, err)
	}
	if list, open, _ := svc.List(context.Background(), repoFor(git), ListFilter{}); len(list) != 0 || open != 0 {
		t.Fatalf("validate-only stored a proposal")
	}
}

func TestDigestBindsBaseContentAndMessage(t *testing.T) {
	files := []File{{Path: "b.go", BlobID: "bb"}, {Path: "a.go", Deleted: true}}
	base := Digest("head1", files, "msg")
	if Digest("head1", []File{files[1], files[0]}, "msg") != base {
		t.Fatal("digest depends on file order")
	}
	for _, other := range []string{Digest("head2", files, "msg"), Digest("head1", files, "msg2"), Digest("head1", []File{{Path: "b.go", BlobID: "bc"}, files[1]}, "msg")} {
		if other == base {
			t.Fatal("digest ignored a bound input")
		}
	}
}

// GCT-039 and GCT-053: edits create revisions, keep unknown trailers and
// survive a refresh; stale revisions are refused.
func TestEditAndRefreshKeepOperatorTextAndRevisionOrder(t *testing.T) {
	svc := testService(t)
	git := NewMemoryGit(map[string]string{"a.go": "a1", "b.go": "b1"})
	repo := repoFor(git)
	git.Write("a.go", "a2")
	git.Write("b.go", "b2")
	view := createProposal(t, svc, repo, "a.go", "b.go")
	body := "Operator body."
	edited, _, err := svc.Edit(context.Background(), repo, EditRequest{
		ID: view.ID, ExpectedRevision: 1, Body: &body, ReplaceTrailers: true,
		Trailers: append(view.Message.Trailers, trailers.Entry{Key: "Vrooli-Legacy-Thing", Value: "keep"}), Actor: humanActor,
	})
	if err != nil {
		t.Fatal(err)
	}
	if edited.Revision != 2 || !edited.Message.OperatorEdited || !strings.HasSuffix(edited.Message.Rendered, "\nVrooli-Legacy-Thing: keep") {
		t.Fatalf("edited = %#v", edited.Message)
	}
	if _, _, err := svc.Edit(context.Background(), repo, EditRequest{ID: view.ID, ExpectedRevision: 1, Body: &body, Actor: agentActor}); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale edit err = %v", err)
	}
	git.Write("a.go", "a3")
	git.CommitOutside(map[string]string{"b.go": "b2"})
	refreshed, dropped, err := svc.Refresh(context.Background(), repo, view.ID, 2, agentActor)
	if err != nil {
		t.Fatal(err)
	}
	if refreshed.Revision != 3 || refreshed.Message.Body != body || len(refreshed.Files) != 1 || refreshed.Files[0].BlobID != memoryBlobID("a3") || !reflect.DeepEqual(dropped, []string{"b.go"}) {
		t.Fatalf("refreshed = %#v dropped=%v", refreshed.Proposal, dropped)
	}
	if refreshed.BaseHead != git.head || refreshed.Freshness.State != FreshnessFresh {
		t.Fatalf("refreshed base/freshness = %s %#v", refreshed.BaseHead, refreshed.Freshness)
	}
	git.CommitOutside(map[string]string{"a.go": "a3"})
	superseded, _, err := svc.Refresh(context.Background(), repo, view.ID, 3, agentActor)
	if err != nil || superseded.State != StateSuperseded {
		t.Fatalf("all-clean refresh = %#v err=%v", superseded.Proposal, err)
	}
}

func TestFreshnessReportsDriftBaseMovedAndForeignStaged(t *testing.T) {
	svc := testService(t)
	git := NewMemoryGit(map[string]string{"a.go": "a1", "b.go": "b1", "c.go": "c1"})
	repo := repoFor(git)
	git.Write("a.go", "a2")
	git.Write("b.go", "b2")
	view := createProposal(t, svc, repo, "a.go", "b.go")
	git.Write("c.go", "c2")
	git.StageDirect("c.go")
	got, _ := svc.Get(context.Background(), repo, view.ID)
	if got.Freshness.State != FreshnessFresh || !reflect.DeepEqual(got.Freshness.ForeignStaged, []string{"c.go"}) {
		t.Fatalf("freshness = %#v", got.Freshness)
	}
	git.CommitOutside(map[string]string{"b.go": "b2"})
	got, _ = svc.Get(context.Background(), repo, view.ID)
	if got.Freshness.State != FreshnessBaseMoved || !reflect.DeepEqual(got.Freshness.BaseChanged, []string{"b.go"}) {
		t.Fatalf("freshness = %#v", got.Freshness)
	}
	git.Write("a.go", "a3")
	got, _ = svc.Get(context.Background(), repo, view.ID)
	if got.Freshness.State != FreshnessDrifted || !reflect.DeepEqual(got.Freshness.Drifted, []string{"a.go"}) {
		t.Fatalf("freshness = %#v", got.Freshness)
	}
}

func TestSubjectEvidenceChangesWithContentAndRevision(t *testing.T) {
	svc := testService(t)
	git := NewMemoryGit(map[string]string{"a.go": "a1"})
	repo := repoFor(git)
	git.Write("a.go", "a2")
	view := createProposal(t, svc, repo, "a.go")
	before, err := svc.SubjectEvidence(context.Background(), repo, SubjectContext(view.ID, 1))
	if err != nil {
		t.Fatal(err)
	}
	git.Write("a.go", "a3")
	after, _ := svc.SubjectEvidence(context.Background(), repo, SubjectContext(view.ID, 1))
	if string(before) == string(after) {
		t.Fatal("subject evidence ignored content drift")
	}
	if _, err := svc.SubjectEvidence(context.Background(), repo, SubjectContext(view.ID, 2)); !errors.Is(err, ErrConflict) {
		t.Fatalf("wrong revision err = %v", err)
	}
	if _, err := svc.SubjectEvidence(context.Background(), repo, "staged"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("bad context err = %v", err)
	}
}

func TestScopeMatcher(t *testing.T) {
	matcher, err := newScopeMatcher([]string{"scenarios/bas", "packages/*/schemas/**", "docs/*.md"})
	if err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string]bool{
		"scenarios/bas/api/x.go":                true,
		"scenarios/bas":                         true,
		"scenarios/basement/x.go":               false,
		"packages/proto/schemas/gct/v1/a.proto": true,
		"packages/proto/gen/x.go":               false,
		"docs/README.md":                        true,
		"docs/sub/README.md":                    false,
	} {
		if matcher.Match(path) != want {
			t.Errorf("match %s = %v", path, !want)
		}
	}
	if _, err := newScopeMatcher([]string{"../outside"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("unsafe scope err = %v", err)
	}
}
