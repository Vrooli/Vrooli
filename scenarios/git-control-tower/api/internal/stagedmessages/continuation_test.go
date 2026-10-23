package stagedmessages

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"git-control-tower/internal/reviewjobs"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/advisory"
	"google.golang.org/protobuf/encoding/protojson"
)

type evidenceSnapshotFixture struct {
	snapshot   *v1.StagedMessageSnapshot
	identity   string
	largePatch bool
}

func (f *evidenceSnapshotFixture) Capture(context.Context, string, string) (Captured, error) {
	return Captured{Snapshot: f.snapshot}, nil
}

func (f *evidenceSnapshotFixture) Identity(context.Context, string, string) (string, error) {
	return f.identity, nil
}

func (f *evidenceSnapshotFixture) EvidenceFile(_ context.Context, _ string, file *v1.StagedMessageFile) (*v1.StagedMessageEvidenceFile, error) {
	patch := "--- before/" + file.Path + "\n+++ staged/" + file.Path + "\n-old immutable content\n+new immutable behavior\n"
	if f.largePatch {
		patch = strings.Repeat("large staged content\n", 5000)
	}
	return &v1.StagedMessageEvidenceFile{Path: file.Path, Patch: patch, CompletePatch: true, OldBlobId: file.OldBlobId, NewBlobId: file.NewBlobId}, nil
}

type evidenceRunnerFixture struct {
	fakeProvider
	s                      *Service
	execute                bool
	starts, authorizations int
	input                  DraftInput
}

func (p *evidenceRunnerFixture) Cancel(_ context.Context, id string, _ DraftInput) (Execution, error) {
	return Execution{ID: id, Status: "cancelled"}, nil
}

func (p *evidenceRunnerFixture) AuthorizeEvidence(_ context.Context, in DraftInput, _ string, execution string) error {
	p.authorizations++
	if execution != "deep-execution" || in.JobID == "" {
		return connect.NewError(connect.CodePermissionDenied, fmt.Errorf("foreign execution"))
	}
	return nil
}

func (p *evidenceRunnerFixture) Start(ctx context.Context, input DraftInput, _ string) (Execution, error) {
	p.starts++
	p.input = input
	if !p.execute {
		return Execution{ID: "deep-execution", Status: "running"}, nil
	}
	if input.FormatVersion == 4 {
		cursor := uint32(0)
		for {
			page, err := p.s.ReadEvidence(ctx, "repo", "fixture", &v1.ReadStagedMessageEvidenceRequest{JobId: input.JobID, SnapshotDigest: input.Snapshot.Digest, Cursor: cursor, InventoryOnly: true, MaxBytes: 49152}, "deep-execution")
			if err != nil {
				return Execution{}, err
			}
			if page.Complete {
				break
			}
			cursor = page.NextCursor
		}
	}
	cursor := uint32(0)
	for {
		page, err := p.s.ReadEvidence(ctx, "repo", "fixture", &v1.ReadStagedMessageEvidenceRequest{JobId: input.JobID, SnapshotDigest: input.Snapshot.Digest, Cursor: cursor, MaxBytes: 49152}, "deep-execution")
		if err != nil {
			return Execution{}, err
		}
		raw, _ := protojson.Marshal(page)
		if len(raw) > 49152 {
			return Execution{}, fmt.Errorf("page byte budget exceeded")
		}
		if page.Complete {
			break
		}
		if page.NextCursor <= cursor {
			return Execution{}, fmt.Errorf("page did not advance")
		}
		cursor = page.NextCursor
	}
	coverage := &v1.StagedMessageCoverage{TotalEdits: int32(len(input.Snapshot.Files)), MatchedEdits: int32(len(input.Snapshot.Files))}
	for _, file := range input.Snapshot.Files {
		coverage.CoveredPaths = append(coverage.CoveredPaths, file.Path)
	}
	result := &v1.StagedMessageSuggestion{Title: "Update behavior across the whole staged selection", Coverage: coverage}
	if input.FormatVersion == 4 {
		result.MessageScope = v1.StagedMessageScope_STAGED_MESSAGE_SCOPE_WHOLE_SELECTION
		result.MessageScopeExplanation = "Complete immutable inventory and all requested patches were supplied."
	}
	return Execution{ID: "deep-execution", Status: "succeeded", Result: result}, nil
}

func continuationFixture(t *testing.T, count int) (*Service, *reviewjobs.Store, *evidenceSnapshotFixture, *evidenceRunnerFixture, reviewjobs.Job) {
	t.Helper()
	s, store, _, _ := serviceFixture(t)
	snapshot := &v1.StagedMessageSnapshot{RepositoryId: "repo", Digest: "immutable-complete-subject"}
	coverage := &v1.StagedMessageCoverage{Partial: true, TotalEdits: int32(count), MatchedEdits: 26}
	for i := 0; i < count; i++ {
		path := fmt.Sprintf("scenarios/scope-%02d/file-%04d.go", i%12, i)
		file := &v1.StagedMessageFile{Path: path, EditCount: 1, OldBlobId: fmt.Sprintf("%040x", i+1), NewBlobId: fmt.Sprintf("%040x", i+count+1)}
		if i >= 64 {
			file.OmissionReason = "omitted by quick sample"
		}
		snapshot.Files = append(snapshot.Files, file)
		if i < 26 {
			coverage.CoveredPaths = append(coverage.CoveredPaths, path)
		} else {
			coverage.UncoveredPaths = append(coverage.UncoveredPaths, path)
		}
	}
	input := DraftInput{FormatVersion: 2, Snapshot: snapshot, OperatorNote: "Preserve this original intent", Contexts: []*v1.StagedMessageContext{{Id: "source-1", Revision: "revision-1"}}}
	raw, _ := json.Marshal(input)
	result, _ := protojson.Marshal(&v1.StagedMessageSuggestion{Title: "Previous partial draft", Coverage: coverage, SourceIds: []string{"source-1"}})
	previous, _, err := store.Create(t.Context(), reviewjobs.Job{ID: "previous-draft", IdempotencyKey: "previous", InputDigest: digest(raw), InputJSON: string(raw), Operation: operation, RepositoryID: "repo", ResultRef: "original-collection", State: reviewjobs.Succeeded, ResultJSON: string(result)})
	if err != nil {
		t.Fatal(err)
	}
	fixture := &evidenceSnapshotFixture{snapshot: snapshot, identity: snapshot.Digest}
	p := &evidenceRunnerFixture{execute: true}
	s.snapshots = fixture
	s.provider = p
	p.s = s
	return s, store, fixture, p, previous
}

func TestDeepContinuationExaminesBeyondQuickSampleAndRetainsOriginalIntent(t *testing.T) {
	for _, count := range []int{285, 4096} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			s, store, _, runner, prior := continuationFixture(t, count)
			req := &v1.StartStagedMessageRequest{ContinuationOfJobId: prior.ID, SnapshotDigest: "immutable-complete-subject", RequestKey: "explicit-deep"}
			job, err := s.Start(t.Context(), "repo", "fixture", req)
			if err != nil || job.Status != "succeeded" {
				t.Fatalf("deep result: %v %v", job, err)
			}
			if len(job.Result.Coverage.CoveredPaths) != count || job.Result.Coverage.Partial || job.EvidenceFilesRead != uint32(count-26) || runner.authorizations == 0 || runner.input.OperatorNote != "Preserve this original intent" || len(runner.input.Contexts) != 1 {
				t.Fatalf("deep evidence/intent lost: covered=%d read=%d authorizations=%d", len(job.Result.Coverage.CoveredPaths), job.EvidenceFilesRead, runner.authorizations)
			}
			old, err := store.Get(t.Context(), prior.ID)
			if err != nil || old.ResultJSON != prior.ResultJSON || old.State != reviewjobs.Succeeded {
				t.Fatal("previous draft changed", err)
			}
			again, err := s.Start(t.Context(), "repo", "fixture", req)
			if err != nil || again.Id != job.Id || runner.starts != 1 {
				t.Fatal("same-key continuation launched again", err)
			}
			t.Logf("files=%d fully supplied remaining=%d owner starts=%d", count, job.EvidenceFilesRead, runner.starts)
		})
	}
}

func TestDeepContinuationRejectsDriftChangedIntentAndUnissuedCoverage(t *testing.T) {
	s, _, snapshot, runner, prior := continuationFixture(t, 285)
	req := &v1.StartStagedMessageRequest{ContinuationOfJobId: prior.ID, SnapshotDigest: snapshot.identity, RequestKey: "deep"}
	snapshot.identity = "drifted"
	if _, err := s.Start(t.Context(), "repo", "fixture", req); err == nil || runner.starts != 0 {
		t.Fatal("drift started work")
	}
	snapshot.identity = req.SnapshotDigest
	req.OperatorNote = "different motivation"
	if _, err := s.Start(t.Context(), "repo", "fixture", req); err == nil {
		t.Fatal("continuation replaced original intent")
	}
	req.OperatorNote = ""
	runner.execute = false
	job, err := s.Start(t.Context(), "repo", "fixture", req)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.ReadEvidence(t.Context(), "repo", "fixture", &v1.ReadStagedMessageEvidenceRequest{JobId: job.Id, SnapshotDigest: req.SnapshotDigest}, "foreign-execution")
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatal("foreign workflow read evidence", err)
	}
	input := runner.input
	result := &v1.StagedMessageSuggestion{Title: "Invented full result", Coverage: &v1.StagedMessageCoverage{TotalEdits: 285, MatchedEdits: 285}}
	for _, file := range input.Snapshot.Files {
		result.Coverage.CoveredPaths = append(result.Coverage.CoveredPaths, file.Path)
	}
	if err := validateResult(input, result); err == nil {
		t.Fatal("never-supplied patches counted as covered")
	}
	// Cancellation changes lifecycle while the old draft remains independently
	// stored; receipt writes never revive it or overwrite lifecycle revisions.
	cancelled, err := s.Job(t.Context(), "repo", "fixture", job.Id, "", "cancel", time.Second)
	if err != nil || cancelled.Status != "cancelled" || cancelled.Result != nil {
		t.Fatal("cancellation did not preserve terminal result boundary", err)
	}
}

func TestImmutableEvidenceReadsPinnedBlobsAfterIndexAndWorktreeDrift(t *testing.T) {
	dir := t.TempDir()
	git := func(input string, args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"--no-optional-locks", "-C", dir}, args...)...)
		cmd.Stdin = strings.NewReader(input)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("fixture git %v: %v %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("", "init", "-q")
	before := git("old immutable content\n", "hash-object", "-w", "--stdin")
	staged := git("staged immutable behavior\n", "hash-object", "-w", "--stdin")
	later := git("later changed index\n", "hash-object", "-w", "--stdin")
	git("", "update-index", "--add", "--cacheinfo", "100644", later, "file.txt")
	index := git("", "ls-files", "--stage")
	evidence, err := (GitSnapshotter{}).EvidenceFile(t.Context(), dir, &v1.StagedMessageFile{Path: "file.txt", OldBlobId: before, NewBlobId: staged, OldMode: "100644", NewMode: "100644"})
	if err != nil || !evidence.CompletePatch || !strings.Contains(evidence.Patch, "staged immutable behavior") || strings.Contains(evidence.Patch, "later changed index") || git("", "ls-files", "--stage") != index {
		t.Fatalf("immutable blob read substituted or changed index: %v %v", evidence, err)
	}
	modeOnly, err := (GitSnapshotter{}).EvidenceFile(t.Context(), dir, &v1.StagedMessageFile{Path: "renamed.txt", OldPath: "file.txt", OldBlobId: staged, NewBlobId: staged, OldMode: "100644", NewMode: "100755"})
	if err != nil || !modeOnly.CompletePatch || modeOnly.OldPath != "file.txt" || !strings.Contains(modeOnly.Patch, "new mode 100755") {
		t.Fatal("mode/rename metadata omitted", modeOnly, err)
	}
	for _, content := range []string{"binary\x00content", strings.Repeat("x", maxFileBytes+1)} {
		blob := git(content, "hash-object", "-w", "--stdin")
		limited, err := (GitSnapshotter{}).EvidenceFile(t.Context(), dir, &v1.StagedMessageFile{Path: "limited", NewBlobId: blob, NewMode: "100644"})
		if err != nil || limited.CompletePatch || limited.Patch != "" || limited.OmissionReason == "" {
			t.Fatal("unreadable whole file claimed as evidence", limited, err)
		}
	}
}

func TestEvidencePageOmissionDoesNotIssueWholePatchCoverage(t *testing.T) {
	s, _, snapshot, runner, prior := continuationFixture(t, 285)
	runner.execute = false
	snapshot.largePatch = true
	job, err := s.Start(t.Context(), "repo", "fixture", &v1.StartStagedMessageRequest{ContinuationOfJobId: prior.ID, SnapshotDigest: snapshot.identity, RequestKey: "large-page"})
	if err != nil {
		t.Fatal(err)
	}
	page, err := s.ReadEvidence(t.Context(), "repo", "fixture", &v1.ReadStagedMessageEvidenceRequest{JobId: job.Id, SnapshotDigest: snapshot.identity, MaxBytes: 1024}, "deep-execution")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := protojson.Marshal(page)
	if len(raw) > 1024 || len(page.Files) != 1 || page.Files[0].CompletePatch || page.Files[0].OmissionReason == "" || page.Files[0].Patch != "" || page.NextCursor <= 26 || page.Complete {
		t.Fatalf("oversized whole patch was silently clipped/credited: %v size=%d", page, len(raw))
	}
	ids, err := s.EvidenceEligibility(t.Context(), job.Id, "repo", snapshot.identity, "deep-execution")
	if err != nil || len(ids) != 0 {
		t.Fatal("omission produced coverage eligibility", ids, err)
	}
}
