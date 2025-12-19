// Package events provides timeline building utilities shared between recording and execution.
//
// This file contains common helpers used by both RecordedActionToTimelineEntry (recording path)
// and StepOutcomeToTimelineEntry (execution path). By consolidating these helpers, we ensure
// consistent handling of telemetry data across both modes.
//
// Architecture:
//   - Recording: RecordedAction → TimelineEntry (via recording_convert.go)
//   - Execution: StepOutcome → TimelineEntry (via unified_convert.go)
//   - Both use helpers from this file for common conversions
package events

import (
	"strings"

	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/internal/enums"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basdomain "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/domain"
)

// =============================================================================
// CONSOLE LOG CONVERSION
// =============================================================================

// ConvertConsoleLogs converts a slice of contracts.ConsoleLogEntry to proto format.
// Used by both recording and execution paths to ensure consistent console log handling.
func ConvertConsoleLogs(logs []contracts.ConsoleLogEntry) []*basdomain.ConsoleLogEntry {
	if len(logs) == 0 {
		return nil
	}
	result := make([]*basdomain.ConsoleLogEntry, 0, len(logs))
	for _, log := range logs {
		entry := &basdomain.ConsoleLogEntry{
			Level: enums.StringToLogLevel(log.Type),
			Text:  log.Text,
		}
		if log.Stack != "" {
			entry.Stack = &log.Stack
		}
		if log.Location != "" {
			entry.Location = &log.Location
		}
		if !log.Timestamp.IsZero() {
			entry.Timestamp = contracts.TimeToTimestamp(log.Timestamp)
		}
		result = append(result, entry)
	}
	return result
}

// =============================================================================
// NETWORK EVENT CONVERSION
// =============================================================================

// ConvertNetworkEvents converts a slice of contracts.NetworkEvent to proto format.
// Used by both recording and execution paths to ensure consistent network event handling.
func ConvertNetworkEvents(events []contracts.NetworkEvent) []*basdomain.NetworkEvent {
	if len(events) == 0 {
		return nil
	}
	result := make([]*basdomain.NetworkEvent, 0, len(events))
	for _, net := range events {
		event := &basdomain.NetworkEvent{
			Type: enums.StringToNetworkEventType(net.Type),
			Url:  net.URL,
		}
		if net.Method != "" {
			event.Method = &net.Method
		}
		if net.ResourceType != "" {
			event.ResourceType = &net.ResourceType
		}
		if net.Status != 0 {
			status := int32(net.Status)
			event.Status = &status
		}
		if net.OK {
			event.Ok = &net.OK
		}
		if net.Failure != "" {
			event.Failure = &net.Failure
		}
		if !net.Timestamp.IsZero() {
			event.Timestamp = contracts.TimeToTimestamp(net.Timestamp)
		}
		result = append(result, event)
	}
	return result
}

// =============================================================================
// GEOMETRY HELPERS
// =============================================================================

// BoundingBoxFromCoords creates a BoundingBox proto from coordinate values.
// Returns nil if all values are zero (indicating no bounding box data).
func BoundingBoxFromCoords(x, y, width, height float64) *basbase.BoundingBox {
	if x == 0 && y == 0 && width == 0 && height == 0 {
		return nil
	}
	return &basbase.BoundingBox{
		X:      x,
		Y:      y,
		Width:  width,
		Height: height,
	}
}

// PointFromCoords creates a Point proto from coordinate values.
// Returns nil if both values are zero.
func PointFromCoords(x, y float64) *basbase.Point {
	if x == 0 && y == 0 {
		return nil
	}
	return &basbase.Point{
		X: x,
		Y: y,
	}
}

// =============================================================================
// STRING HELPERS
// =============================================================================

// Capitalize capitalizes the first letter of a string.
// Used for generating human-readable labels.
func Capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// Truncate truncates a string to maxLen and adds "..." if truncated.
// Used for generating concise labels from longer text.
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// =============================================================================
// PAYLOAD EXTRACTION HELPERS
// =============================================================================

// ExtractInt32FromPayload extracts an int32 value from a payload map.
// Handles various numeric types that may be present in JSON-decoded maps.
func ExtractInt32FromPayload(payload map[string]any, key string) (int32, bool) {
	raw, ok := payload[key]
	if !ok {
		return 0, false
	}
	switch v := raw.(type) {
	case int:
		return int32(v), true
	case int32:
		return v, true
	case int64:
		return int32(v), true
	case float64:
		return int32(v), true
	case float32:
		return int32(v), true
	default:
		return 0, false
	}
}

// ExtractStringFromPayload extracts a string value from a payload map.
func ExtractStringFromPayload(payload map[string]any, key string) (string, bool) {
	raw, ok := payload[key]
	if !ok {
		return "", false
	}
	s, ok := raw.(string)
	return s, ok
}

// ExtractBoolFromPayload extracts a bool value from a payload map.
func ExtractBoolFromPayload(payload map[string]any, key string) (bool, bool) {
	raw, ok := payload[key]
	if !ok {
		return false, false
	}
	b, ok := raw.(bool)
	return b, ok
}

// =============================================================================
// TELEMETRY BUILDER HELPERS
// =============================================================================

// TelemetryBuilder provides a fluent interface for building ActionTelemetry.
// This helps reduce code duplication between recording and execution paths.
type TelemetryBuilder struct {
	tel *basdomain.ActionTelemetry
}

// NewTelemetryBuilder creates a new TelemetryBuilder with the given URL.
func NewTelemetryBuilder(url string) *TelemetryBuilder {
	return &TelemetryBuilder{
		tel: &basdomain.ActionTelemetry{
			Url: url,
		},
	}
}

// WithFrameID sets the frame ID.
func (b *TelemetryBuilder) WithFrameID(frameID string) *TelemetryBuilder {
	if frameID != "" {
		b.tel.FrameId = &frameID
	}
	return b
}

// WithBoundingBox sets the element bounding box.
func (b *TelemetryBuilder) WithBoundingBox(bbox *basbase.BoundingBox) *TelemetryBuilder {
	b.tel.ElementBoundingBox = bbox
	return b
}

// WithClickPosition sets the click position.
func (b *TelemetryBuilder) WithClickPosition(pos *basbase.Point) *TelemetryBuilder {
	b.tel.ClickPosition = pos
	return b
}

// WithCursorPosition sets the cursor position.
func (b *TelemetryBuilder) WithCursorPosition(pos *basbase.Point) *TelemetryBuilder {
	b.tel.CursorPosition = pos
	return b
}

// WithCursorTrail sets the cursor trail from CursorPosition slice.
func (b *TelemetryBuilder) WithCursorTrail(trail []contracts.CursorPosition) *TelemetryBuilder {
	if len(trail) > 0 {
		b.tel.CursorTrail = make([]*basbase.Point, 0, len(trail))
		for i := range trail {
			if trail[i].Point != nil {
				b.tel.CursorTrail = append(b.tel.CursorTrail, trail[i].Point)
			}
		}
	}
	return b
}

// WithScreenshot sets screenshot metadata.
func (b *TelemetryBuilder) WithScreenshot(screenshot *contracts.Screenshot) *TelemetryBuilder {
	if screenshot != nil {
		b.tel.Screenshot = &basdomain.TimelineScreenshot{
			Width:       int32(screenshot.Width),
			Height:      int32(screenshot.Height),
			ContentType: screenshot.MediaType,
		}
	}
	return b
}

// WithDOMSnapshot sets DOM snapshot data.
func (b *TelemetryBuilder) WithDOMSnapshot(snapshot *contracts.DOMSnapshot) *TelemetryBuilder {
	if snapshot != nil {
		if snapshot.Preview != "" {
			b.tel.DomSnapshotPreview = &snapshot.Preview
		}
		if snapshot.HTML != "" {
			b.tel.DomSnapshotHtml = &snapshot.HTML
		}
	}
	return b
}

// WithConsoleLogs sets console logs.
func (b *TelemetryBuilder) WithConsoleLogs(logs []contracts.ConsoleLogEntry) *TelemetryBuilder {
	b.tel.ConsoleLogs = ConvertConsoleLogs(logs)
	return b
}

// WithNetworkEvents sets network events.
func (b *TelemetryBuilder) WithNetworkEvents(events []contracts.NetworkEvent) *TelemetryBuilder {
	b.tel.NetworkEvents = ConvertNetworkEvents(events)
	return b
}

// WithZoomFactor sets the zoom factor.
func (b *TelemetryBuilder) WithZoomFactor(zoom float32) *TelemetryBuilder {
	if zoom != 0 {
		z := float64(zoom)
		b.tel.ZoomFactor = &z
	}
	return b
}

// WithHighlightRegions sets highlight regions.
func (b *TelemetryBuilder) WithHighlightRegions(regions []*basdomain.HighlightRegion) *TelemetryBuilder {
	b.tel.HighlightRegions = regions
	return b
}

// WithMaskRegions sets mask regions.
func (b *TelemetryBuilder) WithMaskRegions(regions []*basdomain.MaskRegion) *TelemetryBuilder {
	b.tel.MaskRegions = regions
	return b
}

// Build returns the constructed ActionTelemetry.
func (b *TelemetryBuilder) Build() *basdomain.ActionTelemetry {
	return b.tel
}
