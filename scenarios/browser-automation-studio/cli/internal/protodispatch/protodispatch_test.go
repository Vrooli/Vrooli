package protodispatch

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/ai"
)

// TestRenderProtoJSONRedacted_RedactsScreenshotBytes verifies that the
// human render path strips bytes payloads (ai preview-screenshot's
// ScreenshotPng was the motivating regression — see the BAS migration
// follow-up "Teach protodispatch's human formatter to redact bytes
// fields"). The raw JSON path must keep the bytes payload intact.
func TestRenderProtoJSONRedacted_RedactsScreenshotBytes(t *testing.T) {
	// 600 bytes of fake PNG — large enough that base64 expansion would
	// dominate the output if not redacted.
	payload := bytes.Repeat([]byte{0xDE, 0xAD, 0xBE, 0xEF}, 150)
	msg := &aiv1.TakePreviewScreenshotResponse{
		ScreenshotPng:  payload,
		Url:            "https://example.com",
		ViewportWidth:  390,
		ViewportHeight: 844,
		ContentType:    "image/png",
	}

	var redacted bytes.Buffer
	if err := renderProtoJSONRedacted(&redacted, msg); err != nil {
		t.Fatalf("redacted render: %v", err)
	}

	got := redacted.String()
	if !strings.Contains(got, "screenshotPng") {
		t.Errorf("expected screenshotPng key, got: %s", got)
	}
	if !strings.Contains(got, "redacted") {
		t.Errorf("expected redaction marker in human render, got: %s", got)
	}
	if !strings.Contains(got, "pass --json for raw") {
		t.Errorf("expected pointer to --json escape hatch, got: %s", got)
	}

	// Approximate byte count survives in the summary (we computed it
	// from the base64 length; ~600 bytes ±2).
	if !strings.Contains(got, "600") && !strings.Contains(got, "598") && !strings.Contains(got, "602") {
		t.Errorf("expected byte count near 600 in summary, got: %s", got)
	}

	// Re-check the redacted output is valid JSON (so downstream tooling
	// like `| jq` still works on human output).
	var generic map[string]interface{}
	if err := json.Unmarshal(redacted.Bytes(), &generic); err != nil {
		t.Errorf("redacted output is not valid JSON: %v\n%s", err, got)
	}

	// Sibling structured fields must survive redaction.
	if generic["url"] != "https://example.com" {
		t.Errorf("url field missing: %v", generic["url"])
	}
	if v, ok := generic["viewportWidth"]; !ok || v == nil {
		t.Errorf("viewportWidth missing: %v", generic["viewportWidth"])
	}
}

// TestRenderProtoJSON_KeepsScreenshotBytes verifies the --json path is
// unaffected: raw bytes still surface as base64 for callers that need
// the full payload.
func TestRenderProtoJSON_KeepsScreenshotBytes(t *testing.T) {
	payload := bytes.Repeat([]byte{0xDE, 0xAD, 0xBE, 0xEF}, 150)
	msg := &aiv1.TakePreviewScreenshotResponse{
		ScreenshotPng: payload,
	}

	var raw bytes.Buffer
	if err := renderProtoJSON(&raw, msg); err != nil {
		t.Fatalf("renderProtoJSON: %v", err)
	}
	if !strings.Contains(raw.String(), "screenshotPng") {
		t.Errorf("expected screenshotPng key in raw render, got: %s", raw.String())
	}
	// Raw render contains the base64 encoding (>= 800 chars for 600 bytes).
	// "redacted" must NOT be present.
	if strings.Contains(raw.String(), "redacted") {
		t.Errorf("raw render must not redact bytes")
	}
}
