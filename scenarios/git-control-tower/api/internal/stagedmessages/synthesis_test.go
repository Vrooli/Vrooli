package stagedmessages

import (
	"context"
	"encoding/json"
	"fmt"
	"git-control-tower/internal/reviewjobs"
	"testing"

	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/advisory"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type synthesisSnapshotFixture struct {
	*evidenceSnapshotFixture
	blobReads int
}

func (f *synthesisSnapshotFixture) EvidenceFile(ctx context.Context, dir string, file *v1.StagedMessageFile) (*v1.StagedMessageEvidenceFile, error) {
	f.blobReads++
	return f.evidenceSnapshotFixture.EvidenceFile(ctx, dir, file)
}

type synthesisRunnerFixture struct {
	*evidenceRunnerFixture
	pages, inventoryFiles, maxPage int
	incompleteInventory            bool
	noProgress                     bool
}

func (p *synthesisRunnerFixture) Start(ctx context.Context, input DraftInput, _ string) (Execution, error) {
	p.starts++
	p.input = input
	if input.FormatVersion != 4 {
		return Execution{}, fmt.Errorf("new investigation still uses inspection-only format %d", input.FormatVersion)
	}
	cursor := uint32(0)
	for {
		page, err := p.s.ReadEvidence(ctx, "repo", "fixture", &v1.ReadStagedMessageEvidenceRequest{JobId: input.JobID, SnapshotDigest: input.Snapshot.Digest, Cursor: cursor, MaxBytes: 16384, InventoryOnly: true}, "deep-execution")
		if err != nil {
			return Execution{}, err
		}
		p.pages++
		p.inventoryFiles += len(page.Files)
		raw, _ := protojson.Marshal(page)
		p.maxPage = max(p.maxPage, len(raw))
		for _, file := range page.Files {
			if file.Patch != "" || file.OldBlobId != "" || file.NewBlobId != "" || file.CompletePatch {
				return Execution{}, fmt.Errorf("inventory leaked patch/blob evidence")
			}
		}
		if page.Complete || p.incompleteInventory {
			break
		}
		if page.NextCursor <= cursor {
			return Execution{}, fmt.Errorf("inventory did not advance")
		}
		cursor = page.NextCursor
	}
	if p.noProgress {
		result := proto.Clone(input.PriorResult).(*v1.StagedMessageSuggestion)
		result.MessageScope = v1.StagedMessageScope_STAGED_MESSAGE_SCOPE_LIMITED
		result.MessageScopeExplanation = "Inventory obtained, but important immutable patch reads could not be completed; previous wording retained."
		return Execution{ID: "deep-execution", Status: "succeeded", Result: result}, nil
	}
	ids := []uint32{uint32(len(input.Snapshot.Files) - 1), uint32(len(input.Snapshot.Files))}
	page, err := p.s.ReadEvidence(ctx, "repo", "fixture", &v1.ReadStagedMessageEvidenceRequest{JobId: input.JobID, SnapshotDigest: input.Snapshot.Digest, MaxBytes: 16384, FileIds: ids}, "deep-execution")
	if err != nil {
		return Execution{}, err
	}
	if len(page.Files) != 2 || !page.Complete {
		return Execution{}, fmt.Errorf("targeted evidence missing")
	}
	covered := priorCovered(input)
	for _, file := range page.Files {
		if file.CompletePatch {
			covered[file.Path] = true
		}
	}
	coverage := &v1.StagedMessageCoverage{TotalEdits: int32(len(input.Snapshot.Files)), Partial: true}
	for _, file := range input.Snapshot.Files {
		if covered[file.Path] {
			coverage.CoveredPaths = append(coverage.CoveredPaths, file.Path)
			coverage.MatchedEdits++
		} else {
			coverage.UncoveredPaths = append(coverage.UncoveredPaths, file.Path)
		}
	}
	return Execution{ID: "deep-execution", Status: "succeeded", Result: &v1.StagedMessageSuggestion{Title: "Summarize all staged themes using selective evidence", Coverage: coverage, MessageScope: v1.StagedMessageScope_STAGED_MESSAGE_SCOPE_WHOLE_SELECTION, MessageScopeExplanation: "Reviewed the complete immutable inventory and representative changes; other patches were not inspected."}}, nil
}

func TestWholeSelectionSynthesisUsesCompleteInventoryAndTwoTargetedPatches(t *testing.T) {
	for _, count := range []int{285, 4096} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			s, _, snapshot, base, prior := continuationFixture(t, count)
			spy := &synthesisSnapshotFixture{evidenceSnapshotFixture: snapshot}
			s.snapshots = spy
			runner := &synthesisRunnerFixture{evidenceRunnerFixture: base}
			s.provider = runner
			job, err := s.Start(t.Context(), "repo", "fixture", &v1.StartStagedMessageRequest{ContinuationOfJobId: prior.ID, SnapshotDigest: snapshot.identity, RequestKey: "selective-synthesis"})
			if err != nil || job.Status != "succeeded" {
				t.Fatalf("whole synthesis failed: status=%s error=%s %v", job.GetStatus(), job.GetError(), err)
			}
			if runner.inventoryFiles != count || spy.blobReads != 2 || len(job.Result.Coverage.CoveredPaths) != 28 || !job.Result.Coverage.Partial || job.Result.MessageScope != v1.StagedMessageScope_STAGED_MESSAGE_SCOPE_WHOLE_SELECTION {
				t.Fatalf("scope/inspection conflated: inventory=%d blobReads=%d result=%v", runner.inventoryFiles, spy.blobReads, job.Result)
			}
			if runner.maxPage > 16384 || runner.pages > (count+63)/64 {
				t.Fatalf("unbounded inventory: pages=%d bytes=%d", runner.pages, runner.maxPage)
			}
			t.Logf("selection=%d inventoryPages=%d maximumBytes=%d patchReads=%d inspected=%d", count, runner.pages, runner.maxPage, spy.blobReads, len(job.Result.Coverage.CoveredPaths))
		})
	}
}
func TestWholeSelectionSynthesisRejectsMissingInventory(t *testing.T) {
	s, _, snapshot, base, prior := continuationFixture(t, 285)
	s.provider = &synthesisRunnerFixture{evidenceRunnerFixture: base, incompleteInventory: true}
	job, err := s.Start(t.Context(), "repo", "fixture", &v1.StartStagedMessageRequest{ContinuationOfJobId: prior.ID, SnapshotDigest: snapshot.identity, RequestKey: "unsubstantiated-whole"})
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != "failed" {
		t.Fatalf("whole scope accepted without complete inventory: status=%s error=%s", job.GetStatus(), job.GetError())
	}
}

func TestTargetedSynthesisEvidenceCanRereadPriorCoverageAndRejectInvalidSelection(t *testing.T) {
	s, _, snapshot, base, prior := continuationFixture(t, 285)
	base.execute = false
	spy := &synthesisSnapshotFixture{evidenceSnapshotFixture: snapshot}
	s.snapshots = spy
	job, err := s.Start(t.Context(), "repo", "fixture", &v1.StartStagedMessageRequest{ContinuationOfJobId: prior.ID, SnapshotDigest: snapshot.identity, RequestKey: "target"})
	if err != nil {
		t.Fatal(err)
	}
	page, err := s.ReadEvidence(t.Context(), "repo", "fixture", &v1.ReadStagedMessageEvidenceRequest{JobId: job.Id, SnapshotDigest: snapshot.identity, FileIds: []uint32{1, 285}, MaxBytes: 16384}, "deep-execution")
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Files) != 2 || page.Files[0].FileId != 1 || spy.blobReads != 2 {
		t.Fatal("targeted reread skipped important inherited coverage")
	}
	for _, req := range []*v1.ReadStagedMessageEvidenceRequest{{FileIds: []uint32{0}}, {FileIds: []uint32{286}}, {FileIds: []uint32{1, 1}}, {FileIds: []uint32{1}, InventoryOnly: true}, {FileIds: []uint32{1}, Cursor: 1}} {
		req.JobId = job.Id
		req.SnapshotDigest = snapshot.identity
		if _, err := s.ReadEvidence(t.Context(), "repo", "fixture", req, "deep-execution"); err == nil {
			t.Fatal("invalid target accepted")
		}
	}
	if spy.blobReads != 2 {
		t.Fatal("invalid selection read blobs")
	}
}

func TestTargetedSynthesisPageRetriesOnlyUndeliveredIDs(t *testing.T) {
	s, _, snapshot, runner, prior := continuationFixture(t, 285)
	runner.execute = false
	job, err := s.Start(t.Context(), "repo", "fixture", &v1.StartStagedMessageRequest{ContinuationOfJobId: prior.ID, SnapshotDigest: snapshot.identity, RequestKey: "target-page"})
	if err != nil {
		t.Fatal(err)
	}
	ids := []uint32{280, 281, 282, 283, 284, 285}
	returned := map[uint32]bool{}
	for len(ids) > 0 {
		page, err := s.ReadEvidence(t.Context(), "repo", "fixture", &v1.ReadStagedMessageEvidenceRequest{JobId: job.Id, SnapshotDigest: snapshot.identity, MaxBytes: 1024, FileIds: ids}, "deep-execution")
		if err != nil {
			t.Fatal(err)
		}
		if len(page.Files) == 0 || page.NextCursor != 0 || page.Complete != (len(page.Files) == len(ids)) {
			t.Fatal("targeted completeness/cursor mismatch")
		}
		raw, _ := protojson.Marshal(page)
		if len(raw) > 1024 {
			t.Fatal("target page bound exceeded")
		}
		for _, file := range page.Files {
			if returned[file.FileId] {
				t.Fatal("duplicate target across retries")
			}
			returned[file.FileId] = true
		}
		remaining := []uint32{}
		for _, id := range ids {
			if !returned[id] {
				remaining = append(remaining, id)
			}
		}
		ids = remaining
	}
	if len(returned) != 6 {
		t.Fatal("targeted partial page lost IDs")
	}
}

func TestSynthesisNoProgressKeepsPriorDraftWithExplicitLimitedExplanation(t *testing.T) {
	s, store, snapshot, base, prior := continuationFixture(t, 285)
	s.provider = &synthesisRunnerFixture{evidenceRunnerFixture: base, noProgress: true}
	job, err := s.Start(t.Context(), "repo", "fixture", &v1.StartStagedMessageRequest{ContinuationOfJobId: prior.ID, SnapshotDigest: snapshot.identity, RequestKey: "no-progress"})
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != "succeeded" || job.Result.MessageScope != v1.StagedMessageScope_STAGED_MESSAGE_SCOPE_LIMITED || job.Result.MessageScopeExplanation == "" || job.Result.Title != "Previous partial draft" || job.EvidenceFilesRead != 0 {
		t.Fatalf("no-progress explanation missing: status=%s error=%s", job.Status, job.Error)
	}
	old, err := store.Get(t.Context(), prior.ID)
	if err != nil || old.ResultJSON != prior.ResultJSON {
		t.Fatal("previous draft overwritten")
	}
	var input DraftInput
	input.FormatVersion = 4
	input.Snapshot = snapshot.snapshot
	input.PriorResult = job.Result
	job.Result.MessageScopeExplanation = ""
	if validateResult(input, job.Result) == nil {
		t.Fatal("unexplained unchanged synthesis accepted")
	}
}
func TestSynthesisRetryPreservesAlreadyAdmittedFormatThree(t *testing.T) {
	s, store, snapshot, runner, prior := continuationFixture(t, 285)
	runner.execute = false
	var input DraftInput
	if err := json.Unmarshal([]byte(prior.InputJSON), &input); err != nil {
		t.Fatal(err)
	}
	input.FormatVersion = 3
	input.JobID = "old-deep-job"
	input.ContinuationOfJobID = prior.ID
	input.PriorResult = &v1.StagedMessageSuggestion{}
	if err := protojson.Unmarshal([]byte(prior.ResultJSON), input.PriorResult); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(input)
	_, _, err := store.Create(t.Context(), reviewjobs.Job{ID: input.JobID, IdempotencyKey: key("repo", "old-deep-key"), Operation: operation, RepositoryID: "repo", InputJSON: string(raw), InputDigest: digest(raw), State: reviewjobs.Running})
	if err != nil {
		t.Fatal(err)
	}
	job, err := s.Start(t.Context(), "repo", "fixture", &v1.StartStagedMessageRequest{ContinuationOfJobId: prior.ID, SnapshotDigest: snapshot.identity, RequestKey: "old-deep-key"})
	if err != nil {
		t.Fatal(err)
	}
	if job.Id != input.JobID || runner.input.FormatVersion != 3 || runner.starts != 1 {
		t.Fatal("old admitted retry silently repackaged")
	}
}
