package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/vrooli/packages/artifactledger"
	"github.com/vrooli/vrooli/packages/artifactpaths"
	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

const testGenieCleanupProviderID = "test-genie-run-retention"

const (
	defaultCleanupMinAgeSeconds = int64((30 * 24 * time.Hour) / time.Second)
	defaultCleanupKeepCount     = 10
	defaultCleanupMaxBytes      = int64(20 << 30)
)

type ownerCleanupEstimate struct {
	ProviderID     string    `json:"provider_id"`
	EstimatedBytes int64     `json:"estimated_bytes"`
	ItemCount      int       `json:"item_count"`
	BlockedReason  string    `json:"blocked_reason,omitempty"`
	ObservedAt     time.Time `json:"observed_at"`
	MinAgeSeconds  int64     `json:"min_age_seconds,omitempty"`
	KeepCount      int       `json:"keep_count,omitempty"`
	MaxBytes       int64     `json:"max_bytes,omitempty"`
}

type ownerCleanupPreviewRequest struct {
	Estimate ownerCleanupEstimate `json:"estimate"`
}

type ownerCleanupItem struct {
	ID         string `json:"id"`
	Path       string `json:"path"`
	Bytes      int64  `json:"bytes"`
	AgeSeconds int64  `json:"age_seconds"`
	Protected  bool   `json:"protected"`
}

type ownerCleanupPreview struct {
	ProviderID    string             `json:"provider_id"`
	Items         []ownerCleanupItem `json:"items"`
	BlockedReason string             `json:"blocked_reason,omitempty"`
	Warnings      []string           `json:"warnings,omitempty"`
	MinAgeSeconds int64              `json:"min_age_seconds,omitempty"`
	KeepCount     int                `json:"keep_count,omitempty"`
	MaxBytes      int64              `json:"max_bytes,omitempty"`
}

type ownerCleanupApplyRequest struct {
	ProviderID     string              `json:"provider_id"`
	Preview        ownerCleanupPreview `json:"preview"`
	IdempotencyKey string              `json:"idempotency_key"`
	ApprovalMode   string              `json:"approval_mode"`
}

type ownerCleanupApplyResponse struct {
	ReclaimedBytes int64    `json:"reclaimed_bytes"`
	RemovedItemIDs []string `json:"removed_item_ids"`
	SkippedItemIDs []string `json:"skipped_item_ids"`
	Warnings       []string `json:"warnings,omitempty"`
	AlreadyDone    bool     `json:"already_done,omitempty"`
}

type cleanupPolicy struct {
	MinAgeSeconds int64
	KeepCount     int
	MaxBytes      int64
}

type cleanupCandidate struct {
	ownerCleanupItem
	scenario string
	runID    string
}

func (s *Server) handleCleanupEstimate(w http.ResponseWriter, r *http.Request) {
	policy, err := cleanupPolicyFromQuery(r)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	candidates, err := s.cleanupCandidates(r.Context(), policy)
	if err != nil {
		s.writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	var bytes int64
	for _, candidate := range candidates {
		bytes += candidate.Bytes
	}
	s.writeJSON(w, http.StatusOK, ownerCleanupEstimate{
		ProviderID: testGenieCleanupProviderID, EstimatedBytes: bytes, ItemCount: len(candidates),
		ObservedAt: time.Now().UTC(), MinAgeSeconds: policy.MinAgeSeconds,
		KeepCount: policy.KeepCount, MaxBytes: policy.MaxBytes,
	})
}

func (s *Server) handleCleanupPreview(w http.ResponseWriter, r *http.Request) {
	var request ownerCleanupPreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.writeError(w, http.StatusBadRequest, "valid cleanup estimate is required")
		return
	}
	policy := cleanupPolicy{MinAgeSeconds: request.Estimate.MinAgeSeconds, KeepCount: request.Estimate.KeepCount, MaxBytes: request.Estimate.MaxBytes}
	candidates, err := s.cleanupCandidates(r.Context(), policy)
	if err != nil {
		s.writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	items := make([]ownerCleanupItem, 0, len(candidates))
	for _, candidate := range candidates {
		items = append(items, candidate.ownerCleanupItem)
	}
	s.writeJSON(w, http.StatusOK, ownerCleanupPreview{ProviderID: testGenieCleanupProviderID, Items: items, MinAgeSeconds: policy.MinAgeSeconds, KeepCount: policy.KeepCount, MaxBytes: policy.MaxBytes})
}

func (s *Server) handleCleanupApply(w http.ResponseWriter, r *http.Request) {
	var request ownerCleanupApplyRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil || strings.TrimSpace(request.IdempotencyKey) == "" {
		s.writeError(w, http.StatusBadRequest, "idempotency_key and valid cleanup preview are required")
		return
	}
	if request.ProviderID != testGenieCleanupProviderID || request.Preview.ProviderID != testGenieCleanupProviderID {
		s.writeError(w, http.StatusBadRequest, "cleanup provider does not match test-genie owner")
		return
	}
	if request.ApprovalMode != "owner" && request.ApprovalMode != "operator" {
		s.writeError(w, http.StatusForbidden, "owner or operator approval is required")
		return
	}

	s.cleanupMu.Lock()
	defer s.cleanupMu.Unlock()
	if result, ok := s.cleanupResults[request.IdempotencyKey]; ok {
		result.AlreadyDone = true
		result.ReclaimedBytes = 0
		result.RemovedItemIDs = []string{}
		s.writeJSON(w, http.StatusOK, result)
		return
	}
	policy := cleanupPolicy{MinAgeSeconds: request.Preview.MinAgeSeconds, KeepCount: request.Preview.KeepCount, MaxBytes: request.Preview.MaxBytes}
	current, err := s.cleanupCandidates(r.Context(), policy)
	if err != nil {
		s.writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	byID := make(map[string]cleanupCandidate, len(current))
	for _, candidate := range current {
		byID[candidate.ID] = candidate
	}
	ledger, err := s.cleanupLedger()
	if err != nil {
		s.writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	result := ownerCleanupApplyResponse{RemovedItemIDs: []string{}, SkippedItemIDs: []string{}}
	for _, previewed := range request.Preview.Items {
		candidate, ok := byID[previewed.ID]
		if !ok || candidate.Protected {
			result.SkippedItemIDs = append(result.SkippedItemIDs, previewed.ID)
			continue
		}
		err := ledger.Record(artifactledger.Removal{
			Path: candidate.Path, Kind: "test-run", Component: "test-genie-owner-cleanup",
			Predicate: fmt.Sprintf("run remained terminal, unleased, older than %ds, and outside keep-count %d", policy.MinAgeSeconds, policy.KeepCount),
		}, func() error {
			_, deleteErr := s.runsService.DeleteRun(r.Context(), connect.NewRequest(&runspb.DeleteRunRequest{Target: candidate.scenario, RunId: candidate.runID}))
			return deleteErr
		})
		if errors.Is(err, fs.ErrNotExist) {
			result.SkippedItemIDs = append(result.SkippedItemIDs, previewed.ID)
			continue
		}
		if err != nil {
			result.SkippedItemIDs = append(result.SkippedItemIDs, previewed.ID)
			result.Warnings = append(result.Warnings, candidate.ID+": "+err.Error())
			continue
		}
		result.RemovedItemIDs = append(result.RemovedItemIDs, candidate.ID)
		result.ReclaimedBytes += candidate.Bytes
	}
	s.cleanupResults[request.IdempotencyKey] = result
	s.writeJSON(w, http.StatusOK, result)
}

func cleanupPolicyFromQuery(r *http.Request) (cleanupPolicy, error) {
	// These are the owner declaration's retention defaults. Storage Manager may
	// tighten an explicitly supplied bound, but an omitted query parameter must
	// never silently weaken the owner policy to zero.
	policy := cleanupPolicy{MinAgeSeconds: defaultCleanupMinAgeSeconds, KeepCount: defaultCleanupKeepCount, MaxBytes: defaultCleanupMaxBytes}
	var err error
	if raw := strings.TrimSpace(r.URL.Query().Get("min_age_seconds")); raw != "" {
		policy.MinAgeSeconds, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || policy.MinAgeSeconds < 0 {
			return cleanupPolicy{}, fmt.Errorf("min_age_seconds must be a non-negative integer")
		}
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("keep_count")); raw != "" {
		policy.KeepCount, err = strconv.Atoi(raw)
		if err != nil || policy.KeepCount < 0 {
			return cleanupPolicy{}, fmt.Errorf("keep_count must be a non-negative integer")
		}
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("max_bytes")); raw != "" {
		policy.MaxBytes, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || policy.MaxBytes < 0 {
			return cleanupPolicy{}, fmt.Errorf("max_bytes must be a non-negative integer")
		}
	}
	return policy, nil
}

func (s *Server) cleanupCandidates(ctx context.Context, policy cleanupPolicy) ([]cleanupCandidate, error) {
	if s.runsService == nil {
		return nil, fmt.Errorf("test-genie cleanup dependencies are unavailable")
	}
	scenarioIDs, err := cleanupScenarioIDs(s.repoRoot)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	var candidates []cleanupCandidate
	for _, scenario := range scenarioIDs {
		response, err := s.runsService.ListRuns(ctx, connect.NewRequest(&runspb.ListRunsRequest{Target: scenario}))
		if err != nil {
			return nil, fmt.Errorf("list %s runs for cleanup: %w", scenario, err)
		}
		root, err := artifactpaths.ScenarioRoot(scenario)
		if err != nil {
			return nil, fmt.Errorf("resolve %s artifact root: %w", scenario, err)
		}
		for index, run := range response.Msg.GetRuns() {
			if run == nil || strings.TrimSpace(run.GetRunId()) == "" {
				continue
			}
			protected := run.GetStatus() == "queued" || run.GetStatus() == "in_progress" || len(run.GetPins()) > 0 || index < policy.KeepCount
			observed, parseErr := time.Parse(time.RFC3339, run.GetCompletedAt())
			if parseErr != nil {
				observed, parseErr = time.Parse(time.RFC3339, run.GetStartedAt())
			}
			if parseErr != nil {
				protected = true
				observed = now
			}
			age := int64(now.Sub(observed).Seconds())
			if age < 0 {
				age = 0
			}
			if protected || age < policy.MinAgeSeconds {
				continue
			}
			runPath := artifactpaths.RunDir(root, run.GetRunId())
			logsPath := artifactpaths.RunLogsDir(root, run.GetRunId())
			bytes := cleanupDirectoryBytes(runPath) + cleanupDirectoryBytes(logsPath)
			candidates = append(candidates, cleanupCandidate{ownerCleanupItem: ownerCleanupItem{ID: scenario + "/" + run.GetRunId(), Path: cleanupReceiptPath(runPath, logsPath), Bytes: bytes, AgeSeconds: age}, scenario: scenario, runID: run.GetRunId()})
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].AgeSeconds > candidates[j].AgeSeconds })
	if policy.MaxBytes > 0 {
		var selected []cleanupCandidate
		var total int64
		for _, candidate := range candidates {
			if total+candidate.Bytes > policy.MaxBytes {
				continue
			}
			selected = append(selected, candidate)
			total += candidate.Bytes
		}
		candidates = selected
	}
	return candidates, nil
}

func cleanupReceiptPath(runPath, logsPath string) string {
	if _, err := os.Stat(runPath); err == nil {
		return runPath
	}
	if _, err := os.Stat(logsPath); err == nil {
		return logsPath
	}
	return runPath
}

func cleanupScenarioIDs(repoRoot string) ([]string, error) {
	repoRoot = filepath.Clean(strings.TrimSpace(repoRoot))
	if repoRoot == "." || repoRoot == "" {
		return nil, fmt.Errorf("repository root is unavailable for cleanup")
	}
	manifests, err := filepath.Glob(filepath.Join(repoRoot, "scenarios", "*", ".vrooli", "service.json"))
	if err != nil {
		return nil, fmt.Errorf("list scenario manifests for cleanup: %w", err)
	}
	ids := make([]string, 0, len(manifests))
	for _, manifest := range manifests {
		id := filepath.Base(filepath.Dir(filepath.Dir(manifest)))
		if id != "." && id != "" {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids, nil
}

func (s *Server) cleanupLedger() (*artifactledger.Ledger, error) {
	if s.removalLedger != nil {
		return s.removalLedger, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve removal-ledger home: %w", err)
	}
	ledger, err := artifactledger.New(home)
	if err != nil {
		return nil, fmt.Errorf("open removal ledger: %w", err)
	}
	s.removalLedger = ledger
	return ledger, nil
}

func cleanupDirectoryBytes(root string) int64 {
	var total int64
	_ = filepath.WalkDir(root, func(_ string, entry os.DirEntry, err error) error {
		if err == nil && !entry.IsDir() {
			if info, infoErr := entry.Info(); infoErr == nil {
				total += info.Size()
			}
		}
		return nil
	})
	return total
}
