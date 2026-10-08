package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// handleReviewSummary handles GET /api/v1/review/summary?scenarioName=X
func (s *Server) handleReviewSummary(w http.ResponseWriter, r *http.Request) {
	hctx := RepoRead(w, r, s.git, s.repos, 30*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	scenarioName := strings.TrimSpace(r.URL.Query().Get("scenarioName"))
	if scenarioName == "" {
		hctx.Resp.BadRequest("scenarioName query parameter is required")
		return
	}
	if !IsValidScenarioName(scenarioName) {
		hctx.Resp.BadRequest("scenarioName contains invalid characters")
		return
	}

	detailCount, _ := strconv.Atoi(r.URL.Query().Get("details"))
	if detailCount < 0 {
		detailCount = 0
	}

	summary := s.buildReviewSummary(hctx.Ctx, hctx.RepoID, hctx.RepoDir, scenarioName, detailCount, DefaultReadinessThresholds(), "", nil)
	hctx.Resp.OK(summary)
}

// handleReviewRun handles POST /api/v1/review/run
func (s *Server) handleReviewRun(w http.ResponseWriter, r *http.Request) {
	hctx := RepoRead(w, r, s.git, s.repos, 10*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	var req ReviewRunRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}

	response, err := s.startReviewRun(hctx.Ctx, hctx.RepoID, hctx.RepoDir, req)
	if err != nil {
		if conflict, ok := err.(*reviewRunConflictError); ok {
			hctx.Resp.JSON(http.StatusConflict, errorResponse{
				Error: "a review run is already in progress for this scenario",
				JobID: conflict.JobID,
			})
			return
		}
		hctx.Resp.BadRequest(err.Error())
		return
	}
	hctx.Resp.OK(response)
}

type reviewRunConflictError struct{ JobID string }

func (e *reviewRunConflictError) Error() string { return "review run already in progress" }

func (s *Server) startReviewRun(ctx context.Context, repoID int64, repoDir string, req ReviewRunRequest) (ReviewRunResponse, error) {
	scenarioName := strings.TrimSpace(req.ScenarioName)
	if scenarioName == "" {
		return ReviewRunResponse{}, fmt.Errorf("scenarioName is required")
	}
	if !IsValidScenarioName(scenarioName) {
		return ReviewRunResponse{}, fmt.Errorf("scenarioName contains invalid characters")
	}
	checks, invalidCheck := resolveReviewChecks(req.Checks)
	if invalidCheck != "" {
		return ReviewRunResponse{}, fmt.Errorf("unknown check: %s", invalidCheck)
	}
	detailCount := clampNonNegative(req.Details)
	thresholds := resolveThresholds(req.Thresholds)
	idempotencyKey := reviewRequestIdempotencyKey(strconv.FormatInt(repoID, 10), scenarioName, checks, detailCount, thresholds)
	if existingID, ok := s.reviewJobStore.FindByIdempotency(idempotencyKey); ok {
		return ReviewRunResponse{JobID: existingID}, nil
	}
	if existingID := s.reviewJobStore.ActiveJobForScenario(scenarioName); existingID != "" {
		return ReviewRunResponse{}, &reviewRunConflictError{JobID: existingID}
	}
	jobID := uuid.New().String()
	s.reviewJobStore.CreateWithIdempotency(jobID, idempotencyKey, checks, scenarioName, detailCount, thresholds)
	go s.executeReviewRun(jobID, scenarioName, checks, repoID, repoDir)
	return ReviewRunResponse{JobID: jobID}, nil
}

func reviewRequestIdempotencyKey(repoID, scenario string, checks []string, details int, thresholds ReadinessThresholds) string {
	payload := struct {
		RepoID, Scenario string
		Checks           []string
		Details          int
		Thresholds       ReadinessThresholds
	}{repoID, scenario, checks, details, thresholds}
	data, _ := json.Marshal(payload)
	digest := sha256.Sum256(data)
	return "review:" + hex.EncodeToString(digest[:])
}

// resolveReviewChecks returns the validated checks list and the first invalid check (if any).
func resolveReviewChecks(checks []string) ([]string, string) {
	if len(checks) == 0 {
		return []string{"tidiness", "tests", "rules"}, ""
	}
	for _, c := range checks {
		if !validReviewChecks[c] {
			return nil, c
		}
	}
	return checks, ""
}

// clampNonNegative returns 0 if v is negative.
func clampNonNegative(v int) int {
	if v < 0 {
		return 0
	}
	return v
}

// resolveThresholds returns the provided thresholds or defaults.
func resolveThresholds(t *ReadinessThresholds) ReadinessThresholds {
	if t != nil {
		return *t
	}
	return DefaultReadinessThresholds()
}

// handleReviewJobStatus handles GET /api/v1/review/run/{jobId}
func (s *Server) handleReviewJobStatus(w http.ResponseWriter, r *http.Request) {
	hctx := RepoRead(w, r, s.git, s.repos, 5*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	jobID := strings.TrimSpace(mux.Vars(r)["jobId"])
	if jobID == "" {
		hctx.Resp.BadRequest("jobId is required")
		return
	}

	job, ok := s.reviewJobStore.Get(jobID)
	if !ok {
		hctx.Resp.NotFound("review job not found")
		return
	}

	hctx.Resp.OK(job)
}

// buildReviewSummary fans out concurrently to available downstream services and
// assembles a ReviewSummaryResponse. When detailCount > 0, each dimension
// includes top-K detail items extracted from the already-fetched service data.
func (s *Server) buildReviewSummary(ctx context.Context, repoID int64, repoDir, scenarioName string, detailCount int, thresholds ReadinessThresholds, testExecutionID string, checkStatuses map[string]CheckStatus) *ReviewSummaryResponse {
	var dims ReviewDimensions
	caps := make(map[string]bool)
	var mu sync.Mutex
	var wg sync.WaitGroup

	wg.Add(5)
	go func() {
		defer wg.Done()
		dim, avail := s.fetchCodeQualityDimension(ctx, scenarioName, detailCount)
		mu.Lock()
		caps["tidiness-manager"] = avail
		dims.CodeQuality = dim
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		dim, avail := s.fetchTestsDimension(ctx, scenarioName, testExecutionID, detailCount)
		mu.Lock()
		caps["test-genie"] = avail
		dims.Tests = dim
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		dim, avail := s.fetchStandardsDimension(ctx, scenarioName, detailCount)
		mu.Lock()
		caps["scenario-auditor"] = avail
		dims.Standards = dim
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		dim, avail := s.fetchVisualDimension(ctx, repoID, scenarioName, detailCount)
		mu.Lock()
		caps["browser-automation-studio"] = avail
		dims.Visual = dim
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		dim, avail := s.fetchProvenanceDimension(ctx, repoDir, detailCount)
		mu.Lock()
		caps["workspace-sandbox"] = avail
		dims.Provenance = dim
		mu.Unlock()
	}()

	wg.Wait()

	readiness := CalculateReadiness(dims, thresholds)
	dimStatuses := CalculateDimensionStatuses(dims, thresholds)

	return &ReviewSummaryResponse{
		ScenarioName:      scenarioName,
		Readiness:         readiness,
		Dimensions:        dims,
		DimensionStatuses: dimStatuses,
		Capabilities:      caps,
		CheckStatuses:     checkStatuses,
		Timestamp:         time.Now().UTC().Format(time.RFC3339),
	}
}

// executeReviewRun runs checks in the background and updates the job store.
func (s *Server) executeReviewRun(jobID, scenarioName string, checks []string, repoID int64, repoDir string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	var wg sync.WaitGroup
	for _, check := range checks {
		check := check
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.runSingleCheck(ctx, jobID, check, scenarioName)
		}()
	}
	wg.Wait()

	if allChecksFailed(s.reviewJobStore, jobID) {
		s.reviewJobStore.Fail(jobID, "all checks failed or were skipped")
		return
	}

	dc := s.reviewJobStore.DetailCount(jobID)
	th := s.reviewJobStore.Thresholds(jobID)
	job, _ := s.reviewJobStore.Get(jobID)
	var checkStatuses map[string]CheckStatus
	if job != nil {
		checkStatuses = job.Checks
	}
	summary := s.buildReviewSummary(ctx, repoID, repoDir, scenarioName, dc, th, s.reviewJobStore.CheckExecutionID(jobID, "tests"), checkStatuses)
	s.reviewJobStore.Complete(jobID, summary)
}

// runSingleCheck dispatches a single review check and updates the job store.
func (s *Server) runSingleCheck(ctx context.Context, jobID, check, scenarioName string) {
	s.reviewJobStore.UpdateCheck(jobID, check, CheckRunning)

	executionID, err := s.dispatchCheck(ctx, check, scenarioName)
	if err == errCheckSkipped {
		s.reviewJobStore.UpdateCheck(jobID, check, CheckSkipped)
		return
	}
	if err != nil {
		log.Printf("review run %s check %s failed: %v", jobID, check, err)
		s.reviewJobStore.UpdateCheck(jobID, check, CheckFailed)
		return
	}
	s.reviewJobStore.UpdateCheckWithExecution(jobID, check, CheckCompleted, executionID)
}

// errCheckSkipped is a sentinel indicating the check was skipped.
var errCheckSkipped = fmt.Errorf("check skipped")

// dispatchCheck runs the appropriate check and returns an error or errCheckSkipped.
func (s *Server) dispatchCheck(ctx context.Context, check, scenarioName string) (string, error) {
	switch check {
	case "tidiness":
		if !s.capabilities.IsAvailable(ctx, "tidiness-manager") {
			return "", errCheckSkipped
		}
		repoRoot := strings.TrimSpace(s.git.ResolveRepoRoot(ctx))
		if repoRoot == "" {
			return "", nil
		}
		scenarioPath, err := resolveScenarioPath(repoRoot, scenarioName)
		if err != nil {
			return "", err
		}
		_, err = s.tidinessClient.TriggerLightScan(ctx, TidinessLightScanRequest{
			ScenarioPath: scenarioPath,
		})
		return "", err
	case "tests":
		if !s.capabilities.IsAvailable(ctx, "test-genie") {
			return "", errCheckSkipped
		}
		result, err := s.testGenieClient.ExecuteSuite(ctx, TestExecutionRequest{
			ScenarioName: scenarioName,
		})
		if result == nil {
			return "", err
		}
		return result.ExecutionID, err
	case "rules":
		if !s.capabilities.IsAvailable(ctx, "scenario-auditor") {
			return "", errCheckSkipped
		}
		result, err := s.auditorClient.StartCheck(ctx, scenarioName, "full")
		if result == nil {
			return "", err
		}
		return result.JobID, err
	default:
		return "", errCheckSkipped
	}
}

// allChecksFailed returns true if no check completed successfully.
func allChecksFailed(store *ReviewJobStore, jobID string) bool {
	job, _ := store.Get(jobID)
	if job == nil {
		return true
	}
	for _, status := range job.Checks {
		if status == CheckCompleted {
			return false
		}
	}
	return true
}
