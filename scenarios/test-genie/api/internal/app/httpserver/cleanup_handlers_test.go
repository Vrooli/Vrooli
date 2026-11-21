package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	appruns "test-genie/internal/app/runs"
	sharedruns "test-genie/internal/shared/runs"

	"github.com/vrooli/vrooli/packages/artifactledger"
	"github.com/vrooli/vrooli/packages/artifactpaths"
)

func TestOwnerCleanupRequiresIdempotencyAndRecordsDeletion(t *testing.T) {
	storageRoot := filepath.Join(t.TempDir(), "storage")
	t.Setenv("VROOLI_STORAGE_ROOT", storageRoot)
	scenariosRoot := filepath.Join(t.TempDir(), "scenarios")
	writeCleanupScenarioManifest(t, scenariosRoot, "demo")

	artifactRoot, err := artifactpaths.ScenarioRoot("demo")
	if err != nil {
		t.Fatal(err)
	}
	runID := "old-run"
	now := time.Now().UTC()
	index := sharedruns.NewIndex(artifactRoot)
	runs := []struct {
		id  string
		age time.Duration
	}{
		{id: "recent-run", age: 29 * 24 * time.Hour},
		{id: "kept-1", age: 31 * 24 * time.Hour},
		{id: "kept-2", age: 32 * 24 * time.Hour},
		{id: "kept-3", age: 33 * 24 * time.Hour},
		{id: "kept-4", age: 34 * 24 * time.Hour},
		{id: "kept-5", age: 35 * 24 * time.Hour},
		{id: "kept-6", age: 36 * 24 * time.Hour},
		{id: "kept-7", age: 37 * 24 * time.Hour},
		{id: "kept-8", age: 38 * 24 * time.Hour},
		{id: "kept-9", age: 39 * 24 * time.Hour},
		{id: runID, age: 40 * 24 * time.Hour},
	}
	for _, run := range runs {
		startedAt := now.Add(-run.age)
		if err := index.Append(sharedruns.RunRecord{
			RunID: run.id, Scenario: "demo", StartedAt: startedAt,
			CompletedAt: startedAt.Add(time.Hour), Status: sharedruns.StatusPassed,
		}); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(artifactpaths.RunDir(artifactRoot, run.id), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	runPath := artifactpaths.RunDir(artifactRoot, runID)
	if err := os.MkdirAll(runPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runPath, "evidence.bin"), bytes.Repeat([]byte("x"), 32), 0o644); err != nil {
		t.Fatal(err)
	}

	ledger := artifactledger.NewAt(filepath.Join(t.TempDir(), "receipts"))
	server := &Server{
		repoRoot:       filepath.Dir(scenariosRoot),
		runsService:    appruns.NewService(scenariosRoot, nil, nil, nil),
		logger:         log.New(io.Discard, "", 0),
		cleanupResults: make(map[string]ownerCleanupApplyResponse),
		removalLedger:  ledger,
	}

	estimateRecorder := httptest.NewRecorder()
	weakMaxBytes := strconv.FormatInt(defaultCleanupMaxBytes*2, 10)
	server.handleCleanupEstimate(estimateRecorder, httptest.NewRequest(http.MethodGet, "/api/v1/cleanup/estimate?min_age_seconds=3600&keep_count=0&max_bytes="+weakMaxBytes, nil))
	if estimateRecorder.Code != http.StatusOK {
		t.Fatalf("estimate status = %d body=%s", estimateRecorder.Code, estimateRecorder.Body.String())
	}
	var estimate ownerCleanupEstimate
	if err := json.NewDecoder(estimateRecorder.Body).Decode(&estimate); err != nil {
		t.Fatal(err)
	}
	if estimate.ItemCount != 1 || estimate.EstimatedBytes != 32 {
		t.Fatalf("bounded estimate = %#v, want only the 32-byte old run", estimate)
	}
	if estimate.MinAgeSeconds != defaultCleanupMinAgeSeconds || estimate.KeepCount != defaultCleanupKeepCount || estimate.MaxBytes != defaultCleanupMaxBytes {
		t.Fatalf("estimate policy = %#v, want declared 30d/10/20GiB bounds", estimate)
	}

	// Treat estimate terms as caller-controlled input at the preview boundary.
	// Zero means omitted/weak, not permission to disable the owner bounds.
	estimate.MinAgeSeconds = 0
	estimate.KeepCount = 0
	estimate.MaxBytes = 0
	previewRecorder := httptest.NewRecorder()
	server.handleCleanupPreview(previewRecorder, cleanupJSONRequest(t, "/api/v1/cleanup/preview", ownerCleanupPreviewRequest{Estimate: estimate}))
	if previewRecorder.Code != http.StatusOK {
		t.Fatalf("preview status = %d body=%s", previewRecorder.Code, previewRecorder.Body.String())
	}
	var preview ownerCleanupPreview
	if err := json.NewDecoder(previewRecorder.Body).Decode(&preview); err != nil {
		t.Fatal(err)
	}
	if len(preview.Items) != 1 || preview.Items[0].ID != "demo/old-run" {
		t.Fatalf("preview = %#v", preview)
	}
	if preview.MinAgeSeconds != defaultCleanupMinAgeSeconds || preview.KeepCount != defaultCleanupKeepCount || preview.MaxBytes != defaultCleanupMaxBytes {
		t.Fatalf("preview policy = %#v, want declared 30d/10/20GiB bounds", preview)
	}

	missingKey := httptest.NewRecorder()
	server.handleCleanupApply(missingKey, cleanupJSONRequest(t, "/api/v1/cleanup/apply", ownerCleanupApplyRequest{ProviderID: testGenieCleanupProviderID, Preview: preview, ApprovalMode: "owner"}))
	if missingKey.Code != http.StatusBadRequest {
		t.Fatalf("missing-key status = %d, want 400", missingKey.Code)
	}

	// A forged apply may replace the policy with zero/unbounded values and add
	// IDs from recent or keep-count-protected runs. Apply must normalize the
	// owner bounds again and only delete IDs that remain eligible under them.
	preview.MinAgeSeconds = 0
	preview.KeepCount = 0
	preview.MaxBytes = 0
	preview.Items = append(preview.Items,
		ownerCleanupItem{ID: "demo/recent-run", Path: artifactpaths.RunDir(artifactRoot, "recent-run")},
		ownerCleanupItem{ID: "demo/kept-1", Path: artifactpaths.RunDir(artifactRoot, "kept-1")},
	)
	applyRequest := ownerCleanupApplyRequest{ProviderID: testGenieCleanupProviderID, Preview: preview, IdempotencyKey: "cleanup-once", ApprovalMode: "owner"}
	firstRecorder := httptest.NewRecorder()
	server.handleCleanupApply(firstRecorder, cleanupJSONRequest(t, "/api/v1/cleanup/apply", applyRequest))
	if firstRecorder.Code != http.StatusOK {
		t.Fatalf("apply status = %d body=%s", firstRecorder.Code, firstRecorder.Body.String())
	}
	var first ownerCleanupApplyResponse
	if err := json.NewDecoder(firstRecorder.Body).Decode(&first); err != nil {
		t.Fatal(err)
	}
	if first.ReclaimedBytes != 32 || len(first.RemovedItemIDs) != 1 || first.RemovedItemIDs[0] != "demo/old-run" {
		t.Fatalf("apply = %#v", first)
	}
	if len(first.SkippedItemIDs) != 2 || first.SkippedItemIDs[0] != "demo/recent-run" || first.SkippedItemIDs[1] != "demo/kept-1" {
		t.Fatalf("forged ineligible IDs were not skipped: %#v", first.SkippedItemIDs)
	}
	if _, err := os.Stat(artifactpaths.RunDir(artifactRoot, "recent-run")); err != nil {
		t.Fatalf("recent run was removed by forged weaker terms: %v", err)
	}
	if _, err := os.Stat(artifactpaths.RunDir(artifactRoot, "kept-1")); err != nil {
		t.Fatalf("keep-count run was removed by forged weaker terms: %v", err)
	}
	if _, err := os.Stat(runPath); !os.IsNotExist(err) {
		t.Fatalf("run path remains after owner cleanup: %v", err)
	}

	secondRecorder := httptest.NewRecorder()
	invalidReplay := applyRequest
	invalidReplay.Preview.MinAgeSeconds = -1
	invalidRecorder := httptest.NewRecorder()
	server.handleCleanupApply(invalidRecorder, cleanupJSONRequest(t, "/api/v1/cleanup/apply", invalidReplay))
	if invalidRecorder.Code != http.StatusBadRequest {
		t.Fatalf("negative-bound idempotent replay status = %d body=%s, want 400", invalidRecorder.Code, invalidRecorder.Body.String())
	}

	server.handleCleanupApply(secondRecorder, cleanupJSONRequest(t, "/api/v1/cleanup/apply", applyRequest))
	var second ownerCleanupApplyResponse
	if err := json.NewDecoder(secondRecorder.Body).Decode(&second); err != nil {
		t.Fatal(err)
	}
	if !second.AlreadyDone || second.ReclaimedBytes != 0 || len(second.RemovedItemIDs) != 0 {
		t.Fatalf("idempotent replay = %#v, first = %#v", second, first)
	}

	receipts, err := ledger.Read()
	if err != nil {
		t.Fatal(err)
	}
	var removed bool
	for _, receipt := range receipts {
		if receipt.Outcome == artifactledger.OutcomeRemoved && receipt.Path == runPath && receipt.Component == "test-genie-owner-cleanup" {
			removed = true
		}
	}
	if !removed {
		t.Fatalf("removal receipt not found in %#v", receipts)
	}
}

func TestOwnerCleanupProtectsActivePinnedAndKeptRuns(t *testing.T) {
	storageRoot := filepath.Join(t.TempDir(), "storage")
	t.Setenv("VROOLI_STORAGE_ROOT", storageRoot)
	scenariosRoot := filepath.Join(t.TempDir(), "scenarios")
	writeCleanupScenarioManifest(t, scenariosRoot, "demo")
	artifactRoot, err := artifactpaths.ScenarioRoot("demo")
	if err != nil {
		t.Fatal(err)
	}
	index := sharedruns.NewIndex(artifactRoot)
	now := time.Now().UTC()
	for _, record := range []sharedruns.RunRecord{
		{RunID: "active", Scenario: "demo", StartedAt: now.Add(-72 * time.Hour), Status: sharedruns.StatusInProgress},
		{RunID: "newest", Scenario: "demo", StartedAt: now.Add(-24 * time.Hour), CompletedAt: now.Add(-23 * time.Hour), Status: sharedruns.StatusPassed},
		{RunID: "old", Scenario: "demo", StartedAt: now.Add(-96 * time.Hour), CompletedAt: now.Add(-95 * time.Hour), Status: sharedruns.StatusFailed},
	} {
		if err := index.Append(record); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(artifactpaths.RunDir(artifactRoot, record.RunID), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := sharedruns.NewPinLeaseStore(artifactRoot).Grant("old", "baseline", "test", time.Hour, now); err != nil {
		t.Fatal(err)
	}
	server := &Server{
		repoRoot:    filepath.Dir(scenariosRoot),
		runsService: appruns.NewService(scenariosRoot, nil, nil, nil), logger: log.New(io.Discard, "", 0),
	}
	candidates, err := server.cleanupCandidates(context.Background(), cleanupPolicy{KeepCount: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 0 {
		t.Fatalf("protected runs became cleanup candidates: %#v", candidates)
	}
}

func TestCleanupPolicyDefaultsMatchOwnerRetentionDeclaration(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/cleanup/estimate", nil)
	policy, err := cleanupPolicyFromQuery(request)
	if err != nil {
		t.Fatal(err)
	}
	if policy.MinAgeSeconds != int64((30*24*time.Hour)/time.Second) || policy.KeepCount != 10 || policy.MaxBytes != int64(20<<30) {
		t.Fatalf("default cleanup policy = %#v", policy)
	}
}

func TestBoundedCleanupPolicyRejectsNegativeBounds(t *testing.T) {
	cases := []cleanupPolicy{
		{MinAgeSeconds: -1, KeepCount: defaultCleanupKeepCount, MaxBytes: defaultCleanupMaxBytes},
		{MinAgeSeconds: defaultCleanupMinAgeSeconds, KeepCount: -1, MaxBytes: defaultCleanupMaxBytes},
		{MinAgeSeconds: defaultCleanupMinAgeSeconds, KeepCount: defaultCleanupKeepCount, MaxBytes: -1},
	}
	for _, policy := range cases {
		if _, err := boundedCleanupPolicy(policy); err == nil {
			t.Fatalf("boundedCleanupPolicy(%#v) error=nil, want refusal", policy)
		}
	}
}

func TestCleanupHTTPHandlersRejectNegativeBounds(t *testing.T) {
	server := &Server{cleanupResults: make(map[string]ownerCleanupApplyResponse)}

	estimate := httptest.NewRecorder()
	server.handleCleanupEstimate(estimate, httptest.NewRequest(http.MethodGet, "/api/v1/cleanup/estimate?min_age_seconds=-1", nil))
	if estimate.Code != http.StatusBadRequest {
		t.Fatalf("negative estimate status = %d body=%s, want 400", estimate.Code, estimate.Body.String())
	}

	preview := httptest.NewRecorder()
	server.handleCleanupPreview(preview, cleanupJSONRequest(t, "/api/v1/cleanup/preview", ownerCleanupPreviewRequest{
		Estimate: ownerCleanupEstimate{MinAgeSeconds: -1, KeepCount: defaultCleanupKeepCount, MaxBytes: defaultCleanupMaxBytes},
	}))
	if preview.Code != http.StatusBadRequest {
		t.Fatalf("negative preview status = %d body=%s, want 400", preview.Code, preview.Body.String())
	}

	apply := httptest.NewRecorder()
	server.handleCleanupApply(apply, cleanupJSONRequest(t, "/api/v1/cleanup/apply", ownerCleanupApplyRequest{
		ProviderID: testGenieCleanupProviderID, IdempotencyKey: "negative-bounds", ApprovalMode: "owner",
		Preview: ownerCleanupPreview{MinAgeSeconds: defaultCleanupMinAgeSeconds, KeepCount: -1, MaxBytes: defaultCleanupMaxBytes},
	}))
	if apply.Code != http.StatusBadRequest {
		t.Fatalf("negative apply status = %d body=%s, want 400", apply.Code, apply.Body.String())
	}
}

func TestBoundedCleanupPolicyKeepsTighterPositiveByteCeiling(t *testing.T) {
	policy, err := boundedCleanupPolicy(cleanupPolicy{MinAgeSeconds: defaultCleanupMinAgeSeconds, KeepCount: defaultCleanupKeepCount, MaxBytes: 1024})
	if err != nil {
		t.Fatal(err)
	}
	if policy.MaxBytes != 1024 {
		t.Fatalf("tighter max-bytes policy = %#v, want 1024-byte ceiling", policy)
	}
}

func TestCleanupReceiptPathUsesExistingLogsForLogsOnlyRun(t *testing.T) {
	root := t.TempDir()
	runPath := filepath.Join(root, "runs", "old")
	logsPath := filepath.Join(root, "logs", "old")
	if err := os.MkdirAll(logsPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := cleanupReceiptPath(runPath, logsPath); got != logsPath {
		t.Fatalf("receipt path = %q, want existing logs path %q", got, logsPath)
	}
}

func cleanupJSONRequest(t *testing.T, path string, payload any) *http.Request {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return httptest.NewRequest(http.MethodPost, path, bytes.NewReader(data))
}

func writeCleanupScenarioManifest(t *testing.T, scenariosRoot, scenario string) {
	t.Helper()
	dir := filepath.Join(scenariosRoot, scenario, ".vrooli")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "service.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
}
