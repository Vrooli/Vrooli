package services

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"app-monitor-api/repository"
)

// =============================================================================
// Scenario Status Diagnostics
// =============================================================================

// GetAppScenarioStatus executes vrooli scenario status for detailed diagnostics.
// Returns the raw, agent-optimized CLI output instead of re-interpreting it.
func (s *AppService) GetAppScenarioStatus(ctx context.Context, appID string) (*AppScenarioStatus, error) {
	id := strings.TrimSpace(appID)
	if id == "" {
		return nil, ErrAppIdentifierRequired
	}

	app, err := s.GetApp(ctx, id)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			return nil, fmt.Errorf("%w: %v", ErrAppNotFound, err)
		}
		return nil, err
	}

	scenarioName := strings.TrimSpace(app.ScenarioName)
	if scenarioName == "" {
		scenarioName = strings.TrimSpace(app.Name)
	}
	if scenarioName == "" {
		scenarioName = id
	}

	commandIdentifier := resolveScenarioCommandIdentifier(app, id)
	if commandIdentifier == "" {
		commandIdentifier = scenarioName
	}

	// Execute vrooli scenario status WITHOUT --json to get raw, agent-optimized text output
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctxWithTimeout, "vrooli", "scenario", "status", commandIdentifier)
	cmd.Env = append(os.Environ(), "TERM=dumb")
	output, cmdErr := cmd.CombinedOutput()
	if cmdErr != nil {
		trimmed := strings.TrimSpace(string(output))
		if trimmed != "" {
			return nil, fmt.Errorf("failed to execute vrooli scenario status: %s", trimmed)
		}
		return nil, fmt.Errorf("failed to execute vrooli scenario status: %w", cmdErr)
	}

	// Split raw output into lines for Details field
	rawOutput := strings.TrimSpace(string(output))
	details := strings.Split(rawOutput, "\n")

	// Also fetch JSON for metadata extraction (ports, severity, etc)
	cmdJSON := exec.CommandContext(ctxWithTimeout, "vrooli", "scenario", "status", commandIdentifier, "--json")
	cmdJSON.Env = append(os.Environ(), "TERM=dumb")
	jsonOutput, jsonErr := cmdJSON.CombinedOutput()

	// Default values in case JSON parsing fails
	statusLabel := "UNKNOWN"
	severity := ScenarioStatusSeverityWarn
	ports := make(map[string]int)
	var recs []string
	capturedAt := s.timeNow().UTC().Format(time.RFC3339)
	runtime := ""
	processCount := 0

	// Parse JSON if available for structured metadata
	if jsonErr == nil {
		cleanJSON, extractErr := extractFirstJSONDocument(jsonOutput)
		if extractErr == nil {
			var resp scenarioStatusCLIResponse
			if err := json.Unmarshal(cleanJSON, &resp); err == nil {
				statusValue := strings.TrimSpace(resp.ScenarioData.Status)
				if statusValue == "" {
					statusValue = strings.TrimSpace(resp.RawResponse.Data.Status)
				}
				statusLabel, severity = formatScenarioStatusLabel(statusValue)
				processCount = len(resp.ScenarioData.Processes)
				if processCount == 0 && severity == ScenarioStatusSeverityOK {
					severity = ScenarioStatusSeverityWarn
				}

				for key, value := range resp.ScenarioData.AllocatedPorts {
					ports[strings.ToUpper(strings.TrimSpace(key))] = value
				}

				runtime = strings.TrimSpace(resp.ScenarioData.Runtime)

				recs = resp.Recommendations
				if resp.TestInfrastructure.Overall != nil {
					recs = append(recs, resp.TestInfrastructure.Overall.Recommendations...)
					if resp.TestInfrastructure.Overall.Recommendation != "" {
						recs = append(recs, resp.TestInfrastructure.Overall.Recommendation)
					}
				}
				recs = dedupeStrings(filterNonEmptyStrings(recs))

				if ts := strings.TrimSpace(resp.Metadata.Timestamp); ts != "" {
					capturedAt = ts
				}
			}
		}
	}

	return &AppScenarioStatus{
		AppID:           app.ID,
		Scenario:        scenarioName,
		CapturedAt:      capturedAt,
		StatusLabel:     statusLabel,
		Severity:        severity,
		Runtime:         runtime,
		ProcessCount:    processCount,
		Ports:           ports,
		Recommendations: recs,
		Details:         details,
	}, nil
}

func resolveScenarioCommandIdentifier(app *repository.App, fallback string) string {
	var candidates []string
	if app != nil {
		candidates = append(candidates,
			sanitizeCommandIdentifier(app.ID),
			sanitizeCommandIdentifier(app.ScenarioName),
			sanitizeCommandIdentifier(app.Name),
		)
	}
	candidates = append(candidates, sanitizeCommandIdentifier(fallback))

	for _, candidate := range candidates {
		if candidate != "" {
			return candidate
		}
	}

	var rawCandidates []string
	if app != nil {
		rawCandidates = append(rawCandidates,
			strings.TrimSpace(app.ID),
			strings.TrimSpace(app.ScenarioName),
			strings.TrimSpace(app.Name),
		)
	}
	rawCandidates = append(rawCandidates, strings.TrimSpace(fallback))

	for _, candidate := range rawCandidates {
		if candidate != "" {
			return candidate
		}
	}

	return ""
}

func extractFirstJSONDocument(output []byte) ([]byte, error) {
	trimmed := bytes.TrimSpace(output)
	if len(trimmed) == 0 {
		return nil, errors.New("empty response")
	}

	start := -1
	for i, b := range trimmed {
		if b == '{' || b == '[' {
			start = i
			break
		}
	}
	if start == -1 {
		return nil, errors.New("no JSON document found")
	}

	data := trimmed[start:]
	depth := 0
	inString := false
	escaped := false

	for i, b := range data {
		if escaped {
			escaped = false
			continue
		}

		if inString {
			switch b {
			case '\\':
				escaped = true
			case '"':
				inString = false
			}
			continue
		}

		switch b {
		case '"':
			inString = true
		case '{', '[':
			depth++
		case '}', ']':
			depth--
			if depth == 0 {
				return data[:i+1], nil
			}
		}
	}

	return nil, errors.New("incomplete JSON document")
}

// =============================================================================
// Health Check Diagnostics
// =============================================================================

// CheckAppHealth queries the previewed scenario's /health endpoints for API and UI services.
func (s *AppService) CheckAppHealth(ctx context.Context, appID string) (*AppHealthDiagnostics, error) {
	id := strings.TrimSpace(appID)
	if id == "" {
		return nil, ErrAppIdentifierRequired
	}

	app, err := s.GetApp(ctx, id)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			return nil, fmt.Errorf("%w: %v", ErrAppNotFound, err)
		}
		return nil, err
	}

	displayName := strings.TrimSpace(app.Name)
	if displayName == "" {
		displayName = strings.TrimSpace(app.ScenarioName)
	}
	if displayName == "" {
		displayName = id
	}

	apiPort := resolveAppPort(app, []string{"api", "api_port", "backend", "server", "API", "API_PORT"})
	uiPort := resolveAppPort(app, []string{"ui", "ui_port", "frontend", "web", "UI", "UI_PORT", "WEB", "WEB_PORT"})

	ports := make(map[string]int)
	if apiPort > 0 {
		ports["api"] = apiPort
	}
	if uiPort > 0 {
		ports["ui"] = uiPort
	}

	checks := make([]IssueHealthCheckEntry, 0, 2)
	var diagnosticsNotes []string

	if apiPort > 0 {
		endpoint := buildLocalAPIURL(apiPort, "/health")
		entry, notes := s.executeScenarioHealthCheck(ctx, "api", fmt.Sprintf("%s API", displayName), endpoint)
		checks = append(checks, entry)
		diagnosticsNotes = append(diagnosticsNotes, notes...)
	} else {
		message := "API port unavailable for previewed app; unable to query /health."
		checks = append(checks, IssueHealthCheckEntry{
			ID:       "api",
			Name:     fmt.Sprintf("%s API", displayName),
			Status:   "fail",
			Endpoint: "",
			Message:  message,
			Code:     "port_missing",
		})
		diagnosticsNotes = append(diagnosticsNotes, message)
	}

	if uiPort > 0 {
		endpoint := buildLocalAPIURL(uiPort, "/health")
		entry, notes := s.executeScenarioHealthCheck(ctx, "ui", fmt.Sprintf("%s UI", displayName), endpoint)
		checks = append(checks, entry)
		diagnosticsNotes = append(diagnosticsNotes, notes...)
	} else {
		message := "UI port unavailable for previewed app; unable to query /health."
		checks = append(checks, IssueHealthCheckEntry{
			ID:       "ui",
			Name:     fmt.Sprintf("%s UI", displayName),
			Status:   "fail",
			Endpoint: "",
			Message:  message,
			Code:     "port_missing",
		})
		diagnosticsNotes = append(diagnosticsNotes, message)
	}

	for i := range checks {
		if checks[i].Status == "" {
			checks[i].Status = "fail"
		}
	}

	diagnostics := &AppHealthDiagnostics{
		AppID:      app.ID,
		AppName:    strings.TrimSpace(app.Name),
		Scenario:   strings.TrimSpace(app.ScenarioName),
		CapturedAt: s.timeNow().UTC().Format(time.RFC3339),
		Checks:     checks,
	}
	if len(ports) > 0 {
		diagnostics.Ports = ports
	}
	if notes := dedupeStrings(diagnosticsNotes); len(notes) > 0 {
		diagnostics.Errors = notes
	}

	return diagnostics, nil
}

// =============================================================================
// Iframe Bridge Rule Validation
// =============================================================================

// CheckIframeBridgeRule validates scenario iframe bridge compliance via scenario-auditor.
func (s *AppService) CheckIframeBridgeRule(ctx context.Context, appID string) (*BridgeDiagnosticsReport, error) {
	id := strings.TrimSpace(appID)
	if id == "" {
		return nil, ErrAppIdentifierRequired
	}

	app, err := s.GetApp(ctx, id)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			return nil, fmt.Errorf("%w: %v", ErrAppNotFound, err)
		}
		return nil, err
	}

	scenarioDisplayName := strings.TrimSpace(app.ScenarioName)
	scenarioSlug := ""

	scenarioPath := strings.TrimSpace(app.Path)
	if scenarioPath != "" {
		cleaned := filepath.Clean(scenarioPath)
		base := filepath.Base(cleaned)
		scenarioSlug = sanitizeCommandIdentifier(base)
	}

	if scenarioSlug == "" {
		scenarioSlug = sanitizeCommandIdentifier(scenarioDisplayName)
	}
	if scenarioSlug == "" {
		scenarioSlug = sanitizeCommandIdentifier(app.ID)
	}
	if scenarioSlug == "" {
		scenarioSlug = sanitizeCommandIdentifier(id)
	}
	if scenarioSlug == "" {
		scenarioSlug = "scenario"
	}

	if scenarioDisplayName == "" {
		scenarioDisplayName = strings.TrimSpace(app.Name)
	}
	if scenarioDisplayName == "" {
		scenarioDisplayName = scenarioSlug
	}

	port, err := s.locateScenarioAuditorAPIPort(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrScenarioAuditorUnavailable, err)
	}

	type auditorRuleSpec struct {
		ID   string
		Name string
	}

	rules := []auditorRuleSpec{
		{ID: "iframe_bridge_quality", Name: "Scenario UI Bridge Quality"},
		{ID: "localhost_proxy_compact", Name: "Proxy-Compatible UI Base"},
		{ID: "proxy_base_override", Name: "Proxy Base Preservation"},
		{ID: "secure_tunnel", Name: "Secure Tunnel Setup"},
		{ID: "service_ports", Name: "Ports Configuration"},
		{ID: "service_health_lifecycle", Name: "Lifecycle Health Configuration"},
	}

	results := make([]BridgeRuleReport, 0, len(rules))
	allViolations := make([]BridgeRuleViolation, 0)
	warningsSet := make(map[string]struct{})
	targetsSet := make(map[string]struct{})
	var (
		aggregatedScenario string
		aggregatedDuration int64
		aggregatedFiles    int
	)

	checkedAt := s.timeNow().UTC()

	for _, rule := range rules {
		payload := struct {
			Scenario string `json:"scenario"`
		}{Scenario: scenarioSlug}
		body, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}

		endpoint := buildLocalAPIURL(port, fmt.Sprintf("/api/v1/rules/%s/scenario-test", rule.ID))
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := s.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrScenarioAuditorUnavailable, err)
		}

		bodyBytes, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("failed to read scenario-auditor response: %w", readErr)
		}

		if resp.StatusCode == http.StatusNotFound {
			var apiErr struct {
				Error   string `json:"error"`
				Message string `json:"message"`
			}

			message := strings.TrimSpace(string(bodyBytes))
			if err := json.Unmarshal(bodyBytes, &apiErr); err == nil {
				if strings.TrimSpace(apiErr.Error) != "" {
					message = strings.TrimSpace(apiErr.Error)
				} else if strings.TrimSpace(apiErr.Message) != "" {
					message = strings.TrimSpace(apiErr.Message)
				}
			}
			if message == "" {
				message = http.StatusText(resp.StatusCode)
			}

			warningMessage := fmt.Sprintf("Scenario auditor could not evaluate %s: %s", rule.Name, message)
			warningsSet[warningMessage] = struct{}{}
			results = append(results, BridgeRuleReport{
				RuleID:       rule.ID,
				Name:         rule.Name,
				Scenario:     scenarioSlug,
				FilesScanned: 0,
				DurationMs:   0,
				Warning:      warningMessage,
				Warnings:     []string{warningMessage},
				Violations:   []BridgeRuleViolation{},
				CheckedAt:    checkedAt,
			})
			continue
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			reason := strings.TrimSpace(string(bodyBytes))
			if reason == "" {
				reason = http.StatusText(resp.StatusCode)
			}
			return nil, fmt.Errorf("scenario-auditor returned status %d: %s", resp.StatusCode, reason)
		}

		var auditorResp scenarioAuditorRuleResponse
		if decodeErr := json.Unmarshal(bodyBytes, &auditorResp); decodeErr != nil {
			return nil, fmt.Errorf("failed to decode scenario-auditor response: %w", decodeErr)
		}

		scenario := strings.TrimSpace(auditorResp.Scenario)
		if scenario == "" {
			scenario = scenarioSlug
		}

		converted := BridgeRuleReport{
			RuleID:       strings.TrimSpace(auditorResp.RuleID),
			Name:         rule.Name,
			Scenario:     scenario,
			FilesScanned: auditorResp.FilesScanned,
			DurationMs:   auditorResp.DurationMs,
			Warning:      strings.TrimSpace(auditorResp.Warning),
			Warnings:     nil,
			Targets:      auditorResp.Targets,
			Violations:   make([]BridgeRuleViolation, 0, len(auditorResp.Violations)),
			CheckedAt:    checkedAt,
		}

		if trimmed := strings.TrimSpace(auditorResp.Warning); trimmed != "" {
			converted.Warnings = []string{trimmed}
			warningsSet[trimmed] = struct{}{}
		}

		for _, target := range converted.Targets {
			target = strings.TrimSpace(target)
			if target == "" {
				continue
			}
			targetsSet[target] = struct{}{}
		}

		for _, violation := range auditorResp.Violations {
			converted.Violations = append(converted.Violations, BridgeRuleViolation{
				RuleID:         converted.RuleID,
				Type:           strings.TrimSpace(violation.Type),
				Title:          strings.TrimSpace(violation.Title),
				Description:    strings.TrimSpace(violation.Description),
				FilePath:       strings.TrimSpace(violation.FilePath),
				Line:           violation.LineNumber,
				Recommendation: strings.TrimSpace(violation.Recommendation),
				Severity:       strings.TrimSpace(violation.Severity),
				Standard:       strings.TrimSpace(violation.Standard),
			})
		}

		if converted.FilesScanned > aggregatedFiles {
			aggregatedFiles = converted.FilesScanned
		}
		aggregatedDuration += converted.DurationMs

		if aggregatedScenario == "" {
			aggregatedScenario = converted.Scenario
		}

		results = append(results, converted)
		allViolations = append(allViolations, converted.Violations...)
	}

	warnings := make([]string, 0, len(warningsSet))
	for warning := range warningsSet {
		warnings = append(warnings, warning)
	}
	if len(warnings) > 1 {
		sort.Strings(warnings)
	}

	if aggregatedScenario == "" {
		aggregatedScenario = scenarioSlug
	}

	targets := make([]string, 0, len(targetsSet))
	for target := range targetsSet {
		targets = append(targets, target)
	}
	if len(targets) > 1 {
		sort.Strings(targets)
	}

	return &BridgeDiagnosticsReport{
		Scenario:     aggregatedScenario,
		CheckedAt:    checkedAt,
		FilesScanned: aggregatedFiles,
		DurationMs:   aggregatedDuration,
		Warning:      strings.Join(warnings, "\n"),
		Warnings:     warnings,
		Targets:      targets,
		Violations:   allViolations,
		Results:      results,
	}, nil
}

// =============================================================================
// Localhost Usage Scanning
// =============================================================================

// CheckLocalhostUsage scans scenario files for hard-coded localhost references that would break proxying.
func (s *AppService) CheckLocalhostUsage(ctx context.Context, appID string) (*LocalhostUsageReport, error) {
	id := strings.TrimSpace(appID)
	if id == "" {
		return nil, ErrAppIdentifierRequired
	}

	app, err := s.GetApp(ctx, id)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			return nil, fmt.Errorf("%w: %v", ErrAppNotFound, err)
		}
		return nil, err
	}

	scenarioName := strings.TrimSpace(app.ScenarioName)
	if scenarioName == "" {
		scenarioName = strings.TrimSpace(app.ID)
	}
	if scenarioName == "" {
		scenarioName = id
	}

	root := strings.TrimSpace(app.Path)
	report := &LocalhostUsageReport{
		Scenario:  scenarioName,
		CheckedAt: s.timeNow().UTC(),
		Findings:  make([]LocalhostUsageFinding, 0),
	}

	if root == "" {
		report.Warnings = append(report.Warnings, "scenario path unknown; skipping filesystem scan")
		return report, nil
	}

	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect scenario path: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("scenario path is not a directory: %s", root)
	}

	warnings := make([]string, 0)
	scannedFiles := 0

	scanTargets := make([]string, 0, 2)
	for _, subdir := range []string{"api", "ui"} {
		candidate := filepath.Join(root, subdir)
		info, statErr := os.Stat(candidate)
		if statErr != nil {
			if !errors.Is(statErr, fs.ErrNotExist) {
				warnings = append(warnings, fmt.Sprintf("failed to inspect %s/: %v", subdir, statErr))
			}
			continue
		}
		if !info.IsDir() {
			warnings = append(warnings, fmt.Sprintf("%s exists but is not a directory; skipping", filepath.ToSlash(subdir)))
			continue
		}
		scanTargets = append(scanTargets, candidate)
	}

	if len(scanTargets) == 0 {
		warnings = append(warnings, "scenario has no api/ or ui/ directory; skipping filesystem scan")
		report.Warnings = warnings
		return report, nil
	}

	for _, scanRoot := range scanTargets {
		err = filepath.WalkDir(scanRoot, func(path string, d fs.DirEntry, walkErr error) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			if walkErr != nil {
				return walkErr
			}

			name := strings.ToLower(d.Name())
			if d.IsDir() {
				if _, skip := localhostSkipDirectories[name]; skip {
					return filepath.SkipDir
				}
				return nil
			}

			if _, skip := localhostSkipFiles[name]; skip {
				return nil
			}

			info, err := d.Info()
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("failed to stat %s: %v", path, err))
				return nil
			}

			if info.Size() > maxLocalhostScanFileSize {
				relative, relErr := filepath.Rel(root, path)
				if relErr != nil {
					relative = path
				}
				warnings = append(warnings, fmt.Sprintf("skipped large file %s (%d bytes)", filepath.ToSlash(relative), info.Size()))
				return nil
			}

			ext := strings.ToLower(filepath.Ext(d.Name()))
			if _, ok := localhostAllowedExtensions[ext]; !ok {
				return nil
			}

			file, err := os.Open(path)
			if err != nil {
				relative, relErr := filepath.Rel(root, path)
				if relErr != nil {
					relative = path
				}
				warnings = append(warnings, fmt.Sprintf("failed to open %s: %v", filepath.ToSlash(relative), err))
				return nil
			}

			scanner := bufio.NewScanner(file)
			scanner.Buffer(make([]byte, 0, 64*1024), int(maxLocalhostScanFileSize))
			scannedFiles++
			lineNumber := 0
			relativePath, relErr := filepath.Rel(root, path)
			if relErr != nil {
				relativePath = path
			}
			relativePath = filepath.ToSlash(relativePath)

			for scanner.Scan() {
				lineNumber++
				text := scanner.Text()
				trimmed := strings.TrimSpace(text)
				if trimmed == "" {
					continue
				}
				for _, candidate := range localhostPatterns {
					match := candidate.Regex.FindString(trimmed)
					if match == "" {
						continue
					}

					snippet := trimmed
					if len(snippet) > 180 {
						snippet = snippet[:180] + "…"
					}
					item := LocalhostUsageFinding{
						FilePath: relativePath,
						Line:     lineNumber,
						Snippet:  snippet,
						Pattern:  fmt.Sprintf("%s: %s", candidate.Label, match),
					}
					report.Findings = append(report.Findings, item)
					break
				}
			}

			if err := scanner.Err(); err != nil {
				warnings = append(warnings, fmt.Sprintf("failed to scan %s: %v", relativePath, err))
			}

			if closeErr := file.Close(); closeErr != nil {
				warnings = append(warnings, fmt.Sprintf("failed to close %s: %v", relativePath, closeErr))
			}

			return nil
		})

		if err != nil {
			return nil, err
		}
	}

	report.Scanned = scannedFiles
	if len(warnings) > 0 {
		report.Warnings = warnings
	}

	return report, nil
}

// =============================================================================
// Health Check Execution
// =============================================================================

func (s *AppService) executeScenarioHealthCheck(ctx context.Context, checkID, name, endpoint string) (IssueHealthCheckEntry, []string) {
	entry := IssueHealthCheckEntry{
		ID:       checkID,
		Name:     name,
		Status:   "fail",
		Endpoint: endpoint,
	}

	var notes []string

	curlPath, err := exec.LookPath("curl")
	if err != nil {
		entry.Message = "curl command not available on host"
		entry.Code = "curl_not_found"
		notes = append(notes, "curl is required to capture preview health diagnostics")
		return entry, notes
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	args := []string{
		"-sS",
		"--show-error",
		"--max-time", "8",
		"--connect-timeout", "4",
		"-H", "Accept: application/json",
		"-w", "\n%{http_code}",
		endpoint,
	}

	start := s.timeNow()
	cmd := exec.CommandContext(ctxWithTimeout, curlPath, args...)
	output, cmdErr := cmd.CombinedOutput()
	latency := int(s.timeNow().Sub(start).Milliseconds())
	entry.LatencyMs = &latency

	trimmedOutput := strings.TrimRight(string(output), "\r\n")
	body := trimmedOutput
	statusCode := 0
	if idx := strings.LastIndex(trimmedOutput, "\n"); idx != -1 {
		body = trimmedOutput[:idx]
		codeStr := strings.TrimSpace(trimmedOutput[idx+1:])
		if parsed, parseErr := strconv.Atoi(codeStr); parseErr == nil {
			statusCode = parsed
		}
	}

	if cmdErr != nil {
		message := strings.TrimSpace(body)
		if message == "" {
			message = cmdErr.Error()
		}
		entry.Message = message
		entry.Code = "curl_error"
		entry.Status = "fail"
		notes = append(notes, fmt.Sprintf("curl error for %s health: %v", checkID, cmdErr))
		return entry, notes
	}

	trimmedBody := strings.TrimSpace(body)
	if trimmedBody != "" {
		entry.Response = trimmedBody
	}
	if trimmedBody == "" {
		entry.Status = statusFromHTTP(statusCode)
		entry.Message = "Health response was empty."
		return entry, notes
	}

	var data map[string]interface{}
	if decodeErr := json.Unmarshal([]byte(trimmedBody), &data); decodeErr != nil {
		entry.Status = statusFromHTTP(statusCode)
		entry.Message = trimmedBody
		entry.Code = "non_json_health_response"
		notes = append(notes, fmt.Sprintf("health response for %s was not JSON: %v", checkID, decodeErr))
		return entry, notes
	}

	if pretty, err := json.MarshalIndent(data, "", "  "); err == nil {
		entry.Response = string(pretty)
	}

	normalized := normalizeHealthStatus(trimmedString(data["status"]))
	if normalized == "" {
		normalized = statusFromHTTP(statusCode)
	}
	if normalized == "" {
		normalized = "fail"
	}
	entry.Status = normalized
	entry.Message = summarizeHealthResponse(data)
	if entry.Message == "" {
		entry.Message = trimmedBody
	}
	entry.Code = firstNonEmpty(trimmedString(data["code"]), anyStringFromMap(data, "error", "code"))

	return entry, notes
}

func normalizeHealthStatus(value string) string {
	switch normalizeLower(value) {
	case "", "unknown":
		return ""
	case "pass", "ok", "healthy", "ready", "up", "online":
		return "pass"
	case "warn", "warning", "degraded":
		return "warn"
	default:
		return "fail"
	}
}

func statusFromHTTP(code int) string {
	if code >= 200 && code < 300 {
		return "pass"
	}
	if code >= 300 && code < 400 {
		return "warn"
	}
	if code > 0 {
		return "fail"
	}
	return ""
}

func summarizeHealthResponse(data map[string]interface{}) string {
	if len(data) == 0 {
		return ""
	}

	var parts []string
	if status := trimmedString(data["status"]); status != "" {
		parts = append(parts, fmt.Sprintf("Status %s", status))
	}
	if message := trimmedString(data["message"]); message != "" {
		parts = append(parts, message)
	}
	if readiness, ok := data["readiness"].(bool); ok {
		if readiness {
			parts = append(parts, "Ready")
		} else {
			parts = append(parts, "Not ready")
		}
	}
	if metrics, ok := data["metrics"].(map[string]interface{}); ok {
		if uptime := anyNumber(metrics["uptime_seconds"]); uptime > 0 {
			parts = append(parts, fmt.Sprintf("Uptime %ds", int(uptime)))
		}
	}

	if len(parts) == 0 {
		encoded, err := json.Marshal(data)
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(encoded))
	}

	return strings.Join(parts, " • ")
}

// =============================================================================
// Scenario Status Formatting
// =============================================================================

func formatScenarioStatusLabel(status string) (string, ScenarioStatusSeverity) {
	normalized := strings.TrimSpace(strings.ToLower(status))
	if normalized == "" {
		return "[WARN] UNKNOWN", ScenarioStatusSeverityWarn
	}
	label := strings.ToUpper(normalized)
	switch normalized {
	case "running", "healthy", "good", "ready", "ok":
		return "[OK] " + label, ScenarioStatusSeverityOK
	case "stopped", "failed", "error", "crashed", "down", "terminated", "exited":
		return "[FAIL] " + label, ScenarioStatusSeverityError
	case "starting", "initializing", "pending", "booting", "paused":
		return "[WARN] " + label, ScenarioStatusSeverityWarn
	case "degraded", "warn", "warning", "unstable":
		return "[WARN] " + label, ScenarioStatusSeverityWarn
	default:
		return "[WARN] " + label, ScenarioStatusSeverityWarn
	}
}

func escalateScenarioSeverity(current, candidate ScenarioStatusSeverity) ScenarioStatusSeverity {
	order := map[ScenarioStatusSeverity]int{
		ScenarioStatusSeverityOK:    0,
		ScenarioStatusSeverityWarn:  1,
		ScenarioStatusSeverityError: 2,
	}
	if order[candidate] > order[current] {
		return candidate
	}
	return current
}

func summarizeScenarioHealth(health map[string]scenarioStatusHealthCheck) (ScenarioStatusSeverity, []string) {
	if len(health) == 0 {
		return ScenarioStatusSeverityWarn, []string{"  No health diagnostics returned."}
	}

	keys := make([]string, 0, len(health))
	for key := range health {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	severity := ScenarioStatusSeverityOK
	lines := make([]string, 0, len(keys)*4)
	for _, key := range keys {
		entry := health[key]
		displayName := strings.TrimSpace(entry.Name)
		if displayName == "" {
			// Format key as title case (e.g., "api_health" -> "Api Health")
			formatted := strings.ReplaceAll(key, "_", " ")
			if formatted != "" {
				// Simple title case: capitalize first letter of each word
				words := strings.Fields(formatted)
				for i, word := range words {
					if len(word) > 0 {
						words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
					}
				}
				displayName = strings.Join(words, " ")
			}
		}
		if displayName == "" {
			displayName = strings.ToUpper(key)
		}

		header := fmt.Sprintf("  %s", displayName)
		if entry.Port > 0 {
			header = fmt.Sprintf("%s (port %d):", header, entry.Port)
		} else {
			header += ":"
		}
		lines = append(lines, header)

		icon, entrySeverity, label := formatHealthStatusLabel(entry.Status)
		severity = escalateScenarioSeverity(severity, entrySeverity)
		lines = append(lines, fmt.Sprintf("    Status:      %s %s", icon, label))

		if entry.APIConnectivity != nil {
			conn := entry.APIConnectivity
			if conn.Connected {
				message := "[OK] Connected"
				if conn.APIURL != "" {
					message = fmt.Sprintf("%s to %s", message, conn.APIURL)
				}
				if conn.LatencyMs != nil && *conn.LatencyMs >= 0 {
					latency := int(*conn.LatencyMs + 0.5)
					message = fmt.Sprintf("%s (%dms)", message, latency)
				}
				lines = append(lines, fmt.Sprintf("    API link:    %s", message))
			} else {
				severity = escalateScenarioSeverity(severity, ScenarioStatusSeverityError)
				note := "[FAIL] Not connected"
				if strings.TrimSpace(conn.Error) != "" {
					note = fmt.Sprintf("%s (%s)", note, strings.TrimSpace(conn.Error))
				}
				lines = append(lines, fmt.Sprintf("    API link:    %s", note))
			}
		}

		if entry.ResponseTime != nil && *entry.ResponseTime > 0 {
			latency := int((*entry.ResponseTime * 1000) + 0.5)
			lines = append(lines, fmt.Sprintf("    Latency:     %dms", latency))
		}

		if entry.SchemaValid != nil {
			if *entry.SchemaValid {
				lines = append(lines, "    Schema:      [OK] Valid")
			} else {
				severity = escalateScenarioSeverity(severity, ScenarioStatusSeverityWarn)
				lines = append(lines, "    Schema:      [WARN] Invalid response schema")
			}
		}

		if strings.TrimSpace(entry.Message) != "" {
			lines = append(lines, fmt.Sprintf("    Note:        %s", strings.TrimSpace(entry.Message)))
		}
	}

	return severity, lines
}

func formatHealthStatusLabel(status string) (string, ScenarioStatusSeverity, string) {
	normalized := strings.TrimSpace(strings.ToLower(status))
	if normalized == "" {
		return "[WARN]", ScenarioStatusSeverityWarn, "UNKNOWN"
	}
	label := strings.ToUpper(normalized)
	switch normalized {
	case "healthy", "pass", "ok", "good":
		return "[OK]", ScenarioStatusSeverityOK, label
	case "fail", "failed", "error", "critical", "down":
		return "[FAIL]", ScenarioStatusSeverityError, label
	case "degraded", "warn", "warning", "unstable":
		return "[WARN]", ScenarioStatusSeverityWarn, label
	default:
		return "[WARN]", ScenarioStatusSeverityWarn, label
	}
}

func summarizeScenarioTests(infra scenarioStatusTestInfrastructure) (ScenarioStatusSeverity, []string, []string) {
	entries := []struct {
		label string
		entry *scenarioStatusTestEntry
	}{
		{"Overall Status", infra.Overall},
		{"Test Lifecycle", infra.TestLifecycle},
		{"Phased Testing", infra.PhasedStructure},
		{"Unit Tests", infra.UnitTests},
		{"CLI Tests", infra.CliTests},
		{"UI Tests", infra.UiTests},
	}

	total := 0
	for _, item := range entries {
		if item.entry != nil && (strings.TrimSpace(item.entry.Status) != "" || strings.TrimSpace(item.entry.Message) != "") {
			total++
		}
	}

	if total == 0 {
		return ScenarioStatusSeverityWarn, []string{"  No test diagnostics returned."}, nil
	}

	severity := ScenarioStatusSeverityOK
	lines := make([]string, 0, total)
	recommendations := make([]string, 0)
	index := 0
	for _, item := range entries {
		entry := item.entry
		if entry == nil {
			continue
		}
		if strings.TrimSpace(entry.Status) == "" && strings.TrimSpace(entry.Message) == "" {
			continue
		}
		index++
		prefix := "|-"
		if index == total {
			prefix = "`-"
		}

		icon, entrySeverity := formatTestStatusIndicator(entry.Status)
		severity = escalateScenarioSeverity(severity, entrySeverity)

		message := strings.TrimSpace(entry.Message)
		if message == "" {
			statusLabel := strings.TrimSpace(entry.Status)
			if statusLabel == "" {
				message = "No diagnostics reported"
			} else {
				message = strings.ToUpper(statusLabel)
			}
		}

		lines = append(lines, fmt.Sprintf("%s %s: %s %s", prefix, item.label, icon, message))

		if entry.Recommendation != "" {
			recommendations = append(recommendations, entry.Recommendation)
		}
		if len(entry.Recommendations) > 0 {
			recommendations = append(recommendations, entry.Recommendations...)
		}
	}

	return severity, lines, recommendations
}

func formatTestStatusIndicator(status string) (string, ScenarioStatusSeverity) {
	normalized := strings.TrimSpace(strings.ToLower(status))
	if normalized == "" {
		return "[WARN]", ScenarioStatusSeverityWarn
	}
	switch normalized {
	case "good", "complete", "present", "passing", "ready":
		return "[OK]", ScenarioStatusSeverityOK
	case "missing", "absent", "error", "failed", "critical":
		return "[FAIL]", ScenarioStatusSeverityError
	case "partial", "legacy", "warning", "degraded", "incomplete":
		return "[WARN]", ScenarioStatusSeverityWarn
	default:
		return "[WARN]", ScenarioStatusSeverityWarn
	}
}
