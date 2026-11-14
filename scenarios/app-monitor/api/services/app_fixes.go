package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ListScenarioFixes returns cached Swarm Manager fix backlog information for a scenario.
func (s *AppService) ListScenarioFixes(ctx context.Context, appID string) (*AppFixesSummary, error) {
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
		return s.buildAppFixesSummary(entry, true, false), nil
	}

	fetchedEntry, fetchErr := s.fetchScenarioFixes(ctx, id, scenarioName)
	if fetchErr != nil {
		if cached {
			return s.buildAppFixesSummary(entry, true, true), nil
		}
		return nil, fetchErr
	}

	s.issueCacheMu.Lock()
	if s.issueCache == nil {
		s.issueCache = make(map[string]*fixCacheEntry)
	}
	s.issueCache[cacheKey] = fetchedEntry
	s.issueCacheMu.Unlock()

	return s.buildAppFixesSummary(fetchedEntry, false, false), nil
}

func (s *AppService) fetchScenarioFixes(ctx context.Context, appID, scenarioName string) (*fixCacheEntry, error) {
	baseURL, err := s.resolveSwarmManagerURL(ctx)
	if err != nil {
		return nil, ErrSwarmManagerUnavailable
	}

	endpoint := strings.TrimRight(baseURL, "/") + "/api/v1/scenarios/" + url.PathEscape(scenarioName) + "/context"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query swarm-manager: %w", err)
	}
	if resp == nil {
		return nil, errors.New("http client returned nil response without error")
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("swarm-manager returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}

	var contextResp swarmScenarioContextResponse
	if err := json.NewDecoder(resp.Body).Decode(&contextResp); err != nil {
		return nil, fmt.Errorf("failed to decode swarm-manager scenario context: %w", err)
	}

	entry := &fixCacheEntry{
		active:    mapSwarmFixes(contextResp.Fixes.Active),
		archived:  mapSwarmFixes(contextResp.Fixes.Archived),
		scenario:  scenarioName,
		appID:     appID,
		fixesURL:  appMonitorSwarmPath("/"),
		fetchedAt: s.timeNow().UTC(),
	}
	return entry, nil
}

func mapSwarmFixes(fixes []swarmScenarioFix) []AppFixSummary {
	if len(fixes) == 0 {
		return []AppFixSummary{}
	}
	out := make([]AppFixSummary, 0, len(fixes))
	for _, fix := range fixes {
		archivedAt := ""
		if fix.ArchivedAt != nil {
			archivedAt = *fix.ArchivedAt
		}
		path := strings.TrimSpace(fix.Path)
		if path == "" {
			path = "fix/" + fix.Name
		}
		out = append(out, AppFixSummary{
			ID:         "fix/" + fix.Name,
			Kind:       "fix",
			Name:       fix.Name,
			Title:      fix.Title,
			Status:     fix.Status,
			Priority:   fix.Priority,
			UpdatedAt:  fix.Updated,
			ArchivedAt: archivedAt,
			Initiative: fix.Initiative,
			Path:       path,
			URL:        appMonitorSwarmPath("/backlog/" + strings.TrimPrefix(path, "/")),
		})
	}
	return out
}

func (s *AppService) buildAppFixesSummary(entry *fixCacheEntry, fromCache, stale bool) *AppFixesSummary {
	if entry == nil {
		return &AppFixesSummary{
			Active:      []AppFixSummary{},
			Archived:    []AppFixSummary{},
			Fixes:       []AppFixSummary{},
			LastFetched: s.timeNow().UTC().Format(time.RFC3339),
			FromCache:   fromCache,
			Stale:       stale,
		}
	}

	active := append([]AppFixSummary(nil), entry.active...)
	archived := append([]AppFixSummary(nil), entry.archived...)
	all := append(append([]AppFixSummary{}, active...), archived...)

	return &AppFixesSummary{
		Scenario:      entry.scenario,
		AppID:         entry.appID,
		Active:        active,
		Archived:      archived,
		Fixes:         all,
		ActiveCount:   len(active),
		ArchivedCount: len(archived),
		TotalCount:    len(all),
		SwarmURL:      entry.fixesURL,
		LastFetched:   entry.fetchedAt.Format(time.RFC3339),
		FromCache:     fromCache,
		Stale:         stale,
	}
}

func (s *AppService) resolveSwarmManagerURL(ctx context.Context) (string, error) {
	resolver := s.scenarioURL
	if resolver == nil {
		return "", ErrSwarmManagerUnavailable
	}
	baseURL, err := resolver(ctx, swarmManagerScenarioID)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(baseURL) == "" {
		return "", ErrSwarmManagerUnavailable
	}
	return strings.TrimRight(baseURL, "/"), nil
}

func appMonitorSwarmPath(path string) string {
	cleanPath := "/" + strings.TrimLeft(path, "/")
	return fmt.Sprintf("/apps/%s/proxy%s", url.PathEscape(swarmManagerScenarioID), cleanPath)
}
