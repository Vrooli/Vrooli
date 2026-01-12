package telemetry

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// DefaultService is the standard implementation of the telemetry Service.
type DefaultService struct {
	vrooliRoot string
	mu         sync.Mutex
}

// NewService creates a new telemetry service.
func NewService(vrooliRoot string) *DefaultService {
	return &DefaultService{
		vrooliRoot: vrooliRoot,
	}
}

// GetFilePath returns the path to the telemetry file for a scenario.
func (s *DefaultService) GetFilePath(scenario string) string {
	return filepath.Join(s.vrooliRoot, ".vrooli", "deployment", "telemetry", fmt.Sprintf("%s.jsonl", scenario))
}

// telemetryDir returns the telemetry directory path.
func (s *DefaultService) telemetryDir() string {
	return filepath.Join(s.vrooliRoot, ".vrooli", "deployment", "telemetry")
}

// IngestEvents ingests telemetry events for a scenario.
func (s *DefaultService) IngestEvents(ctx context.Context, scenario, deploymentMode, source string, events []map[string]interface{}) (string, int, error) {
	if scenario == "" {
		return "", 0, fmt.Errorf("scenario_name is required")
	}
	if len(events) == 0 {
		return "", 0, fmt.Errorf("events array cannot be empty")
	}
	if deploymentMode == "" {
		deploymentMode = "external-server"
	}

	telemetryDir := s.telemetryDir()
	if err := os.MkdirAll(telemetryDir, 0o755); err != nil {
		return "", 0, fmt.Errorf("failed to prepare telemetry directory: %s", err)
	}

	filePath := s.GetFilePath(scenario)
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return "", 0, fmt.Errorf("failed to open telemetry file: %s", err)
	}
	defer file.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	ingested := 0
	for _, event := range events {
		normalized := normalizeEvent(event, scenario, deploymentMode, source, now)
		data, err := json.Marshal(normalized)
		if err != nil {
			return "", 0, fmt.Errorf("failed to marshal telemetry event: %s", err)
		}
		if _, err := file.Write(append(data, '\n')); err != nil {
			return "", 0, fmt.Errorf("failed to write telemetry: %s", err)
		}
		ingested++
	}
	return filePath, ingested, nil
}

// normalizeEvent normalizes a telemetry event with standard fields.
func normalizeEvent(event map[string]interface{}, scenario, deploymentMode, source, ingestedAt string) map[string]interface{} {
	normalized := make(map[string]interface{}, len(event)+6)
	for key, value := range event {
		normalized[key] = value
	}

	// Remove fields that will be normalized
	delete(normalized, "deploymentMode")
	delete(normalized, "deployment_mode")
	delete(normalized, "serverType")
	delete(normalized, "server_type")
	delete(normalized, "scenario_name")
	delete(normalized, "source")
	delete(normalized, "ingested_at")

	// Normalize server_type
	if serverType, ok := event["serverType"].(string); ok && serverType != "" {
		normalized["server_type"] = serverType
	}
	if serverType, ok := event["server_type"].(string); ok && serverType != "" {
		normalized["server_type"] = serverType
	}

	// Normalize deployment_mode
	if deploymentMode == "" {
		if raw, ok := event["deploymentMode"].(string); ok && raw != "" {
			deploymentMode = raw
		} else if raw, ok := event["deployment_mode"].(string); ok && raw != "" {
			deploymentMode = raw
		} else {
			deploymentMode = "unknown"
		}
	}
	if source == "" {
		source = "desktop-runtime"
	}

	normalized["scenario_name"] = scenario
	normalized["deployment_mode"] = deploymentMode
	normalized["source"] = source
	normalized["ingested_at"] = ingestedAt
	if _, ok := normalized["level"]; !ok {
		normalized["level"] = "info"
	}

	return normalized
}

// GetSummary returns a summary of telemetry for a scenario.
func (s *DefaultService) GetSummary(ctx context.Context, scenario string) (*SummaryResult, error) {
	filePath := s.GetFilePath(scenario)
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &SummaryResult{Exists: false}, nil
		}
		return nil, fmt.Errorf("failed to read telemetry file: %s", err)
	}

	eventCount, lastIngestedAt, err := readSummary(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read telemetry summary: %s", err)
	}

	return &SummaryResult{
		Exists:         true,
		FilePath:       filePath,
		FileSizeBytes:  info.Size(),
		EventCount:     eventCount,
		LastIngestedAt: lastIngestedAt,
	}, nil
}

// readSummary reads telemetry file and returns count and last ingested time.
func readSummary(filePath string) (int, string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var count int
	var lastIngested time.Time
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		count++
		var payload map[string]interface{}
		if err := json.Unmarshal(line, &payload); err != nil {
			continue
		}
		if raw, ok := payload["ingested_at"].(string); ok && raw != "" {
			if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
				if parsed.After(lastIngested) {
					lastIngested = parsed
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, "", err
	}
	lastIngestedAt := ""
	if !lastIngested.IsZero() {
		lastIngestedAt = lastIngested.Format(time.RFC3339)
	}
	return count, lastIngestedAt, nil
}

// GetInsights analyzes telemetry and returns insights.
func (s *DefaultService) GetInsights(ctx context.Context, scenario string) (*InsightsResult, error) {
	filePath := s.GetFilePath(scenario)
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			return &InsightsResult{Exists: false}, nil
		}
		return nil, fmt.Errorf("failed to read telemetry file: %s", err)
	}

	insights, err := readInsights(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read telemetry insights: %s", err)
	}

	return &InsightsResult{
		Exists:        true,
		LastSession:   insights.lastSession,
		LastSmokeTest: insights.lastSmokeTest,
		LastError:     insights.lastError,
	}, nil
}

// readInsights parses the telemetry file for insights.
func readInsights(filePath string) (*parsedInsights, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	insights := &parsedInsights{}
	var lastSessionTime time.Time
	var lastSmokeTime time.Time
	var lastErrorTime time.Time
	var lastAppStart time.Time
	var lastAppReady time.Time
	var lastAppShutdown time.Time

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var payload map[string]interface{}
		if err := json.Unmarshal(line, &payload); err != nil {
			continue
		}
		event, _ := payload["event"].(string)
		if event == "" {
			continue
		}
		ts, ok := ParseEventTimestamp(payload)
		if !ok {
			continue
		}

		switch event {
		case "app_start":
			if ts.After(lastAppStart) {
				lastAppStart = ts
			}
		case "app_ready":
			if ts.After(lastAppReady) {
				lastAppReady = ts
			}
		case "app_shutdown":
			if ts.After(lastAppShutdown) {
				lastAppShutdown = ts
			}
		}

		if event == "app_session_succeeded" || event == "app_session_failed" {
			if ts.After(lastSessionTime) {
				lastSessionTime = ts
				insights.lastSession = buildSessionInsight(payload, event, ts)
			}
		}

		if event == "smoke_test_started" || event == "smoke_test_passed" || event == "smoke_test_failed" {
			if ts.After(lastSmokeTime) {
				lastSmokeTime = ts
				insights.lastSmokeTest = buildSmokeTestInsight(payload, event, ts)
			}
		}

		if isError(payload, event) && ts.After(lastErrorTime) {
			lastErrorTime = ts
			insights.lastError = buildErrorInsight(payload, event, ts)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Infer session from app lifecycle events if no explicit session events
	if insights.lastSession == nil && !lastAppStart.IsZero() {
		status := "failed"
		if lastAppReady.After(lastAppStart) {
			status = "succeeded"
		}
		insights.lastSession = &SessionInsight{
			Status:      status,
			StartedAt:   lastAppStart.Format(time.RFC3339),
			ReadyAt:     FormatTimeIfSet(lastAppReady),
			CompletedAt: FormatTimeIfSet(lastAppShutdown),
			Reason:      "",
		}
		if status == "failed" {
			insights.lastSession.Reason = "app_exit_before_ready"
		}
	}

	return insights, nil
}

func buildSessionInsight(payload map[string]interface{}, event string, ts time.Time) *SessionInsight {
	status := "succeeded"
	if event == "app_session_failed" {
		status = "failed"
	}
	insight := &SessionInsight{
		SessionID:   StringFromPayload(payload, "session_id"),
		Status:      status,
		CompletedAt: ts.Format(time.RFC3339),
	}
	if details, ok := payload["details"].(map[string]interface{}); ok {
		insight.StartedAt = StringFromPayload(details, "started_at")
		insight.ReadyAt = StringFromPayload(details, "ready_at")
		insight.Reason = StringFromPayload(details, "reason")
	}
	return insight
}

func buildSmokeTestInsight(payload map[string]interface{}, event string, ts time.Time) *SmokeTestInsight {
	status := "started"
	if event == "smoke_test_passed" {
		status = "passed"
	}
	if event == "smoke_test_failed" {
		status = "failed"
	}
	insight := &SmokeTestInsight{
		SessionID: StringFromPayload(payload, "session_id"),
		Status:    status,
	}
	if event == "smoke_test_started" {
		insight.StartedAt = ts.Format(time.RFC3339)
	} else {
		insight.CompletedAt = ts.Format(time.RFC3339)
	}
	if details, ok := payload["details"].(map[string]interface{}); ok {
		insight.Error = StringFromPayload(details, "error")
	}
	return insight
}

func buildErrorInsight(payload map[string]interface{}, event string, ts time.Time) *ErrorInsight {
	message := ""
	if details, ok := payload["details"].(map[string]interface{}); ok {
		message = StringFromPayload(details, "error")
		if message == "" {
			message = StringFromPayload(details, "message")
		}
	}
	return &ErrorInsight{
		Event:     event,
		Timestamp: ts.Format(time.RFC3339),
		Message:   message,
	}
}

func isError(payload map[string]interface{}, event string) bool {
	if level, ok := payload["level"].(string); ok && level == "error" {
		return true
	}
	return IsErrorEvent(event)
}

// GetTail returns the last N telemetry entries.
func (s *DefaultService) GetTail(ctx context.Context, scenario string, limit int) (*TailResult, error) {
	filePath := s.GetFilePath(scenario)
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			return &TailResult{Exists: false, Limit: limit}, nil
		}
		return nil, fmt.Errorf("failed to read telemetry file: %s", err)
	}

	entries, totalLines, err := readTail(filePath, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to read telemetry tail: %s", err)
	}

	return &TailResult{
		Exists:     true,
		Limit:      limit,
		TotalLines: totalLines,
		Entries:    entries,
	}, nil
}

// readTail reads the last N lines from a telemetry file.
func readTail(filePath string, limit int) ([]TailEntry, int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var totalLines int
	ring := make([]string, 0, limit)
	index := 0
	full := false

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		totalLines++
		if len(ring) < limit {
			ring = append(ring, line)
			continue
		}
		ring[index] = line
		full = true
		index++
		if index >= limit {
			index = 0
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, 0, err
	}

	ordered := ring
	if full && len(ring) > 0 {
		ordered = append(ring[index:], ring[:index]...)
	}

	entries := make([]TailEntry, 0, len(ordered))
	for _, line := range ordered {
		entry := TailEntry{Raw: line}
		var payload map[string]interface{}
		if err := json.Unmarshal([]byte(line), &payload); err != nil {
			entry.Error = err.Error()
		} else {
			entry.Event = payload
		}
		entries = append(entries, entry)
	}

	return entries, totalLines, nil
}

// Delete removes all telemetry for a scenario.
func (s *DefaultService) Delete(ctx context.Context, scenario string) error {
	filePath := s.GetFilePath(scenario)
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("telemetry not found")
		}
		return fmt.Errorf("failed to delete telemetry: %s", err)
	}
	return nil
}
