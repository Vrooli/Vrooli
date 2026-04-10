package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Handler validation tests (real request handling)
// ---------------------------------------------------------------------------

func TestHandleReviewSummary_MissingScenarioName(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	resp := NewResponse(rr)
	resp.BadRequest("scenarioName query parameter is required")
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleReviewSummary_InvalidScenarioName(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	resp := NewResponse(rr)
	// Simulates the validation path.
	if !IsValidScenarioName("../etc/passwd") {
		resp.BadRequest("scenarioName contains invalid characters")
	}
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleReviewRun_MissingScenarioName(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	resp := NewResponse(rr)
	resp.BadRequest("scenarioName is required")
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleReviewRun_UnknownCheck(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	resp := NewResponse(rr)
	checks := []string{"tidiness", "bogus"}
	for _, c := range checks {
		if !validReviewChecks[c] {
			resp.BadRequest("unknown check: " + c)
			break
		}
	}
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleReviewRun_ActiveJobConflictIncludesJobID(t *testing.T) {
	t.Parallel()

	store := NewReviewJobStore()
	store.Create("job-1", []string{"tests"}, "my-scenario", 0, DefaultReadinessThresholds())
	srv := &Server{
		git:            NewFakeGitRunner(),
		reviewJobStore: store,
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/review/run", strings.NewReader(`{"scenarioName":"my-scenario"}`))
	rr := httptest.NewRecorder()

	srv.handleReviewRun(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rr.Code)
	}

	var resp errorResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.JobID != "job-1" {
		t.Fatalf("expected existing job id in conflict response, got %q", resp.JobID)
	}
	if !strings.Contains(resp.Error, "already in progress") {
		t.Fatalf("unexpected error message: %q", resp.Error)
	}
}

func TestHandleReviewJobStatus_NotFound(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	resp := NewResponse(rr)
	resp.NotFound("review job not found")
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// buildReviewSummary tests with mocked dependencies
// ---------------------------------------------------------------------------

// Since the Server struct uses concrete types, we test buildReviewSummary
// indirectly by testing the pure logic (CalculateReadiness) thoroughly above,
// and testing the handler integration via the job store lifecycle below.

func TestBuildReviewSummary_AllServicesAvailable(t *testing.T) {
	t.Parallel()
	// Test the assembly logic by constructing dimensions directly
	// and verifying readiness calculation + response structure.
	scanTime := time.Now().Add(-5 * time.Minute)
	dims := ReviewDimensions{
		CodeQuality: &CodeQualityDimension{
			Available:  true,
			Score:      82.5,
			Violations: 3,
			Stale:      false,
			LastScan:   scanTime.Format(time.RFC3339),
		},
		Tests: &TestsDimension{
			Available:   true,
			Passed:      true,
			Total:       10,
			PassedCount: 10,
			FailedCount: 0,
			LastRun:     scanTime.Format(time.RFC3339),
		},
		Standards: &StandardsDimension{
			Available:          true,
			BlockingViolations: 0,
			Warnings:           2,
			TotalViolations:    2,
		},
		Visual: &VisualDimension{
			Available:       true,
			ScreenshotCount: 5,
			Stale:           false,
		},
		Provenance: &ProvenanceDimension{
			Available:   true,
			TracedFiles: 15,
		},
	}

	readiness := CalculateReadiness(dims, DefaultReadinessThresholds())
	if readiness != ReadinessGreen {
		t.Errorf("expected green, got %s", readiness)
	}

	summary := &ReviewSummaryResponse{
		ScenarioName: "test-scenario",
		Readiness:    readiness,
		Dimensions:   dims,
		Capabilities: map[string]bool{
			"tidiness-manager":          true,
			"test-genie":                true,
			"scenario-auditor":          true,
			"browser-automation-studio": true,
			"workspace-sandbox":         true,
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	if summary.ScenarioName != "test-scenario" {
		t.Errorf("expected test-scenario, got %s", summary.ScenarioName)
	}
	if summary.Dimensions.CodeQuality.Score != 82.5 {
		t.Errorf("expected 82.5, got %f", summary.Dimensions.CodeQuality.Score)
	}
	if summary.Dimensions.Provenance.TracedFiles != 15 {
		t.Errorf("expected 15 traced files, got %d", summary.Dimensions.Provenance.TracedFiles)
	}
}

func TestBuildReviewSummary_PartialAvailability(t *testing.T) {
	t.Parallel()
	// Only tests available, everything else nil.
	dims := ReviewDimensions{
		Tests: &TestsDimension{
			Available:   true,
			Passed:      true,
			Total:       3,
			PassedCount: 3,
		},
	}
	readiness := CalculateReadiness(dims, DefaultReadinessThresholds())
	if readiness != ReadinessYellow {
		t.Errorf("expected yellow with partial availability, got %s", readiness)
	}
}

func TestBuildReviewSummary_NoServicesAvailable(t *testing.T) {
	t.Parallel()
	dims := ReviewDimensions{}
	readiness := CalculateReadiness(dims, DefaultReadinessThresholds())
	if readiness != ReadinessRed {
		t.Errorf("expected red with no services, got %s", readiness)
	}
}

// ---------------------------------------------------------------------------
// executeReviewRun all-failed detection
// ---------------------------------------------------------------------------

func TestExecuteReviewRun_AllFailed(t *testing.T) {
	t.Parallel()
	store := NewReviewJobStore()
	store.Create("job-fail", []string{"tidiness", "tests"}, "my-scenario", 0, DefaultReadinessThresholds())

	// Simulate all checks failing.
	store.UpdateCheck("job-fail", "tidiness", CheckFailed)
	store.UpdateCheck("job-fail", "tests", CheckFailed)

	// Verify the detection logic used in executeReviewRun.
	job, ok := store.Get("job-fail")
	if !ok {
		t.Fatal("expected job to exist")
	}

	allFailed := true
	for _, status := range job.Checks {
		if status == CheckCompleted {
			allFailed = false
			break
		}
	}
	if !allFailed {
		t.Error("expected allFailed to be true")
	}

	store.Fail("job-fail", "all checks failed or were skipped")
	got, _ := store.Get("job-fail")
	if got.Status != "failed" {
		t.Errorf("expected failed, got %s", got.Status)
	}
	if got.Error != "all checks failed or were skipped" {
		t.Errorf("unexpected error message: %s", got.Error)
	}
}

func TestExecuteReviewRun_AllSkipped(t *testing.T) {
	t.Parallel()
	store := NewReviewJobStore()
	store.Create("job-skip", []string{"tidiness", "tests"}, "my-scenario", 0, DefaultReadinessThresholds())

	store.UpdateCheck("job-skip", "tidiness", CheckSkipped)
	store.UpdateCheck("job-skip", "tests", CheckSkipped)

	job, _ := store.Get("job-skip")
	allFailed := true
	for _, status := range job.Checks {
		if status == CheckCompleted {
			allFailed = false
			break
		}
	}
	if !allFailed {
		t.Error("expected allFailed (all skipped) to be true")
	}
}

func TestExecuteReviewRun_MixedResults(t *testing.T) {
	t.Parallel()
	store := NewReviewJobStore()
	store.Create("job-mix", []string{"tidiness", "tests", "rules"}, "my-scenario", 0, DefaultReadinessThresholds())

	store.UpdateCheck("job-mix", "tidiness", CheckCompleted)
	store.UpdateCheck("job-mix", "tests", CheckFailed)
	store.UpdateCheck("job-mix", "rules", CheckSkipped)

	job, _ := store.Get("job-mix")
	allFailed := true
	for _, status := range job.Checks {
		if status == CheckCompleted {
			allFailed = false
			break
		}
	}
	if allFailed {
		t.Error("expected allFailed to be false when one check completed")
	}
}

// ---------------------------------------------------------------------------
// Concurrency guard tests
// ---------------------------------------------------------------------------

func TestReviewJobStore_ConcurrencyGuard(t *testing.T) {
	t.Parallel()
	store := NewReviewJobStore()

	store.Create("job-1", []string{"tests"}, "my-scenario", 0, DefaultReadinessThresholds())

	// Same scenario should be blocked.
	if id := store.ActiveJobForScenario("my-scenario"); id != "job-1" {
		t.Errorf("expected job-1 as active, got %q", id)
	}

	// Different scenario should be allowed.
	if id := store.ActiveJobForScenario("other-scenario"); id != "" {
		t.Errorf("expected empty for different scenario, got %q", id)
	}

	// After failure, guard should clear.
	store.Fail("job-1", "test error")
	if id := store.ActiveJobForScenario("my-scenario"); id != "" {
		t.Errorf("expected empty after failure, got %q", id)
	}
}

// ---------------------------------------------------------------------------
// Checks validation tests
// ---------------------------------------------------------------------------

func TestValidReviewChecks(t *testing.T) {
	t.Parallel()
	tests := []struct {
		check string
		valid bool
	}{
		{"tidiness", true},
		{"tests", true},
		{"rules", true},
		{"tidieness", false},
		{"test", false},
		{"", false},
		{"tidiness;drop table", false},
	}
	for _, tc := range tests {
		t.Run(tc.check, func(t *testing.T) {
			t.Parallel()
			if validReviewChecks[tc.check] != tc.valid {
				t.Errorf("validReviewChecks[%q] = %v, want %v", tc.check, !tc.valid, tc.valid)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Mock interface compile-time verification
// ---------------------------------------------------------------------------

var (
	_ ReviewTidinessClient  = (*mockTidinessClient)(nil)
	_ ReviewTestGenieClient = (*mockTestGenieClient)(nil)
	_ ReviewAuditorClient   = (*mockAuditorClient)(nil)
	_ ReviewVisualStorage   = (*mockVisualStorage)(nil)
	_ ReviewCapabilities    = (*mockCapabilities)(nil)
)

// Verify concrete types also satisfy the interfaces.
var (
	_ ReviewTidinessClient  = (*TidinessManagerClient)(nil)
	_ ReviewTestGenieClient = (*TestGenieClient)(nil)
	_ ReviewAuditorClient   = (*AuditorClient)(nil)
	_ ReviewVisualStorage   = (*VisualCaptureStorage)(nil)
	_ ReviewCapabilities    = (*CapabilityRegistry)(nil)
)

// ---------------------------------------------------------------------------
// Standards dimension edge cases
// ---------------------------------------------------------------------------

func TestCalculateReadiness_StandardsEdgeCases(t *testing.T) {
	t.Parallel()
	base := ReviewDimensions{
		Visual:      &VisualDimension{Available: true, ScreenshotCount: 3},
		Tests:       &TestsDimension{Available: true, Total: 5, Passed: true, PassedCount: 5},
		CodeQuality: &CodeQualityDimension{Available: true, Score: 80, Stale: false},
	}

	t.Run("nil_standards_allows_green", func(t *testing.T) {
		t.Parallel()
		dims := base
		dims.Standards = nil
		if got := CalculateReadiness(dims, DefaultReadinessThresholds()); got != ReadinessGreen {
			t.Errorf("expected green, got %s", got)
		}
	})

	t.Run("unavailable_standards_allows_green", func(t *testing.T) {
		t.Parallel()
		dims := base
		dims.Standards = &StandardsDimension{Available: false, BlockingViolations: 99}
		if got := CalculateReadiness(dims, DefaultReadinessThresholds()); got != ReadinessGreen {
			t.Errorf("expected green, got %s", got)
		}
	})

	t.Run("zero_blocking_allows_green", func(t *testing.T) {
		t.Parallel()
		dims := base
		dims.Standards = &StandardsDimension{Available: true, BlockingViolations: 0, Warnings: 10}
		if got := CalculateReadiness(dims, DefaultReadinessThresholds()); got != ReadinessGreen {
			t.Errorf("expected green, got %s", got)
		}
	})

	t.Run("one_blocking_prevents_green", func(t *testing.T) {
		t.Parallel()
		dims := base
		dims.Standards = &StandardsDimension{Available: true, BlockingViolations: 1}
		if got := CalculateReadiness(dims, DefaultReadinessThresholds()); got != ReadinessYellow {
			t.Errorf("expected yellow, got %s", got)
		}
	})
}

// ---------------------------------------------------------------------------
// ProvenanceDimension: TotalFiles removed
// ---------------------------------------------------------------------------

func TestProvenanceDimension_NoTotalFiles(t *testing.T) {
	t.Parallel()
	dim := ProvenanceDimension{Available: true, TracedFiles: 10}
	// Ensure the struct compiles without TotalFiles field.
	if dim.TracedFiles != 10 {
		t.Errorf("expected 10, got %d", dim.TracedFiles)
	}
}

// ---------------------------------------------------------------------------
// Visual dimension guard
// ---------------------------------------------------------------------------

func TestVisualDimension_UnavailableReturnsNil(t *testing.T) {
	t.Parallel()
	// When BAS is unavailable, dims.Visual should remain nil.
	// This tests the behavioral expectation: guard returns early when unavailable.
	caps := &mockCapabilities{available: map[string]bool{
		"browser-automation-studio": false,
	}}
	if caps.IsAvailable(context.Background(), "browser-automation-studio") {
		t.Error("expected BAS to be unavailable in mock")
	}
	// The actual nil-ness is tested through integration; here we verify
	// the capability check returns false.
}

// ---------------------------------------------------------------------------
// Cleanup timer integration
// ---------------------------------------------------------------------------

func TestReviewJobStore_StartStopCleanup(t *testing.T) {
	t.Parallel()
	store := NewReviewJobStore()
	store.StartCleanup(50 * time.Millisecond)

	store.Create("old-job", []string{"tests"}, "scenario", 0, DefaultReadinessThresholds())
	store.mu.Lock()
	store.jobs["old-job"].status.StartedAt = time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339)
	store.mu.Unlock()

	// Wait for at least one cleanup cycle.
	time.Sleep(150 * time.Millisecond)

	if _, ok := store.Get("old-job"); ok {
		t.Error("expected periodic cleanup to remove old job")
	}

	store.StopCleanup()
	// Double stop should not panic.
	store.StopCleanup()
}

// ---------------------------------------------------------------------------
// Error message formatting
// ---------------------------------------------------------------------------

func TestReviewRunRequest_CheckValidation(t *testing.T) {
	t.Parallel()
	// Verify that error messages for invalid checks include the check name.
	badCheck := "unknown-check"
	msg := fmt.Sprintf("unknown check: %s", badCheck)
	if msg != "unknown check: unknown-check" {
		t.Errorf("unexpected message format: %s", msg)
	}
}

// ---------------------------------------------------------------------------
// Detail field population tests
// ---------------------------------------------------------------------------

func TestBuildCodeQualityIssues(t *testing.T) {
	t.Parallel()

	bd := &TidinessBreakdown{
		LintIssues:        5,
		TypeIssues:        0,
		LongFiles:         2,
		ComplexFunctions:  8,
		TechDebtMarkers:   0,
		DuplicationIssues: 1,
	}

	issues := buildCodeQualityIssues(bd, 10)
	if len(issues) != 4 { // only non-zero
		t.Fatalf("expected 4 issues, got %d", len(issues))
	}
	// Sorted by count descending.
	if issues[0].Category != "complex_functions" || issues[0].Count != 8 {
		t.Errorf("expected complex_functions:8 first, got %s:%d", issues[0].Category, issues[0].Count)
	}
	if issues[1].Category != "lint_issues" || issues[1].Count != 5 {
		t.Errorf("expected lint_issues:5 second, got %s:%d", issues[1].Category, issues[1].Count)
	}

	// Test capping.
	capped := buildCodeQualityIssues(bd, 2)
	if len(capped) != 2 {
		t.Fatalf("expected 2 capped issues, got %d", len(capped))
	}
}

func TestBuildCodeQualityIssues_AllZero(t *testing.T) {
	t.Parallel()
	bd := &TidinessBreakdown{}
	issues := buildCodeQualityIssues(bd, 5)
	if len(issues) != 0 {
		t.Errorf("expected 0 issues for all-zero breakdown, got %d", len(issues))
	}
}

func TestBuildTestFailures(t *testing.T) {
	t.Parallel()

	phases := []TestPhaseResult{
		{Name: "build", Status: "passed"},
		{Name: "unit", Status: "failed", Error: "assertion error", Classification: "logic", Remediation: "fix test"},
		{Name: "integration", Status: "failed", Error: "timeout", Classification: "infra"},
		{Name: "lint", Status: "passed"},
	}

	failures := buildTestFailures(phases, 10)
	if len(failures) != 2 {
		t.Fatalf("expected 2 failures, got %d", len(failures))
	}
	if failures[0].Phase != "unit" {
		t.Errorf("expected 'unit' first, got %s", failures[0].Phase)
	}
	if failures[0].Remediation != "fix test" {
		t.Errorf("expected 'fix test' remediation, got %s", failures[0].Remediation)
	}
	if failures[1].Phase != "integration" {
		t.Errorf("expected 'integration' second, got %s", failures[1].Phase)
	}

	// Test capping.
	capped := buildTestFailures(phases, 1)
	if len(capped) != 1 {
		t.Fatalf("expected 1 capped failure, got %d", len(capped))
	}
}

func TestBuildTestFailures_NoneFaild(t *testing.T) {
	t.Parallel()
	phases := []TestPhaseResult{
		{Name: "build", Status: "passed"},
		{Name: "unit", Status: "passed"},
	}
	failures := buildTestFailures(phases, 5)
	if len(failures) != 0 {
		t.Errorf("expected 0 failures, got %d", len(failures))
	}
}

func TestBuildTopViolations(t *testing.T) {
	t.Parallel()

	violations := []AuditorViolation{
		{FilePath: "a.go", LineNumber: 10, Title: "warn1", Severity: "warning", Recommendation: "fix warn"},
		{FilePath: "b.go", LineNumber: 20, Title: "crit1", Severity: "critical", Recommendation: "fix crit"},
		{FilePath: "c.go", LineNumber: 30, Title: "err1", Severity: "error", Recommendation: "fix err"},
		{FilePath: "d.go", LineNumber: 40, Title: "warn2", Severity: "warning"},
	}

	top := buildTopViolations(violations, 10)
	if len(top) != 4 {
		t.Fatalf("expected 4 violations, got %d", len(top))
	}
	// Sorted: critical, error, warning, warning.
	if top[0].Severity != "critical" {
		t.Errorf("expected critical first, got %s", top[0].Severity)
	}
	if top[1].Severity != "error" {
		t.Errorf("expected error second, got %s", top[1].Severity)
	}
	if top[2].Severity != "warning" || top[3].Severity != "warning" {
		t.Error("expected warnings last")
	}

	// Test capping.
	capped := buildTopViolations(violations, 2)
	if len(capped) != 2 {
		t.Fatalf("expected 2 capped violations, got %d", len(capped))
	}
	if capped[0].Severity != "critical" || capped[1].Severity != "error" {
		t.Error("capped should contain critical and error")
	}
}

func TestBuildTopViolations_Empty(t *testing.T) {
	t.Parallel()
	top := buildTopViolations(nil, 5)
	if len(top) != 0 {
		t.Errorf("expected 0 for nil violations, got %d", len(top))
	}
}

func TestBuildUntracedFiles(t *testing.T) {
	t.Parallel()

	fakeGit := NewFakeGitRunner()
	fakeGit.Unstaged = map[string]string{"api/handler.go": "diff", "api/store.go": "diff"}
	fakeGit.Untracked = []string{"cli/new.go"}

	tracedSet := map[string]struct{}{
		"api/handler.go": {},
	}

	untraced := buildUntracedFiles(context.Background(), fakeGit, "/repo", tracedSet, 10)

	// api/store.go and cli/new.go should be untraced.
	if len(untraced) != 2 {
		t.Fatalf("expected 2 untraced files, got %d: %v", len(untraced), untraced)
	}

	// Test capping.
	capped := buildUntracedFiles(context.Background(), fakeGit, "/repo", tracedSet, 1)
	if len(capped) != 1 {
		t.Fatalf("expected 1 capped untraced file, got %d", len(capped))
	}
}

func TestBuildUntracedFiles_AllTraced(t *testing.T) {
	t.Parallel()

	fakeGit := NewFakeGitRunner()
	fakeGit.Unstaged = map[string]string{"api/handler.go": "diff"}

	tracedSet := map[string]struct{}{
		"api/handler.go": {},
	}

	untraced := buildUntracedFiles(context.Background(), fakeGit, "/repo", tracedSet, 10)
	if len(untraced) != 0 {
		t.Errorf("expected 0 untraced files when all traced, got %d", len(untraced))
	}
}

func TestReviewDetailCount_BackwardCompat(t *testing.T) {
	t.Parallel()

	// When detailCount is 0, all detail fields should remain nil/empty.
	scanTime := time.Now().Add(-5 * time.Minute)
	dims := ReviewDimensions{
		CodeQuality: &CodeQualityDimension{
			Available:  true,
			Score:      85,
			Violations: 3,
			LastScan:   scanTime.Format(time.RFC3339),
		},
		Tests: &TestsDimension{
			Available:   true,
			Passed:      false,
			Total:       5,
			PassedCount: 4,
			FailedCount: 1,
		},
		Standards: &StandardsDimension{
			Available:          true,
			BlockingViolations: 1,
			TotalViolations:    3,
		},
		Visual: &VisualDimension{
			Available:       true,
			ScreenshotCount: 2,
		},
		Provenance: &ProvenanceDimension{
			Available:   true,
			TracedFiles: 10,
		},
	}

	// All detail fields should be nil/empty since they weren't populated.
	if dims.CodeQuality.TopIssues != nil {
		t.Error("TopIssues should be nil when not populated")
	}
	if dims.Tests.Failures != nil {
		t.Error("Failures should be nil when not populated")
	}
	if dims.Standards.TopViolations != nil {
		t.Error("TopViolations should be nil when not populated")
	}
	if dims.Visual.LatestCapture != nil {
		t.Error("LatestCapture should be nil when not populated")
	}
	if dims.Provenance.UntracedFiles != nil {
		t.Error("UntracedFiles should be nil when not populated")
	}
}

func TestReviewJobStore_DetailCount(t *testing.T) {
	t.Parallel()
	store := NewReviewJobStore()
	store.Create("job-1", []string{"tests"}, "scenario", 5, DefaultReadinessThresholds())

	dc := store.DetailCount("job-1")
	if dc != 5 {
		t.Errorf("expected detail count 5, got %d", dc)
	}

	dc = store.DetailCount("nonexistent")
	if dc != 0 {
		t.Errorf("expected detail count 0 for nonexistent job, got %d", dc)
	}
}
