package executionwriter

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
)

func TestSanitizeOutcomeDOMTruncationAddsHash(t *testing.T) {
	longHTML := strings.Repeat("a", contracts.DOMSnapshotMaxBytes+10)
	out := sanitizeOutcome(contracts.StepOutcome{
		DOMSnapshot: &contracts.DOMSnapshot{
			HTML: longHTML,
		},
	})

	if out.DOMSnapshot == nil || !out.DOMSnapshot.Truncated {
		t.Fatalf("expected DOM snapshot to be truncated")
	}
	if len(out.DOMSnapshot.HTML) != contracts.DOMSnapshotMaxBytes {
		t.Fatalf("expected DOM snapshot length %d, got %d", contracts.DOMSnapshotMaxBytes, len(out.DOMSnapshot.HTML))
	}
	if out.DOMSnapshot.Hash == "" {
		t.Fatalf("expected hash to be set on truncated DOM")
	}
	if out.Notes["dom_truncated_hash"] != out.DOMSnapshot.Hash {
		t.Fatalf("expected dom_truncated_hash note to match hash")
	}
}

func TestSanitizeOutcomeConsoleTruncation(t *testing.T) {
	long := strings.Repeat("x", contracts.ConsoleEntryMaxBytes+5)
	now := time.Now()
	out := sanitizeOutcome(contracts.StepOutcome{
		ConsoleLogs: []contracts.ConsoleLogEntry{
			{Type: "log", Text: long, Timestamp: now},
		},
	})
	if len(out.ConsoleLogs) != 1 {
		t.Fatalf("expected one console log")
	}
	entry := out.ConsoleLogs[0]
	if !strings.HasSuffix(entry.Text, "[truncated]") {
		t.Fatalf("expected console text to be truncated, got %s", entry.Text)
	}
	if entry.Timestamp.Location() != time.UTC {
		t.Fatalf("expected console timestamp UTC, got %v", entry.Timestamp.Location())
	}
	if entry.Location == "" {
		t.Fatalf("expected truncation hash to be appended to location")
	}
}

func TestSanitizeOutcomeScreenshotClamping(t *testing.T) {
	oversized := make([]byte, contracts.ScreenshotMaxBytes+10)
	for i := range oversized {
		oversized[i] = 0x01
	}
	out := sanitizeOutcome(contracts.StepOutcome{
		Screenshot: &contracts.Screenshot{
			Data: oversized,
		},
	})
	if out.Screenshot == nil {
		t.Fatalf("expected screenshot to remain present")
	}
	if len(out.Screenshot.Data) != contracts.ScreenshotMaxBytes {
		t.Fatalf("expected screenshot to be clamped to %d bytes, got %d", contracts.ScreenshotMaxBytes, len(out.Screenshot.Data))
	}
	if out.Screenshot.MediaType == "" {
		t.Fatalf("expected default media type set")
	}
	if out.Screenshot.Width == 0 || out.Screenshot.Height == 0 {
		t.Fatalf("expected default dimensions set")
	}
	if out.Notes["screenshot_truncated"] == "" {
		t.Fatalf("expected screenshot_truncated note set")
	}
}

func TestSanitizeOutcomeNetworkTruncation(t *testing.T) {
	long := strings.Repeat("y", contracts.NetworkPayloadPreviewMaxBytes+10)
	now := time.Now()
	out := sanitizeOutcome(contracts.StepOutcome{
		Network: []contracts.NetworkEvent{
			{
				Type:                "request",
				URL:                 "https://example.com",
				RequestBodyPreview:  long,
				ResponseBodyPreview: long,
				Timestamp:           now,
			},
		},
	})
	if len(out.Network) != 1 {
		t.Fatalf("expected one network event")
	}
	ev := out.Network[0]
	if ev.Timestamp.Location() != time.UTC {
		t.Fatalf("expected network timestamp UTC, got %v", ev.Timestamp.Location())
	}
	if len(ev.RequestBodyPreview) != contracts.NetworkPayloadPreviewMaxBytes {
		t.Fatalf("expected request preview truncated to %d, got %d", contracts.NetworkPayloadPreviewMaxBytes, len(ev.RequestBodyPreview))
	}
	if len(ev.ResponseBodyPreview) != contracts.NetworkPayloadPreviewMaxBytes {
		t.Fatalf("expected response preview truncated to %d, got %d", contracts.NetworkPayloadPreviewMaxBytes, len(ev.ResponseBodyPreview))
	}
	if !ev.Truncated {
		t.Fatalf("expected truncated flag to be set")
	}
}

func TestStatusFromOutcome(t *testing.T) {
	if got := statusFromOutcome(contracts.StepOutcome{Success: true}); got != "completed" {
		t.Fatalf("expected completed, got %q", got)
	}
	if got := statusFromOutcome(contracts.StepOutcome{Success: false}); got != "failed" {
		t.Fatalf("expected failed, got %q", got)
	}
}

func TestDeriveStepLabel(t *testing.T) {
	label := deriveStepLabel(contracts.StepOutcome{StepType: "navigate", NodeID: "n1"})
	if label == "" {
		t.Fatalf("expected non-empty label")
	}
}

func TestTruncateRunes(t *testing.T) {
	if got := truncateRunes("hello", 10); got != "hello" {
		t.Fatalf("expected hello, got %q", got)
	}
	if got := truncateRunes("hello", 2); got != "he" {
		t.Fatalf("expected he, got %q", got)
	}
}

func TestHashString(t *testing.T) {
	if got := hashString("hello"); got == "" {
		t.Fatalf("expected hash")
	}
}

func TestAppendHash(t *testing.T) {
	out := appendHash("loc", "payload")
	if !strings.Contains(out, "loc") {
		t.Fatalf("expected prefix to be preserved")
	}
	if out == "loc" {
		t.Fatalf("expected hash suffix")
	}
}

func TestToStringIDs(t *testing.T) {
	ids := []uuid.UUID{uuid.New(), uuid.New()}
	out := toStringIDs(ids)
	if len(out) != 2 {
		t.Fatalf("expected 2 ids, got %d", len(out))
	}
	if out[0] == "" || out[1] == "" {
		t.Fatalf("expected non-empty strings")
	}
}

func TestSanitizeOutcome_EmptyOutcome(t *testing.T) {
	out := sanitizeOutcome(contracts.StepOutcome{})
	if out.Notes == nil {
		t.Fatalf("expected notes map to be initialized")
	}
}

func TestSanitizeOutcome_NilNotesInitialized(t *testing.T) {
	out := sanitizeOutcome(contracts.StepOutcome{Notes: nil})
	if out.Notes == nil {
		t.Fatalf("expected notes map to be initialized")
	}
}

func TestSanitizeOutcome_PreservesExistingNotes(t *testing.T) {
	out := sanitizeOutcome(contracts.StepOutcome{Notes: map[string]string{"a": "b"}})
	if out.Notes["a"] != "b" {
		t.Fatalf("expected notes to be preserved")
	}
}

func TestSanitizeOutcome_ScreenshotDefaults(t *testing.T) {
	out := sanitizeOutcome(contracts.StepOutcome{
		Screenshot: &contracts.Screenshot{Data: []byte{0x00}},
	})
	if out.Screenshot.MediaType == "" || out.Screenshot.Width == 0 || out.Screenshot.Height == 0 {
		t.Fatalf("expected screenshot defaults to be set, got %+v", out.Screenshot)
	}
}

func TestBuildTimelinePayload_MinimalOutcome(t *testing.T) {
	outcome := contracts.StepOutcome{
		NodeID:    "node-1",
		StepType:  "click",
		Success:   true,
		StepIndex: 2,
		DurationMs: 123,
	}

	payload := buildTimelinePayload(outcome, "s3://shot.png", nil, nil, "", nil)
	if payload["nodeId"] != "node-1" {
		t.Fatalf("expected nodeId to propagate, got %v", payload["nodeId"])
	}
}

func TestBuildTimelinePayload_FailureIncludesError(t *testing.T) {
	outcome := contracts.StepOutcome{
		NodeID:    "node-1",
		StepType:  "click",
		Success:   false,
		StepIndex: 2,
		Failure: &contracts.StepFailure{
			Kind:    contracts.FailureKindEngine,
			Message: "boom",
		},
		DurationMs: 123,
	}

	payload := buildTimelinePayload(outcome, "s3://shot.png", nil, nil, "", nil)
	if payload["partial"] != true {
		t.Fatalf("expected partial=true when failure present, got %v", payload["partial"])
	}
	if payload["error"] == "" {
		t.Fatalf("expected error to be included")
	}
}

func TestBuildTimelinePayload_AllOptionalFields(t *testing.T) {
	domID := uuid.New()
	screenshotID := uuid.New()
	artifactIDs := []uuid.UUID{uuid.New(), uuid.New()}
	outcome := contracts.StepOutcome{
		NodeID:     "node-1",
		StepType:   "screenshot",
		Success:    true,
		StepIndex:  0,
		DurationMs: 200,
		FinalURL:   "https://example.test",
		ZoomFactor: 1.4,
		ClickPosition: &contracts.Point{
			X: 10,
			Y: 20,
		},
		FocusedElement: &contracts.ElementFocus{
			Selector: "#hero",
			BoundingBox: &contracts.BoundingBox{
				X: 10, Y: 20, Width: 200, Height: 120,
			},
		},
		HighlightRegions: []*contracts.HighlightRegion{
			{Selector: "#hero", Padding: 12},
		},
		MaskRegions: []*contracts.MaskRegion{
			{Selector: ".mask", Opacity: 0.5},
		},
		CursorTrail: []contracts.CursorPosition{
			{Point: &contracts.Point{X: 1, Y: 2}, ElapsedMs: 5},
		},
	}

	payload := buildTimelinePayload(outcome, "s3://shot.png", &screenshotID, &domID, "<html>", artifactIDs)
	for _, key := range []string{
		"focusedElement",
		"highlightRegions",
		"maskRegions",
		"cursorTrail",
		"domSnapshotArtifactId",
		"domSnapshotPreview",
		"artifactIds",
		"screenshotUrl",
	} {
		if _, ok := payload[key]; !ok {
			t.Fatalf("expected timeline payload to include %s, got %+v", key, payload)
		}
	}
}
