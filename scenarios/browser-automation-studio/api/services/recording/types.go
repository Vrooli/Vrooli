package recording

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
)

// recordingManifest represents the structure of a Chrome extension recording manifest.
type recordingManifest struct {
	RunID        string            `json:"runId"`
	ProjectID    string            `json:"projectId"`
	ProjectName  string            `json:"projectName"`
	WorkflowID   string            `json:"workflowId"`
	WorkflowName string            `json:"workflowName"`
	RecordedAt   string            `json:"recordedAt"`
	Extension    map[string]any    `json:"extension"`
	Viewport     recordingViewport `json:"viewport"`
	Frames       []recordingFrame  `json:"frames"`
	Metadata     map[string]any    `json:"metadata"`
}

// recordingViewport captures the dimensions of the recording viewport.
type recordingViewport struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// recordingFrame represents a single frame in a recording with all associated metadata.
type recordingFrame struct {
	Index              int                    `json:"index"`
	Timestamp          any                    `json:"timestamp"`
	DurationMs         int                    `json:"durationMs"`
	Event              string                 `json:"event"`
	StepType           string                 `json:"stepType"`
	Node               string                 `json:"nodeId"`
	Title              string                 `json:"title"`
	Screenshot         string                 `json:"screenshot"`
	FinalURL           string                 `json:"url"`
	Cursor             recordingCursor        `json:"cursor"`
	CursorTrail        []recordingPoint       `json:"cursorTrail"`
	ClickPosition      *recordingPoint        `json:"clickPosition"`
	Click              recordingPoint         `json:"click"`
	FocusedElement     *recordingElementFocus `json:"focusedElement"`
	HighlightRegions   recordingRegions       `json:"highlightRegions"`
	MaskRegions        recordingRegions       `json:"maskRegions"`
	ZoomFactor         float64                `json:"zoomFactor"`
	ConsoleLogs        []runtimeConsoleLog    `json:"-"`
	NetworkEvents      []runtimeNetworkEvent  `json:"-"`
	ConsoleLogsRaw     json.RawMessage        `json:"consoleLogs"`
	ConsoleRaw         json.RawMessage        `json:"console"`
	NetworkRaw         json.RawMessage        `json:"network"`
	Assertion          *runtimeAssertion      `json:"assertion"`
	DomSnapshotHTML    string                 `json:"domSnapshotHtml"`
	DomSnapshotPreview string                 `json:"domSnapshotPreview"`
	Payload            map[string]any         `json:"payload"`
}

// recordingCursor represents cursor path data in a recording.
type recordingCursor struct {
	Path [][]float64 `json:"path"`
}

// recordingPoint represents a 2D point (typically cursor or click position).
type recordingPoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// recordingElementFocus captures information about a focused element during recording.
type recordingElementFocus struct {
	Selector    string                `json:"selector"`
	BoundingBox *recordingBoundingBox `json:"boundingBox"`
}

// recordingBoundingBox represents the dimensions and position of an element.
type recordingBoundingBox struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// recordingRegions is a collection of regions (highlights or masks).
type recordingRegions []recordingRegion

// recordingRegion represents a single highlighted or masked region.
type recordingRegion struct {
	Selector    string                `json:"selector"`
	BoundingBox *recordingBoundingBox `json:"boundingBox"`
	Padding     int                   `json:"padding"`
	Color       string                `json:"color"`
	Opacity     float64               `json:"opacity"`
}

// normalise sorts frames by timestamp and ensures consistent frame indexing.
func (m *recordingManifest) normalise() {
	if len(m.Frames) == 0 {
		return
	}
	sort.Slice(m.Frames, func(i, j int) bool {
		return m.Frames[i].TimestampMs() < m.Frames[j].TimestampMs()
	})
	for idx := range m.Frames {
		if m.Frames[idx].Index == 0 {
			m.Frames[idx].Index = idx
		}
		m.Frames[idx].normalise()
	}
}

// normalise ensures consistent cursor, highlight, mask, console, and network data.
func (f *recordingFrame) normalise() {
	if len(f.Cursor.Path) == 0 && len(f.CursorTrail) == 0 && f.ClickPositionPoint() == nil {
		if cp := f.Click.toRuntimePoint(); cp != nil {
			f.CursorTrail = append(f.CursorTrail, f.Click)
		}
	}
	f.HighlightRegions = f.HighlightRegions.normalise()
	f.MaskRegions = f.MaskRegions.normalise()
	if len(f.ConsoleLogs) == 0 {
		f.ConsoleLogs = f.parseConsole()
	}
	if len(f.NetworkEvents) == 0 {
		f.NetworkEvents = f.parseNetwork()
	}
}

// TimestampMs extracts the timestamp in milliseconds, handling multiple type representations.
func (f *recordingFrame) TimestampMs() int {
	switch v := f.Timestamp.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	case json.Number:
		if iv, err := v.Int64(); err == nil {
			return int(iv)
		}
	}
	return f.Index * recordingDefaultFrameDurationMs()
}

// NodeID returns the node ID for this frame, generating one if not present.
func (f *recordingFrame) NodeID() string {
	if strings.TrimSpace(f.Node) != "" {
		return f.Node
	}
	return fmt.Sprintf("recording-frame-%d", f.Index+1)
}

// StepTypeOrDefault returns the step type, deriving it from event if not explicitly set.
func (f *recordingFrame) StepTypeOrDefault() string {
	if strings.TrimSpace(f.StepType) != "" {
		return f.StepType
	}
	if strings.TrimSpace(f.Event) != "" {
		return strings.ToLower(strings.ReplaceAll(f.Event, " ", "_"))
	}
	return "recording_frame"
}

// CursorTrailPoints combines cursor trail and path data into contract-compatible positions.
func (f *recordingFrame) CursorTrailPoints() []autocontracts.CursorPosition {
	points := []autocontracts.CursorPosition{}
	for _, point := range f.CursorTrail {
		if pt := point.toRuntimePoint(); pt != nil {
			points = append(points, autocontracts.CursorPosition{Point: pt})
		}
	}
	for _, pathPoint := range f.Cursor.Path {
		if len(pathPoint) != 2 {
			continue
		}
		pt := &autocontracts.Point{X: pathPoint[0], Y: pathPoint[1]}
		points = append(points, autocontracts.CursorPosition{
			Point: pt,
		})
	}
	return points
}

// ClickPositionPoint returns the click position as a contract-compatible point.
func (f *recordingFrame) ClickPositionPoint() *autocontracts.Point {
	if f.ClickPosition == nil {
		return nil
	}
	return f.ClickPosition.toRuntimePoint()
}

// focusedElementContracts converts the focused element to contract format.
func (f *recordingFrame) focusedElementContracts() *autocontracts.ElementFocus {
	if f.FocusedElement == nil {
		return nil
	}
	return f.FocusedElement.toContracts()
}

// parseConsole attempts to unmarshal console logs from raw JSON data.
func (f *recordingFrame) parseConsole() []runtimeConsoleLog {
	if len(f.ConsoleLogsRaw) > 0 {
		var logs []runtimeConsoleLog
		if err := json.Unmarshal(f.ConsoleLogsRaw, &logs); err == nil {
			return logs
		}
	}
	if len(f.ConsoleRaw) > 0 {
		var logs []runtimeConsoleLog
		if err := json.Unmarshal(f.ConsoleRaw, &logs); err == nil {
			return logs
		}
	}
	return nil
}

// parseNetwork attempts to unmarshal network events from raw JSON data.
func (f *recordingFrame) parseNetwork() []runtimeNetworkEvent {
	if len(f.NetworkRaw) == 0 {
		return nil
	}
	var events []runtimeNetworkEvent
	if err := json.Unmarshal(f.NetworkRaw, &events); err == nil {
		return events
	}
	return nil
}

// toRuntimePoint converts a recording point to a contract point, returning nil for zero points.
func (p recordingPoint) toRuntimePoint() *autocontracts.Point {
	if p.X == 0 && p.Y == 0 {
		return nil
	}
	return &autocontracts.Point{X: p.X, Y: p.Y}
}

// normalise sorts regions by selector for consistent ordering.
func (r recordingRegions) normalise() recordingRegions {
	sort.Slice(r, func(i, j int) bool {
		return r[i].Selector < r[j].Selector
	})
	return r
}

// toContracts converts recording regions to contract highlight regions.
func (r recordingRegions) toContracts() []*autocontracts.HighlightRegion {
	result := make([]*autocontracts.HighlightRegion, 0, len(r))
	for _, region := range r {
		runtimeRegion := &autocontracts.HighlightRegion{Selector: region.Selector}
		if region.BoundingBox != nil {
			runtimeRegion.BoundingBox = region.BoundingBox.toContracts()
		}
		if region.Padding != 0 {
			runtimeRegion.Padding = int32(region.Padding)
		}
			if region.Color != "" {
				color := region.Color
				runtimeRegion.CustomRgba = &color
			}
		result = append(result, runtimeRegion)
	}
	return result
}

// toMaskContracts converts recording regions to contract mask regions.
func (r recordingRegions) toMaskContracts() []*autocontracts.MaskRegion {
	result := make([]*autocontracts.MaskRegion, 0, len(r))
	for _, region := range r {
		maskRegion := &autocontracts.MaskRegion{Selector: region.Selector}
		if region.BoundingBox != nil {
			maskRegion.BoundingBox = region.BoundingBox.toContracts()
		}
		if region.Opacity != 0 {
			maskRegion.Opacity = region.Opacity
		}
		result = append(result, maskRegion)
	}
	return result
}

// toContracts converts a recording bounding box to a contract bounding box.
func (b *recordingBoundingBox) toContracts() *autocontracts.BoundingBox {
	if b == nil {
		return nil
	}
	return &autocontracts.BoundingBox{
		X:      b.X,
		Y:      b.Y,
		Width:  b.Width,
		Height: b.Height,
	}
}

// toContracts converts a recording element focus to a contract element focus.
func (e *recordingElementFocus) toContracts() *autocontracts.ElementFocus {
	if e == nil {
		return nil
	}
	return &autocontracts.ElementFocus{
		Selector:    e.Selector,
		BoundingBox: e.BoundingBox.toContracts(),
	}
}
