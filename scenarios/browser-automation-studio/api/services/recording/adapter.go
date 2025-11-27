package recording

import (
	"archive/zip"
	"fmt"
	"image"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
)

// recordingAdapter converts extension recording frames into contract step outcomes.
type recordingAdapter struct {
	executionID uuid.UUID
	workflowID  uuid.UUID
	manifest    *recordingManifest
	files       map[string]*zip.File
	viewport    recordingViewport
	log         *logrus.Logger
}

func newRecordingAdapter(executionID, workflowID uuid.UUID, manifest *recordingManifest, files map[string]*zip.File, log *logrus.Logger) *recordingAdapter {
	return &recordingAdapter{
		executionID: executionID,
		workflowID:  workflowID,
		manifest:    manifest,
		files:       files,
		viewport:    manifest.Viewport,
		log:         log,
	}
}

func (a *recordingAdapter) executionPlan(startedAt time.Time) autocontracts.ExecutionPlan {
	return autocontracts.ExecutionPlan{
		SchemaVersion:  autocontracts.ExecutionPlanSchemaVersion,
		PayloadVersion: autocontracts.PayloadVersion,
		ExecutionID:    a.executionID,
		WorkflowID:     a.workflowID,
		Instructions:   nil,
		Metadata: map[string]any{
			"origin":   "extension",
			"runId":    a.manifest.RunID,
			"viewport": a.viewport,
			"recorded": a.manifest.RecordedAt,
			"extension": func() any {
				if a.manifest.Extension == nil {
					return nil
				}
				return a.manifest.Extension
			}(),
		},
		CreatedAt: startedAt.UTC(),
	}
}

func (a *recordingAdapter) outcomeForFrame(index int, frame recordingFrame, startTime time.Time, durationMs int) (autocontracts.StepOutcome, bool, error) {
	frame.normalise()

	startedAt := startTime.Add(time.Duration(frame.TimestampMs()) * time.Millisecond)
	completedAt := startedAt.Add(time.Duration(durationMs) * time.Millisecond).UTC()

	outcome := autocontracts.StepOutcome{
		SchemaVersion:  autocontracts.StepOutcomeSchemaVersion,
		PayloadVersion: autocontracts.PayloadVersion,
		ExecutionID:    a.executionID,
		StepIndex:      index,
		Attempt:        1,
		NodeID:         frame.NodeID(),
		StepType:       frame.StepTypeOrDefault(),
		Success:        true,
		StartedAt:      startedAt.UTC(),
		CompletedAt:    &completedAt,
		DurationMs:     durationMs,
		FinalURL:       frame.FinalURL,
		Assertion:      toAssertionOutcome(frame.Assertion),
		ElementBoundingBox: func() *autocontracts.BoundingBox {
			if frame.FocusedElement != nil && frame.FocusedElement.BoundingBox != nil {
				return frame.FocusedElement.BoundingBox.toContracts()
			}
			return nil
		}(),
		ClickPosition:    frame.ClickPositionPoint(),
		FocusedElement:   frame.focusedElementContracts(),
		HighlightRegions: frame.HighlightRegions.toContracts(),
		MaskRegions:      frame.MaskRegions.toMaskContracts(),
		ZoomFactor:       frame.ZoomFactor,
		ConsoleLogs:      toConsoleLogs(frame.ConsoleLogs),
		Network:          toNetworkEvents(frame.NetworkEvents),
		CursorTrail:      frame.CursorTrailPoints(),
		ExtractedData:    frame.Payload,
		DOMSnapshot:      buildDOMSnapshot(frame, startedAt),
		Notes:            map[string]string{"origin": "extension"},
	}

	screenshotSet, err := a.attachScreenshot(&outcome, frame, index)
	if err != nil {
		return autocontracts.StepOutcome{}, false, err
	}

	return outcome, screenshotSet, nil
}

func buildDOMSnapshot(frame recordingFrame, startedAt time.Time) *autocontracts.DOMSnapshot {
	if frame.DomSnapshotHTML == "" && frame.DomSnapshotPreview == "" {
		return nil
	}
	return &autocontracts.DOMSnapshot{
		HTML:        frame.DomSnapshotHTML,
		Preview:     frame.DomSnapshotPreview,
		CollectedAt: startedAt.UTC(),
		Hash:        "",
		Truncated:   false,
	}
}

func (a *recordingAdapter) attachScreenshot(outcome *autocontracts.StepOutcome, frame recordingFrame, index int) (bool, error) {
	if outcome == nil || strings.TrimSpace(frame.Screenshot) == "" {
		return false, nil
	}

	entry := a.files[normalizeArchiveName(frame.Screenshot)]
	if entry == nil {
		if a.log != nil {
			a.log.WithField("path", frame.Screenshot).Warn("Recording frame referenced screenshot that was not present in archive")
		}
		return false, nil
	}

	if entry.UncompressedSize64 > maxRecordingAssetBytes {
		return false, fmt.Errorf("frame screenshot exceeds maximum size (%d bytes)", maxRecordingAssetBytes)
	}

	data, contentType, err := readZipFile(entry)
	if err != nil {
		return false, fmt.Errorf("failed to read screenshot asset: %w", err)
	}
	if len(data) > maxRecordingAssetBytes {
		return false, fmt.Errorf("frame screenshot exceeds maximum size (%d bytes)", maxRecordingAssetBytes)
	}

	width, height := decodeDimensions(data)
	if width == 0 {
		width = a.viewport.Width
	}
	if height == 0 {
		height = a.viewport.Height
	}

	outcome.Screenshot = &autocontracts.Screenshot{
		Data:        data,
		MediaType:   contentType,
		CaptureTime: outcome.CompletedAt.UTC(),
		Width:       width,
		Height:      height,
		Source:      "recording",
	}
	return true, nil
}

func toAssertionOutcome(assertion *runtimeAssertion) *autocontracts.AssertionOutcome {
	if assertion == nil {
		return nil
	}
	return &autocontracts.AssertionOutcome{
		Mode:          assertion.Mode,
		Selector:      assertion.Selector,
		Expected:      assertion.Expected,
		Actual:        assertion.Actual,
		Success:       assertion.Success,
		Negated:       assertion.Negated,
		CaseSensitive: assertion.CaseSensitive,
		Message:       assertion.Message,
	}
}

func toConsoleLogs(logs []runtimeConsoleLog) []autocontracts.ConsoleLogEntry {
	if len(logs) == 0 {
		return nil
	}
	entries := make([]autocontracts.ConsoleLogEntry, 0, len(logs))
	for _, l := range logs {
		entries = append(entries, autocontracts.ConsoleLogEntry{
			Type:      l.Type,
			Text:      l.Text,
			Timestamp: time.UnixMilli(l.Timestamp).UTC(),
		})
	}
	return entries
}

func toNetworkEvents(events []runtimeNetworkEvent) []autocontracts.NetworkEvent {
	if len(events) == 0 {
		return nil
	}
	out := make([]autocontracts.NetworkEvent, 0, len(events))
	for _, ev := range events {
		out = append(out, autocontracts.NetworkEvent{
			Type:         ev.Type,
			URL:          ev.URL,
			Method:       ev.Method,
			ResourceType: ev.ResourceType,
			Status:       ev.Status,
			OK:           ev.OK,
			Failure:      ev.Failure,
			Timestamp:    time.UnixMilli(ev.Timestamp).UTC(),
		})
	}
	return out
}

// runtime* aliases keep manifest parsing decoupled from automation contracts.
type runtimeConsoleLog struct {
	Type      string `json:"type"`
	Text      string `json:"text"`
	Timestamp int64  `json:"timestamp"`
}

type runtimeNetworkEvent struct {
	Type         string `json:"type"`
	URL          string `json:"url"`
	Method       string `json:"method,omitempty"`
	ResourceType string `json:"resourceType,omitempty"`
	Status       int    `json:"status,omitempty"`
	OK           bool   `json:"ok,omitempty"`
	Failure      string `json:"failure,omitempty"`
	Timestamp    int64  `json:"timestamp"`
}

type runtimeAssertion struct {
	Mode          string `json:"mode,omitempty"`
	Selector      string `json:"selector,omitempty"`
	Expected      any    `json:"expected,omitempty"`
	Actual        any    `json:"actual,omitempty"`
	Success       bool   `json:"success"`
	Negated       bool   `json:"negated,omitempty"`
	CaseSensitive bool   `json:"caseSensitive,omitempty"`
	Message       string `json:"message,omitempty"`
}

func init() {
	// keep image import alive for decodeDimensions
	_ = image.Rect(0, 0, 0, 0)
}
