package main

import (
	"context"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// handleReviewSummary handles GET /api/v1/review/summary?scenarioName=X
func (s *Server) handleReviewSummary(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, nil, 30*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	scenarioName := strings.TrimSpace(r.URL.Query().Get("scenarioName"))
	if scenarioName == "" {
		hctx.Resp.BadRequest("scenarioName query parameter is required")
		return
	}

	summary := s.buildReviewSummary(hctx.Ctx, hctx.RepoID, hctx.RepoDir, scenarioName)
	hctx.Resp.OK(summary)
}

// handleReviewRun handles POST /api/v1/review/run
func (s *Server) handleReviewRun(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, nil, 10*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	var req ReviewRunRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}

	scenarioName := strings.TrimSpace(req.ScenarioName)
	if scenarioName == "" {
		hctx.Resp.BadRequest("scenarioName is required")
		return
	}

	jobID := uuid.New().String()
	checks := req.Checks
	if len(checks) == 0 {
		checks = []string{"tidiness", "tests", "rules"}
	}

	s.reviewJobStore.Create(jobID, checks)

	repoID := hctx.RepoID
	repoDir := hctx.RepoDir

	go s.executeReviewRun(jobID, scenarioName, checks, repoID, repoDir)

	hctx.Resp.OK(ReviewRunResponse{JobID: jobID})
}

// handleReviewJobStatus handles GET /api/v1/review/run/{jobId}
func (s *Server) handleReviewJobStatus(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, nil, 5*time.Second)
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
// assembles a ReviewSummaryResponse.
func (s *Server) buildReviewSummary(ctx context.Context, repoID int64, repoDir, scenarioName string) *ReviewSummaryResponse {
	var dims ReviewDimensions
	caps := make(map[string]bool)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Code quality from tidiness-manager
	wg.Add(1)
	go func() {
		defer wg.Done()
		available := s.capabilities.IsAvailable(ctx, "tidiness-manager")
		mu.Lock()
		caps["tidiness-manager"] = available
		mu.Unlock()
		if !available {
			return
		}

		dim := &CodeQualityDimension{Available: true}
		score, err := s.tidinessClient.GetTidinessScore(ctx, scenarioName)
		if err == nil && score != nil {
			dim.Score = score.Score
			dim.Violations = score.Violations
			if score.LastScan != nil {
				dim.LastScan = score.LastScan.Format(time.RFC3339)
			}
		}
		staleness, err := s.tidinessClient.GetStaleness(ctx, scenarioName)
		if err == nil && staleness != nil {
			dim.Stale = staleness.IsStale
		}
		mu.Lock()
		dims.CodeQuality = dim
		mu.Unlock()
	}()

	// Tests from test-genie
	wg.Add(1)
	go func() {
		defer wg.Done()
		available := s.capabilities.IsAvailable(ctx, "test-genie")
		mu.Lock()
		caps["test-genie"] = available
		mu.Unlock()
		if !available {
			return
		}

		dim := &TestsDimension{Available: true}
		list, err := s.testGenieClient.ListExecutions(ctx, scenarioName, 1)
		if err == nil && list != nil && len(list.Items) > 0 {
			latest := list.Items[0]
			dim.Passed = latest.Success
			dim.Total = latest.PhaseSummary.Total
			dim.PassedCount = latest.PhaseSummary.Passed
			dim.FailedCount = latest.PhaseSummary.Failed
			if latest.CompletedAt != "" {
				dim.LastRun = latest.CompletedAt
			}
		}
		mu.Lock()
		dims.Tests = dim
		mu.Unlock()
	}()

	// Standards from scenario-auditor
	wg.Add(1)
	go func() {
		defer wg.Done()
		available := s.capabilities.IsAvailable(ctx, "scenario-auditor")
		mu.Lock()
		caps["scenario-auditor"] = available
		mu.Unlock()
		if !available {
			return
		}

		dim := &StandardsDimension{Available: true}
		violations, err := s.auditorClient.GetViolations(ctx, scenarioName)
		if err == nil && violations != nil {
			for _, v := range violations.Violations {
				dim.TotalViolations++
				switch v.Severity {
				case "critical", "error":
					dim.BlockingViolations++
				case "warning":
					dim.Warnings++
				default:
					dim.Warnings++
				}
			}
		}
		mu.Lock()
		dims.Standards = dim
		mu.Unlock()
	}()

	// Visual captures from local storage
	wg.Add(1)
	go func() {
		defer wg.Done()
		available := s.capabilities.IsAvailable(ctx, "browser-automation-studio")
		mu.Lock()
		caps["browser-automation-studio"] = available
		mu.Unlock()

		dim := &VisualDimension{Available: available}
		snapshots, err := s.visualCaptureStorage.ListSnapshotSets(repoID, scenarioName)
		if err == nil {
			for _, snap := range snapshots {
				if snap.Status == "complete" {
					dim.ScreenshotCount += snap.ScreenshotCount
				}
			}
		}
		mu.Lock()
		dims.Visual = dim
		mu.Unlock()
	}()

	// Provenance from workspace-sandbox
	wg.Add(1)
	go func() {
		defer wg.Done()
		available := s.capabilities.IsAvailable(ctx, "workspace-sandbox")
		mu.Lock()
		caps["workspace-sandbox"] = available
		mu.Unlock()
		if !available {
			return
		}

		dim := &ProvenanceDimension{Available: true}
		prov, err := s.sandbox.GetProvenanceByRun(ctx, repoDir)
		if err == nil && prov != nil {
			fileSet := make(map[string]struct{})
			for _, g := range prov.RunGroups {
				for _, f := range g.Files {
					fileSet[f.FilePath] = struct{}{}
				}
			}
			dim.TracedFiles = len(fileSet)
		}
		mu.Lock()
		dims.Provenance = dim
		mu.Unlock()
	}()

	wg.Wait()

	readiness := CalculateReadiness(dims)

	return &ReviewSummaryResponse{
		ScenarioName: scenarioName,
		Readiness:    readiness,
		Dimensions:   dims,
		Capabilities: caps,
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
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
			s.reviewJobStore.UpdateCheck(jobID, check, CheckRunning)

			var err error
			switch check {
			case "tidiness":
				if s.capabilities.IsAvailable(ctx, "tidiness-manager") {
					repoRoot := strings.TrimSpace(s.git.ResolveRepoRoot(ctx))
					if repoRoot != "" {
						req := TidinessLightScanRequest{
							ScenarioPath: repoRoot + "/scenarios/" + scenarioName,
						}
						_, err = s.tidinessClient.TriggerLightScan(ctx, req)
					}
				} else {
					s.reviewJobStore.UpdateCheck(jobID, check, CheckSkipped)
					return
				}
			case "tests":
				if s.capabilities.IsAvailable(ctx, "test-genie") {
					_, err = s.testGenieClient.ExecuteSuite(ctx, TestExecutionRequest{
						ScenarioName: scenarioName,
					})
				} else {
					s.reviewJobStore.UpdateCheck(jobID, check, CheckSkipped)
					return
				}
			case "rules":
				if s.capabilities.IsAvailable(ctx, "scenario-auditor") {
					_, err = s.auditorClient.StartCheck(ctx, scenarioName, "full")
				} else {
					s.reviewJobStore.UpdateCheck(jobID, check, CheckSkipped)
					return
				}
			default:
				s.reviewJobStore.UpdateCheck(jobID, check, CheckSkipped)
				return
			}

			if err != nil {
				log.Printf("review run %s check %s failed: %v", jobID, check, err)
				s.reviewJobStore.UpdateCheck(jobID, check, CheckFailed)
			} else {
				s.reviewJobStore.UpdateCheck(jobID, check, CheckCompleted)
			}
		}()
	}

	wg.Wait()

	// Build final summary after all checks complete.
	summary := s.buildReviewSummary(ctx, repoID, repoDir, scenarioName)
	s.reviewJobStore.Complete(jobID, summary)
}
