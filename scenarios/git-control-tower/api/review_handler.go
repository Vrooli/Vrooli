package main

import (
	"context"
	"log"
	"net/http"
	"sort"
	"strconv"
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
	if !IsValidScenarioName(scenarioName) {
		hctx.Resp.BadRequest("scenarioName contains invalid characters")
		return
	}

	detailCount, _ := strconv.Atoi(r.URL.Query().Get("details"))
	if detailCount < 0 {
		detailCount = 0
	}

	summary := s.buildReviewSummary(hctx.Ctx, hctx.RepoID, hctx.RepoDir, scenarioName, detailCount, DefaultReadinessThresholds())
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
	if !IsValidScenarioName(scenarioName) {
		hctx.Resp.BadRequest("scenarioName contains invalid characters")
		return
	}

	checks := req.Checks
	if len(checks) == 0 {
		checks = []string{"tidiness", "tests", "rules"}
	}
	for _, c := range checks {
		if !validReviewChecks[c] {
			hctx.Resp.BadRequest("unknown check: " + c)
			return
		}
	}

	// Reject concurrent runs for the same scenario.
	if existingID := s.reviewJobStore.ActiveJobForScenario(scenarioName); existingID != "" {
		hctx.Resp.JSON(http.StatusConflict, errorResponse{
			Error: "a review run is already in progress for this scenario",
		})
		return
	}

	detailCount := req.Details
	if detailCount < 0 {
		detailCount = 0
	}

	thresholds := DefaultReadinessThresholds()
	if req.Thresholds != nil {
		thresholds = *req.Thresholds
	}

	jobID := uuid.New().String()
	s.reviewJobStore.Create(jobID, checks, scenarioName, detailCount, thresholds)

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
// assembles a ReviewSummaryResponse. When detailCount > 0, each dimension
// includes top-K detail items extracted from the already-fetched service data.
func (s *Server) buildReviewSummary(ctx context.Context, repoID int64, repoDir, scenarioName string, detailCount int, thresholds ReadinessThresholds) *ReviewSummaryResponse {
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
			if detailCount > 0 && score.Breakdown != nil {
				dim.TopIssues = buildCodeQualityIssues(score.Breakdown, detailCount)
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
			if detailCount > 0 {
				dim.Failures = buildTestFailures(latest.Phases, detailCount)
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
			if detailCount > 0 {
				dim.TopViolations = buildTopViolations(violations.Violations, detailCount)
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
		if !available {
			return
		}

		dim := &VisualDimension{Available: true}
		snapshots, err := s.visualCaptureStorage.ListSnapshotSets(repoID, scenarioName)
		if err == nil {
			var latestTime time.Time
			for _, snap := range snapshots {
				if snap.Status == "complete" {
					dim.ScreenshotCount += snap.ScreenshotCount
					if detailCount > 0 && snap.CreatedAt.After(latestTime) {
						latestTime = snap.CreatedAt
						dim.LatestCapture = &VisualCaptureMeta{
							CapturedAt:      snap.CreatedAt.UTC().Format(time.RFC3339),
							CommitHash:      snap.CommitHash,
							ScreenshotCount: snap.ScreenshotCount,
						}
					}
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
		tracedSet := make(map[string]struct{})
		if err == nil && prov != nil {
			for _, g := range prov.RunGroups {
				for _, f := range g.Files {
					tracedSet[f.FilePath] = struct{}{}
				}
			}
			dim.TracedFiles = len(tracedSet)
		}

		if detailCount > 0 && repoDir != "" {
			dim.UntracedFiles = buildUntracedFiles(ctx, s.git, repoDir, tracedSet, detailCount)
		}

		mu.Lock()
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
		Timestamp:         time.Now().UTC().Format(time.RFC3339),
	}
}

// buildCodeQualityIssues converts a TidinessBreakdown into a sorted slice of
// non-zero category issues, capped at limit.
func buildCodeQualityIssues(bd *TidinessBreakdown, limit int) []CodeQualityIssue {
	candidates := []CodeQualityIssue{
		{Category: "lint_issues", Count: bd.LintIssues},
		{Category: "type_issues", Count: bd.TypeIssues},
		{Category: "long_files", Count: bd.LongFiles},
		{Category: "complex_functions", Count: bd.ComplexFunctions},
		{Category: "tech_debt_markers", Count: bd.TechDebtMarkers},
		{Category: "duplication_issues", Count: bd.DuplicationIssues},
	}

	// Filter to non-zero and sort by count descending.
	var issues []CodeQualityIssue
	for _, c := range candidates {
		if c.Count > 0 {
			issues = append(issues, c)
		}
	}
	sort.Slice(issues, func(i, j int) bool {
		return issues[i].Count > issues[j].Count
	})
	if len(issues) > limit {
		issues = issues[:limit]
	}
	return issues
}

// buildTestFailures extracts failed phases from a test execution, capped at limit.
func buildTestFailures(phases []TestPhaseResult, limit int) []TestFailure {
	var failures []TestFailure
	for _, p := range phases {
		if p.Status != "failed" {
			continue
		}
		failures = append(failures, TestFailure{
			Phase:          p.Name,
			Error:          p.Error,
			Classification: p.Classification,
			Remediation:    p.Remediation,
		})
		if len(failures) >= limit {
			break
		}
	}
	return failures
}

// severityOrder maps severity strings to sort priority (lower = more severe).
var severityOrder = map[string]int{
	"critical": 0,
	"error":    1,
	"warning":  2,
}

// buildTopViolations sorts violations by severity (blocking first) and returns
// the top limit entries.
func buildTopViolations(violations []AuditorViolation, limit int) []StandardsViolationDetail {
	sorted := make([]AuditorViolation, len(violations))
	copy(sorted, violations)
	sort.SliceStable(sorted, func(i, j int) bool {
		oi, oki := severityOrder[sorted[i].Severity]
		oj, okj := severityOrder[sorted[j].Severity]
		if !oki {
			oi = 3
		}
		if !okj {
			oj = 3
		}
		return oi < oj
	})

	n := limit
	if n > len(sorted) {
		n = len(sorted)
	}
	details := make([]StandardsViolationDetail, n)
	for i := 0; i < n; i++ {
		v := sorted[i]
		details[i] = StandardsViolationDetail{
			FilePath:       v.FilePath,
			LineNumber:     v.LineNumber,
			Title:          v.Title,
			Severity:       v.Severity,
			Recommendation: v.Recommendation,
		}
	}
	return details
}

// buildUntracedFiles computes files in git status that are not in the provenance
// traced set. Returns up to limit file paths.
func buildUntracedFiles(ctx context.Context, git GitRunner, repoDir string, tracedSet map[string]struct{}, limit int) []string {
	raw, err := git.StatusPorcelainV2(ctx, repoDir)
	if err != nil {
		return nil
	}
	status, err := ParsePorcelainV2Status(raw)
	if err != nil {
		return nil
	}

	// Collect all changed file paths.
	var allFiles []string
	allFiles = append(allFiles, status.Files.Staged...)
	allFiles = append(allFiles, status.Files.Unstaged...)
	allFiles = append(allFiles, status.Files.Untracked...)

	// Deduplicate and find files not in traced set.
	seen := make(map[string]struct{})
	var untraced []string
	for _, f := range allFiles {
		if _, dup := seen[f]; dup {
			continue
		}
		seen[f] = struct{}{}
		if _, traced := tracedSet[f]; !traced {
			untraced = append(untraced, f)
			if len(untraced) >= limit {
				break
			}
		}
	}
	return untraced
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

	// Check if all checks failed or were skipped.
	job, _ := s.reviewJobStore.Get(jobID)
	if job != nil {
		allFailed := true
		for _, status := range job.Checks {
			if status == CheckCompleted {
				allFailed = false
				break
			}
		}
		if allFailed {
			s.reviewJobStore.Fail(jobID, "all checks failed or were skipped")
			return
		}
	}

	// Build final summary after at least one check succeeded.
	dc := s.reviewJobStore.DetailCount(jobID)
	th := s.reviewJobStore.Thresholds(jobID)
	summary := s.buildReviewSummary(ctx, repoID, repoDir, scenarioName, dc, th)
	s.reviewJobStore.Complete(jobID, summary)
}
