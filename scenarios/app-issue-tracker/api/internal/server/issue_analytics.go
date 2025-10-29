package server

import (
	"sort"
	"strings"
	"time"
)

type appIssueStats struct {
	total int
	open  int
}

type issueAnalytics struct {
	totalIssues       int
	openIssues        int
	inProgressIssues  int
	completedToday    int
	resolvedCount     int
	totalResolution   time.Duration
	appStats          map[string]appIssueStats
	failureReasons    map[string]int
	manualFailures    int
	autoFailures      int
	completedIssues   int
	totalCompletedAge time.Duration
}

func newIssueAnalytics(issues []Issue, now time.Time) issueAnalytics {
	analytics := issueAnalytics{
		appStats:       make(map[string]appIssueStats),
		failureReasons: make(map[string]int),
	}

	today := now.UTC().Format("2006-01-02")

	for _, issue := range issues {
		analytics.totalIssues++

		switch strings.TrimSpace(issue.Status) {
		case "open":
			analytics.openIssues++
		case "active":
			analytics.inProgressIssues++
		case "completed":
			analytics.completedIssues++
			if strings.HasPrefix(strings.TrimSpace(issue.Metadata.ResolvedAt), today) {
				analytics.completedToday++
			}
			// Track completed issue age
			createdAt := strings.TrimSpace(issue.Metadata.CreatedAt)
			if createdAt != "" {
				if createdTime, err := time.Parse(time.RFC3339, createdAt); err == nil {
					age := now.Sub(createdTime)
					if age > 0 {
						analytics.totalCompletedAge += age
					}
				}
			}
		case "failed":
			if issue.ManualReview.MarkedAsFailed {
				analytics.manualFailures++
				if reason := strings.TrimSpace(issue.ManualReview.FailureReason); reason != "" {
					analytics.failureReasons[reason]++
				} else {
					analytics.failureReasons["unspecified"]++
				}
			} else {
				analytics.autoFailures++
				// Track auto-failure reason from metadata if available
				if issue.Metadata.Extra != nil {
					if reason := strings.TrimSpace(issue.Metadata.Extra["agent_last_error"]); reason != "" {
						// Simplify error messages for analytics
						simplified := simplifyErrorReason(reason)
						analytics.failureReasons[simplified]++
					}
				}
			}
		}

		stats := analytics.appStats[issue.AppID]
		stats.total++
		if issue.Status == "open" || issue.Status == "active" {
			stats.open++
		}
		analytics.appStats[issue.AppID] = stats

		createdAt := strings.TrimSpace(issue.Metadata.CreatedAt)
		resolvedAt := strings.TrimSpace(issue.Metadata.ResolvedAt)
		if createdAt != "" && resolvedAt != "" {
			createdTime, createdErr := time.Parse(time.RFC3339, createdAt)
			resolvedTime, resolvedErr := time.Parse(time.RFC3339, resolvedAt)
			if createdErr == nil && resolvedErr == nil && resolvedTime.After(createdTime) {
				analytics.totalResolution += resolvedTime.Sub(createdTime)
				analytics.resolvedCount++
			}
		}
	}

	return analytics
}

func (a issueAnalytics) avgResolutionHours() float64 {
	if a.resolvedCount == 0 {
		return 0
	}
	return a.totalResolution.Hours() / float64(a.resolvedCount)
}

type issueAppCount struct {
	AppName    string `json:"app_name"`
	IssueCount int    `json:"issue_count"`
}

func (a issueAnalytics) topApps(limit int) []issueAppCount {
	counts := make([]issueAppCount, 0, len(a.appStats))
	for appID, stats := range a.appStats {
		counts = append(counts, issueAppCount{AppName: appID, IssueCount: stats.total})
	}

	sort.Slice(counts, func(i, j int) bool {
		if counts[i].IssueCount == counts[j].IssueCount {
			return counts[i].AppName < counts[j].AppName
		}
		return counts[i].IssueCount > counts[j].IssueCount
	})

	if limit > 0 && len(counts) > limit {
		counts = counts[:limit]
	}

	return counts
}

func (a issueAnalytics) appSummaries() map[string]appIssueStats {
	return a.appStats
}

func (a issueAnalytics) totals() (total, open, inProgress, completedToday int, avgHours float64) {
	return a.totalIssues, a.openIssues, a.inProgressIssues, a.completedToday, a.avgResolutionHours()
}

func (a issueAnalytics) avgCompletedAgeHours() float64 {
	if a.completedIssues == 0 {
		return 0
	}
	return a.totalCompletedAge.Hours() / float64(a.completedIssues)
}

func (a issueAnalytics) failureMetrics() (manual, auto int, reasons map[string]int) {
	return a.manualFailures, a.autoFailures, a.failureReasons
}

func simplifyErrorReason(fullError string) string {
	lower := strings.ToLower(fullError)

	if strings.Contains(lower, "timeout") || strings.Contains(lower, "timed out") {
		return "timeout"
	}
	if strings.Contains(lower, "max turns") || strings.Contains(lower, "turn limit") {
		return "max_turns_reached"
	}
	if strings.Contains(lower, "parsing") || strings.Contains(lower, "parse error") {
		return "parsing_error"
	}
	if strings.Contains(lower, "rate limit") {
		return "rate_limited"
	}
	if strings.Contains(lower, "network") || strings.Contains(lower, "connection") {
		return "network_error"
	}
	if strings.Contains(lower, "permission") || strings.Contains(lower, "access denied") {
		return "permission_error"
	}

	return "other"
}
