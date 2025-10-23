package server

import (
	"regexp"
	"strings"
	"sync"
	"time"

	"app-issue-tracker-api/internal/logging"
	"app-issue-tracker-api/internal/server/metadata"
	services "app-issue-tracker-api/internal/server/services"
)

type RateLimitStatus struct {
	RateLimited        bool   `json:"rate_limited"`
	RateLimitedCount   int    `json:"rate_limited_count"`
	RateLimitType      string `json:"rate_limit_type,omitempty"`
	RateLimitAgent     string `json:"rate_limit_agent,omitempty"`
	ResetTime          string `json:"reset_time,omitempty"`
	SecondsUntilReset  int64  `json:"seconds_until_reset"`
	WaitingIssuesCount int    `json:"waiting_issues_count"`
}

// RateLimitManager encapsulates rate-limit bookkeeping for investigations.
type RateLimitManager struct {
	server *Server
	now    func() time.Time

	snapshotMu     sync.Mutex
	cachedIssues   []Issue
	cacheFetchedAt time.Time
}

const rateLimitSnapshotTTL = time.Second

var isoTimestampPattern = regexp.MustCompile(`20\d{2}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:Z|[+-]\d{2}:\d{2})?`)

func NewRateLimitManager(server *Server, now func() time.Time) *RateLimitManager {
	if now == nil {
		now = time.Now
	}
	return &RateLimitManager{server: server, now: now}
}

func (rm *RateLimitManager) Handle(issueID, agentID string, result *ClaudeExecutionResult) bool {
	if result == nil || !strings.Contains(result.Error, "RATE_LIMIT") {
		return false
	}

	isRateLimit, resetTime := detectRateLimit(result.Output, rm.now())
	if !isRateLimit {
		return false
	}

	logging.LogWarn("Rate limit detected during investigation", "issue_id", issueID, "reset_time", resetTime)

	issue, issueDir, _, loadErr := rm.server.loadIssueWithStatus(issueID)
	if loadErr == nil {
		services.SetRateLimitMetadata(issue, agentID, resetTime)
		if err := rm.server.writeIssueMetadata(issueDir, issue); err != nil {
			logging.LogWarn("Failed to persist rate limit metadata", "issue_id", issueID, "error", err)
		} else {
			rm.invalidateCache()
		}
	}

	if err := rm.server.moveIssue(issueID, "failed"); err != nil {
		logging.LogWarn("Failed to move rate-limited issue to failed status", "issue_id", issueID, "error", err)
	} else {
		rm.server.hub.Publish(NewEvent(EventAgentFailed, AgentCompletedData{
			IssueID:   issueID,
			AgentID:   agentID,
			Success:   false,
			EndTime:   rm.now(),
			NewStatus: "failed",
		}))
		rm.invalidateCache()
	}

	return true
}

func (rm *RateLimitManager) Status() RateLimitStatus {
	openIssues, err := rm.listOpenIssues()
	if err != nil {
		logging.LogErrorErr("Failed to load open issues for rate limit status", err)
		openIssues = []Issue{}
	}

	var earliestReset time.Time
	rateLimitedCount := 0
	var rateLimitType string
	var rateLimitAgent string

	for _, issue := range openIssues {
		if issue.Metadata.Extra == nil {
			continue
		}

		resetTimeStr := issue.Metadata.Extra[metadata.RateLimitUntilKey]
		if resetTimeStr == "" {
			continue
		}

		resetTime, parseErr := time.Parse(time.RFC3339, resetTimeStr)
		if parseErr != nil {
			logging.LogWarn(
				"Clearing stale rate limit metadata",
				"issue_id", issue.ID,
				"raw_reset_value", resetTimeStr,
			)
			rm.clear(issue.ID)
			continue
		}

		if rm.now().Before(resetTime) {
			rateLimitedCount++
			if earliestReset.IsZero() || resetTime.Before(earliestReset) {
				earliestReset = resetTime
				rateLimitAgent = issue.Metadata.Extra[metadata.RateLimitAgentKey]
				rateLimitType = "api_rate_limit"
			}
			continue
		}

		logging.LogInfo("Rate limit window elapsed via status endpoint", "issue_id", issue.ID)
		rm.clear(issue.ID)
	}

	status := RateLimitStatus{
		RateLimited:        rateLimitedCount > 0,
		RateLimitedCount:   rateLimitedCount,
		RateLimitType:      rateLimitType,
		RateLimitAgent:     rateLimitAgent,
		WaitingIssuesCount: rateLimitedCount,
	}

	if rateLimitedCount > 0 {
		status.ResetTime = earliestReset.Format(time.RFC3339)
		secondsUntilReset := int64(time.Until(earliestReset).Seconds())
		if secondsUntilReset < 0 {
			secondsUntilReset = 0
		}
		status.SecondsUntilReset = secondsUntilReset
	}

	return status
}

func (rm *RateLimitManager) ClearExpired() {
	issues, err := rm.listOpenIssues()
	if err != nil {
		logging.LogErrorErr("Failed to inspect open issues for rate limit metadata", err)
		return
	}

	for _, issue := range issues {
		if issue.Metadata.Extra == nil {
			continue
		}

		resetTimeStr := issue.Metadata.Extra[metadata.RateLimitUntilKey]
		if resetTimeStr == "" {
			continue
		}

		resetTime, parseErr := time.Parse(time.RFC3339, resetTimeStr)
		if parseErr != nil {
			logging.LogWarn(
				"Clearing stale rate limit metadata",
				"issue_id", issue.ID,
				"raw_reset_value", resetTimeStr,
			)
			rm.clear(issue.ID)
			continue
		}

		if rm.now().Before(resetTime) {
			continue
		}

		logging.LogInfo("Rate limit expiry detected", "issue_id", issue.ID)
		rm.clear(issue.ID)
	}
}

func (rm *RateLimitManager) Clear(issueID string) {
	rm.clear(issueID)
}

func (rm *RateLimitManager) clear(issueID string) {
	loaded, issueDir, _, err := rm.server.loadIssueWithStatus(issueID)
	if err != nil {
		return
	}

	services.ClearRateLimitMetadata(loaded)

	if err := rm.server.writeIssueMetadata(issueDir, loaded); err != nil {
		logging.LogWarn("Failed to clear rate limit metadata", "issue_id", issueID, "error", err)
	}

	rm.invalidateCache()
}

func detectRateLimit(output string, now time.Time) (bool, string) {
	outputLower := strings.ToLower(output)

	if strings.Contains(outputLower, "rate limit") ||
		strings.Contains(outputLower, "rate_limit") ||
		strings.Contains(outputLower, "quota exceeded") ||
		strings.Contains(outputLower, "too many requests") ||
		strings.Contains(outputLower, "429") {

		resetTime := ""

		if match := isoTimestampPattern.FindString(output); match != "" {
			if _, err := time.Parse(time.RFC3339, match); err == nil {
				resetTime = match
			} else if len(match) >= 19 {
				if parsed, err := time.Parse("2006-01-02T15:04:05", match[:19]); err == nil {
					resetTime = parsed.UTC().Format(time.RFC3339)
				}
			}
		}

		if resetTime == "" {
			resetTime = now.Add(5 * time.Minute).Format(time.RFC3339)
		}

		return true, resetTime
	}

	return false, ""
}

func (rm *RateLimitManager) listOpenIssues() ([]Issue, error) {
	rm.snapshotMu.Lock()
	defer rm.snapshotMu.Unlock()

	if rm.cachedIssues != nil && rm.now().Sub(rm.cacheFetchedAt) < rateLimitSnapshotTTL {
		return cloneIssuesSlice(rm.cachedIssues), nil
	}

	issues, err := rm.server.loadIssuesFromFolder("open")
	if err != nil {
		return nil, err
	}

	rm.cachedIssues = cloneIssuesSlice(issues)
	rm.cacheFetchedAt = rm.now()
	return cloneIssuesSlice(rm.cachedIssues), nil
}

func (rm *RateLimitManager) invalidateCache() {
	rm.snapshotMu.Lock()
	defer rm.snapshotMu.Unlock()
	rm.cachedIssues = nil
	rm.cacheFetchedAt = time.Time{}
}

func cloneIssuesSlice(src []Issue) []Issue {
	if len(src) == 0 {
		return []Issue{}
	}
	return append([]Issue(nil), src...)
}
