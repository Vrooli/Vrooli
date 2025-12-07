package execution

import (
	"fmt"
	"strings"
	"time"

	"test-genie/internal/playbooks/types"

	browser_automation_studio_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ParsedTimeline contains all extracted data from a BAS timeline response.
// It provides an ergonomic Go interface over the proto timeline.
//
// The Proto field contains the canonical proto timeline for callers needing:
// - Full access to all proto fields (some are not exposed in ParsedTimeline)
// - Type-safe access to nested proto messages
// - Rich error diagnostics via types.PlaybookExecutionError.WithTimeline()
//
// Use FromProtoTimeline() when you already have a parsed proto to avoid re-parsing.
type ParsedTimeline struct {
	// Proto contains the parsed proto timeline for callers needing raw access.
	// This is the canonical source of truth - all other fields are derived from it.
	Proto *browser_automation_studio_v1.ExecutionTimeline `json:"-"`
	// ExecutionID is the BAS execution ID.
	ExecutionID string `json:"execution_id"`
	// WorkflowID is the BAS workflow ID (if saved).
	WorkflowID string `json:"workflow_id,omitempty"`
	// Status is the overall execution status.
	Status string `json:"status"`
	// Progress is the execution progress (0-100).
	Progress int `json:"progress"`
	// StartedAt is when execution started.
	StartedAt *time.Time `json:"started_at,omitempty"`
	// CompletedAt is when execution completed.
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// Frames contains per-step execution data.
	Frames []ParsedFrame `json:"frames"`
	// Logs contains console/execution logs.
	Logs []ParsedLog `json:"logs"`
	// Assertions contains assertion results.
	Assertions []ParsedAssertion `json:"assertions"`

	// FinalDOM is the DOM snapshot from the last frame (or failed frame).
	FinalDOM string `json:"final_dom,omitempty"`
	// FinalDOMPreview is a truncated preview of the DOM.
	FinalDOMPreview string `json:"final_dom_preview,omitempty"`
	// FailedFrame is the frame where execution failed (if any).
	FailedFrame *ParsedFrame `json:"failed_frame,omitempty"`

	// Summary contains aggregate statistics.
	Summary TimelineSummary `json:"summary"`
}

// ParsedFrame contains data for a single execution step.
type ParsedFrame struct {
	StepIndex   int               `json:"step_index"`
	NodeID      string            `json:"node_id"`
	StepType    string            `json:"step_type"`
	Status      string            `json:"status"`
	Success     bool              `json:"success"`
	DurationMs  int               `json:"duration_ms,omitempty"`
	Error       string            `json:"error,omitempty"`
	FinalURL    string            `json:"final_url,omitempty"`
	Progress    int               `json:"progress,omitempty"`
	Screenshot  *FrameScreenshot  `json:"screenshot,omitempty"`
	DOMSnapshot *FrameDOMSnapshot `json:"dom_snapshot,omitempty"`
	Assertion   *ParsedAssertion  `json:"assertion,omitempty"`
}

// FrameScreenshot contains screenshot info embedded in a frame.
type FrameScreenshot struct {
	ArtifactID   string `json:"artifact_id"`
	URL          string `json:"url,omitempty"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
	Width        int    `json:"width,omitempty"`
	Height       int    `json:"height,omitempty"`
}

// FrameDOMSnapshot contains DOM snapshot info.
type FrameDOMSnapshot struct {
	ArtifactID string `json:"artifact_id,omitempty"`
	Preview    string `json:"preview,omitempty"`
	StorageURL string `json:"storage_url,omitempty"`
}

// ParsedLog contains a single log entry.
type ParsedLog struct {
	ID        string    `json:"id"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	StepName  string    `json:"step_name,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// ParsedAssertion contains assertion result details.
type ParsedAssertion struct {
	StepIndex     int    `json:"step_index"`
	NodeID        string `json:"node_id,omitempty"`
	Passed        bool   `json:"passed"`
	AssertionType string `json:"assertion_type,omitempty"`
	Selector      string `json:"selector,omitempty"`
	Expected      string `json:"expected,omitempty"`
	Actual        string `json:"actual,omitempty"`
	Message       string `json:"message,omitempty"`
	Error         string `json:"error,omitempty"`
}

// ParseFullTimeline parses the complete timeline response from BAS.
// It extracts all structured data including logs, DOM snapshots, assertions.
func ParseFullTimeline(data []byte) (*ParsedTimeline, error) {
	if len(data) == 0 {
		return &ParsedTimeline{}, nil
	}

	var timeline browser_automation_studio_v1.ExecutionTimeline
	if err := protoJSONUnmarshal.Unmarshal(data, &timeline); err != nil {
		return nil, &TimelineParseError{
			RawData: data,
			Cause:   err,
		}
	}

	parsed := &ParsedTimeline{
		Proto:       &timeline,
		ExecutionID: timeline.GetExecutionId(),
		WorkflowID:  timeline.GetWorkflowId(),
		Status:      types.ExecutionStatusToString(timeline.GetStatus()),
		Progress:    int(timeline.GetProgress()),
		StartedAt:   timestampToTime(timeline.GetStartedAt()),
		CompletedAt: timestampToTime(timeline.GetCompletedAt()),
		Frames:      make([]ParsedFrame, 0, len(timeline.GetFrames())),
		Logs:        make([]ParsedLog, 0, len(timeline.GetLogs())),
		Assertions:  make([]ParsedAssertion, 0),
	}

	// Parse frames and extract assertions/DOM
	var lastDOMPreview string
	var lastDOMFull string
	for _, rf := range timeline.GetFrames() {
		frame := ParsedFrame{
			StepIndex: int(rf.GetStepIndex()),
			NodeID:    rf.GetNodeId(),
			StepType:  types.StepTypeToString(rf.GetStepType()),
			Status:    types.StepStatusToString(rf.GetStatus()),
			Success:   rf.GetSuccess(),
			DurationMs: func() int {
				if rf.GetTotalDurationMs() > 0 {
					return int(rf.GetTotalDurationMs())
				}
				return int(rf.GetDurationMs())
			}(),
			Error:    rf.GetError(),
			FinalURL: rf.GetFinalUrl(),
			Progress: int(rf.GetProgress()),
		}

		// Extract screenshot info
		if rf.GetScreenshot() != nil {
			frame.Screenshot = &FrameScreenshot{
				ArtifactID:   rf.Screenshot.GetArtifactId(),
				URL:          rf.Screenshot.GetUrl(),
				ThumbnailURL: rf.Screenshot.GetThumbnailUrl(),
				Width:        int(rf.Screenshot.GetWidth()),
				Height:       int(rf.Screenshot.GetHeight()),
			}
		}

		// Extract DOM snapshot info
		if rf.GetDomSnapshot() != nil || rf.GetDomSnapshotPreview() != "" {
			frame.DOMSnapshot = &FrameDOMSnapshot{}
			if rf.GetDomSnapshot() != nil {
				frame.DOMSnapshot.ArtifactID = rf.DomSnapshot.GetId()
				frame.DOMSnapshot.StorageURL = rf.DomSnapshot.GetStorageUrl()
				if html := extractHTML(rf.DomSnapshot.GetPayload()); html != "" {
					lastDOMFull = html
				}
			}
			if rf.GetDomSnapshotPreview() != "" {
				frame.DOMSnapshot.Preview = rf.GetDomSnapshotPreview()
				lastDOMPreview = rf.GetDomSnapshotPreview()
			}
		}

		// Extract assertion info
		if rf.GetStepType() == browser_automation_studio_v1.StepType_STEP_TYPE_ASSERT || rf.Assertion != nil {
			assertion := ParsedAssertion{
				StepIndex: int(rf.GetStepIndex()),
				NodeID:    rf.GetNodeId(),
				Passed:    rf.GetSuccess(),
				Error:     rf.GetError(),
			}
			if rf.Assertion != nil {
				assertion.AssertionType = rf.Assertion.GetMode()
				assertion.Selector = rf.Assertion.GetSelector()
				assertion.Expected = valueToString(rf.Assertion.GetExpected())
				assertion.Actual = valueToString(rf.Assertion.GetActual())
				assertion.Message = rf.Assertion.GetMessage()
				assertion.Passed = rf.Assertion.GetSuccess()
			}
			frame.Assertion = &assertion
			parsed.Assertions = append(parsed.Assertions, assertion)
		}

		// Track failed frame
		if strings.EqualFold(frame.Status, "failed") || frame.Error != "" {
			frameCopy := frame
			parsed.FailedFrame = &frameCopy
		}

		parsed.Frames = append(parsed.Frames, frame)
	}

	// Set final DOM from last frame or failed frame
	parsed.FinalDOMPreview = lastDOMPreview
	parsed.FinalDOM = lastDOMFull

	// Parse logs
	for _, rl := range timeline.GetLogs() {
		parsed.Logs = append(parsed.Logs, ParsedLog{
			ID:        rl.GetId(),
			Level:     types.LogLevelToString(rl.GetLevel()),
			Message:   rl.GetMessage(),
			StepName:  rl.GetStepName(),
			Timestamp: timestampValue(rl.GetTimestamp()),
		})
	}

	// Calculate summary
	parsed.Summary = calculateSummary(parsed)

	return parsed, nil
}

// FromProtoTimeline creates a ParsedTimeline from an already-parsed proto timeline.
// Use this when you have a *browser_automation_studio_v1.ExecutionTimeline to avoid re-parsing JSON.
func FromProtoTimeline(timeline *browser_automation_studio_v1.ExecutionTimeline) *ParsedTimeline {
	if timeline == nil {
		return &ParsedTimeline{}
	}

	parsed := &ParsedTimeline{
		Proto:       timeline,
		ExecutionID: timeline.GetExecutionId(),
		WorkflowID:  timeline.GetWorkflowId(),
		Status:      types.ExecutionStatusToString(timeline.GetStatus()),
		Progress:    int(timeline.GetProgress()),
		StartedAt:   timestampToTime(timeline.GetStartedAt()),
		CompletedAt: timestampToTime(timeline.GetCompletedAt()),
		Frames:      make([]ParsedFrame, 0, len(timeline.GetFrames())),
		Logs:        make([]ParsedLog, 0, len(timeline.GetLogs())),
		Assertions:  make([]ParsedAssertion, 0),
	}

	// Convert frames
	var lastDOMPreview string
	var lastDOMFull string
	for _, rf := range timeline.GetFrames() {
		frame := frameFromProto(rf)

		// Track DOM snapshots
		if rf.GetDomSnapshot() != nil {
			if html := extractHTML(rf.DomSnapshot.GetPayload()); html != "" {
				lastDOMFull = html
			}
		}
		if rf.GetDomSnapshotPreview() != "" {
			lastDOMPreview = rf.GetDomSnapshotPreview()
		}

		// Extract assertions
		if rf.GetStepType() == browser_automation_studio_v1.StepType_STEP_TYPE_ASSERT || rf.Assertion != nil {
			if frame.Assertion != nil {
				parsed.Assertions = append(parsed.Assertions, *frame.Assertion)
			}
		}

		// Track failed frame
		if strings.EqualFold(frame.Status, "failed") || frame.Error != "" {
			frameCopy := frame
			parsed.FailedFrame = &frameCopy
		}

		parsed.Frames = append(parsed.Frames, frame)
	}

	parsed.FinalDOMPreview = lastDOMPreview
	parsed.FinalDOM = lastDOMFull

	// Convert logs
	for _, rl := range timeline.GetLogs() {
		parsed.Logs = append(parsed.Logs, ParsedLog{
			ID:        rl.GetId(),
			Level:     types.LogLevelToString(rl.GetLevel()),
			Message:   rl.GetMessage(),
			StepName:  rl.GetStepName(),
			Timestamp: timestampValue(rl.GetTimestamp()),
		})
	}

	parsed.Summary = calculateSummary(parsed)
	return parsed
}

// frameFromProto converts a proto TimelineFrame to ParsedFrame.
func frameFromProto(rf *browser_automation_studio_v1.TimelineFrame) ParsedFrame {
	frame := ParsedFrame{
		StepIndex: int(rf.GetStepIndex()),
		NodeID:    rf.GetNodeId(),
		StepType:  types.StepTypeToString(rf.GetStepType()),
		Status:    types.StepStatusToString(rf.GetStatus()),
		Success:   rf.GetSuccess(),
		DurationMs: func() int {
			if rf.GetTotalDurationMs() > 0 {
				return int(rf.GetTotalDurationMs())
			}
			return int(rf.GetDurationMs())
		}(),
		Error:    rf.GetError(),
		FinalURL: rf.GetFinalUrl(),
		Progress: int(rf.GetProgress()),
	}

	// Extract screenshot info
	if rf.GetScreenshot() != nil {
		frame.Screenshot = &FrameScreenshot{
			ArtifactID:   rf.Screenshot.GetArtifactId(),
			URL:          rf.Screenshot.GetUrl(),
			ThumbnailURL: rf.Screenshot.GetThumbnailUrl(),
			Width:        int(rf.Screenshot.GetWidth()),
			Height:       int(rf.Screenshot.GetHeight()),
		}
	}

	// Extract DOM snapshot info
	if rf.GetDomSnapshot() != nil || rf.GetDomSnapshotPreview() != "" {
		frame.DOMSnapshot = &FrameDOMSnapshot{}
		if rf.GetDomSnapshot() != nil {
			frame.DOMSnapshot.ArtifactID = rf.DomSnapshot.GetId()
			frame.DOMSnapshot.StorageURL = rf.DomSnapshot.GetStorageUrl()
		}
		if rf.GetDomSnapshotPreview() != "" {
			frame.DOMSnapshot.Preview = rf.GetDomSnapshotPreview()
		}
	}

	// Extract assertion info
	if rf.GetStepType() == browser_automation_studio_v1.StepType_STEP_TYPE_ASSERT || rf.Assertion != nil {
		assertion := ParsedAssertion{
			StepIndex: int(rf.GetStepIndex()),
			NodeID:    rf.GetNodeId(),
			Passed:    rf.GetSuccess(),
			Error:     rf.GetError(),
		}
		if rf.Assertion != nil {
			assertion.AssertionType = rf.Assertion.GetMode()
			assertion.Selector = rf.Assertion.GetSelector()
			assertion.Expected = valueToString(rf.Assertion.GetExpected())
			assertion.Actual = valueToString(rf.Assertion.GetActual())
			assertion.Message = rf.Assertion.GetMessage()
			assertion.Passed = rf.Assertion.GetSuccess()
		}
		frame.Assertion = &assertion
	}

	return frame
}

// ProtoFrameAt returns the proto frame at the given index for direct proto access.
// Returns nil if index is out of bounds.
func (p *ParsedTimeline) ProtoFrameAt(index int) *browser_automation_studio_v1.TimelineFrame {
	if p.Proto == nil {
		return nil
	}
	frames := p.Proto.GetFrames()
	if index < 0 || index >= len(frames) {
		return nil
	}
	return frames[index]
}

// FailedProtoFrame returns the first failed proto frame for rich error diagnostics.
// Returns nil if no frame failed.
func (p *ParsedTimeline) FailedProtoFrame() *browser_automation_studio_v1.TimelineFrame {
	if p.Proto == nil {
		return nil
	}
	for _, frame := range p.Proto.GetFrames() {
		if !frame.GetSuccess() || frame.GetError() != "" {
			return frame
		}
	}
	return nil
}

// calculateSummary computes aggregate statistics from parsed timeline.
func calculateSummary(parsed *ParsedTimeline) TimelineSummary {
	summary := TimelineSummary{
		TotalSteps: len(parsed.Frames),
	}

	for _, frame := range parsed.Frames {
		if frame.StepType == "assert" {
			summary.TotalAsserts++
			if frame.Success {
				summary.AssertsPassed++
			}
		}
	}

	return summary
}

func timestampToTime(ts *timestamppb.Timestamp) *time.Time {
	if ts == nil {
		return nil
	}
	t := ts.AsTime()
	return &t
}

func timestampValue(ts *timestamppb.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	return ts.AsTime()
}

func extractHTML(payload map[string]*structpb.Value) string {
	if payload == nil {
		return ""
	}
	if val, ok := payload["html"]; ok {
		if s := val.GetStringValue(); s != "" {
			return s
		}
		if str, ok := val.AsInterface().(string); ok {
			return str
		}
	}
	return ""
}

func valueToString(v *structpb.Value) string {
	if v == nil {
		return ""
	}
	if s := v.GetStringValue(); s != "" {
		return s
	}
	switch val := v.AsInterface().(type) {
	case string:
		return val
	default:
		return fmt.Sprint(val)
	}
}
