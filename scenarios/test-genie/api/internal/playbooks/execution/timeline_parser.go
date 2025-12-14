package execution

import (
	"fmt"
	"strings"
	"time"

	"test-genie/internal/playbooks/types"

	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
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
	Proto *bastimeline.ExecutionTimeline `json:"-"`
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

	var timeline bastimeline.ExecutionTimeline
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
		Frames:      make([]ParsedFrame, 0, len(timeline.GetEntries())),
		Logs:        make([]ParsedLog, 0, len(timeline.GetLogs())),
		Assertions:  make([]ParsedAssertion, 0),
	}

	// Parse entries and extract assertions/DOM
	var lastDOMPreview string
	var lastDOMFull string
	for _, entry := range timeline.GetEntries() {
		frame := entryToFrame(entry)

		// Track DOM snapshots from aggregates
		if agg := entry.GetAggregates(); agg != nil {
			if agg.GetDomSnapshot() != nil {
				if html := extractHTMLFromArtifact(agg.GetDomSnapshot()); html != "" {
					lastDOMFull = html
				}
			}
			if agg.GetDomSnapshotPreview() != "" {
				lastDOMPreview = agg.GetDomSnapshotPreview()
			}
		}

		// Also check telemetry for DOM snapshots
		if tel := entry.GetTelemetry(); tel != nil {
			if tel.GetDomSnapshotHtml() != "" {
				lastDOMFull = tel.GetDomSnapshotHtml()
			}
			if tel.GetDomSnapshotPreview() != "" {
				lastDOMPreview = tel.GetDomSnapshotPreview()
			}
		}

		// Extract assertion info
		action := entry.GetAction()
		ctx := entry.GetContext()
		if action != nil && action.GetType() == basactions.ActionType_ACTION_TYPE_ASSERT {
			assertion := ParsedAssertion{
				StepIndex: int(entry.GetStepIndex()),
				NodeID:    entry.GetNodeId(),
				Passed:    ctx != nil && ctx.GetSuccess(),
				Error:     "",
			}
			if ctx != nil {
				assertion.Error = ctx.GetError()
				if ar := ctx.GetAssertion(); ar != nil {
					assertion.AssertionType = ar.GetMode().String()
					assertion.Selector = ar.GetSelector()
					assertion.Expected = jsonValueToString(ar.GetExpected())
					assertion.Actual = jsonValueToString(ar.GetActual())
					assertion.Message = ar.GetMessage()
					assertion.Passed = ar.GetSuccess()
				}
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

	// Set final DOM from last entry or failed entry
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
// Use this when you have a *bastimeline.ExecutionTimeline to avoid re-parsing JSON.
func FromProtoTimeline(timeline *bastimeline.ExecutionTimeline) *ParsedTimeline {
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
		Frames:      make([]ParsedFrame, 0, len(timeline.GetEntries())),
		Logs:        make([]ParsedLog, 0, len(timeline.GetLogs())),
		Assertions:  make([]ParsedAssertion, 0),
	}

	// Convert entries
	var lastDOMPreview string
	var lastDOMFull string
	for _, entry := range timeline.GetEntries() {
		frame := entryToFrame(entry)

		// Track DOM snapshots from aggregates
		if agg := entry.GetAggregates(); agg != nil {
			if agg.GetDomSnapshot() != nil {
				if html := extractHTMLFromArtifact(agg.GetDomSnapshot()); html != "" {
					lastDOMFull = html
				}
			}
			if agg.GetDomSnapshotPreview() != "" {
				lastDOMPreview = agg.GetDomSnapshotPreview()
			}
		}

		// Also check telemetry for DOM snapshots
		if tel := entry.GetTelemetry(); tel != nil {
			if tel.GetDomSnapshotHtml() != "" {
				lastDOMFull = tel.GetDomSnapshotHtml()
			}
			if tel.GetDomSnapshotPreview() != "" {
				lastDOMPreview = tel.GetDomSnapshotPreview()
			}
		}

		// Extract assertions
		action := entry.GetAction()
		ctx := entry.GetContext()
		if action != nil && action.GetType() == basactions.ActionType_ACTION_TYPE_ASSERT {
			if frame.Assertion != nil {
				parsed.Assertions = append(parsed.Assertions, *frame.Assertion)
			} else if ctx != nil {
				// Build assertion from context if not already set
				assertion := ParsedAssertion{
					StepIndex: int(entry.GetStepIndex()),
					NodeID:    entry.GetNodeId(),
					Passed:    ctx.GetSuccess(),
					Error:     ctx.GetError(),
				}
				if ar := ctx.GetAssertion(); ar != nil {
					assertion.AssertionType = ar.GetMode().String()
					assertion.Selector = ar.GetSelector()
					assertion.Expected = jsonValueToString(ar.GetExpected())
					assertion.Actual = jsonValueToString(ar.GetActual())
					assertion.Message = ar.GetMessage()
					assertion.Passed = ar.GetSuccess()
				}
				frame.Assertion = &assertion
				parsed.Assertions = append(parsed.Assertions, assertion)
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

// entryToFrame converts a proto TimelineEntry to ParsedFrame.
func entryToFrame(entry *bastimeline.TimelineEntry) ParsedFrame {
	frame := ParsedFrame{
		StepIndex: int(entry.GetStepIndex()),
		NodeID:    entry.GetNodeId(),
	}

	// Get action type from action definition
	if action := entry.GetAction(); action != nil {
		frame.StepType = types.ActionTypeToString(action.GetType())
	}

	// Get status and other context fields
	if ctx := entry.GetContext(); ctx != nil {
		frame.Success = ctx.GetSuccess()
		frame.Error = ctx.GetError()
	}

	// Get aggregates (status, final_url, progress)
	if agg := entry.GetAggregates(); agg != nil {
		frame.Status = types.StepStatusToString(agg.GetStatus())
		frame.FinalURL = agg.GetFinalUrl()
		frame.Progress = int(agg.GetProgress())
	}

	// Duration
	if entry.GetTotalDurationMs() > 0 {
		frame.DurationMs = int(entry.GetTotalDurationMs())
	} else {
		frame.DurationMs = int(entry.GetDurationMs())
	}

	// Extract screenshot info from telemetry
	if tel := entry.GetTelemetry(); tel != nil {
		if ss := tel.GetScreenshot(); ss != nil {
			frame.Screenshot = &FrameScreenshot{
				ArtifactID:   ss.GetArtifactId(),
				URL:          ss.GetUrl(),
				ThumbnailURL: ss.GetThumbnailUrl(),
				Width:        int(ss.GetWidth()),
				Height:       int(ss.GetHeight()),
			}
		}
	}

	// Extract DOM snapshot info from aggregates
	if agg := entry.GetAggregates(); agg != nil {
		if agg.GetDomSnapshot() != nil || agg.GetDomSnapshotPreview() != "" {
			frame.DOMSnapshot = &FrameDOMSnapshot{}
			if ds := agg.GetDomSnapshot(); ds != nil {
				frame.DOMSnapshot.ArtifactID = ds.GetId()
				frame.DOMSnapshot.StorageURL = ds.GetStorageUrl()
			}
			if agg.GetDomSnapshotPreview() != "" {
				frame.DOMSnapshot.Preview = agg.GetDomSnapshotPreview()
			}
		}
	}

	// Extract assertion info from context
	if action := entry.GetAction(); action != nil && action.GetType() == basactions.ActionType_ACTION_TYPE_ASSERT {
		if ctx := entry.GetContext(); ctx != nil {
			assertion := ParsedAssertion{
				StepIndex: int(entry.GetStepIndex()),
				NodeID:    entry.GetNodeId(),
				Passed:    ctx.GetSuccess(),
				Error:     ctx.GetError(),
			}
			if ar := ctx.GetAssertion(); ar != nil {
				assertion.AssertionType = ar.GetMode().String()
				assertion.Selector = ar.GetSelector()
				assertion.Expected = jsonValueToString(ar.GetExpected())
				assertion.Actual = jsonValueToString(ar.GetActual())
				assertion.Message = ar.GetMessage()
				assertion.Passed = ar.GetSuccess()
			}
			frame.Assertion = &assertion
		}
	}

	return frame
}

// ProtoEntryAt returns the proto entry at the given index for direct proto access.
// Returns nil if index is out of bounds.
func (p *ParsedTimeline) ProtoEntryAt(index int) *bastimeline.TimelineEntry {
	if p.Proto == nil {
		return nil
	}
	entries := p.Proto.GetEntries()
	if index < 0 || index >= len(entries) {
		return nil
	}
	return entries[index]
}

// FailedProtoEntry returns the first failed proto entry for rich error diagnostics.
// Returns nil if no entry failed.
func (p *ParsedTimeline) FailedProtoEntry() *bastimeline.TimelineEntry {
	if p.Proto == nil {
		return nil
	}
	for _, entry := range p.Proto.GetEntries() {
		ctx := entry.GetContext()
		if ctx != nil && (!ctx.GetSuccess() || ctx.GetError() != "") {
			return entry
		}
	}
	return nil
}

// ProtoFrameAt is deprecated; use ProtoEntryAt instead.
func (p *ParsedTimeline) ProtoFrameAt(index int) *bastimeline.TimelineEntry {
	return p.ProtoEntryAt(index)
}

// FailedProtoFrame is deprecated; use FailedProtoEntry instead.
func (p *ParsedTimeline) FailedProtoFrame() *bastimeline.TimelineEntry {
	return p.FailedProtoEntry()
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

// extractHTMLFromArtifact extracts HTML content from a TimelineArtifact's payload.
func extractHTMLFromArtifact(artifact *bastimeline.TimelineArtifact) string {
	if artifact == nil {
		return ""
	}
	payload := artifact.GetPayload()
	if payload == nil {
		return ""
	}
	if val, ok := payload["html"]; ok {
		return jsonValueToString(val)
	}
	return ""
}

// jsonValueToString converts a commonv1.JsonValue to a string representation.
func jsonValueToString(v *commonv1.JsonValue) string {
	if v == nil {
		return ""
	}
	switch kind := v.GetKind().(type) {
	case *commonv1.JsonValue_StringValue:
		return kind.StringValue
	case *commonv1.JsonValue_IntValue:
		return fmt.Sprintf("%d", kind.IntValue)
	case *commonv1.JsonValue_DoubleValue:
		return fmt.Sprintf("%v", kind.DoubleValue)
	case *commonv1.JsonValue_BoolValue:
		return fmt.Sprintf("%v", kind.BoolValue)
	case *commonv1.JsonValue_NullValue:
		return "null"
	default:
		return fmt.Sprintf("%v", v)
	}
}
