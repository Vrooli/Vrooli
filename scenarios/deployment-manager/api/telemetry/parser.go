package telemetry

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"
)

// UploadRequest is the request body for the upload endpoint.
type UploadRequest struct {
	ScenarioName   string                   `json:"scenario_name"`
	DeploymentMode string                   `json:"deployment_mode"`
	Source         string                   `json:"source"`
	Events         []map[string]interface{} `json:"events"`
}

// UploadParams holds resolved parameters for telemetry uploads.
type UploadParams struct {
	Scenario string
	Mode     string
	Source   string
}

// ParseJSONL parses a JSONL (JSON Lines) format input.
func ParseJSONL(r io.Reader) ([]map[string]interface{}, error) {
	var events []map[string]interface{}
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 1024*64), 1024*1024) // allow up to 1MB per line
	lineNum := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		lineNum++
		if line == "" {
			continue
		}
		var evt map[string]interface{}
		if err := json.Unmarshal([]byte(line), &evt); err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}
		events = append(events, evt)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

// ParseEventsFromRequest extracts telemetry events from the request body.
// Supports both JSON (with wrapper) and JSONL formats based on Content-Type.
func ParseEventsFromRequest(r io.Reader, contentType string, maxBytes int64) ([]map[string]interface{}, *UploadRequest, error) {
	limited := io.LimitReader(r, maxBytes)

	// JSON format: expects {"events": [...], "scenario_name": "...", ...}
	if strings.Contains(contentType, "application/json") && !strings.Contains(contentType, "jsonl") {
		var req UploadRequest
		if err := json.NewDecoder(limited).Decode(&req); err != nil {
			return nil, nil, fmt.Errorf("invalid json payload: %w", err)
		}
		return req.Events, &req, nil
	}

	// JSONL format: one JSON object per line
	events, err := ParseJSONL(limited)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid jsonl payload: %w", err)
	}
	return events, nil, nil
}

// ResolveUploadParams extracts scenario/mode/source from query params and request body,
// with query params taking precedence. Returns sensible defaults for missing values.
func ResolveUploadParams(query url.Values, req *UploadRequest) UploadParams {
	params := UploadParams{
		Scenario: "unknown",
		Mode:     "bundled",
		Source:   "upload",
	}

	// Try query params first, then request body
	if v := strings.TrimSpace(query.Get("scenario")); v != "" {
		params.Scenario = v
	} else if req != nil && strings.TrimSpace(req.ScenarioName) != "" {
		params.Scenario = strings.TrimSpace(req.ScenarioName)
	}

	if v := strings.TrimSpace(query.Get("mode")); v != "" {
		params.Mode = v
	} else if req != nil && strings.TrimSpace(req.DeploymentMode) != "" {
		params.Mode = strings.TrimSpace(req.DeploymentMode)
	}

	if v := strings.TrimSpace(query.Get("source")); v != "" {
		params.Source = v
	} else if req != nil && strings.TrimSpace(req.Source) != "" {
		params.Source = strings.TrimSpace(req.Source)
	}

	return params
}

// EnrichEvents adds default fields (timestamp, scenario, mode, source) to events that lack them.
func EnrichEvents(events []map[string]interface{}, params UploadParams, timestamp string) {
	for _, evt := range events {
		if _, ok := evt["timestamp"]; !ok {
			evt["timestamp"] = timestamp
		}
		if _, ok := evt["scenario_name"]; !ok {
			evt["scenario_name"] = params.Scenario
		}
		if _, ok := evt["deployment_mode"]; !ok {
			evt["deployment_mode"] = params.Mode
		}
		if _, ok := evt["source"]; !ok {
			evt["source"] = params.Source
		}
	}
}
