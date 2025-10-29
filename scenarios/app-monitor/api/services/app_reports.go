package services

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"app-monitor-api/logger"
)

// =============================================================================
// Issue Report Builder
// =============================================================================

// ReportAppIssue forwards an issue report to the app-issue-tracker scenario
func (s *AppService) ReportAppIssue(ctx context.Context, req *IssueReportRequest) (*IssueReportResult, error) {
	if req == nil {
		return nil, errors.New("request payload is required")
	}

	appID := strings.TrimSpace(req.AppID)
	if appID == "" {
		return nil, errors.New("app identifier is required")
	}

	message := strings.TrimSpace(req.Message)
	if message == "" {
		return nil, errors.New("issue message is required")
	}

	primaryDescription := trimmedStringPtr(req.PrimaryDescription)
	includeDiagnosticsSummary := false
	if req.IncludeDiagnosticsSummary != nil && *req.IncludeDiagnosticsSummary {
		includeDiagnosticsSummary = true
	}

	reportedAt := s.timeNow().UTC()

	appName := trimmedStringPtr(req.AppName)
	scenarioName := trimmedStringPtr(req.ScenarioName)

	if app, err := s.GetApp(ctx, appID); err == nil && app != nil {
		if appName == "" {
			appName = app.Name
		}
		if scenarioName == "" {
			scenarioName = app.ScenarioName
		}
	}

	if appName == "" {
		appName = appID
	}
	if scenarioName == "" {
		scenarioName = appID
	}

	previewURL := ""
	if req.PreviewURL != nil {
		previewURL = normalizePreviewURL(*req.PreviewURL)
	}

	screenshotData := trimmedStringPtr(req.ScreenshotData)
	if screenshotData != "" {
		if _, err := base64.StdEncoding.DecodeString(screenshotData); err != nil {
			logger.Warn("invalid screenshot data provided, ignoring", err)
			screenshotData = ""
		}
	}

	const (
		maxReportLogs           = 300
		maxConsoleLogs          = 200
		maxNetworkEntries       = 150
		maxConsoleTextLength    = 2000
		maxNetworkURLLength     = 2048
		maxNetworkErrLength     = 1500
		maxRequestIDLength      = 128
		maxHealthEntries        = 20
		maxHealthNameLength     = 120
		maxHealthEndpointLength = 512
		maxHealthMessageLength  = 400
		maxHealthCodeLength     = 120
		maxHealthResponseLength = 4000
		maxStatusLines          = 120
		maxCaptureEntries       = 12
		maxCaptureNoteLength    = 600
		maxCaptureLabelLength   = 160
		maxCaptureTextLength    = 900
	)
	captures := sanitizeIssueCaptures(req.Captures, maxCaptureEntries, maxCaptureNoteLength, maxCaptureLabelLength, maxCaptureTextLength)
	elementCaptureCount := 0
	pageCaptureCount := 0
	for _, capture := range captures {
		if capture.Type == "page" {
			pageCaptureCount++
		} else {
			elementCaptureCount++
		}
	}
	sanitizedLogs := make([]string, 0, len(req.Logs))
	for _, line := range req.Logs {
		trimmed := strings.TrimRight(line, "\r\n")
		sanitizedLogs = append(sanitizedLogs, trimmed)
	}
	if len(sanitizedLogs) > maxReportLogs {
		sanitizedLogs = sanitizedLogs[len(sanitizedLogs)-maxReportLogs:]
	}

	logsTotal := valueOrDefault(req.LogsTotal, len(req.Logs))
	if logsTotal <= 0 {
		logsTotal = len(req.Logs)
	}

	logsCapturedAt := trimmedStringPtr(req.LogsCapturedAt)
	consoleCapturedAt := trimmedStringPtr(req.ConsoleCapturedAt)
	networkCapturedAt := trimmedStringPtr(req.NetworkCapturedAt)

	consoleLogs := sanitizeConsoleLogs(req.ConsoleLogs, maxConsoleLogs, maxConsoleTextLength)
	consoleTotal := valueOrDefault(req.ConsoleLogsTotal, len(req.ConsoleLogs))
	if consoleTotal <= 0 {
		consoleTotal = len(req.ConsoleLogs)
	}

	networkEntries := sanitizeNetworkRequests(req.NetworkRequests, maxNetworkEntries, maxNetworkURLLength, maxNetworkErrLength, maxRequestIDLength)
	networkTotal := valueOrDefault(req.NetworkTotal, len(req.NetworkRequests))
	if networkTotal <= 0 {
		networkTotal = len(req.NetworkRequests)
	}

	healthEntries := sanitizeHealthChecks(
		req.HealthChecks,
		maxHealthEntries,
		maxHealthNameLength,
		maxHealthEndpointLength,
		maxHealthMessageLength,
		maxHealthCodeLength,
		maxHealthResponseLength,
	)
	healthTotal := valueOrDefault(req.HealthChecksTotal, len(req.HealthChecks))
	if healthTotal <= 0 {
		healthTotal = len(req.HealthChecks)
	}
	healthCapturedAt := trimmedStringPtr(req.HealthChecksCapturedAt)

	statusLines := filterNonEmptyStrings(req.AppStatusLines)
	if len(statusLines) > maxStatusLines {
		statusLines = statusLines[len(statusLines)-maxStatusLines:]
	}
	statusLabel := trimmedStringPtr(req.AppStatusLabel)
	statusSeverity := strings.ToLower(trimmedStringPtr(req.AppStatusSeverity))
	switch statusSeverity {
	case "", "ok", "warn", "error":
		// use as-is
	case "warning":
		statusSeverity = "warn"
	case "fail", "failed", "critical":
		statusSeverity = "error"
	default:
		statusSeverity = "warn"
	}
	statusCapturedAt := trimmedStringPtr(req.AppStatusCapturedAt)

	includeScreenshot := pageCaptureCount > 0 || screenshotData != ""
	title := deriveIssueTitle(primaryDescription, message, captures, includeDiagnosticsSummary, includeScreenshot)
	reportSource := trimmedStringPtr(req.Source)
	description := buildIssueDescription(
		appName,
		scenarioName,
		previewURL,
		reportSource,
		message,
		screenshotData,
		captures,
		reportedAt,
		sanitizedLogs,
		logsTotal,
		logsCapturedAt,
		consoleLogs,
		consoleTotal,
		consoleCapturedAt,
		networkEntries,
		networkTotal,
		networkCapturedAt,
		healthEntries,
		healthTotal,
		healthCapturedAt,
		statusLines,
		statusLabel,
		statusCapturedAt,
		statusSeverity,
		s.repoRoot,
	)
	tags := buildIssueTags(scenarioName)
	environment := buildIssueEnvironment(appID, appName, previewURL, reportSource, reportedAt)

	port, err := s.locateIssueTrackerAPIPort(ctx)
	if err != nil {
		return nil, err
	}

	metadataExtra := map[string]string{}
	if reportSource != "" {
		metadataExtra["report_source"] = reportSource
	}
	if previewURL != "" {
		metadataExtra["preview_url"] = previewURL
	}
	if logsTotal > 0 {
		metadataExtra["logs_total"] = strconv.Itoa(logsTotal)
	}
	if logsCapturedAt != "" {
		metadataExtra["logs_captured_at"] = logsCapturedAt
	}
	if consoleTotal > 0 {
		metadataExtra["console_total"] = strconv.Itoa(consoleTotal)
	}
	if consoleCapturedAt != "" {
		metadataExtra["console_captured_at"] = consoleCapturedAt
	}
	if networkTotal > 0 {
		metadataExtra["network_total"] = strconv.Itoa(networkTotal)
	}
	if networkCapturedAt != "" {
		metadataExtra["network_captured_at"] = networkCapturedAt
	}
	if healthTotal > 0 {
		metadataExtra["health_total"] = strconv.Itoa(healthTotal)
	}
	if healthCapturedAt != "" {
		metadataExtra["health_captured_at"] = healthCapturedAt
	}
	if screenshotData != "" {
		metadataExtra["screenshot_included"] = "true"
	}
	if len(statusLines) > 0 {
		metadataExtra["app_status_lines"] = strconv.Itoa(len(statusLines))
	}
	if statusLabel != "" {
		metadataExtra["app_status_label"] = statusLabel
	}
	if statusSeverity != "" {
		metadataExtra["app_status_severity"] = statusSeverity
	}
	if statusCapturedAt != "" {
		metadataExtra["app_status_captured_at"] = statusCapturedAt
	}
	if len(captures) > 0 {
		metadataExtra["captures_total"] = strconv.Itoa(len(captures))
	}
	if elementCaptureCount > 0 {
		metadataExtra["captures_element_total"] = strconv.Itoa(elementCaptureCount)
	}
	if pageCaptureCount > 0 {
		metadataExtra["captures_page_total"] = strconv.Itoa(pageCaptureCount)
	}

	artifacts := make([]map[string]interface{}, 0, 5)
	if len(sanitizedLogs) > 0 {
		artifactContent := strings.Join(sanitizedLogs, "\n")
		description := fmt.Sprintf("Scenario startup logs (%d lines captured", len(sanitizedLogs))
		if logsTotal > len(sanitizedLogs) {
			description = fmt.Sprintf("%s, %d total", description, logsTotal)
		}
		description = description + ")"
		artifacts = append(artifacts, map[string]interface{}{
			"name":         attachmentLifecycleName,
			"category":     "logs",
			"content":      artifactContent,
			"encoding":     "plain",
			"content_type": "text/plain",
			"description":  description,
		})
	}
	if len(consoleLogs) > 0 {
		consolePayload, err := json.MarshalIndent(consoleLogs, "", "  ")
		consoleContent := ""
		if err == nil {
			consoleContent = string(consolePayload)
		} else {
			consoleContent = "[]"
		}
		errorCount := countConsoleErrors(consoleLogs)
		description := fmt.Sprintf("Browser console events (%d captured)", len(consoleLogs))
		if errorCount > 0 {
			description = fmt.Sprintf("Browser console events (%d error%s logged)", errorCount, plural(errorCount))
		}
		artifacts = append(artifacts, map[string]interface{}{
			"name":         attachmentConsoleName,
			"category":     "console",
			"content":      consoleContent,
			"encoding":     "plain",
			"content_type": "application/json",
			"description":  description,
		})
	}
	if len(networkEntries) > 0 {
		networkPayload, err := json.MarshalIndent(networkEntries, "", "  ")
		networkContent := ""
		if err == nil {
			networkContent = string(networkPayload)
		} else {
			networkContent = "[]"
		}
		failedCount := countFailedRequests(networkEntries)
		description := fmt.Sprintf("Network requests (%d captured)", len(networkEntries))
		if failedCount > 0 {
			firstFailed := getFirstFailedRequest(networkEntries)
			if firstFailed != nil && firstFailed.Status != nil {
				method := strings.ToUpper(firstFailed.Method)
				description = fmt.Sprintf("Network requests (%d failed: %s %s → %d)", failedCount, method, firstFailed.URL, *firstFailed.Status)
			} else {
				description = fmt.Sprintf("Network requests (%d failed)", failedCount)
			}
		}
		artifacts = append(artifacts, map[string]interface{}{
			"name":         attachmentNetworkName,
			"category":     "network",
			"content":      networkContent,
			"encoding":     "plain",
			"content_type": "application/json",
			"description":  description,
		})
	}
	if len(healthEntries) > 0 {
		healthPayload, err := json.MarshalIndent(healthEntries, "", "  ")
		healthContent := ""
		if err == nil {
			healthContent = string(healthPayload)
		} else {
			healthContent = "[]"
		}
		passCount := countPassingHealthChecks(healthEntries)
		failCount := len(healthEntries) - passCount
		description := fmt.Sprintf("Health checks (%d passed, %d failed)", passCount, failCount)
		if failCount == 0 {
			description = fmt.Sprintf("Health checks (all %d passed)", passCount)
		}
		artifacts = append(artifacts, map[string]interface{}{
			"name":         attachmentHealthName,
			"category":     "health",
			"content":      healthContent,
			"encoding":     "plain",
			"content_type": "application/json",
			"description":  description,
		})
	}
	if len(statusLines) > 0 {
		artifactContent := strings.Join(statusLines, "\n")
		description := fmt.Sprintf("Scenario status output (%s severity)", strings.ToUpper(statusSeverity))
		artifacts = append(artifacts, map[string]interface{}{
			"name":         attachmentStatusName,
			"category":     "status",
			"content":      artifactContent,
			"encoding":     "plain",
			"content_type": "text/plain",
			"description":  description,
		})
	}
	if screenshotData != "" {
		artifacts = append(artifacts, map[string]interface{}{
			"name":         attachmentScreenshotName,
			"category":     "screenshot",
			"content":      screenshotData,
			"encoding":     "base64",
			"content_type": "image/png",
			"description":  "Full page screenshot captured at report time",
		})
	}

	elementAttachmentIndex := 0
	for _, capture := range captures {
		if capture.Type != "element" {
			continue
		}
		elementAttachmentIndex++
		name := fmt.Sprintf("element-%02d.png", elementAttachmentIndex)
		description := fmt.Sprintf("Element capture: %s", resolveCaptureLabel(capture, elementAttachmentIndex-1))
		if note := strings.TrimSpace(capture.Note); note != "" {
			description = fmt.Sprintf("%s - %s", description, truncateTitle(note, 80))
		}
		artifacts = append(artifacts, map[string]interface{}{
			"name":         name,
			"category":     "screenshot",
			"content":      capture.Data,
			"encoding":     "base64",
			"content_type": "image/png",
			"description":  description,
		})
	}

	if len(metadataExtra) == 0 {
		metadataExtra = nil
	}

	reporterName := "App Monitor"
	if reportSource != "" {
		reporterName = fmt.Sprintf("App Monitor – %s", reportSource)
	}
	reporterEmail := "monitor@vrooli.local"

	payload := map[string]interface{}{
		"title":          title,
		"description":    description,
		"type":           "bug",
		"priority":       "medium",
		"app_id":         scenarioName,
		"tags":           tags,
		"environment":    environment,
		"metadata_extra": metadataExtra,
		"artifacts":      artifacts,
		"reporter_name":  reporterName,
		"reporter_email": reporterEmail,
	}

	result, err := s.submitIssueToTracker(ctx, port, payload)
	if err != nil {
		return nil, err
	}

	if result != nil && result.IssueID != "" {
		if uiPort, err := resolveScenarioPortViaCLI(ctx, issueTrackerScenarioID, "UI_PORT"); err == nil && uiPort > 0 {
			u := url.URL{
				Path: fmt.Sprintf("/apps/%s/proxy/", url.PathEscape(issueTrackerScenarioID)),
			}
			query := u.Query()
			query.Set("issue", result.IssueID)
			u.RawQuery = query.Encode()
			result.IssueURL = u.String()
		} else if err != nil {
			logger.Warn("failed to resolve app-issue-tracker UI port", err)
		}
	}

	return result, nil
}

// =============================================================================
// Issue Tracker HTTP Client
// =============================================================================

func (s *AppService) submitIssueToTracker(ctx context.Context, port int, payload map[string]interface{}) (*IssueReportResult, error) {
	if port <= 0 {
		return nil, errors.New("invalid app-issue-tracker port")
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("http://localhost:%d/api/v1/issues", port)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call app-issue-tracker: %w", err)
	}
	if resp == nil {
		return nil, errors.New("http client returned nil response without error")
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
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
			message = "app-issue-tracker rejected the issue report"
		}
		return nil, errors.New(message)
	}

	issueID := ""
	if trackerResp.Data != nil {
		if value, ok := trackerResp.Data["issue_id"].(string); ok {
			issueID = value
		} else if value, ok := trackerResp.Data["issueId"].(string); ok {
			issueID = value
		} else if rawIssue, ok := trackerResp.Data["issue"]; ok {
			switch v := rawIssue.(type) {
			case map[string]interface{}:
				if nested, ok := v["id"].(string); ok {
					issueID = nested
				}
			case struct{ ID string }:
				issueID = strings.TrimSpace(v.ID)
			}
		}
	}

	resultMessage := strings.TrimSpace(trackerResp.Message)
	if resultMessage == "" {
		resultMessage = "Issue reported successfully"
	}

	return &IssueReportResult{
		IssueID: issueID,
		Message: resultMessage,
	}, nil
}

// =============================================================================
// Sanitization Functions
// =============================================================================

func sanitizeConsoleLogs(entries []IssueConsoleLogEntry, maxEntries, maxText int) []IssueConsoleLogEntry {
	if len(entries) == 0 {
		return nil
	}
	sanitized := make([]IssueConsoleLogEntry, 0, len(entries))
	for _, entry := range entries {
		level := strings.ToLower(strings.TrimSpace(entry.Level))
		switch level {
		case "log", "info", "warn", "error", "debug":
			// keep as-is
		default:
			level = "log"
		}

		source := strings.ToLower(strings.TrimSpace(entry.Source))
		if source != "console" && source != "runtime" {
			source = "console"
		}

		text := strings.TrimSpace(entry.Text)
		if text == "" {
			text = "(no message supplied)"
		}

		sanitized = append(sanitized, IssueConsoleLogEntry{
			Timestamp: entry.Timestamp,
			Level:     level,
			Source:    source,
			Text:      truncateString(text, maxText),
		})
	}

	if len(sanitized) > maxEntries {
		sanitized = sanitized[len(sanitized)-maxEntries:]
	}

	return sanitized
}

func sanitizeNetworkRequests(entries []IssueNetworkEntry, maxEntries, maxURLLength, maxErrLength, maxIDLength int) []IssueNetworkEntry {
	if len(entries) == 0 {
		return nil
	}

	sanitized := make([]IssueNetworkEntry, 0, len(entries))
	for _, entry := range entries {
		kind := strings.ToLower(strings.TrimSpace(entry.Kind))
		if kind != "fetch" && kind != "xhr" {
			kind = "fetch"
		}

		method := strings.ToUpper(strings.TrimSpace(entry.Method))
		if method == "" {
			method = "GET"
		}

		urlValue := strings.TrimSpace(entry.URL)
		if urlValue == "" {
			urlValue = "(unknown URL)"
		}
		urlValue = truncateString(urlValue, maxURLLength)

		errorText := truncateString(strings.TrimSpace(entry.Error), maxErrLength)

		var statusPtr *int
		if entry.Status != nil {
			val := *entry.Status
			if val >= 0 {
				statusPtr = intPtr(val)
			}
		}

		var okPtr *bool
		if entry.OK != nil {
			okPtr = boolPtr(*entry.OK)
		}

		var durationPtr *int
		if entry.DurationMs != nil {
			val := *entry.DurationMs
			if val < 0 {
				val = 0
			}
			durationPtr = intPtr(val)
		}

		requestID := truncateString(strings.TrimSpace(entry.RequestID), maxIDLength)

		sanitized = append(sanitized, IssueNetworkEntry{
			Timestamp:  entry.Timestamp,
			Kind:       kind,
			Method:     method,
			URL:        urlValue,
			Status:     statusPtr,
			OK:         okPtr,
			DurationMs: durationPtr,
			Error:      errorText,
			RequestID:  requestID,
		})
	}

	if len(sanitized) > maxEntries {
		sanitized = sanitized[len(sanitized)-maxEntries:]
	}

	return sanitized
}

func sanitizeHealthChecks(entries []IssueHealthCheckEntry, maxEntries, maxNameLength, maxEndpointLength, maxMessageLength, maxCodeLength, maxResponseLength int) []IssueHealthCheckEntry {
	if len(entries) == 0 {
		return nil
	}

	sanitized := make([]IssueHealthCheckEntry, 0, len(entries))
	for _, entry := range entries {
		id := strings.TrimSpace(entry.ID)
		if id == "" {
			id = strings.TrimSpace(entry.Name)
		}
		if id == "" {
			id = "health-check"
		} else {
			id = truncateString(id, maxNameLength)
		}

		name := strings.TrimSpace(entry.Name)
		if name == "" {
			name = "Health Check"
		} else {
			name = truncateString(name, maxNameLength)
		}

		status := strings.ToLower(strings.TrimSpace(entry.Status))
		switch status {
		case "pass", "ok", "healthy":
			status = "pass"
		case "warn", "warning", "degraded":
			status = "warn"
		case "fail", "failed", "error", "critical", "unhealthy":
			status = "fail"
		default:
			status = "warn"
		}

		endpoint := strings.TrimSpace(entry.Endpoint)
		if endpoint != "" {
			endpoint = truncateString(endpoint, maxEndpointLength)
		}

		message := strings.TrimSpace(entry.Message)
		if message != "" {
			message = truncateString(message, maxMessageLength)
		}

		code := strings.TrimSpace(entry.Code)
		if code != "" {
			code = truncateString(code, maxCodeLength)
		}

		response := strings.TrimSpace(entry.Response)
		if response != "" {
			response = truncateString(response, maxResponseLength)
		}

		var latency *int
		if entry.LatencyMs != nil && *entry.LatencyMs >= 0 {
			value := *entry.LatencyMs
			latency = &value
		}

		sanitized = append(sanitized, IssueHealthCheckEntry{
			ID:        id,
			Name:      name,
			Status:    status,
			Endpoint:  endpoint,
			LatencyMs: latency,
			Message:   message,
			Code:      code,
			Response:  response,
		})
	}

	if len(sanitized) > maxEntries {
		sanitized = sanitized[:maxEntries]
	}

	return sanitized
}

func sanitizeIssueCaptures(entries []IssueCapture, maxEntries, maxNoteLength, maxLabelLength, maxTextLength int) []IssueCapture {
	if len(entries) == 0 {
		return nil
	}

	sanitized := make([]IssueCapture, 0, len(entries))
	seen := make(map[string]struct{})

	for _, entry := range entries {
		if len(sanitized) >= maxEntries {
			break
		}

		data := strings.TrimSpace(entry.Data)
		if data == "" {
			continue
		}

		// Validate and size-check base64 data
		// Support both raw base64 and data URLs (data:image/png;base64,...)
		base64Payload := data
		if strings.HasPrefix(data, "data:") {
			if idx := strings.Index(data, ","); idx > 0 {
				base64Payload = data[idx+1:]
			}
		}

		// Reject oversized captures (>3 MiB base64 = ~2.25 MiB decoded)
		if len(base64Payload) > 3*1024*1024 {
			logger.Warn(fmt.Sprintf("capture too large (%d bytes base64), skipping", len(base64Payload)))
			continue
		}

		if _, err := base64.StdEncoding.DecodeString(base64Payload); err != nil {
			logger.Warn("invalid capture data provided, ignoring element capture", err)
			continue
		}

		captureID := strings.TrimSpace(entry.ID)
		if captureID == "" {
			captureID = fmt.Sprintf("capture-%d", len(sanitized)+1)
		}
		if _, exists := seen[captureID]; exists {
			captureID = fmt.Sprintf("%s-%d", captureID, len(sanitized)+1)
		}
		seen[captureID] = struct{}{}

		mode := strings.TrimSpace(entry.Mode)
		kind := strings.ToLower(strings.TrimSpace(entry.Type))
		if kind != "element" && kind != "page" {
			kind = "element"
		}

		sanitizedCapture := IssueCapture{
			ID:        captureID,
			Type:      kind,
			Width:     clampCaptureDimension(entry.Width),
			Height:    clampCaptureDimension(entry.Height),
			Data:      data,
			Note:      truncateString(strings.TrimSpace(entry.Note), maxNoteLength),
			Selector:  truncateString(strings.TrimSpace(entry.Selector), maxLabelLength),
			TagName:   truncateString(strings.TrimSpace(entry.TagName), maxLabelLength),
			ElementID: truncateString(strings.TrimSpace(entry.ElementID), maxLabelLength),
			Label:     truncateString(strings.TrimSpace(entry.Label), maxLabelLength),
			AriaDesc:  truncateString(strings.TrimSpace(entry.AriaDesc), maxLabelLength),
			Title:     truncateString(strings.TrimSpace(entry.Title), maxLabelLength),
			Role:      truncateString(strings.TrimSpace(entry.Role), 60),
			Text:      truncateString(strings.TrimSpace(entry.Text), maxTextLength),
			Mode:      truncateString(mode, 32),
			Filename:  truncateString(strings.TrimSpace(entry.Filename), 120),
			CreatedAt: sanitizeCaptureTimestamp(entry.CreatedAt),
		}

		if classes := sanitizeCaptureClasses(entry.Classes, 8, maxLabelLength); len(classes) > 0 {
			sanitizedCapture.Classes = classes
		}
		sanitizedCapture.BoundingBox = sanitizeCaptureBox(entry.BoundingBox)
		sanitizedCapture.Clip = sanitizeCaptureBox(entry.Clip)

		sanitized = append(sanitized, sanitizedCapture)
	}

	return sanitized
}

func sanitizeCaptureBox(box *IssueCaptureBox) *IssueCaptureBox {
	if box == nil {
		return nil
	}

	sanitized := &IssueCaptureBox{
		X:      roundFloat(box.X, 2),
		Y:      roundFloat(box.Y, 2),
		Width:  roundFloat(box.Width, 2),
		Height: roundFloat(box.Height, 2),
	}

	if sanitized.Width < 0 {
		sanitized.Width = 0
	}
	if sanitized.Height < 0 {
		sanitized.Height = 0
	}

	return sanitized
}

func sanitizeCaptureTimestamp(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if _, err := time.Parse(time.RFC3339, trimmed); err != nil {
		return ""
	}
	return trimmed
}

func clampCaptureDimension(value int) int {
	if value < 0 {
		return 0
	}
	if value > 20000 {
		return 20000
	}
	return value
}

func roundFloat(value float64, precision int) float64 {
	if precision <= 0 {
		return math.Round(value)
	}
	factor := math.Pow(10, float64(precision))
	return math.Round(value*factor) / factor
}

// =============================================================================
// Title and Description Builders
// =============================================================================

func deriveIssueTitle(primaryDescription, message string, captures []IssueCapture, includeDiagnosticsSummary bool, includeScreenshot bool) string {
	trimmedPrimary := strings.TrimSpace(primaryDescription)
	hasPrimary := trimmedPrimary != ""

	notedCaptures := make([]IssueCapture, 0, len(captures))
	for _, capture := range captures {
		if strings.ToLower(strings.TrimSpace(capture.Type)) != "element" {
			continue
		}
		if strings.TrimSpace(capture.Note) == "" {
			continue
		}
		notedCaptures = append(notedCaptures, capture)
	}

	captureCount := len(notedCaptures)
	firstCaptureLabel := ""
	if captureCount > 0 {
		firstCaptureLabel = resolveCaptureLabel(notedCaptures[0], 0)
	}

	if includeDiagnosticsSummary {
		if !hasPrimary && captureCount == 0 {
			// Try to extract specific diagnostic issue from message
			lowerMessage := strings.ToLower(message)
			if strings.Contains(lowerMessage, "bridge") || strings.Contains(lowerMessage, "iframe") {
				return "Fix iframe bridge compliance issues"
			}
			if strings.Contains(lowerMessage, "localhost") || strings.Contains(lowerMessage, "proxy") {
				return "Fix localhost proxy bypass violations"
			}
			return "Address diagnostics findings"
		}
		if hasPrimary && captureCount == 0 {
			return "Address diagnostics findings and feedback"
		}
		if !hasPrimary && captureCount == 1 && firstCaptureLabel != "" {
			return truncateTitle("Diagnostics issues and feedback on "+firstCaptureLabel, reportTitleMaxLength)
		}
		if !hasPrimary && captureCount > 1 {
			return fmt.Sprintf("Diagnostics issues with feedback on %d elements", captureCount)
		}
		if hasPrimary && captureCount == 1 && firstCaptureLabel != "" {
			return truncateTitle("Address diagnostics issues and "+firstCaptureLabel+" feedback", reportTitleMaxLength)
		}
		if hasPrimary && captureCount > 1 {
			return "Address diagnostics issues with captured feedback"
		}
	}

	if !hasPrimary && captureCount > 0 {
		if captureCount == 1 && firstCaptureLabel != "" {
			return truncateTitle("Feedback on "+firstCaptureLabel, reportTitleMaxLength)
		}
		return fmt.Sprintf("Feedback on %d elements", captureCount)
	}

	if hasPrimary {
		if line := firstLine(trimmedPrimary); line != "" {
			return truncateTitle(line, reportTitleMaxLength)
		}
	}

	if includeScreenshot {
		return "Screenshot feedback"
	}

	if line := firstLine(message); line != "" {
		return truncateTitle(line, reportTitleMaxLength)
	}

	return "Issue reported from App Monitor"
}

func buildIssueDescription(
	appName, scenarioName, previewURL, source, message, screenshotData string,
	captures []IssueCapture,
	reportedAt time.Time,
	logs []string,
	logsTotal int,
	logsCapturedAt string,
	consoleLogs []IssueConsoleLogEntry,
	consoleTotal int,
	consoleCapturedAt string,
	network []IssueNetworkEntry,
	networkTotal int,
	networkCapturedAt string,
	health []IssueHealthCheckEntry,
	healthTotal int,
	healthCapturedAt string,
	statusLines []string,
	statusLabel string,
	statusCapturedAt string,
	statusSeverity string,
	repoRoot string,
) string {
	var builder strings.Builder

	scenarioPath := filepath.Join(repoRoot, "scenarios", scenarioName)
	safeScenarioName := sanitizeScenarioIdentifier(scenarioName)
	previewPath := extractPreviewPath(previewURL)
	screenshotCommand := fmt.Sprintf(
		"resource-browserless screenshot --scenario %q --output /tmp/%s-validation.png --fullpage",
		scenarioName,
		safeScenarioName,
	)
	if previewPath != "" {
		screenshotCommand = fmt.Sprintf(
			"resource-browserless screenshot --scenario %q --path %q --output /tmp/%s-validation.png --fullpage",
			scenarioName,
			previewPath,
			safeScenarioName,
		)
	}

	builder.WriteString("\nWhile reviewing this scenario in the app-monitor, an issue or missing capability was observed. Investigate the details below and implement the necessary change(s) in the scenario.\n")

	builder.WriteString("## Context\n\n")
	builder.WriteString(fmt.Sprintf("- Scenario: `%s`\n", scenarioName))
	builder.WriteString(fmt.Sprintf("- App Name: `%s`\n", appName))
	if trimmed := strings.TrimSpace(statusLabel); trimmed != "" {
		builder.WriteString(fmt.Sprintf("- Scenario status: %s\n", trimmed))
	}
	if trimmedSeverity := strings.TrimSpace(statusSeverity); trimmedSeverity != "" {
		builder.WriteString(fmt.Sprintf("- Status severity: %s\n", strings.ToUpper(trimmedSeverity)))
	}
	if previewURL != "" {
		builder.WriteString(fmt.Sprintf("- Preview URL: %s\n", previewURL))
	} else {
		builder.WriteString(fmt.Sprintf("- Preview URL: _Not captured_. Launch with:\n  ```bash\n  cd %s && make start\n  ```\n", scenarioPath))
	}
	if source != "" {
		builder.WriteString(fmt.Sprintf("- Reported via: %s\n", source))
	}
	builder.WriteString(fmt.Sprintf("- Reported at: %s\n", reportedAt.Format(time.RFC3339)))

	trimmedMessage := strings.TrimSpace(message)
	builder.WriteString("\n## Reporter Notes\n\n")
	if trimmedMessage != "" {
		builder.WriteString(trimmedMessage)
		builder.WriteString("\n")
	} else {
		builder.WriteString("_No additional reporter notes were provided._\n")
	}

	builder.WriteString("\n## Captured Evidence\n\n")
	builder.WriteString(formatEvidenceLine("Lifecycle logs", len(logs), logsTotal, logsCapturedAt, attachmentLifecycleName))
	builder.WriteString("\n")
	builder.WriteString(formatEvidenceLine("Console events", len(consoleLogs), consoleTotal, consoleCapturedAt, attachmentConsoleName))
	builder.WriteString("\n")
	builder.WriteString(formatEvidenceLine("Network requests", len(network), networkTotal, networkCapturedAt, attachmentNetworkName))
	builder.WriteString("\n")
	builder.WriteString(formatEvidenceLine("Health checks", len(health), healthTotal, healthCapturedAt, attachmentHealthName))
	builder.WriteString("\n")
	builder.WriteString(formatEvidenceLine("App status", len(statusLines), len(statusLines), statusCapturedAt, attachmentStatusName))
	builder.WriteString("\n")
	if screenshotData != "" {
		builder.WriteString(fmt.Sprintf("- Screenshot attached as `%s` (base64 PNG).\n", attachmentScreenshotName))
	} else {
		builder.WriteString("- Screenshot: not captured for this report.\n")
	}

	if summaries := buildCaptureSummaries(captures); len(summaries) > 0 {
		builder.WriteString("\n### Element Captures\n\n")
		for _, summary := range summaries {
			builder.WriteString(summary)
		}
	}

	builder.WriteString("\nUse the artifacts above to reproduce the behavior before making any changes.\n")

	builder.WriteString("\n## Investigation Checklist\n\n")
	builder.WriteString(fmt.Sprintf("1. Open the scenario locally: `cd %s`.\n", scenarioPath))
	builder.WriteString("2. Review the Reporter Notes and attachments to understand the observed behavior.\n")
	builder.WriteString("3. Reproduce the issue in the preview environment (use the captured Preview URL if present).\n")
	builder.WriteString("4. Identify the root cause in the scenario's code or configuration before implementation.\n")

	builder.WriteString("\n## Implementation Guardrails\n\n")
	builder.WriteString(fmt.Sprintf("- Modify only files under `%s/`.\n", scenarioPath))
	builder.WriteString("- Do not run git commands (commit, push, rebase, etc.).\n")
	builder.WriteString("- Coordinate changes through the scenario's lifecycle commands; avoid editing shared resources unless explicitly required.\n")

	builder.WriteString("\n## Testing & Validation\n\n")
	builder.WriteString(fmt.Sprintf("- Run scenario tests:\n  ```bash\n  cd %s && make test\n  ```\n", scenarioPath))
	builder.WriteString(fmt.Sprintf("- Restart the scenario so App Monitor picks up the update:\n  ```bash\n  vrooli scenario restart %s\n  ```\n", scenarioName))
	builder.WriteString("- Reproduce the original preview flow in App Monitor to confirm the issue is resolved.\n")

	builder.WriteString("\n## Visual Validation\n\n")
	builder.WriteString("- Capture before/after screenshots when visual confirmation is needed. Browserless can target the running scenario directly:\n")
	builder.WriteString("  ```bash\n")
	builder.WriteString(fmt.Sprintf("  %s\n", screenshotCommand))
	builder.WriteString("  ```\n")
	if previewPath == "" {
		builder.WriteString(fmt.Sprintf("  Use `--path /route` if you need a specific page within `%s`.\n", scenarioName))
	}
	builder.WriteString("- Attach relevant screenshots to this issue so future reviewers can validate the UI changes quickly.\n")
	builder.WriteString("- If App Monitor still shows cached content, rerun the scenario lifecycle commands above before taking the screenshot.\n")

	builder.WriteString("\n## Success Criteria\n\n")
	builder.WriteString("- [ ] Issue no longer occurs (or requested improvement is present) when using the App Monitor preview.\n")
	builder.WriteString(fmt.Sprintf("- [ ] `make test` passes for `%s`.\n", scenarioPath))
	builder.WriteString("- [ ] Any new logs are clean (no unexpected errors).\n")
	builder.WriteString("- [ ] Scenario lifecycle commands complete without errors (`make start`, `make stop`).\n")
	builder.WriteString("- [ ] **Scenario remains RUNNING after fix validation.**\n")
	builder.WriteString("- [ ] **Health checks continue to pass after changes.**\n")

	builder.WriteString("\n## Completion Notes\n\n")
	builder.WriteString("Document the fix in the issue, including the files touched and any follow-up work that may be required. Attach updated artifacts if additional evidence is gathered during the fix.\n")

	return builder.String()
}

func formatEvidenceLine(label string, capturedCount, reportedTotal int, capturedAt, attachmentName string) string {
	var builder strings.Builder
	if capturedCount <= 0 && reportedTotal <= 0 {
		builder.WriteString(fmt.Sprintf("- %s: not captured in this report.", label))
		return builder.String()
	}

	reported := reportedTotal
	if reported < capturedCount {
		reported = capturedCount
	}

	if capturedCount > 0 {
		builder.WriteString(fmt.Sprintf("- %s: %d captured", label, capturedCount))
		if reported > capturedCount {
			builder.WriteString(fmt.Sprintf(" (subset of %d)", reported))
		}
		builder.WriteString(fmt.Sprintf(" — see `%s`.", attachmentName))
	} else {
		builder.WriteString(fmt.Sprintf("- %s: report indicated %d available, but none were attached.", label, reported))
	}

	if capturedAt != "" {
		builder.WriteString(fmt.Sprintf(" Captured at %s.", parseOrEchoTimestamp(capturedAt)))
	}

	return builder.String()
}

func buildIssueTags(scenarioName string) []string {
	tags := []string{"app-monitor"}
	if trimmed := strings.TrimSpace(scenarioName); trimmed != "" {
		tags = append(tags, trimmed)
	}
	return dedupeStrings(tags)
}

func buildIssueEnvironment(appID, appName, previewURL, source string, reportedAt time.Time) map[string]string {
	environment := map[string]string{
		"app_id":      appID,
		"app_name":    appName,
		"reported_at": reportedAt.Format(time.RFC3339),
	}

	if previewURL != "" {
		environment["preview_url"] = previewURL
	}

	if source != "" {
		environment["source"] = source
	}

	return environment
}

// =============================================================================
// Capture Formatting
// =============================================================================

func resolveCaptureLabel(capture IssueCapture, index int) string {
	// Priority 1: CSS selector (most semantic for debugging - prioritize over Label which might be auto-generated text)
	if selector := strings.TrimSpace(capture.Selector); selector != "" {
		// For short selectors, use as-is
		if len(selector) <= reportLabelMaxLength {
			return selector
		}
		// For long selectors, try to extract meaningful ID
		if elementID := strings.TrimSpace(capture.ElementID); elementID != "" {
			return fmt.Sprintf("#%s", elementID)
		}
		// Truncate long selector
		return truncateTitle(selector, reportLabelMaxLength)
	}

	// Priority 2: Element ID (if selector wasn't available but ID is)
	if elementID := strings.TrimSpace(capture.ElementID); elementID != "" {
		return fmt.Sprintf("#%s", elementID)
	}

	// Priority 3: User's custom label (only if it looks intentional/semantic, not auto-generated text)
	if label := strings.TrimSpace(capture.Label); label != "" {
		// Use label if it's short and doesn't look like element text content
		// (i.e., doesn't contain symbols like +−, ©, |, typical of UI text)
		if len(label) <= 40 && !strings.ContainsAny(label, "©®™±×÷+−|•◦▪▫") {
			return truncateTitle(label, reportLabelMaxLength)
		}
	}

	// Priority 4: ARIA description (accessibility label)
	if aria := strings.TrimSpace(capture.AriaDesc); aria != "" {
		return truncateTitle(aria, reportLabelMaxLength)
	}

	// Priority 5: Title attribute
	if title := strings.TrimSpace(capture.Title); title != "" {
		return truncateTitle(title, reportLabelMaxLength)
	}

	// Priority 6: Role + first class (semantic structure)
	role := strings.TrimSpace(capture.Role)
	if role != "" {
		if len(capture.Classes) > 0 {
			firstClass := strings.TrimSpace(capture.Classes[0])
			if firstClass != "" {
				combined := fmt.Sprintf("%s.%s", role, firstClass)
				if len(combined) <= reportLabelMaxLength {
					return combined
				}
			}
		}
		return truncateTitle(role, 32)
	}

	// Priority 7: Tag name + first class
	tagName := strings.TrimSpace(capture.TagName)
	if tagName != "" {
		if len(capture.Classes) > 0 {
			firstClass := strings.TrimSpace(capture.Classes[0])
			if firstClass != "" {
				combined := fmt.Sprintf("<%s.%s>", strings.ToLower(tagName), firstClass)
				if len(combined) <= reportLabelMaxLength {
					return combined
				}
			}
		}
		return fmt.Sprintf("<%s>", strings.ToLower(tagName))
	}

	// Priority 8: Use Label even if it looks like text (better than nothing)
	if label := strings.TrimSpace(capture.Label); label != "" {
		truncated := truncateTitle(label, 28) // Leave room for " element"
		return truncated + " element"
	}

	// Priority 9 (LAST RESORT): Truncated text content with indicator
	if text := strings.TrimSpace(capture.Text); text != "" {
		truncated := truncateTitle(text, 28) // Leave room for " element"
		return truncated + " element"
	}

	return fmt.Sprintf("element %d", index+1)
}

func buildCaptureSummaries(captures []IssueCapture) []string {
	if len(captures) == 0 {
		return nil
	}

	summaries := make([]string, 0, len(captures))
	for idx, capture := range captures {
		if capture.Type == "page" && capture.Note == "" && capture.Mode == "" && capture.Clip == nil {
			continue
		}
		summaries = append(summaries, formatCaptureSummary(capture, idx+1))
	}

	return summaries
}

func formatCaptureSummary(capture IssueCapture, index int) string {
	var builder strings.Builder
	label := strings.TrimSpace(capture.Label)
	selector := strings.TrimSpace(capture.Selector)
	if label == "" {
		label = selector
	}
	if label == "" {
		if capture.Type == "page" {
			label = fmt.Sprintf("Preview capture #%d", index)
		} else {
			label = fmt.Sprintf("Element capture #%d", index)
		}
	}

	builder.WriteString("- ")
	builder.WriteString(label)

	details := make([]string, 0, 5)
	if selector != "" && selector != label {
		details = append(details, fmt.Sprintf("selector `%s`", selector))
	}
	if capture.Width > 0 && capture.Height > 0 {
		details = append(details, fmt.Sprintf("%dx%d px", capture.Width, capture.Height))
	}
	if capture.Role != "" {
		details = append(details, fmt.Sprintf("role %s", capture.Role))
	}
	if len(capture.Classes) > 0 {
		classes := strings.Join(capture.Classes, ", ")
		details = append(details, fmt.Sprintf("classes %s", classes))
	}
	if capture.Mode != "" && capture.Mode != "full" {
		details = append(details, fmt.Sprintf("mode %s", capture.Mode))
	}

	if len(details) > 0 {
		builder.WriteString(" • ")
		builder.WriteString(strings.Join(details, " • "))
	}

	if capture.Note != "" {
		builder.WriteString("\n  Note: ")
		builder.WriteString(capture.Note)
	}

	if capture.Text != "" {
		builder.WriteString("\n  Text: ")
		builder.WriteString(capture.Text)
	}

	if capture.BoundingBox != nil {
		builder.WriteString("\n  Bounding box: ")
		builder.WriteString(fmt.Sprintf("%.0fx%.0f at (%.0f, %.0f)", capture.BoundingBox.Width, capture.BoundingBox.Height, capture.BoundingBox.X, capture.BoundingBox.Y))
	}

	if capture.CreatedAt != "" {
		builder.WriteString("\n  Captured at: ")
		builder.WriteString(capture.CreatedAt)
	}

	builder.WriteString("\n")
	return builder.String()
}

// =============================================================================
// Helper Functions
// =============================================================================

func countConsoleErrors(logs []IssueConsoleLogEntry) int {
	count := 0
	for _, log := range logs {
		if strings.ToLower(strings.TrimSpace(log.Level)) == "error" {
			count++
		}
	}
	return count
}

func countFailedRequests(requests []IssueNetworkEntry) int {
	count := 0
	for _, req := range requests {
		if (req.OK != nil && !*req.OK) || (req.Status != nil && *req.Status >= 400) {
			count++
		}
	}
	return count
}

func getFirstFailedRequest(requests []IssueNetworkEntry) *IssueNetworkEntry {
	for i := range requests {
		req := &requests[i]
		if (req.OK != nil && !*req.OK) || (req.Status != nil && *req.Status >= 400) {
			return req
		}
	}
	return nil
}

func countPassingHealthChecks(checks []IssueHealthCheckEntry) int {
	count := 0
	for _, check := range checks {
		if strings.ToLower(strings.TrimSpace(check.Status)) == "pass" {
			count++
		}
	}
	return count
}

func normalizePreviewURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return ""
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return ""
	}

	return parsed.String()
}

func extractPreviewPath(previewURL string) string {
	trimmed := strings.TrimSpace(previewURL)
	if trimmed == "" {
		return ""
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return ""
	}

	path := strings.TrimSpace(parsed.Path)
	if path == "" || path == "/" {
		path = ""
	} else if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	if strings.HasPrefix(path, "/apps/") {
		if idx := strings.Index(path, "/proxy"); idx >= 0 {
			path = path[idx+len("/proxy"):]
			if path != "" && !strings.HasPrefix(path, "/") {
				path = "/" + path
			}
		}
	}

	if path == "" || path == "/" {
		path = ""
	}

	if path != "" && parsed.RawQuery != "" {
		path = fmt.Sprintf("%s?%s", path, parsed.RawQuery)
	}
	if path != "" && parsed.Fragment != "" {
		path = fmt.Sprintf("%s#%s", path, parsed.Fragment)
	}

	return path
}

func sanitizeScenarioIdentifier(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "scenario"
	}

	trimmed = strings.ToLower(trimmed)
	var builder strings.Builder
	for _, r := range trimmed {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r == '-' || r == '_':
			builder.WriteRune(r)
		case r == ' ' || r == '/' || r == '\\':
			builder.WriteRune('-')
		default:
			builder.WriteRune('-')
		}
	}

	result := builder.String()
	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}
	result = strings.Trim(result, "-")
	if result == "" {
		return "scenario"
	}

	return result
}

func truncateTitle(value string, limit int) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	runes := []rune(trimmed)
	if len(runes) <= limit {
		return trimmed
	}
	return string(runes[:limit]) + "..."
}

func firstLine(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	for index, r := range trimmed {
		if r == '\n' || r == '\r' {
			return strings.TrimSpace(trimmed[:index])
		}
	}
	return trimmed
}
