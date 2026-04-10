package main

import (
	"context"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Mock implementations for review handler dependencies
// ---------------------------------------------------------------------------

type mockTidinessClient struct {
	score     *TidinessScoreResponse
	scoreErr  error
	stale     *TidinessStalenessInfo
	staleErr  error
	scanRes   *TidinessLightScanResult
	scanErr   error
	scanCalls int
}

func (m *mockTidinessClient) GetTidinessScore(_ context.Context, _ string) (*TidinessScoreResponse, error) {
	return m.score, m.scoreErr
}

func (m *mockTidinessClient) GetStaleness(_ context.Context, _ string) (*TidinessStalenessInfo, error) {
	return m.stale, m.staleErr
}

func (m *mockTidinessClient) TriggerLightScan(_ context.Context, _ TidinessLightScanRequest) (*TidinessLightScanResult, error) {
	m.scanCalls++
	return m.scanRes, m.scanErr
}

type mockTestGenieClient struct {
	listRes   *TestExecutionListResponse
	listErr   error
	execRes   *TestExecutionResult
	execErr   error
	execCalls int
}

func (m *mockTestGenieClient) ListExecutions(_ context.Context, _ string, _ int) (*TestExecutionListResponse, error) {
	return m.listRes, m.listErr
}

func (m *mockTestGenieClient) ExecuteSuite(_ context.Context, _ TestExecutionRequest) (*TestExecutionResult, error) {
	m.execCalls++
	return m.execRes, m.execErr
}

type mockAuditorClient struct {
	violations    *AuditorViolationsResponse
	violationsErr error
	checkRes      *AuditorCheckJobResponse
	checkErr      error
	checkCalls    int
}

func (m *mockAuditorClient) GetViolations(_ context.Context, _ string) (*AuditorViolationsResponse, error) {
	return m.violations, m.violationsErr
}

func (m *mockAuditorClient) StartCheck(_ context.Context, _, _ string) (*AuditorCheckJobResponse, error) {
	m.checkCalls++
	return m.checkRes, m.checkErr
}

type mockVisualStorage struct {
	snapshots []SnapshotSetMeta
	err       error
}

func (m *mockVisualStorage) ListSnapshotSets(_ int64, _ string) ([]SnapshotSetMeta, error) {
	return m.snapshots, m.err
}

type mockCapabilities struct {
	available map[string]bool
}

func (m *mockCapabilities) IsAvailable(_ context.Context, id string) bool {
	return m.available[id]
}

// ---------------------------------------------------------------------------
// IsValidScenarioName tests
// ---------------------------------------------------------------------------

func TestIsValidScenarioName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"simple", "my-scenario", true},
		{"with_underscore", "my_scenario", true},
		{"alphanumeric", "scenario123", true},
		{"single_char", "a", true},
		{"empty", "", false},
		{"path_traversal", "../etc", false},
		{"slash", "foo/bar", false},
		{"dot_dot", "..", false},
		{"starts_with_hyphen", "-bad", false},
		{"spaces", "my scenario", false},
		{"special_chars", "sc@nario!", false},
		{"too_long", string(make([]byte, 129)), false},
		{"max_length", func() string {
			b := make([]byte, 128)
			for i := range b {
				b[i] = 'a'
			}
			return string(b)
		}(), true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := IsValidScenarioName(tc.input)
			if got != tc.want {
				t.Errorf("IsValidScenarioName(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// CalculateReadiness tests (expanded)
// ---------------------------------------------------------------------------

func TestCalculateReadiness(t *testing.T) {
	t.Parallel()
	scanTime := time.Now().Add(-5 * time.Minute)

	tests := []struct {
		name string
		dims ReviewDimensions
		want Readiness
	}{
		{
			name: "all_green",
			dims: ReviewDimensions{
				Visual:      &VisualDimension{Available: true, ScreenshotCount: 3},
				Tests:       &TestsDimension{Available: true, Total: 5, Passed: true, PassedCount: 5},
				CodeQuality: &CodeQualityDimension{Available: true, Score: 80, Stale: false, LastScan: scanTime.Format(time.RFC3339)},
			},
			want: ReadinessGreen,
		},
		{
			name: "green_with_standards_clean",
			dims: ReviewDimensions{
				Visual:      &VisualDimension{Available: true, ScreenshotCount: 3},
				Tests:       &TestsDimension{Available: true, Total: 5, Passed: true, PassedCount: 5},
				CodeQuality: &CodeQualityDimension{Available: true, Score: 80, Stale: false},
				Standards:   &StandardsDimension{Available: true, BlockingViolations: 0, Warnings: 2},
			},
			want: ReadinessGreen,
		},
		{
			name: "blocking_violations_prevents_green",
			dims: ReviewDimensions{
				Visual:      &VisualDimension{Available: true, ScreenshotCount: 3},
				Tests:       &TestsDimension{Available: true, Total: 5, Passed: true, PassedCount: 5},
				CodeQuality: &CodeQualityDimension{Available: true, Score: 80, Stale: false},
				Standards:   &StandardsDimension{Available: true, BlockingViolations: 1},
			},
			want: ReadinessYellow,
		},
		{
			name: "standards_unavailable_ignored",
			dims: ReviewDimensions{
				Visual:      &VisualDimension{Available: true, ScreenshotCount: 3},
				Tests:       &TestsDimension{Available: true, Total: 5, Passed: true, PassedCount: 5},
				CodeQuality: &CodeQualityDimension{Available: true, Score: 80, Stale: false},
				Standards:   &StandardsDimension{Available: false, BlockingViolations: 99},
			},
			want: ReadinessGreen,
		},
		{
			name: "yellow_only_screenshots",
			dims: ReviewDimensions{
				Visual: &VisualDimension{Available: true, ScreenshotCount: 1},
			},
			want: ReadinessYellow,
		},
		{
			name: "yellow_only_tests",
			dims: ReviewDimensions{
				Tests: &TestsDimension{Available: true, Total: 1, Passed: false, PassedCount: 0, FailedCount: 1},
			},
			want: ReadinessYellow,
		},
		{
			name: "yellow_only_quality",
			dims: ReviewDimensions{
				CodeQuality: &CodeQualityDimension{Available: true, Score: 70, Stale: false},
			},
			want: ReadinessYellow,
		},
		{
			name: "yellow_quality_at_threshold",
			dims: ReviewDimensions{
				CodeQuality: &CodeQualityDimension{Available: true, Score: 60, Stale: false},
			},
			want: ReadinessYellow,
		},
		{
			name: "red_quality_below_threshold",
			dims: ReviewDimensions{
				CodeQuality: &CodeQualityDimension{Available: true, Score: 59.9, Stale: false},
			},
			want: ReadinessRed,
		},
		{
			name: "red_quality_stale",
			dims: ReviewDimensions{
				CodeQuality: &CodeQualityDimension{Available: true, Score: 80, Stale: true},
			},
			want: ReadinessRed,
		},
		{
			name: "red_empty_dimensions",
			dims: ReviewDimensions{},
			want: ReadinessRed,
		},
		{
			name: "red_all_nil",
			dims: ReviewDimensions{
				Visual:      nil,
				Tests:       nil,
				CodeQuality: nil,
				Standards:   nil,
				Provenance:  nil,
			},
			want: ReadinessRed,
		},
		{
			name: "red_tests_exist_but_zero_total",
			dims: ReviewDimensions{
				Tests: &TestsDimension{Available: true, Total: 0},
			},
			want: ReadinessRed,
		},
		{
			name: "red_screenshots_zero_count",
			dims: ReviewDimensions{
				Visual: &VisualDimension{Available: true, ScreenshotCount: 0},
			},
			want: ReadinessRed,
		},
		{
			name: "yellow_tests_fail_but_present",
			dims: ReviewDimensions{
				Tests:       &TestsDimension{Available: true, Total: 2, Passed: false, PassedCount: 1, FailedCount: 1},
				CodeQuality: &CodeQualityDimension{Available: true, Score: 80, Stale: true},
			},
			want: ReadinessYellow,
		},
		{
			name: "green_not_reached_when_tests_fail",
			dims: ReviewDimensions{
				Visual:      &VisualDimension{Available: true, ScreenshotCount: 3},
				Tests:       &TestsDimension{Available: true, Total: 5, Passed: false, PassedCount: 3, FailedCount: 2},
				CodeQuality: &CodeQualityDimension{Available: true, Score: 80, Stale: false},
			},
			want: ReadinessYellow,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := CalculateReadiness(tc.dims, DefaultReadinessThresholds())
			if got != tc.want {
				t.Errorf("CalculateReadiness() = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestCalculateReadiness_CustomThresholds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		dims       ReviewDimensions
		thresholds ReadinessThresholds
		want       Readiness
	}{
		{
			name: "lower_quality_threshold_allows_green",
			dims: ReviewDimensions{
				Visual:      &VisualDimension{Available: true, ScreenshotCount: 3},
				Tests:       &TestsDimension{Available: true, Total: 5, Passed: true, PassedCount: 5},
				CodeQuality: &CodeQualityDimension{Available: true, Score: 45, Stale: false},
			},
			thresholds: ReadinessThresholds{
				CodeQualityMinScore: 40, TestMinPassRate: 1.0,
				RequireScreenshots: true, RequireTests: true, MaxWarnings: -1,
			},
			want: ReadinessGreen,
		},
		{
			name: "partial_test_pass_rate_allows_green",
			dims: ReviewDimensions{
				Visual:      &VisualDimension{Available: true, ScreenshotCount: 1},
				Tests:       &TestsDimension{Available: true, Total: 10, Passed: false, PassedCount: 8, FailedCount: 2},
				CodeQuality: &CodeQualityDimension{Available: true, Score: 80, Stale: false},
			},
			thresholds: ReadinessThresholds{
				CodeQualityMinScore: 60, TestMinPassRate: 0.8,
				RequireScreenshots: true, RequireTests: true, MaxWarnings: -1,
			},
			want: ReadinessGreen,
		},
		{
			name: "allow_blocking_violations",
			dims: ReviewDimensions{
				Visual:      &VisualDimension{Available: true, ScreenshotCount: 1},
				Tests:       &TestsDimension{Available: true, Total: 5, Passed: true, PassedCount: 5},
				CodeQuality: &CodeQualityDimension{Available: true, Score: 80, Stale: false},
				Standards:   &StandardsDimension{Available: true, BlockingViolations: 3},
			},
			thresholds: ReadinessThresholds{
				CodeQualityMinScore: 60, TestMinPassRate: 1.0,
				MaxBlockingViolations: 5, RequireScreenshots: true, RequireTests: true, MaxWarnings: -1,
			},
			want: ReadinessGreen,
		},
		{
			name: "screenshots_not_required",
			dims: ReviewDimensions{
				Tests:       &TestsDimension{Available: true, Total: 5, Passed: true, PassedCount: 5},
				CodeQuality: &CodeQualityDimension{Available: true, Score: 80, Stale: false},
			},
			thresholds: ReadinessThresholds{
				CodeQualityMinScore: 60, TestMinPassRate: 1.0,
				RequireScreenshots: false, RequireTests: true, MaxWarnings: -1,
			},
			want: ReadinessGreen,
		},
		{
			name: "tests_not_required",
			dims: ReviewDimensions{
				Visual:      &VisualDimension{Available: true, ScreenshotCount: 1},
				CodeQuality: &CodeQualityDimension{Available: true, Score: 80, Stale: false},
			},
			thresholds: ReadinessThresholds{
				CodeQualityMinScore: 60, TestMinPassRate: 1.0,
				RequireScreenshots: true, RequireTests: false, MaxWarnings: -1,
			},
			want: ReadinessGreen,
		},
		{
			name: "max_warnings_enforced",
			dims: ReviewDimensions{
				Visual:      &VisualDimension{Available: true, ScreenshotCount: 1},
				Tests:       &TestsDimension{Available: true, Total: 5, Passed: true, PassedCount: 5},
				CodeQuality: &CodeQualityDimension{Available: true, Score: 80, Stale: false},
				Standards:   &StandardsDimension{Available: true, BlockingViolations: 0, Warnings: 15},
			},
			thresholds: ReadinessThresholds{
				CodeQualityMinScore: 60, TestMinPassRate: 1.0,
				MaxBlockingViolations: 0, MaxWarnings: 10,
				RequireScreenshots: true, RequireTests: true,
			},
			want: ReadinessYellow,
		},
		{
			name: "pass_rate_below_threshold_prevents_green",
			dims: ReviewDimensions{
				Visual:      &VisualDimension{Available: true, ScreenshotCount: 1},
				Tests:       &TestsDimension{Available: true, Total: 10, Passed: false, PassedCount: 7, FailedCount: 3},
				CodeQuality: &CodeQualityDimension{Available: true, Score: 80, Stale: false},
			},
			thresholds: ReadinessThresholds{
				CodeQualityMinScore: 60, TestMinPassRate: 0.8,
				RequireScreenshots: true, RequireTests: true, MaxWarnings: -1,
			},
			want: ReadinessYellow,
		},
		{
			name: "max_warnings_unlimited_ignores_warnings",
			dims: ReviewDimensions{
				Visual:      &VisualDimension{Available: true, ScreenshotCount: 1},
				Tests:       &TestsDimension{Available: true, Total: 5, Passed: true, PassedCount: 5},
				CodeQuality: &CodeQualityDimension{Available: true, Score: 80, Stale: false},
				Standards:   &StandardsDimension{Available: true, BlockingViolations: 0, Warnings: 100},
			},
			thresholds: ReadinessThresholds{
				CodeQualityMinScore: 60, TestMinPassRate: 1.0,
				MaxBlockingViolations: 0, MaxWarnings: -1,
				RequireScreenshots: true, RequireTests: true,
			},
			want: ReadinessGreen,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := CalculateReadiness(tc.dims, tc.thresholds)
			if got != tc.want {
				t.Errorf("CalculateReadiness() = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestCalculateDimensionStatuses(t *testing.T) {
	t.Parallel()
	thresholds := DefaultReadinessThresholds()

	tests := []struct {
		name     string
		dims     ReviewDimensions
		wantKeys map[string]string
	}{
		{
			name: "all_green",
			dims: ReviewDimensions{
				CodeQuality: &CodeQualityDimension{Available: true, Score: 80, Stale: false, Violations: 0},
				Tests:       &TestsDimension{Available: true, Total: 5, PassedCount: 5},
				Standards:   &StandardsDimension{Available: true, BlockingViolations: 0, Warnings: 0},
				Visual:      &VisualDimension{Available: true, ScreenshotCount: 3, Stale: false},
				Provenance:  &ProvenanceDimension{Available: true, TracedFiles: 5},
			},
			wantKeys: map[string]string{
				"codeQuality": "green",
				"tests":       "green",
				"standards":   "green",
				"visual":      "green",
				"provenance":  "green",
			},
		},
		{
			name: "all_unavailable",
			dims: ReviewDimensions{},
			wantKeys: map[string]string{
				"codeQuality": "skipped",
				"tests":       "skipped",
				"standards":   "skipped",
				"visual":      "skipped",
				"provenance":  "skipped",
			},
		},
		{
			name: "code_quality_red",
			dims: ReviewDimensions{
				CodeQuality: &CodeQualityDimension{Available: true, Score: 30, Stale: false},
			},
			wantKeys: map[string]string{
				"codeQuality": "red",
			},
		},
		{
			name: "code_quality_yellow_violations",
			dims: ReviewDimensions{
				CodeQuality: &CodeQualityDimension{Available: true, Score: 80, Stale: false, Violations: 5},
			},
			wantKeys: map[string]string{
				"codeQuality": "yellow",
			},
		},
		{
			name: "code_quality_yellow_stale",
			dims: ReviewDimensions{
				CodeQuality: &CodeQualityDimension{Available: true, Score: 80, Stale: true},
			},
			wantKeys: map[string]string{
				"codeQuality": "yellow",
			},
		},
		{
			name: "tests_red_all_failed",
			dims: ReviewDimensions{
				Tests: &TestsDimension{Available: true, Total: 5, PassedCount: 0, FailedCount: 5},
			},
			wantKeys: map[string]string{
				"tests": "red",
			},
		},
		{
			name: "tests_yellow_partial",
			dims: ReviewDimensions{
				Tests: &TestsDimension{Available: true, Total: 5, PassedCount: 3, FailedCount: 2},
			},
			wantKeys: map[string]string{
				"tests": "yellow",
			},
		},
		{
			name: "standards_red_blocking",
			dims: ReviewDimensions{
				Standards: &StandardsDimension{Available: true, BlockingViolations: 2},
			},
			wantKeys: map[string]string{
				"standards": "red",
			},
		},
		{
			name: "standards_yellow_warnings",
			dims: ReviewDimensions{
				Standards: &StandardsDimension{Available: true, BlockingViolations: 0, Warnings: 5},
			},
			wantKeys: map[string]string{
				"standards": "yellow",
			},
		},
		{
			name: "visual_red_no_screenshots",
			dims: ReviewDimensions{
				Visual: &VisualDimension{Available: true, ScreenshotCount: 0},
			},
			wantKeys: map[string]string{
				"visual": "red",
			},
		},
		{
			name: "visual_yellow_stale",
			dims: ReviewDimensions{
				Visual: &VisualDimension{Available: true, ScreenshotCount: 3, Stale: true},
			},
			wantKeys: map[string]string{
				"visual": "yellow",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := CalculateDimensionStatuses(tc.dims, thresholds)
			for key, want := range tc.wantKeys {
				if got[key] != want {
					t.Errorf("dimension %s: got %s, want %s", key, got[key], want)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ReviewJobStore tests (expanded)
// ---------------------------------------------------------------------------

func TestReviewJobStore_Lifecycle(t *testing.T) {
	t.Parallel()
	store := NewReviewJobStore()

	t.Run("create", func(t *testing.T) {
		assertJobStoreCreate(t, store)
	})
	t.Run("get", func(t *testing.T) {
		assertJobStoreGet(t, store)
	})
	t.Run("update check", func(t *testing.T) {
		assertJobStoreUpdateCheck(t, store)
	})
	t.Run("complete", func(t *testing.T) {
		assertJobStoreComplete(t, store)
	})
	t.Run("fail", func(t *testing.T) {
		assertJobStoreFail(t, store)
	})
	t.Run("not found", func(t *testing.T) {
		_, ok := store.Get("nonexistent")
		if ok {
			t.Error("expected not found for nonexistent job")
		}
	})
}

func assertJobStoreCreate(t *testing.T, store *ReviewJobStore) {
	t.Helper()
	job := store.Create("job-1", []string{"tidiness", "tests"}, "test-scenario", 0, DefaultReadinessThresholds())
	if job.Status != "running" {
		t.Errorf("expected running, got %s", job.Status)
	}
	if job.Checks["tidiness"] != CheckPending {
		t.Errorf("expected pending, got %s", job.Checks["tidiness"])
	}
}

func assertJobStoreGet(t *testing.T, store *ReviewJobStore) {
	t.Helper()
	got, ok := store.Get("job-1")
	if !ok {
		t.Fatal("expected job to be found")
	}
	if got.JobID != "job-1" {
		t.Errorf("expected job-1, got %s", got.JobID)
	}
}

func assertJobStoreUpdateCheck(t *testing.T, store *ReviewJobStore) {
	t.Helper()
	store.UpdateCheck("job-1", "tidiness", CheckRunning)
	got, _ := store.Get("job-1")
	if got.Checks["tidiness"] != CheckRunning {
		t.Errorf("expected running, got %s", got.Checks["tidiness"])
	}
}

func assertJobStoreComplete(t *testing.T, store *ReviewJobStore) {
	t.Helper()
	summary := &ReviewSummaryResponse{ScenarioName: "test", Readiness: ReadinessGreen}
	store.Complete("job-1", summary)
	got, _ := store.Get("job-1")
	if got.Status != "completed" {
		t.Errorf("expected completed, got %s", got.Status)
	}
	if got.Summary == nil || got.Summary.Readiness != ReadinessGreen {
		t.Error("expected green readiness in summary")
	}
}

func assertJobStoreFail(t *testing.T, store *ReviewJobStore) {
	t.Helper()
	store.Create("job-2", []string{"rules"}, "test-scenario-2", 0, DefaultReadinessThresholds())
	store.Fail("job-2", "something went wrong")
	got, _ := store.Get("job-2")
	if got.Status != "failed" {
		t.Errorf("expected failed, got %s", got.Status)
	}
	if got.Error != "something went wrong" {
		t.Errorf("expected error message, got %s", got.Error)
	}
}

func TestReviewJobStore_ActiveJobForScenario(t *testing.T) {
	t.Parallel()
	store := NewReviewJobStore()

	// No active job initially.
	if id := store.ActiveJobForScenario("foo"); id != "" {
		t.Errorf("expected empty, got %s", id)
	}

	// Create a running job.
	store.Create("job-1", []string{"tests"}, "foo", 0, DefaultReadinessThresholds())
	if id := store.ActiveJobForScenario("foo"); id != "job-1" {
		t.Errorf("expected job-1, got %s", id)
	}

	// Different scenario → no conflict.
	if id := store.ActiveJobForScenario("bar"); id != "" {
		t.Errorf("expected empty for bar, got %s", id)
	}

	// Complete the job → no longer active.
	store.Complete("job-1", nil)
	if id := store.ActiveJobForScenario("foo"); id != "" {
		t.Errorf("expected empty after complete, got %s", id)
	}
}

func TestReviewJobStore_Cleanup(t *testing.T) {
	t.Parallel()
	store := NewReviewJobStore()

	// Create a job with a timestamp in the past.
	store.Create("old-job", []string{"tests"}, "scenario", 0, DefaultReadinessThresholds())
	// Manually backdate the entry.
	store.mu.Lock()
	store.jobs["old-job"].status.StartedAt = time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339)
	store.mu.Unlock()

	store.Create("recent-job", []string{"tests"}, "scenario2", 0, DefaultReadinessThresholds())

	store.Cleanup()

	if _, ok := store.Get("old-job"); ok {
		t.Error("expected old-job to be cleaned up")
	}
	if _, ok := store.Get("recent-job"); !ok {
		t.Error("expected recent-job to survive cleanup")
	}
}

func TestReviewJobStore_GetReturnsCopy(t *testing.T) {
	t.Parallel()
	store := NewReviewJobStore()
	store.Create("job-1", []string{"tests"}, "scenario", 0, DefaultReadinessThresholds())

	got1, _ := store.Get("job-1")
	got1.Checks["tests"] = CheckCompleted // mutate the copy

	got2, _ := store.Get("job-1")
	if got2.Checks["tests"] != CheckPending {
		t.Error("Get() returned a reference, not a copy — mutation leaked")
	}
}
