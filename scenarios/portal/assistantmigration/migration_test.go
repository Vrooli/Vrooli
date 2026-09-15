package assistantmigration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildReviewReconcilesIssueContextAndStableOwnerRequest(t *testing.T) {
	// [REQ:MIG-01]
	root := t.TempDir()
	mustNoError(t, os.MkdirAll(filepath.Join(root, "data", "tasks"), 0o700))
	mustNoError(t, os.MkdirAll(filepath.Join(root, "data", "contexts"), 0o700))
	task := `# Issue: Button is clipped

**Scenario**: web-console
**URL**: http://localhost:3000/dashboard
**Captured**: 2025-10-03T06:27:33-04:00

## Description
The button is clipped after resizing.

## Context
- Issue ID: issue-1
- Screenshot: /tmp/capture.png
- Status: captured

## Resolution
_Pending agent assignment_
`
	ctx := "Issue ID: issue-1\nDescription: clipped button\nScreenshot: /tmp/capture.png\nScenario: web-console\n"
	mustNoError(t, os.WriteFile(filepath.Join(root, "data/tasks", "issue-1.md"), []byte(task), 0o600))
	mustNoError(t, os.WriteFile(filepath.Join(root, "data/contexts", "capture-1.txt"), []byte(ctx), 0o600))
	manifest, err := Inventory(root)
	mustNoError(t, err)
	review, err := BuildReview(root, manifest)
	mustNoError(t, err)
	if review.SourceRecords != 2 || review.TaskRecords != 1 || review.ContextRecords != 1 || len(review.Requests) != 1 {
		t.Fatalf("unexpected review counts: %+v", review)
	}
	if review.Requests[0].Owner != "web-console" || review.Requests[0].CaptureKey == "" || len(review.Requests[0].EvidenceLinks) != 3 {
		t.Fatalf("unexpected request: %+v", review.Requests[0])
	}
	first, err := BuildReview(root, manifest)
	mustNoError(t, err)
	if review.Requests[0].CaptureKey != first.Requests[0].CaptureKey || len(review.Requests[0].EvidenceLinks) != len(first.Requests[0].EvidenceLinks) {
		t.Fatal("review projection is not stable")
	}
}

func TestBuildReviewRoutesManualIssuesToScenarioQA(t *testing.T) {
	root := t.TempDir()
	mustNoError(t, os.MkdirAll(filepath.Join(root, "data", "tasks"), 0o700))
	task := `# Issue: Manual report

**Scenario**: manual
**URL**: cli
**Captured**: 2025-10-03T06:27:33-04:00

## Description
Something needs review.

## Context
- Issue ID: issue-2
- Status: captured
`
	mustNoError(t, os.WriteFile(filepath.Join(root, "data/tasks", "issue-2.md"), []byte(task), 0o600))
	manifest, err := Inventory(root)
	mustNoError(t, err)
	review, err := BuildReview(root, manifest)
	mustNoError(t, err)
	if len(review.Requests) != 1 || review.Requests[0].Owner != "scenario-qa" {
		t.Fatalf("unexpected manual owner: %+v", review.Requests)
	}
}

type captureRouter struct {
	requests []CaptureRequest
}

func (r *captureRouter) Capture(_ context.Context, request CaptureRequest) (CaptureReceipt, error) {
	r.requests = append(r.requests, request)
	return CaptureReceipt{CaptureKey: request.CaptureKey, Owner: request.Owner, TaskID: "task-" + request.LegacyID}, nil
}

func TestRouteDeduplicatesCaptureKeysAndChecksReceipts(t *testing.T) {
	// [REQ:MIG-02]
	review := ReviewManifest{Requests: []CaptureRequest{
		{LegacyID: "issue-1", CaptureKey: "key-1", Owner: "web-console"},
		{LegacyID: "issue-1", CaptureKey: "key-1", Owner: "web-console"},
	}}
	router := &captureRouter{}
	receipts, err := Route(context.Background(), review, router)
	mustNoError(t, err)
	if len(receipts) != 1 || len(router.requests) != 1 || receipts[0].TaskID != "task-issue-1" {
		t.Fatalf("unexpected receipts: %+v requests=%d", receipts, len(router.requests))
	}
	review.Requests[1].Issue.Description = "different payload"
	if _, err := Route(context.Background(), review, &captureRouter{}); err != ErrConflict {
		t.Fatalf("conflicting duplicate key error = %v, want %v", err, ErrConflict)
	}
}

func TestFileRouterIsIdempotentAndRejectsConflicts(t *testing.T) {
	router, err := NewFileRouter(t.TempDir())
	mustNoError(t, err)
	request := CaptureRequest{LegacyID: "issue-1", CaptureKey: "key-1", Owner: "web-console", Issue: Issue{ID: "issue-1", Title: "Title", Description: "Description"}}
	first, err := router.Capture(context.Background(), request)
	mustNoError(t, err)
	second, err := router.Capture(context.Background(), request)
	mustNoError(t, err)
	if first != second {
		t.Fatalf("idempotent retry changed receipt: first=%+v second=%+v", first, second)
	}
	request.Issue.Description = "changed"
	if _, err := router.Capture(context.Background(), request); err != ErrConflict {
		t.Fatalf("conflicting retry error = %v, want %v", err, ErrConflict)
	}
	if _, err := router.Capture(context.Background(), CaptureRequest{LegacyID: "../escape", CaptureKey: "key-2", Owner: "web-console"}); err == nil {
		t.Fatal("unsafe legacy identity was accepted")
	}
}

func TestFileRouterCopiesVerifiedContextEvidence(t *testing.T) {
	// [REQ:MIG-02]
	source := t.TempDir()
	destination := t.TempDir()
	contextPath := filepath.Join(source, "data", "contexts", "capture.txt")
	mustNoError(t, os.MkdirAll(filepath.Dir(contextPath), 0o700))
	data := []byte("Issue ID: issue-1\ncontext\n")
	mustNoError(t, os.WriteFile(contextPath, data, 0o600))
	digest := sha256.Sum256(data)
	router, err := NewFileRouter(destination, source)
	mustNoError(t, err)
	request := CaptureRequest{LegacyID: "issue-1", CaptureKey: "key-context", Owner: "scenario-qa", Issue: Issue{ID: "issue-1", Title: "Title", Description: "Description"}, EvidenceLinks: []EvidenceLink{{IssueID: "issue-1", Kind: "context", RelativePath: "data/contexts/capture.txt", SHA256: hex.EncodeToString(digest[:])}}}
	_, err = router.Capture(context.Background(), request)
	mustNoError(t, err)
	copied, err := os.ReadFile(filepath.Join(destination, "scenario-qa", "evidence", "issue-1", "capture.txt"))
	mustNoError(t, err)
	if string(copied) != string(data) {
		t.Fatalf("copied context = %q, want %q", copied, data)
	}
}

func TestParseIssueRejectsIdentityMismatch(t *testing.T) {
	root := t.TempDir()
	mustNoError(t, os.MkdirAll(filepath.Join(root, "data", "tasks"), 0o700))
	data := []byte("# Issue: Example\n\n**Scenario**: manual\n\n## Description\nA description.\n\n## Context\n- Issue ID: other\n")
	path := filepath.Join(root, "data/tasks", "issue-1.md")
	mustNoError(t, os.WriteFile(path, data, 0o600))
	manifest, err := Inventory(root)
	mustNoError(t, err)
	_, err = ParseIssue(root, manifest.Records[0])
	if err == nil || !strings.Contains(err.Error(), "identity mismatch") {
		t.Fatalf("expected identity mismatch, got %v", err)
	}
}

func mustNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
