package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"app-monitor-api/logger"
)

// =============================================================================
// Issue Tracker Integration
// =============================================================================

// ListScenarioIssues returns cached issue information for a scenario.
func (s *AppService) ListScenarioIssues(ctx context.Context, appID string) (*AppIssuesSummary, error) {
	id := strings.TrimSpace(appID)
	if id == "" {
		return nil, ErrAppIdentifierRequired
	}

	app, err := s.GetApp(ctx, id)
	if err != nil {
		return nil, err
	}

	scenarioName := strings.TrimSpace(app.ScenarioName)
	if scenarioName == "" {
		scenarioName = id
	}

	cacheKey := strings.ToLower(scenarioName)

	s.issueCacheMu.RLock()
	entry, cached := s.issueCache[cacheKey]
	cacheFresh := cached && time.Since(entry.fetchedAt) < s.issueCacheTTL
	s.issueCacheMu.RUnlock()

	if cacheFresh {
		return s.buildAppIssuesSummary(entry, true, false), nil
	}

	fetchedEntry, fetchErr := s.fetchScenarioIssues(ctx, id, scenarioName)
	if fetchErr != nil {
		if cached {
			return s.buildAppIssuesSummary(entry, true, true), nil
		}
		return nil, fetchErr
	}

	s.issueCacheMu.Lock()
	if s.issueCache == nil {
		s.issueCache = make(map[string]*issueCacheEntry)
	}
	s.issueCache[cacheKey] = fetchedEntry
	s.issueCacheMu.Unlock()

	return s.buildAppIssuesSummary(fetchedEntry, false, false), nil
}

func (s *AppService) fetchScenarioIssues(ctx context.Context, appID, scenarioName string) (*issueCacheEntry, error) {
	port, err := s.locateIssueTrackerAPIPort(ctx)
	if err != nil {
		return nil, ErrIssueTrackerUnavailable
	}

	query := url.Values{}
	query.Set("limit", strconv.Itoa(issueTrackerFetchLimit))
	query.Set("app_id", scenarioName)

	endpoint := fmt.Sprintf("http://localhost:%d/api/v1/issues?%s", port, query.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query app-issue-tracker: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("app-issue-tracker returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}

	var trackerResp issueTrackerAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&trackerResp); err != nil {
		return nil, fmt.Errorf("failed to decode app-issue-tracker response: %w", err)
	}

	if !trackerResp.Success {
		message := strings.TrimSpace(trackerResp.Error)
		if message == "" {
			message = strings.TrimSpace(trackerResp.Message)
		}
		if message == "" {
			message = "app-issue-tracker rejected the request"
		}
		return nil, errors.New(message)
	}

	issueSummaries := parseIssueTrackerIssues(trackerResp.Data)
	openCount := 0
	activeCount := 0
	for _, issue := range issueSummaries {
		switch strings.ToLower(issue.Status) {
		case "open":
			openCount++
		case "active":
			activeCount++
		}
	}

	totalCount := len(issueSummaries)

	trackerURL := ""
	var baseURL *url.URL
	if uiPort, err := resolveScenarioPortViaCLI(ctx, issueTrackerScenarioID, "UI_PORT"); err == nil && uiPort > 0 {
		base := &url.URL{Path: fmt.Sprintf("/apps/%s/proxy/", url.PathEscape(issueTrackerScenarioID))}
		params := base.Query()
		params.Set("app_id", scenarioName)
		base.RawQuery = params.Encode()
		trackerURL = base.String()
		baseURL = base
	}

	if baseURL != nil {
		for index, issue := range issueSummaries {
			if issue.ID == "" {
				continue
			}
			issueURL := *baseURL
			values := issueURL.Query()
			values.Set("issue", issue.ID)
			issueURL.RawQuery = values.Encode()
			issueSummaries[index].IssueURL = issueURL.String()
		}
	}

	entry := &issueCacheEntry{
		issues:      issueSummaries,
		scenario:    scenarioName,
		appID:       appID,
		trackerURL:  trackerURL,
		fetchedAt:   s.timeNow().UTC(),
		openCount:   openCount,
		activeCount: activeCount,
		totalCount:  totalCount,
	}

	return entry, nil
}

func parseIssueTrackerIssues(data map[string]interface{}) []AppIssueSummary {
	if data == nil {
		return []AppIssueSummary{}
	}

	rawIssues, ok := data["issues"].([]interface{})
	if !ok || len(rawIssues) == 0 {
		return []AppIssueSummary{}
	}

	issues := make([]AppIssueSummary, 0, len(rawIssues))
	for _, raw := range rawIssues {
		issueMap, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}

		id := strings.TrimSpace(anyString(issueMap["id"]))
		title := strings.TrimSpace(anyString(issueMap["title"]))
		status := strings.TrimSpace(anyString(issueMap["status"]))
		priority := strings.TrimSpace(anyString(issueMap["priority"]))

		createdAt := ""
		updatedAt := ""
		if meta, ok := issueMap["metadata"].(map[string]interface{}); ok {
			createdAt = strings.TrimSpace(anyString(meta["created_at"]))
			updatedAt = strings.TrimSpace(anyString(meta["updated_at"]))
		}

		reporter := ""
		if reporterMap, ok := issueMap["reporter"].(map[string]interface{}); ok {
			reporter = strings.TrimSpace(anyString(reporterMap["name"]))
			if reporter == "" {
				reporter = strings.TrimSpace(anyString(reporterMap["email"]))
			}
		}

		issues = append(issues, AppIssueSummary{
			ID:        id,
			Title:     title,
			Status:    status,
			Priority:  priority,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
			Reporter:  reporter,
		})
	}

	return issues
}

func (s *AppService) buildAppIssuesSummary(entry *issueCacheEntry, fromCache, stale bool) *AppIssuesSummary {
	if entry == nil {
		return &AppIssuesSummary{
			Issues:      []AppIssueSummary{},
			LastFetched: s.timeNow().UTC().Format(time.RFC3339),
			FromCache:   fromCache,
			Stale:       stale,
		}
	}

	clonedIssues := append([]AppIssueSummary(nil), entry.issues...)

	return &AppIssuesSummary{
		Scenario:    entry.scenario,
		AppID:       entry.appID,
		Issues:      clonedIssues,
		OpenCount:   entry.openCount,
		ActiveCount: entry.activeCount,
		TotalCount:  entry.totalCount,
		TrackerURL:  entry.trackerURL,
		LastFetched: entry.fetchedAt.Format(time.RFC3339),
		FromCache:   fromCache,
		Stale:       stale,
	}
}

// =============================================================================
// CLI Port Resolution
// =============================================================================

func (s *AppService) locateIssueTrackerAPIPort(ctx context.Context) (int, error) {
	if port, err := resolveScenarioPortViaCLI(ctx, issueTrackerScenarioID, "API_PORT"); err == nil && port > 0 {
		return port, nil
	} else if err != nil {
		logger.Warn("failed to resolve app-issue-tracker port via CLI", err)
	}

	apps, err := s.GetAppsFromOrchestrator(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to inspect scenarios: %w", err)
	}

	for _, candidate := range apps {
		name := strings.ToLower(strings.TrimSpace(candidate.ScenarioName))
		if name == "" {
			name = strings.ToLower(strings.TrimSpace(candidate.ID))
		}
		if name != issueTrackerScenarioID {
			continue
		}

		port := resolvePort(candidate.PortMappings, []string{"api", "api_port", "API", "API_PORT"})
		if port > 0 {
			return port, nil
		}
	}

	return 0, errors.New("app-issue-tracker is not running or no API port was found")
}

func (s *AppService) locateScenarioAuditorAPIPort(ctx context.Context) (int, error) {
	if port, err := resolveScenarioPortViaCLI(ctx, "scenario-auditor", "API_PORT"); err == nil && port > 0 {
		return port, nil
	} else if err != nil {
		logger.Warn("failed to resolve scenario-auditor port via CLI", err)
	}

	apps, err := s.GetAppsFromOrchestrator(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to inspect scenarios: %w", err)
	}

	for _, candidate := range apps {
		name := strings.ToLower(strings.TrimSpace(candidate.ScenarioName))
		if name == "" {
			name = strings.ToLower(strings.TrimSpace(candidate.ID))
		}
		if name != "scenario-auditor" {
			continue
		}

		port := resolvePort(candidate.PortMappings, []string{"api", "api_port", "API", "API_PORT"})
		if port > 0 {
			return port, nil
		}
	}

	return 0, errors.New("scenario-auditor is not running or no API port was found")
}

func resolveScenarioPortViaCLI(ctx context.Context, scenarioName, portLabel string) (int, error) {
	port, err := executeScenarioPortCommand(ctx, scenarioName, portLabel)
	if err == nil {
		return port, nil
	}

	fallbackPorts, fallbackErr := executeScenarioPortList(ctx, scenarioName)
	if fallbackErr == nil {
		candidate := strings.ToUpper(strings.TrimSpace(portLabel))
		if value, ok := fallbackPorts[candidate]; ok && value > 0 {
			return value, nil
		}
	}

	if fallbackErr != nil {
		return 0, fmt.Errorf("%v; fallback error: %w", err, fallbackErr)
	}

	return 0, err
}

func executeScenarioPortCommand(ctx context.Context, scenarioName, portLabel string) (int, error) {
	if strings.TrimSpace(scenarioName) == "" || strings.TrimSpace(portLabel) == "" {
		return 0, errors.New("scenario and port labels are required")
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctxWithTimeout, "vrooli", "scenario", "port", scenarioName, portLabel)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("vrooli scenario port %s %s failed: %s", scenarioName, portLabel, strings.TrimSpace(string(output)))
	}

	return parsePortValueFromString(strings.TrimSpace(string(output)))
}

func executeScenarioPortList(ctx context.Context, scenarioName string) (map[string]int, error) {
	if strings.TrimSpace(scenarioName) == "" {
		return nil, errors.New("scenario name is required")
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctxWithTimeout, "vrooli", "scenario", "port", scenarioName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("vrooli scenario port %s failed: %s", scenarioName, strings.TrimSpace(string(output)))
	}

	ports := make(map[string]int)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.ToUpper(strings.TrimSpace(parts[0]))
		value := strings.TrimSpace(parts[1])
		port, err := parsePortValueFromString(value)
		if err != nil {
			continue
		}

		if port > 0 {
			ports[key] = port
		}
	}

	if len(ports) == 0 {
		return nil, errors.New("no port mappings returned")
	}

	return ports, nil
}
