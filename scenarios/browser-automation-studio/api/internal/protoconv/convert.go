package protoconv

import (
	"encoding/json"
	"fmt"
	"strings"

	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/typeconv"
	"github.com/vrooli/browser-automation-studio/services/export"
	"github.com/vrooli/browser-automation-studio/services/workflow"
	basv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// String to enum converters

func stringToExecutionStatus(s string) basv1.ExecutionStatus {
	normalized := strings.ToUpper(strings.TrimSpace(s))
	switch normalized {
	case "PENDING", "EXECUTION_STATUS_PENDING":
		return basv1.ExecutionStatus_EXECUTION_STATUS_PENDING
	case "RUNNING", "EXECUTION_STATUS_RUNNING":
		return basv1.ExecutionStatus_EXECUTION_STATUS_RUNNING
	case "COMPLETED", "EXECUTION_STATUS_COMPLETED":
		return basv1.ExecutionStatus_EXECUTION_STATUS_COMPLETED
	case "FAILED", "EXECUTION_STATUS_FAILED":
		return basv1.ExecutionStatus_EXECUTION_STATUS_FAILED
	case "CANCELLED", "CANCELED", "EXECUTION_STATUS_CANCELLED":
		return basv1.ExecutionStatus_EXECUTION_STATUS_CANCELLED
	default:
		return basv1.ExecutionStatus_EXECUTION_STATUS_UNSPECIFIED
	}
}

func stringToTriggerType(s string) basv1.TriggerType {
	normalized := strings.ToUpper(strings.TrimSpace(s))
	switch normalized {
	case "MANUAL", "TRIGGER_TYPE_MANUAL":
		return basv1.TriggerType_TRIGGER_TYPE_MANUAL
	case "SCHEDULED", "TRIGGER_TYPE_SCHEDULED":
		return basv1.TriggerType_TRIGGER_TYPE_SCHEDULED
	case "API", "TRIGGER_TYPE_API":
		return basv1.TriggerType_TRIGGER_TYPE_API
	case "WEBHOOK", "TRIGGER_TYPE_WEBHOOK":
		return basv1.TriggerType_TRIGGER_TYPE_WEBHOOK
	default:
		return basv1.TriggerType_TRIGGER_TYPE_UNSPECIFIED
	}
}

func stringToExportStatus(s string) basv1.ExportStatus {
	normalized := strings.ToUpper(strings.TrimSpace(s))
	switch normalized {
	case "READY", "EXPORT_STATUS_READY":
		return basv1.ExportStatus_EXPORT_STATUS_READY
	case "PENDING", "NOT_READY", "EXPORT_STATUS_PENDING":
		return basv1.ExportStatus_EXPORT_STATUS_PENDING
	case "ERROR", "EXPORT_STATUS_ERROR":
		return basv1.ExportStatus_EXPORT_STATUS_ERROR
	case "UNAVAILABLE", "EXPORT_STATUS_UNAVAILABLE":
		return basv1.ExportStatus_EXPORT_STATUS_UNAVAILABLE
	default:
		return basv1.ExportStatus_EXPORT_STATUS_UNSPECIFIED
	}
}

func stringToStepType(s string) basv1.StepType {
	normalized := strings.ToLower(strings.TrimSpace(s))
	switch normalized {
	case "navigate", "step_type_navigate":
		return basv1.StepType_STEP_TYPE_NAVIGATE
	case "click", "step_type_click":
		return basv1.StepType_STEP_TYPE_CLICK
	case "input", "type", "step_type_input":
		return basv1.StepType_STEP_TYPE_INPUT
	case "assert", "step_type_assert":
		return basv1.StepType_STEP_TYPE_ASSERT
	case "subflow", "step_type_subflow":
		return basv1.StepType_STEP_TYPE_SUBFLOW
	case "custom", "step_type_custom":
		return basv1.StepType_STEP_TYPE_CUSTOM
	// Map other step types to CUSTOM since they're not in the proto
	case "wait", "extract", "screenshot", "scroll", "select", "hover", "keyboard", "condition", "loop":
		return basv1.StepType_STEP_TYPE_CUSTOM
	default:
		return basv1.StepType_STEP_TYPE_UNSPECIFIED
	}
}

func stringToStepStatus(s string) basv1.StepStatus {
	normalized := strings.ToUpper(strings.TrimSpace(s))
	switch normalized {
	case "PENDING", "STEP_STATUS_PENDING":
		return basv1.StepStatus_STEP_STATUS_PENDING
	case "RUNNING", "STEP_STATUS_RUNNING":
		return basv1.StepStatus_STEP_STATUS_RUNNING
	case "COMPLETED", "STEP_STATUS_COMPLETED":
		return basv1.StepStatus_STEP_STATUS_COMPLETED
	case "FAILED", "STEP_STATUS_FAILED":
		return basv1.StepStatus_STEP_STATUS_FAILED
	case "CANCELLED", "STEP_STATUS_CANCELLED":
		return basv1.StepStatus_STEP_STATUS_CANCELLED
	case "SKIPPED", "STEP_STATUS_SKIPPED":
		return basv1.StepStatus_STEP_STATUS_SKIPPED
	case "RETRYING", "STEP_STATUS_RETRYING":
		return basv1.StepStatus_STEP_STATUS_RETRYING
	default:
		return basv1.StepStatus_STEP_STATUS_UNSPECIFIED
	}
}

func stringToLogLevel(s string) basv1.LogLevel {
	normalized := strings.ToUpper(strings.TrimSpace(s))
	switch normalized {
	case "DEBUG", "LOG_LEVEL_DEBUG":
		return basv1.LogLevel_LOG_LEVEL_DEBUG
	case "INFO", "LOG_LEVEL_INFO":
		return basv1.LogLevel_LOG_LEVEL_INFO
	case "WARN", "WARNING", "LOG_LEVEL_WARN":
		return basv1.LogLevel_LOG_LEVEL_WARN
	case "ERROR", "LOG_LEVEL_ERROR":
		return basv1.LogLevel_LOG_LEVEL_ERROR
	default:
		return basv1.LogLevel_LOG_LEVEL_UNSPECIFIED
	}
}

func stringToArtifactType(s string) basv1.ArtifactType {
	normalized := strings.ToLower(strings.TrimSpace(s))
	switch normalized {
	case "screenshot", "artifact_type_screenshot":
		return basv1.ArtifactType_ARTIFACT_TYPE_SCREENSHOT
	case "dom", "dom_snapshot", "artifact_type_dom_snapshot":
		return basv1.ArtifactType_ARTIFACT_TYPE_DOM_SNAPSHOT
	case "timeline_frame", "artifact_type_timeline_frame":
		return basv1.ArtifactType_ARTIFACT_TYPE_TIMELINE_FRAME
	case "console_log", "artifact_type_console_log":
		return basv1.ArtifactType_ARTIFACT_TYPE_CONSOLE_LOG
	case "network_event", "artifact_type_network_event":
		return basv1.ArtifactType_ARTIFACT_TYPE_NETWORK_EVENT
	case "trace", "artifact_type_trace":
		return basv1.ArtifactType_ARTIFACT_TYPE_TRACE
	case "custom", "artifact_type_custom", "extracted_data", "video", "har":
		return basv1.ArtifactType_ARTIFACT_TYPE_CUSTOM
	default:
		return basv1.ArtifactType_ARTIFACT_TYPE_UNSPECIFIED
	}
}

// ExecutionToProto converts a database.Execution into the generated proto message.
func ExecutionToProto(execution *database.Execution) (*basv1.Execution, error) {
	if execution == nil {
		return nil, fmt.Errorf("execution is nil")
	}

	triggerMetadata, err := toStructMap(map[string]any(execution.TriggerMetadata), "trigger_metadata")
	if err != nil {
		return nil, err
	}

	parameters, err := toStructMap(map[string]any(execution.Parameters), "parameters")
	if err != nil {
		return nil, err
	}

	result, err := toStructMap(map[string]any(execution.Result), "result")
	if err != nil {
		return nil, err
	}

	pb := &basv1.Execution{
		Id:              execution.ID.String(),
		WorkflowId:      execution.WorkflowID.String(),
		WorkflowVersion: int32(execution.WorkflowVersion),
		Status:          stringToExecutionStatus(execution.Status),
		TriggerType:     stringToTriggerType(execution.TriggerType),
		Progress:        int32(execution.Progress),
		CurrentStep:     execution.CurrentStep,
		StartedAt:       timestamppb.New(execution.StartedAt),
		CreatedAt:       timestamppb.New(execution.CreatedAt),
		UpdatedAt:       timestamppb.New(execution.UpdatedAt),
	}

	if len(triggerMetadata) > 0 {
		pb.TriggerMetadata = triggerMetadata
	}
	if len(parameters) > 0 {
		pb.Parameters = parameters
	}
	if execution.CompletedAt != nil {
		pb.CompletedAt = timestamppb.New(*execution.CompletedAt)
	}
	if execution.LastHeartbeat != nil {
		pb.LastHeartbeat = timestamppb.New(*execution.LastHeartbeat)
	}
	if execution.Error.Valid {
		errMsg := execution.Error.String
		pb.Error = &errMsg
	}
	if len(result) > 0 {
		pb.Result = result
	}

	return pb, nil
}

// ExecutionExportPreviewToProto converts the workflow.ExecutionExportPreview to the proto message.
func ExecutionExportPreviewToProto(preview *workflow.ExecutionExportPreview) (*basv1.ExecutionExportPreview, error) {
	if preview == nil {
		return nil, fmt.Errorf("preview is nil")
	}

	var pkg *structpb.Struct
	if preview.Package != nil {
		raw, err := json.Marshal(preview.Package)
		if err != nil {
			return nil, fmt.Errorf("package marshal: %w", err)
		}
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, fmt.Errorf("package unmarshal: %w", err)
		}
		pkg, err = structpb.NewStruct(m)
		if err != nil {
			return nil, fmt.Errorf("package: %w", err)
		}
	}

	return &basv1.ExecutionExportPreview{
		ExecutionId:         preview.ExecutionID.String(),
		SpecId:              preview.SpecID,
		Status:              stringToExportStatus(preview.Status),
		Message:             preview.Message,
		CapturedFrameCount:  int32(preview.CapturedFrameCount),
		AvailableAssetCount: int32(preview.AvailableAssetCount),
		TotalDurationMs:     int32(preview.TotalDurationMs),
		Package:             pkg,
	}, nil
}

// ScreenshotsToProto converts database screenshots to the proto response.
func ScreenshotsToProto(screenshots []*database.Screenshot) (*basv1.GetScreenshotsResponse, error) {
	if len(screenshots) == 0 {
		return &basv1.GetScreenshotsResponse{}, nil
	}

	result := make([]*basv1.Screenshot, 0, len(screenshots))
	for idx, shot := range screenshots {
		if shot == nil {
			return nil, fmt.Errorf("screenshots[%d] is nil", idx)
		}

		var stepIndex int32
		if shot.Metadata != nil {
			if v, ok := shot.Metadata["step_index"]; ok {
				switch t := v.(type) {
				case int:
					stepIndex = int32(t)
				case int32:
					stepIndex = t
				case int64:
					stepIndex = int32(t)
				case float64:
					stepIndex = int32(t)
				case float32:
					stepIndex = int32(t)
				}
			}
		}

		thumbURL := shot.ThumbnailURL
		if thumbURL == "" && shot.Metadata != nil {
			if v, ok := shot.Metadata["thumbnail_url"].(string); ok {
				thumbURL = v
			}
		}

		result = append(result, &basv1.Screenshot{
			Id:           shot.ID.String(),
			ExecutionId:  shot.ExecutionID.String(),
			StepName:     shot.StepName,
			StepIndex:    stepIndex,
			Timestamp:    timestamppb.New(shot.Timestamp),
			StorageUrl:   shot.StorageURL,
			ThumbnailUrl: thumbURL,
			Width:        int32(shot.Width),
			Height:       int32(shot.Height),
			SizeBytes:    shot.SizeBytes,
		})
	}

	return &basv1.GetScreenshotsResponse{Screenshots: result}, nil
}

// TimelineToProto converts the replay-focused ExecutionTimeline into the proto message.
func TimelineToProto(timeline *export.ExecutionTimeline) (*basv1.ExecutionTimeline, error) {
	if timeline == nil {
		return nil, fmt.Errorf("timeline is nil")
	}

	pb := &basv1.ExecutionTimeline{
		ExecutionId: timeline.ExecutionID.String(),
		WorkflowId:  timeline.WorkflowID.String(),
		Status:      stringToExecutionStatus(timeline.Status),
		Progress:    int32(timeline.Progress),
		StartedAt:   timestamppb.New(timeline.StartedAt),
	}

	if timeline.CompletedAt != nil {
		pb.CompletedAt = timestamppb.New(*timeline.CompletedAt)
	}

	for idx, frame := range timeline.Frames {
		pbFrame, err := timelineFrameToProto(frame)
		if err != nil {
			return nil, fmt.Errorf("frame %d: %w", idx, err)
		}
		pb.Frames = append(pb.Frames, pbFrame)
	}

	for _, log := range timeline.Logs {
		pb.Logs = append(pb.Logs, timelineLogToProto(log))
	}

	return pb, nil
}

func timelineFrameToProto(frame export.TimelineFrame) (*basv1.TimelineFrame, error) {
	extractedPreview, err := toProtoValue(frame.ExtractedDataPreview, "frames.extracted_data_preview")
	if err != nil {
		return nil, err
	}

	var startedAt *timestamppb.Timestamp
	if frame.StartedAt != nil {
		startedAt = timestamppb.New(*frame.StartedAt)
	}
	var completedAt *timestamppb.Timestamp
	if frame.CompletedAt != nil {
		completedAt = timestamppb.New(*frame.CompletedAt)
	}

	pbFrame := &basv1.TimelineFrame{
		StepIndex:            int32(frame.StepIndex),
		NodeId:               frame.NodeID,
		StepType:             stringToStepType(frame.StepType),
		Status:               stringToStepStatus(frame.Status),
		Success:              frame.Success,
		DurationMs:           int32(frame.DurationMs),
		TotalDurationMs:      int32(frame.TotalDurationMs),
		Progress:             int32(frame.Progress),
		StartedAt:            startedAt,
		CompletedAt:          completedAt,
		FinalUrl:             frame.FinalURL,
		Error:                frame.Error,
		ConsoleLogCount:      int32(frame.ConsoleLogCount),
		NetworkEventCount:    int32(frame.NetworkEventCount),
		ExtractedDataPreview: extractedPreview,
		ZoomFactor:           frame.ZoomFactor,
		RetryAttempt:         int32(frame.RetryAttempt),
		RetryMaxAttempts:     int32(frame.RetryMaxAttempts),
		RetryConfigured:      int32(frame.RetryConfigured),
		RetryDelayMs:         int32(frame.RetryDelayMs),
		RetryBackoffFactor:   frame.RetryBackoffFactor,
		RetryHistory:         convertRetryHistory(frame.RetryHistory),
		DomSnapshotPreview:   frame.DomSnapshotPreview,
	}

	if frame.HighlightRegions != nil {
		for _, region := range frame.HighlightRegions {
			pbFrame.HighlightRegions = append(pbFrame.HighlightRegions, convertHighlightRegion(region))
		}
	}

	if frame.MaskRegions != nil {
		for _, region := range frame.MaskRegions {
			pbFrame.MaskRegions = append(pbFrame.MaskRegions, convertMaskRegion(region))
		}
	}

	if frame.FocusedElement != nil {
		pbFrame.FocusedElement = convertElementFocus(frame.FocusedElement)
	}

	if frame.ElementBoundingBox != nil {
		pbFrame.ElementBoundingBox = convertBoundingBox(frame.ElementBoundingBox)
	}

	if frame.ClickPosition != nil {
		pbFrame.ClickPosition = convertPoint(frame.ClickPosition)
	}

	if frame.CursorTrail != nil {
		for _, pt := range frame.CursorTrail {
			point := pt
			pbFrame.CursorTrail = append(pbFrame.CursorTrail, convertPoint(&point))
		}
	}

	if frame.Screenshot != nil {
		pbFrame.Screenshot = convertScreenshot(frame.Screenshot)
	}

	if frame.Artifacts != nil {
		for _, artifact := range frame.Artifacts {
			pbArtifact, err := convertArtifact(artifact)
			if err != nil {
				return nil, fmt.Errorf("artifact %s: %w", artifact.ID, err)
			}
			pbFrame.Artifacts = append(pbFrame.Artifacts, pbArtifact)
		}
	}

	if frame.Assertion != nil {
		assertion, err := convertAssertion(frame.Assertion)
		if err != nil {
			return nil, err
		}
		pbFrame.Assertion = assertion
	}

	if frame.DomSnapshot != nil {
		dom, err := convertArtifact(*frame.DomSnapshot)
		if err != nil {
			return nil, fmt.Errorf("dom_snapshot: %w", err)
		}
		pbFrame.DomSnapshot = dom
	}

	return pbFrame, nil
}

func timelineLogToProto(log export.TimelineLog) *basv1.TimelineLog {
	return &basv1.TimelineLog{
		Id:        log.ID,
		Level:     stringToLogLevel(log.Level),
		Message:   log.Message,
		StepName:  log.StepName,
		Timestamp: timestamppb.New(log.Timestamp),
	}
}

func convertRetryHistory(entries []typeconv.RetryHistoryEntry) []*basv1.RetryHistoryEntry {
	if len(entries) == 0 {
		return nil
	}
	result := make([]*basv1.RetryHistoryEntry, 0, len(entries))
	for _, entry := range entries {
		result = append(result, &basv1.RetryHistoryEntry{
			Attempt:        int32(entry.Attempt),
			Success:        entry.Success,
			DurationMs:     int32(entry.DurationMs),
			CallDurationMs: int32(entry.CallDurationMs),
			Error:          entry.Error,
		})
	}
	return result
}

func convertHighlightRegion(region autocontracts.HighlightRegion) *basv1.HighlightRegion {
	pb := &basv1.HighlightRegion{
		Selector: region.Selector,
		Padding:  int32(region.Padding),
		Color:    region.Color,
	}
	if region.BoundingBox != nil {
		pb.BoundingBox = convertBoundingBox(region.BoundingBox)
	}
	return pb
}

func convertMaskRegion(region autocontracts.MaskRegion) *basv1.MaskRegion {
	pb := &basv1.MaskRegion{
		Selector: region.Selector,
		Opacity:  region.Opacity,
	}
	if region.BoundingBox != nil {
		pb.BoundingBox = convertBoundingBox(region.BoundingBox)
	}
	return pb
}

func convertElementFocus(focus *autocontracts.ElementFocus) *basv1.ElementFocus {
	pb := &basv1.ElementFocus{
		Selector: focus.Selector,
	}
	if focus.BoundingBox != nil {
		pb.BoundingBox = convertBoundingBox(focus.BoundingBox)
	}
	return pb
}

func convertBoundingBox(bbox *autocontracts.BoundingBox) *basv1.BoundingBox {
	if bbox == nil {
		return nil
	}
	return &basv1.BoundingBox{
		X:      bbox.X,
		Y:      bbox.Y,
		Width:  bbox.Width,
		Height: bbox.Height,
	}
}

func convertPoint(pt *autocontracts.Point) *basv1.Point {
	if pt == nil {
		return nil
	}
	return &basv1.Point{
		X: pt.X,
		Y: pt.Y,
	}
}

func convertScreenshot(screenshot *typeconv.TimelineScreenshot) *basv1.TimelineScreenshot {
	if screenshot == nil {
		return nil
	}
	pb := &basv1.TimelineScreenshot{
		ArtifactId:   screenshot.ArtifactID,
		Url:          screenshot.URL,
		ThumbnailUrl: screenshot.ThumbnailURL,
		Width:        int32(screenshot.Width),
		Height:       int32(screenshot.Height),
		ContentType:  screenshot.ContentType,
	}
	if screenshot.SizeBytes != nil {
		pb.SizeBytes = *screenshot.SizeBytes
	}
	return pb
}

func convertArtifact(artifact typeconv.TimelineArtifact) (*basv1.TimelineArtifact, error) {
	payload, err := toStructMap(artifact.Payload, "artifacts.payload")
	if err != nil {
		return nil, err
	}

	pb := &basv1.TimelineArtifact{
		Id:           artifact.ID,
		Type:         stringToArtifactType(artifact.Type),
		Label:        artifact.Label,
		StorageUrl:   artifact.StorageURL,
		ThumbnailUrl: artifact.ThumbnailURL,
		ContentType:  artifact.ContentType,
	}
	if artifact.SizeBytes != nil {
		pb.SizeBytes = *artifact.SizeBytes
	}
	if artifact.StepIndex != nil {
		stepIndex := int32(*artifact.StepIndex)
		pb.StepIndex = &stepIndex
	}
	if len(payload) > 0 {
		pb.Payload = payload
	}
	return pb, nil
}

func convertAssertion(assertion *autocontracts.AssertionOutcome) (*basv1.AssertionOutcome, error) {
	if assertion == nil {
		return nil, nil
	}
	pb := &basv1.AssertionOutcome{
		Mode:          assertion.Mode,
		Selector:      assertion.Selector,
		Success:       assertion.Success,
		Negated:       assertion.Negated,
		CaseSensitive: assertion.CaseSensitive,
		Message:       assertion.Message,
	}

	if assertion.Expected != nil {
		val, err := toProtoValue(assertion.Expected, "assertion.expected")
		if err != nil {
			return nil, err
		}
		pb.Expected = val
	}

	if assertion.Actual != nil {
		val, err := toProtoValue(assertion.Actual, "assertion.actual")
		if err != nil {
			return nil, err
		}
		pb.Actual = val
	}

	return pb, nil
}

func toStructMap(source map[string]any, field string) (map[string]*structpb.Value, error) {
	if len(source) == 0 {
		return nil, nil
	}
	result := make(map[string]*structpb.Value, len(source))
	for key, value := range source {
		val, err := structpb.NewValue(value)
		if err != nil {
			return nil, fmt.Errorf("%s[%s]: %w", field, key, err)
		}
		result[key] = val
	}
	return result, nil
}

func toProtoValue(value any, field string) (*structpb.Value, error) {
	if value == nil {
		return nil, nil
	}
	val, err := structpb.NewValue(value)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", field, err)
	}
	return val, nil
}
