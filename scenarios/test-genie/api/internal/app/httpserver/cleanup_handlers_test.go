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
	if err := sharedruns.NewIndex(artifactRoot).Append(sharedruns.RunRecord{
		RunID: runID, Scenario: "demo", StartedAt: time.Now().UTC().Add(-48 * time.Hour),
		CompletedAt: time.Now().UTC().Add(-47 * time.Hour), Status: sharedruns.StatusPassed,
	}); err != nil {
		t.Fatal(err)
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
	server.handleCleanupEstimate(estimateRecorder, httptest.NewRequest(http.MethodGet, "/api/v1/cleanup/estimate?min_age_seconds=3600&keep_count=0", nil))
	if estimateRecorder.Code != http.StatusOK {
		t.Fatalf("estimate status = %d body=%s", estimateRecorder.Code, estimateRecorder.Body.String())
	}
	var estimate ownerCleanupEstimate
	if err := json.NewDecoder(estimateRecorder.Body).Decode(&estimate); err != nil {
		t.Fatal(err)
	}
	if estimate.ItemCount != 1 || estimate.EstimatedBytes != 32 {
		t.Fatalf("estimate = %#v, want one 32-byte run", estimate)
	}

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

	missingKey := httptest.NewRecorder()
	server.handleCleanupApply(missingKey, cleanupJSONRequest(t, "/api/v1/cleanup/apply", ownerCleanupApplyRequest{ProviderID: testGenieCleanupProviderID, Preview: preview, ApprovalMode: "owner"}))
	if missingKey.Code != http.StatusBadRequest {
		t.Fatalf("missing-key status = %d, want 400", missingKey.Code)
	}

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
	if first.ReclaimedBytes != 32 || len(first.RemovedItemIDs) != 1 {
		t.Fatalf("apply = %#v", first)
	}
	if _, err := os.Stat(runPath); !os.IsNotExist(err) {
		t.Fatalf("run path remains after owner cleanup: %v", err)
	}

	secondRecorder := httptest.NewRecorder()
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
