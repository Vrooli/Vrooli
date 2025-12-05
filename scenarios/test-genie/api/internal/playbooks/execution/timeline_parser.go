package execution

import (
	"encoding/json"
	"time"
)

// ParsedTimeline contains all extracted data from a BAS timeline response.
type ParsedTimeline struct {
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
	StepIndex   int    `json:"step_index"`
	NodeID      string `json:"node_id"`
	StepType    string `json:"step_type"`
	Status      string `json:"status"`
	Success     bool   `json:"success"`
	DurationMs  int    `json:"duration_ms,omitempty"`
	Error       string `json:"error,omitempty"`
	FinalURL    string `json:"final_url,omitempty"`
	Progress    int    `json:"progress,omitempty"`
	Screenshot  *FrameScreenshot `json:"screenshot,omitempty"`
	DOMSnapshot *FrameDOMSnapshot `json:"dom_snapshot,omitempty"`
	Assertion   *ParsedAssertion `json:"assertion,omitempty"`
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

	// First, parse into a flexible structure
	var raw rawTimeline
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, &TimelineParseError{
			RawData: data,
			Cause:   err,
		}
	}

	parsed := &ParsedTimeline{
		ExecutionID: raw.ExecutionID,
		WorkflowID:  raw.WorkflowID,
		Status:      raw.Status,
		Progress:    raw.Progress,
		StartedAt:   raw.StartedAt,
		CompletedAt: raw.CompletedAt,
		Frames:      make([]ParsedFrame, 0, len(raw.Frames)),
		Logs:        make([]ParsedLog, 0, len(raw.Logs)),
		Assertions:  make([]ParsedAssertion, 0),
	}

	// Parse frames and extract assertions/DOM
	var lastDOMPreview string
	var lastDOMFull string
	for _, rf := range raw.Frames {
		frame := ParsedFrame{
			StepIndex:  rf.StepIndex,
			NodeID:     rf.NodeID,
			StepType:   rf.StepType,
			Status:     rf.Status,
			Success:    rf.Success,
			DurationMs: rf.DurationMs,
			Error:      rf.Error,
			FinalURL:   rf.FinalURL,
			Progress:   rf.Progress,
		}

		// Extract screenshot info
		if rf.Screenshot != nil {
			frame.Screenshot = &FrameScreenshot{
				ArtifactID:   rf.Screenshot.ArtifactID,
				URL:          rf.Screenshot.URL,
				ThumbnailURL: rf.Screenshot.ThumbnailURL,
				Width:        rf.Screenshot.Width,
				Height:       rf.Screenshot.Height,
			}
		}

		// Extract DOM snapshot info
		if rf.DOMSnapshot != nil || rf.DOMSnapshotPreview != "" {
			frame.DOMSnapshot = &FrameDOMSnapshot{}
			if rf.DOMSnapshot != nil {
				frame.DOMSnapshot.ArtifactID = rf.DOMSnapshot.ID
				frame.DOMSnapshot.StorageURL = rf.DOMSnapshot.StorageURL
				if payload, ok := rf.DOMSnapshot.Payload["html"].(string); ok {
					lastDOMFull = payload
				}
			}
			if rf.DOMSnapshotPreview != "" {
				frame.DOMSnapshot.Preview = rf.DOMSnapshotPreview
				lastDOMPreview = rf.DOMSnapshotPreview
			}
		}

		// Extract assertion info
		if rf.StepType == "assert" || rf.Assertion != nil {
			assertion := ParsedAssertion{
				StepIndex: rf.StepIndex,
				NodeID:    rf.NodeID,
				Passed:    rf.Success,
				Error:     rf.Error,
			}
			if rf.Assertion != nil {
				assertion.AssertionType = rf.Assertion.Type
				assertion.Selector = rf.Assertion.Selector
				assertion.Expected = rf.Assertion.Expected
				assertion.Actual = rf.Assertion.Actual
				assertion.Message = rf.Assertion.Message
				assertion.Passed = rf.Assertion.Passed
			}
			frame.Assertion = &assertion
			parsed.Assertions = append(parsed.Assertions, assertion)
		}

		// Track failed frame
		if rf.Status == "failed" || rf.Error != "" {
			frameCopy := frame
			parsed.FailedFrame = &frameCopy
		}

		parsed.Frames = append(parsed.Frames, frame)
	}

	// Set final DOM from last frame or failed frame
	parsed.FinalDOMPreview = lastDOMPreview
	parsed.FinalDOM = lastDOMFull

	// Parse logs
	for _, rl := range raw.Logs {
		parsed.Logs = append(parsed.Logs, ParsedLog{
			ID:        rl.ID,
			Level:     rl.Level,
			Message:   rl.Message,
			StepName:  rl.StepName,
			Timestamp: rl.Timestamp,
		})
	}

	// Calculate summary
	parsed.Summary = calculateSummary(parsed)

	return parsed, nil
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

// Raw types for flexible JSON parsing

type rawTimeline struct {
	ExecutionID string           `json:"execution_id"`
	WorkflowID  string           `json:"workflow_id"`
	Status      string           `json:"status"`
	Progress    int              `json:"progress"`
	StartedAt   *time.Time       `json:"started_at"`
	CompletedAt *time.Time       `json:"completed_at"`
	Frames      []rawFrame       `json:"frames"`
	Logs        []rawLog         `json:"logs"`
}

type rawFrame struct {
	StepIndex          int           `json:"step_index"`
	NodeID             string        `json:"node_id"`
	StepType           string        `json:"step_type"`
	Status             string        `json:"status"`
	Success            bool          `json:"success"`
	DurationMs         int           `json:"duration_ms"`
	Error              string        `json:"error"`
	FinalURL           string        `json:"final_url"`
	Progress           int           `json:"progress"`
	DOMSnapshotPreview string        `json:"dom_snapshot_preview"`
	Screenshot         *rawScreenshot `json:"screenshot"`
	DOMSnapshot        *rawArtifact   `json:"dom_snapshot"`
	Assertion          *rawAssertion  `json:"assertion"`
}

type rawScreenshot struct {
	ArtifactID   string `json:"artifact_id"`
	URL          string `json:"url"`
	ThumbnailURL string `json:"thumbnail_url"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
}

type rawArtifact struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"`
	StorageURL   string         `json:"storage_url"`
	ThumbnailURL string         `json:"thumbnail_url"`
	ContentType  string         `json:"content_type"`
	StepIndex    *int           `json:"step_index"`
	Payload      map[string]any `json:"payload"`
}

type rawAssertion struct {
	Type     string `json:"type"`
	Selector string `json:"selector"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Message  string `json:"message"`
	Passed   bool   `json:"passed"`
}

type rawLog struct {
	ID        string    `json:"id"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	StepName  string    `json:"step_name"`
	Timestamp time.Time `json:"timestamp"`
}
