package main

import (
	"context"
	"sort"
	"time"
)

// fetchCodeQualityDimension queries the tidiness-manager for code quality data.
func (s *Server) fetchCodeQualityDimension(ctx context.Context, scenarioName string, detailCount int) (*CodeQualityDimension, bool) {
	available := s.capabilities.IsAvailable(ctx, "tidiness-manager")
	if !available {
		return nil, false
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
	return dim, true
}

// fetchTestsDimension queries test-genie for test execution data.
func (s *Server) fetchTestsDimension(ctx context.Context, scenarioName string, detailCount int) (*TestsDimension, bool) {
	available := s.capabilities.IsAvailable(ctx, "test-genie")
	if !available {
		return nil, false
	}

	dim := &TestsDimension{Available: true}
	list, err := s.testGenieClient.ListExecutions(ctx, scenarioName, 1)
	if err != nil || list == nil || len(list.Items) == 0 {
		return dim, true
	}

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
	return dim, true
}

// fetchStandardsDimension queries scenario-auditor for standards violations.
func (s *Server) fetchStandardsDimension(ctx context.Context, scenarioName string, detailCount int) (*StandardsDimension, bool) {
	available := s.capabilities.IsAvailable(ctx, "scenario-auditor")
	if !available {
		return nil, false
	}

	dim := &StandardsDimension{Available: true}
	violations, err := s.auditorClient.GetViolations(ctx, scenarioName)
	if err != nil || violations == nil {
		return dim, true
	}

	for _, v := range violations.Violations {
		dim.TotalViolations++
		switch v.Severity {
		case "critical", "error":
			dim.BlockingViolations++
		default:
			dim.Warnings++
		}
	}
	if detailCount > 0 {
		dim.TopViolations = buildTopViolations(violations.Violations, detailCount)
	}
	return dim, true
}

// fetchVisualDimension queries local storage for visual capture snapshots.
func (s *Server) fetchVisualDimension(ctx context.Context, repoID int64, scenarioName string, detailCount int) (*VisualDimension, bool) {
	available := s.capabilities.IsAvailable(ctx, "browser-automation-studio")
	if !available {
		return nil, false
	}

	dim := &VisualDimension{Available: true}
	snapshots, err := s.visualCaptureStorage.ListSnapshotSets(repoID, scenarioName)
	if err != nil {
		return dim, true
	}

	var latestTime time.Time
	for _, snap := range snapshots {
		if snap.Status != "complete" {
			continue
		}
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
	return dim, true
}

// fetchProvenanceDimension queries workspace-sandbox for provenance data.
func (s *Server) fetchProvenanceDimension(ctx context.Context, repoDir string, detailCount int) (*ProvenanceDimension, bool) {
	available := s.capabilities.IsAvailable(ctx, "workspace-sandbox")
	if !available {
		return nil, false
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
	return dim, true
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
