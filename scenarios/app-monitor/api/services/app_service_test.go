package services

import (
	"encoding/json"
	"testing"
)

func TestIssueReportRequestAllowsNullFields(t *testing.T) {
	jsonPayload := []byte(`{
        "message": "Example issue",
        "includeScreenshot": false,
        "previewUrl": null,
        "appName": null,
        "scenarioName": null,
        "source": null,
        "screenshotData": null,
        "logs": [],
        "logsTotal": null,
        "logsCapturedAt": null,
        "consoleLogs": [],
        "consoleLogsTotal": null,
        "consoleLogsCapturedAt": null,
        "networkRequests": [],
        "networkRequestsTotal": null,
        "networkCapturedAt": null
    }`)

	var req IssueReportRequest
	if err := json.Unmarshal(jsonPayload, &req); err != nil {
		t.Fatalf("unexpected error unmarshalling payload: %v", err)
	}

	if req.PreviewURL != nil {
		t.Fatalf("expected PreviewURL to be nil when JSON contains null")
	}
	if req.AppName != nil {
		t.Fatalf("expected AppName to be nil when JSON contains null")
	}
	if req.ScenarioName != nil {
		t.Fatalf("expected ScenarioName to be nil when JSON contains null")
	}
	if req.Source != nil {
		t.Fatalf("expected Source to be nil when JSON contains null")
	}
	if req.ScreenshotData != nil {
		t.Fatalf("expected ScreenshotData to be nil when JSON contains null")
	}
	if req.LogsTotal != nil {
		t.Fatalf("expected LogsTotal pointer to be nil when JSON contains null")
	}
	if req.LogsCapturedAt != nil {
		t.Fatalf("expected LogsCapturedAt pointer to be nil when JSON contains null")
	}
	if req.ConsoleLogsTotal != nil {
		t.Fatalf("expected ConsoleLogsTotal pointer to be nil when JSON contains null")
	}
	if req.ConsoleCapturedAt != nil {
		t.Fatalf("expected ConsoleCapturedAt pointer to be nil when JSON contains null")
	}
	if req.NetworkTotal != nil {
		t.Fatalf("expected NetworkTotal pointer to be nil when JSON contains null")
	}
	if req.NetworkCapturedAt != nil {
		t.Fatalf("expected NetworkCapturedAt pointer to be nil when JSON contains null")
	}

	if req.Message != "Example issue" {
		t.Fatalf("expected message to be decoded, got %q", req.Message)
	}
}

func TestStringHelpers(t *testing.T) {
	if got := stringValue(nil); got != "" {
		t.Fatalf("expected empty string for nil pointer, got %q", got)
	}
	ptr := "hello"
	if got := stringValue(&ptr); got != "hello" {
		t.Fatalf("expected original string, got %q", got)
	}

	if got := valueOrDefault(nil, 42); got != 42 {
		t.Fatalf("expected fallback value, got %d", got)
	}
	number := 7
	if got := valueOrDefault(&number, 42); got != 7 {
		t.Fatalf("expected pointer value, got %d", got)
	}
}
