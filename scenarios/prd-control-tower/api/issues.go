package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	issueTrackerScenarioID = "app-issue-tracker"
	issueTrackerFetchLimit = 50
)

var (
	scenarioIssuesCacheMu  sync.RWMutex
	scenarioIssuesCache    = make(map[string]*scenarioIssuesCacheEntry)
	scenarioIssuesCacheTTL = 2 * time.Minute

	issueTrackerHTTPClient = &http.Client{Timeout: 15 * time.Second}

	issueTrackerAPIPortResolver = locateIssueTrackerAPIPort
	issueTrackerUIPortResolver  = locateIssueTrackerUIPort
)

type scenarioIssuesCacheEntry struct {
	summary   ScenarioIssuesSummary
	fetchedAt time.Time
}

// IssueTrackerIssueSummary represents a trimmed issue entry suitable for the UI.
type IssueTrackerIssueSummary struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	Status        string `json:"status"`
	Priority      string `json:"priority,omitempty"`
	CreatedAt     string `json:"created_at,omitempty"`
	UpdatedAt     string `json:"updated_at,omitempty"`
	Reporter      string `json:"reporter,omitempty"`
	IssueURL      string `json:"issue_url,omitempty"`
	LocalIssueURL string `json:"local_issue_url,omitempty"`
}

// ScenarioIssuesSummary is returned by GET /issues/status.
type ScenarioIssuesSummary struct {
	EntityType      string                     `json:"entity_type"`
	EntityName      string                     `json:"entity_name"`
	Issues          []IssueTrackerIssueSummary `json:"issues"`
	OpenCount       int                        `json:"open_count"`
	ActiveCount     int                        `json:"active_count"`
	TotalCount      int                        `json:"total_count"`
	TrackerURL      string                     `json:"tracker_url,omitempty"`
	LocalTrackerURL string                     `json:"local_tracker_url,omitempty"`
	LastFetched     string                     `json:"last_fetched"`
	FromCache       bool                       `json:"from_cache"`
	Stale           bool                       `json:"stale"`
}

// ScenarioIssueReportRequest defines the payload accepted by POST /issues/report.
type ScenarioIssueReportRequest struct {
	EntityType  string                  `json:"entity_type"`
	EntityName  string                  `json:"entity_name"`
	Source      string                  `json:"source"`
	Title       string                  `json:"title"`
	Description string                  `json:"description"`
	Priority    string                  `json:"priority,omitempty"`
	Summary     string                  `json:"summary,omitempty"`
	Tags        []string                `json:"tags,omitempty"`
	Labels      map[string]string       `json:"labels,omitempty"`
	Metadata    map[string]string       `json:"metadata,omitempty"`
	Selections  []IssueReportSelection  `json:"selections"`
	Attachments []IssueReportAttachment `json:"attachments,omitempty"`
}

// IssueReportSelection captures a single checkbox entry from the UI.
type IssueReportSelection struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Detail    string `json:"detail"`
	Category  string `json:"category"`
	Severity  string `json:"severity"`
	Reference string `json:"reference"`
	Notes     string `json:"notes"`
}

// IssueReportAttachment allows callers to attach rich context without hitting the filesystem.
type IssueReportAttachment struct {
	Name        string `json:"name"`
	Content     string `json:"content"`
	ContentType string `json:"content_type,omitempty"`
	Encoding    string `json:"encoding,omitempty"`
	Category    string `json:"category,omitempty"`
	Description string `json:"description,omitempty"`
}

// IssueReportResponse mirrors the key data returned by app-issue-tracker.
type IssueReportResponse struct {
	IssueID  string `json:"issue_id"`
	IssueURL string `json:"issue_url,omitempty"`
	Message  string `json:"message"`
}

type issueTrackerListResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error"`
	Data    struct {
		Issues []issueTrackerIssue `json:"issues"`
		Count  int                 `json:"count"`
	} `json:"data"`
}

type issueTrackerCreateResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error"`
	Data    struct {
		IssueID string            `json:"issue_id"`
		Issue   issueTrackerIssue `json:"issue"`
	} `json:"data"`
}

type issueTrackerIssue struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	Priority string `json:"priority"`
	Reporter struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"reporter"`
	Metadata struct {
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
	} `json:"metadata"`
}

func handleGetScenarioIssuesStatus(w http.ResponseWriter, r *http.Request) {
	entityType := strings.TrimSpace(r.URL.Query().Get("entity_type"))
	entityName := strings.TrimSpace(r.URL.Query().Get("entity_name"))

	if entityType == "" || entityName == "" {
		respondBadRequest(w, "entity_type and entity_name are required")
		return
	}
	if !isValidEntityType(entityType) {
		respondInvalidEntityType(w)
		return
	}

	useCache := parseBoolQuery(r, "use_cache", true)
	summary, err := fetchScenarioIssuesSummary(r.Context(), entityType, entityName, useCache)
	if err != nil {
		slog.Warn("issue tracker query failed", "entity", entityName, "error", err)
		respondError(w, fmt.Sprintf("app-issue-tracker unavailable: %v", err), http.StatusBadGateway)
		return
	}

	respondJSON(w, http.StatusOK, summary)
}

func handleSubmitIssueReport(w http.ResponseWriter, r *http.Request) {
	var req ScenarioIssueReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondInvalidJSON(w, err)
		return
	}

	req.EntityType = strings.TrimSpace(req.EntityType)
	req.EntityName = strings.TrimSpace(req.EntityName)
	req.Source = strings.TrimSpace(req.Source)
	req.Title = strings.TrimSpace(req.Title)
	req.Description = strings.TrimSpace(req.Description)

	if req.EntityType == "" || req.EntityName == "" {
		respondBadRequest(w, "entity_type and entity_name are required")
		return
	}
	if !isValidEntityType(req.EntityType) {
		respondInvalidEntityType(w)
		return
	}
	if req.Title == "" {
		respondBadRequest(w, "title is required")
		return
	}
	if req.Description == "" {
		respondBadRequest(w, "description is required")
		return
	}
	if len(req.Selections) == 0 {
		respondBadRequest(w, "at least one issue selection is required")
		return
	}

	response, err := submitIssueReport(r.Context(), &req)
	if err != nil {
		slog.Error("issue report submission failed", "entity", req.EntityName, "error", err)
		respondError(w, fmt.Sprintf("failed to submit issue report: %v", err), http.StatusBadGateway)
		return
	}

	invalidateScenarioIssueCache(cacheKeyForScenario(req.EntityType, req.EntityName))

	respondJSON(w, http.StatusOK, response)
}

func fetchScenarioIssuesSummary(ctx context.Context, entityType, entityName string, useCache bool) (ScenarioIssuesSummary, error) {
	key := cacheKeyForScenario(entityType, entityName)
	if key == "" {
		return ScenarioIssuesSummary{}, errors.New("invalid cache key")
	}

	if useCache {
		scenarioIssuesCacheMu.RLock()
		entry, ok := scenarioIssuesCache[key]
		scenarioIssuesCacheMu.RUnlock()
		if ok && time.Since(entry.fetchedAt) < scenarioIssuesCacheTTL {
			cached := entry.summary
			cached.FromCache = true
			return cached, nil
		}
	} else {
		scenarioIssuesCacheMu.Lock()
		delete(scenarioIssuesCache, key)
		scenarioIssuesCacheMu.Unlock()
	}

	summary, err := queryIssueTrackerIssues(ctx, entityType, entityName)
	if err != nil {
		// Attempt to fall back to stale cache
		scenarioIssuesCacheMu.RLock()
		entry, ok := scenarioIssuesCache[key]
		scenarioIssuesCacheMu.RUnlock()
		if ok {
			cached := entry.summary
			cached.FromCache = true
			cached.Stale = true
			return cached, nil
		}
		return ScenarioIssuesSummary{}, err
	}

	scenarioIssuesCacheMu.Lock()
	scenarioIssuesCache[key] = &scenarioIssuesCacheEntry{summary: summary, fetchedAt: time.Now()}
	scenarioIssuesCacheMu.Unlock()

	return summary, nil
}

func cacheKeyForScenario(entityType, entityName string) string {
	if entityType == "" || entityName == "" {
		return ""
	}
	return strings.ToLower(fmt.Sprintf("%s:%s", entityType, entityName))
}

func invalidateScenarioIssueCache(key string) {
	if key == "" {
		return
	}
	scenarioIssuesCacheMu.Lock()
	delete(scenarioIssuesCache, key)
	scenarioIssuesCacheMu.Unlock()
}

func queryIssueTrackerIssues(ctx context.Context, entityType, entityName string) (ScenarioIssuesSummary, error) {
	port, err := issueTrackerAPIPortResolver(ctx)
	if err != nil {
		return ScenarioIssuesSummary{}, err
	}
	if port <= 0 {
		return ScenarioIssuesSummary{}, errors.New("invalid app-issue-tracker port")
	}

	values := url.Values{}
	values.Set("target_type", entityType)
	values.Set("target_id", entityName)
	values.Set("limit", strconv.Itoa(issueTrackerFetchLimit))

	endpoint := fmt.Sprintf("http://localhost:%d/api/v1/issues?%s", port, values.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return ScenarioIssuesSummary{}, err
	}

	resp, err := issueTrackerHTTPClient.Do(req)
	if err != nil {
		return ScenarioIssuesSummary{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return ScenarioIssuesSummary{}, fmt.Errorf("issue tracker returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var trackerResp issueTrackerListResponse
	if err := json.NewDecoder(resp.Body).Decode(&trackerResp); err != nil {
		return ScenarioIssuesSummary{}, err
	}
	if !trackerResp.Success {
		message := strings.TrimSpace(trackerResp.Error)
		if message == "" {
			message = strings.TrimSpace(trackerResp.Message)
		}
		if message == "" {
			message = "issue tracker rejected the request"
		}
		return ScenarioIssuesSummary{}, errors.New(message)
	}

	uiPort, err := issueTrackerUIPortResolver(ctx)
	if err != nil {
		slog.Debug("issue tracker UI port unavailable", "error", err)
	}

	trackerURL, localURL := buildIssueTrackerURLs(entityType, entityName, uiPort)

	summary := ScenarioIssuesSummary{
		EntityType:      entityType,
		EntityName:      entityName,
		TrackerURL:      trackerURL,
		LocalTrackerURL: localURL,
		Issues:          make([]IssueTrackerIssueSummary, 0, len(trackerResp.Data.Issues)),
		LastFetched:     time.Now().UTC().Format(time.RFC3339),
	}

	for _, issue := range trackerResp.Data.Issues {
		summary.TotalCount++
		status := strings.ToLower(strings.TrimSpace(issue.Status))
		switch status {
		case "open":
			summary.OpenCount++
		case "active":
			summary.ActiveCount++
		}

		reporter := strings.TrimSpace(issue.Reporter.Name)
		if reporter == "" {
			reporter = strings.TrimSpace(issue.Reporter.Email)
		}

		item := IssueTrackerIssueSummary{
			ID:        issue.ID,
			Title:     issue.Title,
			Status:    issue.Status,
			Priority:  issue.Priority,
			Reporter:  reporter,
			CreatedAt: issue.Metadata.CreatedAt,
			UpdatedAt: issue.Metadata.UpdatedAt,
		}
		item.IssueURL = appendIssueParam(trackerURL, issue.ID)
		item.LocalIssueURL = appendIssueParam(localURL, issue.ID)

		summary.Issues = append(summary.Issues, item)
	}

	return summary, nil
}

func buildIssueTrackerURLs(entityType, entityName string, uiPort int) (string, string) {
	query := url.Values{}
	query.Set("target_type", entityType)
	query.Set("target_id", entityName)
	query.Set("app_id", entityName)

	proxy := &url.URL{Path: fmt.Sprintf("/apps/%s/proxy/", url.PathEscape(issueTrackerScenarioID))}
	proxy.RawQuery = query.Encode()

	var local string
	if uiPort > 0 {
		direct := &url.URL{Scheme: "http", Host: fmt.Sprintf("localhost:%d", uiPort)}
		direct.RawQuery = query.Encode()
		local = direct.String()
	}

	return proxy.String(), local
}

func appendIssueParam(base, issueID string) string {
	if base == "" || issueID == "" {
		return ""
	}
	u, err := url.Parse(base)
	if err != nil {
		return base
	}
	query := u.Query()
	query.Set("issue", issueID)
	u.RawQuery = query.Encode()
	return u.String()
}

func submitIssueReport(ctx context.Context, req *ScenarioIssueReportRequest) (IssueReportResponse, error) {
	port, err := issueTrackerAPIPortResolver(ctx)
	if err != nil {
		return IssueReportResponse{}, err
	}
	if port <= 0 {
		return IssueReportResponse{}, errors.New("invalid app-issue-tracker port")
	}

	payload := buildIssueTrackerPayload(req)
	body, err := json.Marshal(payload)
	if err != nil {
		return IssueReportResponse{}, err
	}

	endpoint := fmt.Sprintf("http://localhost:%d/api/v1/issues", port)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return IssueReportResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := issueTrackerHTTPClient.Do(httpReq)
	if err != nil {
		return IssueReportResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return IssueReportResponse{}, fmt.Errorf("issue tracker returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}

	var trackerResp issueTrackerCreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&trackerResp); err != nil {
		return IssueReportResponse{}, err
	}
	if !trackerResp.Success {
		message := strings.TrimSpace(trackerResp.Error)
		if message == "" {
			message = strings.TrimSpace(trackerResp.Message)
		}
		if message == "" {
			message = "issue tracker rejected the request"
		}
		return IssueReportResponse{}, errors.New(message)
	}

	trackerURL, _ := buildIssueTrackerURLs(req.EntityType, req.EntityName, 0)
	response := IssueReportResponse{
		IssueID:  trackerResp.Data.IssueID,
		Message:  trackerResp.Message,
		IssueURL: appendIssueParam(trackerURL, trackerResp.Data.IssueID),
	}
	if response.Message == "" {
		response.Message = "Issue reported successfully"
	}

	return response, nil
}

func buildIssueTrackerPayload(req *ScenarioIssueReportRequest) map[string]any {
	description := strings.TrimSpace(req.Description)
	if len(req.Selections) > 0 {
		var b strings.Builder
		b.WriteString(description)
		b.WriteString("\n\n## Selected issues\n")
		for _, sel := range req.Selections {
			label := strings.TrimSpace(sel.Title)
			detail := strings.TrimSpace(sel.Detail)
			category := strings.TrimSpace(sel.Category)
			severity := strings.TrimSpace(sel.Severity)
			var parts []string
			if category != "" {
				parts = append(parts, category)
			}
			if severity != "" {
				parts = append(parts, strings.ToUpper(severity))
			}
			if label != "" {
				parts = append(parts, label)
			}
			line := strings.Join(parts, " · ")
			if detail != "" {
				line = fmt.Sprintf("%s — %s", line, detail)
			}
			b.WriteString("- ")
			b.WriteString(strings.TrimSpace(line))
			if ref := strings.TrimSpace(sel.Reference); ref != "" {
				b.WriteString(" (ref: ")
				b.WriteString(ref)
				b.WriteString(")")
			}
			if notes := strings.TrimSpace(sel.Notes); notes != "" {
				b.WriteString("\n  Notes: ")
				b.WriteString(notes)
			}
			b.WriteString("\n")
		}
		description = b.String()
	}

	tags := normalizeTags(req.Tags)
	tags = append(tags, fmt.Sprintf("entity:%s/%s", req.EntityType, req.EntityName))
	if req.Source != "" {
		tags = append(tags, fmt.Sprintf("source:%s", req.Source))
	}

	labels := normalizeStringMap(req.Labels)
	if req.Source != "" {
		if labels == nil {
			labels = make(map[string]string)
		}
		if _, exists := labels["report_source"]; !exists {
			labels["report_source"] = req.Source
		}
	}

	metadataExtra := normalizeStringMap(req.Metadata)
	if metadataExtra == nil {
		metadataExtra = make(map[string]string)
	}
	if len(req.Selections) > 0 {
		metadataExtra["selection_count"] = strconv.Itoa(len(req.Selections))
	}

	environment := map[string]string{
		"entity_type": req.EntityType,
		"entity_name": req.EntityName,
	}
	if req.Source != "" {
		environment["source"] = req.Source
	}

	target := map[string]string{
		"type": req.EntityType,
		"id":   req.EntityName,
	}
	if displayName, ok := metadataExtra["display_name"]; ok && displayName != "" {
		target["name"] = displayName
	}
	targets := []map[string]string{target}

	artifacts := buildArtifacts(req.Attachments)

	payload := map[string]any{
		"title":          req.Title,
		"description":    description,
		"type":           "quality_issue",
		"priority":       determineIssuePriority(req),
		"status":         "open",
		"targets":        targets,
		"tags":           dedupeStrings(tags),
		"reporter_name":  "PRD Control Tower",
		"reporter_email": "prd-control-tower@vrooli.local",
	}
	if len(environment) > 0 {
		payload["environment"] = environment
	}
	if len(labels) > 0 {
		payload["labels"] = labels
	}
	if len(metadataExtra) > 0 {
		payload["metadata_extra"] = metadataExtra
	}

	if strings.TrimSpace(req.Summary) != "" {
		payload["notes"] = strings.TrimSpace(req.Summary)
	}
	if len(artifacts) > 0 {
		payload["artifacts"] = artifacts
	}

	return payload
}

func buildArtifacts(attachments []IssueReportAttachment) []map[string]string {
	if len(attachments) == 0 {
		return nil
	}
	artifacts := make([]map[string]string, 0, len(attachments))
	for _, attachment := range attachments {
		name := strings.TrimSpace(attachment.Name)
		content := strings.TrimSpace(attachment.Content)
		if name == "" || content == "" {
			continue
		}
		encoding := strings.TrimSpace(attachment.Encoding)
		if encoding == "" {
			encoding = "plain"
		}
		contentType := strings.TrimSpace(attachment.ContentType)
		if contentType == "" {
			contentType = "text/markdown"
		}
		artifacts = append(artifacts, map[string]string{
			"name":         name,
			"category":     strings.TrimSpace(attachment.Category),
			"content":      content,
			"encoding":     encoding,
			"content_type": contentType,
			"description":  strings.TrimSpace(attachment.Description),
		})
	}
	if len(artifacts) == 0 {
		return nil
	}
	return artifacts
}

func determineIssuePriority(req *ScenarioIssueReportRequest) string {
	priority := strings.TrimSpace(strings.ToLower(req.Priority))
	if priority != "" {
		return priority
	}

	highest := "medium"
	for _, selection := range req.Selections {
		switch strings.ToLower(selection.Severity) {
		case "critical", "blocker", "p0":
			return "critical"
		case "high", "major", "p1":
			highest = "high"
		case "low", "p3":
			if highest != "high" {
				highest = "low"
			}
		}
	}
	return highest
}

func normalizeTags(tags []string) []string {
	cleaned := make([]string, 0, len(tags))
	for _, tag := range tags {
		trimmed := strings.TrimSpace(tag)
		if trimmed == "" {
			continue
		}
		cleaned = append(cleaned, strings.ToLower(trimmed))
	}
	return cleaned
}

func normalizeStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	normalized := make(map[string]string, len(values))
	for key, value := range values {
		trimmedKey := strings.TrimSpace(key)
		trimmedValue := strings.TrimSpace(value)
		if trimmedKey == "" || trimmedValue == "" {
			continue
		}
		normalized[trimmedKey] = trimmedValue
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func dedupeStrings(values []string) []string {
	if len(values) == 0 {
		return values
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func locateIssueTrackerAPIPort(ctx context.Context) (int, error) {
	return resolveScenarioPortViaCLI(ctx, issueTrackerScenarioID, "API_PORT")
}

func locateIssueTrackerUIPort(ctx context.Context) (int, error) {
	return resolveScenarioPortViaCLI(ctx, issueTrackerScenarioID, "UI_PORT")
}
